package ticket_management

import (
	"fmt"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// TicketSubmitReq 提交工单请求
type TicketSubmitReq struct {
	Title      string `json:"title" widget:"name:工单标题;type:input" validate:"required,min=2,max=200"`
	Content    string `json:"content" widget:"name:详细内容;type:richtext;height:360" validate:"required"`
	Priority   string `json:"priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:67C23A,E6A23C,F56C6C;render_default:中" validate:"required,oneof=低 中 高"`
	Status     string `json:"status" widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;options_colors:909399,E6A23C,67C23A,F56C6C;render_default:待处理" validate:"required,oneof=待处理 处理中 已完成 已关闭"`
	Classify   string `json:"classify" widget:"name:问题分类;type:select;options:系统问题,账户问题,支付问题,咨询,其他;options_colors:F56C6C,E6A23C,FF9800,409EFF,909399" validate:"required,oneof=系统问题 账户问题 支付问题 咨询 其他"`
	Phone      string `json:"phone" widget:"name:联系电话;type:input" validate:"required"`
	Source     string `json:"source" widget:"name:工单来源;type:radio;options:电话,邮件,在线,现场,其他;render_default:在线" validate:"required,oneof=电话 邮件 在线 现场 其他"`
	Attachment string `json:"attachment" widget:"name:附件;type:files;accept:.pdf,.doc,.docx,.png,.jpg,.jpeg;max_count:5;max_size:50MB"`
}

// TicketSubmitResp 提交工单响应
type TicketSubmitResp struct {
	Result string `json:"result" widget:"name:提交结果;type:text_area"`
}

// TicketSubmit 提交工单入口
func TicketSubmit(ctx *app.Context, resp response.Response) error {
	var req TicketSubmitReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()
	ticket := Ticket{
		Title:      req.Title,
		Content:    req.Content,
		Priority:   req.Priority,
		Status:     req.Status,
		Classify:   req.Classify,
		Phone:      req.Phone,
		Source:     req.Source,
		Attachment: req.Attachment,
		CreatedBy:  ctx.GetRequestUser(),
	}

	if err := db.Create(&ticket).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[TicketSubmit] 创建工单失败, req: %+v, err: %v", req, err)
		return fmt.Errorf("[系统错误]-[TicketSubmit] 创建工单失败: %w", err)
	}

	return resp.Form(&TicketSubmitResp{
		Result: fmt.Sprintf("工单创建成功，工单ID: %d", ticket.ID),
	}).Build()
}

// TicketSubmitTemplate 提交工单配置
var TicketSubmitTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "提交工单",
		Desc:     `客户或客服提交新工单。`,
		Tags:     []string{"工单管理"},
		Request:  &TicketSubmitReq{},
		Response: &TicketSubmitResp{},
	},
}

func init() {
	packageContext.POST("ticket_submit.form", TicketSubmit, TicketSubmitTemplate)
}
