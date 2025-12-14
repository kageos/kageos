package model

import (
	"encoding/json"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// FormOperateLog Form 操作日志表
// 用于记录 Form 的提交操作（RequestApp）
// 策略：社区版和企业版都记录完整日志，但只有企业版可以查看
type FormOperateLog struct {
	models.Base

	// ✅ 合规要求：Who、When、What、Where
	// CreatedAt 已由 models.Base 提供，无需重复定义
	TenantUser  string `json:"tenant_user" gorm:"type:varchar(255);not null;index:idx_tenant_user;comment:租户用户（app的所有者）"` // 租户用户（app 的所有者）
	RequestUser string `json:"request_user" gorm:"type:varchar(255);not null;index:idx_user;comment:请求用户（实际执行操作的用户）"` // ✅ Who（谁）：实际执行操作的用户
	Action       string `json:"action" gorm:"type:varchar(50);not null;index:idx_action;comment:操作类型"`                      // ✅ What（做了什么）：request_app 或 form_submit
	IPAddress    string `json:"ip_address" gorm:"type:varchar(50);comment:IP地址"`                                               // ✅ Where（从哪里：IP）
	UserAgent    string `json:"user_agent" gorm:"type:varchar(500);comment:User Agent"`                                          // ✅ Where（从哪里：User Agent）

	// Form 特有字段
	App            string         `json:"app" gorm:"type:varchar(100);not null;comment:应用名"`                           // 应用名（如：demo）
	FullCodePath   string         `json:"full_code_path" gorm:"type:varchar(500);not null;index:idx_full_code_path;index:idx_user_full_code_path;index:idx_full_code_path_created;comment:完整代码路径（与服务树对齐）"` // ⭐ 必须：完整代码路径（与服务树对齐，如：/luobei/demo/biz/biz_vote_system_submit）
	FunctionMethod string         `json:"function_method" gorm:"type:varchar(10);comment:HTTP方法"`                      // HTTP 方法（如：GET、POST）

	// ✅ 合规要求：请求和响应（相当于变更前后值）
	RequestBody  json.RawMessage `json:"request_body" gorm:"type:json;comment:请求参数"`   // 请求参数（如：{"topic_id": 2, "option_ids": [6]}）
	ResponseBody json.RawMessage `json:"response_body" gorm:"type:json;comment:响应参数"`  // 响应参数（如：{"code": -1, "msg": "您已经投过票了"}）
	Code         int             `json:"code" gorm:"type:int;comment:响应码（从 response_body.code 提取）"` // 响应码（从 response_body.code 提取）
	Msg          string          `json:"msg" gorm:"type:varchar(500);comment:错误消息（如果失败，从 response_body.msg 提取）"` // 错误消息（如果失败，从 response_body.msg 提取）

	// 元数据
	TraceID string `json:"trace_id" gorm:"type:varchar(100);comment:追踪ID（用于关联请求）"` // 追踪ID（用于关联请求）
	Version string `json:"version" gorm:"type:varchar(50);comment:应用版本"`              // 应用版本（如：v156）
}

// TableName 指定表名
func (FormOperateLog) TableName() string {
	return "form_operate_logs"
}

