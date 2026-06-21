//<文件名>send_to_user.go</文件名>

package message

import (
	"fmt"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// SendToUserReq 发送给指定用户请求结构体
type SendToUserReq struct {
	// 框架标签：widget:"type:user" - 用户选择组件
	// 消息接收人
	ToUser string `json:"to_user" widget:"name:接收用户;type:user" validate:"required"`

	// 框架标签：widget:"type:select" - 下拉选择组件
	// 内容类型：markdown（默认）、html、text
	ContentType string `json:"content_type" widget:"name:内容类型;type:select;options:markdown,html,text;options_colors:409EFF,E6A23C,909399;render_default:markdown"`

	// 框架标签：widget:"type:input" - 文本输入组件
	// 消息标题
	Title string `json:"title" widget:"name:消息标题;type:input" validate:"required,min=1,max=200"`

	// 框架标签：widget:"type:text_area" - 多行文本组件
	// 消息内容，根据内容类型不同，支持不同格式
	Content string `json:"content" widget:"name:消息内容;type:text_area;rows:8" validate:"required,min=1"`
}

// SendToUserResp 发送给指定用户响应结构体
type SendToUserResp struct {
	// 发送结果
	Result string `json:"result" widget:"name:发送结果;type:text_area"`
}

// SendToUser 发送给指定用户入口
func SendToUser(ctx *app.Context, resp response.Response) error {
	var req SendToUserReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	// 默认 contentType 为 markdown
	contentType := req.ContentType
	if contentType == "" {
		contentType = "markdown"
	}

	_, err := sendMessage(ctx, sendMessagePayload{
		ToUsers:     req.ToUser,
		Title:       req.Title,
		Content:     req.Content,
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("[系统错误]-[SendToUser] 发送消息失败, req: %+v, err: %w", req, err)
	}

	return resp.Form(&SendToUserResp{
		Result: fmt.Sprintf("消息发送成功！\n\n接收用户: %s\n内容类型: %s\n标题: %s", req.ToUser, contentType, req.Title),
	}).Build()
}

// SendToUserTemplate 发送给指定用户配置
var SendToUserTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "发送给指定用户",
		Desc:     `向指定用户发送消息通知。支持三种内容类型：Markdown（默认，支持加粗、列表、链接等）、HTML（富文本）、纯文本。`,
		Tags:     []string{"消息通知", "发送", "用户"},
		Request:  &SendToUserReq{},
		Response: &SendToUserResp{},
	},
}

func init() {
	// 注册 Form 函数 - 发送给指定用户
	packageContext.POST("to_user.form", SendToUser, SendToUserTemplate)
}
