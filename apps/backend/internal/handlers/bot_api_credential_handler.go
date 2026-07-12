package handlers

import (
	"errors"
	"net/http"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const BotPrincipalContextKey = "bot_principal"

type BotAPICredentialHandler struct {
	service *services.BotAPICredentialService
}

func NewBotAPICredentialHandler(service *services.BotAPICredentialService) *BotAPICredentialHandler {
	return &BotAPICredentialHandler{service: service}
}

func (h *BotAPICredentialHandler) Create(c *gin.Context) {
	ownerID, botID, ok := credentialIDs(c)
	if !ok {
		return
	}
	var req models.CreateBotAPICredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	result, err := h.service.Create(c.Request.Context(), ownerID, botID, req.Name, req.ExpiresAt)
	if err != nil {
		respondCredentialError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (h *BotAPICredentialHandler) List(c *gin.Context) {
	ownerID, botID, ok := credentialIDs(c)
	if !ok {
		return
	}
	items, err := h.service.List(c.Request.Context(), ownerID, botID)
	if err != nil {
		respondCredentialError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"credentials": items, "total": len(items)})
}

func (h *BotAPICredentialHandler) Rotate(c *gin.Context) {
	ownerID, botID, credentialID, ok := credentialOperationIDs(c)
	if !ok {
		return
	}
	result, err := h.service.Rotate(c.Request.Context(), ownerID, botID, credentialID)
	if err != nil {
		respondCredentialError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *BotAPICredentialHandler) Revoke(c *gin.Context) {
	ownerID, botID, credentialID, ok := credentialOperationIDs(c)
	if !ok {
		return
	}
	result, err := h.service.Revoke(c.Request.Context(), ownerID, botID, credentialID)
	if err != nil {
		respondCredentialError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func BotCredentialAuthMiddleware(service *services.BotAPICredentialService) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, err := service.Authenticate(c.Request.Context(), c.GetHeader("Authorization"))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid bot credential"})
			c.Abort()
			return
		}
		c.Set(BotPrincipalContextKey, principal)
		c.Next()
	}
}

func credentialIDs(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	ownerRaw, ok := getUserID(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	ownerID, ownerErr := uuid.Parse(ownerRaw)
	botID, botErr := uuid.Parse(c.Param("id"))
	if ownerErr != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return uuid.Nil, uuid.Nil, false
	}
	if botErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bot id"})
		return uuid.Nil, uuid.Nil, false
	}
	return ownerID, botID, true
}

func credentialOperationIDs(c *gin.Context) (uuid.UUID, uuid.UUID, uuid.UUID, bool) {
	ownerID, botID, ok := credentialIDs(c)
	if !ok {
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	credentialID, err := uuid.Parse(c.Param("credential_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential id"})
		return uuid.Nil, uuid.Nil, uuid.Nil, false
	}
	return ownerID, botID, credentialID, true
}

func respondCredentialError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, services.ErrResourceNotFound):
		status = http.StatusNotFound
	case errors.Is(err, services.ErrCredentialForbidden):
		status = http.StatusForbidden
	case errors.Is(err, services.ErrCredentialInvalid), errors.Is(err, services.ErrCredentialExpired), errors.Is(err, services.ErrCredentialRevoked), errors.Is(err, services.ErrBotNotExternal):
		status = http.StatusBadRequest
	}
	message := "credential operation failed"
	if status != http.StatusInternalServerError {
		message = err.Error()
	}
	c.JSON(status, gin.H{"error": message})
}
