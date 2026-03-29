package config

import (
	"fmt"
	"log"
	"os"
)

type Config struct {
	Port      string
	GinMode   string
	DB        DBConfig
	JWT       JWTConfig
	Log       LogConfig
	WebSocket WebSocketConfig
	MinIO     MinIOConfig
}

type LogConfig struct {
	Directory string // 日志文件目录
	MaxFiles  int    // 保留的最大日志文件数量
	MaxLines  int    // 单个日志文件最大行数
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type JWTConfig struct {
	Secret     string
	Expiration string
}

type WebSocketConfig struct {
	MaxConnections     int
	MaxUserConnections int
}

type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	UseSSL          bool
	PublicURL       string // 外部访问地址（客户端下载用）
}

func Load() *Config {
	return &Config{
		Port:    getEnv("PORT", "8080"),
		GinMode: getEnv("GIN_MODE", "debug"),
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "purrchat"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "default_secret_change_me"),
			Expiration: getEnv("JWT_EXPIRATION", "24h"),
		},
		Log: LogConfig{
			Directory: getEnv("LOG_DIR", "logs"),
			MaxFiles:  getEnvInt("LOG_MAX_FILES", 10),
			MaxLines:  getEnvInt("LOG_MAX_LINES", 100000),
		},
		WebSocket: WebSocketConfig{
			MaxConnections:     getEnvInt("MAX_CONNECTIONS", 20000),
			MaxUserConnections: getEnvInt("MAX_USER_CONNECTIONS", 3),
		},
		MinIO: MinIOConfig{
			Endpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKeyID:     getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretAccessKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:          getEnv("MINIO_BUCKET", "purrchat"),
			UseSSL:          getEnv("MINIO_USE_SSL", "false") == "true",
			PublicURL:       getEnv("MINIO_PUBLIC_URL", "http://localhost:9000"),
		},
	}
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var result int
	_, err := fmt.Sscanf(value, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func GetDSN(cfg *DBConfig) string {
	// 添加时区参数，确保数据库使用 UTC 时间存储时间戳
	// 这可以避免时区转换问题，确保前端显示的时间戳一致
	return "postgres://" + cfg.User + ":" + cfg.Password + "@" + cfg.Host + ":" + cfg.Port + "/" + cfg.Name + "?timezone=UTC"
}

func Validate(cfg *Config) {
	if cfg.DB.Password == "" {
		log.Fatal("DB_PASSWORD is required")
	}
	if cfg.JWT.Secret == "default_secret_change_me" {
		log.Println("WARNING: Using default JWT secret. Please change it in production!")
	}
}
