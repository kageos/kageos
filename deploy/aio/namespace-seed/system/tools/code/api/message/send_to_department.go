//<文件名>send_to_department.go</文件名>

package message

import (
	"fmt"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// SendToDepartmentReq 发送给指定部门请求结构体
type SendToDepartmentReq struct {
	// 框架标签：widget:"type:department" - 部门选择组件
	// 消息接收部门
	ToDepartment string `json:"to_department" widget:"name:接收部门;type:department" validate:"required"`

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

// SendToDepartmentResp 发送给指定部门响应结构体
type SendToDepartmentResp struct {
	// 发送结果
	Result string `json:"result" widget:"name:发送结果;type:text_area"`
}

// SendToDepartment 发送给指定部门入口
func SendToDepartment(ctx *app.Context, resp response.Response) error {
	var req SendToDepartmentReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	// 默认 contentType 为 markdown
	contentType := req.ContentType
	if contentType == "" {
		contentType = "markdown"
	}

	_, err := sendMessage(ctx, sendMessagePayload{
		ToDepartments: req.ToDepartment,
		Title:         req.Title,
		Content:       req.Content,
		ContentType:   contentType,
	})
	if err != nil {
		return fmt.Errorf("[系统错误]-[SendToDepartment] 发送消息失败, req: %+v, err: %w", req, err)
	}

	return resp.Form(&SendToDepartmentResp{
		Result: fmt.Sprintf("消息发送成功！\n\n接收部门: %s\n内容类型: %s\n标题: %s", req.ToDepartment, contentType, req.Title),
	}).Build()
}

// SendToDepartmentTemplate 发送给指定部门配置
var SendToDepartmentTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "发送给指定部门",
		Desc:     `向指定部门发送消息通知。支持三种内容类型：Markdown（默认，支持加粗、列表、链接等）、HTML（富文本）、纯文本。`,
		Tags:     []string{"消息通知", "发送", "部门"},
		Request:  &SendToDepartmentReq{},
		Response: &SendToDepartmentResp{},
	},
}

func init() {
	// 注册 Form 函数 - 发送给指定部门
	packageContext.POST("to_department.form", SendToDepartment, SendToDepartmentTemplate)
}
