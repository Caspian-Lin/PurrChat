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
	Storage   StorageConfig
}

type LogConfig struct {
	Directory string
	MaxFiles  int
	MaxLines  int
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type JWTConfig struct {
	Secret string
}

type StorageConfig struct {
	Provider string // "minio" or "r2"
	Endpoint string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	UseSSL          bool
	PublicURL       string
	Region          string // R2 专用
}

func Load() *Config {
	provider := getEnv("STORAGE_PROVIDER", "minio")
	cfg := &Config{
		Port:    getEnv("STORAGE_PORT", "8081"),
		GinMode: getEnv("GIN_MODE", "debug"),
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "purrchat"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "default_secret_change_me"),
		},
		Log: LogConfig{
			Directory: getEnv("LOG_DIR", "logs"),
			MaxFiles:  getEnvInt("LOG_MAX_FILES", 10),
			MaxLines:  getEnvInt("LOG_MAX_LINES", 100000),
		},
	}

	switch provider {
	case "r2":
		accountID := getEnv("R2_ACCOUNT_ID", "")
		if accountID == "" {
			log.Fatal("R2_ACCOUNT_ID is required when STORAGE_PROVIDER=r2")
		}
		cfg.Storage = StorageConfig{
			Provider:        "r2",
			Endpoint:        fmt.Sprintf("%s.r2.cloudflarestorage.com", accountID),
			AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
			Bucket:          getEnv("R2_BUCKET", "purrchat"),
			UseSSL:          true,
			PublicURL:       getEnv("R2_PUBLIC_URL", ""),
			Region:          getEnv("R2_REGION", "auto"),
		}
	default:
		cfg.Storage = StorageConfig{
			Provider:        "minio",
			Endpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKeyID:     getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretAccessKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:          getEnv("MINIO_BUCKET", "purrchat"),
			UseSSL:          getEnv("MINIO_USE_SSL", "false") == "true",
			PublicURL:       getEnv("MINIO_PUBLIC_URL", "http://localhost:9000"),
		}
	}

	return cfg
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
