package main

import (
	"context"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"purr-chat-server/internal/botaction"
	"purr-chat-server/internal/botengine"
	"purr-chat-server/internal/botws"
	"purr-chat-server/internal/handlers"
	"purr-chat-server/internal/messaging"
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

	// 配置 CORS 中间件（支持 Cookie 认证）
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, If-Match, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Set-Cookie, ETag")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 初始化依赖
	userRepo := repository.NewUserRepository()
	conversationRepo := repository.NewConversationRepository()
	friendshipRepo := repository.NewFriendshipRepository()
	enrollmentRepo := repository.NewEnrollmentRepository()
	conversationMessageRepo := repository.NewConversationMessageRepository()
	botRepo := repository.NewBotRepository()
	botDeployRepo := repository.NewBotDeploymentRepository()
	installationRepo := repository.NewBotInstallationRepository()
	callLogRepo := repository.NewBotCallLogRepository()
	outboxRepo := repository.NewBotEventOutboxRepository()
	authService := services.NewAuthService(userRepo, botRepo, cfg.JWT.Secret)
	botEngine := botengine.NewBotEngine(botDeployRepo, botRepo, conversationMessageRepo, enrollmentRepo, os.Getenv("BOT_ENGINE_URL"))
	botEngine.SetCallLogRepo(callLogRepo)
	botEngine.SetInstallationRepo(installationRepo)
	conversationService := services.NewConversationService(userRepo, conversationRepo, enrollmentRepo, conversationMessageRepo, friendshipRepo)
	conversationService.SetBotRepo(botRepo)

	// 消息事件发布器：fan-out 到用户 WS、Workflow Bot、External Bot
	publisher := messaging.NewPublisher(30 * time.Second)
	messageService := services.NewMessageService(userRepo, conversationRepo, enrollmentRepo, conversationMessageRepo, botRepo, installationRepo, publisher)
	botEngine.SetMessageSender(messageService)
	friendService := services.NewFriendService(userRepo, friendshipRepo, enrollmentRepo, conversationMessageRepo)
	memberService := services.NewMemberService(userRepo, conversationRepo, enrollmentRepo)
	userService := services.NewUserService(userRepo)
	botService := services.NewBotService(botRepo, installationRepo, userRepo, friendshipRepo, conversationRepo, enrollmentRepo, conversationMessageRepo, callLogRepo)
	actionDispatcher := botaction.NewDispatcher(messageService, botRepo, userRepo, conversationRepo, enrollmentRepo, conversationMessageRepo, installationRepo, outboxRepo)
	botWSManager := botws.NewManager(botws.DefaultConfig(), actionDispatcher)
	botWSManager.SetReplayer(services.NewOutboxReplayer(outboxRepo))
	botService.SetConnectionCloser(botWSManager)
	reliablePublisher := services.NewReliableEventPublisher(outboxRepo, botWSManager)
	noticeEmitter := services.NewBotNoticeEmitter(installationRepo, botRepo, reliablePublisher)
	installationService := services.NewInstallationService(installationRepo, botRepo, enrollmentRepo, conversationMessageRepo)
	installationService.SetBotNoticeEmitter(noticeEmitter)
	memberService.SetBotNoticeEmitter(noticeEmitter)
	secretRepo := repository.NewBotAppSecretRepository()
	secretService := services.NewSecretService(secretRepo, botRepo)
	secretHandler := handlers.NewSecretHandler(secretService)
	credentialRepo := repository.NewBotAPICredentialRepository()
	credentialService := services.NewBotAPICredentialService(credentialRepo, botRepo, botWSManager)
	credentialHandler := handlers.NewBotAPICredentialHandler(credentialService)
	botWSHandler := botws.NewHandler(botWSManager, credentialService, botService)
	botActionHandler := handlers.NewBotActionHandler(actionDispatcher)
	botCapabilityHandler := handlers.NewBotCapabilityHandler()
	botEngine.SetSecretResolver(secretService)

	// 注册消息事件 sink
	publisher.RegisterSink("user_ws", services.NewUserWebSocketSink())
	publisher.RegisterSink("workflow", botEngine)
	externalBotSink := services.NewExternalBotSink(installationRepo, botRepo, reliablePublisher)
	publisher.RegisterSink("external_bot", externalBotSink)

	eventRelay := services.NewBotEventRelay(outboxRepo)
	eventRelayCtx, stopEventRelay := context.WithCancel(context.Background())
	go eventRelay.Start(eventRelayCtx)
	authHandler := handlers.NewAuthHandler(authService, cfg.JWT.Secret, cfg.Port == "443" || os.Getenv("FORCE_SECURE_COOKIES") == "true", &cfg.Turnstile)
	chatHandler := handlers.NewChatHandler(authService, userService, conversationService, messageService, friendService, memberService)
	botHandler := handlers.NewBotHandler(botService, botEngine)
	installationHandler := handlers.NewInstallationHandler(installationService)
	settingsRepo := repository.NewSettingsRepository()
	settingsService := services.NewSettingsService(settingsRepo)
	settingsHandler := handlers.NewSettingsHandler(settingsService)
	workflowRepo := repository.NewWorkflowRepository()
	botEngine.SetWorkflowRepo(workflowRepo)
	var tsExecutor services.TSDebugExecutor
	if tsClient := botEngine.GetTSClient(); tsClient != nil {
		tsExecutor = tsClient
	}
	workflowService := services.NewWorkflowService(workflowRepo, botRepo, tsExecutor)
	workflowHandler := handlers.NewWorkflowHandler(workflowService)

	// 初始化WebSocket hub
	writeTimeout, _ := time.ParseDuration(cfg.WebSocket.WriteTimeout)
	readTimeout, _ := time.ParseDuration(cfg.WebSocket.ReadTimeout)
	pingInterval, _ := time.ParseDuration(cfg.WebSocket.PingInterval)
	websocket.InitHub(websocket.HubConfig{
		MaxConnections:     cfg.WebSocket.MaxConnections,
		MaxUserConnections: cfg.WebSocket.MaxUserConnections,
		SendQueueSize:      cfg.WebSocket.SendQueueSize,
		ReadLimit:          cfg.WebSocket.ReadLimit,
		WriteTimeout:       writeTimeout,
		ReadTimeout:        readTimeout,
		PingInterval:       pingInterval,
		AllowedOrigins:     cfg.WebSocket.AllowedOrigins,
		AllowQueryToken:    cfg.WebSocket.AllowQueryToken,
	})
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
		auth.GET("/turnstile-config", authHandler.TurnstileConfig)
		auth.POST("/register", authRateLimit, authHandler.Register)
		auth.POST("/login", authRateLimit, authHandler.Login)
		auth.POST("/logout", handlers.AuthMiddleware(cfg.JWT.Secret), authHandler.Logout)
		auth.GET("/me", handlers.AuthMiddleware(cfg.JWT.Secret), userRateLimit, authHandler.Me)
		auth.PUT("/profile", handlers.AuthMiddleware(cfg.JWT.Secret), userRateLimit, chatHandler.UpdateProfile)
		auth.PUT("/password", handlers.AuthMiddleware(cfg.JWT.Secret), userRateLimit, authHandler.ChangePassword)
		auth.DELETE("/account", handlers.AuthMiddleware(cfg.JWT.Secret), userRateLimit, authHandler.DeleteAccount)
	}

	// 用户路由（per-User 限流）
	users := r.Group("/api/users")
	users.Use(handlers.AuthMiddleware(cfg.JWT.Secret))
	users.Use(userRateLimit)
	{
		users.GET("/search", chatHandler.SearchUsers)
		users.GET("/:id", chatHandler.GetUserByID)
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
		messages.POST("", sensitiveRateLimit, chatHandler.SendMessage)      // 发送消息使用更严格的限流
		messages.POST("/poke", sensitiveRateLimit, chatHandler.PokeMessage) // 拍一拍使用更严格的限流
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

	// WebSocket路由（通过 Cookie/子协议/query 传递 token，不使用 AuthMiddleware）
	r.GET("/api/ws", websocket.HandleWebSocket)
	// Bot Universal WebSocket uses the strict Bot credential middleware and never accepts query credentials.
	r.GET("/api/bot/v1/ws", handlers.BotCredentialAuthMiddleware(credentialService), botWSHandler.Connect)
	r.GET("/api/bot/v1/capabilities", botCapabilityHandler.Get)
	r.GET("/api/bot/v1/health", botWSHandler.Health)
	// Bot HTTP Action endpoint shares the same dispatcher as Universal WS.
	r.POST("/api/bot/v1/actions/:action", handlers.BotCredentialAuthMiddleware(credentialService), botActionHandler.HandleAction)

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
		bots.POST("/:id/workflow/activate", botHandler.ActivateWorkflow)
		bots.POST("/:id/workflow/deactivate", botHandler.DeactivateWorkflow)
		bots.GET("/:id/deployable-conversations", botHandler.GetDeployableConversations)
		bots.POST("/:id/debug", botHandler.DebugBot)
		bots.POST("/:id/debug/step", botHandler.DebugStep)
		bots.POST("/:id/debug/reset", botHandler.DebugReset)
		bots.GET("/:id/call-logs", botHandler.GetBotCallLogs)
		bots.POST("/:id/installations", installationHandler.CreateInstallation)
		bots.GET("/:id/installations", installationHandler.ListByApp)
		// Workflow Document API (#13)
		bots.GET("/:id/workflow", workflowHandler.GetWorkflow)
		bots.PUT("/:id/workflow", workflowHandler.UpdateWorkflow)
		bots.POST("/:id/workflow/validate", workflowHandler.ValidateWorkflow)
		bots.POST("/:id/workflow/publish", workflowHandler.PublishWorkflow)
		bots.GET("/:id/workflow/versions", workflowHandler.ListPublishedVersions)
		bots.POST("/:id/workflow/versions/:revision/rollback", workflowHandler.RollbackWorkflow)
		bots.POST("/:id/workflow/test-runs", workflowHandler.TestRunWorkflow)
		bots.POST("/:id/workflow/test-runs/step", workflowHandler.TestRunStep)
		// Secret 管理(owner-only CRUD,不返回明文)
		bots.GET("/:id/secrets", secretHandler.ListSecrets)
		bots.PUT("/:id/secrets/:key", sensitiveRateLimit, secretHandler.SetSecret)
		bots.DELETE("/:id/secrets/:key", sensitiveRateLimit, secretHandler.DeleteSecret)
		// External Bot credentials: owner JWT management; plaintext is returned only on create/rotate.
		bots.POST("/:id/credentials", sensitiveRateLimit, credentialHandler.Create)
		bots.GET("/:id/credentials", credentialHandler.List)
		bots.POST("/:id/credentials/:credential_id/rotate", sensitiveRateLimit, credentialHandler.Rotate)
		bots.DELETE("/:id/credentials/:credential_id", sensitiveRateLimit, credentialHandler.Revoke)
		bots.GET("/:id/ws-status", botWSHandler.Status)
	}

	// Bot 安装路由(per-User 限流)
	installations := r.Group("/api/installations")
	installations.Use(handlers.AuthMiddleware(cfg.JWT.Secret))
	installations.Use(userRateLimit)
	{
		installations.GET("", installationHandler.ListByTarget)
		installations.GET("/mine", installationHandler.ListMine)
		installations.GET("/:iid", installationHandler.GetInstallation)
		installations.PATCH("/:iid", installationHandler.UpdateInstallation)
		installations.DELETE("/:iid", installationHandler.UninstallInstallation)
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
	stopEventRelay()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := botWSManager.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shut down Bot WebSocket manager:", err)
	}
	websocket.GlobalHub.Shutdown()
}
