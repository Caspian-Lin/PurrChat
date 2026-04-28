package websocket

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/pkg/logger"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// DeviceType 设备类型枚举
type DeviceType string

const (
	DeviceTypeUnknown DeviceType = "unknown"
	DeviceTypeWeb     DeviceType = "web"
	DeviceTypeMobile  DeviceType = "mobile"
	DeviceTypeDesktop DeviceType = "desktop"
	DeviceTypeTablet  DeviceType = "tablet"
)

// Client WebSocket客户端
type Client struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Conn        *websocket.Conn
	Send        chan []byte
	DeviceType  DeviceType
	ConnectedAt time.Time
	UserAgent   string
}

// Hub WebSocket连接管理器
type Hub struct {
	// 注册的客户端
	clients map[uuid.UUID]*Client

	// 用户ID到客户端的映射（一个用户可能有多个连接）
	userClients map[uuid.UUID][]*Client

	// 用户ID到设备类型到客户端的映射（用于设备类型限制）
	userDeviceClients map[uuid.UUID]map[DeviceType][]*Client

	// 注册和注销通道
	register   chan *Client
	unregister chan *Client

	// 广播通道
	broadcast chan []byte

	// 私聊消息通道
	privateMessage chan *PrivateMessage

	// 配置
	maxConnections     int
	maxUserConnections int

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
func NewHub(maxConnections, maxUserConnections int) *Hub {
	return &Hub{
		clients:            make(map[uuid.UUID]*Client),
		userClients:        make(map[uuid.UUID][]*Client),
		userDeviceClients:  make(map[uuid.UUID]map[DeviceType][]*Client),
		register:           make(chan *Client),
		unregister:         make(chan *Client),
		broadcast:          make(chan []byte, 256),
		privateMessage:     make(chan *PrivateMessage, 256),
		maxConnections:     maxConnections,
		maxUserConnections: maxUserConnections,
	}
}

// Run 运行Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			if err := h.RegisterClient(client); err != nil {
				logger.InfofWithCaller("Failed to register client: %v", err)
			}

		case client := <-h.unregister:
			h.UnregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case privateMsg := <-h.privateMessage:
			h.sendPrivateMessage(privateMsg)
		}
	}
}

// RegisterClient 注册客户端（导出方法用于测试）
func (h *Hub) RegisterClient(client *Client) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 检查全局连接数限制
	if len(h.clients) >= h.maxConnections {
		logger.InfofWithCaller("WebSocket connection rejected: max connections reached (%d)", h.maxConnections)
		return errors.New("server is at maximum capacity, please try again later")
	}

	// 检查用户连接数限制
	userConnections := h.userClients[client.UserID]
	if len(userConnections) >= h.maxUserConnections {
		// 超过限制，断开最早的那个连接
		oldestClient := userConnections[0]
		logger.InfofWithCaller("User %s has %d connections, disconnecting oldest connection %s",
			client.UserID, len(userConnections), oldestClient.ID)

		// 关闭最早连接
		close(oldestClient.Send)
		if oldestClient.Conn != nil {
			oldestClient.Conn.Close()
		}

		// 从映射中移除
		delete(h.clients, oldestClient.ID)
		h.userClients[client.UserID] = userConnections[1:]

		// 从设备类型映射中移除
		if deviceMap, ok := h.userDeviceClients[client.UserID]; ok {
			if clients, ok := deviceMap[oldestClient.DeviceType]; ok {
				for i, c := range clients {
					if c.ID == oldestClient.ID {
						deviceMap[oldestClient.DeviceType] = append(clients[:i], clients[i+1:]...)
						break
					}
				}
				if len(deviceMap[oldestClient.DeviceType]) == 0 {
					delete(deviceMap, oldestClient.DeviceType)
				}
			}
		}
	}

	// 注册新客户端
	h.clients[client.ID] = client
	h.userClients[client.UserID] = append(h.userClients[client.UserID], client)

	// 初始化设备类型映射
	if h.userDeviceClients[client.UserID] == nil {
		h.userDeviceClients[client.UserID] = make(map[DeviceType][]*Client)
	}
	h.userDeviceClients[client.UserID][client.DeviceType] = append(
		h.userDeviceClients[client.UserID][client.DeviceType], client)

	logger.InfofWithCaller("WebSocket client registered: ClientID=%s, UserID=%s, DeviceType=%s, TotalConnections=%d",
		client.ID, client.UserID, client.DeviceType, len(h.userClients[client.UserID]))

	return nil
}

// UnregisterClient 注销客户端（导出方法用于测试）
func (h *Hub) UnregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 从clients中移除
	if _, ok := h.clients[client.ID]; ok {
		delete(h.clients, client.ID)
		close(client.Send)
		logger.InfofWithCaller("WebSocket client unregistered: ClientID=%s, UserID=%s, DeviceType=%s",
			client.ID, client.UserID, client.DeviceType)
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

	// 从userDeviceClients中移除
	if deviceMap, ok := h.userDeviceClients[client.UserID]; ok {
		if clients, ok := deviceMap[client.DeviceType]; ok {
			for i, c := range clients {
				if c.ID == client.ID {
					deviceMap[client.DeviceType] = append(clients[:i], clients[i+1:]...)
					break
				}
			}
			// 如果该设备类型没有其他连接，删除该设备类型的映射
			if len(deviceMap[client.DeviceType]) == 0 {
				delete(deviceMap, client.DeviceType)
			}
		}
		// 如果用户没有其他设备连接，删除该用户的设备映射
		if len(deviceMap) == 0 {
			delete(h.userDeviceClients, client.UserID)
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

// sendPrivateMessage 发送私聊消息给指定用户的所有在线设备
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

// SendToConversation 发送消息给会话中的所有成员（包括发送者），发送到所有在线设备
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

	// 发送给会话中的所有成员（包括发送者）
	for _, memberID := range memberIDs {
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

// SendToUser 发送消息给指定用户的所有在线设备
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

// GetClientCount 获取客户端数量
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients)
}

// GetUserConnectionCount 获取指定用户的连接数
func (h *Hub) GetUserConnectionCount(userID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.userClients[userID]
	if !ok {
		return 0
	}
	return len(clients)
}

// GetUserDeviceConnectionCount 获取指定用户指定设备类型的连接数
func (h *Hub) GetUserDeviceConnectionCount(userID uuid.UUID, deviceType DeviceType) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	deviceMap, ok := h.userDeviceClients[userID]
	if !ok {
		return 0
	}

	clients, ok := deviceMap[deviceType]
	if !ok {
		return 0
	}
	return len(clients)
}

// DisconnectUserDevice 断开指定用户指定设备类型的所有连接（用于设备类型限制扩展）
// 这是一个预留接口，用于未来实现特定设备类型只能有一台设备登录的功能
func (h *Hub) DisconnectUserDevice(userID uuid.UUID, deviceType DeviceType) {
	h.mu.Lock()
	defer h.mu.Unlock()

	deviceMap, ok := h.userDeviceClients[userID]
	if !ok {
		return
	}

	clients, ok := deviceMap[deviceType]
	if !ok {
		return
	}

	// 断开该设备类型的所有连接
	for _, client := range clients {
		close(client.Send)
		if client.Conn != nil {
			client.Conn.Close()
		}
		delete(h.clients, client.ID)
		logger.InfofWithCaller("Disconnected client %s for user %s device type %s",
			client.ID, userID, deviceType)
	}

	// 从设备类型映射中移除
	delete(deviceMap, deviceType)

	// 从用户连接列表中移除
	if userClients, ok := h.userClients[userID]; ok {
		newClients := make([]*Client, 0, len(userClients))
		for _, c := range userClients {
			if c.DeviceType != deviceType {
				newClients = append(newClients, c)
			}
		}
		if len(newClients) == 0 {
			delete(h.userClients, userID)
		} else {
			h.userClients[userID] = newClients
		}
	}

	// 如果用户没有其他设备连接，删除该用户的设备映射
	if len(deviceMap) == 0 {
		delete(h.userDeviceClients, userID)
	}
}

// DisconnectOldestUserDevice 断开指定用户指定设备类型的最早连接（用于设备类型限制扩展）
// 这是一个预留接口，用于未来实现特定设备类型只能有一台设备登录的功能
func (h *Hub) DisconnectOldestUserDevice(userID uuid.UUID, deviceType DeviceType) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	deviceMap, ok := h.userDeviceClients[userID]
	if !ok {
		return false
	}

	clients, ok := deviceMap[deviceType]
	if !ok || len(clients) == 0 {
		return false
	}

	// 断开最早的连接
	oldestClient := clients[0]
	close(oldestClient.Send)
	if oldestClient.Conn != nil {
		oldestClient.Conn.Close()
	}
	delete(h.clients, oldestClient.ID)
	logger.InfofWithCaller("Disconnected oldest client %s for user %s device type %s",
		oldestClient.ID, userID, deviceType)

	// 从设备类型映射中移除
	deviceMap[deviceType] = clients[1:]
	if len(deviceMap[deviceType]) == 0 {
		delete(deviceMap, deviceType)
	}

	// 从用户连接列表中移除
	if userClients, ok := h.userClients[userID]; ok {
		for i, c := range userClients {
			if c.ID == oldestClient.ID {
				h.userClients[userID] = append(userClients[:i], userClients[i+1:]...)
				break
			}
		}
		if len(h.userClients[userID]) == 0 {
			delete(h.userClients, userID)
		}
	}

	// 如果用户没有其他设备连接，删除该用户的设备映射
	if len(deviceMap) == 0 {
		delete(h.userDeviceClients, userID)
	}

	return true
}

// GetConnectionStats 获取连接统计信息
func (h *Hub) GetConnectionStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := map[string]interface{}{
		"total_connections":    len(h.clients),
		"total_users":          len(h.userClients),
		"max_connections":      h.maxConnections,
		"max_user_connections": h.maxUserConnections,
	}

	// 统计各设备类型的连接数
	deviceStats := make(map[DeviceType]int)
	for _, deviceMap := range h.userDeviceClients {
		for deviceType, clients := range deviceMap {
			deviceStats[deviceType] += len(clients)
		}
	}
	stats["device_connections"] = deviceStats

	return stats
}
