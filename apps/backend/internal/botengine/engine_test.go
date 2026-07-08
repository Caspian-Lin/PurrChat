package botengine

import (
	"testing"
	"time"
)

func TestFormatMessageCreatedAtPreservesSubsecondPrecision(t *testing.T) {
	createdAt := time.Date(2026, 7, 7, 10, 0, 0, 123456789, time.UTC)

	got := formatMessageCreatedAt(createdAt)

	want := "2026-07-07T10:00:00.123456789Z"
	if got != want {
		t.Fatalf("formatMessageCreatedAt() = %q, want %q", got, want)
	}
}
