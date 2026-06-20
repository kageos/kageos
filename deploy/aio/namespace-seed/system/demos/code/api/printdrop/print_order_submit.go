package printdrop

import (
	"errors"
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// PrintOrderSubmitReq 提交打印订单请求
type PrintOrderSubmitReq struct {
	UploadFile  string `json:"upload_file" widget:"name:上传文件;type:files;accept:.pdf,.doc,.docx,.png,.jpg,.jpeg,.ppt,.pptx;max_size:50MB;max_count:1" validate:"required"`
	PrintSize   string `json:"print_size" widget:"name:打印尺寸;type:select;options:A4,A3,A5;options_colors:409EFF,67C23A,FF9800;render_default:A4" validate:"required,oneof=A4 A3 A5"`
	PaperColor  string `json:"paper_color" widget:"name:纸张颜色;type:select;options:白色,彩色;options_colors:909399,FF9800;render_default:白色" validate:"required,oneof=白色 彩色"`
	PrintCount  int    `json:"print_count" widget:"name:打印份数;type:integer;min:1;max:999;step:1;render_default:1;unit:份" validate:"required,min=1,max=999"`
	PrintMethod string `json:"print_method" widget:"name:打印方式;type:select;options:单面打印,双面打印;options_colors:409EFF,67C23A;render_default:单面打印" validate:"oneof=单面打印 双面打印"`
	Remark      string `json:"remark" widget:"name:备注;type:text_area;placeholder:如装订要求、特殊纸张等"`
}

// PrintOrderSubmitResp 提交打印订单响应
type PrintOrderSubmitResp struct {
	OrderNo      int    `json:"order_no" widget:"name:订单号;type:integer"`
	SubmitResult string `json:"submit_result" widget:"name:提交结果;type:text"`
}

// DoPrintOrderSubmit 提交打印订单业务逻辑
func DoPrintOrderSubmit(ctx *app.Context, req *PrintOrderSubmitReq) (*PrintOrderSubmitResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoPrintOrderSubmit] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoPrintOrderSubmit] 数据库连接失败, req: %+v", req)
	}

	if req.UploadFile == "" {
		return nil, fmt.Errorf("请上传要打印的文件")
	}

	// 获取当前用户
	submitter := ctx.GetRequestUser()
	if submitter == "" {
		submitter = "匿名用户"
	}

	var orderNo int
	err := db.Transaction(func(tx *gorm.DB) error {
		// 生成当天订单号
		today := time.Now().Format("2006-01-02")
		var maxOrder PrintOrder
		err := tx.Where("DATE(created_at) = ?", today).Order("order_no DESC").First(&maxOrder).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			orderNo = 1
		} else {
			orderNo = maxOrder.OrderNo + 1
		}

		// 解析文件路径获取文件名
		fileName := "未命名文件"
		if req.UploadFile != "" {
			// 文件路径格式为 bucket/object_key，取最后一段作为文件名
			parts := splitFilePath(req.UploadFile)
			if len(parts) > 0 {
				fileName = parts[len(parts)-1]
			}
		}

		// 创建打印订单
		order := PrintOrder{
			OrderNo:     orderNo,
			FileName:    fileName,
			FilePath:    req.UploadFile,
			PrintSize:   req.PrintSize,
			PaperColor:  req.PaperColor,
			PrintCount:  req.PrintCount,
			PrintMethod: req.PrintMethod,
			Remark:      req.Remark,
			Status:      "待打印",
			SubmitTime:  types.Time(time.Now()),
			Submitter:   submitter,
		}

		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoPrintOrderSubmit] 创建订单失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoPrintOrderSubmit] 创建订单失败, req: %+v, err: %w", req, err)
	}

	return &PrintOrderSubmitResp{
		OrderNo:      orderNo,
		SubmitResult: fmt.Sprintf("订单提交成功！您的订单号是 %d，请等待打印", orderNo),
	}, nil
}

// splitFilePath 分割文件路径获取文件名
func splitFilePath(filePath string) []string {
	if filePath == "" {
		return nil
	}

	result := make([]string, 0)
	var current string

	for i := 0; i < len(filePath); i++ {
		if filePath[i] == '/' || filePath[i] == '\\' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(filePath[i])
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}

// PrintOrderSubmit 提交打印订单入口
func PrintOrderSubmit(ctx *app.Context, resp response.Response) error {
	var req PrintOrderSubmitReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoPrintOrderSubmit(ctx, &req)
	if err != nil {
		return err
	}

	return resp.Form(res).Build()
}

// PrintOrderSubmitTemplate 提交打印订单配置
var PrintOrderSubmitTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "提交打印订单",
		Desc:     `客户上传文件并填写打印参数，提交后生成打印订单`,
		Tags:     []string{"打印管理", "订单提交"},
		Request:  &PrintOrderSubmitReq{},
		Response: &PrintOrderSubmitResp{},
	},
}

func init() {
	packageContext.POST("print_order_submit.form", PrintOrderSubmit, PrintOrderSubmitTemplate)
}
