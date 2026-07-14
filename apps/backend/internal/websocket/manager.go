package websocket

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
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

// Close codes
const (
	CloseNormal             = 1000
	CloseQueueOverflow      = 1013
	CloseConnectionLimit    = 1013
	CloseConnectionReplaced = 4001
	CloseServerShutdown     = 1001
	CloseAuthFailure        = 1008
	CloseMessageTooBig      = 1009
	CloseProtocolError      = 1002
)

// HubConfig WebSocket Hub 配置
type HubConfig struct {
	MaxConnections           int
	MaxUserDeviceConnections int
	SendQueueSize            int
	ReadLimit                int64
	WriteTimeout             time.Duration
	ReadTimeout              time.Duration
	PingInterval             time.Duration
	AllowedOrigins           []string
	AllowQueryToken          bool
}

// HubMetrics WebSocket 连接指标
type HubMetrics struct {
	TotalConnections   atomic.Int64
	AuthFailures       atomic.Int64
	OriginRejections   atomic.Int64
	QueueOverflows     atomic.Int64
	ProtocolErrors     atomic.Int64
	PingTimeouts       atomic.Int64
	RegistrationErrors atomic.Int64
}

// Client WebSocket客户端
type Client struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Conn        *websocket.Conn
	Send        chan []byte
	DeviceType  DeviceType
	ConnectedAt time.Time
	UserAgent   string
	hub         *Hub

	closeOnce   sync.Once
	done        chan struct{}
	closeCode   int
	closeReason string
}

func (c *Client) close(code int, reason string) {
	c.closeOnce.Do(func() {
		c.closeCode = code
		c.closeReason = reason
		close(c.done)
	})
}

// Hub WebSocket连接管理器
type Hub struct {
	clients           map[uuid.UUID]*Client
	userClients       map[uuid.UUID][]*Client
	userDeviceClients map[uuid.UUID]map[DeviceType][]*Client

	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte

	privateMessage chan *PrivateMessage

	config  HubConfig
	metrics *HubMetrics

	mu       sync.RWMutex
	closed   atomic.Bool
	shutCh   chan struct{}
	shutOnce sync.Once
}

// PrivateMessage 私聊消息
type PrivateMessage struct {
	RecipientID uuid.UUID      `json:"recipient_id"`
	Message     models.Message `json:"message"`
}

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// NewHub 创建新的Hub
func NewHub(cfg HubConfig) *Hub {
	if cfg.SendQueueSize <= 0 {
		cfg.SendQueueSize = 256
	}
	if cfg.MaxUserDeviceConnections <= 0 {
		cfg.MaxUserDeviceConnections = 5
	}
	if cfg.ReadLimit <= 0 {
		cfg.ReadLimit = 1 << 20
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 10 * time.Second
	}
	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = 60 * time.Second
	}
	if cfg.PingInterval <= 0 {
		cfg.PingInterval = 54 * time.Second
	}
	return &Hub{
		clients:           make(map[uuid.UUID]*Client),
		userClients:       make(map[uuid.UUID][]*Client),
		userDeviceClients: make(map[uuid.UUID]map[DeviceType][]*Client),
		register:          make(chan *Client, 64),
		unregister:        make(chan *Client, 512),
		broadcast:         make(chan []byte, 256),
		privateMessage:    make(chan *PrivateMessage, 256),
		config:            cfg,
		metrics:           &HubMetrics{},
		shutCh:            make(chan struct{}),
	}
}

// Run 运行Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			if err := h.RegisterClient(client); err != nil {
				h.metrics.RegistrationErrors.Add(1)
				logger.InfofWithCaller("Failed to register client: %v", err)
				client.close(CloseConnectionLimit, err.Error())
			}
		case client := <-h.unregister:
			h.UnregisterClient(client)
		case message := <-h.broadcast:
			h.broadcastMessage(message)
		case privateMsg := <-h.privateMessage:
			h.sendPrivateMessage(privateMsg)
		case <-h.shutCh:
			return
		}
	}
}

// RegisterClient 注册客户端（线程安全，可从 Run 或直接调用）
func (h *Hub) RegisterClient(client *Client) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.done == nil {
		client.done = make(chan struct{})
	}

	if h.closed.Load() {
		return errors.New("server is shutting down")
	}

	if len(h.clients) >= h.config.MaxConnections {
		logger.InfofWithCaller("WebSocket connection rejected: max connections reached (%d)", h.config.MaxConnections)
		return errors.New("server is at maximum capacity, please try again later")
	}

	deviceConnections := h.userDeviceClients[client.UserID][client.DeviceType]
	if len(deviceConnections) >= h.config.MaxUserDeviceConnections {
		oldestClient := deviceConnections[0]
		logger.InfofWithCaller("User %s has %d %s connections, disconnecting oldest connection %s",
			client.UserID, len(deviceConnections), client.DeviceType, oldestClient.ID)
		oldestClient.close(CloseConnectionReplaced, "connection replaced by newer session")
		h.removeClientLocked(oldestClient)
	}

	h.clients[client.ID] = client
	h.userClients[client.UserID] = append(h.userClients[client.UserID], client)

	if h.userDeviceClients[client.UserID] == nil {
		h.userDeviceClients[client.UserID] = make(map[DeviceType][]*Client)
	}
	h.userDeviceClients[client.UserID][client.DeviceType] = append(
		h.userDeviceClients[client.UserID][client.DeviceType], client)

	h.metrics.TotalConnections.Add(1)
	logger.InfofWithCaller("WebSocket client registered: ClientID=%s, UserID=%s, DeviceType=%s, TotalConnections=%d",
		client.ID, client.UserID, client.DeviceType, len(h.userClients[client.UserID]))

	return nil
}

// UnregisterClient 注销客户端
func (h *Hub) UnregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.removeClientLocked(client)
}

// removeClientLocked 从所有索引中移除客户端（调用者必须持有写锁）
func (h *Hub) removeClientLocked(client *Client) {
	if _, ok := h.clients[client.ID]; ok {
		delete(h.clients, client.ID)
		logger.InfofWithCaller("WebSocket client unregistered: ClientID=%s, UserID=%s, DeviceType=%s",
			client.ID, client.UserID, client.DeviceType)
	}

	if clients, ok := h.userClients[client.UserID]; ok {
		for i, c := range clients {
			if c.ID == client.ID {
				h.userClients[client.UserID] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
		if len(h.userClients[client.UserID]) == 0 {
			delete(h.userClients, client.UserID)
		}
	}

	if deviceMap, ok := h.userDeviceClients[client.UserID]; ok {
		if clients, ok := deviceMap[client.DeviceType]; ok {
			for i, c := range clients {
				if c.ID == client.ID {
					deviceMap[client.DeviceType] = append(clients[:i], clients[i+1:]...)
					break
				}
			}
			if len(deviceMap[client.DeviceType]) == 0 {
				delete(deviceMap, client.DeviceType)
			}
		}
		if len(deviceMap) == 0 {
			delete(h.userDeviceClients, client.UserID)
		}
	}
}

// broadcastMessage 广播消息给所有客户端
func (h *Hub) broadcastMessage(message []byte) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	var overflowed []*Client
	for _, client := range clients {
		select {
		case client.Send <- message:
		default:
			overflowed = append(overflowed, client)
		}
	}

	for _, client := range overflowed {
		h.metrics.QueueOverflows.Add(1)
		logger.ErrorfWithCaller("Send queue full for client %s, disconnecting", client.ID)
		client.close(CloseQueueOverflow, "send queue overflow")
		h.mu.Lock()
		h.removeClientLocked(client)
		h.mu.Unlock()
	}
}

// sendPrivateMessage 发送私聊消息给指定用户的所有在线设备
func (h *Hub) sendPrivateMessage(privateMsg *PrivateMessage) {
	h.mu.RLock()
	clients := h.userClients[privateMsg.RecipientID]
	snapshot := make([]*Client, len(clients))
	copy(snapshot, clients)
	h.mu.RUnlock()

	if len(snapshot) == 0 {
		logger.InfofWithCaller("No active connections for user %s", privateMsg.RecipientID)
		return
	}

	messageData, err := json.Marshal(BroadcastMessage{
		Type:      "new_message",
		Data:      privateMsg.Message,
		Timestamp: privateMsg.Message.CreatedAt.Unix(),
	})
	if err != nil {
		logger.ErrorfWithCaller("Failed to marshal message: %v", err)
		return
	}

	sentCount := 0
	for _, client := range snapshot {
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

	messageData, err := json.Marshal(BroadcastMessage{
		Type:      "new_message",
		Data:      message,
		Timestamp: message.CreatedAt.Unix(),
	})
	if err != nil {
		logger.ErrorfWithCaller("Failed to marshal message: %v", err)
		return
	}

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
	return len(h.userClients[userID])
}

// GetUserDeviceConnectionCount 获取指定用户指定设备类型的连接数
func (h *Hub) GetUserDeviceConnectionCount(userID uuid.UUID, deviceType DeviceType) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	deviceMap, ok := h.userDeviceClients[userID]
	if !ok {
		return 0
	}
	return len(deviceMap[deviceType])
}

// DisconnectUserDevice 断开指定用户指定设备类型的所有连接
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

	for _, client := range clients {
		client.close(CloseNormal, "device disconnected")
		h.removeClientLocked(client)
		logger.InfofWithCaller("Disconnected client %s for user %s device type %s",
			client.ID, userID, deviceType)
	}
}

// DisconnectOldestUserDevice 断开指定用户指定设备类型的最早连接
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

	oldestClient := clients[0]
	oldestClient.close(CloseConnectionReplaced, "connection replaced by newer session")
	h.removeClientLocked(oldestClient)
	logger.InfofWithCaller("Disconnected oldest client %s for user %s device type %s",
		oldestClient.ID, userID, deviceType)

	return true
}

// GetConnectionStats 获取连接统计信息
func (h *Hub) GetConnectionStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := map[string]interface{}{
		"total_connections":           len(h.clients),
		"total_users":                 len(h.userClients),
		"max_connections":             h.config.MaxConnections,
		"max_user_device_connections": h.config.MaxUserDeviceConnections,
	}

	deviceStats := make(map[DeviceType]int)
	for _, deviceMap := range h.userDeviceClients {
		for deviceType, clients := range deviceMap {
			deviceStats[deviceType] += len(clients)
		}
	}
	stats["device_connections"] = deviceStats
	stats["metrics"] = map[string]int64{
		"total_connections":   h.metrics.TotalConnections.Load(),
		"auth_failures":       h.metrics.AuthFailures.Load(),
		"origin_rejections":   h.metrics.OriginRejections.Load(),
		"queue_overflows":     h.metrics.QueueOverflows.Load(),
		"protocol_errors":     h.metrics.ProtocolErrors.Load(),
		"ping_timeouts":       h.metrics.PingTimeouts.Load(),
		"registration_errors": h.metrics.RegistrationErrors.Load(),
	}

	return stats
}

// Shutdown 优雅关闭所有连接
func (h *Hub) Shutdown() {
	h.shutOnce.Do(func() {
		h.closed.Store(true)
		close(h.shutCh)
	})

	h.mu.Lock()
	clients := make([]*Client, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}
	h.clients = make(map[uuid.UUID]*Client)
	h.userClients = make(map[uuid.UUID][]*Client)
	h.userDeviceClients = make(map[uuid.UUID]map[DeviceType][]*Client)
	h.mu.Unlock()

	for _, client := range clients {
		client.close(CloseServerShutdown, "server is shutting down")
	}
}

// Metrics 返回指标快照
func (h *Hub) Metrics() *HubMetrics {
	return h.metrics
}

// checkOrigin 校验请求 Origin
func (h *Hub) checkOrigin(r *http.Request) bool {
	if len(h.config.AllowedOrigins) == 0 {
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	for _, allowed := range h.config.AllowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}
