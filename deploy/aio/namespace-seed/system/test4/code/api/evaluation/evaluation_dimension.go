package evaluation

import (
	"fmt"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// EvaluationDimension 评价维度表
type EvaluationDimension struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	ActivityID  int    `json:"activity_id" gorm:"column:activity_id;index" widget:"name:所属活动;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Name        string `json:"name" gorm:"column:name" widget:"name:评价维度名称;type:input;placeholder:例如服务态度、专业能力" validate:"required,min=1,max=50"`
	Description string `json:"description" gorm:"column:description;type:text" widget:"name:维度描述;type:text_area;placeholder:对该维度的评价说明"`
	SortOrder   int    `json:"sort_order" gorm:"column:sort_order;default:0" widget:"name:排序序号;type:integer;min:0;render_default:1"`

	Activity     *EvaluationActivity `json:"-" widget:"-" gorm:"foreignKey:ActivityID"`
	ActivityName string              `json:"activity_name" gorm:"-" widget:"name:所属活动;type:text" hide:"create,update"`
}

func (EvaluationDimension) TableName() string {
	return "evaluation_dimension"
}

// ================ 列表 ================

// EvaluationDimensionListReq 评价维度列表请求
type EvaluationDimensionListReq struct {
	ActivityID int    `json:"activity_id" form:"activity_id" widget:"name:所属活动;type:select" callback:"OnSelectFuzzy"`
	Name       string `json:"name" form:"name" widget:"name:评价维度名称;type:input"`
	StartTime  string `json:"start_time" form:"start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime    string `json:"end_time" form:"end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

// EvaluationDimensionList 评价维度管理
func EvaluationDimensionList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	var req EvaluationDimensionListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&EvaluationDimension{}).Preload("Activity")
	if req.ActivityID > 0 {
		queryDB = queryDB.Where("activity_id = ?", req.ActivityID)
	}
	if req.Name != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.StartTime != "" {
		queryDB = queryDB.Where("created_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		queryDB = queryDB.Where("created_at <= ?", req.EndTime)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("sort_order ASC, id ASC")
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var items []EvaluationDimension
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&items).Error; err != nil {
		return err
	}

	for i := range items {
		if items[i].Activity != nil {
			items[i].ActivityName = items[i].Activity.Name
		}
	}

	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// EvaluationDimensionListTemplate 评价维度管理配置
var EvaluationDimensionListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "评价维度管理",
		Desc:         "维护每个评价活动下的评分维度。",
		Request:      &EvaluationDimensionListReq{},
		CreateTables: []interface{}{&EvaluationDimension{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"activity_id": evaluationOnSelectFuzzyActivity,
		},
	},
	AutoCrudTable: &EvaluationDimension{},

	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		var row EvaluationDimension
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		var activity EvaluationActivity
		if err := ctx.GetGormDB().Where("id = ?", row.ActivityID).First(&activity).Error; err != nil {
			return nil, fmt.Errorf("评价活动不存在")
		}
		if err := ctx.GetGormDB().Create(&row).Error; err != nil {
			logger.Errorf(ctx, "[系统错误]-[EvaluationDimension-Add] 创建失败, err: %v", err)
			return nil, fmt.Errorf("[系统错误]-[EvaluationDimension-Add] 创建失败, err: %w", err)
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},

	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		updates := req.ChangedFields()
		if err := ctx.GetGormDB().Model(&EvaluationDimension{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
			logger.Errorf(ctx, "[系统错误]-[EvaluationDimension-Update] 更新失败, err: %v", err)
			return nil, fmt.Errorf("[系统错误]-[EvaluationDimension-Update] 更新失败, err: %w", err)
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},

	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		if err := ctx.GetGormDB().Where("id IN ?", req.GetIds()).Delete(&EvaluationDimension{}).Error; err != nil {
			logger.Errorf(ctx, "[系统错误]-[EvaluationDimension-Delete] 删除失败, err: %v", err)
			return nil, fmt.Errorf("[系统错误]-[EvaluationDimension-Delete] 删除失败, err: %w", err)
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func init() {
	packageContext.GET("evaluation_dimension_list.table", EvaluationDimensionList, EvaluationDimensionListTemplate)
}
