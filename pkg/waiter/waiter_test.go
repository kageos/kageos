package waiter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

type waitResult struct {
	resp *dto.RequestAppResp
	err  error
}

func TestGetDefaultWaiterReturnsSingleton(t *testing.T) {
	first := GetDefaultWaiter()
	second := GetDefaultWaiter()

	if first == nil || second == nil {
		t.Fatal("expected default waiter to be initialized")
	}
	if first != second {
		t.Fatal("expected GetDefaultWaiter to return the same instance")
	}
}

func TestWaitTimeoutRemovesWaiter(t *testing.T) {
	w := New()

	_, err := w.Wait(context.Background(), "trace-timeout", 10*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}

	if w.Notify("trace-timeout", &dto.RequestAppResp{TraceId: "trace-timeout"}) {
		t.Fatal("expected no waiter after timeout")
	}
}

func TestOldWaiterCleanupDoesNotDeleteNewWaiter(t *testing.T) {
	w := New()
	key := "trace-reused"

	ctxA, cancelA := context.WithCancel(context.Background())
	defer cancelA()

	resultA := make(chan waitResult, 1)
	go func() {
		resp, err := w.Wait(ctxA, key, time.Second)
		resultA <- waitResult{resp: resp, err: err}
	}()

	entryA := waitForEntry(t, w, key)

	ctxB, cancelB := context.WithCancel(context.Background())
	defer cancelB()

	resultB := make(chan waitResult, 1)
	go func() {
		resp, err := w.Wait(ctxB, key, time.Second)
		resultB <- waitResult{resp: resp, err: err}
	}()

	entryB := waitForDifferentEntry(t, w, key, entryA)

	cancelA()

	resA := <-resultA
	if !errors.Is(resA.err, context.Canceled) {
		t.Fatalf("expected first waiter canceled, got %v", resA.err)
	}

	w.mu.RLock()
	current := w.waiters[key]
	w.mu.RUnlock()
	if current != entryB {
		t.Fatal("expected newer waiter to remain registered")
	}

	expected := &dto.RequestAppResp{TraceId: key}
	if !w.Notify(key, expected) {
		t.Fatal("expected notify to reach the newer waiter")
	}

	select {
	case resB := <-resultB:
		if resB.err != nil {
			t.Fatalf("expected second waiter success, got %v", resB.err)
		}
		if resB.resp == nil || resB.resp.TraceId != key {
			t.Fatalf("unexpected response: %+v", resB.resp)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second waiter result")
	}

	w.mu.RLock()
	_, ok := w.waiters[key]
	w.mu.RUnlock()
	if ok {
		t.Fatal("expected waiter to be removed after response")
	}
}

func waitForEntry(t *testing.T, w *ResponseWaiter, key string) *responseWaiterEntry {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		w.mu.RLock()
		entry := w.waiters[key]
		w.mu.RUnlock()
		if entry != nil {
			return entry
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatalf("waiter %q was not registered in time", key)
	return nil
}

func waitForDifferentEntry(t *testing.T, w *ResponseWaiter, key string, previous *responseWaiterEntry) *responseWaiterEntry {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		w.mu.RLock()
		entry := w.waiters[key]
		w.mu.RUnlock()
		if entry != nil && entry != previous {
			return entry
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatalf("new waiter %q was not registered in time", key)
	return nil
}
