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
	"github.com/google/uuid"
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
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	bot, err := h.botService.CreateBot(c.Request.Context(), userIDStr, &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create bot: %v", err)
		// 区分客户端错误和服务端内部错误
		errMsg := err.Error()
		if containsBadInput(errMsg) {
			c.JSON(http.StatusBadRequest, models.MessageResponse{
				Success: false,
				Message: errMsg,
			})
		} else {
			c.JSON(http.StatusInternalServerError, models.MessageResponse{
				Success: false,
				Message: "Internal server error",
			})
		}
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{
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
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: "bot id is required",
		})
		return
	}

	bot, err := h.botService.GetBot(c.Request.Context(), botID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.MessageResponse{
			Success: false,
			Message: "Bot not found",
		})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{
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
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	bots, err := h.botService.ListBots(c.Request.Context(), userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.MessageResponse{
			Success: false,
			Message: "Failed to get bots",
		})
		return
	}

	var botSlice []models.Bot
	for _, bot := range bots {
		botSlice = append(botSlice, *bot)
	}

	c.JSON(http.StatusOK, models.MessageResponse{
		Success: true,
		Data:    botSlice,
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
		c.JSON(http.StatusInternalServerError, models.MessageResponse{
			Success: false,
			Message: "Failed to search bots",
		})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{
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
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: "bot id is required",
		})
		return
	}

	var req models.UpdateBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	bot, err := h.botService.UpdateBot(c.Request.Context(), botID, userIDStr, &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to update bot: %v", err)
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{
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
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: "bot id is required",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	err := h.botService.DeleteBot(c.Request.Context(), botID, userIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to delete bot: %v", err)
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{
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
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: "bot id is required",
		})
		return
	}

	var req models.DeployBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	deployment, err := h.botService.DeployBot(c.Request.Context(), botID, userIDStr, &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to deploy bot: %v", err)
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{
		Success: true,
		Message: "Bot deployed successfully",
		Data:    deployment,
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
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: "bot id is required",
		})
		return
	}

	conversationID := c.Query("conversation_id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: "conversation_id is required",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	err := h.botService.UndeployBot(c.Request.Context(), botID, conversationID, userIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to undeploy bot: %v", err)
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{
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
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	deployments, err := h.botService.GetBotDeployments(c.Request.Context(), userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.MessageResponse{
			Success: false,
			Message: "Failed to get deployments",
		})
		return
	}

	var depSlice []models.BotDeployment
	for _, d := range deployments {
		depSlice = append(depSlice, *d)
	}

	c.JSON(http.StatusOK, models.MessageResponse{
		Success: true,
		Data:    depSlice,
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
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: "bot id is required",
		})
		return
	}

	var req models.UpdateDeploymentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	err := h.botService.UpdateDeploymentStatus(c.Request.Context(), botID, userIDStr, &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to update deployment status: %v", err)
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{
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
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: "bot id is required",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	conversation, err := h.botService.CreateBotConversation(c.Request.Context(), botID, userIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create bot conversation: %v", err)
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{
		Success: true,
		Message: "Bot conversation created successfully",
		Data:    conversation,
	})
}

// ActivateSpecialMode 激活特殊模式
// @Summary 激活 Bot 特殊模式
// @Tags Bot
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Param request body models.ActivateSpecialModeRequest true "激活信息"
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/{id}/special-mode/activate [post]
func (h *BotHandler) ActivateSpecialMode(c *gin.Context) {
	botID := c.Param("id")
	if botID == "" {
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: "bot id is required"})
		return
	}

	var req models.ActivateSpecialModeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: "Invalid request: " + err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{Success: false, Message: "Unauthorized"})
		return
	}
	userIDStr, _ := userID.(string)

	// 验证权限
	err := h.botService.ActivateSpecialMode(c.Request.Context(), botID, userIDStr, req.ConversationID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to activate special mode: %v", err)
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: err.Error()})
		return
	}

	// 调用引擎激活特殊模式
	botUUID, _ := uuid.Parse(botID)
	err = h.botEngine.ActivateSpecialMode(c.Request.Context(), botUUID, req.ConversationID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to activate special mode (engine): %v", err)
		c.JSON(http.StatusInternalServerError, models.MessageResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{Success: true, Message: "Special mode activated"})
}

// DeactivateSpecialMode 停用特殊模式
// @Summary 停用 Bot 特殊模式
// @Tags Bot
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Param request body models.ActivateSpecialModeRequest true "停用信息"
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/{id}/special-mode/deactivate [post]
func (h *BotHandler) DeactivateSpecialMode(c *gin.Context) {
	botID := c.Param("id")
	if botID == "" {
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: "bot id is required"})
		return
	}

	var req models.ActivateSpecialModeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: "Invalid request: " + err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{Success: false, Message: "Unauthorized"})
		return
	}
	userIDStr, _ := userID.(string)

	err := h.botService.DeactivateSpecialMode(c.Request.Context(), botID, userIDStr, req.ConversationID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to deactivate special mode: %v", err)
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: err.Error()})
		return
	}

	// 调用引擎停用特殊模式
	botUUID, _ := uuid.Parse(botID)
	err = h.botEngine.DeactivateSpecialMode(c.Request.Context(), botUUID, req.ConversationID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to deactivate special mode (engine): %v", err)
		c.JSON(http.StatusInternalServerError, models.MessageResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{Success: true, Message: "Special mode deactivated"})
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
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: "bot id is required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.MessageResponse{Success: false, Message: "Unauthorized"})
		return
	}
	userIDStr, _ := userID.(string)

	conversations, err := h.botService.GetDeployableConversations(c.Request.Context(), userIDStr, botID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get deployable conversations: %v", err)
		c.JSON(http.StatusInternalServerError, models.MessageResponse{Success: false, Message: "Failed to get deployable conversations"})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{Success: true, Data: conversations})
}

// DebugBot 调试执行 Bot 事件链
// @Summary 调试执行 Bot 事件链
// @Tags Bot
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Param request body models.DebugBotRequest true "调试请求"
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/{id}/debug [post]
func (h *BotHandler) DebugBot(c *gin.Context) {
	botIDStr := c.Param("id")
	if botIDStr == "" {
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: "bot id is required"})
		return
	}

	var req models.DebugBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: "Invalid request: " + err.Error()})
		return
	}

	botID, err := uuid.Parse(botIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: "invalid bot id"})
		return
	}

	result, err := h.botEngine.DebugExecute(c.Request.Context(), botID, &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to debug bot: %v", err)
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{Success: true, Data: result})
}

// DebugStep 调试逐步执行
// @Summary 调试逐步执行下一个事件
// @Tags Bot
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Param request body models.DebugStepRequest true "逐步执行请求"
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/{id}/debug/step [post]
func (h *BotHandler) DebugStep(c *gin.Context) {
	botIDStr := c.Param("id")
	if botIDStr == "" {
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: "bot id is required"})
		return
	}

	var req models.DebugStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: "Invalid request: " + err.Error()})
		return
	}

	botID, err := uuid.Parse(botIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: "invalid bot id"})
		return
	}

	result, err := h.botEngine.DebugStep(c.Request.Context(), botID, req.SessionID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to debug step: %v", err)
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{Success: true, Data: result})
}

// DebugReset 重置调试会话
// @Summary 重置调试会话
// @Tags Bot
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Bot ID"
// @Param request body models.DebugResetRequest true "重置请求"
// @Success 200 {object} models.MessageResponse
// @Router /api/bots/{id}/debug/reset [post]
func (h *BotHandler) DebugReset(c *gin.Context) {
	var req models.DebugResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: "Invalid request: " + err.Error()})
		return
	}

	h.botEngine.DebugReset(req.SessionID)

	c.JSON(http.StatusOK, models.MessageResponse{Success: true, Message: "Debug session reset"})
}

// GetConversationBots 获取会话中活跃的 Bot 列表
func (h *BotHandler) GetConversationBots(c *gin.Context) {
	conversationID := c.Param("id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, models.MessageResponse{Success: false, Message: "conversation id is required"})
		return
	}

	deployments, err := h.botService.GetActiveBotsForConversation(c.Request.Context(), conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.MessageResponse{Success: false, Message: "Failed to get conversation bots"})
		return
	}

	c.JSON(http.StatusOK, models.MessageResponse{Success: true, Data: deployments})
}
