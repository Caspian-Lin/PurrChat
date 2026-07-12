package handlers

import (
	"net/http"

	"purr-chat-server/internal/onebot"

	"github.com/gin-gonic/gin"
)

// BotCapabilityHandler exposes the protocol registry for developer tooling.
// It deliberately requires no credential because it never reveals account data.
type BotCapabilityHandler struct{}

func NewBotCapabilityHandler() *BotCapabilityHandler {
	return &BotCapabilityHandler{}
}

func (h *BotCapabilityHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, onebot.Capabilities())
}
