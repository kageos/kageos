package server

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	publicShareWriteLimitPerMinute = 30
	publicShareRateLimitMaxEntries = 10_000
	publicShareMaxConcurrentWrites = 64
)

type publicShareRateEntry struct {
	windowStart time.Time
	count       int
}

type publicShareRateLimiter struct {
	mu      sync.Mutex
	entries map[string]publicShareRateEntry
	now     func() time.Time
	slots   chan struct{}
}

func newPublicShareRateLimiter() *publicShareRateLimiter {
	return &publicShareRateLimiter{
		entries: make(map[string]publicShareRateEntry),
		now:     time.Now,
		slots:   make(chan struct{}, publicShareMaxConcurrentWrites),
	}
}

func (l *publicShareRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := publicShareRateKey(c)
		if !l.allow(key, publicShareWriteLimitPerMinute, time.Minute) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": "RATE_LIMITED",
				"msg":  "请求过于频繁，请稍后再试",
				"data": nil,
			})
			return
		}
		if !l.acquire() {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"code": "PUBLIC_SHARE_BUSY",
				"msg":  "公开服务繁忙，请稍后再试",
				"data": nil,
			})
			return
		}
		defer l.release()
		c.Next()
	}
}

func (l *publicShareRateLimiter) acquire() bool {
	select {
	case l.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (l *publicShareRateLimiter) release() {
	<-l.slots
}

func (l *publicShareRateLimiter) allow(key string, limit int, window time.Duration) bool {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok || now.Sub(entry.windowStart) >= window {
		if len(l.entries) >= publicShareRateLimitMaxEntries {
			for existingKey, existing := range l.entries {
				if now.Sub(existing.windowStart) >= window {
					delete(l.entries, existingKey)
				}
			}
			if len(l.entries) >= publicShareRateLimitMaxEntries {
				return false
			}
		}
		l.entries[key] = publicShareRateEntry{windowStart: now, count: 1}
		return true
	}
	if entry.count >= limit {
		return false
	}
	entry.count++
	l.entries[key] = entry
	return true
}

func publicShareRateKey(c *gin.Context) string {
	ip := strings.TrimSpace(c.GetHeader("X-Real-IP"))
	if net.ParseIP(ip) == nil {
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err == nil {
			ip = host
		} else {
			ip = c.Request.RemoteAddr
		}
	}
	return ip + "|" + strings.TrimSpace(c.Param("share_id")) + "|" + c.FullPath()
}
