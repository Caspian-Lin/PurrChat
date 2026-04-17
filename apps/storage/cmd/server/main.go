package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"purr-chat-storage/internal/handlers"
	"purr-chat-storage/internal/repository"
	"purr-chat-storage/internal/services"
	"purr-chat-storage/internal/storage"
	"purr-chat-storage/pkg/config"
	"purr-chat-storage/pkg/database"
	"purr-chat-storage/pkg/logger"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg := config.Load()
	config.Validate(cfg)

	// 初始化日志
	logConfig := &logger.LogConfig{
		Directory: cfg.Log.Directory,
		MaxFiles:  cfg.Log.MaxFiles,
		MaxLines:  cfg.Log.MaxLines,
	}
	if err := logger.InitWithConfig(logConfig); err != nil {
		logger.Init()
		logger.Error("Failed to initialize file logger:", err)
	}

	// 设置时区为 UTC
	time.Local = time.UTC
	logger.Info("Timezone set to UTC for consistent timestamp handling")

	logger.Info("Starting PurrChat Storage Service...")

	// 初始化数据库
	dsn := config.GetDSN(&cfg.DB)
	if err := database.Init(dsn); err != nil {
		logger.Error("Failed to connect to database:", err)
		os.Exit(1)
	}
	defer database.Close()

	// 初始化存储提供者
	var storageProvider storage.StorageProvider
	switch cfg.Storage.Provider {
	case "r2":
		storageProvider = storage.NewR2Provider(cfg.Storage)
		logger.Info("Using Cloudflare R2 storage provider, endpoint:", cfg.Storage.Endpoint)
	default:
		storageProvider = storage.NewMinIOProvider(cfg.Storage)
		logger.Info("Using MinIO storage provider, endpoint:", cfg.Storage.Endpoint)
	}

	if storageProvider == nil {
		logger.Error("Failed to create storage provider, file operations will be unavailable")
	}

	// 初始化存储后端（失败不退出，服务照常启动但文件操作不可用）
	if storageProvider != nil {
		if err := storageProvider.Initialize(context.Background()); err != nil {
			logger.Error("Failed to initialize storage:", err)
			logger.Error("Storage service will start but file operations will be unavailable")
			storageProvider = nil
		}
	}

	if storageProvider != nil {
		logger.Infof("Storage provider initialized: %s, bucket: %s", cfg.Storage.Provider, cfg.Storage.Bucket)
		if cfg.Storage.PublicURL != "" {
			logger.Infof("Storage public URL: %s", cfg.Storage.PublicURL)
		} else {
			logger.Info("WARNING: Storage public URL is NOT configured — generated file URLs may be empty")
		}
	} else {
		logger.Info("WARNING: Storage provider NOT initialized — file upload/download will return errors")
	}

	// 初始化依赖
	fileRepo := repository.NewFileRepository()
	fileService := services.NewFileService(fileRepo, storageProvider)
	fileHandler := handlers.NewFileHandler(fileService)

	// 设置 Gin 模式
	gin.SetMode(cfg.GinMode)

	// 创建 Gin 路由
	r := gin.Default()

	// 配置日志中间件
	r.Use(handlers.LoggingMiddleware())

	// 配置 CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"service":   "purrchat-storage",
			"provider":  cfg.Storage.Provider,
			"message":   "PurrChat Storage Service is running",
		})
	})
	r.HEAD("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "purrchat-storage",
		})
	})

	// 文件路由
	files := r.Group("/api/files")
	{
		files.POST("/upload/request", handlers.AuthMiddleware(cfg.JWT.Secret), fileHandler.RequestUpload)
		files.POST("/upload/confirm", handlers.AuthMiddleware(cfg.JWT.Secret), fileHandler.ConfirmUpload)
		files.GET("/download/url", handlers.AuthMiddleware(cfg.JWT.Secret), fileHandler.GetDownloadURL)
		files.DELETE("/:file_id", handlers.AuthMiddleware(cfg.JWT.Secret), fileHandler.DeleteFile)
	}

	// 启动服务器
	go func() {
		logger.Infof("Storage server is running on port %s", cfg.Port)
		if err := r.Run(":" + cfg.Port); err != nil {
			logger.Error("Failed to start server:", err)
			os.Exit(1)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down storage server...")
}
