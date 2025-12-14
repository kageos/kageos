package model

import (
	"encoding/json"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// TableOperateLog Table 操作日志表
// 用于记录 Table 的操作（OnTableAddRow、OnTableUpdateRow、OnTableDeleteRows）
// 策略：社区版和企业版都记录完整日志，但只有企业版可以查看
type TableOperateLog struct {
	models.Base

	// ✅ 合规要求：Who、When、What、Where
	// CreatedAt 已由 models.Base 提供，无需重复定义
	TenantUser  string `json:"tenant_user" gorm:"type:varchar(255);not null;index:idx_tenant_user;comment:租户用户（app的所有者）"` // 租户用户（app 的所有者）
	RequestUser string `json:"request_user" gorm:"type:varchar(255);not null;index:idx_user;comment:请求用户（实际执行操作的用户）"`     // ✅ Who（谁）：实际执行操作的用户
	Action      string `json:"action" gorm:"type:varchar(50);not null;index:idx_action;comment:操作类型"`                     // ✅ What（做了什么）：OnTableAddRow、OnTableUpdateRow、OnTableDeleteRows
	IPAddress   string `json:"ip_address" gorm:"type:varchar(50);comment:IP地址"`                                           // ✅ Where（从哪里：IP）
	UserAgent   string `json:"user_agent" gorm:"type:varchar(500);comment:User Agent"`                                    // ✅ Where（从哪里：User Agent）

	// Table 特有字段
	App          string `json:"app" gorm:"type:varchar(100);not null;comment:应用名"`                                                                                                                                                // 应用名（如：demo）
	FullCodePath string `json:"full_code_path" gorm:"type:varchar(500);not null;index:idx_full_code_path;index:idx_full_code_path_row;index:idx_user_full_code_path_row;index:idx_full_code_path_created;comment:完整代码路径（与服务树对齐）"` // ⭐ 必须：完整代码路径（与服务树对齐，如：/luobei/demo/crm/crm_ticket）
	RowID        int64  `json:"row_id" gorm:"type:bigint;index:idx_row_id;index:idx_full_code_path_row;index:idx_user_full_code_path_row;comment:记录ID（OnTableAddRow 时为 0，OnTableUpdateRow 和 OnTableDeleteRows 时需要）"`              // ⭐ 记录ID（OnTableAddRow 时为 0，OnTableUpdateRow 和 OnTableDeleteRows 时需要）

	// ✅ 合规要求：变更前后值
	Updates   json.RawMessage `json:"updates" gorm:"type:json;comment:更新的字段和值"`  // 更新的字段和值（如：{"description": "新值"}）
	OldValues json.RawMessage `json:"old_values" gorm:"type:json;comment:更新前的值"` // 更新前的值（如：{"description": "旧值"}）

	// 元数据
	TraceID string `json:"trace_id" gorm:"type:varchar(100);comment:追踪ID（用于关联请求）"` // 追踪ID（用于关联请求）
	Version string `json:"version" gorm:"type:varchar(50);comment:应用版本"`           // 应用版本（如：v156）
}

// TableName 指定表名
func (TableOperateLog) TableName() string {
	return "table_operate_logs"
}
