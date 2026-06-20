package printdrop

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// PrintOrder 打印订单表
type PrintOrder struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`                                                        // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	OrderNo     int        `json:"order_no" gorm:"column:order_no" widget:"name:订单号;type:integer" hide:"create,update"`                                                            // 前端仅在列表展示，不进入新增/编辑表单。
	FileName    string     `json:"file_name" gorm:"column:file_name" widget:"name:文件名;type:text" hide:"create,update"`                                                             // 前端仅在列表展示，不进入新增/编辑表单。
	FilePath    string     `json:"file_path" gorm:"column:file_path;type:text" widget:"name:文件;type:files;accept:.pdf,.doc,.docx,.png,.jpg,.jpeg,.ppt,.pptx" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	PrintSize   string     `json:"print_size" gorm:"column:print_size" widget:"name:打印尺寸;type:select;options:A4,A3,A5;options_colors:409EFF,67C23A,FF9800" validate:"required,oneof=A4 A3 A5"`
	PaperColor  string     `json:"paper_color" gorm:"column:paper_color" widget:"name:纸张颜色;type:select;options:白色,彩色;options_colors:909399,FF9800" validate:"required,oneof=白色 彩色"`
	PrintCount  int        `json:"print_count" gorm:"column:print_count" widget:"name:打印份数;type:integer;min:1;max:999;unit:份" validate:"required,min=1,max=999"`
	PrintMethod string     `json:"print_method" gorm:"column:print_method" widget:"name:打印方式;type:select;options:单面打印,双面打印;options_colors:409EFF,67C23A;render_default:单面打印" validate:"oneof=单面打印 双面打印"`
	Remark      string     `json:"remark" gorm:"column:remark;type:text" widget:"name:备注;type:text_area"`
	Status      string     `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:待打印,已打印;options_colors:E6A23C,67C23A;render_default:待打印" validate:"oneof=待打印 已打印"`
	SubmitTime  types.Time `json:"submit_time" gorm:"column:submit_time;type:datetime" widget:"name:提交时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	Submitter   string     `json:"submitter" gorm:"column:submitter" widget:"name:提交人;type:user" hide:"create,update"`                                                   // 前端仅在列表展示，不进入新增/编辑表单。
}

func (PrintOrder) TableName() string {
	return "print_order"
}

// PrintOrderListReq 打印订单列表请求
type PrintOrderListReq struct {
	FileName     string `json:"file_name" form:"file_name" widget:"name:文件名;type:input"`
	Status       string `json:"status" form:"status" widget:"name:状态;type:select;options:待打印,已打印;options_colors:E6A23C,67C23A"`
	Submitter    string `json:"submitter" form:"submitter" widget:"name:提交人;type:user"`
	CreatedStart string `json:"created_start" form:"created_start" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	CreatedEnd   string `json:"created_end" form:"created_end" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

// PrintOrderList 打印订单列表
func PrintOrderList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req PrintOrderListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&PrintOrder{})

	if req.FileName != "" {
		queryDB = queryDB.Where("file_name LIKE ?", "%"+req.FileName+"%")
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}
	if req.Submitter != "" {
		queryDB = queryDB.Where("submitter = ?", req.Submitter)
	}
	if req.CreatedStart != "" {
		queryDB = queryDB.Where("created_at >= ?", req.CreatedStart)
	}
	if req.CreatedEnd != "" {
		queryDB = queryDB.Where("created_at <= ?", req.CreatedEnd)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("created_at DESC")
	}

	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var orders []PrintOrder
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&orders).Error; err != nil {
		return err
	}

	return resp.Table(response.TableResult{
		Items:      orders,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// PrintOrderListTemplate 打印订单管理配置
var PrintOrderListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "打印订单管理",
		Desc:         `管理打印订单，记录客户上传的文件和打印要求`,
		Tags:         []string{"打印管理", "订单管理"},
		Request:      &PrintOrderListReq{},
		CreateTables: []interface{}{&PrintOrder{}},
	},
	AutoCrudTable: &PrintOrder{},

	// 只允许更新状态字段，不允许直接修改文件路径等核心信息
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()

		var updateFields PrintOrder
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, err
		}

		updates := req.ChangedFields()

		// 只允许更新以下字段
		allowedFields := map[string]bool{
			"status": true,
		}

		filteredUpdates := make(map[string]interface{})
		for key, value := range updates {
			if allowedFields[key] {
				filteredUpdates[key] = value
			}
		}

		if len(filteredUpdates) == 0 {
			return nil, fmt.Errorf("没有可更新的字段")
		}

		err := db.Model(&PrintOrder{}).Where("id = ?", req.GetId()).Updates(filteredUpdates).Error
		if err != nil {
			logger.Errorf(ctx, "Update print order err: %v", err)
			return nil, err
		}

		return &callback.OnTableUpdateRowResp{}, nil
	},
}

// generateOrderNo 生成当天订单号
func generateOrderNo(db *gorm.DB) (int, error) {
	today := time.Now().Format("2006-01-02")

	var maxOrder PrintOrder
	err := db.Where("DATE(created_at) = ?", today).Order("order_no DESC").First(&maxOrder).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 1, nil
		}
		return 0, err
	}

	return maxOrder.OrderNo + 1, nil
}

func init() {
	packageContext.GET("print_order_list.table", PrintOrderList, PrintOrderListTemplate)
}
