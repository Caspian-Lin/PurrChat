package handlers

import (
	"net/http"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/services"
	"purr-chat-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ChatHandler 聊天处理器
type ChatHandler struct {
	authService *services.AuthService
	chatService *services.ChatService
}

// NewChatHandler 创建聊天处理器
func NewChatHandler(authService *services.AuthService, chatService *services.ChatService) *ChatHandler {
	return &ChatHandler{
		authService: authService,
		chatService: chatService,
	}
}

// SearchUsers 搜索用户
// @Summary 搜索用户
// @Tags 聊天
// @Produce json
// @Security BearerAuth
// @Param query query string true "搜索查询（UID、手机号或邮箱）"
// @Success 200 {object} models.AuthResponse
// @Router /api/users/search [get]
func (h *ChatHandler) SearchUsers(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		logger.ErrorfWithCaller("Missing query parameter for user search")
		c.JSON(http.StatusBadRequest, models.AuthResponse{
			Success: false,
			Message: "query parameter is required",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		logger.ErrorfWithCaller("Unauthorized access attempt for user search")
		c.JSON(http.StatusUnauthorized, models.AuthResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	users, err := h.authService.SearchUsers(c.Request.Context(), query)
	if err != nil {
		logger.ErrorfWithCaller("Failed to search users with query %s: %v", query, err)
		c.JSON(http.StatusInternalServerError, models.AuthResponse{
			Success: false,
			Message: "Failed to search users",
		})
		return
	}

	// 过滤掉自己
	var filteredUsers []*models.User
	for _, user := range users {
		if user.ID.String() != userID {
			filteredUsers = append(filteredUsers, user)
		}
	}

	logger.InfofWithCaller("User search completed: query=%s, results=%d", query, len(filteredUsers))

	c.JSON(http.StatusOK, models.AuthResponse{
		Success: true,
		Data:    filteredUsers,
	})
}

// UpdateProfile 更新个人资料
// @Summary 更新个人资料
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UpdateProfileRequest true "个人资料信息"
// @Success 200 {object} models.AuthResponse
// @Router /api/profile [put]
func (h *ChatHandler) UpdateProfile(c *gin.Context) {
	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid update profile request: %v", err)
		c.JSON(http.StatusBadRequest, models.AuthResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		logger.ErrorfWithCaller("Unauthorized access attempt for profile update")
		c.JSON(http.StatusUnauthorized, models.AuthResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		logger.ErrorfWithCaller("Invalid user ID type for profile update")
		c.JSON(http.StatusUnauthorized, models.AuthResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	user, err := h.authService.UpdateProfile(c.Request.Context(), userIDStr, &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to update profile for user %s: %v", userIDStr, err)
		c.JSON(http.StatusBadRequest, models.AuthResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Profile updated successfully for user %s", user.Username)

	c.JSON(http.StatusOK, models.AuthResponse{
		Success: true,
		Message: "Profile updated successfully",
		Data:    user,
	})
}

// GetConversations 获取会话列表
// @Summary 获取会话列表
// @Tags 聊天
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.ConversationListResponse
// @Router /api/conversations [get]
func (h *ChatHandler) GetConversations(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		logger.ErrorfWithCaller("Unauthorized access attempt for get conversations")
		c.JSON(http.StatusUnauthorized, models.AuthResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		logger.ErrorfWithCaller("Invalid user ID type for get conversations")
		c.JSON(http.StatusUnauthorized, models.AuthResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	conversations, err := h.chatService.GetConversations(c.Request.Context(), userIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get conversations for user %s: %v", userIDStr, err)
		c.JSON(http.StatusInternalServerError, models.ConversationListResponse{
			Success: false,
			Message: "Failed to get conversations",
		})
		return
	}

	// 转换为切片
	var convSlice []models.Conversation
	for _, conv := range conversations {
		convSlice = append(convSlice, *conv)
	}

	logger.InfofWithCaller("Retrieved %d conversations for user %s", len(convSlice), userIDStr)

	c.JSON(http.StatusOK, models.ConversationListResponse{
		Success: true,
		Data:    convSlice,
	})
}

// GetMessages 获取消息列表
// @Summary 获取消息列表
// @Tags 聊天
// @Produce json
// @Security BearerAuth
// @Param conversation_id query string true "会话ID"
// @Param limit query int false "限制数量" default(50)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} models.MessagesResponse
// @Router /api/messages [get]
func (h *ChatHandler) GetMessages(c *gin.Context) {
	var req models.GetMessagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.ErrorfWithCaller("Invalid get messages request: %v", err)
		c.JSON(http.StatusBadRequest, models.MessagesResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	messages, err := h.chatService.GetMessages(c.Request.Context(), req.ConversationID, req.Limit, req.Offset)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get messages for conversation %s: %v", req.ConversationID, err)
		c.JSON(http.StatusInternalServerError, models.MessagesResponse{
			Success: false,
			Message: "Failed to get messages",
		})
		return
	}

	// 转换为切片
	var msgSlice []models.Message
	for _, msg := range messages {
		msgSlice = append(msgSlice, *msg)
	}

	logger.InfofWithCaller("Retrieved %d messages for conversation %s", len(msgSlice), req.ConversationID)

	c.JSON(http.StatusOK, models.MessagesResponse{
		Success: true,
		Data:    msgSlice,
	})
}

// ExportMessages 导出会话的所有消息
// @Summary 导出会话的所有消息
// @Tags 聊天
// @Produce json
// @Security BearerAuth
// @Param conversation_id query string true "会话ID"
// @Success 200 {object} models.MessagesResponse
// @Router /api/messages/export [get]
func (h *ChatHandler) ExportMessages(c *gin.Context) {
	conversationID := c.Query("conversation_id")
	if conversationID == "" {
		logger.ErrorfWithCaller("Missing conversation_id parameter for export messages")
		c.JSON(http.StatusBadRequest, models.MessagesResponse{
			Success: false,
			Message: "conversation_id is required",
		})
		return
	}

	messages, err := h.chatService.GetAllMessages(c.Request.Context(), conversationID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to export messages for conversation %s: %v", conversationID, err)
		c.JSON(http.StatusInternalServerError, models.MessagesResponse{
			Success: false,
			Message: "Failed to export messages",
		})
		return
	}

	// 转换为切片
	var msgSlice []models.Message
	for _, msg := range messages {
		msgSlice = append(msgSlice, *msg)
	}

	logger.InfofWithCaller("Exported %d messages for conversation %s", len(msgSlice), conversationID)

	c.JSON(http.StatusOK, models.MessagesResponse{
		Success: true,
		Data:    msgSlice,
	})
}

// GetMessagesIncremental 增量获取会话的消息
// @Summary 增量获取会话的消息（从指定时间之后）
// @Tags 聊天
// @Produce json
// @Security BearerAuth
// @Param conversation_id query string true "会话ID"
// @Param since_timestamp query int64 true "起始时间戳（毫秒）"
// @Success 200 {object} models.MessagesResponse
// @Router /api/messages/incremental [get]
func (h *ChatHandler) GetMessagesIncremental(c *gin.Context) {
	var req models.GetMessagesIncrementalRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.ErrorfWithCaller("Invalid get incremental messages request: %v", err)
		c.JSON(http.StatusBadRequest, models.MessagesResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	messages, err := h.chatService.GetMessagesIncremental(c.Request.Context(), req.ConversationID, req.SinceTimestamp)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get incremental messages for conversation %s: %v", req.ConversationID, err)
		c.JSON(http.StatusInternalServerError, models.MessagesResponse{
			Success: false,
			Message: "Failed to get incremental messages",
		})
		return
	}

	// 转换为切片
	var msgSlice []models.Message
	for _, msg := range messages {
		msgSlice = append(msgSlice, *msg)
	}

	logger.InfofWithCaller("Retrieved %d incremental messages for conversation %s", len(msgSlice), req.ConversationID)

	c.JSON(http.StatusOK, models.MessagesResponse{
		Success: true,
		Data:    msgSlice,
	})
}

// SendMessage 发送消息
// @Summary 发送消息
// @Tags 聊天
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.SendMessageRequest true "消息信息"
// @Success 200 {object} models.MessageResponse
// @Router /api/messages [post]
func (h *ChatHandler) SendMessage(c *gin.Context) {
	var req models.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid send message request: %v", err)
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		logger.ErrorfWithCaller("Unauthorized access attempt for send message")
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		logger.ErrorfWithCaller("Invalid user ID type for send message")
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	message, err := h.chatService.SendMessage(c.Request.Context(), userIDStr, &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to send message from user %s to conversation %s: %v", userIDStr, req.ConversationID, err)
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Message sent successfully: ID=%s, ConversationID=%s, SenderID=%s", message.ID, message.ConversationID, message.SenderID)

	c.JSON(http.StatusOK, models.MessageResponse{
		Success: true,
		Message: "Message sent successfully",
		Data:    message,
	})
}

// CreateConversation 创建会话
// @Summary 创建会话
// @Tags 聊天
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.FriendRequest true "目标用户ID"
// @Success 200 {object} models.MessageResponse
// @Router /api/conversations [post]
func (h *ChatHandler) CreateConversation(c *gin.Context) {
	var req models.FriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid create conversation request: %v", err)
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		logger.ErrorfWithCaller("Unauthorized access attempt for create conversation")
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		logger.ErrorfWithCaller("Invalid user ID type for create conversation")
		c.JSON(http.StatusUnauthorized, models.MessageResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	conversation, err := h.chatService.CreateConversation(c.Request.Context(), userIDStr, req.TargetUserID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create conversation between %s and %s: %v", userIDStr, req.TargetUserID, err)
		c.JSON(http.StatusBadRequest, models.MessageResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Conversation created successfully: ID=%s, Name=%s", conversation.ID, conversation.Name)

	c.JSON(http.StatusOK, models.MessageResponse{
		Success: true,
		Message: "Conversation created successfully",
		Data:    conversation,
	})
}

// GetFriends 获取好友列表
// @Summary 获取好友列表
// @Tags 好友
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.FriendListResponse
// @Router /api/friends [get]
func (h *ChatHandler) GetFriends(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		logger.ErrorfWithCaller("Unauthorized access attempt for get friends")
		c.JSON(http.StatusUnauthorized, models.FriendListResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		logger.ErrorfWithCaller("Invalid user ID type for get friends")
		c.JSON(http.StatusUnauthorized, models.FriendListResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	friendships, err := h.chatService.GetFriends(c.Request.Context(), userIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get friends for user %s: %v", userIDStr, err)
		c.JSON(http.StatusInternalServerError, models.FriendListResponse{
			Success: false,
			Message: "Failed to get friends",
		})
		return
	}

	// 转换为切片
	var fsSlice []models.Friendship
	for _, fs := range friendships {
		fsSlice = append(fsSlice, *fs)
	}

	logger.InfofWithCaller("Retrieved %d friends for user %s", len(fsSlice), userIDStr)

	c.JSON(http.StatusOK, models.FriendListResponse{
		Success: true,
		Data:    fsSlice,
	})
}

// GetUserByID 根据ID获取用户信息
// @Summary 根据ID获取用户信息
// @Tags 用户
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户ID"
// @Success 200 {object} models.AuthResponse
// @Router /api/users/{id} [get]
func (h *ChatHandler) GetUserByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		logger.ErrorfWithCaller("Missing user id parameter")
		c.JSON(http.StatusBadRequest, models.AuthResponse{
			Success: false,
			Message: "user id is required",
		})
		return
	}

	// 验证UUID格式
	_, err := uuid.Parse(id)
	if err != nil {
		logger.ErrorfWithCaller("Invalid user ID format: %s", id)
		c.JSON(http.StatusBadRequest, models.AuthResponse{
			Success: false,
			Message: "Invalid user ID format",
		})
		return
	}

	user, err := h.chatService.GetUserByID(c.Request.Context(), id)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get user by ID %s: %v", id, err)
		c.JSON(http.StatusNotFound, models.AuthResponse{
			Success: false,
			Message: "User not found",
		})
		return
	}

	logger.InfofWithCaller("User retrieved: ID=%s, Username=%s", user.ID, user.Username)

	c.JSON(http.StatusOK, models.AuthResponse{
		Success: true,
		Data:    user,
	})
}

// GetUserByUID 根据UID获取用户信息
// @Summary 根据UID获取用户信息
// @Tags 用户
// @Produce json
// @Security BearerAuth
// @Param uid path string true "用户UID"
// @Success 200 {object} models.AuthResponse
// @Router /api/users/uid/{uid} [get]
func (h *ChatHandler) GetUserByUID(c *gin.Context) {
	uidStr := c.Param("uid")
	if uidStr == "" {
		logger.ErrorfWithCaller("Missing user uid parameter")
		c.JSON(http.StatusBadRequest, models.AuthResponse{
			Success: false,
			Message: "user uid is required",
		})
		return
	}

	// 暂时返回错误
	logger.ErrorfWithCaller("GetUserByUID not implemented yet for uid %s", uidStr)
	c.JSON(http.StatusNotImplemented, models.AuthResponse{
		Success: false,
		Message: "Not implemented yet",
	})
}

// SendFriendRequest 发送好友请求
// @Summary 发送好友请求
// @Tags 好友
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.FriendRequest true "好友请求信息"
// @Success 200 {object} models.FriendRequestResponse
// @Router /api/friends/request [post]
func (h *ChatHandler) SendFriendRequest(c *gin.Context) {
	var req models.FriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid send friend request: %v", err)
		c.JSON(http.StatusBadRequest, models.FriendRequestResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		logger.ErrorfWithCaller("Unauthorized access attempt for send friend request")
		c.JSON(http.StatusUnauthorized, models.FriendRequestResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		logger.ErrorfWithCaller("Invalid user ID type for send friend request")
		c.JSON(http.StatusUnauthorized, models.FriendRequestResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	// 发送好友请求（会创建好友关系记录和会话）
	conversation, err := h.chatService.SendFriendRequest(c.Request.Context(), userIDStr, req.TargetUserID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to send friend request from %s to %s: %v", userIDStr, req.TargetUserID, err)
		c.JSON(http.StatusBadRequest, models.FriendRequestResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Friend request sent successfully from %s to %s", userIDStr, req.TargetUserID)

	c.JSON(http.StatusOK, models.FriendRequestResponse{
		Success: true,
		Message: "Friend request sent successfully",
		Data:    conversation,
	})
}

// HandleFriendRequest 处理好友请求
// @Summary 处理好友请求
// @Tags 好友
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.HandleFriendRequestRequest true "处理好友请求信息"
// @Success 200 {object} models.HandleFriendRequestResponse
// @Router /api/friends/handle [post]
func (h *ChatHandler) HandleFriendRequest(c *gin.Context) {
	var req models.HandleFriendRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid handle friend request: %v", err)
		c.JSON(http.StatusBadRequest, models.HandleFriendRequestResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		logger.ErrorfWithCaller("Unauthorized access attempt for handle friend request")
		c.JSON(http.StatusUnauthorized, models.HandleFriendRequestResponse{
			Success: false,
			Message: "Unauthorized",
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		logger.ErrorfWithCaller("Invalid user ID type for handle friend request")
		c.JSON(http.StatusUnauthorized, models.HandleFriendRequestResponse{
			Success: false,
			Message: "Invalid user ID",
		})
		return
	}

	// 验证操作
	if req.Action != "accept" && req.Action != "reject" {
		logger.ErrorfWithCaller("Invalid action for handle friend request: %s", req.Action)
		c.JSON(http.StatusBadRequest, models.HandleFriendRequestResponse{
			Success: false,
			Message: "Invalid action. Must be 'accept' or 'reject'",
		})
		return
	}

	// 处理好友请求
	err := h.chatService.HandleFriendRequest(c.Request.Context(), userIDStr, req.ConversationID.String(), req.Action)
	if err != nil {
		logger.ErrorfWithCaller("Failed to handle friend request: %v", err)
		c.JSON(http.StatusBadRequest, models.HandleFriendRequestResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Friend request %s successfully by user %s", req.Action, userIDStr)

	c.JSON(http.StatusOK, models.HandleFriendRequestResponse{
		Success: true,
		Message: "Friend request " + req.Action + "ed successfully",
	})
}
