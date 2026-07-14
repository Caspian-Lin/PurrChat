package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"purr-chat-server/internal/botengine"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/services"
	"purr-chat-server/pkg/logger"

	"github.com/gin-gonic/gin"
)

// containsBadInput 判断错误是否属于客户端输入错误（应返回 400）
// 数据库约束冲突、唯一键冲突等也属于客户端可预见错误
func containsBadInput(errMsg string) bool {
	lower := strings.ToLower(errMsg)
	// 唯一键/约束冲突
	if strings.Contains(lower, "duplicate") || strings.Contains(lower, "unique constraint") {
		return true
	}
	// 应用层已明确返回的业务错误
	businessErrors := []string{
		"bot not found", "not the bot owner", "not authorized",
		"this bot is private", "already a member", "invalid",
	}
	for _, s := range businessErrors {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

// BotHandler Bot HTTP 处理器
type BotHandler struct {
	botService *services.BotService
	botEngine  *botengine.BotEngine
}

// NewBotHandler 创建 Bot 处理器
func NewBotHandler(botService *services.BotService, botEngine *botengine.BotEngine) *BotHandler {
	return &BotHandler{botService: botService, botEngine: botEngine}
}

// CreateBot 创建 Bot
// @Summary 创建 Bot
// @Tags Bot
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateBotRequest true "Bot 信息"
// @Success 200 {object} models.MessageResponse
// @Router /api/bots [post]
func (h *BotHandler) CreateBot(c *gin.Context) {
	var req models.CreateBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	bot, err := h.botService.CreateBot(c.Request.Context(), userIDStr, &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create bot: %v", err)
		// 区分客户端错误和服务端内部错误
		errMsg := err.Error()
		if containsBadInput(errMsg) {
			c.JSON(http.StatusBadRequest, models.APIResponse{
				Success: false,
				Message: errMsg,
			})
		} else {
			c.JSON(http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Internal server error",
			})
		}
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Bot created successfully",
		Data:    bot,
	})
}

// GetBot 获取 Bot 详情
// @Summary 获取 Bot 详情
// @Tags Bot
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/{id} [get]
func (h *BotHandler) GetBot(c *gin.Context) {
	botID := c.Param("id")
	if botID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "bot id is required",
		})
		return
	}

	bot, err := h.botService.GetBot(c.Request.Context(), botID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "Bot not found",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    bot,
	})
}

// ListBots 获取用户创建的 Bot 列表
// @Summary 获取用户创建的 Bot 列表
// @Tags Bot
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.MessageResponse
// @Router /api/bots [get]
func (h *BotHandler) ListBots(c *gin.Context) {
	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	bots, err := h.botService.ListBots(c.Request.Context(), userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to get bots",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    dereferenceSlice(bots),
	})
}

// SearchBots 搜索公开 Bot（分页）
// @Summary 搜索公开 Bot
// @Tags Bot
// @Produce json
// @Security BearerAuth
// @Param query query string false "搜索关键词"
// @Param limit query int false "限制数量" default(20)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/search [get]
func (h *BotHandler) SearchBots(c *gin.Context) {
	query := c.Query("query")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	result, err := h.botService.SearchPublicBotsPaginated(c.Request.Context(), query, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to search bots",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    result,
	})
}

// UpdateBot 更新 Bot 配置
// @Summary 更新 Bot 配置
// @Tags Bot
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Param request body models.UpdateBotRequest true "更新 Bot 信息"
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/{id} [put]
func (h *BotHandler) UpdateBot(c *gin.Context) {
	botID := c.Param("id")
	if botID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "bot id is required",
		})
		return
	}

	var req models.UpdateBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	bot, err := h.botService.UpdateBot(c.Request.Context(), botID, userIDStr, &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to update bot: %v", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Bot updated successfully",
		Data:    bot,
	})
}

// DeleteBot 删除 Bot
// @Summary 删除 Bot
// @Tags Bot
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/{id} [delete]
func (h *BotHandler) DeleteBot(c *gin.Context) {
	botID := c.Param("id")
	if botID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "bot id is required",
		})
		return
	}

	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	err := h.botService.DeleteBot(c.Request.Context(), botID, userIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to delete bot: %v", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Bot deleted successfully",
	})
}

// DeployBot 部署 Bot 到会话
// @Summary 部署 Bot 到会话
// @Tags Bot
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Param request body models.DeployBotRequest true "部署信息"
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/{id}/deploy [post]
func (h *BotHandler) DeployBot(c *gin.Context) {
	botID := c.Param("id")
	if botID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "bot id is required",
		})
		return
	}

	var req models.DeployBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	installation, err := h.botService.DeployBot(c.Request.Context(), botID, userIDStr, &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to deploy bot: %v", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Bot deployed successfully",
		Data:    installation,
	})
}

// UndeployBot 从会话移除 Bot
// @Summary 从会话移除 Bot
// @Tags Bot
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Param conversation_id query string true "会话ID"
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/{id}/deploy [delete]
func (h *BotHandler) UndeployBot(c *gin.Context) {
	botID := c.Param("id")
	if botID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "bot id is required",
		})
		return
	}

	conversationID := c.Query("conversation_id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "conversation_id is required",
		})
		return
	}

	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	err := h.botService.UndeployBot(c.Request.Context(), botID, conversationID, userIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to undeploy bot: %v", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Bot undeployed successfully",
	})
}

// GetBotDeployments 获取 Bot 部署列表
// @Summary 获取 Bot 部署列表
// @Tags Bot
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/deployments [get]
func (h *BotHandler) GetBotDeployments(c *gin.Context) {
	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	deployments, err := h.botService.GetBotDeployments(c.Request.Context(), userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to get deployments",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    dereferenceSlice(deployments),
	})
}

// UpdateDeploymentStatus 更新部署状态
// @Summary 更新部署状态
// @Tags Bot
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Param request body models.UpdateDeploymentStatusRequest true "部署状态"
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/{id}/deploy/status [put]
func (h *BotHandler) UpdateDeploymentStatus(c *gin.Context) {
	botID := c.Param("id")
	if botID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "bot id is required",
		})
		return
	}

	var req models.UpdateDeploymentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	err := h.botService.UpdateDeploymentStatus(c.Request.Context(), botID, userIDStr, &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to update deployment status: %v", err)
		respondProtectedResourceError(c, err, "Failed to update deployment status")
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Deployment status updated successfully",
	})
}

// CreateBotConversation 创建与 Bot 的私聊会话
// @Summary 创建与 Bot 的私聊会话
// @Tags Bot
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/{id}/conversation [post]
func (h *BotHandler) CreateBotConversation(c *gin.Context) {
	botID := c.Param("id")
	if botID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "bot id is required",
		})
		return
	}

	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	conversation, err := h.botService.CreateBotConversation(c.Request.Context(), botID, userIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create bot conversation: %v", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Bot conversation created successfully",
		Data:    conversation,
	})
}

// GetDeployableConversations 获取可部署 Bot 的群聊列表
// @Summary 获取可部署 Bot 的群聊列表
// @Tags Bot
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/{id}/deployable-conversations [get]
func (h *BotHandler) GetDeployableConversations(c *gin.Context) {
	botID := c.Param("id")
	if botID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "bot id is required"})
		return
	}

	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	conversations, err := h.botService.GetDeployableConversations(c.Request.Context(), userIDStr, botID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get deployable conversations: %v", err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Failed to get deployable conversations"})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: conversations})
}

// GetConversationBots 获取会话中活跃的 Bot 列表
func (h *BotHandler) GetConversationBots(c *gin.Context) {
	conversationID := c.Param("id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "conversation id is required"})
		return
	}

	requesterID, ok := getUserID(c)
	if !ok {
		return
	}
	deployments, err := h.botService.GetActiveBotsForConversation(c.Request.Context(), requesterID, conversationID)
	if err != nil {
		respondProtectedResourceError(c, err, "Failed to get conversation bots")
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: deployments})
}

// GetBotCallLogs 获取 Bot 调用日志
// @Summary 获取 Bot 调用日志
// @Tags Bot
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Param limit query int false "限制数量" default(20)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/{id}/call-logs [get]
func (h *BotHandler) GetBotCallLogs(c *gin.Context) {
	botID := c.Param("id")
	if botID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "bot id is required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	result, err := h.botService.GetBotCallLogs(c.Request.Context(), botID, userIDStr, limit, offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{Success: true, Data: result})
}
