package waiter

import (
	"context"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

// ResponseWaiter 针对 string 键与 *dto.RequestAppResp 的并发安全等待器
type ResponseWaiter struct {
	mu      sync.RWMutex
	waiters map[string]*responseWaiterEntry
}

type responseWaiterEntry struct {
	ch chan *dto.RequestAppResp
}

var (
	defaultWaiter     *ResponseWaiter
	defaultWaiterOnce sync.Once
)

// New 创建等待器实例
func New() *ResponseWaiter {
	return &ResponseWaiter{waiters: make(map[string]*responseWaiterEntry)}
}

func GetDefaultWaiter() *ResponseWaiter {
	defaultWaiterOnce.Do(func() {
		defaultWaiter = New()
	})
	return defaultWaiter
}

// Wait 在指定超时时间内等待 key 对应的响应
func (w *ResponseWaiter) Wait(ctx context.Context, key string, timeout time.Duration) (*dto.RequestAppResp, error) {
	entry := &responseWaiterEntry{
		ch: make(chan *dto.RequestAppResp, 1),
	}

	w.mu.Lock()
	w.waiters[key] = entry
	w.mu.Unlock()

	defer w.removeWaiter(key, entry)

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case resp := <-entry.ch:
		return resp, nil
	case <-timer.C:
		return nil, context.DeadlineExceeded
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (w *ResponseWaiter) removeWaiter(key string, entry *responseWaiterEntry) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if current, ok := w.waiters[key]; ok && current == entry {
		delete(w.waiters, key)
	}
}

// Notify 投递响应，若无等待者或不可写则返回 false
func (w *ResponseWaiter) Notify(key string, resp *dto.RequestAppResp) bool {
	w.mu.RLock()
	entry, ok := w.waiters[key]
	w.mu.RUnlock()
	if !ok || entry == nil {
		return false
	}
	select {
	case entry.ch <- resp:
		return true
	default:
		return false
	}
}
