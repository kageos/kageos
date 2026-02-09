package app

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/env"
	"github.com/nats-io/nats.go"
)

// registerNATS 注册所有 NATS 订阅，subject 硬编码在此，方便阅读（与 app-runtime nats_router 一致）
func registerNATS(a *App) error {
	var err error
	var sub *nats.Subscription

	// 应用请求：app_runtime.app.{user}.{app}.{version}
	subjRequest := fmt.Sprintf("app_runtime.app.%s.%s.%s", env.User, env.App, env.Version)
	sub, err = a.conn.Subscribe(subjRequest, a.handleMessageAsync)
	if err != nil {
		return fmt.Errorf("subscribe app request %s: %w", subjRequest, err)
	}
	a.subs = append(a.subs, sub)
	logger.Infof(a, "Subscribed to app request: %s", subjRequest)

	// App 状态：app.status.{user}.{app}.{version}
	subjStatus := fmt.Sprintf("app.status.%s.%s.%s", env.User, env.App, env.Version)
	sub, err = a.conn.Subscribe(subjStatus, a.handleAppStatusMessage)
	if err != nil {
		return fmt.Errorf("subscribe app status %s: %w", subjStatus, err)
	}
	a.subs = append(a.subs, sub)
	logger.Infof(a, "Subscribed to app status: %s", subjStatus)

	// 服务发现：固定主题
	subjDiscovery := "ai-agent-os.runtime.discovery"
	sub, err = a.conn.Subscribe(subjDiscovery, a.handleDiscovery)
	if err != nil {
		return fmt.Errorf("subscribe discovery %s: %w", subjDiscovery, err)
	}
	a.subs = append(a.subs, sub)
	logger.Infof(a, "Subscribed to discovery: %s", subjDiscovery)

	return nil
}
