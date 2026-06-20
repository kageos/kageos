//<文件名>send_to_user_and_department.go</文件名>

package message

import (
	"fmt"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// SendToUserAndDepartmentReq 混合发送请求结构体
type SendToUserAndDepartmentReq struct {
	// 框架标签：widget:"type:users" - 多用户选择组件
	// 消息接收人，可选多个
	ToUsers string `json:"to_users" widget:"name:接收用户;type:users"`

	// 框架标签：widget:"type:departments" - 多部门选择组件
	// 消息接收部门，可选多个
	ToDepartments string `json:"to_departments" widget:"name:接收部门;type:departments"`

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

// SendToUserAndDepartmentResp 混合发送响应结构体
type SendToUserAndDepartmentResp struct {
	// 发送结果
	Result string `json:"result" widget:"name:发送结果;type:text_area"`
}

// SendToUserAndDepartment 混合发送入口（同时发送给用户和部门）
func SendToUserAndDepartment(ctx *app.Context, resp response.Response) error {
	var req SendToUserAndDepartmentReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	// 校验：用户和部门至少需要选择一个
	if req.ToUsers == "" && req.ToDepartments == "" {
		return fmt.Errorf("请至少选择一个接收用户或接收部门")
	}

	// 默认 contentType 为 markdown
	contentType := req.ContentType
	if contentType == "" {
		contentType = "markdown"
	}

	_, err := sendMessage(ctx, sendMessagePayload{
		ToUsers:       req.ToUsers,
		ToDepartments: req.ToDepartments,
		Title:         req.Title,
		Content:       req.Content,
		ContentType:   contentType,
	})
	if err != nil {
		return fmt.Errorf("[系统错误]-[SendToUserAndDepartment] 发送消息失败, req: %+v, err: %w", req, err)
	}

	resultMsg := fmt.Sprintf("消息发送成功！\n\n")
	if req.ToUsers != "" {
		resultMsg += fmt.Sprintf("接收用户: %s\n", req.ToUsers)
	}
	if req.ToDepartments != "" {
		resultMsg += fmt.Sprintf("接收部门: %s\n", req.ToDepartments)
	}
	resultMsg += fmt.Sprintf("内容类型: %s\n标题: %s", contentType, req.Title)

	return resp.Form(&SendToUserAndDepartmentResp{
		Result: resultMsg,
	}).Build()
}

// SendToUserAndDepartmentTemplate 混合发送配置
var SendToUserAndDepartmentTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "混合发送（用户+部门）",
		Desc:     `同时向指定用户和部门发送消息通知。支持三种内容类型：Markdown（默认，支持加粗、列表、链接等）、HTML（富文本）、纯文本。用户和部门至少需要选择其中一个。`,
		Tags:     []string{"消息通知", "发送", "用户", "部门", "混合"},
		Request:  &SendToUserAndDepartmentReq{},
		Response: &SendToUserAndDepartmentResp{},
	},
}

func init() {
	// 注册 Form 函数 - 混合发送（用户+部门）
	packageContext.POST("to_recipients.form", SendToUserAndDepartment, SendToUserAndDepartmentTemplate)
}
