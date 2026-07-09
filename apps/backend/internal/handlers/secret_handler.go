package handlers

import (
	"net/http"
	"strings"

	"purr-chat-server/internal/services"
	"purr-chat-server/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SecretHandler secret 管理 HTTP 处理器(owner-only CRUD,不返回明文)
type SecretHandler struct {
	secretService *services.SecretService
}

func NewSecretHandler(secretService *services.SecretService) *SecretHandler {
	return &SecretHandler{secretService: secretService}
}

// ListSecrets GET /api/bots/:id/secrets — 返回 key_name 列表
func (h *SecretHandler) ListSecrets(c *gin.Context) {
	ownerID, appID, ok := h.parseIDs(c)
	if !ok {
		return
	}

	secrets, err := h.secretService.ListSecrets(c.Request.Context(), ownerID, appID)
	if err != nil {
		logger.ErrorfWithCaller("[SecretHandler] ListSecrets: %v", err)
		status := http.StatusInternalServerError
		if isSecretClientErr(err) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"secrets": secrets, "total": len(secrets)})
}

// SetSecret PUT /api/bots/:id/secrets/:key — 加密并存储
func (h *SecretHandler) SetSecret(c *gin.Context) {
	ownerID, appID, ok := h.parseIDs(c)
	if !ok {
		return
	}
	keyName := c.Param("key")

	var req struct {
		Value string `json:"value" binding:"required,min=1,max=8192"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.secretService.SetSecret(c.Request.Context(), ownerID, appID, keyName, req.Value)
	if err != nil {
		logger.ErrorfWithCaller("[SecretHandler] SetSecret: %v", err)
		status := http.StatusInternalServerError
		if isSecretClientErr(err) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"key_name": keyName, "has_value": true})
}

// DeleteSecret DELETE /api/bots/:id/secrets/:key
func (h *SecretHandler) DeleteSecret(c *gin.Context) {
	ownerID, appID, ok := h.parseIDs(c)
	if !ok {
		return
	}
	keyName := c.Param("key")

	err := h.secretService.DeleteSecret(c.Request.Context(), ownerID, appID, keyName)
	if err != nil {
		logger.ErrorfWithCaller("[SecretHandler] DeleteSecret: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *SecretHandler) parseIDs(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	ownerIDStr, exists := getUserID(c)
	if !exists {
		return uuid.Nil, uuid.Nil, false
	}
	ownerID, err := uuid.Parse(ownerIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return uuid.Nil, uuid.Nil, false
	}
	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bot id"})
		return uuid.Nil, uuid.Nil, false
	}
	return ownerID, appID, true
}

func isSecretClientErr(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "forbidden") ||
		strings.Contains(msg, "not the bot owner") ||
		strings.Contains(msg, "bot not found") ||
		strings.Contains(msg, "invalid key_name")
}
