package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"purr-chat-server/internal/handlers"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"
	"purr-chat-server/pkg/database"
	"purr-chat-server/pkg/jwt"
	"purr-chat-server/pkg/logger"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	testRouter  *gin.Engine
	authHandler *handlers.AuthHandler
	chatHandler *handlers.ChatHandler
	jwtSecret   = "test_jwt_secret_key_for_testing_only"
)

// SetupTestDB 设置测试数据库（使用PostgreSQL）
func SetupTestDB(t *testing.T) {
	// 从环境变量获取数据库连接信息
	dbHost := os.Getenv("TEST_DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("TEST_DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("TEST_DB_USER")
	if dbUser == "" {
		dbUser = "testuser"
	}
	dbPassword := os.Getenv("TEST_DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "testpass"
	}
	dbName := os.Getenv("TEST_DB_NAME")
	if dbName == "" {
		dbName = "testdb"
	}

	// 构建连接字符串
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// 初始化数据库连接
	ctx := context.Background()
	err := database.Init(dsn)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// 清理之前测试的数据
	CleanupTestTables(t)

	// 创建表结构
	CreateTestTables(t, ctx)
}

// CreateTestTables 创建测试表
func CreateTestTables(t *testing.T, ctx context.Context) {
	// 创建用户表
	_, err := database.GetPool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			uid SERIAL UNIQUE,
			username VARCHAR(20) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			salt VARCHAR(255) NOT NULL,
			avatar_url TEXT,
			email VARCHAR(255) UNIQUE,
			email_verified BOOLEAN DEFAULT FALSE,
			phone VARCHAR(20) UNIQUE,
			phone_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// 创建会话表
	_, err = database.GetPool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS conversations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			conversation_type VARCHAR(20) NOT NULL DEFAULT 'stranger',
			user1_id UUID NOT NULL,
			user2_id UUID NOT NULL,
			has_pending_request BOOLEAN DEFAULT FALSE,
			request_status VARCHAR(20) DEFAULT 'none',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user1_id, user2_id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create conversations table: %v", err)
	}

	// 创建消息表
	_, err = database.GetPool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS messages (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			conversation_id UUID NOT NULL,
			sender_id UUID NOT NULL,
			content TEXT NOT NULL,
			msg_type VARCHAR(20) NOT NULL DEFAULT 'text',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create messages table: %v", err)
	}

	// 创建好友关系表
	_, err = database.GetPool().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS friendships (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL,
			friend_id UUID NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, friend_id)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create friendships table: %v", err)
	}
}

// SetupTestRouter 设置测试路由
func SetupTestRouter() {
	gin.SetMode(gin.TestMode)
	testRouter = gin.New()

	// 初始化依赖
	userRepo := repository.NewUserRepository()
	conversationRepo := repository.NewConversationRepository()
	messageRepo := repository.NewMessageRepository()
	friendshipRepo := repository.NewFriendshipRepository()

	authService := services.NewAuthService(userRepo, jwtSecret)
	chatService := services.NewChatService(userRepo, conversationRepo, messageRepo, friendshipRepo)

	authHandler = handlers.NewAuthHandler(authService, jwtSecret)
	chatHandler = handlers.NewChatHandler(authService, chatService)

	// 配置路由
	testRouter.POST("/api/register", authHandler.Register)
	testRouter.POST("/api/login", authHandler.Login)
	testRouter.GET("/api/me", handlers.AuthMiddleware(jwtSecret), authHandler.Me)
	testRouter.PUT("/api/profile", handlers.AuthMiddleware(jwtSecret), chatHandler.UpdateProfile)

	testRouter.GET("/api/users/search", handlers.AuthMiddleware(jwtSecret), chatHandler.SearchUsers)
	testRouter.GET("/api/users/:id", handlers.AuthMiddleware(jwtSecret), chatHandler.GetUserByID)

	testRouter.GET("/api/conversations", handlers.AuthMiddleware(jwtSecret), chatHandler.GetConversations)
	testRouter.POST("/api/conversations", handlers.AuthMiddleware(jwtSecret), chatHandler.CreateConversation)

	testRouter.GET("/api/messages", handlers.AuthMiddleware(jwtSecret), chatHandler.GetMessages)
	testRouter.POST("/api/messages", handlers.AuthMiddleware(jwtSecret), chatHandler.SendMessage)

	testRouter.GET("/api/friends", handlers.AuthMiddleware(jwtSecret), chatHandler.GetFriends)
	testRouter.POST("/api/friends/request", handlers.AuthMiddleware(jwtSecret), chatHandler.SendFriendRequest)
	testRouter.POST("/api/friends/handle", handlers.AuthMiddleware(jwtSecret), chatHandler.HandleFriendRequest)
}

// CleanupTestDB 清理测试数据库
func CleanupTestDB(t *testing.T) {
	// 清理数据库连接
	if database.GetPool() != nil {
		database.Close()
	}
}

// CleanupTestTables 清理测试表中的所有数据
func CleanupTestTables(t *testing.T) {
	ctx := context.Background()

	// 清理表数据（注意顺序：先清理有外键约束的表）
	tables := []string{
		"messages",
		"friendships",
		"conversations",
		"users",
	}

	for _, table := range tables {
		_, err := database.GetPool().Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Logf("Warning: Failed to cleanup table %s: %v", table, err)
		}
	}

	// 重置序列
	_, err := database.GetPool().Exec(ctx, "ALTER SEQUENCE users_uid_seq RESTART WITH 1")
	if err != nil {
		t.Logf("Warning: Failed to reset sequence: %v", err)
	}
}

// CreateTestUser 创建测试用户
func CreateTestUser(t *testing.T, username, email, password string) *models.User {
	ctx := context.Background()

	userRepo := repository.NewUserRepository()

	// 生成唯一的 phone 值（基于 username）
	phone := "test_" + username + "_phone"

	user := &models.User{
		Username:      username,
		PasswordHash:  password,
		Salt:          "test_salt",
		AvatarURL:     "",
		Email:         email,
		EmailVerified: false,
		Phone:         phone,
		PhoneVerified: false,
	}

	err := userRepo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// 清除密码相关字段
	user.PasswordHash = ""
	user.Salt = ""

	return user
}

// GetAuthToken 获取认证令牌
func GetAuthToken(t *testing.T, userID string) string {
	token, err := jwt.GenerateToken(userID, jwtSecret, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	return token
}

// TestMain 测试主函数
func TestMain(m *testing.M) {
	// 初始化日志
	logger.Init()

	// 运行测试
	code := m.Run()

	// 清理
	CleanupTestDB(nil)

	os.Exit(code)
}
