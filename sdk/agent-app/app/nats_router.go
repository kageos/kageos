package app

import (
	"fmt"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/nats-io/nats.go"
)

// registerNATS 注册所有 NATS 订阅。
func registerNATS(a *App) error {
	var err error
	var sub *nats.Subscription

	// runtime -> app 的调用命令
	sub, err = a.conn.Subscribe(a.subjects.InvokeCommand, a.handleMessageAsync)
	if err != nil {
		return fmt.Errorf("subscribe app invoke command %s: %w", a.subjects.InvokeCommand, err)
	}
	a.subs = append(a.subs, sub)
	logger.Infof(a, "Subscribed to app invoke command: %s", a.subjects.InvokeCommand)

	// runtime -> app 的控制命令
	sub, err = a.conn.Subscribe(a.subjects.ControlCommand, a.handleAppControlMessage)
	if err != nil {
		return fmt.Errorf("subscribe app control command %s: %w", a.subjects.ControlCommand, err)
	}
	a.subs = append(a.subs, sub)
	logger.Infof(a, "Subscribed to app control command: %s", a.subjects.ControlCommand)

	// runtime 广播发现命令
	sub, err = a.conn.Subscribe(a.subjects.DiscoveryRequest, a.handleDiscovery)
	if err != nil {
		return fmt.Errorf("subscribe discovery request %s: %w", a.subjects.DiscoveryRequest, err)
	}
	a.subs = append(a.subs, sub)
	logger.Infof(a, "Subscribed to discovery request: %s", a.subjects.DiscoveryRequest)

	return nil
}
