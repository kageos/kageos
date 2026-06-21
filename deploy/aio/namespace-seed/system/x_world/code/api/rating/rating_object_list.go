// rating_object_list.go
// 评价对象管理：数据模型、列表 Handler、Template

package rating

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

// RatingObject 评价对象表
type RatingObject struct {
	ID            int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt     types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	Name          string         `json:"name" gorm:"column:name;comment:事物名称" widget:"name:事物名称;type:input" validate:"required,min=1,max=200"`
	ObjectType    string         `json:"object_type" gorm:"column:object_type;comment:类型" widget:"name:类型;type:select;options:电影,图书,音乐,餐厅,酒店,商品,服务,课程,景点,其他;options_colors:E91E63,9C27B0,673AB7,3F51B5,2196F3,00BCD4,009688,4CAF50,8BC34A,FF9800" validate:"required"`
	Description   string         `json:"description" gorm:"column:description;type:text;comment:描述" widget:"name:描述;type:text_area" validate:"max=500"`
	DisplayImages string         `json:"display_images" gorm:"column:display_images;type:text;comment:展示图片" widget:"name:展示图片;type:files;accept:image/*;max_count:9;thumbnail:true;list_preview:true"`
	DisplayVideo  string         `json:"display_video" gorm:"column:display_video;type:text;comment:展示视频" widget:"name:展示视频;type:files;accept:video/*;max_count:1;thumbnail:true"`
	AverageRating float64        `json:"average_rating" gorm:"column:average_rating;type:decimal(3,2);comment:平均评分;default:0" widget:"name:平均评分;type:progress;min:0;max:5;unit:分" hide:"create,update"`
	RatingCount   int            `json:"rating_count" gorm:"column:rating_count;comment:评分次数;default:0" widget:"name:评分次数;type:integer;unit:次" hide:"create,update"`
	CreatedBy     string         `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
	RecordsLink   string         `json:"records_link" gorm:"-" widget:"name:评价记录;type:link;target:_blank" hide:"create,update"`
}

func (RatingObject) TableName() string {
	return "rating_object"
}

// ================ 评价对象管理 ================

// RatingObjectListReq 评价对象列表请求
type RatingObjectListReq struct {
	Name       string `json:"name" form:"name" widget:"name:事物名称;type:input"`
	ObjectType string `json:"object_type" form:"object_type" widget:"name:类型;type:select;options:电影,图书,音乐,餐厅,酒店,商品,服务,课程,景点,其他;options_colors:E91E63,9C27B0,673AB7,3F51B5,2196F3,00BCD4,009688,4CAF50,8BC34A,FF9800"`
	StartTime  string `json:"start_time" form:"start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime    string `json:"end_time" form:"end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	CreatedBy  string `json:"created_by" form:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`

	query.PageSortReq `widget:"-"`
}

// RatingObjectList 评价对象管理
func RatingObjectList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req RatingObjectListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&RatingObject{})
	if req.Name != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.ObjectType != "" {
		queryDB = queryDB.Where("object_type = ?", req.ObjectType)
	}
	if req.StartTime != "" {
		queryDB = queryDB.Where("created_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		queryDB = queryDB.Where("created_at <= ?", req.EndTime)
	}
	if req.CreatedBy != "" {
		queryDB = queryDB.Where("created_by = ?", req.CreatedBy)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}

	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var objects []RatingObject
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&objects).Error; err != nil {
		return err
	}

	for i := range objects {
		params := RatingRecord{ObjectID: objects[i].ID}
		objects[i].RecordsLink, _ = ctx.BuildFunctionUrlWithText("rating_record_list.table", params, "查看评价记录")
	}

	return resp.Table(response.TableResult{
		Items:      objects,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// RatingObjectListTemplate 评价对象管理配置
var RatingObjectListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "评价对象管理",
		Desc:         `维护待评价的事物，包括名称、类型、描述、展示资料等`,
		Tags:         []string{"评分系统", "对象管理"},
		Request:      &RatingObjectListReq{},
		CreateTables: []interface{}{&RatingObject{}, &RatingRecord{}},
	},
	AutoCrudTable: &RatingObject{},

	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var obj RatingObject
		if err := ctx.ShouldBindValidate(&obj); err != nil {
			return nil, err
		}
		obj.CreatedBy = ctx.GetRequestUser()
		obj.AverageRating = 0
		obj.RatingCount = 0

		if err := db.Create(&obj).Error; err != nil {
			logger.Errorf(ctx, "Create rating object err: %v", err)
			return nil, err
		}

		return &callback.OnTableAddRowResp{Data: &obj}, nil
	},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("rating_object_list.table", RatingObjectList, RatingObjectListTemplate)
}
