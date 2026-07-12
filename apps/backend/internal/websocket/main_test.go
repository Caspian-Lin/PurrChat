package websocket

import (
	"os"
	"testing"

	"purr-chat-server/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.Init()
	code := m.Run()
	os.Exit(code)
}
