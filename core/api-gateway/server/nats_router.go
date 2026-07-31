package server

import (
	"context"
	"fmt"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// RegisterNATS 注册 api-gateway 的所有 NATS 订阅。
// subject 真值统一放在 pkg/subjects，这里只负责路由装配。
func RegisterNATS(ctx context.Context, conn *nats.Conn, subs *[]*nats.Subscription, tokenHandler *TokenCommandHandler) error {
	var err error
	var sub *nats.Subscription

	sub, err = conn.Subscribe(subjects.GatewayTokenInvalidateCommandSubject, tokenHandler.HandleTokenInvalidate)
	if err != nil {
		return fmt.Errorf("subscribe token invalidate: %w", err)
	}
	*subs = append(*subs, sub)

	sub, err = conn.Subscribe(subjects.GatewayOpenAPITokenRevokedCommandSubject, tokenHandler.HandleOpenAPITokenRevoked)
	if err != nil {
		return fmt.Errorf("subscribe OpenAPI Token revoked: %w", err)
	}
	*subs = append(*subs, sub)

	logger.Infof(ctx, "[NATS Router] Registered %d subscriptions", len(*subs))
	return nil
}
