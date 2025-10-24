package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/discovery"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

//还有一点，就是发起请求时候，需要快速判断当前的版本是否在运行中，（默认请求的就是当前版本）不考虑其他情况，不在运行有两种情况，1 连容器都没运行，这种情况需要启动容器即可以解决问题，2容器在运行，
//但是目标版本的可执行程序没在运行，这种一般是特殊情况，可能是应用挂了，我们需要像更新那样来钻进容器启动，或者更新的过程中失败了，导致新的没起来，但是版本更新了，旧的还没停掉，这时候新版本流量过来了，我们需要再兜底启动

// AppDiscoveryService 应用发现服务
type AppDiscoveryService struct {
	nats      *nats.Conn
	apps      map[string]*discovery.AppInfo
	mutex     sync.RWMutex
	ticker    *time.Ticker
	runtimeID string
	sub       *nats.Subscription // 存储订阅对象
	basePath  string             // 应用基础路径
}

// NewAppDiscoveryService 创建应用发现服务
func NewAppDiscoveryService(natsConn *nats.Conn, basePath string) *AppDiscoveryService {
	return &AppDiscoveryService{
		nats:      natsConn,
		apps:      make(map[string]*discovery.AppInfo),
		runtimeID: "runtime-1", // TODO: 从配置获取
		basePath:  basePath,
	}
}

// Start 启动发现服务
func (s *AppDiscoveryService) Start() error {
	// 订阅应用发现响应
	sub, err := s.nats.Subscribe(subjects.GetAppDiscoveryResponseSubject(), s.handleDiscoveryResponse)
	if err != nil {
		return err
	}
	s.sub = sub // 存储订阅对象

	// 启动定期心跳检测
	//go s.startHeartbeat()

	// 立即执行一次发现
	go s.discoverApps()

	logger.Infof(context.Background(), "[AppDiscoveryService] Started successfully")
	return nil
}

// Stop 停止发现服务
func (s *AppDiscoveryService) Stop() {
	// 停止定时器
	if s.ticker != nil {
		s.ticker.Stop()
	}

	// 取消订阅
	if s.sub != nil {
		s.sub.Unsubscribe()
	}

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

	data, err := json.Marshal(discoveryMsg)
	if err != nil {
		logger.Errorf(ctx, "[AppDiscoveryService] Failed to marshal discovery message: %v", err)
		return
	}

	err = s.nats.Publish(subjects.GetRuntimeDiscoverySubject(), data)
	if err != nil {
		logger.Errorf(ctx, "[AppDiscoveryService] Failed to publish discovery message: %v", err)
		return
	}

	//logger.Infof(ctx, "[AppDiscoveryService] Discovery message sent")
}

// handleDiscoveryResponse 处理应用发现响应
func (s *AppDiscoveryService) handleDiscoveryResponse(msg *nats.Msg) {
	ctx := context.Background()

	var response discovery.DiscoveryResponse
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		logger.Errorf(ctx, "[AppDiscoveryService] Failed to unmarshal discovery response: %v", err)
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
		LastSeen:  response.Timestamp,
	}
	appInfo.AddVersion(version)

	logger.Infof(ctx, "[AppDiscoveryService] Updated app state: %s/%s %s (started: %s, total versions: %d)",
		response.User, response.App, response.Version, response.StartTime.Format("15:04:05"), appInfo.GetVersionCount())
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

// readCurrentVersion 读取应用的当前版本
func (s *AppDiscoveryService) readCurrentVersion(user, app string) string {
	versionFile := filepath.Join(s.basePath, user, app, "workplace/metadata/current_version.txt")

	data, err := os.ReadFile(versionFile)
	if err != nil {
		// 文件不存在或读取失败，返回空字符串
		return ""
	}

	return strings.TrimSpace(string(data))
}
