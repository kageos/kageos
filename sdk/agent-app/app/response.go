package app

import (
	"time"
)

// UpdateResponse API更新响应结构
type UpdateResponse struct {
	Status    string    `json:"status"`    // 状态: success, error
	Message   string    `json:"message"`   // 响应消息
	Data      *DiffData `json:"data"`      // 差异数据
	Version   string    `json:"version"`   // 当前版本
	Timestamp time.Time `json:"timestamp"` // 响应时间
}

// PackageInfo SDK 返回的 package 元信息，app-server 用于目录对账
type PackageInfo struct {
	Code        string `json:"code"`         // 目录名（如 "pdf"）
	Name        string `json:"name"`         // 显示名称
	Desc        string `json:"desc"`         // 描述
	RouterGroup string `json:"router_group"` // 路由组路径（如 "/plugins/pdf"）
	FullPath    string `json:"full_path"`    // 完整路径（如 "/user/app/plugins/pdf"）
}

// DiffData API差异数据
type DiffData struct {
	Add      []*ApiInfo     `json:"add"`      // 新增的API
	Update   []*ApiInfo     `json:"update"`   // 修改的API
	Delete   []*ApiInfo     `json:"delete"`   // 删除的API
	Packages []*PackageInfo `json:"packages"` // 全量 package 列表，每次 update 都返回，用于 app-server 目录对账
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Error     string    `json:"error,omitempty"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}
