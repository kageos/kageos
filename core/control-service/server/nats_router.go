package server

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// RegisterNATS 注册 control-service 的所有 NATS 订阅。
// subject 真值统一放在 pkg/subjects，这里只负责路由装配。
func RegisterNATS(ctx context.Context, conn *nats.Conn, subs *[]*nats.Subscription, licenseHandler *LicenseCommandHandler) error {
	var err error
	var sub *nats.Subscription

	sub, err = conn.Subscribe(subjects.ControlLicenseKeyGetQuerySubject, licenseHandler.HandleLicenseKeyGetQuery)
	if err != nil {
		return fmt.Errorf("subscribe license key get query: %w", err)
	}
	*subs = append(*subs, sub)

	logger.Infof(ctx, "[NATS Router] Registered %d subscriptions", len(*subs))
	return nil
}
