package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"purr-chat-server/internal/onebot"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotCapabilityHandlerReturnsRegistryCatalog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/bot/v1/capabilities", NewBotCapabilityHandler().Get)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/bot/v1/capabilities", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var catalog onebot.CapabilityCatalog
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &catalog))
	assert.Equal(t, onebot.Profile(), catalog.Profile)
	assert.Equal(t, len(onebot.Actions()), len(catalog.Actions))
	assert.Equal(t, len(onebot.Events()), len(catalog.Events))
}
