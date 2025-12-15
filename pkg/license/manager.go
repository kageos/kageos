package license

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

var (
	// 全局 License 管理器实例
	globalManager *Manager
	managerOnce   sync.Once
	managerMu     sync.RWMutex
)

// Manager License 管理器
// 负责加载、验证和管理 License
type Manager struct {
	license     *License       // 当前 License（nil 表示社区版）
	licensePath string         // License 文件路径
	publicKey   *rsa.PublicKey // RSA 公钥（用于验证签名）
	mu          sync.RWMutex
}

// GetManager 获取全局 License 管理器实例
func GetManager() *Manager {
	managerOnce.Do(func() {
		globalManager = &Manager{
			licensePath: getDefaultLicensePath(),
		}
	})
	return globalManager
}

// LoadLicense 加载 License 文件
// 参数：
//   - path: License 文件路径（如果为空，使用默认路径）
//
// 返回：
//   - error: 如果加载失败返回错误
//
// 说明：
//   - 如果文件不存在，返回 nil（社区版）
//   - 如果文件存在但验证失败，返回错误
//   - 如果验证成功，设置当前 License
func (m *Manager) LoadLicense(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果指定了路径，使用指定路径
	if path != "" {
		m.licensePath = path
	}

	// 检查文件是否存在
	if _, err := os.Stat(m.licensePath); os.IsNotExist(err) {
		// 文件不存在，社区版
		m.license = nil
		logger.Infof(nil, "[License] License file not found, using community edition")
		return nil
	}

	// 读取 License 文件
	data, err := os.ReadFile(m.licensePath)
	if err != nil {
		return fmt.Errorf("failed to read license file: %w", err)
	}

	// 解析 JSON
	var licenseFile LicenseFile
	if err := json.Unmarshal(data, &licenseFile); err != nil {
		return fmt.Errorf("failed to parse license file: %w", err)
	}

	// 验证签名
	if err := m.verifySignature(&licenseFile); err != nil {
		return fmt.Errorf("license signature verification failed: %w", err)
	}

	// 验证 License 有效性
	if !licenseFile.License.IsValid() {
		return fmt.Errorf("license has expired")
	}

	// 验证硬件绑定（如果启用）
	if licenseFile.License.HardwareID != "" {
		currentHardwareID := getHardwareID()
		if currentHardwareID != licenseFile.License.HardwareID {
			return fmt.Errorf("license hardware binding mismatch")
		}
	}

	// 设置 License
	m.license = &licenseFile.License

	logger.Infof(nil, "[License] License loaded successfully: Edition=%s, Customer=%s, ExpiresAt=%v",
		m.license.Edition, m.license.Customer, m.license.ExpiresAt.Time)

	return nil
}

// GetLicense 获取当前 License
// 返回：
//   - *License: 当前 License（nil 表示社区版）
func (m *Manager) GetLicense() *License {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.license
}

// HasFeature 检查是否有某个功能
// 参数：
//   - featureName: 功能名称（如 "operate_log", "workflow" 等）
//
// 返回：
//   - bool: 是否有该功能
func (m *Manager) HasFeature(featureName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.license == nil {
		return false // 社区版，没有企业功能
	}

	return m.license.HasFeature(featureName)
}

// HasOperateLogFeature 检查是否有操作日志功能
// 返回：
//   - bool: 是否有操作日志功能
//
// 说明：
//   - 向后兼容方法，推荐使用 HasFeature(enterprise.FeatureOperateLog)
func (m *Manager) HasOperateLogFeature() bool {
	return m.HasFeature("operate_log")
}

// GetEdition 获取当前版本
// 返回：
//   - Edition: 当前版本（社区版、专业版、企业版、旗舰版）
func (m *Manager) GetEdition() Edition {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.license == nil {
		return EditionCommunity
	}

	return m.license.GetEdition()
}

// IsEnterprise 检查是否是企业版
// 返回：
//   - bool: 是否是企业版
func (m *Manager) IsEnterprise() bool {
	edition := m.GetEdition()
	return edition == EditionEnterprise || edition == EditionFlagship
}

// IsActivated 检查是否已激活（激活 + 企业版 + 未过期）
// 返回：
//   - bool: 是否已激活且企业版且未过期
func (m *Manager) IsActivated() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 检查 License 是否存在
	if m.license == nil {
		return false
	}

	// 检查是否有效（未过期）
	if !m.license.IsValid() {
		return false
	}

	// 检查是否是企业版（enterprise 或 flagship）
	edition := m.license.GetEdition()
	return edition == EditionEnterprise || edition == EditionFlagship
}

// GetMaxApps 获取最大应用数量
// 返回：
//   - int: 最大应用数量（-1 表示无限制）
func (m *Manager) GetMaxApps() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.license == nil {
		return CommunityMaxApps // 社区版默认限制
	}

	return m.license.GetMaxApps()
}

// GetMaxUsers 获取最大用户数量
// 返回：
//   - int: 最大用户数量（-1 表示无限制）
func (m *Manager) GetMaxUsers() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.license == nil {
		return CommunityMaxUsers // 社区版默认限制
	}

	return m.license.GetMaxUsers()
}

// CheckAppLimit 检查应用数量限制
// 参数：
//   - currentCount: 当前应用数量
//
// 返回：
//   - error: 如果超过限制返回错误
func (m *Manager) CheckAppLimit(currentCount int) error {
	maxApps := m.GetMaxApps()
	if maxApps == -1 {
		return nil // 无限制
	}

	if currentCount >= maxApps {
		edition := m.GetEdition()
		if edition == EditionCommunity {
			return fmt.Errorf("社区版最多支持 %d 个应用，当前已有 %d 个。请升级到企业版以支持更多应用", maxApps, currentCount)
		}
		return fmt.Errorf("已达到最大应用数量限制：%d 个", maxApps)
	}

	return nil
}

// CheckUserLimit 检查用户数量限制
// 参数：
//   - currentCount: 当前用户数量
//
// 返回：
//   - error: 如果超过限制返回错误
func (m *Manager) CheckUserLimit(currentCount int) error {
	maxUsers := m.GetMaxUsers()
	if maxUsers == -1 {
		return nil // 无限制
	}

	if currentCount >= maxUsers {
		edition := m.GetEdition()
		if edition == EditionCommunity {
			return fmt.Errorf("社区版最多支持 %d 个用户，当前已有 %d 个。请升级到企业版以支持更多用户", maxUsers, currentCount)
		}
		return fmt.Errorf("已达到最大用户数量限制：%d 个", maxUsers)
	}

	return nil
}

// verifySignature 验证 License 签名
func (m *Manager) verifySignature(licenseFile *LicenseFile) error {
	// 加载公钥（如果还没有加载）
	if m.publicKey == nil {
		if err := m.loadPublicKey(); err != nil {
			return fmt.Errorf("failed to load public key: %w", err)
		}
	}

	logger.Infof(nil, "[License] 开始验证签名...")
	logger.Infof(nil, "[License] License ID: %s", licenseFile.License.ID)
	logger.Infof(nil, "[License] License Edition: %s", licenseFile.License.Edition)

	// 将 License 数据转换为 JSON
	licenseJSON, err := licenseFile.License.ToJSON()
	if err != nil {
		logger.Errorf(nil, "[License] 序列化 License 失败: %v", err)
		return fmt.Errorf("failed to marshal license: %w", err)
	}
	logger.Infof(nil, "[License] License JSON 长度: %d 字节", len(licenseJSON))
	logger.Debugf(nil, "[License] License JSON 内容: %s", string(licenseJSON))

	// 检查签名字段是否为空
	if licenseFile.Signature == "" {
		logger.Errorf(nil, "[License] ❌ 签名字段为空")
		return fmt.Errorf("signature field is empty")
	}
	logger.Infof(nil, "[License] 原始签名字符串长度: %d 字符", len(licenseFile.Signature))
	// 显示签名字符串的前100个字符（用于调试）
	previewLen := 100
	if len(licenseFile.Signature) < previewLen {
		previewLen = len(licenseFile.Signature)
	}
	logger.Debugf(nil, "[License] 原始签名字符串（前%d字符）: %s", previewLen, licenseFile.Signature[:previewLen])

	// 解码签名
	signature, err := base64.StdEncoding.DecodeString(licenseFile.Signature)
	if err != nil {
		logger.Errorf(nil, "[License] ❌ Base64 解码签名失败: %v", err)
		logger.Errorf(nil, "[License] 原始签名字符串: %s", licenseFile.Signature)
		return fmt.Errorf("failed to decode signature: %w", err)
	}
	logger.Infof(nil, "[License] 签名长度: %d 字节", len(signature))
	if len(signature) == 0 {
		logger.Errorf(nil, "[License] ❌ 签名解码后长度为0，可能是Base64格式错误或签名字段为空")
		logger.Errorf(nil, "[License] 原始签名字符串: %s", licenseFile.Signature)
		return fmt.Errorf("signature is empty after decoding")
	}

	// 计算哈希
	hash := sha256.Sum256(licenseJSON)
	logger.Infof(nil, "[License] SHA256 哈希: %x", hash)

	// 验证签名
	logger.Infof(nil, "[License] 使用公钥验证签名...")
	if err := rsa.VerifyPKCS1v15(m.publicKey, crypto.SHA256, hash[:], signature); err != nil {
		logger.Errorf(nil, "[License] 签名验证失败: %v", err)
		logger.Errorf(nil, "[License] 可能的原因：")
		logger.Errorf(nil, "[License]   1. License 文件是用不同的私钥签名的")
		logger.Errorf(nil, "[License]   2. 代码中的公钥与签名私钥不匹配")
		logger.Errorf(nil, "[License]   3. License 文件被修改过")
		logger.Errorf(nil, "[License]   4. JSON 序列化格式不一致（签名时和验证时）")
		return fmt.Errorf("signature verification failed: %w", err)
	}

	logger.Infof(nil, "[License] ✅ 签名验证成功")
	return nil
}

// loadPublicKey 加载 RSA 公钥
// 优先级：
// 1. 从编译到程序中的公钥加载（如果存在）
// 2. 从文件加载（向后兼容）
func (m *Manager) loadPublicKey() error {
	logger.Infof(nil, "[License] 开始加载公钥...")

	// 1. 优先尝试从编译到程序中的公钥加载
	if embeddedPublicKey != "" {
		logger.Infof(nil, "[License] 尝试从嵌入的公钥加载...")
		err := m.loadPublicKeyFromString(embeddedPublicKey)
		if err == nil {
			logger.Infof(nil, "[License] ✅ 从嵌入数据加载公钥成功")
			logger.Debugf(nil, "[License] 嵌入公钥长度: %d 字符", len(embeddedPublicKey))
			return nil
		}
		logger.Warnf(nil, "[License] ⚠️  从嵌入数据加载公钥失败: %v, 尝试从文件加载", err)
	} else {
		logger.Infof(nil, "[License] 嵌入公钥为空，尝试从文件加载...")
	}

	// 2. 从文件加载（向后兼容）
	publicKeyPath := getPublicKeyPath()
	logger.Infof(nil, "[License] 公钥文件路径: %s", publicKeyPath)
	data, err := os.ReadFile(publicKeyPath)
	if err != nil {
		logger.Errorf(nil, "[License] ❌ 读取公钥文件失败: %v", err)
		return fmt.Errorf("failed to read public key file: %w", err)
	}

	logger.Infof(nil, "[License] ✅ 从文件加载公钥成功，文件大小: %d 字节", len(data))
	return m.loadPublicKeyFromString(string(data))
}

// loadPublicKeyFromString 从字符串加载公钥
func (m *Manager) loadPublicKeyFromString(publicKeyStr string) error {
	logger.Debugf(nil, "[License] 解析公钥字符串，长度: %d 字符", len(publicKeyStr))

	// 解析 PEM
	block, _ := pem.Decode([]byte(publicKeyStr))
	if block == nil {
		logger.Errorf(nil, "[License] ❌ PEM 解码失败：无法找到 PEM 块")
		return fmt.Errorf("failed to decode PEM block")
	}
	logger.Debugf(nil, "[License] PEM 块类型: %s", block.Type)

	// 解析公钥
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		logger.Errorf(nil, "[License] ❌ 解析公钥失败: %v", err)
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	// 转换为 RSA 公钥
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		logger.Errorf(nil, "[License] ❌ 不是 RSA 公钥")
		return fmt.Errorf("not an RSA public key")
	}

	m.publicKey = rsaPub
	logger.Infof(nil, "[License] ✅ 公钥解析成功，密钥大小: %d 位", rsaPub.N.BitLen())
	return nil
}

// embeddedPublicKey 编译到程序中的公钥（通过 build tag 或 embed 设置）
// 如果为空，则从文件加载
var embeddedPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAnqJ0ZmMBWOC2CM+diH8l
Pr0Qqsd9n+DiLV4b0VeM6/NyjHb5eriuFuBpDutQk8sAO7o+uNFJgAfjhvpIr2oh
I7eh/Wqwcxo4zqFQXdrDCeZst3YqoU2UITr4a6LzyqK/U7lE4nICEOE8Qmb118Yi
h1Pp2lk7No/xLcBGGzJiKDk7CEd2cEZqnNWtKWCrYB1ubz9HPc3R39TkygN+9rAO
qTU4uDFLJfblEIor68itFVNTIu6vDyt+2i492zwr/20jFj6HP/R8yLxeTx+bwaal
MrfPWWdOdtkYbuV1m52XnCegZ3dQW9IGwXArWhhs15wx+jvGcF/NvOJmcx6174+h
OwIDAQAB
-----END PUBLIC KEY-----`

// getDefaultLicensePath 获取默认 License 文件路径
func getDefaultLicensePath() string {
	// 优先级：
	// 1. 环境变量 LICENSE_PATH
	// 2. ./license.json
	// 3. ~/.ai-agent-os/license.json

	if path := os.Getenv("LICENSE_PATH"); path != "" {
		return path
	}

	// 检查当前目录
	if _, err := os.Stat("./license.json"); err == nil {
		return "./license.json"
	}

	// 使用用户目录
	homeDir, err := os.UserHomeDir()
	if err == nil {
		licensePath := filepath.Join(homeDir, ".ai-agent-os", "license.json")
		return licensePath
	}

	// 默认返回当前目录
	return "./license.json"
}

// getPublicKeyPath 获取公钥文件路径
func getPublicKeyPath() string {
	// 优先级：
	// 1. 环境变量 LICENSE_PUBLIC_KEY_PATH
	// 2. ./license_public_key.pem
	// 3. ~/.ai-agent-os/license_public_key.pem

	if path := os.Getenv("LICENSE_PUBLIC_KEY_PATH"); path != "" {
		return path
	}

	// 检查当前目录
	if _, err := os.Stat("./license_public_key.pem"); err == nil {
		return "./license_public_key.pem"
	}

	// 使用用户目录
	homeDir, err := os.UserHomeDir()
	if err == nil {
		publicKeyPath := filepath.Join(homeDir, ".ai-agent-os", "license_public_key.pem")
		return publicKeyPath
	}

	// 默认返回当前目录
	return "./license_public_key.pem"
}

// getHardwareID 获取硬件ID（用于硬件绑定）
func getHardwareID() string {
	// TODO: 实现硬件ID获取逻辑
	// 注意：集群部署时，硬件绑定策略需要重新设计
	return "hardware-id-placeholder"
}

// setLicense 设置 License（内部方法，供 Client 使用）
func (m *Manager) setLicense(lic *License) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.license = lic
}

// ClearLicense 清除 License（回到社区版）
// 主要用于注销功能，清除当前 License 状态
func (m *Manager) ClearLicense() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.license = nil
}

// SetLicensePath 设置 License 文件路径（用于测试）
func (m *Manager) SetLicensePath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.licensePath = path
}

// SetPublicKey 设置公钥（用于测试）
func (m *Manager) SetPublicKey(key *rsa.PublicKey) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publicKey = key
}
