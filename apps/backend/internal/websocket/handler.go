package websocket

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"purr-chat-server/pkg/jwt"
	"purr-chat-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应该更严格
	},
}

// Hub 全局WebSocket Hub实例
var GlobalHub *Hub

// JWT secret 用于验证WebSocket连接
var jwtSecret string

// InitHub 初始化全局Hub
func InitHub() {
	GlobalHub = NewHub(20000, 3) // 默认最大连接数20000，每用户最大连接数3
	go GlobalHub.Run()
	logger.Info("WebSocket Hub initialized")
}

// InitHubWithConfig 使用配置初始化全局Hub
func InitHubWithConfig(maxConnections, maxUserConnections int) {
	GlobalHub = NewHub(maxConnections, maxUserConnections)
	go GlobalHub.Run()
	logger.Infof("WebSocket Hub initialized with maxConnections=%d, maxUserConnections=%d",
		maxConnections, maxUserConnections)
}

// InitJWTSecret 初始化JWT secret
func InitJWTSecret(secret string) {
	jwtSecret = secret
	logger.Info("WebSocket JWT secret initialized")
}

// detectDeviceType 根据User-Agent检测设备类型
func detectDeviceType(userAgent string) DeviceType {
	userAgent = strings.ToLower(userAgent)

	// 检测平板设备（优先检测，因为iPad的UA中可能包含mobile）
	if strings.Contains(userAgent, "ipad") || strings.Contains(userAgent, "tablet") {
		return DeviceTypeTablet
	}

	// 检测移动设备
	if strings.Contains(userAgent, "mobile") || strings.Contains(userAgent, "android") ||
		strings.Contains(userAgent, "iphone") {
		return DeviceTypeMobile
	}

	// 检测桌面浏览器
	if strings.Contains(userAgent, "mozilla") || strings.Contains(userAgent, "chrome") ||
		strings.Contains(userAgent, "safari") || strings.Contains(userAgent, "firefox") ||
		strings.Contains(userAgent, "edge") {
		return DeviceTypeWeb
	}

	// 默认返回未知设备类型
	return DeviceTypeUnknown
}

// HandleWebSocket 处理WebSocket连接
func HandleWebSocket(c *gin.Context) {
	// 从查询参数获取token
	token := c.Query("token")
	if token == "" {
		logger.ErrorfWithCaller("WebSocket connection rejected: missing token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	// 验证JWT token并获取用户ID
	userIDStr, err := jwt.ExtractUserID(token, jwtSecret)
	if err != nil {
		logger.ErrorfWithCaller("WebSocket connection rejected: invalid token: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// 解析用户ID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.ErrorfWithCaller("WebSocket connection rejected: invalid user_id: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// 升级HTTP连接到WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.ErrorfWithCaller("Failed to upgrade to WebSocket: %v", err)
		return
	}

	// 获取User-Agent并检测设备类型
	userAgent := c.Request.Header.Get("User-Agent")
	deviceType := detectDeviceType(userAgent)

	// 创建客户端
	client := &Client{
		ID:          uuid.New(),
		UserID:      userID,
		Conn:        conn,
		Send:        make(chan []byte, 256),
		DeviceType:  deviceType,
		ConnectedAt: time.Now(),
		UserAgent:   userAgent,
	}

	// 注册客户端
	err = GlobalHub.RegisterClient(client)
	if err != nil {
		// 如果注册失败（例如超过最大连接数），关闭连接
		conn.Close()
		logger.InfofWithCaller("Failed to register client: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	// 启动读写协程
	go client.writePump()
	go client.readPump()
}

// readPump 从WebSocket连接读取消息
func (c *Client) readPump() {
	defer func() {
		GlobalHub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.ErrorfWithCaller("WebSocket read error: %v", err)
			}
			break
		}

		// 处理接收到的消息
		c.handleMessage(message)
	}
}

// writePump 向WebSocket连接写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub关闭了通道
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				logger.ErrorfWithCaller("WebSocket write error: %v", err)
				return
			}
			if _, err := w.Write(message); err != nil {
				logger.ErrorfWithCaller("WebSocket write message error: %v", err)
				return
			}

			// 排队队列中的消息
			n := len(c.Send)
			for i := 0; i < n; i++ {
				if _, err := w.Write(<-c.Send); err != nil {
					logger.ErrorfWithCaller("WebSocket write queued message error: %v", err)
					return
				}
			}

			if err := w.Close(); err != nil {
				logger.ErrorfWithCaller("WebSocket close error: %v", err)
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.ErrorfWithCaller("WebSocket ping error: %v", err)
				return
			}
		}
	}
}

// handleMessage 处理从客户端接收到的消息
func (c *Client) handleMessage(message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		logger.ErrorfWithCaller("Failed to unmarshal message: %v", err)
		return
	}

	// 根据消息类型处理
	msgType, ok := msg["type"].(string)
	if !ok {
		logger.ErrorfWithCaller("Message missing type field")
		return
	}

	switch msgType {
	case "ping":
		// 响应ping消息
		pongMsg := map[string]interface{}{
			"type": "pong",
		}
		data, _ := json.Marshal(pongMsg)
		select {
		case c.Send <- data:
		default:
			logger.ErrorfWithCaller("Failed to send pong to client %s", c.ID)
		}

	case "typing":
		// 处理输入状态
		// TODO: 实现输入状态广播

	default:
		logger.ErrorfWithCaller("Unknown message type: %s", msgType)
	}
}
