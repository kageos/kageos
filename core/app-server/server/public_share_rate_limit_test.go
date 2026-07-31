package server

import (
	"testing"
	"time"
)

func TestPublicShareRateLimiterBoundsRequestsAndEntries(t *testing.T) {
	limiter := newPublicShareRateLimiter()
	now := time.Now()
	limiter.now = func() time.Time { return now }

	for i := 0; i < publicShareWriteLimitPerMinute; i++ {
		if !limiter.allow("127.0.0.1|share|submit", publicShareWriteLimitPerMinute, time.Minute) {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if limiter.allow("127.0.0.1|share|submit", publicShareWriteLimitPerMinute, time.Minute) {
		t.Fatal("request over the per-minute limit should be rejected")
	}

	now = now.Add(time.Minute)
	if !limiter.allow("127.0.0.1|share|submit", publicShareWriteLimitPerMinute, time.Minute) {
		t.Fatal("new time window should allow requests again")
	}
}

func TestPublicShareRateLimiterBoundsConcurrentWrites(t *testing.T) {
	limiter := newPublicShareRateLimiter()
	for i := 0; i < publicShareMaxConcurrentWrites; i++ {
		if !limiter.acquire() {
			t.Fatalf("slot %d should be available", i+1)
		}
	}
	if limiter.acquire() {
		t.Fatal("request over the concurrent write limit should be rejected")
	}
	limiter.release()
	if !limiter.acquire() {
		t.Fatal("released slot should be reusable")
	}
}
