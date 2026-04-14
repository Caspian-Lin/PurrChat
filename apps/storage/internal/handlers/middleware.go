package handlers

import (
	"net/http"
	"strings"
	"time"

	"purr-chat-storage/pkg/jwt"
	"purr-chat-storage/pkg/logger"

	"github.com/gin-gonic/gin"
)

// LoggingMiddleware 记录所有API请求的中间件
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.DisableConsoleOutput()
		defer logger.EnableConsoleOutput()

		startTime := time.Now()

		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		userID, _ := c.Get("user_id")

		logger.InfofWithCaller("API Request: Method=%s, Path=%s, ClientIP=%s, UserID=%v, UserAgent=%s",
			method, path, clientIP, userID, userAgent)

		c.Next()

		latency := time.Since(startTime)
		statusCode := c.Writer.Status()

		logger.InfofWithCaller("API Response: Method=%s, Path=%s, Status=%d, Latency=%v, UserID=%v",
			method, path, statusCode, latency, userID)
	}
}

// AuthMiddleware JWT 认证中间件
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.ErrorfWithCaller("Missing Authorization header for %s %s", c.Request.Method, c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		userID, err := jwt.ExtractUserID(tokenString, jwtSecret)
		if err != nil {
			logger.ErrorfWithCaller("Invalid token for %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		logger.InfofWithCaller("User %s authenticated for %s %s", userID, c.Request.Method, c.Request.URL.Path)
		c.Set("user_id", userID)
		c.Next()
	}
}
