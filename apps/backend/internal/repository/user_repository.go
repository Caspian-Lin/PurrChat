package repository

import (
	"context"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/database"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

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
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string, salt string) error
}

type userRepository struct {
}

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
        INSERT INTO users (id, username, password_hash, salt,avatar_url, email, email_verified, phone, phone_verified, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
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

// FindByUsername 根据用户名查找用户
func (r *userRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
        SELECT id, uid, username, password_hash, salt, avatar_url, email, email_verified, phone, phone_verified, created_at
        FROM users
        WHERE username = $1
    `

	user := &models.User{}
	err := database.GetPool().QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.UID,
		&user.Username,
		&user.PasswordHash,
		&user.Salt,
		&user.AvatarURL,
		&user.Email,
		&user.EmailVerified,
		&user.Phone,
		&user.PhoneVerified,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// FindByID 根据ID查找用户
func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
        SELECT id, uid, username, password_hash, salt, avatar_url, email, email_verified, phone, phone_verified, created_at
        FROM users
        WHERE id = $1
    `

	user := &models.User{}
	err := database.GetPool().QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.UID,
		&user.Username,
		&user.PasswordHash,
		&user.Salt,
		&user.AvatarURL,
		&user.Email,
		&user.EmailVerified,
		&user.Phone,
		&user.PhoneVerified,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// FindByUID 根据UID查找用户
func (r *userRepository) FindByUID(ctx context.Context, uid int) (*models.User, error) {
	query := `
        SELECT id, uid, username, password_hash, salt, avatar_url, email, email_verified, phone, phone_verified, created_at
        FROM users
        WHERE uid = $1
    `

	user := &models.User{}
	err := database.GetPool().QueryRow(ctx, query, uid).Scan(
		&user.ID,
		&user.UID,
		&user.Username,
		&user.PasswordHash,
		&user.Salt,
		&user.AvatarURL,
		&user.Email,
		&user.EmailVerified,
		&user.Phone,
		&user.PhoneVerified,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// FindByEmail 根据邮箱查找用户
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
        SELECT id, uid, username, password_hash, salt, avatar_url, email, email_verified, phone, phone_verified, created_at
        FROM users
        WHERE email = $1
    `

	user := &models.User{}
	err := database.GetPool().QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.UID,
		&user.Username,
		&user.PasswordHash,
		&user.Salt,
		&user.AvatarURL,
		&user.Email,
		&user.EmailVerified,
		&user.Phone,
		&user.PhoneVerified,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// FindByPhone 根据手机号查找用户
func (r *userRepository) FindByPhone(ctx context.Context, phone string) (*models.User, error) {
	query := `
        SELECT id, uid, username, password_hash, salt, avatar_url, email, email_verified, phone, phone_verified, created_at
        FROM users
        WHERE phone = $1
    `

	user := &models.User{}
	err := database.GetPool().QueryRow(ctx, query, phone).Scan(
		&user.ID,
		&user.UID,
		&user.Username,
		&user.PasswordHash,
		&user.Salt,
		&user.AvatarURL,
		&user.Email,
		&user.EmailVerified,
		&user.Phone,
		&user.PhoneVerified,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// Search 搜索用户（通过UID、手机号、邮箱的模糊搜索，取并集）
func (r *userRepository) Search(ctx context.Context, query string) ([]*models.User, error) {
	logger.InfofWithCaller("Search called with query: '%s'", query)

	// 构建查询：对 uid、手机号、邮箱分别进行模糊搜索（LIKE），然后取并集
	dbQuery := `
        SELECT DISTINCT id, uid, username, password_hash, salt, avatar_url, email, email_verified, phone, phone_verified, created_at
        FROM users
        WHERE
            CAST(uid AS TEXT) LIKE $1 OR
            email LIKE $1 OR
            phone LIKE $1
        LIMIT 20
    `

	// 添加通配符实现模糊搜索
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
		err := rows.Scan(
			&user.ID,
			&user.UID,
			&user.Username,
			&user.PasswordHash,
			&user.Salt,
			&user.AvatarURL,
			&user.Email,
			&user.EmailVerified,
			&user.Phone,
			&user.PhoneVerified,
			&user.CreatedAt,
		)
		if err != nil {
			logger.ErrorfWithCaller("Row scan failed: %v", err)
			return nil, err
		}
		users = append(users, user)
		logger.InfofWithCaller("Found user: ID=%s, UID=%d, Username=%s, Email=%s, Phone=%s", user.ID, user.UID, user.Username, user.Email, user.Phone)
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
        WHERE id = $7
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

// UpdatePassword 更新用户密码
func (r *userRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string, salt string) error {
	logger.InfofWithCaller("Updating password for user ID: %s", userID)

	query := `UPDATE users SET password_hash = $1, salt = $2 WHERE id = $3`

	_, err := database.GetPool().Exec(ctx, query, passwordHash, salt, userID)

	if err != nil {
		logger.ErrorfWithCaller("Failed to update password for user %s: %v", userID, err)
	} else {
		logger.InfofWithCaller("Password updated successfully for user ID: %s", userID)
	}

	return err
}
