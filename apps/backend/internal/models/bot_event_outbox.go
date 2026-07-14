package models

import (
	"time"

	"github.com/google/uuid"
)

type BotEventOutbox struct {
	ID        uuid.UUID
	BotID     uuid.UUID
	EventID   string
	Seq       int64
	Payload   []byte
	CreatedAt time.Time
	ACKedAt   *time.Time
}

type BotEventAckState struct {
	CredentialID uuid.UUID
	BotID        uuid.UUID
	LastAckedSeq int64
	UpdatedAt    time.Time
}
