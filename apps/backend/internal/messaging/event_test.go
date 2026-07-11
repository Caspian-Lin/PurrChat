package messaging

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"purr-chat-server/internal/models"

	"github.com/google/uuid"
)

type fakeSink struct {
	name      string
	delay     time.Duration
	err       error
	panicVal  any
	invoked   atomic.Int64
	lastEvent *MessageCreatedEvent
}

func (f *fakeSink) OnMessageCreated(ctx context.Context, event *MessageCreatedEvent) error {
	f.invoked.Add(1)
	f.lastEvent = event
	if f.panicVal != nil {
		panic(f.panicVal)
	}
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return f.err
}

func makeEvent() *MessageCreatedEvent {
	return &MessageCreatedEvent{
		Message: &models.Message{
			ID:             uuid.New(),
			ConversationID: uuid.New(),
			SenderID:       uuid.New(),
			Content:        "hello",
			MsgType:        models.MsgTypeText,
			CreatedAt:      time.Now().UTC(),
		},
		ActorType: ActorUser,
		Source:    SourceUser,
		MemberIDs: []uuid.UUID{uuid.New()},
	}
}

func TestPublisher_FanOut(t *testing.T) {
	pub := NewPublisher(5 * time.Second)
	s1 := &fakeSink{name: "s1"}
	s2 := &fakeSink{name: "s2"}
	s3 := &fakeSink{name: "s3"}
	pub.RegisterSink("s1", s1)
	pub.RegisterSink("s2", s2)
	pub.RegisterSink("s3", s3)

	pub.Publish(context.Background(), makeEvent())

	for _, s := range []*fakeSink{s1, s2, s3} {
		if s.invoked.Load() != 1 {
			t.Errorf("%s invoked %d, want 1", s.name, s.invoked.Load())
		}
	}
}

func TestPublisher_ErrorIsolation(t *testing.T) {
	pub := NewPublisher(5 * time.Second)
	s1 := &fakeSink{name: "s1", err: errors.New("fail")}
	s2 := &fakeSink{name: "s2"}
	pub.RegisterSink("s1", s1)
	pub.RegisterSink("s2", s2)

	pub.Publish(context.Background(), makeEvent())

	if s1.invoked.Load() != 1 {
		t.Errorf("s1 invoked %d, want 1", s1.invoked.Load())
	}
	if s2.invoked.Load() != 1 {
		t.Errorf("s2 invoked %d, want 1", s2.invoked.Load())
	}

	m := pub.Metrics()
	if m["s1"].Failed != 1 {
		t.Errorf("s1 failed %d, want 1", m["s1"].Failed)
	}
	if m["s2"].Succeeded != 1 {
		t.Errorf("s2 succeeded %d, want 1", m["s2"].Succeeded)
	}
}

func TestPublisher_PanicIsolation(t *testing.T) {
	pub := NewPublisher(5 * time.Second)
	s1 := &fakeSink{name: "s1", panicVal: "boom"}
	s2 := &fakeSink{name: "s2"}
	pub.RegisterSink("s1", s1)
	pub.RegisterSink("s2", s2)

	pub.Publish(context.Background(), makeEvent())

	if s1.invoked.Load() != 1 {
		t.Errorf("s1 invoked %d, want 1", s1.invoked.Load())
	}
	if s2.invoked.Load() != 1 {
		t.Errorf("s2 invoked %d, want 1", s2.invoked.Load())
	}

	m := pub.Metrics()
	if m["s1"].Failed != 1 {
		t.Errorf("s1 failed %d, want 1", m["s1"].Failed)
	}
	if m["s2"].Succeeded != 1 {
		t.Errorf("s2 succeeded %d, want 1", m["s2"].Succeeded)
	}
}

func TestPublisher_Timeout(t *testing.T) {
	pub := NewPublisher(50 * time.Millisecond)
	s1 := &fakeSink{name: "s1", delay: 5 * time.Second}
	s2 := &fakeSink{name: "s2"}
	pub.RegisterSink("s1", s1)
	pub.RegisterSink("s2", s2)

	pub.Publish(context.Background(), makeEvent())

	m := pub.Metrics()
	if m["s1"].TimedOut != 1 {
		t.Errorf("s1 timedOut %d, want 1", m["s1"].TimedOut)
	}
	if m["s2"].Succeeded != 1 {
		t.Errorf("s2 succeeded %d, want 1", m["s2"].Succeeded)
	}
}

func TestPublisher_Empty(t *testing.T) {
	pub := NewPublisher(5 * time.Second)
	pub.Publish(context.Background(), makeEvent())
}

func TestEvent_ShouldTriggerBots(t *testing.T) {
	tests := []struct {
		name  string
		event *MessageCreatedEvent
		want  bool
	}{
		{
			name: "user text message",
			event: &MessageCreatedEvent{
				ActorType: ActorUser,
				Message:   &models.Message{MsgType: models.MsgTypeText},
			},
			want: true,
		},
		{
			name: "bot message skips",
			event: &MessageCreatedEvent{
				ActorType: ActorBot,
				Message:   &models.Message{MsgType: models.MsgTypeText},
			},
			want: false,
		},
		{
			name: "system message skips",
			event: &MessageCreatedEvent{
				ActorType: ActorSystem,
				Message:   &models.Message{MsgType: models.MsgTypeSystem},
			},
			want: false,
		},
		{
			name: "user system message skips",
			event: &MessageCreatedEvent{
				ActorType: ActorUser,
				Message:   &models.Message{MsgType: models.MsgTypeSystem},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.ShouldTriggerBots(); got != tt.want {
				t.Errorf("ShouldTriggerBots() = %v, want %v", got, tt.want)
			}
		})
	}
}
