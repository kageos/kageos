package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// LicenseService License 服务
type LicenseService struct {
	natsConn      *nats.Conn
	licensePath   string
	encryptionKey []byte
	manager       *license.Manager
	mu            sync.RWMutex
}

// NewLicenseService 创建 License 服务
func NewLicenseService(natsConn *nats.Conn, licensePath string, encryptionKey []byte) *LicenseService {
	return &LicenseService{
		natsConn:      natsConn,
		licensePath:   licensePath,
		encryptionKey:  encryptionKey,
		manager:       license.GetManager(),
	}
}

// LoadAndPublish 加载并发布 License 密钥
func (s *LicenseService) LoadAndPublish(ctx context.Context) error {
	// 1. 加载 License
	if err := s.manager.LoadLicense(s.licensePath); err != nil {
		return fmt.Errorf("failed to load license: %w", err)
	}

	// 2. 发布密钥
	return s.PublishKey(ctx)
}

// PublishKey 发布 License 密钥
func (s *LicenseService) PublishKey(ctx context.Context) error {
	s.mu.RLock()
	lic := s.manager.GetLicense()
	s.mu.RUnlock()

	if lic == nil {
		// 没有 License，不发布密钥（社区版）
		logger.Infof(ctx, "[License Service] No license found, skipping key distribution")
		return nil
	}

	// 序列化 License
	licenseData, err := json.Marshal(lic)
	if err != nil {
		return fmt.Errorf("failed to marshal license: %w", err)
	}

	// 加密 License
	encrypted, err := s.encryptLicense(licenseData)
	if err != nil {
		return fmt.Errorf("failed to encrypt license: %w", err)
	}

	// 构建消息
	msg := &LicenseKeyMessage{
		EncryptedLicense: base64.StdEncoding.EncodeToString(encrypted),
		Algorithm:        "aes-256-gcm",
		Timestamp:        getCurrentTimestamp(),
	}

	// 发布到 NATS
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := s.natsConn.Publish(subjects.GetControlLicenseKeySubject(), data); err != nil {
		return fmt.Errorf("failed to publish license key: %w", err)
	}

	logger.Infof(ctx, "[License Service] Published license key to NATS")
	return nil
}

// CheckExpiry 检查 License 是否过期
func (s *LicenseService) CheckExpiry(ctx context.Context) error {
	s.mu.RLock()
	lic := s.manager.GetLicense()
	s.mu.RUnlock()

	if lic == nil {
		return nil // 社区版，无需检查
	}

	if !lic.IsValid() {
		logger.Warnf(ctx, "[License Service] License has expired")
		return fmt.Errorf("license has expired")
	}

	return nil
}

// GetStatus 获取 License 状态
func (s *LicenseService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	lic := s.manager.GetLicense()
	s.mu.RUnlock()

	status := map[string]interface{}{
		"is_valid":     false,
		"is_community": true,
		"edition":     "community",
	}

	if lic != nil && lic.IsValid() {
		status["is_valid"] = true
		status["is_community"] = false
		status["edition"] = lic.Edition
		status["customer"] = lic.Customer
		status["expires_at"] = lic.ExpiresAt
		status["features"] = lic.Features
	}

	return status
}

// encryptLicense 加密 License（AES-256-GCM）
func (s *LicenseService) encryptLicense(data []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// LicenseKeyMessage License 密钥消息
type LicenseKeyMessage struct {
	EncryptedLicense string `json:"encrypted_license"` // 加密的 License（Base64 编码）
	Algorithm        string `json:"algorithm"`         // 加密算法（如 "aes-256-gcm"）
	Timestamp        int64  `json:"timestamp"`        // 时间戳
}

// getCurrentTimestamp 获取当前时间戳
func getCurrentTimestamp() int64 {
	return int64(time.Now().Unix())
}
