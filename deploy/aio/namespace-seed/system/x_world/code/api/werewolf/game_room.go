// game_room.go
// 游戏房间管理：数据模型、列表 Handler、Template

package werewolf

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

// ================ 数据模型 ================

// GameRoom 游戏房间表
type GameRoom struct {
	ID           int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt    types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt    types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	RoomNo       string         `json:"room_no" gorm:"column:room_no;uniqueIndex;comment:房间号" widget:"name:房间号;type:input" hide:"create,update"`
	HostName     string         `json:"host_name" gorm:"column:host_name;comment:房主" widget:"name:房主;type:input" validate:"required"`
	Status       string         `json:"status" gorm:"column:status;comment:状态;default:等待开始" widget:"name:状态;type:select;options:等待开始,夜晚,白天,投票,结算;options_colors:909399,673AB7,FF9800,409EFF,67C23A;render_default:等待开始"`
	PlayerCount  int            `json:"player_count" gorm:"column:player_count;comment:存活玩家数;default:0" widget:"name:存活玩家数;type:integer;unit:人"`
	MaxPlayers   int            `json:"max_players" gorm:"column:max_players;comment:人数上限;default:6" widget:"name:人数上限;type:integer;min:4;max:12;render_default:6"`
	CurrentRound int            `json:"current_round" gorm:"column:current_round;comment:当前轮次;default:0" widget:"name:当前轮次;type:integer;unit:轮"`
	Creator      string         `json:"creator" gorm:"column:creator" widget:"name:创建人;type:user" hide:"create,update"`
}

func (GameRoom) TableName() string {
	return "game_room"
}

// ================ 辅助函数 ================

// getRoomStatus 获取房间状态描述
func getRoomStatusDisplay(status string) string {
	switch status {
	case "等待开始":
		return "等待开始"
	case "夜晚":
		return "夜晚"
	case "白天":
		return "白天"
	case "投票":
		return "投票"
	case "结算":
		return "结算"
	default:
		return status
	}
}

// ================ 游戏房间列表 ================

// GameRoomListReq 游戏房间列表请求
type GameRoomListReq struct {
	RoomNo            string `json:"room_no" form:"room_no" widget:"name:房间号;type:input"`
	HostName          string `json:"host_name" form:"host_name" widget:"name:房主;type:input"`
	CreatorName       string `json:"creator_name" form:"creator_name" widget:"name:创建人;type:input"`
	Status            string `json:"status" form:"status" widget:"name:状态;type:select;options:等待开始,夜晚,白天,投票,结算;options_colors:909399,673AB7,FF9800,409EFF,67C23A"`
	CreatedStartTime  string `json:"created_start_time" form:"created_start_time" widget:"name:创建开始时间;type:datetime"`
	CreatedEndTime    string `json:"created_end_time" form:"created_end_time" widget:"name:创建结束时间;type:datetime"`
	query.PageSortReq `widget:"-"`
}

// GameRoomList 游戏房间列表
func GameRoomList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req GameRoomListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&GameRoom{})
	if req.RoomNo != "" {
		queryDB = queryDB.Where("room_no LIKE ?", "%"+req.RoomNo+"%")
	}
	if req.HostName != "" {
		queryDB = queryDB.Where("host_name LIKE ?", "%"+req.HostName+"%")
	}
	if req.CreatorName != "" {
		queryDB = queryDB.Where("creator = ?", req.CreatorName)
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
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

	var rooms []GameRoom
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&rooms).Error; err != nil {
		return err
	}

	return resp.Table(response.TableResult{
		Items:      rooms,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// GameRoomListTemplate 游戏房间管理配置
var GameRoomListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "游戏房间管理",
		Desc:         `管理狼人杀游戏房间，包括创建房间、查看房间状态等`,
		Tags:         []string{"狼人杀", "房间管理"},
		Request:      &GameRoomListReq{},
		CreateTables: []interface{}{&GameRoom{}},
	},
	AutoCrudTable: &GameRoom{},

	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var room GameRoom
		if err := ctx.ShouldBindValidate(&room); err != nil {
			return nil, err
		}

		room.Creator = ctx.GetRequestUser()
		room.RoomNo = generateRoomNo()
		room.Status = "等待开始"
		room.PlayerCount = 0
		if room.MaxPlayers == 0 {
			room.MaxPlayers = 6
		}
		room.CurrentRound = 0

		err := db.Create(&room).Error
		if err != nil {
			logger.Errorf(ctx, "Create game room err: %v", err)
			return nil, err
		}

		return &callback.OnTableAddRowResp{Data: &room}, nil
	},

	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		var updateFields GameRoom
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, err
		}

		updates := req.ChangedFields()
		err := db.Model(&GameRoom{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "Update game room err: %v", err)
			return nil, err
		}

		return &callback.OnTableUpdateRowResp{}, nil
	},

	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()

		err := db.Transaction(func(tx *gorm.DB) error {
			var rooms []GameRoom
			if err := tx.Where("id IN ?", req.GetIds()).Find(&rooms).Error; err != nil {
				return err
			}

			roomNos := make([]string, 0)
			for _, room := range rooms {
				roomNos = append(roomNos, room.RoomNo)
			}

			tx.Where("room_no IN ?", roomNos).Delete(&GameRecord{})
			tx.Where("room_no IN ?", roomNos).Delete(&Player{})
			tx.Where("id IN ?", req.GetIds()).Delete(&GameRoom{})
			return nil
		})

		if err != nil {
			logger.Errorf(ctx, "Delete game room err: %v", err)
			return nil, err
		}

		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

// generateRoomNo 生成房间号
func generateRoomNo() string {
	timestamp := time.Now().UnixNano() / 1e6
	return fmt.Sprintf("ROOM%06d", timestamp%1000000)
}

// ================ API 注册 ================

func init() {
	packageContext.GET("game_room_list.table", GameRoomList, GameRoomListTemplate)
}
