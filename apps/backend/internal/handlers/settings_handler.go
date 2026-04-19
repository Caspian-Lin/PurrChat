package handlers

import (
    "net/http"

    "purr-chat-server/internal/models"
    "purr-chat-server/internal/services"
    "purr-chat-server/pkg/logger"

    "github.com/gin-gonic/gin"
)

// SettingsHandler 设置处理器
type SettingsHandler struct {
    settingsService *services.SettingsService
}

// NewSettingsHandler 创建设置处理器
func NewSettingsHandler(settingsService *services.SettingsService) *SettingsHandler {
    return &SettingsHandler{settingsService: settingsService}
}

// GetSettings 获取用户设置
// @Summary 获取用户设置
// @Tags 设置
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.AuthResponse
// @Router /api/settings [get]
func (h *SettingsHandler) GetSettings(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, models.AuthResponse{
            Success: false,
            Message: "Unauthorized",
        })
        return
    }

    userIDStr, ok := userID.(string)
    if !ok {
        c.JSON(http.StatusUnauthorized, models.AuthResponse{
            Success: false,
            Message: "Invalid user ID",
        })
        return
    }

    settings, err := h.settingsService.GetSettings(c.Request.Context(), userIDStr)
    if err != nil {
        logger.ErrorfWithCaller("Failed to get settings for user %s: %v", userIDStr, err)
        c.JSON(http.StatusInternalServerError, models.AuthResponse{
            Success: false,
            Message: "Failed to get settings",
        })
        return
    }

    c.JSON(http.StatusOK, models.AuthResponse{
        Success: true,
        Data:    settings,
    })
}

// UpdateSettings 更新用户设置
// @Summary 更新用户设置
// @Tags 设置
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UpdateSettingsRequest true "设置数据"
// @Success 200 {object} models.AuthResponse
// @Router /api/settings [put]
func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
    var req models.UpdateSettingsRequest
    if err := c.ShouldBindJSON(&req); err != nil {
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

    userIDStr, ok := userID.(string)
    if !ok {
        c.JSON(http.StatusUnauthorized, models.AuthResponse{
            Success: false,
            Message: "Invalid user ID",
        })
        return
    }

    updated, err := h.settingsService.UpdateSettings(c.Request.Context(), userIDStr, req.Settings)
    if err != nil {
        logger.ErrorfWithCaller("Failed to update settings for user %s: %v", userIDStr, err)
        c.JSON(http.StatusInternalServerError, models.AuthResponse{
            Success: false,
            Message: "Failed to update settings",
        })
        return
    }

    c.JSON(http.StatusOK, models.AuthResponse{
        Success: true,
        Message: "Settings updated successfully",
        Data:    updated,
    })
}
