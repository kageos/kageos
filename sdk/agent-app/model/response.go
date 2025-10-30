package model

import "time"

// UpdateResponse API更新响应结构
type UpdateResponse struct {
	Status    string    `json:"status"`    // 状态: success, error
	Message   string    `json:"message"`   // 响应消息
	Data      *DiffData `json:"data"`      // 差异数据
	Version   string    `json:"version"`   // 当前版本
	Timestamp time.Time `json:"timestamp"` // 响应时间
}

// DiffData API差异数据
type DiffData struct {
	Add    []*ApiInfo `json:"add"`    // 新增的API
	Update []*ApiInfo `json:"update"` // 修改的API
	Delete []*ApiInfo `json:"delete"` // 删除的API
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Error     string    `json:"error,omitempty"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}
