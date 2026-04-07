package server

import (
	"context"
	"fmt"

	v1 "github.com/ai-agent-os/ai-agent-os/core/app-runtime/api/v1"
	"github.com/ai-agent-os/ai-agent-os/core/app-runtime/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/appinvoke"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/nats-io/nats.go"
)

// AppRequestTransport 负责 runtime -> app 的请求转发与兜底拉起。
// 它实现 v1.InvokeTransport，但保持在 server 层，避免 api handler 依赖 server 结构体。
type AppRequestTransport struct {
	natsConn            *nats.Conn
	containerService    service.ContainerOperator
	appManageService    *service.AppManageService
	appDiscoveryService *service.AppDiscoveryService
}

var _ v1.InvokeTransport = (*AppRequestTransport)(nil)

func NewAppRequestTransport(
	natsConn *nats.Conn,
	containerService service.ContainerOperator,
	appManageService *service.AppManageService,
	appDiscoveryService *service.AppDiscoveryService,
) *AppRequestTransport {
	return &AppRequestTransport{
		natsConn:            natsConn,
		containerService:    containerService,
		appManageService:    appManageService,
		appDiscoveryService: appDiscoveryService,
	}
}

// ForwardToApp 转发请求给应用（实现 v1.InvokeTransport）。
func (t *AppRequestTransport) ForwardToApp(msg *nats.Msg) error {
	if t == nil || t.natsConn == nil {
		return fmt.Errorf("NATS connection is nil")
	}

	req, err := appinvoke.ParseRuntimeRequest(msg)
	if err != nil {
		return err
	}

	appMsg := &nats.Msg{
		Subject: req.AppSubject(),
		Data:    msg.Data,
		Header:  msg.Header,
	}

	if err := t.natsConn.PublishMsg(appMsg); err != nil {
		return fmt.Errorf("failed to publish to %s: %w", req.AppSubject(), err)
	}

	return nil
}

// IsAppVersionRunning 快速判断应用版本是否在运行（实现 v1.InvokeTransport）。
func (t *AppRequestTransport) IsAppVersionRunning(user, app, version string) bool {
	if t == nil || t.appDiscoveryService == nil {
		return false
	}
	return t.appDiscoveryService.IsAppVersionRunning(user, app, version)
}

// EnsureAppVersionRunning 确保应用版本正在运行（实现 v1.InvokeTransport）。
func (t *AppRequestTransport) EnsureAppVersionRunning(ctx context.Context, user, app, version string) error {
	if t == nil {
		return fmt.Errorf("request transport is nil")
	}

	logger.Infof(ctx, "[EnsureAppVersionRunning] Target version %s/%s/%s is not running, attempting to start...", user, app, version)

	containerName := service.BuildContainerName(user, app, version)

	containerRunning, err := t.containerService.IsContainerRunning(ctx, containerName)
	if err != nil {
		logger.Warnf(ctx, "[EnsureAppVersionRunning] Failed to check container status: %v", err)
		containerRunning = false
	}

	if containerRunning {
		if t.IsAppVersionRunning(user, app, version) {
			logger.Infof(ctx, "[EnsureAppVersionRunning] Container %s is running and app version %s is already running", containerName, version)
			return nil
		}

		logger.Infof(ctx, "[EnsureAppVersionRunning] Container %s is running but app version %s is not running, attempting to restart...", containerName, version)
		if err := t.appManageService.StartAppVersion(ctx, user, app, version); err != nil {
			logger.Warnf(ctx, "[EnsureAppVersionRunning] Failed to restart app version: %v", err)
			return err
		}
		logger.Infof(ctx, "[EnsureAppVersionRunning] App version %s restarted successfully", version)
		return nil
	}

	logger.Infof(ctx, "[EnsureAppVersionRunning] Container %s not running, creating/starting version container...", containerName)
	if err := t.appManageService.StartAppVersion(ctx, user, app, version); err != nil {
		return fmt.Errorf("failed to start app version: %w", err)
	}

	logger.Infof(ctx, "[EnsureAppVersionRunning] Version %s/%s/%s started successfully", user, app, version)
	return nil
}
