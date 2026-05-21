package server

import (
	"context"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/nats-io/nats.go"
)

// RegisterNATS 注册 hr-server 的所有 NATS 订阅。
// subject 真值统一放在 pkg/subjects，这里只负责路由装配。
func RegisterNATS(ctx context.Context, conn *nats.Conn, subs *[]*nats.Subscription) error {
	_ = conn
	_ = subs
	logger.Infof(ctx, "[NATS Router] Registered %d subscriptions", len(*subs))
	return nil
}
