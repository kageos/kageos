// create_room.go
// 创建房间表单

package werewolf

import (
	"fmt"
	"time"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// ================ 请求/响应结构 ================

// CreateRoomReq 创建房间请求
type CreateRoomReq struct {
	HostName   string `json:"host_name" widget:"name:房主;type:input" validate:"required"`
	MaxPlayers int    `json:"max_players" widget:"name:人数上限;type:integer;min:4;max:12;step:1;render_default:6;unit:人"`
}

// CreateRoomResp 创建房间响应
type CreateRoomResp struct {
	RoomNo  string `json:"room_no" widget:"name:房间号;type:input"`
	Status  string `json:"status" widget:"name:状态;type:input"`
	Message string `json:"message" widget:"name:提示信息;type:text_area"`
}

// ================ 创建房间 ================

// CreateRoom 创建房间入口
func CreateRoom(ctx *app.Context, resp response.Response) error {
	var req CreateRoomReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoCreateRoom(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoCreateRoom 创建房间业务逻辑
func DoCreateRoom(ctx *app.Context, req *CreateRoomReq) (*CreateRoomResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoCreateRoom] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoCreateRoom]： 数据库连接失败, req: %+v", req)
	}

	maxPlayers := req.MaxPlayers
	if maxPlayers == 0 {
		maxPlayers = 6
	}
	if maxPlayers < 4 || maxPlayers > 12 {
		return nil, fmt.Errorf("游戏人数必须在4-12人之间")
	}

	roomNo := fmt.Sprintf("ROOM%06d", time.Now().UnixNano()/1e6%1000000)
	creator := ctx.GetRequestUser()

	room := &GameRoom{
		RoomNo:       roomNo,
		HostName:     req.HostName,
		Status:       "等待开始",
		PlayerCount:  0,
		MaxPlayers:   maxPlayers,
		CurrentRound: 0,
		Creator:      creator,
	}

	if err := db.Create(room).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoCreateRoom] 创建房间失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoCreateRoom]： 创建房间失败, req: %+v, err: %w", req, err)
	}

	return &CreateRoomResp{
		RoomNo:  roomNo,
		Status:  "等待开始",
		Message: fmt.Sprintf("房间 %s 创建成功，可容纳 %d 人", roomNo, maxPlayers),
	}, nil
}

// CreateRoomTemplate 创建房间配置
var CreateRoomTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "创建房间",
		Desc:     `创建一个新的狼人杀游戏房间`,
		Tags:     []string{"狼人杀", "房间操作"},
		Request:  &CreateRoomReq{},
		Response: &CreateRoomResp{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("create_room.form", CreateRoom, CreateRoomTemplate)
}
