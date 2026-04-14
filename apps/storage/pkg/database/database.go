package database

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

// Init 初始化数据库连接池
func Init(dsn string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	Pool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}

	// 验证连接
	if err = Pool.Ping(ctx); err != nil {
		return err
	}

	log.Println("Database connection pool initialized successfully")
	return nil
}

// Close 关闭数据库连接池
func Close() {
	if Pool != nil {
		Pool.Close()
		log.Println("Database connection pool closed")
	}
}

// GetPool 获取数据库连接池
func GetPool() *pgxpool.Pool {
	return Pool
}
