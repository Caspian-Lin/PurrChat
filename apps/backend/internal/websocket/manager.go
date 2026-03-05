package websocket

import (
	"encoding/json"
	"sync"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Client WebSocket客户端
type Client struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Conn   *websocket.Conn
	Send   chan []byte
}

// Hub WebSocket连接管理器
type Hub struct {
	// 注册的客户端
	clients map[uuid.UUID]*Client

	// 用户ID到客户端的映射（一个用户可能有多个连接）
	userClients map[uuid.UUID][]*Client

	// 注册和注销通道
	register   chan *Client
	unregister chan *Client

	// 广播通道
	broadcast chan []byte

	// 私聊消息通道
	privateMessage chan *PrivateMessage

	mu sync.RWMutex
}

// PrivateMessage 私聊消息
type PrivateMessage struct {
	RecipientID uuid.UUID      `json:"recipient_id"`
	Message     models.Message `json:"message"`
}

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	Type      string      `json:"type"` // "message", "conversation_update", "friend_request", etc.
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// NewHub 创建新的Hub
func NewHub() *Hub {
	return &Hub{
		clients:        make(map[uuid.UUID]*Client),
		userClients:    make(map[uuid.UUID][]*Client),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		broadcast:      make(chan []byte, 256),
		privateMessage: make(chan *PrivateMessage, 256),
	}
}

// Run 运行Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case privateMsg := <-h.privateMessage:
			h.sendPrivateMessage(privateMsg)
		}
	}
}

// registerClient 注册客户端
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.ID] = client
	h.userClients[client.UserID] = append(h.userClients[client.UserID], client)

	logger.InfofWithCaller("WebSocket client registered: ClientID=%s, UserID=%s", client.ID, client.UserID)
}

// unregisterClient 注销客户端
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 从clients中移除
	if _, ok := h.clients[client.ID]; ok {
		delete(h.clients, client.ID)
		close(client.Send)
		logger.InfofWithCaller("WebSocket client unregistered: ClientID=%s, UserID=%s", client.ID, client.UserID)
	}

	// 从userClients中移除
	if clients, ok := h.userClients[client.UserID]; ok {
		for i, c := range clients {
			if c.ID == client.ID {
				h.userClients[client.UserID] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
		// 如果用户没有其他连接，删除该用户的映射
		if len(h.userClients[client.UserID]) == 0 {
			delete(h.userClients, client.UserID)
		}
	}
}

// broadcastMessage 广播消息给所有客户端
func (h *Hub) broadcastMessage(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		select {
		case client.Send <- message:
		default:
			// 如果发送通道已满，关闭客户端
			close(client.Send)
			delete(h.clients, client.ID)
		}
	}
}

// sendPrivateMessage 发送私聊消息给指定用户
func (h *Hub) sendPrivateMessage(privateMsg *PrivateMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 获取接收者的所有连接
	clients, ok := h.userClients[privateMsg.RecipientID]
	if !ok {
		logger.InfofWithCaller("No active connections for user %s", privateMsg.RecipientID)
		return
	}

	// 序列化消息
	messageData, err := json.Marshal(BroadcastMessage{
		Type:      "new_message",
		Data:      privateMsg.Message,
		Timestamp: privateMsg.Message.CreatedAt.Unix(),
	})
	if err != nil {
		logger.ErrorfWithCaller("Failed to marshal message: %v", err)
		return
	}

	// 发送给接收者的所有连接
	sentCount := 0
	for _, client := range clients {
		select {
		case client.Send <- messageData:
			sentCount++
		default:
			logger.ErrorfWithCaller("Failed to send message to client %s (channel full)", client.ID)
		}
	}

	logger.InfofWithCaller("Message sent to %d clients for user %s", sentCount, privateMsg.RecipientID)
}

// SendToConversation 发送消息给会话中的所有成员（不包括发送者）
func (h *Hub) SendToConversation(conversationID uuid.UUID, senderID uuid.UUID, message models.Message, memberIDs []uuid.UUID) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 序列化消息
	messageData, err := json.Marshal(BroadcastMessage{
		Type:      "new_message",
		Data:      message,
		Timestamp: message.CreatedAt.Unix(),
	})
	if err != nil {
		logger.ErrorfWithCaller("Failed to marshal message: %v", err)
		return
	}

	// 发送给会话中的所有成员（不包括发送者）
	for _, memberID := range memberIDs {
		if memberID == senderID {
			continue // 跳过发送者
		}

		clients, ok := h.userClients[memberID]
		if !ok {
			continue
		}

		for _, client := range clients {
			select {
			case client.Send <- messageData:
			default:
				logger.ErrorfWithCaller("Failed to send message to client %s (channel full)", client.ID)
			}
		}
	}

	logger.InfofWithCaller("Message sent to conversation %s", conversationID)
}

// SendToUser 发送消息给指定用户
func (h *Hub) SendToUser(userID uuid.UUID, messageType string, data interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.userClients[userID]
	if !ok {
		return
	}

	messageData, err := json.Marshal(BroadcastMessage{
		Type:      messageType,
		Data:      data,
		Timestamp: 0,
	})
	if err != nil {
		logger.ErrorfWithCaller("Failed to marshal message: %v", err)
		return
	}

	for _, client := range clients {
		select {
		case client.Send <- messageData:
		default:
			logger.ErrorfWithCaller("Failed to send message to client %s (channel full)", client.ID)
		}
	}
}

// GetOnlineUsers 获取在线用户列表
func (h *Hub) GetOnlineUsers() []uuid.UUID {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users := make([]uuid.UUID, 0, len(h.userClients))
	for userID := range h.userClients {
		users = append(users, userID)
	}
	return users
}

// IsUserOnline 检查用户是否在线
func (h *Hub) IsUserOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, ok := h.userClients[userID]
	return ok
}

// GetClientCount 获取客户端数量
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients)
}
