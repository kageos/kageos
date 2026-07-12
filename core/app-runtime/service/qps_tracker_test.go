package service

import (
	"math"
	"testing"
	"time"
)

func TestQPSTrackerAggregatesRequestsBySecond(t *testing.T) {
	t.Parallel()

	tracker := NewQPSTracker(time.Minute, time.Second)
	tracker.RecordRequest("alice", "demo", "v1")
	tracker.RecordRequest("alice", "demo", "v1")

	vqps := mustVersionQPS(t, tracker, "alice", "demo", "v1")
	vqps.mu.RLock()
	total := int64(0)
	for _, count := range vqps.Requests {
		total += count
	}
	vqps.mu.RUnlock()

	if total != 2 {
		t.Fatalf("aggregated request count = %d, want 2", total)
	}
}

func TestQPSTrackerCalculatesQPSFromBucketsAndDropsExpiredBuckets(t *testing.T) {
	t.Parallel()

	const window = time.Minute
	tracker := NewQPSTracker(window, time.Second)
	tracker.ObserveVersion("alice", "demo", "v1")
	vqps := mustVersionQPS(t, tracker, "alice", "demo", "v1")
	now := time.Now().Unix()

	vqps.mu.Lock()
	vqps.Requests = map[int64]int64{
		now:                               3,
		now - 1:                           2,
		now - int64(window.Seconds()) - 1: 100,
	}
	vqps.mu.Unlock()

	got := tracker.GetQPS("alice", "demo", "v1")
	want := float64(5) / window.Seconds()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("QPS = %f, want %f", got, want)
	}

	vqps.mu.RLock()
	_, expiredBucketExists := vqps.Requests[now-int64(window.Seconds())-1]
	remainingBuckets := len(vqps.Requests)
	vqps.mu.RUnlock()
	if expiredBucketExists {
		t.Fatal("expired request bucket was not removed")
	}
	if remainingBuckets != 2 {
		t.Fatalf("remaining bucket count = %d, want 2", remainingBuckets)
	}
}

func TestQPSTrackerIdleRequiresObservation(t *testing.T) {
	t.Parallel()

	tracker := NewQPSTracker(time.Minute, time.Second)

	if tracker.IsIdleFor("alice", "demo", "v1", time.Second) {
		t.Fatal("unobserved version must not be treated as idle")
	}

	tracker.ObserveVersion("alice", "demo", "v1")
	if tracker.IsIdleFor("alice", "demo", "v1", time.Minute) {
		t.Fatal("freshly observed version must wait for the quiet period")
	}

	setObservedAt(t, tracker, "alice", "demo", "v1", time.Now().Add(-2*time.Minute))
	if !tracker.IsIdleFor("alice", "demo", "v1", time.Minute) {
		t.Fatal("observed version with no requests beyond quiet period should be idle")
	}
}

func TestQPSTrackerRequestResetsIdlePeriod(t *testing.T) {
	t.Parallel()

	tracker := NewQPSTracker(time.Minute, time.Second)
	tracker.ObserveVersion("alice", "demo", "v1")
	setObservedAt(t, tracker, "alice", "demo", "v1", time.Now().Add(-2*time.Minute))
	if !tracker.IsIdleFor("alice", "demo", "v1", time.Minute) {
		t.Fatal("expected version to be idle before a new request")
	}

	tracker.RecordRequest("alice", "demo", "v1")
	if tracker.IsIdleFor("alice", "demo", "v1", time.Minute) {
		t.Fatal("recent request must reset idle period")
	}

	setLastRequestAtAndClearWindow(t, tracker, "alice", "demo", "v1", time.Now().Add(-2*time.Minute))
	if !tracker.IsIdleFor("alice", "demo", "v1", time.Minute) {
		t.Fatal("version should be idle after the last request is older than quiet period")
	}
}

func setObservedAt(t *testing.T, tracker *QPSTracker, user, app, version string, observedAt time.Time) {
	t.Helper()

	vqps := mustVersionQPS(t, tracker, user, app, version)
	vqps.mu.Lock()
	vqps.ObservedAt = observedAt
	vqps.mu.Unlock()
}

func setLastRequestAtAndClearWindow(t *testing.T, tracker *QPSTracker, user, app, version string, lastRequestAt time.Time) {
	t.Helper()

	vqps := mustVersionQPS(t, tracker, user, app, version)
	vqps.mu.Lock()
	vqps.LastRequestAt = lastRequestAt
	vqps.Requests = nil
	vqps.mu.Unlock()
}

func mustVersionQPS(t *testing.T, tracker *QPSTracker, user, app, version string) *VersionQPS {
	t.Helper()

	key := tracker.buildKey(user, app, version)
	tracker.mu.RLock()
	vqps := tracker.versionQPS[key]
	tracker.mu.RUnlock()
	if vqps == nil {
		t.Fatalf("missing tracker entry for %s", key)
	}
	return vqps
}
