package vote

import (
	"fmt"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// VoteRecord 投票记录表（只读查询，不允许手工增删改）
type VoteRecord struct {
	ID            int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:记录ID;type:ID" hide:"create,update"`
	CreatedAt     types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:投票时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	TopicID       int            `json:"topic_id" gorm:"column:topic_id;comment:主题ID;index" widget:"name:投票主题ID;type:integer" hide:"create,update"`
	OptionID      int            `json:"option_id" gorm:"column:option_id;comment:选项ID;index" widget:"name:选项ID;type:integer" hide:"create,update"`
	TopicTitle    string         `json:"topic_title" gorm:"-" widget:"name:投票标题;type:input" hide:"create,update"`
	OptionContent string         `json:"option_content" gorm:"-" widget:"name:选项内容;type:input" hide:"create,update"`
	VoterName     string         `json:"voter_name" gorm:"column:voter_name;comment:投票人" widget:"name:投票人;type:user" hide:"create,update"`
	IsAnonymous   bool           `json:"is_anonymous" gorm:"column:is_anonymous;comment:是否匿名;default:false" widget:"name:是否匿名;type:switch" hide:"create,update"`
	Remark        string         `json:"remark" gorm:"column:remark;comment:投票备注" widget:"name:投票备注;type:text_area" hide:"create,update"`
	Topic         *VoteTopic     `json:"-" widget:"-" gorm:"foreignKey:TopicID"`
	Option        *VoteOption    `json:"-" widget:"-" gorm:"foreignKey:OptionID"`
}

func (VoteRecord) TableName() string {
	return "vote_record"
}

// ================ 投票记录查询 ================

// VoteRecordListReq 投票记录列表请求
type VoteRecordListReq struct {
	TopicTitle    string `json:"topic_title" form:"topic_title" gorm:"-" widget:"name:投票标题;type:input"`
	OptionContent string `json:"option_content" form:"option_content" gorm:"-" widget:"name:选项内容;type:input"`

	query.PageSortReq `widget:"-"`
}

// VoteRecordList 投票记录查询
func VoteRecordList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req VoteRecordListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&VoteRecord{})

	if req.TopicTitle != "" {
		var topicIDs []int
		if err := db.Model(&VoteTopic{}).
			Where("title LIKE ?", "%"+req.TopicTitle+"%").
			Pluck("id", &topicIDs).Error; err == nil && len(topicIDs) > 0 {
			queryDB = queryDB.Where("topic_id IN ?", topicIDs)
		} else {
			queryDB = queryDB.Where("1 = 0")
		}
	}

	if req.OptionContent != "" {
		var optionIDs []int
		if err := db.Model(&VoteOption{}).
			Where("content LIKE ?", "%"+req.OptionContent+"%").
			Pluck("id", &optionIDs).Error; err == nil && len(optionIDs) > 0 {
			queryDB = queryDB.Where("option_id IN ?", optionIDs)
		} else {
			queryDB = queryDB.Where("1 = 0")
		}
	}

	queryDB = queryDB.Preload("Topic").Preload("Option")

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var records []VoteRecord
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&records).Error; err != nil {
		return err
	}

	for i := range records {
		if records[i].Topic != nil && records[i].Topic.ID > 0 {
			records[i].TopicTitle = records[i].Topic.Title
			if records[i].Topic.IsAnonymous {
				records[i].VoterName = "匿名用户"
			}
		}
		if records[i].Option != nil && records[i].Option.ID > 0 {
			records[i].OptionContent = records[i].Option.Content
		}
	}

	return resp.Table(response.TableResult{
		Items:      records,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// VoteRecordListTemplate 投票记录查询配置（只读，无增删改）
var VoteRecordListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "投票记录查询",
		Desc:         `只读查看用户提交投票后产生的投票记录，不允许手工新增、编辑、删除`,
		Tags:         []string{"投票系统", "记录查询"},
		Request:      &VoteRecordListReq{},
		CreateTables: []interface{}{&VoteRecord{}},
	},
	AutoCrudTable: &VoteRecord{},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("vote_record_list.table", VoteRecordList, VoteRecordListTemplate)
}
