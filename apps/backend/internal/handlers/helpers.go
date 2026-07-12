package handlers

import (
	"errors"
	"net/http"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/services"

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

func respondProtectedResourceError(c *gin.Context, err error, internalMessage string) {
	switch {
	case errors.Is(err, services.ErrInvalidID):
		c.JSON(http.StatusBadRequest, models.APIResponse{Success: false, Message: "Invalid ID"})
	case errors.Is(err, services.ErrResourceNotFound):
		c.JSON(http.StatusNotFound, models.APIResponse{Success: false, Message: "Resource not found"})
	default:
		c.JSON(http.StatusInternalServerError, models.APIResponse{Success: false, Message: internalMessage})
	}
}

// dereferenceSlice 将 []*T 转为 []T
func dereferenceSlice[T any](items []*T) []T {
	out := make([]T, len(items))
	for i, p := range items {
		out[i] = *p
	}
	return out
}
