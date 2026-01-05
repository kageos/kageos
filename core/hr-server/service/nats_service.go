package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/hr-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/hr-server/repository"
	appconfig "github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/nats-io/nats.go"
)

// NATSService NATS 服务
type NATSService struct {
	conn *nats.Conn
}

// NewNATSService 创建 NATS 服务
func NewNATSService() (*NATSService, error) {
	// 从全局配置获取 NATS 连接
	globalConfig := appconfig.GetGlobalSharedConfig()
	natsURL := globalConfig.Nats.URL
	if natsURL == "" {
		natsURL = "nats://127.0.0.1:4222" // 默认值
	}

	conn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("连接 NATS 失败: %w", err)
	}

	return &NATSService{
		conn: conn,
	}, nil
}

// hashToken 计算 token hash
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// InvalidateUserToken 使用户的 token 失效（通过 NATS 通知网关）
func (s *NATSService) InvalidateUserToken(ctx context.Context, userID int64, username string, reason string, userSessionRepo *repository.UserSessionRepository) error {
	// 获取用户的所有活跃 token（从 UserSession 表查询）
	activeSessions, err := userSessionRepo.GetActiveSessionsByUserID(userID)
	if err != nil {
		return fmt.Errorf("查询活跃会话失败: %w", err)
	}

	// 计算所有 token 的 hash
	tokenHashes := make([]string, 0, len(activeSessions))
	for _, session := range activeSessions {
		tokenHash := hashToken(session.Token)
		tokenHashes = append(tokenHashes, tokenHash)
	}

	subject := "hr.token.invalidate"

	message := map[string]interface{}{
		"user_id":   userID,
		"username":  username,
		"tokens":    tokenHashes, // 所有活跃 token hash 列表
		"reason":    reason,      // department_changed, leader_changed
		"timestamp": time.Now().Unix(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	if err := s.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("发布消息失败: %w", err)
	}

	logger.Infof(ctx, "[NATSService] Token 失效通知已发送: userID=%d, reason=%s, tokenCount=%d", userID, reason, len(tokenHashes))
	return nil
}

// RemoveTokenFromBlacklist 从黑名单移除 token（通过 NATS 通知网关）
func (s *NATSService) RemoveTokenFromBlacklist(ctx context.Context, userID int64, username string, oldSessions []*model.UserSession) error {
	// 计算所有旧 token 的 hash
	tokenHashes := make([]string, 0, len(oldSessions))
	for _, session := range oldSessions {
		tokenHash := hashToken(session.Token)
		tokenHashes = append(tokenHashes, tokenHash)
	}

	subject := "hr.token.remove_blacklist"

	message := map[string]interface{}{
		"user_id":   userID,
		"username":  username,
		"tokens":    tokenHashes,  // 要移除的旧 token hash 列表
		"reason":    "user_relogin", // 移除原因：用户重新登录
		"timestamp": time.Now().Unix(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	if err := s.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("发布消息失败: %w", err)
	}

	logger.Infof(ctx, "[NATSService] 移除黑名单通知已发送: userID=%d, reason=user_relogin, tokenCount=%d", userID, len(tokenHashes))
	return nil
}

// Close 关闭 NATS 连接
func (s *NATSService) Close() error {
	if s.conn != nil {
		s.conn.Close()
	}
	return nil
}

