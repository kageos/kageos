// join_room.go
// 加入房间表单

package werewolf

import (
	"fmt"
	"time"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"gorm.io/gorm"
)

// ================ 请求/响应结构 ================

// JoinRoomReq 加入房间请求
type JoinRoomReq struct {
	RoomNo     string `json:"room_no" widget:"name:房间号;type:input" validate:"required"`
	PlayerName string `json:"player_name" widget:"name:玩家名;type:input" validate:"required"`
}

// JoinRoomResp 加入房间响应
type JoinRoomResp struct {
	Result       string `json:"result" widget:"name:结果;type:input"`
	RoleAssigned string `json:"role_assigned" widget:"name:角色分配;type:input"`
	Message      string `json:"message" widget:"name:提示信息;type:text_area"`
}

// ================ 加入房间 ================

// JoinRoom 加入房间入口
func JoinRoom(ctx *app.Context, resp response.Response) error {
	var req JoinRoomReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoJoinRoom(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoJoinRoom 加入房间业务逻辑
func DoJoinRoom(ctx *app.Context, req *JoinRoomReq) (*JoinRoomResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoJoinRoom] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoJoinRoom]： 数据库连接失败, req: %+v", req)
	}

	var room GameRoom
	if err := db.Where("room_no = ?", req.RoomNo).First(&room).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("房间 %s 不存在", req.RoomNo)
		}
		logger.Errorf(ctx, "[系统错误]-[DoJoinRoom] 查询房间失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoJoinRoom]： 查询房间失败, req: %+v, err: %w", req, err)
	}

	if room.Status != "等待开始" {
		return nil, fmt.Errorf("游戏已开始，无法加入房间")
	}

	if room.PlayerCount >= room.MaxPlayers {
		return nil, fmt.Errorf("房间已满，无法加入")
	}

	var existingPlayer Player
	if err := db.Where("player_name = ? AND room_no = ? AND deleted_at IS NULL", req.PlayerName, req.RoomNo).First(&existingPlayer).Error; err == nil {
		return nil, fmt.Errorf("玩家 '%s' 已在房间中", req.PlayerName)
	}

	playerID := fmt.Sprintf("%s_%s_%d", req.RoomNo, req.PlayerName, time.Now().UnixNano()/1e6)

	player := &Player{
		PlayerID:   playerID,
		RoomNo:     req.RoomNo,
		PlayerName: req.PlayerName,
		Status:     "存活",
	}

	var roleAssigned string
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(player).Error; err != nil {
			return err
		}
		if err := tx.Model(&GameRoom{}).Where("room_no = ?", req.RoomNo).
			Update("player_count", gorm.Expr("player_count + ?", 1)).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoJoinRoom] 加入房间失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoJoinRoom]： 加入房间失败, req: %+v, err: %w", req, err)
	}

	if room.Status == "等待开始" {
		roleAssigned = "等待游戏开始"
	} else {
		roleAssigned = player.Role
	}

	return &JoinRoomResp{
		Result:       "加入成功",
		RoleAssigned: roleAssigned,
		Message:      fmt.Sprintf("玩家 %s 成功加入房间 %s", req.PlayerName, req.RoomNo),
	}, nil
}

// JoinRoomTemplate 加入房间配置
var JoinRoomTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "加入房间",
		Desc:     `玩家加入已有房间`,
		Tags:     []string{"狼人杀", "房间操作"},
		Request:  &JoinRoomReq{},
		Response: &JoinRoomResp{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("join_room.form", JoinRoom, JoinRoomTemplate)
}
