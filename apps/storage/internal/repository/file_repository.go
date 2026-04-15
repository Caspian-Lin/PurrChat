package repository

import (
	"context"
	"time"

	"purr-chat-storage/internal/models"
	"purr-chat-storage/pkg/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FileRepository 文件元数据仓库
type FileRepository interface {
	Create(ctx context.Context, meta *models.FileMetadata) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.FileMetadata, error)
	GetByObjectKey(ctx context.Context, objectKey string) (*models.FileMetadata, error)
	GetConfirmedByUploaderAndCategory(ctx context.Context, uploaderID uuid.UUID, category string) (*models.FileMetadata, error)
	ConfirmUpload(ctx context.Context, id uuid.UUID, etag *string, publicURL *string) error
	DeleteByID(ctx context.Context, id uuid.UUID) error
	DeleteUnconfirmedBefore(ctx context.Context, before time.Time) (int64, error)
}

type fileRepository struct {
	pool *pgxpool.Pool
}

// NewFileRepository 创建文件仓库
func NewFileRepository() FileRepository {
	return &fileRepository{pool: database.GetPool()}
}

func (r *fileRepository) Create(ctx context.Context, meta *models.FileMetadata) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO file_metadata (id, object_key, file_name, file_size, content_type, category, usage, uploader_id, confirmed)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, false)`,
		meta.ID, meta.ObjectKey, meta.FileName, meta.FileSize, meta.ContentType, meta.Category, meta.Usage, meta.UploaderID)
	return err
}

func (r *fileRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.FileMetadata, error) {
	var meta models.FileMetadata
	err := r.pool.QueryRow(ctx,
		`SELECT id, object_key, file_name, file_size, content_type, category, usage, uploader_id,
			        public_url, etag, confirmed, created_at, confirmed_at
			 FROM file_metadata WHERE id = $1`, id).Scan(
		&meta.ID, &meta.ObjectKey, &meta.FileName, &meta.FileSize, &meta.ContentType,
		&meta.Category, &meta.Usage, &meta.UploaderID, &meta.PublicURL, &meta.ETag,
		&meta.Confirmed, &meta.CreatedAt, &meta.ConfirmedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func (r *fileRepository) GetByObjectKey(ctx context.Context, objectKey string) (*models.FileMetadata, error) {
	var meta models.FileMetadata
	err := r.pool.QueryRow(ctx,
		`SELECT id, object_key, file_name, file_size, content_type, category, usage, uploader_id,
			        public_url, etag, confirmed, created_at, confirmed_at
			 FROM file_metadata WHERE object_key = $1`, objectKey).Scan(
		&meta.ID, &meta.ObjectKey, &meta.FileName, &meta.FileSize, &meta.ContentType,
		&meta.Category, &meta.Usage, &meta.UploaderID, &meta.PublicURL, &meta.ETag,
		&meta.Confirmed, &meta.CreatedAt, &meta.ConfirmedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

// GetConfirmedByUploaderAndCategory 查询用户指定分类下最新的已确认文件
func (r *fileRepository) GetConfirmedByUploaderAndCategory(ctx context.Context, uploaderID uuid.UUID, category string) (*models.FileMetadata, error) {
	var meta models.FileMetadata
	err := r.pool.QueryRow(ctx,
		`SELECT id, object_key, file_name, file_size, content_type, category, usage, uploader_id,
				        public_url, etag, confirmed, created_at, confirmed_at
				 FROM file_metadata WHERE uploader_id = $1 AND category = $2 AND confirmed = true
				 ORDER BY confirmed_at DESC LIMIT 1`, uploaderID, category).Scan(
		&meta.ID, &meta.ObjectKey, &meta.FileName, &meta.FileSize, &meta.ContentType,
		&meta.Category, &meta.Usage, &meta.UploaderID, &meta.PublicURL, &meta.ETag,
		&meta.Confirmed, &meta.CreatedAt, &meta.ConfirmedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func (r *fileRepository) ConfirmUpload(ctx context.Context, id uuid.UUID, etag *string, publicURL *string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE file_metadata SET confirmed = true, etag = $1, public_url = $2, confirmed_at = $3
			 WHERE id = $4 AND confirmed = false`,
		etag, publicURL, time.Now().UTC(), id)
	return err
}

func (r *fileRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM file_metadata WHERE id = $1`, id)
	return err
}

// DeleteUnconfirmedBefore 删除指定时间之前未确认的文件记录
func (r *fileRepository) DeleteUnconfirmedBefore(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM file_metadata WHERE confirmed = false AND created_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
