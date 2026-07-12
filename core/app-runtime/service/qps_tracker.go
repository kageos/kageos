package service

import (
	"context"
	"sync"
	"time"
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
	User          string          `json:"user"`
	App           string          `json:"app"`
	Version       string          `json:"version"`
	Requests      map[int64]int64 `json:"requests"`        // key: 秒级时间戳，value: 该秒请求数
	ObservedAt    time.Time       `json:"observed_at"`     // 首次被清理器观察到的时间
	LastRequestAt time.Time       `json:"last_request_at"` // 最近一次请求进入 runtime 的时间
	LastQPS       float64         `json:"last_qps"`        // 最近一次计算的 QPS
	LastCheck     time.Time       `json:"last_check"`      // 最后检查时间
	mu            sync.RWMutex
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
	now := time.Now()
	nowUnix := now.Unix()

	q.mu.Lock()
	defer q.mu.Unlock()

	vqps, exists := q.versionQPS[key]
	if !exists {
		vqps = &VersionQPS{
			User:       user,
			App:        app,
			Version:    version,
			Requests:   make(map[int64]int64),
			ObservedAt: now,
		}
		q.versionQPS[key] = vqps
	}

	vqps.mu.Lock()
	if vqps.Requests == nil {
		vqps.Requests = make(map[int64]int64)
	}
	vqps.Requests[nowUnix]++
	vqps.LastRequestAt = now
	vqps.mu.Unlock()
}

// ObserveVersion 标记清理器已经开始观察某个版本。
// 这样没有任何请求的旧版本也必须先经历完整静默期，避免 runtime 重启后立刻误清理。
func (q *QPSTracker) ObserveVersion(user, app, version string) {
	key := q.buildKey(user, app, version)
	now := time.Now()

	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.versionQPS[key]; exists {
		return
	}

	q.versionQPS[key] = &VersionQPS{
		User:       user,
		App:        app,
		Version:    version,
		Requests:   make(map[int64]int64),
		ObservedAt: now,
	}
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
	return q.IsIdleFor(user, app, version, q.windowSize)
}

// IsIdleFor 判断版本是否已经完整静默一段时间。
// 必须满足两个条件：统计窗口内没有请求，且距离最近一次请求/首次观察已超过 quietPeriod。
func (q *QPSTracker) IsIdleFor(user, app, version string, quietPeriod time.Duration) bool {
	if quietPeriod <= 0 {
		quietPeriod = q.windowSize
	}

	key := q.buildKey(user, app, version)
	q.mu.RLock()
	vqps, exists := q.versionQPS[key]
	q.mu.RUnlock()
	if !exists {
		return false
	}

	if q.calculateQPS(vqps) > 0 {
		return false
	}

	vqps.mu.RLock()
	lastActivity := vqps.LastRequestAt
	if lastActivity.IsZero() {
		lastActivity = vqps.ObservedAt
	}
	vqps.mu.RUnlock()

	if lastActivity.IsZero() {
		return false
	}
	return time.Since(lastActivity) >= quietPeriod
}

// calculateQPS 计算 QPS
func (q *QPSTracker) calculateQPS(vqps *VersionQPS) float64 {
	vqps.mu.Lock()
	defer vqps.mu.Unlock()

	now := time.Now().Unix()
	windowStart := now - int64(q.windowSize.Seconds())

	// 清理过期的请求记录
	requestCount := int64(0)
	for reqTime, count := range vqps.Requests {
		if reqTime < windowStart {
			delete(vqps.Requests, reqTime)
			continue
		}
		requestCount += count
	}

	// 计算 QPS
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
		for reqTime := range vqps.Requests {
			if reqTime < windowStart {
				delete(vqps.Requests, reqTime)
			}
		}
		lastActivity := vqps.LastRequestAt
		if lastActivity.IsZero() {
			lastActivity = vqps.ObservedAt
		}
		vqps.mu.Unlock()

		// 如果长时间没有请求，删除记录
		if len(vqps.Requests) == 0 && !lastActivity.IsZero() && time.Since(lastActivity) > q.windowSize*24 {
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
		User:          vqps.User,
		App:           vqps.App,
		Version:       vqps.Version,
		ObservedAt:    vqps.ObservedAt,
		LastRequestAt: vqps.LastRequestAt,
		LastQPS:       vqps.LastQPS,
		LastCheck:     vqps.LastCheck,
	}
}
