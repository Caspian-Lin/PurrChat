package handlers

import (
	"net/http"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/services"
	"purr-chat-server/pkg/logger"

	"github.com/gin-gonic/gin"
)

// InstallationHandler Bot 安装 HTTP 处理器
type InstallationHandler struct {
	installationService *services.InstallationService
}

// NewInstallationHandler 创建安装处理器
func NewInstallationHandler(installationService *services.InstallationService) *InstallationHandler {
	return &InstallationHandler{installationService: installationService}
}

// CreateInstallation 安装 Bot
// @Summary 安装 Bot 到用户私聊或群聊
// @Tags Bot
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Param request body models.CreateInstallationRequest true "安装信息"
// @Success 200 {object} models.APIResponse
// @Router /api/bots/{id}/installations [post]
func (h *InstallationHandler) CreateInstallation(c *gin.Context) {
	appID := c.Param("id")
	var req models.CreateInstallationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid request: " + err.Error()})
		return
	}

	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	inst, err := h.installationService.CreateInstallation(c.Request.Context(), userIDStr, appID, &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create installation: %v", err)
		if containsBadInput(err.Error()) {
			c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Bot installed successfully",
		Data:    gin.H{"installation": inst},
	})
}

// GetInstallation 获取安装详情
// @Summary 获取安装详情
// @Tags Bot
// @Security BearerAuth
// @Param iid path string true "Installation ID"
// @Router /api/installations/{iid} [get]
func (h *InstallationHandler) GetInstallation(c *gin.Context) {
	requesterID, ok := getUserID(c)
	if !ok {
		return
	}
	inst, err := h.installationService.GetInstallation(c.Request.Context(), requesterID, c.Param("iid"))
	if err != nil {
		respondProtectedResourceError(c, err, "Failed to get installation")
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: gin.H{"installation": inst}})
}

// ListByApp 列出某 Bot 的所有安装(仅 owner)
// @Summary 列出 Bot 的安装
// @Tags Bot
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Router /api/bots/{id}/installations [get]
func (h *InstallationHandler) ListByApp(c *gin.Context) {
	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}
	list, err := h.installationService.ListByApp(c.Request.Context(), userIDStr, c.Param("id"))
	if err != nil {
		if containsBadInput(err.Error()) {
			c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Internal server error"})
		}
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    gin.H{"installations": list, "total": len(list)},
	})
}

// ListByTarget 列出某目标的安装
// @Summary 列出目标(用户/会话)的安装
// @Tags Bot
// @Security BearerAuth
// @Param target_type query string true "user 或 conversation"
// @Param target_id query string true "目标 ID"
// @Router /api/installations [get]
func (h *InstallationHandler) ListByTarget(c *gin.Context) {
	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}
	targetType := models.InstallationTargetType(c.Query("target_type"))
	targetID := c.Query("target_id")
	if targetType == "" || targetID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "target_type and target_id are required"})
		return
	}
	list, err := h.installationService.ListByTarget(c.Request.Context(), userIDStr, targetType, targetID)
	if err != nil {
		if containsBadInput(err.Error()) {
			c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Internal server error"})
		}
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    gin.H{"installations": list, "total": len(list)},
	})
}

// ListMine 列出当前用户作为安装者的安装
// @Summary 列出我的安装
// @Tags Bot
// @Security BearerAuth
// @Router /api/installations/mine [get]
func (h *InstallationHandler) ListMine(c *gin.Context) {
	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}
	list, err := h.installationService.ListMine(c.Request.Context(), userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    gin.H{"installations": list, "total": len(list)},
	})
}

// UpdateInstallation 更新安装(暂停/恢复/重新授权)
// @Summary 更新安装
// @Tags Bot
// @Security BearerAuth
// @Param iid path string true "Installation ID"
// @Param request body models.UpdateInstallationRequest true "更新信息"
// @Router /api/installations/{iid} [patch]
func (h *InstallationHandler) UpdateInstallation(c *gin.Context) {
	var req models.UpdateInstallationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid request: " + err.Error()})
		return
	}
	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}
	inst, err := h.installationService.UpdateInstallation(c.Request.Context(), userIDStr, c.Param("iid"), &req)
	if err != nil {
		if containsBadInput(err.Error()) {
			c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Internal server error"})
		}
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Installation updated",
		Data:    gin.H{"installation": inst},
	})
}

// UninstallInstallation 卸载安装
// @Summary 卸载 Bot
// @Tags Bot
// @Security BearerAuth
// @Param iid path string true "Installation ID"
// @Router /api/installations/{iid} [delete]
func (h *InstallationHandler) UninstallInstallation(c *gin.Context) {
	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}
	if err := h.installationService.UninstallInstallation(c.Request.Context(), userIDStr, c.Param("iid")); err != nil {
		if containsBadInput(err.Error()) {
			c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Internal server error"})
		}
		return
	}
	c.JSON(http.StatusOK, models.APIResponse{Success: true, Message: "Bot uninstalled"})
}
