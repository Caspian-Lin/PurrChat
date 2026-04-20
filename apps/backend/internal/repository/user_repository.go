package repository

import (
	"context"
	"database/sql"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/database"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// 用户查询的 SELECT 列（排除 password_hash 和 salt）
const userSelectCols = `id, uid, username, avatar_url, email, email_verified, phone, phone_verified, is_bot, created_at`

// scanUser 将查询行扫描到 User 结构体（不包含 password_hash 和 salt）
// 使用 sql.NullString 处理可能为 NULL 的列（Bot 用户没有 email/phone）
func scanUser(rows pgx.Rows, user *models.User) error {
	var avatarURL, email, phone sql.NullString
	if err := rows.Scan(
		&user.ID,
		&user.UID,
		&user.Username,
		&avatarURL,
		&email,
		&user.EmailVerified,
		&phone,
		&user.PhoneVerified,
		&user.IsBot,
		&user.CreatedAt,
	); err != nil {
		return err
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if email.Valid {
		user.Email = email.String
	}
	if phone.Valid {
		user.Phone = phone.String
	}
	return nil
}

// 包含 password_hash 和 salt 的完整列（仅内部使用）
const userSelectColsWithAuth = `id, uid, username, password_hash, salt, avatar_url, email, email_verified, phone, phone_verified, is_bot, created_at`

func scanUserRowWithAuth(row pgx.Row, user *models.User) error {
	var avatarURL, email, phone sql.NullString
	if err := row.Scan(
		&user.ID,
		&user.UID,
		&user.Username,
		&user.PasswordHash,
		&user.Salt,
		&avatarURL,
		&email,
		&user.EmailVerified,
		&phone,
		&user.PhoneVerified,
		&user.IsBot,
		&user.CreatedAt,
	); err != nil {
		return err
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if email.Valid {
		user.Email = email.String
	}
	if phone.Valid {
		user.Phone = phone.String
	}
	return nil
}

// UserRepository 用户仓储接口
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByUsername(ctx context.Context, username string) (*models.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	FindByUID(ctx context.Context, uid int) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByPhone(ctx context.Context, phone string) (*models.User, error)
	Search(ctx context.Context, query string) ([]*models.User, error)
	Update(ctx context.Context, user *models.User) error
	UpdateBotProfile(ctx context.Context, userID uuid.UUID, username, avatarURL string) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string, salt string) error
}

type userRepository struct{}

// NewUserRepository 创建用户仓储
func NewUserRepository() UserRepository {
	return &userRepository{}
}

// Create 创建用户
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	logger.InfofWithCaller("Creating user: %s", user.Username)

	user.ID = uuid.New()
	user.CreatedAt = time.Now().UTC()

	query := `
        INSERT INTO users (id, username, password_hash, salt, avatar_url, email, email_verified, phone, phone_verified, is_bot, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        RETURNING id, uid, created_at
    `

	err := pgx.BeginTxFunc(ctx, database.GetPool(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			user.ID,
			user.Username,
			user.PasswordHash,
			user.Salt,
			user.AvatarURL,
			user.Email,
			user.EmailVerified,
			user.Phone,
			user.PhoneVerified,
			user.IsBot,
			user.CreatedAt,
		).Scan(&user.ID, &user.UID, &user.CreatedAt)
	})

	if err != nil {
		logger.ErrorfWithCaller("Failed to create user %s: %v", user.Username, err)
	} else {
		logger.InfofWithCaller("User created successfully: %s (ID: %s)", user.Username, user.ID)
	}

	return err
}

// FindByUsername 根据用户名查找用户（含密码信息，用于认证）
func (r *userRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `SELECT ` + userSelectColsWithAuth + ` FROM users WHERE username = $1`

	user := &models.User{}
	err := scanUserRowWithAuth(database.GetPool().QueryRow(ctx, query, username), user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// FindByID 根据ID查找用户
func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `SELECT ` + userSelectColsWithAuth + ` FROM users WHERE id = $1`

	user := &models.User{}
	err := scanUserRowWithAuth(database.GetPool().QueryRow(ctx, query, id), user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// FindByUID 根据UID查找用户
func (r *userRepository) FindByUID(ctx context.Context, uid int) (*models.User, error) {
	query := `SELECT ` + userSelectColsWithAuth + ` FROM users WHERE uid = $1`

	user := &models.User{}
	err := scanUserRowWithAuth(database.GetPool().QueryRow(ctx, query, uid), user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// FindByEmail 根据邮箱查找用户（含密码信息，用于认证）
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT ` + userSelectColsWithAuth + ` FROM users WHERE email = $1`

	user := &models.User{}
	err := scanUserRowWithAuth(database.GetPool().QueryRow(ctx, query, email), user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// FindByPhone 根据手机号查找用户
func (r *userRepository) FindByPhone(ctx context.Context, phone string) (*models.User, error) {
	query := `SELECT ` + userSelectColsWithAuth + ` FROM users WHERE phone = $1`

	user := &models.User{}
	err := scanUserRowWithAuth(database.GetPool().QueryRow(ctx, query, phone), user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Search 搜索用户（通过UID、用户名、手机号、邮箱的模糊搜索，包含 Bot）
func (r *userRepository) Search(ctx context.Context, query string) ([]*models.User, error) {
	logger.InfofWithCaller("Search called with query: '%s'", query)

	dbQuery := `
        SELECT DISTINCT ` + userSelectCols + `
        FROM users
        WHERE
            CAST(uid AS TEXT) LIKE $1 OR
            username LIKE $1 OR
            email LIKE $1 OR
            phone LIKE $1
        LIMIT 20
    `

	searchPattern := "%" + query + "%"

	logger.InfofWithCaller("Executing search query with LIKE pattern: '%s'", searchPattern)

	rows, err := database.GetPool().Query(ctx, dbQuery, searchPattern)
	if err != nil {
		logger.ErrorfWithCaller("Query execution failed: %v", err)
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		if err := scanUser(rows, user); err != nil {
			logger.ErrorfWithCaller("Row scan failed: %v", err)
			return nil, err
		}
		users = append(users, user)
		logger.InfofWithCaller("Found user: ID=%s, UID=%d, Username=%s, IsBot=%v", user.ID, user.UID, user.Username, user.IsBot)
	}

	logger.InfofWithCaller("Search completed with %d results for query '%s'", len(users), query)

	return users, nil
}

// Update 更新用户信息
func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	logger.InfofWithCaller("Updating user: %s (ID: %s)", user.Username, user.ID)

	query := `
        UPDATE users
        SET username = $1, avatar_url = $2, email = $3, email_verified = $4, phone = $5, phone_verified = $6
        WHERE id = $7 AND is_bot = FALSE
    `

	_, err := database.GetPool().Exec(ctx, query,
		user.Username,
		user.AvatarURL,
		user.Email,
		user.EmailVerified,
		user.Phone,
		user.PhoneVerified,
		user.ID,
	)

	if err != nil {
		logger.ErrorfWithCaller("Failed to update user %s: %v", user.ID, err)
	} else {
		logger.InfofWithCaller("User updated successfully: %s (ID: %s)", user.Username, user.ID)
	}

	return err
}

// UpdateBotProfile 更新 Bot 用户的名字和头像（同步 users 表）
func (r *userRepository) UpdateBotProfile(ctx context.Context, userID uuid.UUID, username, avatarURL string) error {
	query := `
        UPDATE users
        SET username = $1, avatar_url = $2
        WHERE id = $3 AND is_bot = TRUE
    `

	_, err := database.GetPool().Exec(ctx, query, username, avatarURL, userID)
	return err
}

// UpdatePassword 更新用户密码
func (r *userRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string, salt string) error {
	logger.InfofWithCaller("Updating password for user ID: %s", userID)

	query := `UPDATE users SET password_hash = $1, salt = $2 WHERE id = $3 AND is_bot = FALSE`

	_, err := database.GetPool().Exec(ctx, query, passwordHash, salt, userID)

	if err != nil {
		logger.ErrorfWithCaller("Failed to update password for user %s: %v", userID, err)
	} else {
		logger.InfofWithCaller("Password updated successfully for user ID: %s", userID)
	}

	return err
}
