package botws

import "time"

const (
	CloseInvalidMessage    = 4000
	CloseQueueOverflow     = 4002
	CloseConnectionLimit   = 4003
	CloseCredentialInvalid = 4004
	CloseBotUnavailable    = 4005
	CloseServerShutdown    = 4006
	CloseActionTimeout     = 4007
)

type Config struct {
	MaxConnections       int
	MaxBotConnections    int
	MaxConcurrentActions int
	SendQueueSize        int
	MaxFrameBytes        int64
	MaxMessageBytes      int64
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	ActionTimeout        time.Duration
	PingInterval         time.Duration
	HeartbeatInterval    time.Duration
}

func DefaultConfig() Config {
	return Config{
		MaxConnections:       1000,
		MaxBotConnections:    3,
		MaxConcurrentActions: 8,
		SendQueueSize:        64,
		MaxFrameBytes:        16 << 10,
		MaxMessageBytes:      16 << 10,
		ReadTimeout:          90 * time.Second,
		WriteTimeout:         10 * time.Second,
		ActionTimeout:        30 * time.Second,
		PingInterval:         30 * time.Second,
		HeartbeatInterval:    30 * time.Second,
	}
}
