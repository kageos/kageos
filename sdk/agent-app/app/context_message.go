package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
)

// SendMessageOpts 发送消息参数
//
// ContentType 支持以下值：
//   - "markdown"（默认）：正文用 Markdown 书写，消费端按渠道自动转换（邮件→HTML，企微/钉钉原生支持，短信→纯文本）
//   - "html"：正文为原始 HTML，适合需要精确控制排版的场景（如模板邮件），注意自行防 XSS
//   - "text"：纯文本，不做任何格式解析
//
// ToUsers / ToDepartments 为逗号分隔字符串，与 user / departments 组件的存储格式一致
type SendMessageOpts struct {
	ToUsers       string `json:"to_users"`       // 接收用户，逗号分隔，如 "zhangsan,lisi"
	ToDepartments string `json:"to_departments"` // 接收部门（full_code_path），逗号分隔
	Title         string `json:"title"`          // 标题/摘要（可选）
	Content       string `json:"content"`        // 正文（默认 markdown 格式）
	ContentType   string `json:"content_type"`   // "markdown"(默认) | "html" | "text"
}

// SendMessage 发送消息：接收结构体参数，渠道由消息服务内部决定
func (c *Context) SendMessage(opts *SendMessageOpts) error {
	if app == nil {
		return fmt.Errorf("app 未初始化")
	}
	if opts == nil {
		return fmt.Errorf("SendMessageOpts 不能为 nil")
	}
	contentType := strings.TrimSpace(opts.ContentType)
	if contentType == "" {
		contentType = "markdown"
	}
	envelope := &dto.MessageSendEnvelope{
		Meta: c.messageSendMeta(),
		Message: dto.MessageSendPayload{
			ToUsers:       strings.TrimSpace(opts.ToUsers),
			ToDepartments: strings.TrimSpace(opts.ToDepartments),
			Title:         opts.Title,
			Content:       opts.Content,
			ContentType:   contentType,
		},
	}
	return app.PublishMessage(envelope)
}

func (c *Context) messageSendMeta() dto.MessageSendMeta {
	if c == nil {
		return dto.MessageSendMeta{From: "system"}
	}
	requestUser := strings.TrimSpace(c.GetRequestUser())
	from := requestUser
	if from == "" {
		from = "system"
	}

	sourceCtx := c.Context
	if sourceCtx == nil {
		sourceCtx = context.Background()
	}
	return dto.MessageSendMeta{
		From:               from,
		RequestUser:        requestUser,
		DepartmentFullPath: strings.TrimSpace(c.GetRequestUserDept()),
		FullCodePath:       strings.TrimSpace(c.GetFullCodePath()),
		TraceID:            strings.TrimSpace(c.GetTraceId()),
		ClientSource:       strings.TrimSpace(c.GetClientSource()),
		SourceType:         strings.TrimSpace(contextx.GetSourceType(sourceCtx)),
		SourceRef:          strings.TrimSpace(contextx.GetSourceRef(sourceCtx)),
	}
}
