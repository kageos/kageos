package config

import (
	"testing"
	"time"
)

func TestTimerSchedulerDefaultQueuedPickupWindowIsOneHour(t *testing.T) {
	cfg := &TimerSchedulerConfig{}

	if got := cfg.GetQueueAckTimeout(); got != 2*time.Minute {
		t.Fatalf("queue ack timeout = %s, want 2m", got)
	}
	if got := cfg.GetMaxDispatchAttempts(); got != 30 {
		t.Fatalf("max dispatch attempts = %d, want 30", got)
	}
	if got := cfg.GetQueueAckTimeout() * time.Duration(cfg.GetMaxDispatchAttempts()); got != time.Hour {
		t.Fatalf("queued pickup window = %s, want 1h", got)
	}
}
