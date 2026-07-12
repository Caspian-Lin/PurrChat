package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"purr-chat-server/internal/models"
	"purr-chat-server/internal/onebot"
	"purr-chat-server/internal/repository"
	"purr-chat-server/internal/services"
	"purr-chat-server/pkg/database"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type outboxTestEnv struct {
	outboxRepo repository.BotEventOutboxRepository
	botRepo    repository.BotRepository
	instRepo   repository.BotInstallationRepository
	publisher  *services.ReliableEventPublisher
	collector  *eventCollector
	bot        *models.Bot
	cred       *models.BotAPICredential
}

func setupOutboxTestEnv(t *testing.T) *outboxTestEnv {
	t.Helper()
	SetupTestDB(t)
	ctx := context.Background()

	outboxRepo := repository.NewBotEventOutboxRepository()
	botRepo := repository.NewBotRepository()
	instRepo := repository.NewBotInstallationRepository()

	owner := CreateTestUser(t, "obx_owner", "obx_owner@test.com", "pass")

	bot := &models.Bot{
		OwnerID:               owner.ID,
		Name:                  "OutboxBot",
		Status:                models.BotStatusActive,
		BotType:               models.BotTypeExternal,
		Discoverability:       models.DiscoverabilityUnlisted,
		RequestedCapabilities: []string{models.CapabilityReadTrigger, models.CapabilitySend},
	}
	require.NoError(t, botRepo.Create(ctx, bot))

	collector := newEventCollector()
	publisher := services.NewReliableEventPublisher(outboxRepo, collector)

	credID := uuid.New()
	_, err := database.GetPool().Exec(ctx, `
		INSERT INTO bot_api_credentials (id, bot_id, name, token_hash, token_prefix)
		VALUES ($1, $2, 'test-cred', $3, 'purr_bot_test')
	`, credID, bot.ID, []byte("first_credential_hash_32bytes_"))
	require.NoError(t, err)
	cred := &models.BotAPICredential{ID: credID, BotID: bot.ID, Name: "test-cred"}

	return &outboxTestEnv{
		outboxRepo: outboxRepo,
		botRepo:    botRepo,
		instRepo:   instRepo,
		publisher:  publisher,
		collector:  collector,
		bot:        bot,
		cred:       cred,
	}
}

func makeTestEvent(botID uuid.UUID) onebot.Event {
	data, _ := json.Marshal(map[string]any{"text": "hello"})
	return onebot.Event{
		Time:       time.Now().Unix(),
		SelfID:     botID.String(),
		PostType:   onebot.PostTypeMessage,
		EventID:    onebot.GenerateEventID(),
		DetailType: onebot.DetailTypeGroup,
		Data:       data,
	}
}

func TestOutbox_AppendAssignsMonotonicSeq(t *testing.T) {
	env := setupOutboxTestEnv(t)
	ctx := context.Background()

	payload := []byte(`{"post_type":"message"}`)
	seq1, err := env.outboxRepo.Append(ctx, env.bot.ID, "evt_1", payload)
	require.NoError(t, err)
	seq2, err := env.outboxRepo.Append(ctx, env.bot.ID, "evt_2", payload)
	require.NoError(t, err)
	seq3, err := env.outboxRepo.Append(ctx, env.bot.ID, "evt_3", payload)
	require.NoError(t, err)

	assert.Equal(t, int64(1), seq1)
	assert.Equal(t, int64(2), seq2)
	assert.Equal(t, int64(3), seq3)
}

func TestOutbox_FindUnackedReturnsAfterSeq(t *testing.T) {
	env := setupOutboxTestEnv(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := env.outboxRepo.Append(ctx, env.bot.ID, uuid.New().String(), []byte(`{}`))
		require.NoError(t, err)
	}

	unacked, err := env.outboxRepo.FindUnacked(ctx, env.cred.ID, env.bot.ID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, unacked, 5)
	assert.Equal(t, int64(1), unacked[0].Seq)
	assert.Equal(t, int64(5), unacked[4].Seq)

	after3, err := env.outboxRepo.FindUnacked(ctx, env.cred.ID, env.bot.ID, 3, 10)
	require.NoError(t, err)
	assert.Len(t, after3, 2)
	assert.Equal(t, int64(4), after3[0].Seq)
}

func TestOutbox_ACKBySeqMarksEvents(t *testing.T) {
	env := setupOutboxTestEnv(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := env.outboxRepo.Append(ctx, env.bot.ID, uuid.New().String(), []byte(`{}`))
		require.NoError(t, err)
	}

	marked, err := env.outboxRepo.AckUpTo(ctx, env.cred.ID, env.bot.ID, 3)
	require.NoError(t, err)
	assert.Equal(t, int64(3), marked)

	unacked, err := env.outboxRepo.FindUnacked(ctx, env.cred.ID, env.bot.ID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, unacked, 2)
	assert.Equal(t, int64(4), unacked[0].Seq)

	lastAcked, err := env.outboxRepo.GetAckState(ctx, env.cred.ID, env.bot.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), lastAcked)
}

func TestOutbox_ACKIsIdempotent(t *testing.T) {
	env := setupOutboxTestEnv(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := env.outboxRepo.Append(ctx, env.bot.ID, uuid.New().String(), []byte(`{}`))
		require.NoError(t, err)
	}

	_, err := env.outboxRepo.AckUpTo(ctx, env.cred.ID, env.bot.ID, 2)
	require.NoError(t, err)

	_, err = env.outboxRepo.AckUpTo(ctx, env.cred.ID, env.bot.ID, 2)
	require.NoError(t, err)

	lastAcked, err := env.outboxRepo.GetAckState(ctx, env.cred.ID, env.bot.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), lastAcked)

	unacked, err := env.outboxRepo.FindUnacked(ctx, env.cred.ID, env.bot.ID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, unacked, 1)
}

func TestOutbox_ACKDoesNotRegress(t *testing.T) {
	env := setupOutboxTestEnv(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := env.outboxRepo.Append(ctx, env.bot.ID, uuid.New().String(), []byte(`{}`))
		require.NoError(t, err)
	}

	_, err := env.outboxRepo.AckUpTo(ctx, env.cred.ID, env.bot.ID, 4)
	require.NoError(t, err)

	_, err = env.outboxRepo.AckUpTo(ctx, env.cred.ID, env.bot.ID, 1)
	require.NoError(t, err)

	lastAcked, err := env.outboxRepo.GetAckState(ctx, env.cred.ID, env.bot.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(4), lastAcked, "ACK with smaller seq should not regress")
}

func TestOutbox_ACKFutureSequenceRejected(t *testing.T) {
	env := setupOutboxTestEnv(t)
	ctx := context.Background()

	_, err := env.outboxRepo.Append(ctx, env.bot.ID, uuid.New().String(), []byte(`{}`))
	require.NoError(t, err)

	_, err = env.outboxRepo.AckUpTo(ctx, env.cred.ID, env.bot.ID, 2)
	require.ErrorIs(t, err, repository.ErrAckSequenceAhead)
}

func TestOutbox_ACKByEventID(t *testing.T) {
	env := setupOutboxTestEnv(t)
	ctx := context.Background()

	eventID := "evt_test_abc"
	_, err := env.outboxRepo.Append(ctx, env.bot.ID, eventID, []byte(`{}`))
	require.NoError(t, err)
	_, err = env.outboxRepo.Append(ctx, env.bot.ID, "evt_other", []byte(`{}`))
	require.NoError(t, err)

	entry, err := env.outboxRepo.FindByEventID(ctx, env.bot.ID, eventID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), entry.Seq)

	marked, err := env.outboxRepo.AckUpTo(ctx, env.cred.ID, env.bot.ID, entry.Seq)
	require.NoError(t, err)
	assert.Equal(t, int64(1), marked)

	unacked, err := env.outboxRepo.FindUnacked(ctx, env.cred.ID, env.bot.ID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, unacked, 1)
	assert.Equal(t, int64(2), unacked[0].Seq)
}

func TestOutbox_ReliablePublisherPersistsAndPushes(t *testing.T) {
	env := setupOutboxTestEnv(t)

	event := makeTestEvent(env.bot.ID)
	delivered := env.publisher.PublishBotEvent(env.bot.ID, event)
	assert.Equal(t, 1, delivered)

	env.collector.mu.Lock()
	require.Len(t, env.collector.events, 1)
	assert.Greater(t, env.collector.events[0].Event.Seq, int64(0), "event should have seq assigned by publisher")
	env.collector.mu.Unlock()

	ctx := context.Background()
	unacked, err := env.outboxRepo.FindUnacked(ctx, env.cred.ID, env.bot.ID, 0, 10)
	require.NoError(t, err)
	require.Len(t, unacked, 1)
	assert.Equal(t, event.EventID, unacked[0].EventID)
}

func TestOutbox_ReliablePublisherMultipleEventsMonotonicSeq(t *testing.T) {
	env := setupOutboxTestEnv(t)

	for i := 0; i < 5; i++ {
		event := makeTestEvent(env.bot.ID)
		env.publisher.PublishBotEvent(env.bot.ID, event)
	}

	env.collector.mu.Lock()
	require.Len(t, env.collector.events, 5)
	for i, ce := range env.collector.events {
		assert.Equal(t, int64(i+1), ce.Event.Seq, "events should have monotonic seq")
	}
	env.collector.mu.Unlock()
}

func TestOutbox_ACKThenResumeReturnsOnlyUnacked(t *testing.T) {
	env := setupOutboxTestEnv(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		e := makeTestEvent(env.bot.ID)
		env.publisher.PublishBotEvent(env.bot.ID, e)
	}

	_, err := env.outboxRepo.AckUpTo(ctx, env.cred.ID, env.bot.ID, 5)
	require.NoError(t, err)

	lastAcked, err := env.outboxRepo.GetAckState(ctx, env.cred.ID, env.bot.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(5), lastAcked)

	resume, err := env.outboxRepo.FindUnacked(ctx, env.cred.ID, env.bot.ID, lastAcked, 100)
	require.NoError(t, err)
	assert.Len(t, resume, 5)
	assert.Equal(t, int64(6), resume[0].Seq)
	assert.Equal(t, int64(10), resume[4].Seq)
}

func TestOutbox_CountUnacked(t *testing.T) {
	env := setupOutboxTestEnv(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := env.outboxRepo.Append(ctx, env.bot.ID, uuid.New().String(), []byte(`{}`))
		require.NoError(t, err)
	}

	count, err := env.outboxRepo.CountUnacked(ctx, env.bot.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)

	_, err = env.outboxRepo.AckUpTo(ctx, env.cred.ID, env.bot.ID, 2)
	require.NoError(t, err)

	count, err = env.outboxRepo.CountUnacked(ctx, env.bot.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(5), count, "fully acknowledged events remain during the idempotency retention window")
}

func TestOutbox_DeleteAckedAndExpireOld(t *testing.T) {
	env := setupOutboxTestEnv(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := env.outboxRepo.Append(ctx, env.bot.ID, uuid.New().String(), []byte(`{}`))
		require.NoError(t, err)
	}

	_, err := env.outboxRepo.AckUpTo(ctx, env.cred.ID, env.bot.ID, 3)
	require.NoError(t, err)

	deleted, err := env.outboxRepo.DeleteAcked(ctx, 10*time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(0), deleted, "recently acknowledged events remain replayable for idempotency")

	time.Sleep(5 * time.Millisecond)
	deleted, err = env.outboxRepo.DeleteAcked(ctx, 1*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, int64(3), deleted)

	expired, err := env.outboxRepo.ExpireOld(ctx, 1*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, int64(2), expired, "2 unacked events should be expired")

	count, err := env.outboxRepo.CountUnacked(ctx, env.bot.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestOutbox_PerCredentialACKIndependent(t *testing.T) {
	env := setupOutboxTestEnv(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := env.outboxRepo.Append(ctx, env.bot.ID, uuid.New().String(), []byte(`{}`))
		require.NoError(t, err)
	}

	cred1 := env.cred.ID
	cred2 := uuid.New()
	_, err := database.GetPool().Exec(ctx, `
		INSERT INTO bot_api_credentials (id, bot_id, name, token_hash, token_prefix)
		VALUES ($1, $2, 'cred2', $3, 'purr_bot_2')
	`, cred2, env.bot.ID, []byte("second_cred_hash_32_bytes__"))
	require.NoError(t, err)

	_, err = env.outboxRepo.AckUpTo(ctx, cred1, env.bot.ID, 4)
	require.NoError(t, err)

	cred1Events, err := env.outboxRepo.FindUnacked(ctx, cred1, env.bot.ID, 0, 10)
	require.NoError(t, err)
	require.Len(t, cred1Events, 1)
	assert.Equal(t, int64(5), cred1Events[0].Seq)

	cred2Events, err := env.outboxRepo.FindUnacked(ctx, cred2, env.bot.ID, 0, 10)
	require.NoError(t, err)
	require.Len(t, cred2Events, 5)

	last1, err := env.outboxRepo.GetAckState(ctx, cred1, env.bot.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(4), last1)

	_, err = env.outboxRepo.GetAckState(ctx, cred2, env.bot.ID)
	require.Error(t, err, "cred2 should have no ACK state yet")

	_, err = env.outboxRepo.AckUpTo(ctx, cred2, env.bot.ID, 2)
	require.NoError(t, err)

	last2, err := env.outboxRepo.GetAckState(ctx, cred2, env.bot.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), last2)

	cred2Events, err = env.outboxRepo.FindUnacked(ctx, cred2, env.bot.ID, 0, 10)
	require.NoError(t, err)
	require.Len(t, cred2Events, 3)
	assert.Equal(t, int64(3), cred2Events[0].Seq)
}
