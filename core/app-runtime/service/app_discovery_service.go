package service

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/discovery"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/nats-io/nats.go"
)

// AppDiscoveryService 应用发现服务
type AppDiscoveryService struct {
	transport *AppDiscoveryTransport
	apps      map[string]*discovery.AppInfo
	mutex     sync.RWMutex
	ticker    *time.Ticker
	runtimeID string
	basePath  string // 应用基础路径

	// 回调函数，用于通知其他服务
	onStartup func(user, app, version, status, errorMessage string, startTime time.Time)
	onClose   func(user, app, version string)
}

// NewAppDiscoveryService 创建应用发现服务
func NewAppDiscoveryService(natsConn *nats.Conn, basePath string) *AppDiscoveryService {
	return NewAppDiscoveryServiceWithRuntimeID(natsConn, basePath, "")
}

func NewAppDiscoveryServiceWithRuntimeID(natsConn *nats.Conn, basePath, runtimeID string) *AppDiscoveryService {
	if strings.TrimSpace(runtimeID) == "" {
		runtimeID = "runtime-local"
	}

	return &AppDiscoveryService{
		transport: NewAppDiscoveryTransport(natsConn),
		apps:      make(map[string]*discovery.AppInfo),
		runtimeID: runtimeID,
		basePath:  basePath,
	}
}

// SetCallbacks 设置回调函数
func (s *AppDiscoveryService) SetCallbacks(onStartup func(user, app, version, status, errorMessage string, startTime time.Time), onClose func(user, app, version string)) {
	s.onStartup = onStartup
	s.onClose = onClose
}

// Start 启动发现服务
func (s *AppDiscoveryService) Start() error {
	handler := NewAppDiscoveryHandler(s)
	if err := s.transport.SubscribeRuntimeLifecycleEvents(handler.HandleRuntimeLifecycleEvent); err != nil {
		return err
	}

	// 启动定期心跳检测
	//go s.startHeartbeat()

	// 立即执行一次发现
	go s.discoverApps()

	return nil
}

// Stop 停止发现服务
func (s *AppDiscoveryService) Stop() {
	// 停止定时器
	if s.ticker != nil {
		s.ticker.Stop()
	}

	_ = s.transport.Close()

	//logger.Infof(context.Background(), "[AppDiscoveryService] Stopped")
}

// startHeartbeat 启动心跳检测
func (s *AppDiscoveryService) startHeartbeat() {
	s.ticker = time.NewTicker(60 * time.Second)
	for range s.ticker.C {
		s.discoverApps()
	}
}

// discoverApps 发现运行中的应用
func (s *AppDiscoveryService) discoverApps() {
	ctx := context.Background()
	//logger.Infof(ctx, "[AppDiscoveryService] Starting app discovery...")

	// 发送发现广播
	discoveryMsg := discovery.DiscoveryMessage{
		Type:      "discovery",
		RuntimeID: s.runtimeID,
		Timestamp: time.Now(),
		Timeout:   5,
	}

	if err := s.transport.PublishDiscoveryRequest(ctx, &discoveryMsg); err != nil {
		logger.Errorf(ctx, "[AppDiscoveryService] Failed to publish discovery message: %v", err)
		return
	}

	//logger.Infof(ctx, "[AppDiscoveryService] Discovery message sent")
}

// GetRunningApps 获取运行中的应用
func (s *AppDiscoveryService) GetRunningApps() map[string]*discovery.AppInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// 返回副本，避免并发修改
	result := make(map[string]*discovery.AppInfo)
	for k, v := range s.apps {
		if v.IsRunning() {
			// 创建副本，只包含运行中的版本
			runningVersions := make(map[string]*discovery.AppVersion)
			for versionKey, version := range v.Versions {
				if version.IsRunning() {
					runningVersions[versionKey] = &discovery.AppVersion{
						Version:     version.Version,
						Status:      version.Status,
						StartTime:   version.StartTime,
						LastSeen:    version.LastSeen,
						ContainerID: version.ContainerID,
						ProcessID:   version.ProcessID,
					}
				}
			}

			result[k] = &discovery.AppInfo{
				User:           v.User,
				App:            v.App,
				CurrentVersion: v.CurrentVersion,
				Versions:       runningVersions,
			}
		}
	}

	return result
}

// GetAppInfo 获取特定应用信息
func (s *AppDiscoveryService) GetAppInfo(user, app string) *discovery.AppInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	key := user + "/" + app
	if appInfo, exists := s.apps[key]; exists {
		// 创建副本，避免并发修改
		versions := make(map[string]*discovery.AppVersion)
		for versionKey, version := range appInfo.Versions {
			versions[versionKey] = &discovery.AppVersion{
				Version:     version.Version,
				Status:      version.Status,
				StartTime:   version.StartTime,
				LastSeen:    version.LastSeen,
				ContainerID: version.ContainerID,
				ProcessID:   version.ProcessID,
			}
		}

		return &discovery.AppInfo{
			User:           appInfo.User,
			App:            appInfo.App,
			CurrentVersion: appInfo.CurrentVersion,
			Versions:       versions,
		}
	}

	return nil
}

// GetRunningVersions 获取特定应用的所有运行中版本
func (s *AppDiscoveryService) GetRunningVersions(user, app string) []*discovery.AppVersion {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	key := user + "/" + app
	if appInfo, exists := s.apps[key]; exists {
		var running []*discovery.AppVersion
		for _, version := range appInfo.Versions {
			if version.IsRunning() {
				running = append(running, &discovery.AppVersion{
					Version:     version.Version,
					Status:      version.Status,
					StartTime:   version.StartTime,
					LastSeen:    version.LastSeen,
					ContainerID: version.ContainerID,
					ProcessID:   version.ProcessID,
				})
			}
		}
		return running
	}

	return nil
}

// IsAppRunning 检查特定应用是否正在运行
func (s *AppDiscoveryService) IsAppRunning(user, app string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	key := user + "/" + app
	if appInfo, exists := s.apps[key]; exists {
		return appInfo.IsRunning()
	}
	return false
}

// IsAppVersionRunning 检查特定应用的特定版本是否正在运行
func (s *AppDiscoveryService) IsAppVersionRunning(user, app, version string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	key := user + "/" + app
	if appInfo, exists := s.apps[key]; exists {
		if versionInfo, exists := appInfo.Versions[version]; exists {
			return versionInfo.IsRunning()
		}
	}
	return false
}

// readCurrentVersion 读取应用的当前版本
func (s *AppDiscoveryService) readCurrentVersion(user, app string) string {
	versionFile := newRuntimeAppPaths(s.basePath, user, app).CurrentVersionPath()

	data, err := os.ReadFile(versionFile)
	if err != nil {
		// 文件不存在或读取失败，返回空字符串
		return ""
	}

	return strings.TrimSpace(string(data))
}

func (s *AppDiscoveryService) applyStartupNotification(user, app, version, status string, startTime time.Time, errorMessage string) {
	ctx := context.Background()

	// 如果 StartTime 为零值，使用当前时间
	if startTime.IsZero() {
		startTime = time.Now()
	}
	if status == "" || status == "started" {
		status = "running"
	}

	if status == "failed" {
		logger.Warnf(ctx, "[AppDiscoveryService] App startup failed: %s/%s %s error=%s", user, app, version, errorMessage)
		if s.onStartup != nil {
			s.onStartup(user, app, version, status, errorMessage, startTime)
		}
		return
	}

	// 更新应用状态
	key := user + "/" + app

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 获取或创建应用信息
	appInfo, exists := s.apps[key]
	if !exists {
		appInfo = &discovery.AppInfo{
			User:           user,
			App:            app,
			CurrentVersion: s.readCurrentVersion(user, app),
			Versions:       make(map[string]*discovery.AppVersion),
		}
		s.apps[key] = appInfo
	} else {
		// 更新当前版本
		appInfo.CurrentVersion = s.readCurrentVersion(user, app)
	}

	// 添加或更新版本信息
	appVersion := &discovery.AppVersion{
		Version:   version,
		Status:    "running",
		StartTime: startTime,
		LastSeen:  time.Now(),
	}
	appInfo.AddVersion(appVersion)

	logger.Infof(ctx, "[AppDiscoveryService] Updated app state from startup: %s/%s %s (started: %s, total versions: %d)",
		user, app, version, startTime.Format("15:04:05"), appInfo.GetVersionCount())

	// 通知其他服务
	if s.onStartup != nil {
		s.onStartup(user, app, version, status, errorMessage, startTime)
	}
}

func (s *AppDiscoveryService) applyCloseNotification(user, app, version, status string, startTime, closeTime time.Time) {
	ctx := context.Background()
	_ = status
	_ = startTime
	if closeTime.IsZero() {
		closeTime = time.Now()
	}

	// 更新应用状态
	key := user + "/" + app

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 获取应用信息
	appInfo, exists := s.apps[key]
	if !exists {
		logger.Warnf(ctx, "[AppDiscoveryService] App not found for close notification: %s/%s", user, app)
		return
	}

	// 更新版本状态为停止
	if appVersion, exists := appInfo.Versions[version]; exists {
		appVersion.Status = "stopped"
		appVersion.LastSeen = time.Now()
		logger.Infof(ctx, "[AppDiscoveryService] Updated app state from close: %s/%s %s (stopped)",
			user, app, version)

		// 通知其他服务
		if s.onClose != nil {
			s.onClose(user, app, version)
		}
	} else {
		logger.Warnf(ctx, "[AppDiscoveryService] Version not found for close notification: %s/%s/%s",
			user, app, version)
	}
}

func (s *AppDiscoveryService) applyDiscoveryResponse(response *discovery.DiscoveryResponse) {
	ctx := context.Background()
	if response == nil {
		logger.Warnf(ctx, "[AppDiscoveryService] Discovery response is nil")
		return
	}

	// 更新应用状态
	key := response.User + "/" + response.App

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 获取或创建应用信息
	appInfo, exists := s.apps[key]
	if !exists {
		appInfo = &discovery.AppInfo{
			User:           response.User,
			App:            response.App,
			CurrentVersion: s.readCurrentVersion(response.User, response.App),
			Versions:       make(map[string]*discovery.AppVersion),
		}
		s.apps[key] = appInfo
	} else {
		// 更新当前版本
		appInfo.CurrentVersion = s.readCurrentVersion(response.User, response.App)
	}

	// 添加或更新版本信息
	version := &discovery.AppVersion{
		Version:   response.Version,
		Status:    response.Status,
		StartTime: response.StartTime,
		LastSeen:  time.Now(),
	}

	appInfo.AddVersion(version)

	logger.Infof(ctx, "[AppDiscoveryService] Updated app state: %s/%s %s (started: %s, total versions: %d)",
		response.User, response.App, response.Version, response.StartTime.Format("15:04:05"), appInfo.GetVersionCount())
}
