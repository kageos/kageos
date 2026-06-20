// game_record.go
// 游戏记录管理：数据模型、列表 Handler、Template

package werewolf

import (
	"fmt"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// GameRecord 游戏记录表
type GameRecord struct {
	ID         int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt  types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:记录时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	RecordID   string         `json:"record_id" gorm:"column:record_id;uniqueIndex;comment:记录ID" widget:"name:记录ID;type:input" hide:"create,update"`
	RoomNo     string         `json:"room_no" gorm:"column:room_no;index;comment:房间号" widget:"name:房间号;type:input" validate:"required"`
	Round      int            `json:"round" gorm:"column:round;comment:游戏轮次" widget:"name:轮次;type:integer;unit:轮"`
	Phase      string         `json:"phase" gorm:"column:phase;comment:当前阶段" widget:"name:阶段;type:select;options:夜晚,白天发言,投票,遗言;options_colors:673AB7,FF9800,409EFF,E91E63"`
	PlayerName string         `json:"player_name" gorm:"column:player_name;comment:执行操作的玩家" widget:"name:玩家;type:input"`
	Content    string         `json:"content" gorm:"column:content;type:text;comment:操作内容" widget:"name:内容;type:text_area"`
}

func (GameRecord) TableName() string {
	return "game_record"
}

// ================ 游戏记录列表 ================

// GameRecordListReq 游戏记录列表请求
type GameRecordListReq struct {
	RoomNo            string `json:"room_no" form:"room_no" widget:"name:房间号;type:input"`
	Round             int    `json:"round" form:"round" widget:"name:轮次;type:integer"`
	Phase             string `json:"phase" form:"phase" widget:"name:阶段;type:select;options:夜晚,白天发言,投票,遗言;options_colors:673AB7,FF9800,409EFF,E91E63"`
	PlayerName        string `json:"player_name" form:"player_name" widget:"name:玩家;type:input"`
	CreatedStartTime  string `json:"created_start_time" form:"created_start_time" widget:"name:创建开始时间;type:datetime"`
	CreatedEndTime    string `json:"created_end_time" form:"created_end_time" widget:"name:创建结束时间;type:datetime"`
	query.PageSortReq `widget:"-"`
}

// GameRecordList 游戏记录列表
func GameRecordList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req GameRecordListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&GameRecord{})
	if req.RoomNo != "" {
		queryDB = queryDB.Where("room_no = ?", req.RoomNo)
	}
	if req.Round > 0 {
		queryDB = queryDB.Where("round = ?", req.Round)
	}
	if req.Phase != "" {
		queryDB = queryDB.Where("phase = ?", req.Phase)
	}
	if req.PlayerName != "" {
		queryDB = queryDB.Where("player_name LIKE ?", "%"+req.PlayerName+"%")
	}
	if req.CreatedStartTime != "" {
		queryDB = queryDB.Where("created_at >= ?", req.CreatedStartTime)
	}
	if req.CreatedEndTime != "" {
		queryDB = queryDB.Where("created_at <= ?", req.CreatedEndTime)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var records []GameRecord
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&records).Error; err != nil {
		return err
	}

	return resp.Table(response.TableResult{
		Items:      records,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// GameRecordListTemplate 游戏记录管理配置
var GameRecordListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "游戏记录查询",
		Desc:         `查询游戏操作记录，包括发言、投票、行动等历史`,
		Tags:         []string{"狼人杀", "记录管理"},
		Request:      &GameRecordListReq{},
		CreateTables: []interface{}{&GameRecord{}},
	},
	AutoCrudTable: &GameRecord{},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("game_record_list.table", GameRecordList, GameRecordListTemplate)
}
