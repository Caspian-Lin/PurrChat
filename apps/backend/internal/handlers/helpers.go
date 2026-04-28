package handlers

import (
	"net/http"

	"purr-chat-server/internal/models"

	"github.com/gin-gonic/gin"
)

// getUserID 从 context 提取并验证 user_id，失败时自动返回 401
func getUserID(c *gin.Context) (string, bool) {
	raw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.APIResponse{Success: false, Message: "Unauthorized"})
		return "", false
	}
	id, ok := raw.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: "Invalid user ID"})
		return "", false
	}
	return id, true
}

// dereferenceSlice 将 []*T 转为 []T
func dereferenceSlice[T any](items []*T) []T {
	out := make([]T, len(items))
	for i, p := range items {
		out[i] = *p
	}
	return out
}
