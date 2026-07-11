package websocket

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"purr-chat-server/pkg/cookie"
	"purr-chat-server/pkg/jwt"
	"purr-chat-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// GlobalHub 全局WebSocket Hub实例
var GlobalHub *Hub

// jwtSecret 用于验证WebSocket连接
var jwtSecret string

// hubConfig 保存当前配置（用于 upgrader CheckOrigin）
var hubConfig HubConfig

// InitHub 使用配置初始化全局Hub
func InitHub(cfg HubConfig) {
	GlobalHub = NewHub(cfg)
	hubConfig = cfg
	go GlobalHub.Run()
	logger.Infof("WebSocket Hub initialized: maxConnections=%d, maxUserConnections=%d, allowedOrigins=%v, allowQueryToken=%v",
		cfg.MaxConnections, cfg.MaxUserConnections, cfg.AllowedOrigins, cfg.AllowQueryToken)
}

// InitJWTSecret 初始化JWT secret
func InitJWTSecret(secret string) {
	jwtSecret = secret
	logger.Info("WebSocket JWT secret initialized")
}

// detectDeviceType 根据User-Agent检测设备类型
func detectDeviceType(userAgent string) DeviceType {
	userAgent = strings.ToLower(userAgent)

	if strings.Contains(userAgent, "ipad") || strings.Contains(userAgent, "tablet") {
		return DeviceTypeTablet
	}

	if strings.Contains(userAgent, "mobile") || strings.Contains(userAgent, "android") ||
		strings.Contains(userAgent, "iphone") {
		return DeviceTypeMobile
	}

	if strings.Contains(userAgent, "mozilla") || strings.Contains(userAgent, "chrome") ||
		strings.Contains(userAgent, "safari") || strings.Contains(userAgent, "firefox") ||
		strings.Contains(userAgent, "edge") {
		return DeviceTypeWeb
	}

	return DeviceTypeUnknown
}

// upgrader 返回配置化的 WebSocket Upgrader
func newUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		Subprotocols:    []string{"bearer"},
		CheckOrigin: func(r *http.Request) bool {
			if GlobalHub != nil {
				return GlobalHub.checkOrigin(r)
			}
			return true
		},
	}
}

// extractToken 按优先级提取 token，防止身份降级
// 优先级: Cookie → Sec-WebSocket-Protocol → query（仅当显式开启）
func extractToken(r *http.Request, allowQueryToken bool) (string, string) {
	if t, ok := cookie.GetTokenFromCookie(r); ok {
		return t, "cookie"
	}

	for _, proto := range r.Header["Sec-Websocket-Protocol"] {
		if strings.HasPrefix(proto, "bearer,") {
			return strings.TrimPrefix(proto, "bearer,"), "subprotocol"
		}
	}

	if allowQueryToken {
		if t := r.URL.Query().Get("token"); t != "" {
			logger.InfofWithCaller("WebSocket token passed via query parameter (deprecated, will be removed)")
			return t, "query"
		}
	}

	return "", ""
}

// HandleWebSocket 处理WebSocket连接
func HandleWebSocket(c *gin.Context) {
	token, source := extractToken(c.Request, hubConfig.AllowQueryToken)

	if token == "" {
		if GlobalHub != nil {
			GlobalHub.metrics.AuthFailures.Add(1)
		}
		logger.ErrorfWithCaller("WebSocket connection rejected: missing token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	userIDStr, err := jwt.ExtractUserID(token, jwtSecret)
	if err != nil {
		if GlobalHub != nil {
			GlobalHub.metrics.AuthFailures.Add(1)
		}
		logger.ErrorfWithCaller("WebSocket connection rejected: invalid token: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		if GlobalHub != nil {
			GlobalHub.metrics.AuthFailures.Add(1)
		}
		logger.ErrorfWithCaller("WebSocket connection rejected: invalid user_id: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	hub := GlobalHub
	upgrader := newUpgrader()
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.ErrorfWithCaller("Failed to upgrade to WebSocket: %v", err)
		return
	}

	userAgent := c.Request.Header.Get("User-Agent")
	deviceType := detectDeviceType(userAgent)

	client := &Client{
		ID:          uuid.New(),
		UserID:      userID,
		Conn:        conn,
		Send:        make(chan []byte, hub.config.SendQueueSize),
		DeviceType:  deviceType,
		ConnectedAt: time.Now(),
		UserAgent:   userAgent,
		done:        make(chan struct{}),
		hub:         hub,
	}

	if err := hub.RegisterClient(client); err != nil {
		client.close(CloseConnectionLimit, err.Error())
		logger.InfofWithCaller("Failed to register client: %v", err)
		return
	}

	logger.InfofWithCaller("WebSocket connected: ClientID=%s, UserID=%s, DeviceType=%s, TokenSource=%s",
		client.ID, client.UserID, client.DeviceType, source)

	go client.writePump()
	client.readPump()
}

// readPump 从WebSocket连接读取消息
func (c *Client) readPump() {
	defer func() {
		c.close(CloseNormal, "connection closed")
		if c.hub != nil {
			c.hub.unregister <- c
		}
	}()

	cfg := c.hub.config
	c.Conn.SetReadLimit(cfg.ReadLimit)
	_ = c.Conn.SetReadDeadline(time.Now().Add(cfg.ReadTimeout))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(cfg.ReadTimeout))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseMessageTooBig) {
				if c.hub != nil {
					c.hub.metrics.ProtocolErrors.Add(1)
				}
				c.close(CloseMessageTooBig, "message too big")
				return
			}
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure) {
				logger.ErrorfWithCaller("WebSocket read error: %v", err)
			}
			return
		}

		c.handleMessage(message)
	}
}

// writePump 向WebSocket连接写入消息
// 每个逻辑事件写入独立 text frame，不合并多个 JSON
func (c *Client) writePump() {
	cfg := c.hub.config
	ticker := time.NewTicker(cfg.PingInterval)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()

	for {
		select {
		case <-c.done:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(cfg.WriteTimeout))
			_ = c.Conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(c.closeCode, c.closeReason))
			return
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(cfg.WriteTimeout))
			if !ok {
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.ErrorfWithCaller("WebSocket write error: %v", err)
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(cfg.WriteTimeout))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				if c.hub != nil {
					c.hub.metrics.PingTimeouts.Add(1)
				}
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
		if c.hub != nil {
			c.hub.metrics.ProtocolErrors.Add(1)
		}
		logger.ErrorfWithCaller("Failed to unmarshal message: %v", err)
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		if c.hub != nil {
			c.hub.metrics.ProtocolErrors.Add(1)
		}
		logger.ErrorfWithCaller("Message missing type field")
		return
	}

	switch msgType {
	case "ping":
		pongMsg := map[string]interface{}{
			"type": "pong",
		}
		data, _ := json.Marshal(pongMsg)
		select {
		case c.Send <- data:
		default:
			logger.ErrorfWithCaller("Failed to send pong to client %s (queue full)", c.ID)
		}

	case "typing":
		// TODO: 实现输入状态广播

	default:
		logger.ErrorfWithCaller("Unknown message type: %s", msgType)
	}
}
