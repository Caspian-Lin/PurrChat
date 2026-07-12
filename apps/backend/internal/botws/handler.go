package botws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var errConnectionLimit = errors.New("bot websocket connection limit reached")

type AuditRecorder interface {
	RecordConnected(context.Context, *models.BotPrincipal, string) error
	RecordInvoked(context.Context, *models.BotPrincipal, string) error
}

type OwnerChecker interface {
	OwnsBot(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

type Handler struct {
	manager  *Manager
	audit    AuditRecorder
	owner    OwnerChecker
	upgrader websocket.Upgrader
}

func NewHandler(manager *Manager, audit AuditRecorder, owner OwnerChecker) *Handler {
	return &Handler{manager: manager, audit: audit, owner: owner, upgrader: websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}}
}

type outbound struct {
	payload []byte
}

type connection struct {
	manager     *Manager
	audit       AuditRecorder
	ws          *websocket.Conn
	principal   models.BotPrincipal
	send        chan outbound
	done        chan struct{}
	closeOnce   sync.Once
	closeCode   int
	closeReason string
	actions     chan struct{}
	seq         atomic.Uint64
}

func (h *Handler) Connect(c *gin.Context) {
	var resumeFrom *int64
	qs := c.Request.URL.Query()
	if len(qs) > 0 {
		for key := range qs {
			if key != "resume_from" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported query parameter: " + key})
				return
			}
		}
		if val := qs.Get("resume_from"); val != "" {
			parsed, err := strconv.ParseInt(val, 10, 64)
			if err != nil || parsed < 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resume_from value"})
				return
			}
			resumeFrom = &parsed
		}
	}
	value, ok := c.Get("bot_principal")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "bot credential required"})
		return
	}
	principal, ok := value.(*models.BotPrincipal)
	if !ok || principal == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid bot principal"})
		return
	}
	ws, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	conn := &connection{manager: h.manager, audit: h.audit, ws: ws, principal: *principal, send: make(chan outbound, h.manager.config.SendQueueSize), done: make(chan struct{}), actions: make(chan struct{}, h.manager.config.MaxConcurrentActions)}
	if err := h.manager.register(conn); err != nil {
		_ = ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(CloseConnectionLimit, "connection limit reached"), time.Now().Add(h.manager.config.WriteTimeout))
		_ = ws.Close()
		return
	}
	if h.audit != nil {
		_ = h.audit.RecordConnected(c.Request.Context(), principal, c.Request.RemoteAddr)
	}
	go conn.writer()
	conn.enqueue(mustEvent(principal.BotID, "meta_event", "lifecycle", "connect", map[string]any{"version": onebot.ProfileVersion}))
	if resumeFrom != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			h.manager.ReplayConnection(ctx, conn, *resumeFrom)
		}()
	}
	conn.reader()
}

func (c *connection) reader() {
	defer func() { c.close(websocket.CloseNormalClosure, "connection closed"); c.manager.unregister(c) }()
	readLimit := min(c.manager.config.MaxFrameBytes, c.manager.config.MaxMessageBytes)
	c.ws.SetReadLimit(readLimit)
	_ = c.ws.SetReadDeadline(time.Now().Add(c.manager.config.ReadTimeout))
	c.ws.SetPongHandler(func(string) error {
		c.manager.heartbeat(c.principal.BotID)
		return c.ws.SetReadDeadline(time.Now().Add(c.manager.config.ReadTimeout))
	})
	for {
		messageType, payload, err := c.ws.ReadMessage()
		if err != nil {
			if strings.Contains(err.Error(), "read limit") {
				c.manager.metrics.protocolErrors.Add(1)
			}
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.manager.recordError(c.principal.BotID, err.Error())
			}
			return
		}
		c.manager.metrics.read.Add(1)
		if messageType != websocket.TextMessage {
			c.manager.metrics.protocolErrors.Add(1)
			c.close(CloseInvalidMessage, "text messages required")
			return
		}
		request, err := onebot.DecodeActionRequest(payload)
		if err != nil {
			c.manager.metrics.protocolErrors.Add(1)
			response, _ := json.Marshal(onebot.Failure(err, extractEcho(payload), c.traceID()))
			if !c.enqueue(response) {
				return
			}
			continue
		}
		select {
		case c.actions <- struct{}{}:
			c.manager.metrics.actionStarted.Add(1)
			go c.dispatch(request)
		default:
			c.manager.metrics.actionRejected.Add(1)
			response, _ := json.Marshal(onebot.Failure(onebot.NewError(onebot.RetCodeRateLimited, "too many concurrent actions", nil), request.Echo, c.traceID()))
			if !c.enqueue(response) {
				return
			}
		}
	}
}

func (c *connection) dispatch(request onebot.ActionRequest) {
	startedAt := time.Now()
	defer func() {
		<-c.actions
		c.manager.metrics.actionCompleted.Add(1)
		c.manager.metrics.actionLatencyNanos.Add(uint64(time.Since(startedAt)))
	}()
	ctx, cancel := context.WithTimeout(context.Background(), c.manager.config.ActionTimeout)
	defer cancel()
	if c.audit != nil {
		_ = c.audit.RecordInvoked(ctx, &c.principal, request.Action)
	}
	data, err := c.manager.dispatcher.Dispatch(ctx, c.principal, request)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = onebot.NewError(onebot.RetCodeInternal, "action timeout", ctx.Err())
	}
	var response onebot.ActionResponse
	if err != nil {
		c.manager.metrics.actionFailed.Add(1)
		response = onebot.Failure(err, request.Echo, c.traceID())
	} else {
		response = onebot.Success(data, request.Echo, c.traceID())
	}
	payload, _ := json.Marshal(response)
	c.enqueue(payload)
}

func (c *connection) writer() {
	ping := time.NewTicker(c.manager.config.PingInterval)
	defer ping.Stop()
	var heartbeat *time.Ticker
	var heartbeatC <-chan time.Time
	if c.manager.config.HeartbeatInterval > 0 {
		heartbeat = time.NewTicker(c.manager.config.HeartbeatInterval)
		heartbeatC = heartbeat.C
		defer heartbeat.Stop()
	}
	defer c.ws.Close()
	for {
		select {
		case item := <-c.send:
			deadline := time.Now().Add(c.manager.config.WriteTimeout)
			_ = c.ws.SetWriteDeadline(deadline)
			if err := c.ws.WriteMessage(websocket.TextMessage, item.payload); err != nil {
				return
			}
			c.manager.metrics.written.Add(1)
		case <-ping.C:
			_ = c.ws.SetWriteDeadline(time.Now().Add(c.manager.config.WriteTimeout))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-heartbeatC:
			if !c.enqueue(mustEvent(c.principal.BotID, "meta_event", "heartbeat", "", map[string]any{"status": "ok", "interval": c.manager.config.HeartbeatInterval.Milliseconds()})) {
				return
			}
		case <-c.done:
			_ = c.ws.SetWriteDeadline(time.Now().Add(c.manager.config.WriteTimeout))
			_ = c.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(c.closeCode, c.closeReason))
			return
		}
	}
}

func (c *connection) enqueue(payload []byte) bool {
	select {
	case <-c.done:
		return false
	default:
	}
	select {
	case c.send <- outbound{payload: append([]byte(nil), payload...)}:
		return true
	default:
		c.manager.metrics.queueOverflows.Add(1)
		c.manager.recordError(c.principal.BotID, "send queue overflow")
		c.close(CloseQueueOverflow, "send queue overflow")
		return false
	}
}

func (c *connection) close(code int, reason string) {
	c.closeOnce.Do(func() { c.closeCode = code; c.closeReason = reason; close(c.done) })
}
func (c *connection) traceID() string {
	return c.principal.BotID.String() + "-" + strings.ToLower(uuid.NewString()) + "-" + strconv.FormatUint(c.seq.Add(1), 10)
}

func mustEvent(botID uuid.UUID, postType, detailType, subType string, data any) []byte {
	raw, _ := json.Marshal(data)
	payload, _ := json.Marshal(onebot.Event{Time: time.Now().Unix(), SelfID: botID.String(), PostType: postType, EventID: uuid.NewString(), DetailType: detailType, SubType: subType, Data: raw})
	return payload
}

func extractEcho(payload []byte) json.RawMessage {
	var value struct {
		Echo json.RawMessage `json:"echo"`
	}
	if json.Unmarshal(payload, &value) != nil {
		return nil
	}
	return value.Echo
}

func (h *Handler) Status(c *gin.Context) {
	ownerID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}
	botID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bot id"})
		return
	}
	allowed, err := h.owner.OwnsBot(c.Request.Context(), ownerID, botID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "status lookup failed"})
		return
	}
	if !allowed {
		c.JSON(http.StatusNotFound, gin.H{"error": "bot not found"})
		return
	}
	c.JSON(http.StatusOK, h.manager.Status(botID))
}

// Health exposes aggregate, token-free operational signals. Per-bot connection
// details remain owner-protected by Status.
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "metrics": h.manager.Metrics()})
}
