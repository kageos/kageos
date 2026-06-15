package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	v1 "github.com/kageos/kageos/core/app-runtime/api/v1"
	"github.com/kageos/kageos/core/app-runtime/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/appinvoke"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/nats-io/nats.go"
)

// AppRequestTransport 负责 runtime -> app 的请求转发与兜底拉起。
// 它实现 v1.InvokeTransport，但保持在 server 层，避免 api handler 依赖 server 结构体。
type AppRequestTransport struct {
	natsConn            *nats.Conn
	appManageService    *service.AppManageService
	appDiscoveryService *service.AppDiscoveryService
	appDatabaseService  *service.AppDatabaseService
}

var _ v1.InvokeTransport = (*AppRequestTransport)(nil)

func NewAppRequestTransport(
	natsConn *nats.Conn,
	appManageService *service.AppManageService,
	appDiscoveryService *service.AppDiscoveryService,
	appDatabaseService *service.AppDatabaseService,
) *AppRequestTransport {
	return &AppRequestTransport{
		natsConn:            natsConn,
		appManageService:    appManageService,
		appDiscoveryService: appDiscoveryService,
		appDatabaseService:  appDatabaseService,
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

	data := msg.Data
	if t.appDatabaseService != nil && t.appDatabaseService.IsEnabled() {
		data, err = t.withAppDatabaseCapability(req, msg.Data)
		if err != nil {
			return err
		}
	}

	appMsg := &nats.Msg{
		Subject: req.AppSubject(),
		Data:    data,
		Header:  msg.Header,
	}

	if err := t.natsConn.PublishMsg(appMsg); err != nil {
		return fmt.Errorf("failed to publish to %s: %w", req.AppSubject(), err)
	}

	return nil
}

func (t *AppRequestTransport) withAppDatabaseCapability(meta *appinvoke.RequestMeta, data []byte) ([]byte, error) {
	var req dto.RequestAppReq
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("unmarshal app request for database capability: %w", err)
	}
	if req.User == "" {
		req.User = meta.User
	}
	if req.App == "" {
		req.App = meta.App
	}
	if req.Version == "" {
		req.Version = meta.Version
	}
	if req.Method == "" {
		req.Method = meta.Method
	}
	if req.Router == "" {
		req.Router = meta.Router
	}
	if req.TargetRouter == "" {
		req.TargetRouter = meta.TargetRouter
	}
	capabilityRouter, err := appDatabaseCapabilityRouter(req.Router, meta.Router, req.TargetRouter)
	if err != nil {
		return nil, err
	}
	capability, err := t.appDatabaseService.IssueCapability(meta.User, meta.App, meta.Version, capabilityRouter)
	if err != nil {
		return nil, fmt.Errorf("issue app database capability: %w", err)
	}
	req.DBCapability = capability
	out, err := json.Marshal(&req)
	if err != nil {
		return nil, fmt.Errorf("marshal app request with database capability: %w", err)
	}
	return out, nil
}

func appDatabaseCapabilityRouter(reqRouter, metaRouter, targetRouter string) (string, error) {
	if target := strings.TrimSpace(targetRouter); target != "" {
		return target, nil
	}
	router := strings.TrimSpace(reqRouter)
	if router == "" {
		router = strings.TrimSpace(metaRouter)
	}
	if !isAppCallbackRouter(router) {
		return router, nil
	}
	return "", fmt.Errorf("app database callback capability target router is missing")
}

func isAppCallbackRouter(router string) bool {
	return strings.Trim(strings.TrimSpace(router), "/") == "_callback"
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
	if err := t.appManageService.StartAppVersion(ctx, user, app, version); err != nil {
		return fmt.Errorf("failed to start app version: %w", err)
	}

	logger.Infof(ctx, "[EnsureAppVersionRunning] Version %s/%s/%s started successfully", user, app, version)
	return nil
}
