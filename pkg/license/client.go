package license

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// Client License 客户端
// 用于各服务实例获取和更新 License 密钥
type Client struct {
	natsConn            *nats.Conn
	transport           *NATSTransport
	encryptionKey       []byte
	keyStore            *encryptedLicenseStore
	manager             *Manager
	pushSubscription    *nats.Subscription // 推送主题订阅（接收License内容）
	refreshSubscription *nats.Subscription // 刷新指令订阅（通知主动请求）
	mu                  sync.RWMutex
}

// NewClient 创建 License 客户端
// 参数：
//   - natsConn: NATS 连接
//   - encryptionKey: 加密密钥（32字节，与 Control Service 相同）
//   - keyPath: 本地密钥文件路径（可选，默认：~/.ai-agent-os/license.key）
func NewClient(natsConn *nats.Conn, encryptionKey []byte, keyPath string) (*Client, error) {
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes")
	}

	keyStore, err := newEncryptedLicenseStore(keyPath)
	if err != nil {
		return nil, err
	}

	client := &Client{
		natsConn:      natsConn,
		transport:     NewNATSTransport(natsConn),
		encryptionKey: encryptionKey,
		keyStore:      keyStore,
		manager:       GetManager(),
	}

	return client, nil
}

// Start 启动 License 客户端
// 1. 尝试从本地加载密钥
// 2. 如果本地没有，通过 NATS 请求获取
// 3. 订阅推送主题，直接接收推送的License并刷新
// 4. 订阅刷新指令主题（备用，用于启动时主动请求）
func (c *Client) Start(ctx context.Context) error {
	logger.Infof(ctx, "[License Client] Starting license client...")

	// 1. 尝试从本地加载密钥
	if err := c.loadLocalKey(ctx); err == nil {
		logger.Infof(ctx, "[License Client] Loaded license key from local file: %s", c.keyStore.Path())
	} else {
		logger.Warnf(ctx, "[License Client] Failed to load local key: %v, will request from Control Service", err)

		// 2. 从 NATS 请求获取密钥（启动时主动请求）
		if err := c.requestKey(ctx); err != nil {
			logger.Warnf(ctx, "[License Client] Failed to request license key: %v, using community edition", err)
			// 请求失败，使用社区版（不返回错误）
			return nil
		}
	}

	// 3. 订阅推送主题（接收推送的License，直接刷新）
	if err := c.subscribePush(ctx); err != nil {
		logger.Warnf(ctx, "[License Client] Failed to subscribe push topic: %v", err)
		// 订阅失败不影响使用，只是无法接收推送
	}

	// 4. 订阅刷新指令主题（备用，用于启动时主动请求）
	if err := c.subscribeRefresh(ctx); err != nil {
		logger.Warnf(ctx, "[License Client] Failed to subscribe refresh topic: %v", err)
		// 订阅失败不影响使用，只是无法接收刷新指令
	}

	logger.Infof(ctx, "[License Client] License client started successfully")
	return nil
}

// loadLocalKey 从本地加载密钥
func (c *Client) loadLocalKey(ctx context.Context) error {
	data, err := c.keyStore.Read()
	if err != nil {
		return err
	}

	// 解密并设置 License
	return c.applyEncryptedLicense(ctx, data, false)
}

// requestKey 通过 NATS 请求获取密钥
func (c *Client) requestKey(ctx context.Context) error {
	logger.Infof(ctx, "[License Client] Requesting license key from Control Service...")

	resp, err := c.transport.RequestKey(ctx, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to request license key: %w", err)
	}

	// 检查是否有 License（社区版时 EncryptedLicense 为空）
	if resp.EncryptedLicense == "" {
		logger.Infof(ctx, "[License Client] Control Service returned empty license (community edition), using community edition")
		return nil // 社区版，不需要设置 License
	}

	return c.applyLicenseKeyMessage(ctx, resp, false)
}

// subscribePush 订阅推送主题（接收推送的License，直接刷新）
func (c *Client) subscribePush(ctx context.Context) error {
	subject := subjects.LicenseKeyUpdatedEventSubject
	logger.Infof(ctx, "[License Client] 准备订阅推送主题: %s", subject)
	logger.Infof(ctx, "[License Client] NATS 连接状态: Connected=%v, URL=%s", c.natsConn.IsConnected(), c.natsConn.ConnectedUrl())

	sub, err := c.transport.SubscribeKeyUpdates(func(msg *nats.Msg) {
		logger.Infof(ctx, "[License Client] 收到推送主题消息: %s, 数据长度: %d 字节", subject, len(msg.Data))
		c.handlePush(ctx, msg)
	})
	if err != nil {
		logger.Errorf(ctx, "[License Client] ❌ 订阅推送主题失败: %s, 错误: %v", subject, err)
		return fmt.Errorf("failed to subscribe push topic: %w", err)
	}

	c.mu.Lock()
	c.pushSubscription = sub
	c.mu.Unlock()

	logger.Infof(ctx, "[License Client] ✅ 成功订阅推送主题: %s", subject)
	return nil
}

// handlePush 处理推送的License（直接刷新，不需要再请求）
func (c *Client) handlePush(ctx context.Context, msg *nats.Msg) {
	logger.Infof(ctx, "[License Client] Received pushed license, refreshing...")

	keyMsg, err := decodeLicenseKeyPayload(msg.Data)
	if err != nil {
		logger.Errorf(ctx, "[License Client] Failed to unmarshal pushed license: %v", err)
		return
	}

	// 如果推送的是空消息（社区版），跳过
	if keyMsg.EncryptedLicense == "" {
		logger.Infof(ctx, "[License Client] Pushed license is empty (community edition), skipping")
		return
	}

	if err := c.applyLicenseKeyMessage(ctx, keyMsg, true); err != nil {
		logger.Errorf(ctx, "[License Client] Failed to refresh pushed license: %v", err)
	}
}

// subscribeRefresh 订阅刷新指令主题（备用，用于启动时主动请求）
func (c *Client) subscribeRefresh(ctx context.Context) error {
	subject := subjects.LicenseKeyRefreshEventSubject
	logger.Infof(ctx, "[License Client] 准备订阅刷新指令主题: %s", subject)
	logger.Infof(ctx, "[License Client] NATS 连接状态: Connected=%v, URL=%s", c.natsConn.IsConnected(), c.natsConn.ConnectedUrl())

	sub, err := c.transport.SubscribeRefreshInstructions(func(msg *nats.Msg) {
		logger.Infof(ctx, "[License Client] 收到刷新指令主题消息: %s, 数据长度: %d 字节", subject, len(msg.Data))
		c.handleRefresh(ctx, msg)
	})
	if err != nil {
		logger.Errorf(ctx, "[License Client] ❌ 订阅刷新指令主题失败: %s, 错误: %v", subject, err)
		return fmt.Errorf("failed to subscribe refresh topic: %w", err)
	}

	c.mu.Lock()
	c.refreshSubscription = sub
	c.mu.Unlock()

	logger.Infof(ctx, "[License Client] ✅ 成功订阅刷新指令主题: %s", subject)
	return nil
}

// handleRefresh 处理刷新指令（备用方案）
// 注意：推送模式是主要方式，客户端收到推送后直接刷新
// 刷新指令主要用于：服务启动时主动请求，或推送失败时的备用方案
// 同时支持注销指令（action: "deactivate"）
func (c *Client) handleRefresh(ctx context.Context, msg *nats.Msg) {
	logger.Infof(ctx, "[License Client] ========================================")
	logger.Infof(ctx, "[License Client] 收到刷新/注销指令消息")
	logger.Infof(ctx, "[License Client] 消息数据: %s", string(msg.Data))

	// 解析消息，检查是否是注销指令
	instructionMsg, err := decodeLicenseInstructionPayload(msg.Data)
	if err != nil {
		logger.Errorf(ctx, "[License Client] ❌ Failed to unmarshal refresh message: %v", err)
		logger.Errorf(ctx, "[License Client] 原始消息数据: %s", string(msg.Data))
		return
	}

	logger.Infof(ctx, "[License Client] 解析后的指令: action=%s, timestamp=%d", instructionMsg.Action, instructionMsg.Timestamp)

	// 检查是否是注销指令
	if instructionMsg.Action == "deactivate" {
		logger.Infof(ctx, "[License Client] ========================================")
		logger.Infof(ctx, "[License Client] 🔴 检测到注销指令，开始清除 License...")
		logger.Infof(ctx, "[License Client] ========================================")
		c.handleDeactivate(ctx)
		return
	}

	// 默认是刷新指令
	// 优化：如果消息中直接包含了 License 内容，直接刷新，无需再请求
	if instructionMsg.EncryptedLicense != "" {
		logger.Infof(ctx, "[License Client] Received refresh instruction with license content, refreshing directly...")

		if err := c.applyInstructionLicense(ctx, instructionMsg); err != nil {
			logger.Errorf(ctx, "[License Client] Failed to refresh license from instruction: %v", err)
			return
		}

		logger.Infof(ctx, "[License Client] License key refreshed successfully from instruction")
		return
	}

	// 如果消息中没有 License 内容，主动请求（向后兼容，或社区版场景）
	logger.Infof(ctx, "[License Client] Received refresh instruction without license content, requesting new license key...")

	// 主动请求新License（请求-响应模式）
	if err := c.requestKey(ctx); err != nil {
		logger.Errorf(ctx, "[License Client] Failed to refresh license key: %v", err)
		return
	}

	logger.Infof(ctx, "[License Client] License key refreshed successfully via request")
}

// handleDeactivate 处理注销指令（清除 License，回到社区版）
func (c *Client) handleDeactivate(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	logger.Infof(ctx, "[License Client] ========================================")
	logger.Infof(ctx, "[License Client] 开始处理注销指令...")

	// 1. 清除 Manager 中的 License
	c.manager.ClearLicense()
	logger.Infof(ctx, "[License Client] ✅ License 状态已清除，回到社区版")

	// 2. 删除本地存储的 License 密钥文件
	if c.keyStore.Exists() {
		if err := c.keyStore.Delete(); err != nil {
			logger.Warnf(ctx, "[License Client] ❌ 删除本地 License 密钥文件失败: %v", err)
		} else {
			logger.Infof(ctx, "[License Client] ✅ 本地 License 密钥文件已删除: %s", c.keyStore.Path())
		}
	} else {
		logger.Infof(ctx, "[License Client] 本地 License 密钥文件不存在，跳过删除")
	}

	logger.Infof(ctx, "[License Client] ========================================")
	logger.Infof(ctx, "[License Client] ✅ License 注销成功，系统已回到社区版")
	logger.Infof(ctx, "[License Client] ========================================")
}

func (c *Client) applyInstructionLicense(ctx context.Context, instruction *LicenseInstructionMessage) error {
	if instruction == nil || instruction.EncryptedLicense == "" {
		return fmt.Errorf("instruction has no encrypted license")
	}

	encrypted, err := decodeEncryptedLicenseBase64(instruction.EncryptedLicense)
	if err != nil {
		return err
	}

	return c.applyEncryptedLicense(ctx, encrypted, true)
}

func (c *Client) applyLicenseKeyMessage(ctx context.Context, keyMsg *LicenseKeyMessage, skipIfUnchanged bool) error {
	if keyMsg == nil || keyMsg.EncryptedLicense == "" {
		return fmt.Errorf("license key message has no encrypted license")
	}

	encrypted, err := decodeEncryptedLicenseBase64(keyMsg.EncryptedLicense)
	if err != nil {
		return err
	}

	if skipIfUnchanged {
		same, err := c.keyStore.IsSame(encrypted)
		if err == nil && same {
			logger.Infof(ctx, "[License Client] Pushed license unchanged, skipping update")
			return nil
		}
		if err != nil {
			logger.Warnf(ctx, "[License Client] Failed to compare local license key: %v", err)
		}
	}

	return c.applyEncryptedLicense(ctx, encrypted, true)
}

func (c *Client) applyEncryptedLicense(ctx context.Context, encrypted []byte, persist bool) error {
	logger.Infof(ctx, "[License Client] 检测到 License 更新，正在刷新...")

	if err := c.setLicenseFromEncrypted(ctx, encrypted); err != nil {
		return fmt.Errorf("failed to decrypt license: %w", err)
	}

	if !persist {
		return nil
	}

	if err := c.keyStore.Write(encrypted); err != nil {
		logger.Warnf(ctx, "[License Client] Failed to save license key to local: %v", err)
	} else {
		logger.Infof(ctx, "[License Client] License 已刷新并保存到本地: %s", c.keyStore.Path())
	}

	return nil
}

// setLicenseFromEncrypted 从加密数据设置 License
func (c *Client) setLicenseFromEncrypted(ctx context.Context, encrypted []byte) error {
	// 解密
	decrypted, err := c.decryptLicense(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt license: %w", err)
	}

	// 解析 License
	var lic License
	if err := json.Unmarshal(decrypted, &lic); err != nil {
		return fmt.Errorf("failed to unmarshal license: %w", err)
	}

	// 验证 License（过期时间）
	if !lic.IsValid() {
		return fmt.Errorf("license is invalid or expired")
	}
	if !lic.IsEnterpriseEdition() {
		return fmt.Errorf("unsupported license edition: %s", lic.Edition)
	}

	// 设置 License 到 Manager（注意：这里不验证签名，因为签名验证需要在加载 License 文件时完成）
	// 从加密密钥获取的 License 已经由 Control Service 验证过签名
	c.manager.setLicense(&lic)

	// 打印详细的激活信息
	logger.Infof(ctx, "[License Client] ========================================")
	logger.Infof(ctx, "[License Client] License 激活成功！")
	logger.Infof(ctx, "[License Client] ========================================")
	logger.Infof(ctx, "[License Client] License ID: %s", lic.ID)
	logger.Infof(ctx, "[License Client] 版本: %s", lic.Edition)
	logger.Infof(ctx, "[License Client] 客户: %s", lic.Customer)
	logger.Infof(ctx, "[License Client] 签发时间: %v", lic.IssuedAt.Time)
	logger.Infof(ctx, "[License Client] 过期时间: %v", lic.ExpiresAt.Time)
	if lic.MaxApps > 0 {
		logger.Infof(ctx, "[License Client] 最大应用数: %d", lic.MaxApps)
	}
	if lic.MaxUsers > 0 {
		logger.Infof(ctx, "[License Client] 最大用户数: %d", lic.MaxUsers)
	}
	logger.Infof(ctx, "[License Client] 功能列表:")
	if lic.Features.OperateLog {
		logger.Infof(ctx, "[License Client]   - operate_log: ✓")
	}
	if lic.Features.Permission {
		logger.Infof(ctx, "[License Client]   - permission: ✓")
	}
	logger.Infof(ctx, "[License Client] ========================================")

	return nil
}

// decryptLicense 解密 License
func (c *Client) decryptLicense(encrypted []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(encrypted) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// Stop 停止 License 客户端
func (c *Client) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.pushSubscription != nil {
		if err := c.pushSubscription.Unsubscribe(); err != nil {
			logger.Warnf(ctx, "[License Client] Failed to unsubscribe push topic: %v", err)
		}
		c.pushSubscription = nil
	}

	if c.refreshSubscription != nil {
		if err := c.refreshSubscription.Unsubscribe(); err != nil {
			logger.Warnf(ctx, "[License Client] Failed to unsubscribe refresh topic: %v", err)
		}
		c.refreshSubscription = nil
	}

	logger.Infof(ctx, "[License Client] License client stopped")
	return nil
}
