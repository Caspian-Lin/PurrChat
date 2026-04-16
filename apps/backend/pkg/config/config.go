package config

import (
	"fmt"
	"log"
	"os"
)

type Config struct {
	Port       string
	GinMode    string
	DB         DBConfig
	JWT        JWTConfig
	Log        LogConfig
	WebSocket  WebSocketConfig
	RateLimit  RateLimitConfig
}

type RateLimitConfig struct {
	// 全局 per-IP 速率限制
	GlobalRatePerSec float64 // 每秒允许的请求数
	GlobalBurst      int     // 突发最大请求数
	// 认证端点 per-IP 速率限制（register/login）
	AuthRatePerSec float64
	AuthBurst      int
	// 已认证用户速率限制（per-user，退回 per-IP）
	UserRatePerSec float64
	UserBurst      int
	// 敏感操作速率限制（好友请求、消息发送等）
	SensitiveRatePerSec float64
	SensitiveBurst      int
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
	Secret     string
	Expiration string
}

type WebSocketConfig struct {
	MaxConnections     int
	MaxUserConnections int
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
		RateLimit: RateLimitConfig{
			GlobalRatePerSec:    getEnvFloat("RATE_LIMIT_GLOBAL_RPS", 2),     // 2 req/s, ~120 req/min
			GlobalBurst:         getEnvInt("RATE_LIMIT_GLOBAL_BURST", 60),    // 允许 60 个突发
			AuthRatePerSec:      getEnvFloat("RATE_LIMIT_AUTH_RPS", 0.2),     // 每 5 秒 1 次
			AuthBurst:           getEnvInt("RATE_LIMIT_AUTH_BURST", 5),       // 允许 5 次突发
			UserRatePerSec:      getEnvFloat("RATE_LIMIT_USER_RPS", 2),       // 2 req/s
			UserBurst:           getEnvInt("RATE_LIMIT_USER_BURST", 60),      // 允许 60 个突发
			SensitiveRatePerSec: getEnvFloat("RATE_LIMIT_SENSITIVE_RPS", 0.5), // 每 2 秒 1 次
			SensitiveBurst:      getEnvInt("RATE_LIMIT_SENSITIVE_BURST", 10), // 允许 10 次突发
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

func getEnvFloat(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var result float64
	_, err := fmt.Sscanf(value, "%f", &result)
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
