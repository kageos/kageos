package model

import (
	"encoding/json"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"strings"
)

type Function struct {
	models.Base
	Request      json.RawMessage `json:"request" gorm:"type:json"`
	Response     json.RawMessage `json:"response" gorm:"type:json"`
	AppID        int64           `json:"app_id"`
	TreeID       int64           `json:"tree_id"`
	Method       string          `json:"method" gorm:"type:varchar(255);column:method"`
	Router       string          `json:"router" gorm:"type:varchar(255);column:router"`
	HasConfig    bool            `json:"has_config" gorm:"column:has_config;comment:是否存在配置"` // 是否存在配置
	CreateTables string          `json:"create_tables"`                                      //创建该api时候会自动帮忙创建这个数据库表gorm的model列表
	Callbacks    string          `json:"callbacks"`
	TemplateType string          `json:"widget"`                                  // 渲染类型
	SourceCodeID *int64          `json:"source_code_id" gorm:"column:source_code_id;index;comment:指向SourceCode记录的ID"` // 指向 SourceCode 记录（可空，创建时先保存 SourceCode 再设置此字段）
	App          *App            `json:"-" gorm:"foreignKey:AppID;references:ID"` // 预加载的完整应用对象
	SourceCode   *SourceCode     `json:"-" gorm:"foreignKey:SourceCodeID;references:ID"` // 预加载的源代码对象
}

func (Function) TableName() string {
	return "function"
}

// GetMethod 获取HTTP方法
func (f *Function) GetMethod() string {
	return f.Method
}

// GetRouter 获取路由路径
func (f *Function) GetRouter() string {
	return f.Router
}

// GetEndpoint 获取API端点（方法+路由）
func (f *Function) GetEndpoint() string {
	return f.Method + " " + f.Router
}

// HasRequestConfig 是否有请求配置
func (f *Function) HasRequestConfig() bool {
	return len(f.Request) > 0 && f.Request != nil
}

// HasResponseConfig 是否有响应配置
func (f *Function) HasResponseConfig() bool {
	return len(f.Response) > 0 && f.Response != nil
}

// HasCreateTables 是否有创建表配置
func (f *Function) HasCreateTables() bool {
	return f.CreateTables != ""
}

// HasCallbacks 是否有回调配置
func (f *Function) HasCallbacks() bool {
	return f.Callbacks != ""
}

// GetTemplateType 获取模板类型
func (f *Function) GetTemplateType() string {
	if f.TemplateType == "" {
		return "default"
	}
	return f.TemplateType
}

// IsGET 判断是否为GET请求
func (f *Function) IsGET() bool {
	return f.Method == "GET"
}

// IsPOST 判断是否为POST请求
func (f *Function) IsPOST() bool {
	return f.Method == "POST"
}

// IsPUT 判断是否为PUT请求
func (f *Function) IsPUT() bool {
	return f.Method == "PUT"
}

// IsDELETE 判断是否为DELETE请求
func (f *Function) IsDELETE() bool {
	return f.Method == "DELETE"
}

// GetRouterSegments 获取路由段
func (f *Function) GetRouterSegments() []string {
	router := strings.Trim(f.Router, "/")
	if router == "" {
		return []string{}
	}
	return strings.Split(router, "/")
}

// GetLastRouterSegment 获取路由的最后一个段
func (f *Function) GetLastRouterSegment() string {
	segments := f.GetRouterSegments()
	if len(segments) == 0 {
		return ""
	}
	return segments[len(segments)-1]
}
