package service

import (
	"context"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// QPSTracker QPS 跟踪器
type QPSTracker struct {
	// 每个应用版本的 QPS 记录
	versionQPS map[string]*VersionQPS // key: user/app/version
	mu         sync.RWMutex
	
	// 窗口配置
	windowSize    time.Duration // 统计窗口大小
	checkInterval time.Duration // 检查间隔
}

// VersionQPS 单个版本的 QPS 记录
type VersionQPS struct {
	User      string    `json:"user"`
	App       string    `json:"app"`
	Version   string    `json:"version"`
	Requests  []int64   `json:"requests"`  // 时间戳数组，记录请求时间
	LastQPS   float64   `json:"last_qps"`  // 最近一次计算的 QPS
	LastCheck time.Time `json:"last_check"` // 最后检查时间
	mu         sync.RWMutex
}

// NewQPSTracker 创建 QPS 跟踪器
func NewQPSTracker(windowSize, checkInterval time.Duration) *QPSTracker {
	return &QPSTracker{
		versionQPS:    make(map[string]*VersionQPS),
		windowSize:    windowSize,
		checkInterval: checkInterval,
	}
}

// RecordRequest 记录请求
func (q *QPSTracker) RecordRequest(user, app, version string) {
	key := q.buildKey(user, app, version)
	now := time.Now().Unix()
	
	q.mu.Lock()
	defer q.mu.Unlock()
	
	vqps, exists := q.versionQPS[key]
	if !exists {
		vqps = &VersionQPS{
			User:    user,
			App:     app,
			Version: version,
			Requests: make([]int64, 0),
		}
		q.versionQPS[key] = vqps
	}
	
	vqps.mu.Lock()
	vqps.Requests = append(vqps.Requests, now)
	vqps.mu.Unlock()
}

// GetQPS 获取指定版本的当前 QPS
func (q *QPSTracker) GetQPS(user, app, version string) float64 {
	key := q.buildKey(user, app, version)
	
	q.mu.RLock()
	vqps, exists := q.versionQPS[key]
	q.mu.RUnlock()
	
	if !exists {
		return 0
	}
	
	return q.calculateQPS(vqps)
}

// IsSafeToShutdown 检查是否可以安全关闭指定版本
func (q *QPSTracker) IsSafeToShutdown(user, app, version string) bool {
	qps := q.GetQPS(user, app, version)
	logger.Infof(context.Background(), "[QPSTracker] Version %s/%s/%s current QPS: %.2f", user, app, version, qps)
	return qps < 0.1 // QPS 小于 0.1 认为可以安全关闭
}

// calculateQPS 计算 QPS
func (q *QPSTracker) calculateQPS(vqps *VersionQPS) float64 {
	vqps.mu.Lock()
	defer vqps.mu.Unlock()
	
	now := time.Now().Unix()
	windowStart := now - int64(q.windowSize.Seconds())
	
	// 清理过期的请求记录
	validRequests := make([]int64, 0)
	for _, reqTime := range vqps.Requests {
		if reqTime >= windowStart {
			validRequests = append(validRequests, reqTime)
		}
	}
	vqps.Requests = validRequests
	
	// 计算 QPS
	requestCount := len(validRequests)
	windowSeconds := q.windowSize.Seconds()
	qps := float64(requestCount) / windowSeconds
	
	vqps.LastQPS = qps
	vqps.LastCheck = time.Now()
	
	return qps
}

// buildKey 构建版本键
func (q *QPSTracker) buildKey(user, app, version string) string {
	return user + "/" + app + "/" + version
}

// StartCleanup 启动清理任务
func (q *QPSTracker) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(q.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.cleanup()
		}
	}
}

// cleanup 清理过期的数据
func (q *QPSTracker) cleanup() {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	now := time.Now().Unix()
	windowStart := now - int64(q.windowSize.Seconds())
	
	for key, vqps := range q.versionQPS {
		vqps.mu.Lock()
		// 清理过期的请求记录
		validRequests := make([]int64, 0)
		for _, reqTime := range vqps.Requests {
			if reqTime >= windowStart {
				validRequests = append(validRequests, reqTime)
			}
		}
		vqps.Requests = validRequests
		vqps.mu.Unlock()
		
		// 如果长时间没有请求，删除记录
		if len(validRequests) == 0 && time.Since(vqps.LastCheck) > q.windowSize*2 {
			delete(q.versionQPS, key)
		}
	}
}

// GetVersionStats 获取版本统计信息
func (q *QPSTracker) GetVersionStats(user, app, version string) *VersionQPS {
	key := q.buildKey(user, app, version)
	
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	vqps, exists := q.versionQPS[key]
	if !exists {
		return nil
	}
	
	// 返回副本
	return &VersionQPS{
		User:      vqps.User,
		App:       vqps.App,
		Version:   vqps.Version,
		LastQPS:   vqps.LastQPS,
		LastCheck: vqps.LastCheck,
	}
}
