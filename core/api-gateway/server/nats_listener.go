package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/nats-io/nats.go"
)

// InvalidateTokenMessage Token 失效消息
type InvalidateTokenMessage struct {
	UserID    int64    `json:"user_id"`
	Username  string   `json:"username"`
	Tokens    []string `json:"tokens"` // 所有活跃 token hash 列表
	Reason    string   `json:"reason"` // department_changed, leader_changed
	Timestamp int64    `json:"timestamp"`
}

type OpenAPITokenRevokedMessage struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	TokenHash string `json:"token_hash"`
	ExpiresAt int64  `json:"expires_at"`
	Timestamp int64  `json:"timestamp"`
}

// TokenCommandHandler 处理 gateway token 相关命令。
type TokenCommandHandler struct {
	accessTokens  *AccessTokenValidator
	openAPITokens *OpenAPITokenValidator
}

// NewTokenCommandHandler 创建 TokenCommandHandler。
func NewTokenCommandHandler(accessTokens *AccessTokenValidator, openAPITokens *OpenAPITokenValidator) *TokenCommandHandler {
	return &TokenCommandHandler{accessTokens: accessTokens, openAPITokens: openAPITokens}
}

func (h *TokenCommandHandler) HandleOpenAPITokenRevoked(msg *nats.Msg) {
	ctx := context.Background()
	var message OpenAPITokenRevokedMessage
	if err := json.Unmarshal(msg.Data, &message); err != nil {
		logger.Errorf(ctx, "[NATSListener] 解析 OpenAPI Token 吊销消息失败: %v", err)
		return
	}
	if h.openAPITokens == nil {
		logger.Errorf(ctx, "[NATSListener] OpenAPI Token validator 未初始化")
		return
	}
	h.openAPITokens.MarkRevoked(message.TokenHash, message.ExpiresAt)
	logger.Infof(ctx, "[NATSListener] 收到 OpenAPI Token 吊销通知: userID=%d username=%s",
		message.UserID, message.Username)
}

// HandleTokenInvalidate 处理 token 失效命令。
func (h *TokenCommandHandler) HandleTokenInvalidate(msg *nats.Msg) {
	ctx := context.Background()

	var message InvalidateTokenMessage
	if err := json.Unmarshal(msg.Data, &message); err != nil {
		logger.Errorf(ctx, "[NATSListener] 解析失效消息失败: %v", err)
		return
	}

	h.handleTokenInvalidate(ctx, &message)
}

// handleTokenInvalidate 处理 token 失效消息
func (h *TokenCommandHandler) handleTokenInvalidate(ctx context.Context, message *InvalidateTokenMessage) {
	if h.accessTokens == nil {
		logger.Errorf(ctx, "[NATSListener] Access Token validator 未初始化")
		return
	}
	defaultExpireSeconds := config.GetGlobalSharedConfig().JWT.AccessTokenExpire
	if defaultExpireSeconds <= 0 {
		defaultExpireSeconds = 7 * 24 * 3600
	}
	defaultExpireTime := time.Now().Add(time.Duration(defaultExpireSeconds) * time.Second).Unix()

	for _, tokenHash := range message.Tokens {
		h.accessTokens.MarkRevoked(tokenHash, defaultExpireTime)
	}

	logger.Infof(ctx, "[NATSListener] 收到 token 失效通知: userID=%d, reason=%s, tokenCount=%d",
		message.UserID, message.Reason, len(message.Tokens))
}
