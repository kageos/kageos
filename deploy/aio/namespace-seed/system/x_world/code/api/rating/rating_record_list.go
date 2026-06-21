// rating_record_list.go
// 评价记录查询：只读查看用户提交的评价记录

package rating

import (
	"fmt"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// RatingRecord 评价记录表
type RatingRecord struct {
	ID          int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt   types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:评价时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	ObjectID    int            `json:"object_id" gorm:"column:object_id;comment:事物ID;index" widget:"name:评价对象;type:select" callback:"OnSelectFuzzy" validate:"required"`
	ObjectName  string         `json:"object_name" gorm:"-" widget:"name:事物名称;type:text" hide:"create,update"`
	ObjectLink  string         `json:"object_link" gorm:"-" widget:"name:事物详情;type:link;target:_blank" hide:"create,update"`
	Rating      float64        `json:"rating" gorm:"column:rating;type:decimal(2,1);comment:评分" widget:"name:评分;type:rate;max:5;allow_half:true" validate:"required,min=0.5,max=5"`
	Comment     string         `json:"comment" gorm:"column:comment;type:text;comment:评论" widget:"name:评论;type:text_area" validate:"max=1000"`
	Attachments string         `json:"attachments" gorm:"column:attachments;type:text;comment:图片附件" widget:"name:图片附件;type:files;accept:image/*;max_count:9;thumbnail:true"`
	Submitter   string         `json:"submitter" gorm:"column:submitter;comment:提交人" widget:"name:提交人;type:user"`
	Object      *RatingObject  `json:"-" widget:"-" gorm:"foreignKey:ObjectID"`
}

func (RatingRecord) TableName() string {
	return "rating_record"
}

// ================ 评价记录查询 ================

// RatingRecordListReq 评价记录列表请求
type RatingRecordListReq struct {
	ObjectID   int    `json:"object_id" form:"object_id" widget:"name:评价对象;type:select" callback:"OnSelectFuzzy"`
	ObjectName string `json:"object_name" form:"object_name" widget:"name:事物名称;type:input"`
	StartTime  string `json:"start_time" form:"start_time" widget:"name:评价开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime    string `json:"end_time" form:"end_time" widget:"name:评价结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	Submitter  string `json:"submitter" form:"submitter" widget:"name:提交人;type:user"`
	MinRating  int    `json:"min_rating" form:"min_rating" widget:"name:最低评分;type:integer;min:1;max:5;step:1"`

	query.PageSortReq `widget:"-"`
}

// RatingRecordList 评价记录查询
func RatingRecordList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req RatingRecordListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&RatingRecord{}).Preload("Object")
	if req.ObjectID > 0 {
		queryDB = queryDB.Where("object_id = ?", req.ObjectID)
	}
	if req.ObjectName != "" {
		var objectIDs []int
		if err := db.Model(&RatingObject{}).
			Where("name LIKE ?", "%"+req.ObjectName+"%").
			Pluck("id", &objectIDs).Error; err == nil && len(objectIDs) > 0 {
			queryDB = queryDB.Where("object_id IN ?", objectIDs)
		} else {
			queryDB = queryDB.Where("1 = 0")
		}
	}
	if req.StartTime != "" {
		queryDB = queryDB.Where("created_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		queryDB = queryDB.Where("created_at <= ?", req.EndTime)
	}
	if req.Submitter != "" {
		queryDB = queryDB.Where("submitter = ?", req.Submitter)
	}
	if req.MinRating > 0 {
		queryDB = queryDB.Where("rating >= ?", req.MinRating)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}

	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var records []RatingRecord
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&records).Error; err != nil {
		return err
	}

	for i := range records {
		if records[i].Object != nil && records[i].Object.ID > 0 {
			records[i].ObjectName = records[i].Object.Name
			params := RatingObject{ID: records[i].Object.ID}
			records[i].ObjectLink, _ = ctx.BuildFunctionUrlWithText("rating_object_list.table", params, "查看详情")
		}
	}

	return resp.Table(response.TableResult{
		Items:      records,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// recordOnSelectFuzzyObject 评价对象下拉回调
func recordOnSelectFuzzyObject(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()

	var objects []RatingObject
	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		db = db.Where("name LIKE ? OR object_type LIKE ?", "%"+req.Keyword()+"%", "%"+req.Keyword()+"%").
			Limit(20)
	}
	db.Find(&objects)

	items := make([]*callback.SelectFuzzyItem, 0)
	for _, obj := range objects {
		items = append(items, &callback.SelectFuzzyItem{
			Value: obj.ID,
			Label: fmt.Sprintf("%s (%s)", obj.Name, obj.ObjectType),
			DisplayInfo: map[string]interface{}{
				"事物名称": obj.Name,
				"类型":   obj.ObjectType,
				"平均评分": fmt.Sprintf("%.1f", obj.AverageRating),
				"评分次数": obj.RatingCount,
			},
		})
	}

	return &callback.OnSelectFuzzyResp{Items: items}, nil
}

// RatingRecordListTemplate 评价记录查询配置
var RatingRecordListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "评价记录查询",
		Desc:         `查看所有评价记录，支持按对象、时间、提交人等条件筛选`,
		Tags:         []string{"评分系统", "记录查询"},
		Request:      &RatingRecordListReq{},
		CreateTables: []interface{}{&RatingRecord{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"object_id": recordOnSelectFuzzyObject,
		},
	},
	AutoCrudTable: &RatingRecord{},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("rating_record_list.table", RatingRecordList, RatingRecordListTemplate)
}
