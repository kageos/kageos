package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// TokenPublisher 定义 hr-server 向其他服务发布 token 相关命令的能力。
type TokenPublisher interface {
	InvalidateToken(ctx context.Context, userID int64, username string, token string, reason string) error
	InvalidateUserToken(ctx context.Context, userID int64, username string, reason string, userSessionRepo *repository.UserSessionRepository) error
	RemoveTokenFromBlacklist(ctx context.Context, userID int64, username string, oldSessions []*model.UserSession) error
}

// GatewayTokenPublisher 负责向 gateway 发布 token 相关命令。
type GatewayTokenPublisher struct {
	conn *nats.Conn
}

// NewGatewayTokenPublisher 创建 GatewayTokenPublisher。
func NewGatewayTokenPublisher(conn *nats.Conn) *GatewayTokenPublisher {
	return &GatewayTokenPublisher{conn: conn}
}

// hashToken 计算 token hash
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// InvalidateToken 使单个 token 失效（通过 NATS 通知 gateway）。
func (p *GatewayTokenPublisher) InvalidateToken(ctx context.Context, userID int64, username string, token string, reason string) error {
	if p == nil || p.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}

	message := map[string]interface{}{
		"user_id":   userID,
		"username":  username,
		"tokens":    []string{hashToken(token)},
		"reason":    reason,
		"timestamp": time.Now().Unix(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	if err := p.conn.Publish(subjects.GatewayTokenInvalidateCommandSubject, data); err != nil {
		return fmt.Errorf("发布消息失败: %w", err)
	}

	logger.Infof(ctx, "[GatewayTokenPublisher] 单 Token 失效通知已发送: userID=%d, reason=%s", userID, reason)
	return nil
}

// InvalidateUserToken 使用户的 token 失效（通过 NATS 通知 gateway）。
func (p *GatewayTokenPublisher) InvalidateUserToken(ctx context.Context, userID int64, username string, reason string, userSessionRepo *repository.UserSessionRepository) error {
	if p == nil || p.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}

	activeSessions, err := userSessionRepo.GetActiveSessionsByUserID(userID)
	if err != nil {
		return fmt.Errorf("查询活跃会话失败: %w", err)
	}

	tokenHashes := make([]string, 0, len(activeSessions))
	for _, session := range activeSessions {
		tokenHash := hashToken(session.Token)
		tokenHashes = append(tokenHashes, tokenHash)
	}

	message := map[string]interface{}{
		"user_id":   userID,
		"username":  username,
		"tokens":    tokenHashes,
		"reason":    reason,
		"timestamp": time.Now().Unix(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	if err := p.conn.Publish(subjects.GatewayTokenInvalidateCommandSubject, data); err != nil {
		return fmt.Errorf("发布消息失败: %w", err)
	}

	logger.Infof(ctx, "[GatewayTokenPublisher] Token 失效通知已发送: userID=%d, reason=%s, tokenCount=%d", userID, reason, len(tokenHashes))
	return nil
}

// RemoveTokenFromBlacklist 从黑名单移除 token（通过 NATS 通知 gateway）。
func (p *GatewayTokenPublisher) RemoveTokenFromBlacklist(ctx context.Context, userID int64, username string, oldSessions []*model.UserSession) error {
	if p == nil || p.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}

	tokenHashes := make([]string, 0, len(oldSessions))
	for _, session := range oldSessions {
		tokenHash := hashToken(session.Token)
		tokenHashes = append(tokenHashes, tokenHash)
	}

	message := map[string]interface{}{
		"user_id":   userID,
		"username":  username,
		"tokens":    tokenHashes,
		"reason":    "user_relogin",
		"timestamp": time.Now().Unix(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	if err := p.conn.Publish(subjects.GatewayTokenRemoveBlacklistCommandSubject, data); err != nil {
		return fmt.Errorf("发布消息失败: %w", err)
	}

	logger.Infof(ctx, "[GatewayTokenPublisher] 移除黑名单通知已发送: userID=%d, reason=user_relogin, tokenCount=%d", userID, len(tokenHashes))
	return nil
}
