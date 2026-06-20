package midnight_pub

import (
	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// DialogueRecord 对话记录
type DialogueRecord struct {
	ID            int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt     types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt     types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	CharacterName string         `json:"character_name" gorm:"column:character_name" widget:"name:角色名;type:input" validate:"required"`
	CharacterCode string         `json:"character_code" gorm:"column:character_code" widget:"name:角色代码;type:input" validate:"required"`
	Content       string         `json:"content" gorm:"column:content" widget:"name:发言内容;type:text_area" validate:"required"`
	SpeakTime     types.Time     `json:"speak_time" gorm:"column:speak_time;type:datetime" widget:"name:发言时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	TopicTag      string         `json:"topic_tag" gorm:"column:topic_tag" widget:"name:话题标签;type:input"`
}

func (d *DialogueRecord) TableName() string {
	return "midnight_pub_dialogue_record"
}

var DialogueRecordTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:    "对话记录",
		Request: &DialogueRecordListReq{},
		CreateTables: []interface{}{
			&DialogueRecord{},
		},
	},
	AutoCrudTable: &DialogueRecord{},
}

type DialogueRecordListReq struct {
	CharacterName     string `json:"character_name" form:"character_name" widget:"name:角色名;type:input"`
	TopicTag          string `json:"topic_tag" form:"topic_tag" widget:"name:话题标签;type:input"`
	CreatedStart      string `json:"created_start" form:"created_start" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	CreatedEnd        string `json:"created_end" form:"created_end" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	query.PageSortReq `widget:"-"`
}

func DialogueRecordList(ctx *app.Context, resp response.Response) error {
	var req DialogueRecordListReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "DialogueRecordList ShouldBind err: %v", err)
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&DialogueRecord{})
	if req.CharacterName != "" {
		queryDB = queryDB.Where("character_name LIKE ?", "%"+req.CharacterName+"%")
	}
	if req.TopicTag != "" {
		queryDB = queryDB.Where("topic_tag LIKE ?", "%"+req.TopicTag+"%")
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
	var lists []*DialogueRecord
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
	packageContext.GET("dialogue_record_list.table", DialogueRecordList, DialogueRecordTemplate)
}
