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
	"os"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// LicenseStatus License 状态响应结构体
type LicenseStatus struct {
	IsValid     bool              `json:"is_valid"`              // License 是否有效
	IsCommunity bool              `json:"is_community"`          // 是否为社区版
	Edition     string            `json:"edition"`               // 版本类型：community, professional, enterprise, flagship
	Customer    string            `json:"customer,omitempty"`    // 客户名称（可选）
	Description string            `json:"description,omitempty"` // License 描述（可选）
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`  // 过期时间（可选）
	Features    *license.Features `json:"features,omitempty"`    // 功能开关（可选）
}

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
		encryptionKey: encryptionKey,
		manager:       license.GetManager(),
	}
}

// LoadAndPublish 加载并发布 License 密钥
// 注意：启动时推送一次，主要用于通知已运行的服务（如果有）
// 新启动的服务主要通过请求-响应模式主动获取
func (s *LicenseService) LoadAndPublish(ctx context.Context) error {
	// 1. 加载 License
	if err := s.manager.LoadLicense(s.licensePath); err != nil {
		return fmt.Errorf("failed to load license: %w", err)
	}

	// 2. 发布密钥（启动时推送一次，通知已运行的服务）
	// 注意：新启动的服务主要通过请求-响应模式主动获取，不依赖此推送
	return s.PublishKey(ctx)
}

// PublishKey 发布 License 密钥（推送模式）
// 使用场景：
//  1. License 激活/更新后主动推送，通知所有服务
//  2. Control Service 启动时推送一次（可选，主要用于通知已运行的服务）
//
// 注意：新启动的服务主要通过请求-响应模式主动获取，不依赖此推送
func (s *LicenseService) PublishKey(ctx context.Context) error {
	s.mu.RLock()
	lic := s.manager.GetLicense()
	s.mu.RUnlock()

	if lic == nil {
		// 没有 License，不发布密钥（社区版）
		return nil
	}

	// 序列化 License
	licenseData, err := json.Marshal(lic)
	if err != nil {
		logger.Errorf(ctx, "[License Service] 序列化失败: %v", err)
		return fmt.Errorf("failed to marshal license: %w", err)
	}

	// 加密 License
	encrypted, err := s.encryptLicense(licenseData)
	if err != nil {
		logger.Errorf(ctx, "[License Service] 加密失败: %v", err)
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
		logger.Errorf(ctx, "[License Service] Failed to marshal message: %v", err)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 检查 NATS 连接状态
	if s.natsConn == nil {
		logger.Errorf(ctx, "[License Service] NATS connection is nil")
		return fmt.Errorf("NATS connection is nil")
	}
	if !s.natsConn.IsConnected() {
		logger.Errorf(ctx, "[License Service] NATS connection is not connected")
		return fmt.Errorf("NATS connection is not connected")
	}

	if err := s.natsConn.Publish(subjects.GetControlLicenseKeySubject(), data); err != nil {
		logger.Errorf(ctx, "[License Service] Failed to publish license key: %v", err)
		return fmt.Errorf("failed to publish license key: %w", err)
	}

	logger.Infof(ctx, "[License Service] Published license key to NATS (subject: %s)", subjects.GetControlLicenseKeySubject())
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
func (s *LicenseService) GetStatus() *LicenseStatus {
	s.mu.RLock()
	lic := s.manager.GetLicense()
	s.mu.RUnlock()

	status := &LicenseStatus{
		IsValid:     false,
		IsCommunity: true,
		Edition:     "community",
	}

	if lic != nil && lic.IsValid() {
		status.IsValid = true
		status.IsCommunity = false
		status.Edition = lic.Edition
		status.Customer = lic.Customer
		status.Description = lic.Description

		// 设置过期时间（如果有效）
		if !lic.ExpiresAt.IsZero() {
			status.ExpiresAt = &lic.ExpiresAt.Time
		}

		// 设置功能开关
		status.Features = &lic.Features
	}

	return status
}

// GetLicense 获取当前 License（供 Server 使用）
func (s *LicenseService) GetLicense() *license.License {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.manager.GetLicense()
}

// BuildKeyMessage 构建 License 密钥消息（用于请求-响应）
func (s *LicenseService) BuildKeyMessage(ctx context.Context) (*LicenseKeyMessage, error) {
	s.mu.RLock()
	lic := s.manager.GetLicense()
	s.mu.RUnlock()

	if lic == nil {
		// 没有 License，返回空消息（社区版）
		return &LicenseKeyMessage{
			EncryptedLicense: "",
			Algorithm:        "",
			Timestamp:        getCurrentTimestamp(),
		}, nil
	}

	// 序列化 License
	licenseData, err := json.Marshal(lic)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal license: %w", err)
	}

	// 加密 License
	encrypted, err := s.encryptLicense(licenseData)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt license: %w", err)
	}

	// 构建消息
	msg := &LicenseKeyMessage{
		EncryptedLicense: base64.StdEncoding.EncodeToString(encrypted),
		Algorithm:        "aes-256-gcm",
		Timestamp:        getCurrentTimestamp(),
	}

	return msg, nil
}

// Activate 激活 License（上传License文件并激活）
// 参数：
//   - licenseData: License 文件内容（JSON格式）
//
// 返回：
//   - error: 如果激活失败，返回错误
func (s *LicenseService) Activate(ctx context.Context, licenseData []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. 验证License格式（先解析JSON，验证基本格式）
	var licenseFile map[string]interface{}
	if err := json.Unmarshal(licenseData, &licenseFile); err != nil {
		logger.Errorf(ctx, "[License Service] License JSON 格式错误: %v", err)
		return fmt.Errorf("invalid license format: %w", err)
	}

	// 验证 License 文件结构
	// 1. 检查是否有 license 字段
	if _, ok := licenseFile["license"]; !ok {
		logger.Errorf(ctx, "[License Service] License 文件缺少 'license' 字段")
		return fmt.Errorf("license file missing 'license' field")
	}

	// 2. 检查是否有 signature 字段
	signatureValue, ok := licenseFile["signature"]
	if !ok {
		logger.Errorf(ctx, "[License Service] License 文件缺少 'signature' 字段")
		return fmt.Errorf("license file missing 'signature' field")
	}

	// 3. 检查 signature 字段是否为空
	signatureStr, ok := signatureValue.(string)
	if !ok {
		logger.Errorf(ctx, "[License Service] 'signature' 字段类型错误，应该是字符串")
		return fmt.Errorf("signature field must be a string")
	}

	if signatureStr == "" {
		logger.Errorf(ctx, "[License Service] 'signature' 字段为空")
		return fmt.Errorf("signature field is empty")
	}

	// 2. 保存License文件到本地
	if err := os.WriteFile(s.licensePath, licenseData, 0600); err != nil {
		return fmt.Errorf("failed to save license file: %w", err)
	}

	// 3. 重新加载License（会验证签名和有效性）
	if err := s.manager.LoadLicense(s.licensePath); err != nil {
		// 如果加载失败，删除已保存的文件
		os.Remove(s.licensePath)
		return fmt.Errorf("failed to load license: %w", err)
	}

	// 获取激活后的License信息
	lic := s.manager.GetLicense()
	if lic == nil {
		return fmt.Errorf("license loaded but is nil")
	}

	// 打印激活信息
	logger.Infof(ctx, "[License Service] License 激活成功 - ID: %s, 版本: %s, 客户: %s", lic.ID, lic.Edition, lic.Customer)

	// 4. 推送License到NATS（推送模式：直接推送License内容）
	// 各服务订阅推送主题，收到后直接解密并刷新，无需再请求
	// 使用 goroutine 异步发布，避免阻塞 HTTP 响应
	go func() {
		if err := s.PublishKey(ctx); err != nil {
			logger.Warnf(ctx, "[License Service] Failed to publish license key: %v", err)
			// 发布失败不影响激活，因为各服务可以通过请求获取
		}
	}()

	// 5. 发布刷新指令（备用：通知服务主动请求，用于启动时或推送失败时）
	// 注意：推送模式是主要方式，客户端收到推送后直接刷新，不需要此指令
	// 刷新指令主要用于：服务启动时主动请求，或推送失败时的备用方案
	// 使用 goroutine 异步发布，避免阻塞 HTTP 响应
	go func() {
		if err := s.PublishRefresh(ctx); err != nil {
			logger.Warnf(ctx, "[License Service] Failed to publish refresh instruction: %v", err)
			// 刷新指令发布失败不影响激活（推送模式已足够）
		}
	}()

	return nil
}

// Deactivate 注销 License（删除激活信息，回到社区版）
// 主要用于测试场景，方便重新激活
func (s *LicenseService) Deactivate(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. 删除 License 文件
	if _, err := os.Stat(s.licensePath); err == nil {
		if err := os.Remove(s.licensePath); err != nil {
			logger.Errorf(ctx, "[License Service] 删除 License 文件失败: %v", err)
			return fmt.Errorf("failed to remove license file: %w", err)
		}
	}

	// 2. 清除 Manager 中的 License（设置为 nil，回到社区版）
	s.manager.ClearLicense()

	// 3. 发布刷新指令，通知所有服务清除本地 License
	// 注意：这里发布一个特殊的"清除"指令，让所有服务删除本地存储的 License
	if err := s.PublishDeactivate(ctx); err != nil {
		logger.Warnf(ctx, "[License Service] Failed to publish deactivate instruction: %v", err)
		// 发布失败不影响注销，因为各服务可以通过请求获取（会返回社区版）
	}

	logger.Infof(ctx, "[License Service] License 注销成功，系统已回到社区版")
	return nil
}

// PublishDeactivate 发布注销指令（通知所有服务清除 License）
func (s *LicenseService) PublishDeactivate(ctx context.Context) error {
	deactivateMsg := LicenseInstructionMessage{
		Action:    "deactivate",
		Timestamp: getCurrentTimestamp(),
	}

	data, err := json.Marshal(deactivateMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal deactivate message: %w", err)
	}

	if err := s.natsConn.Publish(subjects.GetControlLicenseKeyRefreshSubject(), data); err != nil {
		return fmt.Errorf("failed to publish deactivate instruction: %w", err)
	}

	logger.Infof(ctx, "[License Service] Published deactivate instruction to NATS")
	return nil
}

// PublishRefresh 发布刷新指令（通知所有服务刷新密钥）
// 优化：直接带上 License 内容，客户端收到后直接刷新，无需再请求
func (s *LicenseService) PublishRefresh(ctx context.Context) error {
	s.mu.RLock()
	lic := s.manager.GetLicense()
	s.mu.RUnlock()

	refreshMsg := LicenseInstructionMessage{
		Action:    "refresh",
		Timestamp: getCurrentTimestamp(),
	}

	// 如果有 License，直接带上加密的 License 内容
	if lic != nil {
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

		refreshMsg.EncryptedLicense = base64.StdEncoding.EncodeToString(encrypted)
		refreshMsg.Algorithm = "aes-256-gcm"
	}

	data, err := json.Marshal(refreshMsg)
	if err != nil {
		logger.Errorf(ctx, "[License Service] Failed to marshal refresh message: %v", err)
		return fmt.Errorf("failed to marshal refresh message: %w", err)
	}

	// 检查 NATS 连接状态
	if s.natsConn == nil {
		logger.Errorf(ctx, "[License Service] NATS connection is nil")
		return fmt.Errorf("NATS connection is nil")
	}
	if !s.natsConn.IsConnected() {
		logger.Errorf(ctx, "[License Service] NATS connection is not connected")
		return fmt.Errorf("NATS connection is not connected")
	}

	if err := s.natsConn.Publish(subjects.GetControlLicenseKeyRefreshSubject(), data); err != nil {
		logger.Errorf(ctx, "[License Service] Failed to publish refresh instruction: %v", err)
		return fmt.Errorf("failed to publish refresh instruction: %w", err)
	}

	return nil
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
	Timestamp        int64  `json:"timestamp"`         // 时间戳
}

// LicenseInstructionMessage License 指令消息（用于刷新和注销）
type LicenseInstructionMessage struct {
	Action           string `json:"action"`                      // 指令类型：refresh（刷新）、deactivate（注销）
	Timestamp        int64  `json:"timestamp"`                   // 时间戳
	EncryptedLicense string `json:"encrypted_license,omitempty"` // 加密的 License（Base64 编码，可选，refresh 时携带）
	Algorithm        string `json:"algorithm,omitempty"`         // 加密算法（如 "aes-256-gcm"，可选）
}

// getCurrentTimestamp 获取当前时间戳
func getCurrentTimestamp() int64 {
	return int64(time.Now().Unix())
}
