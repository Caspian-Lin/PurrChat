package handlers

import (
	"time"

	"purr-chat-server/pkg/logger"

	"github.com/gin-gonic/gin"
)

// LoggingMiddleware 记录所有API请求的中间件
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 禁用终端输出，因为Gin框架已经输出了
		logger.DisableConsoleOutput()
		defer logger.EnableConsoleOutput()

		// 记录请求开始时间
		startTime := time.Now()

		// 获取请求信息
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// 获取用户ID（如果已认证）
		userID, _ := c.Get("user_id")

		// 记录请求信息（只写入文件，不输出到终端）
		logger.InfofWithCaller("API Request: Method=%s, Path=%s, ClientIP=%s, UserID=%v, UserAgent=%s",
			method, path, clientIP, userID, userAgent)

		// 处理请求
		c.Next()

		// 记录响应信息（只写入文件，不输出到终端）
		latency := time.Since(startTime)
		statusCode := c.Writer.Status()

		logger.InfofWithCaller("API Response: Method=%s, Path=%s, Status=%d, Latency=%v, UserID=%v",
			method, path, statusCode, latency, userID)
	}
}
