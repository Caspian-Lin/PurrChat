package websocket

import (
	"time"

	"purr-chat-server/pkg/jwt"

	"github.com/google/uuid"
)

func createTestToken(userID uuid.UUID) (string, error) {
	return jwt.GenerateToken(userID.String(), "test_secret", 24*time.Hour)
}
