package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	appconfig "github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/nats-io/nats.go"
)

// InvalidateTokenMessage Token 失效消息
type InvalidateTokenMessage struct {
	UserID    int64    `json:"user_id"`
	Username  string   `json:"username"`
	Tokens    []string `json:"tokens"`    // 所有活跃 token hash 列表
	Reason    string   `json:"reason"`    // department_changed, leader_changed
	Timestamp int64    `json:"timestamp"`
}

// RemoveBlacklistMessage Token 黑名单移除消息
type RemoveBlacklistMessage struct {
	UserID    int64    `json:"user_id"`
	Username  string   `json:"username"`
	Tokens    []string `json:"tokens"`    // 要移除的 token hash 列表
	Reason    string   `json:"reason"`    // user_relogin
	Timestamp int64    `json:"timestamp"`
}

// startNATSListener 启动 NATS 监听器
func (s *Server) startNATSListener(ctx context.Context) error {
	// 从全局配置获取 NATS 连接
	globalConfig := appconfig.GetGlobalSharedConfig()
	natsURL := globalConfig.Nats.URL
	if natsURL == "" {
		natsURL = "nats://127.0.0.1:4222" // 默认值
	}

	conn, err := nats.Connect(natsURL)
	if err != nil {
		return fmt.Errorf("连接 NATS 失败: %w", err)
	}

	// 订阅主题1：hr.token.invalidate（token 失效通知）
	subject1 := "hr.token.invalidate"
	_, err = conn.Subscribe(subject1, func(msg *nats.Msg) {
		var message InvalidateTokenMessage
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			logger.Errorf(ctx, "[NATSListener] 解析失效消息失败: %v", err)
			return
		}

		// 处理 token 失效消息
		s.handleTokenInvalidate(ctx, &message)
	})

	if err != nil {
		conn.Close()
		return fmt.Errorf("订阅 NATS 主题失败: %w", err)
	}

	// 订阅主题2：hr.token.remove_blacklist（移除黑名单通知）
	subject2 := "hr.token.remove_blacklist"
	_, err = conn.Subscribe(subject2, func(msg *nats.Msg) {
		var message RemoveBlacklistMessage
		if err := json.Unmarshal(msg.Data, &message); err != nil {
			logger.Errorf(ctx, "[NATSListener] 解析移除消息失败: %v", err)
			return
		}

		// 处理移除黑名单消息
		s.handleRemoveBlacklist(ctx, &message)
	})

	if err != nil {
		conn.Close()
		return fmt.Errorf("订阅 NATS 主题失败: %w", err)
	}

	logger.Infof(ctx, "[NATSListener] 已订阅主题: %s, %s", subject1, subject2)
	return nil
}

// handleTokenInvalidate 处理 token 失效消息
func (s *Server) handleTokenInvalidate(ctx context.Context, message *InvalidateTokenMessage) {
	// 将所有 token hash 加入黑名单
	// 注意：这里需要知道 token 的过期时间，可以从 JWT 解析或使用默认过期时间
	defaultExpireTime := time.Now().Add(7 * 24 * time.Hour).Unix() // 默认7天过期

	for _, tokenHash := range message.Tokens {
		s.tokenBlacklist.AddTokenByHash(tokenHash, defaultExpireTime)
	}

	logger.Infof(ctx, "[NATSListener] 收到 token 失效通知: userID=%d, reason=%s, tokenCount=%d",
		message.UserID, message.Reason, len(message.Tokens))
}

// handleRemoveBlacklist 处理 token 黑名单移除消息
func (s *Server) handleRemoveBlacklist(ctx context.Context, message *RemoveBlacklistMessage) {
	// 将所有 token hash 从黑名单移除
	for _, tokenHash := range message.Tokens {
		s.tokenBlacklist.RemoveTokenByHash(tokenHash)
	}

	logger.Infof(ctx, "[NATSListener] 收到移除黑名单通知: userID=%d, reason=%s, tokenCount=%d",
		message.UserID, message.Reason, len(message.Tokens))
}

