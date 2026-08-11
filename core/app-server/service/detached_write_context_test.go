package service

import (
	"context"
	"testing"
	"time"
)

func TestNewDetachedWriteContextPreservesValuesAndBoundsLifetime(t *testing.T) {
	type contextKey string
	const key contextKey = "request-value"

	parent, cancelParent := context.WithCancel(context.WithValue(context.Background(), key, "kept"))
	cancelParent()

	ctx, cancel := newDetachedWriteContext(parent)
	defer cancel()

	if err := ctx.Err(); err != nil {
		t.Fatalf("detached context inherited cancellation: %v", err)
	}
	if got := ctx.Value(key); got != "kept" {
		t.Fatalf("context value = %v, want kept", got)
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("detached context must have a deadline")
	}
	remaining := time.Until(deadline)
	if remaining <= 0 || remaining > detachedWriteTimeout {
		t.Fatalf("unexpected detached context lifetime: %v", remaining)
	}
}
