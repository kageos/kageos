package llms

import (
	"context"
	"testing"
	"time"
)

func TestSendStreamChunkStopsWhenContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	chunks := make(chan *StreamChunk)
	done := make(chan bool, 1)

	go func() {
		done <- sendStreamChunk(ctx, chunks, &StreamChunk{Content: "blocked"})
	}()

	select {
	case <-done:
		t.Fatal("send returned before a receiver was available or context was canceled")
	case <-time.After(20 * time.Millisecond):
	}

	cancel()
	select {
	case sent := <-done:
		if sent {
			t.Fatal("send reported success without a receiver")
		}
	case <-time.After(time.Second):
		t.Fatal("send did not stop after context cancellation")
	}
}
