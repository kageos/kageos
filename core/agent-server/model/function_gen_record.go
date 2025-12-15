package model

import (
	"encoding/json"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// FunctionGenRecord 函数生成记录模型
type FunctionGenRecord struct {
	models.Base
	SessionID     string `gorm:"type:varchar(64);not null;index;comment:会话ID" json:"session_id"`
	MessageID    int64  `gorm:"type:bigint;index;comment:消息ID（关联到 AgentChatMessage.ID）" json:"message_id"` // 新增
	AgentID      int64  `gorm:"type:bigint;not null;index;comment:智能体ID" json:"agent_id"`
	TreeID       int64  `gorm:"type:bigint;not null;index;comment:服务目录ID" json:"tree_id"`
	Status       string `gorm:"type:varchar(32);not null;index;comment:状态(generating/completed/failed)" json:"status"`
	Code         string `gorm:"type:longtext;comment:生成的代码" json:"code"`
	ErrorMsg     string `gorm:"type:text;comment:错误信息" json:"error_msg"`
	
	// 生成的函数组代码列表（逗号分隔的字符串，便于使用 FIND_IN_SET 查询）
	// 例如："/luobei/testgroup/plugins/tools_cashier,/luobei/testgroup/plugins/tools_excel"
	FullGroupCodes string `gorm:"type:text;comment:生成的函数组代码列表（逗号分隔）" json:"full_group_codes"`
	
	// 生成过程的元数据（JSON）
	// 包含：用户消息、上传的文件、插件处理结果等
	// 例如：{"user_message": "生成一个工单系统", "files": [{"url": "...", "remark": "..."}], "plugin_data": "..."}
	Metadata *string `gorm:"type:json;comment:生成过程元数据" json:"metadata"`
	
	// 生成耗时（秒，从创建记录到完成/失败的时间）
	Duration int `gorm:"type:int;default:0;comment:生成耗时(秒)" json:"duration"`
	
	User string `gorm:"type:varchar(128);not null;index;comment:创建用户" json:"user"`
}

// GetFullGroupCodes 获取 FullGroupCodes 列表（从逗号分隔的字符串解析）
func (r *FunctionGenRecord) GetFullGroupCodes() []string {
	if r.FullGroupCodes == "" {
		return []string{}
	}
	// 使用逗号分隔，并过滤空字符串
	codes := strings.Split(r.FullGroupCodes, ",")
	result := make([]string, 0, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code != "" {
			result = append(result, code)
		}
	}
	return result
}

// SetFullGroupCodes 设置 FullGroupCodes 列表（转换为逗号分隔的字符串）
func (r *FunctionGenRecord) SetFullGroupCodes(codes []string) {
	if len(codes) == 0 {
		r.FullGroupCodes = ""
		return
	}
	// 过滤空字符串并连接
	validCodes := make([]string, 0, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code != "" {
			validCodes = append(validCodes, code)
		}
	}
	r.FullGroupCodes = strings.Join(validCodes, ",")
}

// GetMetadata 获取元数据
func (r *FunctionGenRecord) GetMetadata() (map[string]interface{}, error) {
	if r.Metadata == nil || *r.Metadata == "" {
		return make(map[string]interface{}), nil
	}
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(*r.Metadata), &metadata)
	return metadata, err
}

// SetMetadata 设置元数据
func (r *FunctionGenRecord) SetMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	metadataStr := string(data)
	r.Metadata = &metadataStr
	return nil
}

// TableName 指定表名
func (FunctionGenRecord) TableName() string {
	return "function_gen_records"
}

// 状态常量
const (
	FunctionGenStatusGenerating = "generating" // 生成中
	FunctionGenStatusCompleted  = "completed"  // 已完成
	FunctionGenStatusFailed    = "failed"      // 失败
)

