package main

import (
	"context"
	"os"
	"os/signal"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"time"

	"purr-chat-server/internal/botengine"
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
	"golang.org/x/time/rate"
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

	// 读取 migrations 目录下的所有 SQL 文件
	migrationDir := "migrations"
	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		logger.Error("Failed to read migrations directory:", err)
		os.Exit(1)
	}

	// 收集所有 .sql 文件
	var migrationFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrationFiles = append(migrationFiles, entry.Name())
		}
	}

	// 按文件名排序（确保按数字顺序执行）
	sort.Strings(migrationFiles)

	if len(migrationFiles) == 0 {
		logger.Info("No migration files found in", migrationDir)
		return
	}

	logger.Info("Found", len(migrationFiles), "migration file(s)")

	// 执行所有迁移SQL文件
	for _, fileName := range migrationFiles {
		migrationPath := migrationDir + "/" + fileName
		logger.Info("Executing migration:", migrationPath)

		content, err := os.ReadFile(migrationPath)
		if err != nil {
			logger.Error("Failed to read migration file:", err)
			os.Exit(1)
		}

		// 执行SQL（使用 IF NOT EXISTS 避免重复创建错误）
		_, err = database.GetPool().Exec(context.Background(), string(content))
		if err != nil {
			// 检查是否是"已存在"错误，如果是则忽略
			if isAlreadyExistsError(err) {
				logger.Info("Migration skipped (already exists):", migrationPath)
				continue
			}
			logger.Error("Failed to execute migration:", err)
			os.Exit(1)
		}

		logger.Info("Migration completed successfully:", migrationPath)
	}

	logger.Info("All migrations completed successfully")
}

// isAlreadyExistsError 检查是否是"已存在"错误
func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// 检查常见的"已存在"错误模式
	return contains(errStr, "already exists") ||
		contains(errStr, "duplicate") ||
		contains(errStr, "42P07") // PostgreSQL 错误码：relation already exists
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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

	// 设置时区为UTC（确保时间戳一致性）
	// 注意：虽然服务器运行在中国时区，但所有时间戳都应使用UTC存储
	// 这样可以避免时区转换问题，确保前端显示的时间戳一致
	time.Local = time.UTC
	logger.Info("Timezone set to UTC for consistent timestamp handling")

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

	// 全局 per-IP 速率限制 — 防止 DDoS 和 API 滥用
	r.Use(handlers.IPRateLimitMiddleware(
		rate.Limit(cfg.RateLimit.GlobalRatePerSec),
		cfg.RateLimit.GlobalBurst,
		10*time.Minute, // visitor 条目过期时间
	))

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
	botRepo := repository.NewBotRepository()
	botDeployRepo := repository.NewBotDeploymentRepository()
	botService := services.NewBotService(botRepo, botDeployRepo, userRepo, friendshipRepo, conversationRepo, enrollmentRepo, conversationMessageRepo)
	authHandler := handlers.NewAuthHandler(authService, cfg.JWT.Secret)
	chatHandler := handlers.NewChatHandler(authService, chatService)
	botEngine := botengine.NewBotEngine(botDeployRepo, botRepo, conversationMessageRepo, enrollmentRepo)
	botHandler := handlers.NewBotHandler(botService, botEngine)
	chatService.SetBotEngine(botEngine)
	chatService.SetBotRepo(botRepo)
	settingsRepo := repository.NewSettingsRepository()
	settingsService := services.NewSettingsService(settingsRepo)
	settingsHandler := handlers.NewSettingsHandler(settingsService)

	// 初始化WebSocket hub
	websocket.InitHubWithConfig(cfg.WebSocket.MaxConnections, cfg.WebSocket.MaxUserConnections)
	websocket.InitJWTSecret(cfg.JWT.Secret)

	// 认证端点 per-IP 速率限制 — 防止暴力破解和批量注册
	authRateLimit := handlers.IPRateLimitMiddleware(
		rate.Limit(cfg.RateLimit.AuthRatePerSec),
		cfg.RateLimit.AuthBurst,
		10*time.Minute,
	)

	// 已认证用户速率限制 — 防止已登录用户滥用 API
	userRateLimit := handlers.UserRateLimitMiddleware(
		rate.Limit(cfg.RateLimit.UserRatePerSec),
		cfg.RateLimit.UserBurst,
		10*time.Minute,
	)

	// 敏感操作速率限制 — 防止好友请求洪水和消息轰炸
	sensitiveRateLimit := handlers.UserRateLimitMiddleware(
		rate.Limit(cfg.RateLimit.SensitiveRatePerSec),
		cfg.RateLimit.SensitiveBurst,
		10*time.Minute,
	)

	// 健康检查（不受速率限制）
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "PurrChat Server is running",
		})
	})
	r.HEAD("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "PurrChat Server is running",
		})
	})

	// 认证路由（严格 per-IP 限流）
	auth := r.Group("/api")
	{
		auth.POST("/register", authRateLimit, authHandler.Register)
		auth.POST("/login", authRateLimit, authHandler.Login)
		auth.GET("/me", handlers.AuthMiddleware(cfg.JWT.Secret), userRateLimit, authHandler.Me)
		auth.PUT("/profile", handlers.AuthMiddleware(cfg.JWT.Secret), userRateLimit, chatHandler.UpdateProfile)
		auth.PUT("/password", handlers.AuthMiddleware(cfg.JWT.Secret), userRateLimit, authHandler.ChangePassword)
	}

	// 用户路由（per-User 限流）
	users := r.Group("/api/users")
	users.Use(handlers.AuthMiddleware(cfg.JWT.Secret))
	users.Use(userRateLimit)
	{
		users.GET("/search", chatHandler.SearchUsers)
		users.GET("/:id", chatHandler.GetUserByID)
		users.GET("/uid/:uid", chatHandler.GetUserByUID)
	}

	// 会话路由（per-User 限流）
	conversations := r.Group("/api/conversations")
	conversations.Use(handlers.AuthMiddleware(cfg.JWT.Secret))
	conversations.Use(userRateLimit)
	{
		conversations.GET("", chatHandler.GetConversations)
		conversations.POST("", chatHandler.CreateConversation)
		conversations.POST("/group", chatHandler.CreateGroupConversation)
		conversations.PUT("", chatHandler.UpdateConversation)
		conversations.DELETE("", chatHandler.DeleteConversation)
		conversations.GET("/members", chatHandler.GetConversationMembers)
		conversations.PUT("/members/role", chatHandler.UpdateMemberRole)
		conversations.POST("/members", chatHandler.AddMemberToConversation)
		conversations.DELETE("/members", chatHandler.RemoveMemberFromConversation)
		conversations.GET("/:id/bots", botHandler.GetConversationBots)
	}

	// 消息路由（发送消息使用严格限流）
	messages := r.Group("/api/messages")
	messages.Use(handlers.AuthMiddleware(cfg.JWT.Secret))
	messages.Use(userRateLimit)
	{
		messages.GET("", chatHandler.GetMessages)
		messages.GET("/export", chatHandler.ExportMessages)
		messages.GET("/incremental", chatHandler.GetMessagesIncremental)
		messages.POST("", sensitiveRateLimit, chatHandler.SendMessage) // 发送消息使用更严格的限流
	}

	// 好友路由（敏感操作使用严格限流）
	friends := r.Group("/api/friends")
	friends.Use(handlers.AuthMiddleware(cfg.JWT.Secret))
	friends.Use(userRateLimit)
	{
		friends.GET("", chatHandler.GetFriends)
		friends.GET("/pending", chatHandler.GetPendingFriendRequests)
		friends.GET("/requests", chatHandler.GetAllFriendRequests)
		friends.POST("/request", sensitiveRateLimit, chatHandler.SendFriendRequest)  // 发送好友请求严格限流
		friends.POST("/handle", sensitiveRateLimit, chatHandler.HandleFriendRequest) // 处理好友请求严格限流
	}

	// WebSocket路由（不使用AuthMiddleware，因为WebSocket通过查询参数传递token）
	r.GET("/api/ws", websocket.HandleWebSocket)

	// Bot 路由（per-User 限流）
	bots := r.Group("/api/bots")
	bots.Use(handlers.AuthMiddleware(cfg.JWT.Secret))
	bots.Use(userRateLimit)
	{
		bots.GET("", botHandler.ListBots)
		bots.GET("/search", botHandler.SearchBots)
		bots.GET("/deployments", botHandler.GetBotDeployments)
		bots.POST("", sensitiveRateLimit, botHandler.CreateBot)
		bots.GET("/:id", botHandler.GetBot)
		bots.PUT("/:id", botHandler.UpdateBot)
		bots.DELETE("/:id", sensitiveRateLimit, botHandler.DeleteBot)
		bots.POST("/:id/deploy", botHandler.DeployBot)
		bots.DELETE("/:id/deploy", botHandler.UndeployBot)
		bots.PUT("/:id/deploy/status", botHandler.UpdateDeploymentStatus)
		bots.POST("/:id/conversation", botHandler.CreateBotConversation)
		bots.POST("/:id/special-mode/activate", botHandler.ActivateSpecialMode)
		bots.POST("/:id/special-mode/deactivate", botHandler.DeactivateSpecialMode)
		bots.GET("/:id/deployable-conversations", botHandler.GetDeployableConversations)
		bots.POST("/:id/debug", botHandler.DebugBot)
		bots.POST("/:id/debug/step", botHandler.DebugStep)
		bots.POST("/:id/debug/reset", botHandler.DebugReset)
	}

	// 设置路由（per-User 限流）
	settingsGroup := r.Group("/api/settings")
	settingsGroup.Use(handlers.AuthMiddleware(cfg.JWT.Secret))
	settingsGroup.Use(userRateLimit)
	{
		settingsGroup.GET("", settingsHandler.GetSettings)
		settingsGroup.PUT("", settingsHandler.UpdateSettings)
	}

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
