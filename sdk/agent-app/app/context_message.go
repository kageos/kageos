package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
)

// SendMessageOpts contains the business-facing message fields.
// ToUsers uses the same comma-separated storage format as the users widget.
type SendMessageOpts struct {
	ToUsers     string `json:"to_users"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

// SendMessage asynchronously publishes a message command. message-service owns
// inbox storage and external channel delivery, so business code does not need
// channel logic or wait for delivery completion.
func (c *Context) SendMessage(opts *SendMessageOpts) error {
	if app == nil {
		return fmt.Errorf("app 未初始化")
	}
	if opts == nil {
		return fmt.Errorf("SendMessageOpts 不能为 nil")
	}
	contentType := strings.ToLower(strings.TrimSpace(opts.ContentType))
	if contentType == "" {
		contentType = "markdown"
	}
	envelope := &dto.MessageSendEnvelope{
		Meta: c.messageSendMeta(),
		Message: dto.MessageSendPayload{
			ToUsers:     strings.TrimSpace(opts.ToUsers),
			Title:       strings.TrimSpace(opts.Title),
			Content:     strings.TrimSpace(opts.Content),
			ContentType: contentType,
		},
	}
	return app.PublishMessage(c.Context, envelope)
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
	sourceType := strings.TrimSpace(contextx.GetSourceType(sourceCtx))
	sourceRef := strings.TrimSpace(contextx.GetSourceRef(sourceCtx))
	if sourceType == "" && sourceRef == "" {
		if fullCodePath := strings.TrimSpace(c.GetFullCodePath()); fullCodePath != "" {
			sourceType = "sdk_function"
			sourceRef = fullCodePath
		}
	}
	return dto.MessageSendMeta{
		From:               from,
		RequestUser:        requestUser,
		DepartmentFullPath: strings.TrimSpace(c.GetRequestUserDept()),
		FullCodePath:       strings.TrimSpace(c.GetFullCodePath()),
		TraceID:            strings.TrimSpace(c.GetTraceId()),
		ClientSource:       strings.TrimSpace(c.GetClientSource()),
		SourceType:         sourceType,
		SourceRef:          sourceRef,
	}
}
