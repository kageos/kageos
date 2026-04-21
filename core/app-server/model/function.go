package model

import (
	"encoding/json"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/functionschema"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

type Function struct {
	models.Base
	Schema       json.RawMessage `json:"schema" gorm:"type:json"`
	AppID        int64           `json:"app_id"`
	TreeID       int64           `json:"tree_id"`
	Method       string          `json:"method" gorm:"type:varchar(255);column:method"`
	Router       string          `json:"router" gorm:"type:varchar(255);column:router"`
	HasConfig    bool            `json:"has_config" gorm:"column:has_config;comment:是否存在配置"` // 是否存在配置
	CreateTables string          `json:"create_tables"`                                      //创建该api时候会自动帮忙创建这个数据库表gorm的model列表
	TemplateType string          `json:"widget"`                                             // 渲染类型
	App          *App            `json:"-" gorm:"foreignKey:AppID;references:ID"`            // 预加载
	// 不在此处关联 ServiceTree，避免 AutoMigrate 为 tree_id 建外键导致历史脏数据迁移失败；搜索函数改为查 ServiceTree 并 Preload Function
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

// HasCreateTables 是否有创建表配置
func (f *Function) HasCreateTables() bool {
	return f.CreateTables != ""
}

// HasCallbacks 是否有回调配置
func (f *Function) HasCallbacks() bool {
	return len(f.GetCallbacks()) > 0
}

// GetCallbacks 获取回调列表
func (f *Function) GetCallbacks() []string {
	if f == nil {
		return nil
	}
	return functionschema.Callbacks(f.Schema)
}

// HasCallback 判断是否声明了某个回调能力
func (f *Function) HasCallback(target string) bool {
	for _, callback := range f.GetCallbacks() {
		if callback == target {
			return true
		}
	}
	return false
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
