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
	waiters map[string]chan *dto.RequestAppResp
}

var waiter *ResponseWaiter

// New 创建等待器实例
func New() *ResponseWaiter {
	return &ResponseWaiter{waiters: make(map[string]chan *dto.RequestAppResp)}
}

func GetDefaultWaiter() *ResponseWaiter {
	if waiter == nil {
		waiter = New()
		return waiter
	}
	return waiter
}

// Wait 在指定超时时间内等待 key 对应的响应
func (w *ResponseWaiter) Wait(ctx context.Context, key string, timeout time.Duration) (*dto.RequestAppResp, error) {
	ch := make(chan *dto.RequestAppResp, 1)

	w.mu.Lock()
	w.waiters[key] = ch
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		delete(w.waiters, key)
		w.mu.Unlock()
		close(ch)
	}()

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(timeout):
		return nil, context.DeadlineExceeded
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Notify 投递响应，若无等待者或不可写则返回 false
func (w *ResponseWaiter) Notify(key string, resp *dto.RequestAppResp) bool {
	w.mu.RLock()
	ch, ok := w.waiters[key]
	w.mu.RUnlock()
	if !ok {
		return false
	}
	select {
	case ch <- resp:
		return true
	default:
		return false
	}
}
