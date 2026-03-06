package main

import (
	"context"
	"os"
	"os/signal"
	"reflect"
	"syscall"

	"purr-chat-server/internal/handlers"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"
	"purr-chat-server/internal/websocket"
	"purr-chat-server/pkg/config"
	"purr-chat-server/pkg/database"
	"purr-chat-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// registerUUIDValidator 注册 UUID 验证器
func registerUUIDValidator(v *validator.Validate) {
	// 注册自定义 UUID 验证器
	v.RegisterCustomTypeFunc(func(field reflect.Value) interface{} {
		if val, ok := field.Interface().(uuid.UUID); ok {
			return val.String()
		}
		return nil
	})

	// 注册 UUID 验证函数
	_ = v.RegisterValidation("uuid", func(fl validator.FieldLevel) bool {
		field := fl.Field()
		if field.Kind() == reflect.String {
			_, err := uuid.Parse(field.String())
			return err == nil
		}
		return true
	})
}

// runMigrate 执行数据库迁移
func runMigrate() {
	// 初始化日志（使用默认配置，输出到控制台）
	logger.Init()

	logger.Info("Running database migrations...")

	// 初始化数据库连接
	cfg := config.Load()
	dsn := config.GetDSN(&cfg.DB)
	if err := database.Init(dsn); err != nil {
		logger.Error("Failed to connect to database:", err)
		os.Exit(1)
	}
	defer database.Close()

	// 执行迁移SQL文件
	migrationFile := "migrations/002_new_conversation_structure.sql"
	content, err := os.ReadFile(migrationFile)
	if err != nil {
		logger.Error("Failed to read migration file:", err)
		os.Exit(1)
	}

	// 执行SQL
	_, err = database.GetPool().Exec(context.Background(), string(content))
	if err != nil {
		logger.Error("Failed to execute migration:", err)
		os.Exit(1)
	}

	logger.Info("Migration completed successfully")
}

func main() {
	// 检查是否是migrate命令
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrate()
		return
	}

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
		// 如果文件日志初始化失败，使用默认的日志初始化
		logger.Init()
		logger.Error("Failed to initialize file logger:", err)
	}

	logger.Info("Starting PurrChat Server...")

	// 初始化数据库
	dsn := config.GetDSN(&cfg.DB)
	if err := database.Init(dsn); err != nil {
		logger.Error("Failed to connect to database:", err)
		os.Exit(1)
	}
	defer database.Close()

	// 注册自定义 UUID 验证器
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		registerUUIDValidator(v)
	}

	// 设置 Gin 模式
	gin.SetMode(cfg.GinMode)

	// 创建 Gin 路由
	r := gin.Default()

	// 配置日志中间件（记录所有API活动）
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

	// 初始化依赖
	userRepo := repository.NewUserRepository()
	conversationRepo := repository.NewConversationRepository()
	messageRepo := repository.NewMessageRepository()
	friendshipRepo := repository.NewFriendshipRepository()
	enrollmentRepo := repository.NewEnrollmentRepository()
	conversationMessageRepo := repository.NewConversationMessageRepository()
	authService := services.NewAuthService(userRepo, cfg.JWT.Secret)
	chatService := services.NewChatService(userRepo, conversationRepo, messageRepo, friendshipRepo, enrollmentRepo, conversationMessageRepo)
	authHandler := handlers.NewAuthHandler(authService, cfg.JWT.Secret)
	chatHandler := handlers.NewChatHandler(authService, chatService)

	// 初始化WebSocket hub
	websocket.InitHub()
	websocket.InitJWTSecret(cfg.JWT.Secret)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "PurrChat Server is running",
		})
	})

	// 认证路由
	auth := r.Group("/api")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.GET("/me", handlers.AuthMiddleware(cfg.JWT.Secret), authHandler.Me)
		auth.PUT("/profile", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.UpdateProfile)
	}

	// 用户路由
	users := r.Group("/api/users")
	{
		users.GET("/search", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.SearchUsers)
		users.GET("/:id", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.GetUserByID)
		users.GET("/uid/:uid", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.GetUserByUID)
	}

	// 会话路由
	conversations := r.Group("/api/conversations")
	{
		conversations.GET("", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.GetConversations)
		conversations.POST("", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.CreateConversation)
		conversations.POST("/group", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.CreateGroupConversation)
		conversations.GET("/members", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.GetConversationMembers)
		conversations.POST("/members", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.AddMemberToConversation)
		conversations.DELETE("/members", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.RemoveMemberFromConversation)
	}

	// 消息路由
	messages := r.Group("/api/messages")
	{
		messages.GET("", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.GetMessages)
		messages.GET("/export", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.ExportMessages)
		messages.GET("/incremental", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.GetMessagesIncremental)
		messages.POST("", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.SendMessage)
	}

	// 好友路由
	friends := r.Group("/api/friends")
	{
		friends.GET("", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.GetFriends)
		friends.GET("/pending", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.GetPendingFriendRequests)
		friends.GET("/requests", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.GetAllFriendRequests)
		friends.POST("/request", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.SendFriendRequest)
		friends.POST("/handle", handlers.AuthMiddleware(cfg.JWT.Secret), chatHandler.HandleFriendRequest)
	}

	// WebSocket路由（不使用AuthMiddleware，因为WebSocket通过查询参数传递token）
	r.GET("/api/ws", websocket.HandleWebSocket)

	// 启动服务器
	go func() {
		logger.Infof("Server is running on port %s", cfg.Port)
		if err := r.Run(":" + cfg.Port); err != nil {
			logger.Error("Failed to start server:", err)
			os.Exit(1)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
}
