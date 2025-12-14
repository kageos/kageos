package license

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// FlexibleTime 时间类型，统一使用 DateTime 格式（2006-01-02 15:04:05）
// 说明：
//   - 配置文件使用 DateTime 格式，保持一致性
//   - 不需要支持 RFC3339 等格式，简化代码
//   - DateTime 格式更易读，符合中文习惯
type FlexibleTime struct {
	time.Time
}

// UnmarshalJSON 自定义 JSON 解析，使用 DateTime 格式（与配置文件保持一致）
func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	// 去除引号
	str := strings.Trim(string(data), `"`)
	if str == "" || str == "null" {
		ft.Time = time.Time{}
		return nil
	}

	// 只支持 DateTime 格式：2006-01-02 15:04:05（与配置文件格式一致）
	var err error
	if ft.Time, err = time.Parse(time.DateTime, str); err != nil {
		return fmt.Errorf("failed to parse time %q as DateTime format (2006-01-02 15:04:05): %w", str, err)
	}

	return nil
}

// MarshalJSON 自定义 JSON 序列化，使用 DateTime 格式（与配置文件保持一致）
func (ft FlexibleTime) MarshalJSON() ([]byte, error) {
	if ft.Time.IsZero() {
		return []byte("null"), nil
	}
	// 使用 DateTime 格式：2006-01-02 15:04:05（与配置文件格式一致）
	return json.Marshal(ft.Time.Format(time.DateTime))
}

// License License 文件结构
// 设计说明：
//   - 社区版：不需要 License，系统自动使用社区功能
//   - 企业版：需要有效的 License 文件才能使用企业功能
//   - License 文件是 JSON 格式，包含 RSA 签名防止篡改
//
// License 文件格式：
//   {
//     "license": {
//       "id": "license-xxx",
//       "edition": "enterprise",
//       "issued_at": "2025-01-01T00:00:00Z",
//       "expires_at": "2026-01-01T00:00:00Z",
//       "customer": "Company Name",
//       "max_apps": 100,
//       "max_users": 50,
//       "features": {
//         "operate_log": true
//       },
//       "hardware_id": "optional-hardware-binding"
//     },
//     "signature": "RSA签名..."
//   }
type License struct {
	// License 基本信息
	ID        string       `json:"id"`         // License ID
	Edition   string       `json:"edition"`     // 版本：community, professional, enterprise, flagship
	IssuedAt  FlexibleTime `json:"issued_at"`  // 签发时间
	ExpiresAt FlexibleTime `json:"expires_at"` // 过期时间（零值表示永久）

	// 客户信息
	Customer string `json:"customer"` // 客户名称

	// 描述信息
	Description string `json:"description,omitempty"` // License 描述（可选，用于展示）

	// 资源限制
	MaxApps int `json:"max_apps"` // 最大应用数量（0 表示无限制）
	MaxUsers int `json:"max_users"` // 最大用户数量（0 表示无限制）

	// 功能开关
	Features Features `json:"features"` // 功能开关

	// 硬件绑定（可选）
	HardwareID string `json:"hardware_id,omitempty"` // 硬件ID（用于硬件绑定，可选）

	// 在线验证（可选）
	VerifyURL string `json:"verify_url,omitempty"` // 在线验证URL（可选）
}

// Features 功能开关
// 定义所有企业功能的开关
// 注意：目前只保留 operate_log，后续新增功能时再加
type Features struct {
	// 操作日志
	OperateLog bool `json:"operate_log"` // 操作日志功能
}

// LicenseFile License 文件结构（包含签名）
type LicenseFile struct {
	License   License `json:"license"`   // License 数据
	Signature string  `json:"signature"` // RSA 签名（Base64 编码）
}

// Edition 版本类型
type Edition string

const (
	EditionCommunity   Edition = "community"   // 社区版（开源，不需要 License）
	EditionProfessional Edition = "professional" // 专业版
	EditionEnterprise  Edition = "enterprise"  // 企业版
	EditionFlagship    Edition = "flagship"    // 旗舰版
)

// IsValid 检查 License 是否有效
func (l *License) IsValid() bool {
	// 检查是否过期
	if !l.ExpiresAt.IsZero() && time.Now().After(l.ExpiresAt.Time) {
		return false
	}
	return true
}

// HasFeature 检查是否有某个功能
func (l *License) HasFeature(featureName string) bool {
	if l == nil {
		return false // 社区版，没有 License
	}

	switch featureName {
	case "operate_log":
		return l.Features.OperateLog
	default:
		return false
	}
}

// HasOperateLogFeature 检查是否有操作日志功能
// 返回：
//   - bool: 是否有操作日志功能
func (l *License) HasOperateLogFeature() bool {
	return l.HasFeature("operate_log")
}

// GetEdition 获取版本类型
func (l *License) GetEdition() Edition {
	if l == nil {
		return EditionCommunity // 社区版
	}
	return Edition(l.Edition)
}

// GetMaxApps 获取最大应用数量
// 社区版返回默认限制，企业版返回 License 中的限制
func (l *License) GetMaxApps() int {
	if l == nil {
		return CommunityMaxApps // 社区版默认限制
	}
	if l.MaxApps == 0 {
		return -1 // 无限制
	}
	return l.MaxApps
}

// GetMaxUsers 获取最大用户数量
// 社区版返回默认限制，企业版返回 License 中的限制
func (l *License) GetMaxUsers() int {
	if l == nil {
		return CommunityMaxUsers // 社区版默认限制
	}
	if l.MaxUsers == 0 {
		return -1 // 无限制
	}
	return l.MaxUsers
}

// 社区版默认限制
const (
	CommunityMaxApps  = 10 // 社区版最大应用数量：10个
	CommunityMaxUsers = 5  // 社区版最大用户数量：5个
)

// ToJSON 将 License 转换为 JSON（用于签名）
// 注意：必须与签名工具中的序列化方式完全一致
func (l *License) ToJSON() ([]byte, error) {
	// 使用紧凑格式（无缩进），确保与签名时一致
	return json.Marshal(l)
}
