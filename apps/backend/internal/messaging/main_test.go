package messaging

import (
	"os"
	"testing"

	"purr-chat-server/pkg/logger"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}
