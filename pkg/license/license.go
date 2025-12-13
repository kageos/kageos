package license

import (
	"encoding/json"
	"time"
)

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
//         "operate_log": true,
//         "workflow": true,
//         "approval": false
//       },
//       "hardware_id": "optional-hardware-binding"
//     },
//     "signature": "RSA签名..."
//   }
type License struct {
	// License 基本信息
	ID        string    `json:"id"`         // License ID
	Edition   string    `json:"edition"`    // 版本：community, professional, enterprise, flagship
	IssuedAt  time.Time `json:"issued_at"`  // 签发时间
	ExpiresAt time.Time `json:"expires_at"` // 过期时间（零值表示永久）

	// 客户信息
	Customer string `json:"customer"` // 客户名称

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
type Features struct {
	// 操作日志
	OperateLog bool `json:"operate_log"` // 操作日志功能

	// 工作流
	Workflow bool `json:"workflow"` // 工作流功能

	// 审批流程
	Approval bool `json:"approval"` // 审批流程功能

	// 评论系统
	Comment bool `json:"comment"` // 评论功能

	// 权限管理
	RBAC bool `json:"rbac"` // 基于角色的访问控制

	// 定时任务
	ScheduledTask bool `json:"scheduled_task"` // 定时任务功能

	// 回收站
	RecycleBin bool `json:"recycle_bin"` // 回收站功能

	// 变更日志
	ChangeLog bool `json:"change_log"` // 变更日志功能

	// 通知中心
	Notification bool `json:"notification"` // 通知中心功能

	// 配置管理
	ConfigManagement bool `json:"config_management"` // 配置管理功能

	// 快链
	QuickLink bool `json:"quick_link"` // 快链功能
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
	if !l.ExpiresAt.IsZero() && time.Now().After(l.ExpiresAt) {
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
	case "workflow":
		return l.Features.Workflow
	case "approval":
		return l.Features.Approval
	case "comment":
		return l.Features.Comment
	case "rbac":
		return l.Features.RBAC
	case "scheduled_task":
		return l.Features.ScheduledTask
	case "recycle_bin":
		return l.Features.RecycleBin
	case "change_log":
		return l.Features.ChangeLog
	case "notification":
		return l.Features.Notification
	case "config_management":
		return l.Features.ConfigManagement
	case "quick_link":
		return l.Features.QuickLink
	default:
		return false
	}
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
func (l *License) ToJSON() ([]byte, error) {
	return json.Marshal(l)
}
