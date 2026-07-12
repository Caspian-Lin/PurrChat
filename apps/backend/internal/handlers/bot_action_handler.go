package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"purr-chat-server/internal/botws"
	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"

	"github.com/gin-gonic/gin"
)

// BotActionHandler HTTP Action 入口，与 Universal WS 共用同一 dispatcher。
type BotActionHandler struct {
	dispatcher botws.ActionDispatcher
}

func NewBotActionHandler(dispatcher botws.ActionDispatcher) *BotActionHandler {
	return &BotActionHandler{dispatcher: dispatcher}
}

// HandleAction 处理 POST /api/bot/v1/actions/:action。
// Body 为 action params（OneBot 12 HTTP 约定），response 为 ActionResponse。
func (h *BotActionHandler) HandleAction(c *gin.Context) {
	principalVal, exists := c.Get(BotPrincipalContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, onebot.Failure(
			onebot.NewError(onebot.RetCodeUnauthenticated, "bot credential required", nil),
			nil, "",
		))
		return
	}
	principal, ok := principalVal.(*models.BotPrincipal)
	if !ok || principal == nil {
		c.JSON(http.StatusUnauthorized, onebot.Failure(
			onebot.NewError(onebot.RetCodeUnauthenticated, "invalid bot principal", nil),
			nil, "",
		))
		return
	}

	action := c.Param("action")

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, onebot.Failure(
			onebot.NewError(onebot.RetCodeBadRequest, "failed to read request body", err),
			nil, "",
		))
		return
	}

	params := json.RawMessage(body)
	if len(params) == 0 {
		params = json.RawMessage(`{}`)
	}

	request := onebot.ActionRequest{
		Action: action,
		Params: params,
	}

	data, dispatchErr := h.dispatcher.Dispatch(c.Request.Context(), *principal, request)

	var response onebot.ActionResponse
	if dispatchErr != nil {
		response = onebot.Failure(dispatchErr, nil, "")
	} else {
		response = onebot.Success(data, nil, "")
	}

	c.JSON(http.StatusOK, response)
}
