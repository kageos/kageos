package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// TokenPublisher 定义 hr-server 向其他服务发布 token 相关命令的能力。
type TokenPublisher interface {
	InvalidateToken(ctx context.Context, userID int64, username string, token string, reason string) error
	InvalidateUserTokens(ctx context.Context, userID int64, username string, sessions []*model.UserSession, reason string) error
	InvalidateOpenAPIToken(ctx context.Context, userID int64, username, tokenHash string, expiresAt *time.Time) error
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
	if err := p.conn.FlushTimeout(2 * time.Second); err != nil {
		return fmt.Errorf("确认 Token 失效消息失败: %w", err)
	}

	logger.Infof(ctx, "[GatewayTokenPublisher] 单 Token 失效通知已发送: userID=%d, reason=%s", userID, reason)
	return nil
}

// InvalidateUserTokens 通知网关立即清除已经在 HR 数据库中停用的会话缓存。
func (p *GatewayTokenPublisher) InvalidateUserTokens(ctx context.Context, userID int64, username string, sessions []*model.UserSession, reason string) error {
	if p == nil || p.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}

	tokenHashes := make([]string, 0, len(sessions))
	for _, session := range sessions {
		if session == nil || session.Token == "" {
			continue
		}
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
	if err := p.conn.FlushTimeout(2 * time.Second); err != nil {
		return fmt.Errorf("确认 Token 失效消息失败: %w", err)
	}

	logger.Infof(ctx, "[GatewayTokenPublisher] Token 失效通知已发送: userID=%d, reason=%s, tokenCount=%d", userID, reason, len(tokenHashes))
	return nil
}

// InvalidateOpenAPIToken notifies every gateway instance to evict an active
// OpenAPI Token cache entry and remember the revocation.
func (p *GatewayTokenPublisher) InvalidateOpenAPIToken(ctx context.Context, userID int64, username, tokenHash string, expiresAt *time.Time) error {
	if p == nil || p.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}
	var expiresAtUnix int64
	if expiresAt != nil {
		expiresAtUnix = expiresAt.Unix()
	}
	message := map[string]interface{}{
		"user_id":    userID,
		"username":   username,
		"token_hash": tokenHash,
		"expires_at": expiresAtUnix,
		"timestamp":  time.Now().Unix(),
	}
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化 OpenAPI Token 吊销消息失败: %w", err)
	}
	if err := p.conn.Publish(subjects.GatewayOpenAPITokenRevokedCommandSubject, data); err != nil {
		return fmt.Errorf("发布 OpenAPI Token 吊销消息失败: %w", err)
	}
	if err := p.conn.FlushTimeout(2 * time.Second); err != nil {
		return fmt.Errorf("确认 OpenAPI Token 吊销消息失败: %w", err)
	}
	logger.Infof(ctx, "[GatewayTokenPublisher] OpenAPI Token 吊销通知已发送: userID=%d username=%s", userID, username)
	return nil
}
