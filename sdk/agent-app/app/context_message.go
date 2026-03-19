package app

import (
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

// SendMessageOpts 发送消息参数
// ToUsers、ToDepartments 为逗号分隔字符串；ContentType 为空时默认 text
// From 可选：不填时由 SDK 用当前请求用户填充，无用户（如定时任务）时为 "system"
// FullCodePath 可选：不填时由 SDK 用当前请求路由推导（/user/app/router），定时任务等可显式传入来源目录
type SendMessageOpts struct {
	ToUsers       string `json:"to_users"`        // 接收用户，逗号分隔，如 "zhangsan,lisi"
	ToDepartments string `json:"to_departments"`  // 接收部门（full_code_path），逗号分隔
	Title         string `json:"title"`           // 标题/摘要（可选）
	Content       string `json:"content"`         // 正文
	ContentType   string `json:"content_type"`    // text | html | markdown，空默认 text
	From          string `json:"from"`            // 发送人（可选，不填则用当前请求用户或 system）
	FullCodePath  string `json:"full_code_path"`  // 来源目录/函数路径（可选，不填则用当前请求的 full_code_path）
}

// SendMessage 发送消息：接收结构体参数，渠道由消息服务内部决定
func (c *Context) SendMessage(opts *SendMessageOpts) error {
	if app == nil {
		return fmt.Errorf("app 未初始化")
	}
	if opts == nil {
		return fmt.Errorf("SendMessageOpts 不能为 nil")
	}
	from := strings.TrimSpace(opts.From)
	if from == "" {
		from = c.GetRequestUser()
		if from == "" {
			from = "system"
		}
	}
	contentType := strings.TrimSpace(opts.ContentType)
	if contentType == "" {
		contentType = "text"
	}
	fullCodePath := strings.TrimSpace(opts.FullCodePath)
	if fullCodePath == "" && c.msg != nil {
		fullCodePath = c.msg.GetFullRouter()
	}
	payload := &dto.MessageSendPayload{
		From:          from,
		FullCodePath:  fullCodePath,
		ToUsers:       strings.TrimSpace(opts.ToUsers),
		ToDepartments: strings.TrimSpace(opts.ToDepartments),
		Title:         opts.Title,
		Content:       opts.Content,
		ContentType:   contentType,
	}
	return app.PublishMessage(payload)
}
