package midnight_pub

import (
	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// OrderRecord 点单记录
type OrderRecord struct {
	ID            int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt     types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt     types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	CharacterName string         `json:"character_name" gorm:"column:character_name" widget:"name:角色名;type:input" validate:"required"`
	DrinkName     string         `json:"drink_name" gorm:"column:drink_name" widget:"name:酒名;type:select;options:whiskey_neat,whiskey_sour,martini,beer,mojito,old_fashioned;options_colors:E6A23C,FF9800,9C27B0,FFC107,4CAF50,795548" validate:"required"`
	OrderTime     types.Time     `json:"order_time" gorm:"column:order_time;type:datetime" widget:"name:点单时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	Mood          string         `json:"mood" gorm:"column:mood" widget:"name:心情;type:select;options:放松,感慨,忧伤,愉悦,沉思;options_colors:4CAF50,FF9800,9C27B0,FFC107,607D8B"`
	Message       string         `json:"message" gorm:"column:message" widget:"name:配文;type:text_area"`
}

func (o *OrderRecord) TableName() string {
	return "midnight_pub_order_record"
}

var OrderRecordTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:    "点单记录",
		Request: &OrderRecordListReq{},
		CreateTables: []interface{}{
			&OrderRecord{},
		},
	},
	AutoCrudTable: &OrderRecord{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row OrderRecord
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		if err := db.Create(&row).Error; err != nil {
			logger.Errorf(ctx, "OrderRecord Create err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
}

type OrderRecordListReq struct {
	CharacterName     string `json:"character_name" form:"character_name" widget:"name:角色名;type:input"`
	DrinkName         string `json:"drink_name" form:"drink_name" widget:"name:酒名;type:select;options:whiskey_neat,whiskey_sour,martini,beer,mojito,old_fashioned;options_colors:E6A23C,FF9800,9C27B0,FFC107,4CAF50,795548"`
	CreatedStart      string `json:"created_start" form:"created_start" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	CreatedEnd        string `json:"created_end" form:"created_end" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	query.PageSortReq `widget:"-"`
}

func OrderRecordList(ctx *app.Context, resp response.Response) error {
	var req OrderRecordListReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "OrderRecordList ShouldBind err: %v", err)
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&OrderRecord{})
	if req.CharacterName != "" {
		queryDB = queryDB.Where("character_name LIKE ?", "%"+req.CharacterName+"%")
	}
	if req.DrinkName != "" {
		queryDB = queryDB.Where("drink_name = ?", req.DrinkName)
	}
	if req.CreatedStart != "" {
		queryDB = queryDB.Where("created_at >= ?", req.CreatedStart)
	}
	if req.CreatedEnd != "" {
		queryDB = queryDB.Where("created_at <= ?", req.CreatedEnd)
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	var lists []*OrderRecord
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{
		Items:      lists,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func init() {
	packageContext.GET("order_record_list.table", OrderRecordList, OrderRecordTemplate)
}
