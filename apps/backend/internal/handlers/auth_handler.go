package handlers

import (
    "net/http"
    "strings"

    "purr-chat-server/internal/models"
    "purr-chat-server/internal/services"
    "purr-chat-server/pkg/jwt"
    "purr-chat-server/pkg/logger"

    "github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
    authService *services.AuthService
    jwtSecret   string
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService *services.AuthService, jwtSecret string) *AuthHandler {
    return &AuthHandler{
        authService: authService,
        jwtSecret:   jwtSecret,
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

    c.JSON(http.StatusOK, models.AuthResponse{
        Success: true,
        Message: "Login successful",
        Data:    resp,
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

// AuthMiddleware JWT认证中间件
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
