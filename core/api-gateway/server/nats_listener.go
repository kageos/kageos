package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/controlauth"
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

// RemoveBlacklistMessage Token 黑名单移除消息
type RemoveBlacklistMessage struct {
	UserID    int64    `json:"user_id"`
	Username  string   `json:"username"`
	Tokens    []string `json:"tokens"` // 要移除的 token hash 列表
	Reason    string   `json:"reason"` // user_relogin
	Timestamp int64    `json:"timestamp"`
}

// TokenCommandHandler 处理 gateway token 相关命令。
type TokenCommandHandler struct {
	tokenBlacklist *TokenBlacklist
	verifier       *controlauth.Verifier
}

// NewTokenCommandHandler 创建 TokenCommandHandler。
func NewTokenCommandHandler(tokenBlacklist *TokenBlacklist, verifier *controlauth.Verifier) *TokenCommandHandler {
	return &TokenCommandHandler{tokenBlacklist: tokenBlacklist, verifier: verifier}
}

// HandleTokenInvalidate 处理 token 失效命令。
func (h *TokenCommandHandler) HandleTokenInvalidate(msg *nats.Msg) {
	ctx := context.Background()
	if err := controlauth.VerifyNATSMessage(msg, h.verifier); err != nil {
		logger.Warnf(ctx, "[NATSListener] 拒绝未认证的 token 失效命令: %v", err)
		return
	}

	var message InvalidateTokenMessage
	if err := json.Unmarshal(msg.Data, &message); err != nil {
		logger.Errorf(ctx, "[NATSListener] 解析失效消息失败: %v", err)
		return
	}

	h.handleTokenInvalidate(ctx, &message)
}

// HandleRemoveBlacklist 处理移除 token 黑名单命令。
func (h *TokenCommandHandler) HandleRemoveBlacklist(msg *nats.Msg) {
	ctx := context.Background()
	if err := controlauth.VerifyNATSMessage(msg, h.verifier); err != nil {
		logger.Warnf(ctx, "[NATSListener] 拒绝未认证的黑名单移除命令: %v", err)
		return
	}

	var message RemoveBlacklistMessage
	if err := json.Unmarshal(msg.Data, &message); err != nil {
		logger.Errorf(ctx, "[NATSListener] 解析移除消息失败: %v", err)
		return
	}

	h.handleRemoveBlacklist(ctx, &message)
}

// handleTokenInvalidate 处理 token 失效消息
func (h *TokenCommandHandler) handleTokenInvalidate(ctx context.Context, message *InvalidateTokenMessage) {
	// 将所有 token hash 加入黑名单
	defaultExpireSeconds := config.GetGlobalSharedConfig().JWT.AccessTokenExpire
	if defaultExpireSeconds <= 0 {
		defaultExpireSeconds = 7 * 24 * 3600
	}
	defaultExpireTime := time.Now().Add(time.Duration(defaultExpireSeconds) * time.Second).Unix()

	for _, tokenHash := range message.Tokens {
		h.tokenBlacklist.AddTokenByHash(tokenHash, defaultExpireTime)
	}

	logger.Infof(ctx, "[NATSListener] 收到 token 失效通知: userID=%d, reason=%s, tokenCount=%d",
		message.UserID, message.Reason, len(message.Tokens))
}

// handleRemoveBlacklist 处理 token 黑名单移除消息
func (h *TokenCommandHandler) handleRemoveBlacklist(ctx context.Context, message *RemoveBlacklistMessage) {
	// 将所有 token hash 从黑名单移除
	for _, tokenHash := range message.Tokens {
		h.tokenBlacklist.RemoveTokenByHash(tokenHash)
	}

	logger.Infof(ctx, "[NATSListener] 收到移除黑名单通知: userID=%d, reason=%s, tokenCount=%d",
		message.UserID, message.Reason, len(message.Tokens))
}
