package tests

import (
	"context"
	"fmt"
	"os"
	"reflect"
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
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
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
	// 先删除现有的表（注意顺序：先删除有外键约束的表）
	tables := []string{
		"user_settings",
		"enrollments",
		"friendships",
		"conversations",
		"users",
	}

	for _, table := range tables {
		_, err := database.GetPool().Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: Failed to drop table %s: %v", table, err)
		}
	}

	// 删除所有conversation_messages表
	_, err := database.GetPool().Exec(ctx, `
		SELECT 'DROP TABLE IF EXISTS conversation_messages.' || table_name || ' CASCADE'
		FROM information_schema.tables
		WHERE table_schema = 'conversation_messages'
	`)
	if err != nil {
		t.Logf("Warning: Failed to drop conversation_messages tables: %v", err)
	}

	// 创建UID序列
	_, err = database.GetPool().Exec(ctx, `CREATE SEQUENCE IF NOT EXISTS user_uid_seq START WITH 1`)
	if err != nil {
		t.Fatalf("Failed to create user_uid_seq sequence: %v", err)
	}

	// 创建用户表
	_, err = database.GetPool().Exec(ctx, `
		CREATE TABLE users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			uid INTEGER UNIQUE NOT NULL DEFAULT nextval('user_uid_seq'),
			username VARCHAR(40) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			salt VARCHAR(255) NOT NULL,
			avatar_url TEXT,
			email VARCHAR(255) UNIQUE,
			email_verified BOOLEAN DEFAULT FALSE,
			phone VARCHAR(20) UNIQUE,
			phone_verified BOOLEAN DEFAULT FALSE,
			is_bot BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// 创建 username 部分唯一索引（Bot 和普通用户可同名）
	_, err = database.GetPool().Exec(ctx, `CREATE UNIQUE INDEX idx_users_username_unique ON users(username) WHERE is_bot = FALSE`)
	if err != nil {
		t.Fatalf("Failed to create username unique index: %v", err)
	}
	_, err = database.GetPool().Exec(ctx, `CREATE UNIQUE INDEX idx_users_bot_username_unique ON users(username) WHERE is_bot = TRUE`)
	if err != nil {
		t.Fatalf("Failed to create bot username unique index: %v", err)
	}

	// 创建会话表（新结构）
	_, err = database.GetPool().Exec(ctx, `
		CREATE TABLE conversations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			conversation_type VARCHAR(20) NOT NULL DEFAULT 'direct',
			name VARCHAR(100),
			created_by UUID REFERENCES users(id) ON DELETE SET NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT check_conversation_type CHECK (conversation_type IN ('direct', 'group'))
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create conversations table: %v", err)
	}

	// 创建好友关系表
	_, err = database.GetPool().Exec(ctx, `
		CREATE TABLE friendships (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			friend_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, friend_id),
			CONSTRAINT check_status CHECK (status IN ('pending', 'accepted', 'rejected', 'blocked'))
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create friendships table: %v", err)
	}

	// 创建conversation_messages schema
	_, err = database.GetPool().Exec(ctx, `CREATE SCHEMA IF NOT EXISTS conversation_messages`)
	if err != nil {
		t.Fatalf("Failed to create conversation_messages schema: %v", err)
	}

	// 创建enrollments表（用户与会话的多对多关系）
	_, err = database.GetPool().Exec(ctx, `
		CREATE TABLE enrollments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			role VARCHAR(20) DEFAULT 'member',
			joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_read_at TIMESTAMP,
			UNIQUE(conversation_id, user_id),
			CONSTRAINT check_role CHECK (role IN ('owner', 'admin', 'member'))
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create enrollments table: %v", err)
	}

	// 创建user_settings表（用户设置）
	_, err = database.GetPool().Exec(ctx, `
		CREATE TABLE user_settings (
			user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			settings JSONB DEFAULT '{}'::jsonb,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create user_settings table: %v", err)
	}

	// 创建更新时间触发器函数
	_, err = database.GetPool().Exec(ctx, `
		CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql
	`)
	if err != nil {
		t.Fatalf("Failed to create update_updated_at_column function: %v", err)
	}

	// 为conversations表创建更新时间触发器
	_, err = database.GetPool().Exec(ctx, `
		CREATE TRIGGER update_conversations_updated_at
		BEFORE UPDATE ON conversations
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column()
	`)
	if err != nil {
		t.Fatalf("Failed to create update_conversations_updated_at trigger: %v", err)
	}

	// 先删除可能已存在的旧版本函数（返回类型变更无法用 CREATE OR REPLACE）
	_, _ = database.GetPool().Exec(ctx, `DROP FUNCTION IF EXISTS insert_conversation_message(UUID, UUID, TEXT, VARCHAR(20))`)
	_, _ = database.GetPool().Exec(ctx, `DROP FUNCTION IF EXISTS get_conversation_messages(UUID, INT, INT)`)
	_, _ = database.GetPool().Exec(ctx, `DROP FUNCTION IF EXISTS get_conversation_messages_incremental(UUID, TIMESTAMP)`)
	_, _ = database.GetPool().Exec(ctx, `DROP FUNCTION IF EXISTS get_conversation_last_message(UUID)`)
	_, _ = database.GetPool().Exec(ctx, `DROP FUNCTION IF EXISTS create_conversation_message_table(UUID)`)
	_, _ = database.GetPool().Exec(ctx, `DROP FUNCTION IF EXISTS drop_conversation_message_table(UUID)`)

	// 创建用于会话消息表的PostgreSQL函数（与迁移 007 保持同步）
	_, err = database.GetPool().Exec(ctx, `
		CREATE OR REPLACE FUNCTION create_conversation_message_table(conversation_uuid UUID)
		RETURNS VOID AS $$
		DECLARE
			table_name TEXT;
			idx_sender_name TEXT;
			idx_created_at_name TEXT;
		BEGIN
			table_name := replace(conversation_uuid::TEXT, '-', '_');

			EXECUTE format('
				CREATE TABLE IF NOT EXISTS conversation_messages.%I (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					content TEXT NOT NULL,
					msg_type VARCHAR(20) NOT NULL DEFAULT ''text'',
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					bot_id UUID,
					bot_name VARCHAR(100),
					CONSTRAINT check_msg_type CHECK (msg_type IN (''text'', ''image'', ''file'', ''system''))
				)',
			table_name);

			idx_sender_name := 'idx_' || table_name || '_sender_id';
			idx_created_at_name := 'idx_' || table_name || '_created_at';

			EXECUTE format('
				CREATE INDEX IF NOT EXISTS %I ON conversation_messages.%I(sender_id)',
			idx_sender_name, table_name);

			EXECUTE format('
				CREATE INDEX IF NOT EXISTS %I ON conversation_messages.%I(created_at DESC)',
			idx_created_at_name, table_name);
		END;
		$$ LANGUAGE plpgsql
	`)
	if err != nil {
		t.Fatalf("Failed to create create_conversation_message_table function: %v", err)
	}

	_, err = database.GetPool().Exec(ctx, `
		CREATE OR REPLACE FUNCTION drop_conversation_message_table(conversation_uuid UUID)
		RETURNS VOID AS $$
		DECLARE
			table_name TEXT;
		BEGIN
			table_name := replace(conversation_uuid::TEXT, '-', '_');
			EXECUTE format('DROP TABLE IF EXISTS conversation_messages.%I CASCADE', table_name);
		END;
		$$ LANGUAGE plpgsql
	`)
	if err != nil {
		t.Fatalf("Failed to create drop_conversation_message_table function: %v", err)
	}

	_, err = database.GetPool().Exec(ctx, `
		CREATE OR REPLACE FUNCTION insert_conversation_message(
			conversation_uuid UUID,
			sender_uuid UUID,
			msg_content TEXT,
			msg_type VARCHAR(20),
			bot_id UUID DEFAULT NULL,
			bot_name VARCHAR(100) DEFAULT NULL
		)
		RETURNS UUID AS $$
		DECLARE
			new_message_id UUID;
			table_name TEXT;
		BEGIN
			table_name := replace(conversation_uuid::TEXT, '-', '_');
			EXECUTE format('
				INSERT INTO conversation_messages.%I (sender_id, content, msg_type, bot_id, bot_name)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id
			', table_name)
			INTO new_message_id
			USING sender_uuid, msg_content, msg_type, bot_id, bot_name;
			RETURN new_message_id;
		END;
		$$ LANGUAGE plpgsql
	`)
	if err != nil {
		t.Fatalf("Failed to create insert_conversation_message function: %v", err)
	}

	_, err = database.GetPool().Exec(ctx, `
		CREATE OR REPLACE FUNCTION get_conversation_messages(
			conversation_uuid UUID,
			msg_limit INT DEFAULT 50,
			msg_offset INT DEFAULT 0
		)
		RETURNS TABLE (
			id UUID,
			sender_id UUID,
			content TEXT,
			msg_type VARCHAR(20),
			created_at TIMESTAMP,
			bot_id UUID,
			bot_name VARCHAR(100)
		) AS $$
		DECLARE
			table_name TEXT;
		BEGIN
			table_name := replace(conversation_uuid::TEXT, '-', '_');
			RETURN QUERY EXECUTE format('
				SELECT id, sender_id, content, msg_type, created_at, bot_id, bot_name
				FROM conversation_messages.%I
				ORDER BY created_at DESC
				LIMIT $1 OFFSET $2
			', table_name)
			USING msg_limit, msg_offset;
		END;
		$$ LANGUAGE plpgsql
	`)
	if err != nil {
		t.Fatalf("Failed to create get_conversation_messages function: %v", err)
	}

	_, err = database.GetPool().Exec(ctx, `
		CREATE OR REPLACE FUNCTION get_conversation_messages_incremental(
			conversation_uuid UUID,
			since_timestamp TIMESTAMP
		)
		RETURNS TABLE (
			id UUID,
			sender_id UUID,
			content TEXT,
			msg_type VARCHAR(20),
			created_at TIMESTAMP,
			bot_id UUID,
			bot_name VARCHAR(100)
		) AS $$
		DECLARE
			table_name TEXT;
		BEGIN
			table_name := replace(conversation_uuid::TEXT, '-', '_');
			RETURN QUERY EXECUTE format('
				SELECT id, sender_id, content, msg_type, created_at, bot_id, bot_name
				FROM conversation_messages.%I
				WHERE created_at > $1
				ORDER BY created_at ASC
			', table_name)
			USING since_timestamp;
		END;
		$$ LANGUAGE plpgsql
	`)
	if err != nil {
		t.Fatalf("Failed to create get_conversation_messages_incremental function: %v", err)
	}

	_, err = database.GetPool().Exec(ctx, `
		CREATE OR REPLACE FUNCTION get_conversation_message_count(conversation_uuid UUID)
		RETURNS BIGINT AS $$
		DECLARE
			table_name TEXT;
			message_count BIGINT;
		BEGIN
			table_name := replace(conversation_uuid::TEXT, '-', '_');
			EXECUTE format('
				SELECT COUNT(*)
				FROM conversation_messages.%I
			', table_name)
			INTO message_count;

			RETURN message_count;
		END;
		$$ LANGUAGE plpgsql
	`)
	if err != nil {
		t.Fatalf("Failed to create get_conversation_message_count function: %v", err)
	}

	_, err = database.GetPool().Exec(ctx, `
		CREATE OR REPLACE FUNCTION get_conversation_last_message(conversation_uuid UUID)
		RETURNS TABLE (
			id UUID,
			sender_id UUID,
			content TEXT,
			msg_type VARCHAR(20),
			created_at TIMESTAMP,
			bot_id UUID,
			bot_name VARCHAR(100)
		) AS $$
		DECLARE
			table_name TEXT;
		BEGIN
			table_name := replace(conversation_uuid::TEXT, '-', '_');
			RETURN QUERY EXECUTE format('
				SELECT id, sender_id, content, msg_type, created_at, bot_id, bot_name
				FROM conversation_messages.%I
				ORDER BY created_at DESC
				LIMIT 1
			', table_name);
		END;
		$$ LANGUAGE plpgsql
	`)
	if err != nil {
		t.Fatalf("Failed to create get_conversation_last_message function: %v", err)
	}
}

// SetupTestRouter 设置测试路由
func SetupTestRouter() {
	gin.SetMode(gin.TestMode)
	testRouter = gin.New()

	// 注册自定义 UUID 验证器
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
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

	// 初始化依赖
	userRepo := repository.NewUserRepository()
	conversationRepo := repository.NewConversationRepository()
	friendshipRepo := repository.NewFriendshipRepository()
	enrollmentRepo := repository.NewEnrollmentRepository()
	conversationMessageRepo := repository.NewConversationMessageRepository()

	authService := services.NewAuthService(userRepo, repository.NewBotRepository(), jwtSecret)
	conversationService := services.NewConversationService(userRepo, conversationRepo, enrollmentRepo, conversationMessageRepo, friendshipRepo)
	messageService := services.NewMessageService(userRepo, conversationRepo, enrollmentRepo, conversationMessageRepo, nil, nil)
	friendService := services.NewFriendService(userRepo, friendshipRepo, enrollmentRepo, conversationMessageRepo)
	memberService := services.NewMemberService(userRepo, conversationRepo, enrollmentRepo)
	userService := services.NewUserService(userRepo)

	authHandler = handlers.NewAuthHandler(authService, jwtSecret, false, nil)
	chatHandler = handlers.NewChatHandler(authService, userService, conversationService, messageService, friendService, memberService)

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

	// 设置服务
	settingsRepo := repository.NewSettingsRepository()
	settingsService := services.NewSettingsService(settingsRepo)
	settingsHandler := handlers.NewSettingsHandler(settingsService)

	testRouter.GET("/api/settings", handlers.AuthMiddleware(jwtSecret), settingsHandler.GetSettings)
	testRouter.PUT("/api/settings", handlers.AuthMiddleware(jwtSecret), settingsHandler.UpdateSettings)
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

	// 删除所有conversation_messages表
	_, err := database.GetPool().Exec(ctx, `
		SELECT 'DROP TABLE IF EXISTS conversation_messages.' || table_name || ' CASCADE'
		FROM information_schema.tables
		WHERE table_schema = 'conversation_messages'
	`)
	if err != nil {
		t.Logf("Warning: Failed to drop conversation_messages tables: %v", err)
	}

	// 清理表数据（注意顺序：先清理有外键约束的表）
	tables := []string{
		"user_settings",
		"enrollments",
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
	_, err = database.GetPool().Exec(ctx, "ALTER SEQUENCE user_uid_seq RESTART WITH 1")
	if err != nil {
		t.Logf("Warning: Failed to reset sequence: %v", err)
	}
}

// CreateTestUser 创建测试用户
func CreateTestUser(t *testing.T, username, email, password string) *models.User {
	ctx := context.Background()

	userRepo := repository.NewUserRepository()

	// 生成唯一的 phone 值（基于 username，保持 VARCHAR(20) 以内）
	phone := "1" + username

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
