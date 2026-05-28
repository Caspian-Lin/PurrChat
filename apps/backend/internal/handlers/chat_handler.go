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
	authService         *services.AuthService
	userService         *services.UserService
	conversationService *services.ConversationService
	messageService      *services.MessageService
	friendService       *services.FriendService
	memberService       *services.MemberService
}

// NewChatHandler 创建聊天处理器
func NewChatHandler(
	authService *services.AuthService,
	userService *services.UserService,
	conversationService *services.ConversationService,
	messageService *services.MessageService,
	friendService *services.FriendService,
	memberService *services.MemberService,
) *ChatHandler {
	return &ChatHandler{
		authService:         authService,
		userService:         userService,
		conversationService: conversationService,
		messageService:      messageService,
		friendService:       friendService,
		memberService:       memberService,
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

	if len(query) > 50 {
		c.JSON(http.StatusBadRequest, models.AuthResponse{
			Success: false,
			Message: "搜索关键词过长，最多50个字符",
		})
		return
	}

	userIDStr, ok := getUserID(c)
	if !ok {
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
		if user.ID.String() != userIDStr {
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

	userIDStr, ok := getUserID(c)
	if !ok {
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
// @Success 200 {object} models.APIResponse
// @Router /api/conversations [get]
func (h *ChatHandler) GetConversations(c *gin.Context) {
	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	conversations, err := h.conversationService.GetConversations(c.Request.Context(), userIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get conversations for user %s: %v", userIDStr, err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to get conversations",
		})
		return
	}

	convSlice := dereferenceSlice(conversations)

	logger.InfofWithCaller("Retrieved %d conversations for user %s", len(convSlice), userIDStr)

	c.JSON(http.StatusOK, models.APIResponse{
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
// @Success 200 {object} models.APIResponse
// @Router /api/messages [get]
func (h *ChatHandler) GetMessages(c *gin.Context) {
	var req models.GetMessagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.ErrorfWithCaller("Invalid get messages request: %v", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	messages, err := h.messageService.GetMessages(c.Request.Context(), req.ConversationID, req.Limit, req.Offset)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get messages for conversation %s: %v", req.ConversationID, err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to get messages",
		})
		return
	}

	msgSlice := dereferenceSlice(messages)

	logger.InfofWithCaller("Retrieved %d messages for conversation %s", len(msgSlice), req.ConversationID)

	c.JSON(http.StatusOK, models.APIResponse{
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
// @Success 200 {object} models.APIResponse
// @Router /api/messages/export [get]
func (h *ChatHandler) ExportMessages(c *gin.Context) {
	conversationID := c.Query("conversation_id")
	if conversationID == "" {
		logger.ErrorfWithCaller("Missing conversation_id parameter for export messages")
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "conversation_id is required",
		})
		return
	}

	messages, err := h.messageService.ExportMessages(c.Request.Context(), conversationID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to export messages for conversation %s: %v", conversationID, err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to export messages",
		})
		return
	}

	msgSlice := dereferenceSlice(messages)

	logger.InfofWithCaller("Exported %d messages for conversation %s", len(msgSlice), conversationID)

	c.JSON(http.StatusOK, models.APIResponse{
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
// @Success 200 {object} models.APIResponse
// @Router /api/messages/incremental [get]
func (h *ChatHandler) GetMessagesIncremental(c *gin.Context) {
	var req models.GetMessagesIncrementalRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.ErrorfWithCaller("Invalid get incremental messages request: %v", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	messages, err := h.messageService.GetMessagesIncremental(c.Request.Context(), req.ConversationID, req.SinceTimestamp)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get incremental messages for conversation %s: %v", req.ConversationID, err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to get incremental messages",
		})
		return
	}

	msgSlice := dereferenceSlice(messages)

	logger.InfofWithCaller("Retrieved %d incremental messages for conversation %s", len(msgSlice), req.ConversationID)

	c.JSON(http.StatusOK, models.APIResponse{
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
// @Success 200 {object} models.APIResponse
// @Router /api/messages [post]
func (h *ChatHandler) SendMessage(c *gin.Context) {
	var req models.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid send message request: %v", err)
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

	message, err := h.messageService.SendMessage(c.Request.Context(), userIDStr, &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to send message from user %s to conversation %s: %v", userIDStr, req.ConversationID, err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Message sent successfully: ID=%s, ConversationID=%s, SenderID=%s", message.ID, message.ConversationID, message.SenderID)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Message sent successfully",
		Data:    message,
	})
}

// PokeRequest 拍一拍请求
type PokeRequest struct {
	ConversationID string `json:"conversation_id" binding:"required,uuid"`
	TargetUserID   string `json:"target_user_id" binding:"required,uuid"`
}

// PokeMessage 拍一拍
// @Summary 拍一拍
// @Tags 聊天
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body PokeRequest true "拍一拍请求"
// @Success 200 {object} models.APIResponse
// @Router /api/messages/poke [post]
func (h *ChatHandler) PokeMessage(c *gin.Context) {
	var req PokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid poke request: %v", err)
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

	conversationUUID, err := uuid.Parse(req.ConversationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid conversation_id",
		})
		return
	}

	targetUserUUID, err := uuid.Parse(req.TargetUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid target_user_id",
		})
		return
	}

	message, err := h.messageService.SendPokeMessage(c.Request.Context(), userIDStr, conversationUUID, targetUserUUID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to send poke from user %s to user %s: %v", userIDStr, req.TargetUserID, err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Poke sent successfully: ID=%s, SenderID=%s, TargetUserID=%s", message.ID, userIDStr, req.TargetUserID)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Poke sent successfully",
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
// @Success 200 {object} models.APIResponse
// @Router /api/conversations [post]
func (h *ChatHandler) CreateConversation(c *gin.Context) {
	var req models.FriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid create conversation request: %v", err)
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

	conversation, err := h.conversationService.CreateConversation(c.Request.Context(), userIDStr, req.TargetUserID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create conversation between %s and %s: %v", userIDStr, req.TargetUserID, err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Conversation created successfully: ID=%s, Name=%s", conversation.ID, conversation.Name)

	c.JSON(http.StatusOK, models.APIResponse{
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
// @Success 200 {object} models.APIResponse
// @Router /api/friends [get]
func (h *ChatHandler) GetFriends(c *gin.Context) {
	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	friendships, err := h.friendService.GetFriends(c.Request.Context(), userIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get friends for user %s: %v", userIDStr, err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to get friends",
		})
		return
	}

	fsSlice := dereferenceSlice(friendships)

	logger.InfofWithCaller("Retrieved %d friends for user %s", len(fsSlice), userIDStr)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    fsSlice,
	})
}

// GetPendingFriendRequests 获取待处理的好友请求
// @Summary 获取待处理的好友请求
// @Tags 好友
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.APIResponse
// @Router /api/friends/pending [get]
func (h *ChatHandler) GetPendingFriendRequests(c *gin.Context) {
	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	friendships, err := h.friendService.GetPendingFriendRequests(c.Request.Context(), userIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get pending friend requests for user %s: %v", userIDStr, err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to get pending friend requests",
		})
		return
	}

	fsSlice := dereferenceSlice(friendships)

	logger.InfofWithCaller("Retrieved %d pending friend requests for user %s", len(fsSlice), userIDStr)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    fsSlice,
	})
}

// GetAllFriendRequests 获取所有好友申请记录
// @Summary 获取所有好友申请记录
// @Tags 好友
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.APIResponse
// @Router /api/friends/requests [get]
func (h *ChatHandler) GetAllFriendRequests(c *gin.Context) {
	userIDStr, ok := getUserID(c)
	if !ok {
		return
	}

	friendships, err := h.friendService.GetAllFriendRequests(c.Request.Context(), userIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get all friend requests for user %s: %v", userIDStr, err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to get all friend requests",
		})
		return
	}

	fsSlice := dereferenceSlice(friendships)

	logger.InfofWithCaller("Retrieved %d friend requests for user %s", len(fsSlice), userIDStr)

	c.JSON(http.StatusOK, models.APIResponse{
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

	user, err := h.userService.GetUserByID(c.Request.Context(), id)
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

// SendFriendRequest 发送好友请求
// @Summary 发送好友请求
// @Tags 好友
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.FriendRequest true "好友请求信息"
// @Success 200 {object} models.APIResponse
// @Router /api/friends/request [post]
func (h *ChatHandler) SendFriendRequest(c *gin.Context) {
	var req models.FriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid send friend request: %v", err)
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

	// 发送好友请求（需要创建会话，使用 conversationService 的方法）
	conversation, err := h.friendService.SendFriendRequest(c.Request.Context(), userIDStr, req.TargetUserID, h.conversationService.CreateConversation)
	if err != nil {
		logger.ErrorfWithCaller("Failed to send friend request from %s to %s: %v", userIDStr, req.TargetUserID, err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Friend request sent successfully from %s to %s", userIDStr, req.TargetUserID)

	c.JSON(http.StatusOK, models.APIResponse{
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
// @Success 200 {object} models.APIResponse
// @Router /api/friends/handle [post]
func (h *ChatHandler) HandleFriendRequest(c *gin.Context) {
	var req models.HandleFriendRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid handle friend request: %v", err)
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

	// 验证操作
	if req.Action != "accept" && req.Action != "reject" {
		logger.ErrorfWithCaller("Invalid action for handle friend request: %s", req.Action)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid action. Must be 'accept' or 'reject'",
		})
		return
	}

	// 处理好友请求
	err := h.friendService.HandleFriendRequest(c.Request.Context(), userIDStr, req.ConversationID.String(), req.Action)
	if err != nil {
		logger.ErrorfWithCaller("Failed to handle friend request: %v", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Friend request %s successfully by user %s", req.Action, userIDStr)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Friend request " + req.Action + "ed successfully",
	})
}

// CreateGroupConversation 创建群聊会话
// @Summary 创建群聊会话
// @Tags 会话
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateGroupRequest true "群聊信息"
// @Success 200 {object} models.APIResponse
// @Router /api/conversations/group [post]
func (h *ChatHandler) CreateGroupConversation(c *gin.Context) {
	var req models.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid create group conversation request: %v", err)
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

	conversation, err := h.conversationService.CreateGroupConversation(c.Request.Context(), userIDStr, req.Name, req.Members)
	if err != nil {
		logger.ErrorfWithCaller("Failed to create group conversation: %v", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Group conversation created successfully: ID=%s, Name=%s", conversation.ID, conversation.Name)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Group conversation created successfully",
		Data:    conversation,
	})
}

// AddMemberToConversation 添加成员到会话
// @Summary 添加成员到会话
// @Tags 会话
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.AddMemberRequest true "添加成员信息"
// @Success 200 {object} models.APIResponse
// @Router /api/conversations/members [post]
func (h *ChatHandler) AddMemberToConversation(c *gin.Context) {
	var req models.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid add member request: %v", err)
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

	err := h.memberService.AddMemberToConversation(c.Request.Context(), req.ConversationID.String(), userIDStr, req.UserID.String(), models.EnrollmentRole(req.Role))
	if err != nil {
		logger.ErrorfWithCaller("Failed to add member to conversation: %v", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Member %s added to conversation %s successfully", req.UserID, req.ConversationID)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Member added successfully",
	})
}

// RemoveMemberFromConversation 从会话中移除成员
// @Summary 从会话中移除成员
// @Tags 会话
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.RemoveMemberRequest true "移除成员信息"
// @Success 200 {object} models.APIResponse
// @Router /api/conversations/members [delete]
func (h *ChatHandler) RemoveMemberFromConversation(c *gin.Context) {
	var req models.RemoveMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid remove member request: %v", err)
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

	err := h.memberService.RemoveMemberFromConversation(c.Request.Context(), req.ConversationID.String(), userIDStr, req.UserID.String())
	if err != nil {
		logger.ErrorfWithCaller("Failed to remove member from conversation: %v", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Member %s removed from conversation %s successfully", req.UserID, req.ConversationID)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Member removed successfully",
	})
}

// GetConversationMembers 获取会话成员
// @Summary 获取会话成员
// @Tags 会话
// @Produce json
// @Security BearerAuth
// @Param conversation_id query string true "会话ID"
// @Success 200 {object} models.APIResponse
// @Router /api/conversations/members [get]
func (h *ChatHandler) GetConversationMembers(c *gin.Context) {
	conversationID := c.Query("conversation_id")
	if conversationID == "" {
		logger.ErrorfWithCaller("Missing conversation_id parameter for get conversation members")
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "conversation_id is required",
		})
		return
	}

	members, err := h.conversationService.GetConversationMembers(c.Request.Context(), conversationID)
	if err != nil {
		logger.ErrorfWithCaller("Failed to get conversation members: %v", err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to get conversation members",
		})
		return
	}

	memberSlice := dereferenceSlice(members)

	logger.InfofWithCaller("Retrieved %d members for conversation %s", len(memberSlice), conversationID)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    memberSlice,
	})
}

// UpdateConversation 更新会话信息（群名称）
// @Summary 更新会话信息
// @Tags 会话
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UpdateConversationRequest true "更新会话信息"
// @Success 200 {object} models.APIResponse
// @Router /api/conversations [put]
func (h *ChatHandler) UpdateConversation(c *gin.Context) {
	var req models.UpdateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid update conversation request: %v", err)
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

	// 从查询参数获取 conversation_id
	conversationID := c.Query("conversation_id")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "conversation_id is required",
		})
		return
	}

	conversation, err := h.conversationService.UpdateConversation(c.Request.Context(), conversationID, userIDStr, &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to update conversation: %v", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Conversation %s updated successfully", conversationID)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Conversation updated successfully",
		Data:    conversation,
	})
}

// UpdateMemberRole 更新成员角色
// @Summary 更新成员角色
// @Tags 会话
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UpdateMemberRoleRequest true "更新成员角色"
// @Success 200 {object} models.APIResponse
// @Router /api/conversations/members/role [put]
func (h *ChatHandler) UpdateMemberRole(c *gin.Context) {
	var req models.UpdateMemberRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid update member role request: %v", err)
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

	err := h.memberService.UpdateMemberRole(c.Request.Context(), req.ConversationID.String(), userIDStr, &req)
	if err != nil {
		logger.ErrorfWithCaller("Failed to update member role: %v", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Member %s role updated to %s", req.UserID, req.Role)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Member role updated successfully",
	})
}

// DeleteConversation 解散群聊
// @Summary 解散群聊
// @Tags 会话
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.DeleteConversationRequest true "删除会话"
// @Success 200 {object} models.APIResponse
// @Router /api/conversations [delete]
func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	var req models.DeleteConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.ErrorfWithCaller("Invalid delete conversation request: %v", err)
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

	err := h.conversationService.DeleteConversation(c.Request.Context(), req.ConversationID.String(), userIDStr)
	if err != nil {
		logger.ErrorfWithCaller("Failed to delete conversation: %v", err)
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	logger.InfofWithCaller("Conversation %s deleted successfully", req.ConversationID)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Conversation deleted successfully",
	})
}
