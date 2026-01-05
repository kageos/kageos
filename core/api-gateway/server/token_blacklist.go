package server

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// TokenBlacklist Token 黑名单管理器
type TokenBlacklist struct {
	mu        sync.RWMutex
	blacklist map[string]int64 // token_hash -> expire_time
}

// NewTokenBlacklist 创建 Token 黑名单管理器
func NewTokenBlacklist() *TokenBlacklist {
	bl := &TokenBlacklist{
		blacklist: make(map[string]int64),
	}

	// 启动定期清理任务
	go bl.cleanupExpired()

	return bl
}

// hashToken 计算 token hash
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// IsBlacklisted 检查 token 是否在黑名单中
func (b *TokenBlacklist) IsBlacklisted(token string) bool {
	tokenHash := hashToken(token)

	b.mu.RLock()
	defer b.mu.RUnlock()

	expireTime, exists := b.blacklist[tokenHash]
	if !exists {
		return false
	}

	// 检查是否过期
	if time.Now().Unix() > expireTime {
		// 过期，从黑名单移除
		b.mu.RUnlock()
		b.mu.Lock()
		delete(b.blacklist, tokenHash)
		b.mu.Unlock()
		b.mu.RLock()
		return false
	}

	return true
}

// AddToken 添加 token 到黑名单
func (b *TokenBlacklist) AddToken(token string, expireTime int64) {
	tokenHash := hashToken(token)
	b.AddTokenByHash(tokenHash, expireTime)
}

// AddTokenByHash 添加 token hash 到黑名单（用于 NATS 消息）
func (b *TokenBlacklist) AddTokenByHash(tokenHash string, expireTime int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.blacklist[tokenHash] = expireTime
	logger.Infof(nil, "[TokenBlacklist] Token 已加入黑名单: hash=%s, expireTime=%d", tokenHash, expireTime)
}

// RemoveToken 移除 token 从黑名单
func (b *TokenBlacklist) RemoveToken(token string) {
	tokenHash := hashToken(token)
	b.RemoveTokenByHash(tokenHash)
}

// RemoveTokenByHash 移除 token hash 从黑名单（用于 NATS 消息）
func (b *TokenBlacklist) RemoveTokenByHash(tokenHash string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.blacklist, tokenHash)
	logger.Infof(nil, "[TokenBlacklist] Token 已从黑名单移除: hash=%s", tokenHash)
}

// cleanupExpired 定期清理过期的黑名单记录
func (b *TokenBlacklist) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().Unix()

		b.mu.Lock()
		removedCount := 0
		for hash, expireTime := range b.blacklist {
			if now > expireTime {
				delete(b.blacklist, hash)
				removedCount++
			}
		}
		b.mu.Unlock()

		if removedCount > 0 {
			logger.Infof(nil, "[TokenBlacklist] 已清理 %d 条过期的黑名单记录", removedCount)
		}
	}
}

