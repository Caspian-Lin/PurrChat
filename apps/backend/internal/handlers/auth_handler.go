package handlers

import (
	"net/http"
	"strings"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/services"
	"purr-chat-server/pkg/config"
	"purr-chat-server/pkg/cookie"
	"purr-chat-server/pkg/jwt"
	"purr-chat-server/pkg/logger"
	"purr-chat-server/pkg/turnstile"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService  *services.AuthService
	jwtSecret    string
	isSecure     bool
	turnstileCfg *config.TurnstileConfig
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService *services.AuthService, jwtSecret string, isSecure bool, turnstileCfg *config.TurnstileConfig) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		jwtSecret:    jwtSecret,
		isSecure:     isSecure,
		turnstileCfg: turnstileCfg,
	}
}

// Register 用户注册
// @Summary 用户注册
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "注册信息"
// @Success 200 {object} models.AuthResponse
// @Router /api/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, models.AuthResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// Turnstile 人机验证（启用时强制要求）
	if h.turnstileCfg != nil && h.turnstileCfg.Enabled {
		if req.TurnstileToken == "" {
			c.JSON(http.StatusBadRequest, models.AuthResponse{
				Success: false,
				Message: "人机验证 token 不能为空",
			})
			return
		}
		result, err := turnstile.Verify(
			h.turnstileCfg.SecretKey,
			req.TurnstileToken,
			c.ClientIP(),
		)
		if err != nil || !result.Success {
			logger.ErrorfWithCaller("Turnstile verification failed: %v, error codes: %v", err, result.ErrorCodes)
			c.JSON(http.StatusForbidden, models.AuthResponse{
				Success: false,
				Message: "人机验证失败，请重试",
			})
			return
		}
	}

	resp, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		logger.ErrorfWithCaller("Registration failed for username %s: %v", req.Username, err)
		c.JSON(http.StatusBadRequest, models.AuthResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("User registered successfully: %s (ID: %s)", resp.User.Username, resp.User.ID)

	// SEC-006: 通过 HttpOnly Cookie 设置 token
	cookie.SetAuthCookie(c.Writer, resp.Token, h.isSecure)

	c.JSON(http.StatusOK, models.AuthResponse{
		Success: true,
		Message: "Registration successful",
		Data:    resp,
	})
}

// Login 用户登录
// @Summary 用户登录
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "登录信息"
// @Success 200 {object} models.AuthResponse
// @Router /api/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid login request: %v", err)
		c.JSON(http.StatusBadRequest, models.AuthResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		logger.ErrorfWithCaller("Login failed for email %s: %v", req.Email, err)
		c.JSON(http.StatusUnauthorized, models.AuthResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("User logged in successfully: %s (ID: %s)", resp.User.Username, resp.User.ID)

	// SEC-006: 通过 HttpOnly Cookie 设置 token
	cookie.SetAuthCookie(c.Writer, resp.Token, h.isSecure)

	c.JSON(http.StatusOK, models.AuthResponse{
		Success: true,
		Message: "Login successful",
		Data:    resp,
	})
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	cookie.ClearAuthCookie(c.Writer)
	c.JSON(http.StatusOK, models.AuthResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}

// TurnstileConfig 返回 Turnstile 公开配置（site_key）
func (h *AuthHandler) TurnstileConfig(c *gin.Context) {
	if h.turnstileCfg == nil || !h.turnstileCfg.Enabled {
		c.JSON(http.StatusOK, gin.H{
			"enabled": false,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled":  true,
		"site_key": h.turnstileCfg.SiteKey,
	})
}

// Me 获取当前用户信息
// @Summary 获取当前用户信息
// @Tags 认证
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.AuthResponse
// @Router /api/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		logger.ErrorfWithCaller("Unauthorized access attempt: missing user_id")
		c.JSON(http.StatusUnauthorized, models.AuthResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		logger.ErrorfWithCaller("Failed to get user by ID %s: %v", userID, err)
		c.JSON(http.StatusNotFound, models.AuthResponse{
			Success: false,
			Message: "User not found",
		})
		return
	}

	logger.InfofWithCaller("User info retrieved: %s (ID: %s)", user.Username, user.ID)

	c.JSON(http.StatusOK, models.AuthResponse{
		Success: true,
		Data:    user,
	})
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ChangePasswordRequest true "修改密码信息"
// @Success 200 {object} models.AuthResponse
// @Router /api/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid change password request: %v", err)
		c.JSON(http.StatusBadRequest, models.AuthResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.AuthResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	err := h.authService.ChangePassword(c.Request.Context(), userID.(string), &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to change password: %v", err)
		c.JSON(http.StatusBadRequest, models.AuthResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Password changed successfully for user: %s", userID)
	c.JSON(http.StatusOK, models.AuthResponse{
		Success: true,
		Message: "Password changed successfully",
	})
}

// DeleteAccount 注销用户账号
// @Summary 注销用户账号
// @Tags 认证
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.DeleteAccountRequest true "注销确认（需输入密码）"
// @Success 200 {object} models.AuthResponse
// @Router /api/account [delete]
func (h *AuthHandler) DeleteAccount(c *gin.Context) {
	var req models.DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid delete account request: %v", err)
		c.JSON(http.StatusBadRequest, models.AuthResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.AuthResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	err := h.authService.DeleteAccount(c.Request.Context(), userID.(string), &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to delete account for user %s: %v", userID, err)
		status := http.StatusBadRequest
		if err.Error() == "user not found" {
			status = http.StatusNotFound
		}
		c.JSON(status, models.AuthResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Account deleted successfully for user: %s", userID)

	// 清除认证 Cookie
	cookie.ClearAuthCookie(c.Writer)

	c.JSON(http.StatusOK, models.AuthResponse{
		Success: true,
		Message: "Account deleted successfully",
	})
}

// AuthMiddleware JWT认证中间件
// 支持 Bearer header 和 HttpOnly Cookie 两种认证方式
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 优先从 Authorization header 获取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// 回退到 Cookie
			if token, ok := cookie.GetTokenFromCookie(c.Request); ok {
				tokenString = token
			}
		}

		if tokenString == "" {
			logger.ErrorfWithCaller("Missing auth token for %s %s", c.Request.Method, c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			c.Abort()
			return
		}

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
