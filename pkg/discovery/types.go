package discovery

import (
	"sync"
	"time"
)

// DiscoveryMessage 发现广播消息
type DiscoveryMessage struct {
	Type      string    `json:"type"`       // 消息类型，固定为 "discovery"
	RuntimeID string    `json:"runtime_id"` // 运行时 ID
	Timestamp time.Time `json:"timestamp"`  // 发送时间
	Timeout   int       `json:"timeout"`    // 超时时间（秒）
}

// DiscoveryResponse 应用发现响应
type DiscoveryResponse struct {
	Type      string    `json:"type"`       // 消息类型，固定为 "response"
	User      string    `json:"user"`       // 用户名
	App       string    `json:"app"`        // 应用名
	Version   string    `json:"version"`    // 版本号
	Status    string    `json:"status"`     // 状态（running, stopped 等）
	RuntimeID string    `json:"runtime_id"` // 运行时 ID
	StartTime time.Time `json:"start_time"` // 应用启动时间
	Timestamp time.Time `json:"timestamp"`  // 响应时间
}

// AppVersion 应用版本信息
type AppVersion struct {
	Version     string    `json:"version"`      // 版本号
	Status      string    `json:"status"`       // 状态（running, stopped, inactive）
	StartTime   time.Time `json:"start_time"`   // 该版本启动时间
	LastSeen    time.Time `json:"last_seen"`    // 最后发现时间
	ContainerID string    `json:"container_id"` // 容器 ID（可选）
	ProcessID   int       `json:"process_id"`   // 进程 ID（可选）
}

// IsRunning 检查该版本是否正在运行
func (v *AppVersion) IsRunning() bool {
	return v.Status == "running"
}

// AppInfo 存储发现到的应用信息（用于 runtime 内部管理）
type AppInfo struct {
	User           string                 `json:"user"`            // 用户名
	App            string                 `json:"app"`             // 应用名
	CurrentVersion string                 `json:"current_version"` // 当前版本（metadata/current_version.txt）
	Versions       map[string]*AppVersion `json:"versions"`        // 版本信息，key 为版本号
	Mutex          sync.RWMutex           `json:"-"`               // 读写锁，不序列化
}

// GetKey 获取应用的唯一标识键
func (a *AppInfo) GetKey() string {
	return a.User + "/" + a.App
}

// AddVersion 添加或更新版本信息
func (a *AppInfo) AddVersion(version *AppVersion) {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()
	a.Versions[version.Version] = version
}

// GetVersion 获取指定版本信息
func (a *AppInfo) GetVersion(version string) *AppVersion {
	a.Mutex.RLock()
	defer a.Mutex.RUnlock()
	return a.Versions[version]
}

// GetRunningVersions 获取所有运行中的版本
func (a *AppInfo) GetRunningVersions() []*AppVersion {
	a.Mutex.RLock()
	defer a.Mutex.RUnlock()

	var running []*AppVersion
	for _, version := range a.Versions {
		if version.IsRunning() {
			running = append(running, version)
		}
	}
	return running
}

// GetLatestVersion 获取最新版本（按启动时间排序）
func (a *AppInfo) GetLatestVersion() *AppVersion {
	a.Mutex.RLock()
	defer a.Mutex.RUnlock()

	var latest *AppVersion
	for _, version := range a.Versions {
		if latest == nil || version.StartTime.After(latest.StartTime) {
			latest = version
		}
	}
	return latest
}

// IsRunning 检查应用是否有任何版本在运行
func (a *AppInfo) IsRunning() bool {
	a.Mutex.RLock()
	defer a.Mutex.RUnlock()

	for _, version := range a.Versions {
		if version.IsRunning() {
			return true
		}
	}
	return false
}

// GetVersionCount 获取版本总数
func (a *AppInfo) GetVersionCount() int {
	a.Mutex.RLock()
	defer a.Mutex.RUnlock()
	return len(a.Versions)
}
