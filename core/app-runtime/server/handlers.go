package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/discovery"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/msgx"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// ============================================================================
// RequestRouter 实现（供 api/v1.RequestHandler 使用，避免 api 层依赖 server 细节）
// ============================================================================

// ForwardToApp 转发请求给应用（实现 v1.RequestRouter）
func (s *Server) ForwardToApp(msg *nats.Msg) error {
	return s.forwardToApp(msg)
}

// IsAppVersionRunning 快速判断应用版本是否在运行（实现 v1.RequestRouter）
func (s *Server) IsAppVersionRunning(user, app, version string) bool {
	return s.isAppVersionRunning(user, app, version)
}

// EnsureAppVersionRunning 确保应用版本正在运行（实现 v1.RequestRouter）
func (s *Server) EnsureAppVersionRunning(ctx context.Context, user, app, version string) error {
	return s.ensureAppVersionRunning(ctx, user, app, version)
}

// forwardToApp 转发请求给应用
func (s *Server) forwardToApp(msg *nats.Msg) error {
	// 构建发送给应用的主题
	appSubject := subjects.BuildAppRuntime2AppSubject(
		msg.Header.Get("user"),
		msg.Header.Get("app"),
		msg.Header.Get("version"),
	)

	// 创建带 header 的消息（传递 trace_id）
	appMsg := &nats.Msg{
		Subject: appSubject,
		Data:    msg.Data,
		Header:  msg.Header, // 直接传递所有 header（包括 trace_id, request_user）
	}

	// 发送请求给应用
	if err := s.natsConn.PublishMsg(appMsg); err != nil {
		return fmt.Errorf("failed to publish to %s: %w", appSubject, err)
	}

	return nil
}

// handleAppDiscoveryResponse 处理应用发现响应
func (s *Server) handleAppDiscoveryResponse(msg *nats.Msg) {
	ctx := context.Background()

	// 使用统一的解析方法
	msgInfo, err := msgx.DecodeNatsMsg[discovery.DiscoveryResponse](msg)
	if err != nil {
		logger.Errorf(ctx, "[handleAppDiscoveryResponse] Failed to decode message: %v", err)
		return
	}

	logger.Infof(ctx, "[handleAppDiscoveryResponse] Received discovery response: user=%s, app=%s, version=%s, status=%s, startTime=%s",
		msgInfo.Data.User, msgInfo.Data.App, msgInfo.Data.Version, msgInfo.Data.Status, msgInfo.Data.StartTime.Format("15:04:05"))

	// 这里不需要额外处理，AppDiscoveryService 已经订阅了同样的主题
	// 这个函数主要是为了日志记录和可能的额外处理
}

// handleAppStartupNotification 处理应用启动完成通知
func (s *Server) handleAppStartupNotification(message subjects.Message) {
	ctx := context.Background()

	// 从 message.Data 中提取业务数据
	var msgData struct {
		Status    string    `json:"status"`
		StartTime time.Time `json:"start_time"`
	}

	// 将 message.Data 转换为具体结构
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		logger.Errorf(ctx, "[handleAppStartupNotification] Failed to marshal message data: %v", err)
		return
	}

	if err := json.Unmarshal(dataBytes, &msgData); err != nil {
		logger.Errorf(ctx, "[handleAppStartupNotification] Failed to decode notification: %v", err)
		return
	}

	// 如果 StartTime 为零值，使用当前时间
	if msgData.StartTime.IsZero() {
		msgData.StartTime = time.Now()
	}

	logger.Infof(ctx, "[handleAppStartupNotification] Received startup notification: user=%s, app=%s, version=%s, status=%s, start_time=%s",
		message.User, message.App, message.Version, msgData.Status, msgData.StartTime.Format(time.RFC3339))

	// 构建通知对象
	notification := &service.StartupNotification{
		User:      message.User,
		App:       message.App,
		Version:   message.Version,
		Status:    msgData.Status,
		StartTime: msgData.StartTime,
	}

	// 通知应用管理服务
	s.appManageService.NotifyStartup(notification)
}

// handleAppCloseNotification 处理应用关闭通知
// 这是应用主动发送的关闭通知（MessageTypeStatusClose），用于优雅关闭流程的第三次握手
func (s *Server) handleAppCloseNotification(message subjects.Message) {
	ctx := context.Background()

	// 从 message.Data 中提取业务数据
	var msgData struct {
		Status    string    `json:"status"`
		StartTime time.Time `json:"start_time"`
		CloseTime time.Time `json:"close_time"`
	}

	// 将 message.Data 转换为具体结构
	dataBytes, err := json.Marshal(message.Data)
	if err != nil {
		logger.Errorf(ctx, "[handleAppCloseNotification] Failed to marshal message data: %v", err)
		return
	}

	if err := json.Unmarshal(dataBytes, &msgData); err != nil {
		logger.Errorf(ctx, "[handleAppCloseNotification] Failed to decode notification: %v", err)
		return
	}

	// 如果 CloseTime 为零值，使用当前时间
	if msgData.CloseTime.IsZero() {
		msgData.CloseTime = time.Now()
	}

	logger.Infof(ctx, "[handleAppCloseNotification] Received close notification: user=%s, app=%s, version=%s, status=%s, close_time=%s",
		message.User, message.App, message.Version, msgData.Status, msgData.CloseTime.Format(time.RFC3339))

	// 构建关闭通知对象
	notification := &service.CloseNotification{
		User:      message.User,
		App:       message.App,
		Version:   message.Version,
		CloseTime: msgData.CloseTime,
	}

	// 通知应用管理服务（用于优雅关闭流程的第三次握手）
	s.appManageService.NotifyClose(notification)

	logger.Infof(ctx, "[handleAppCloseNotification] App closed: %s/%s/%s",
		message.User, message.App, message.Version)
}

// ============================================================================
// 辅助函数
// ============================================================================

// isAppVersionRunning 快速判断应用版本是否在运行（从内存中获取）
func (s *Server) isAppVersionRunning(user, app, version string) bool {
	// 从 AppDiscoveryService 内存中获取应用信息
	appInfo := s.appDiscoveryService.GetAppInfo(user, app)
	if appInfo == nil {
		return false
	}

	// 检查该版本是否存在且正在运行
	if versionInfo := appInfo.GetVersion(version); versionInfo != nil {
		return versionInfo.IsRunning()
	}

	return false
}

// waitForAppStartup 等待应用启动通知（复用 AppManageService 的等待器）
func (s *Server) waitForAppStartup(ctx context.Context, user, app, version string, timeout time.Duration) error {
	logger.Infof(ctx, "[waitForAppStartup] Waiting for %s/%s/%s to start (timeout: %v)...", user, app, version, timeout)

	// 先检查是否已经在运行（可能在我们等待期间已经启动了）
	if s.isAppVersionRunning(user, app, version) {
		logger.Infof(ctx, "[waitForAppStartup] Version %s/%s/%s is already running", user, app, version)
		return nil
	}

	// 注册启动等待器
	key := fmt.Sprintf("%s/%s/%s", user, app, version)
	s.appManageService.RegisterStartupWaiter(key)
	defer s.appManageService.UnregisterStartupWaiter(key)

	// 获取等待 channel
	waiterChan := s.appManageService.GetStartupWaiter(key)
	if waiterChan == nil {
		return fmt.Errorf("failed to get startup waiter")
	}

	// 等待启动通知或超时
	select {
	case notification := <-waiterChan:
		if notification.Status == "running" {
			logger.Infof(ctx, "[waitForAppStartup] Version %s/%s/%s started successfully", user, app, version)
			return nil
		}
		return fmt.Errorf("app started but status is not running: %s", notification.Status)

	case <-time.After(timeout):
		logger.Warnf(ctx, "[waitForAppStartup] Timeout waiting for %s/%s/%s to start", user, app, version)
		return fmt.Errorf("timeout waiting for app startup")

	case <-ctx.Done():
		return ctx.Err()
	}
}

// ensureAppVersionRunning 确保应用版本正在运行
func (s *Server) ensureAppVersionRunning(ctx context.Context, user, app, version string) error {
	logger.Infof(ctx, "[ensureAppVersionRunning] Target version %s/%s/%s is not running, attempting to start...", user, app, version)

	// 检查该应用是否有任何版本在运行
	appInfo := s.appDiscoveryService.GetAppInfo(user, app)
	hasAnyVersionRunning := false

	if appInfo != nil {
		// 检查是否有任何版本在运行
		runningVersions := appInfo.GetRunningVersions()
		hasAnyVersionRunning = len(runningVersions) > 0

		if hasAnyVersionRunning {
			logger.Infof(ctx, "[ensureAppVersionRunning] Found %d running versions, container must be running", len(runningVersions))
		}
	}

	// 使用新的容器命名格式：{user}-{app}-{version}
	containerName := service.BuildContainerName(user, app, version)

	// 检查目标版本的容器是否存在且运行中
	containerRunning, err := s.containerService.IsContainerRunning(ctx, containerName)
	if err != nil {
		logger.Warnf(ctx, "[ensureAppVersionRunning] Failed to check container status: %v", err)
		containerRunning = false
	}

	if containerRunning {
		// 容器已运行，先检查应用是否已经在运行
		if s.isAppVersionRunning(user, app, version) {
			logger.Infof(ctx, "[ensureAppVersionRunning] Container %s is running and app version %s is already running", containerName, version)
			return nil
		}

		// 容器运行但应用未运行，可能是应用进程挂了，尝试重新启动
		logger.Infof(ctx, "[ensureAppVersionRunning] Container %s is running but app version %s is not running, attempting to restart...", containerName, version)
		if err := s.appManageService.StartAppVersion(ctx, user, app, version); err != nil {
			logger.Warnf(ctx, "[ensureAppVersionRunning] Failed to restart app version: %v", err)
			return err
		}
		logger.Infof(ctx, "[ensureAppVersionRunning] App version %s restarted successfully", version)
		return nil
	}

	// 容器不存在或已停止，需要创建或启动版本容器
	// 新架构：每个版本有独立容器，直接调用 StartAppVersion 创建/启动容器
	logger.Infof(ctx, "[ensureAppVersionRunning] Container %s not running, creating/starting version container...", containerName)
	if err := s.appManageService.StartAppVersion(ctx, user, app, version); err != nil {
		return fmt.Errorf("failed to start app version: %w", err)
	}

	logger.Infof(ctx, "[ensureAppVersionRunning] Version %s/%s/%s started successfully", user, app, version)
	return nil
}

// ============================================================================
// NATS Subject 辅助函数
// ============================================================================

func getAppRuntime2AppCreateRequestSubject() string {
	return subjects.GetAppRuntime2AppCreateRequestSubject()
}

func getAppRuntime2AppUpdateRequestSubject() string {
	return subjects.GetAppRuntime2AppUpdateRequestSubject()
}

func getFunctionServer2AppRuntimeRequestSubject() string {
	return subjects.GetFunctionServer2AppRuntimeRequestSubject()
}

// getAppDiscoveryResponseSubject 已移除，现在使用统一的 runtime.status 主题

func getAppServer2AppRuntimeDeleteRequestSubject() string {
	return subjects.GetAppServer2AppRuntimeDeleteRequestSubject()
}

func getAppStartupNotificationSubject() string {
	return subjects.GetAppStartupNotificationSubject()
}

func getAppCloseNotificationSubject() string {
	return subjects.GetAppCloseNotificationSubject()
}

// handleRuntimeStatusMessage 处理 Runtime 状态消息（startup、close、discovery）
func (s *Server) handleRuntimeStatusMessage(msg *nats.Msg) {
	ctx := context.Background()

	var message subjects.Message
	if err := json.Unmarshal(msg.Data, &message); err != nil {
		logger.Errorf(ctx, "[handleRuntimeStatusMessage] Failed to unmarshal message: %v", err)
		return
	}

	logger.Infof(ctx, "[handleRuntimeStatusMessage] Received %s message for %s/%s/%s",
		message.Type, message.User, message.App, message.Version)

	switch message.Type {
	case subjects.MessageTypeStatusStartup:
		s.handleAppStartupNotification(message)
	case subjects.MessageTypeStatusClose:
		s.handleAppCloseNotification(message)
	case subjects.MessageTypeStatusDiscovery:
		// 处理发现消息 - 调用 AppDiscoveryService 的处理逻辑
		logger.Infof(ctx, "[handleRuntimeStatusMessage] Received discovery message")
		s.appDiscoveryService.HandleDiscoveryResponse(message)
	default:
		logger.Warnf(ctx, "[handleRuntimeStatusMessage] Unknown message type: %s", message.Type)
	}
}
