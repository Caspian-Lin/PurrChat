package tests

import (
	"context"
	"testing"

	"purr-chat-server/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBotCallLogRepositoryReturnsEmptySlice(t *testing.T) {
	SetupTestDB(t)
	defer CleanupTestDB(t)

	logs, err := repository.NewBotCallLogRepository().FindAllByBotID(
		context.Background(),
		uuid.New(),
		20,
		0,
	)

	require.NoError(t, err)
	require.NotNil(t, logs)
	require.Empty(t, logs)
}
