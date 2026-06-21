// player.go
// 玩家管理：数据模型、列表 Handler、Template

package werewolf

import (
	"fmt"
	"time"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// Player 玩家表
type Player struct {
	ID         int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt  types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	PlayerID   string         `json:"player_id" gorm:"column:player_id;uniqueIndex;comment:玩家ID" widget:"name:玩家ID;type:input" hide:"create,update"`
	RoomNo     string         `json:"room_no" gorm:"column:room_no;index;comment:房间号" widget:"name:房间号;type:input" validate:"required"`
	PlayerName string         `json:"player_name" gorm:"column:player_name;comment:玩家名称" widget:"name:玩家名;type:input" validate:"required"`
	Role       string         `json:"role" gorm:"column:role;comment:角色身份" widget:"name:角色;type:select;options:村民,狼人,预言家,女巫,猎人;options_colors:67C23A,F56C6C,409EFF,9C27B0,FF9800" hide:"create,update"`
	Status     string         `json:"status" gorm:"column:status;comment:存活状态;default:存活" widget:"name:状态;type:select;options:存活,死亡;options_colors:67C23A,F56C6C;render_default:存活" hide:"create,update"`
	DeathCause string         `json:"death_cause" gorm:"column:death_cause;comment:死亡原因" widget:"name:死因;type:input" hide:"create,update"`
	Room       *GameRoom      `json:"-" widget:"-" gorm:"foreignKey:RoomNo;references:RoomNo"`
	RoomHost   string         `json:"room_host" gorm:"-" widget:"name:房主;type:text" hide:"create,update"`
}

func (Player) TableName() string {
	return "player"
}

// ================ 玩家列表 ================

// PlayerListReq 玩家列表请求
type PlayerListReq struct {
	RoomNo            string `json:"room_no" form:"room_no" widget:"name:房间号;type:input"`
	PlayerName        string `json:"player_name" form:"player_name" widget:"name:玩家名;type:input"`
	Role              string `json:"role" form:"role" widget:"name:角色;type:select;options:村民,狼人,预言家,女巫,猎人;options_colors:67C23A,F56C6C,409EFF,9C27B0,FF9800"`
	Status            string `json:"status" form:"status" widget:"name:状态;type:select;options:存活,死亡;options_colors:67C23A,F56C6C"`
	CreatedStartTime  string `json:"created_start_time" form:"created_start_time" widget:"name:创建开始时间;type:datetime"`
	CreatedEndTime    string `json:"created_end_time" form:"created_end_time" widget:"name:创建结束时间;type:datetime"`
	query.PageSortReq `widget:"-"`
}

// PlayerList 玩家列表
func PlayerList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req PlayerListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&Player{}).Preload("Room")
	if req.RoomNo != "" {
		queryDB = queryDB.Where("room_no = ?", req.RoomNo)
	}
	if req.PlayerName != "" {
		queryDB = queryDB.Where("player_name LIKE ?", "%"+req.PlayerName+"%")
	}
	if req.Role != "" {
		queryDB = queryDB.Where("role = ?", req.Role)
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

	var players []Player
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&players).Error; err != nil {
		return err
	}

	for i := range players {
		if players[i].Room != nil {
			players[i].RoomHost = players[i].Room.HostName
		}
	}

	return resp.Table(response.TableResult{
		Items:      players,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// PlayerListTemplate 玩家管理配置
var PlayerListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "玩家管理",
		Desc:         `管理房间内的玩家，包括角色分配、存活状态等`,
		Tags:         []string{"狼人杀", "玩家管理"},
		Request:      &PlayerListReq{},
		CreateTables: []interface{}{&Player{}},
	},
	AutoCrudTable: &Player{},

	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var player Player
		if err := ctx.ShouldBindValidate(&player); err != nil {
			return nil, err
		}

		var room GameRoom
		if err := db.Where("room_no = ?", player.RoomNo).First(&room).Error; err != nil {
			return nil, fmt.Errorf("房间不存在")
		}

		if room.Status != "等待开始" {
			return nil, fmt.Errorf("游戏已开始，无法加入房间")
		}

		if room.PlayerCount >= room.MaxPlayers {
			return nil, fmt.Errorf("房间已满，无法加入")
		}

		var existCount int64
		db.Model(&Player{}).Where("room_no = ? AND deleted_at IS NULL", player.RoomNo).Count(&existCount)
		if existCount >= int64(room.MaxPlayers) {
			return nil, fmt.Errorf("房间已满，无法加入")
		}

		var existingPlayer Player
		if err := db.Where("player_name = ? AND room_no = ? AND deleted_at IS NULL", player.PlayerName, player.RoomNo).First(&existingPlayer).Error; err == nil {
			return nil, fmt.Errorf("玩家 '%s' 已在房间中", player.PlayerName)
		}

		player.PlayerID = fmt.Sprintf("%s_%s_%d", player.RoomNo, player.PlayerName, time.Now().Unix())
		player.Status = "存活"

		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&player).Error; err != nil {
				return err
			}
			return tx.Model(&GameRoom{}).Where("room_no = ?", player.RoomNo).
				Update("player_count", gorm.Expr("player_count + ?", 1)).Error
		})

		if err != nil {
			logger.Errorf(ctx, "Add player err: %v", err)
			return nil, err
		}

		return &callback.OnTableAddRowResp{Data: &player}, nil
	},

	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		var updateFields Player
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, err
		}

		updates := req.ChangedFields()
		err := db.Model(&Player{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "Update player err: %v", err)
			return nil, err
		}

		return &callback.OnTableUpdateRowResp{}, nil
	},

	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()

		err := db.Transaction(func(tx *gorm.DB) error {
			var players []Player
			if err := tx.Where("id IN ?", req.GetIds()).Find(&players).Error; err != nil {
				return err
			}

			roomNos := make([]string, 0)
			for _, player := range players {
				roomNos = append(roomNos, player.RoomNo)
			}

			tx.Where("player_id IN ?", func() []string {
				ids := make([]string, 0)
				for _, p := range players {
					ids = append(ids, p.PlayerID)
				}
				return ids
			}()).Delete(&GameRecord{})

			tx.Where("id IN ?", req.GetIds()).Delete(&Player{})

			for _, roomNo := range roomNos {
				tx.Model(&GameRoom{}).Where("room_no = ?", roomNo).
					Update("player_count", gorm.Expr("player_count - ?", 1))
			}

			return nil
		})

		if err != nil {
			logger.Errorf(ctx, "Delete player err: %v", err)
			return nil, err
		}

		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("player_list.table", PlayerList, PlayerListTemplate)
}
