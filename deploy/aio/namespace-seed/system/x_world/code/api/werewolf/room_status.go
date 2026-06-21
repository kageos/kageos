// room_status.go
// 查看房间状态表单

package werewolf

import (
	"fmt"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// ================ 请求/响应结构 ================

// RoomStatusReq 查看房间状态请求
type RoomStatusReq struct {
	RoomNo string `json:"room_no" widget:"name:房间号;type:input" validate:"required"`
}

// RoomStatusResp 查看房间状态响应
type RoomStatusResp struct {
	Status    string `json:"status" widget:"name:状态;type:input"`
	Round     int    `json:"round" widget:"name:轮次;type:integer;unit:轮"`
	Survivors string `json:"survivors" widget:"name:存活玩家;type:text_area"`
	Result    string `json:"result" widget:"name:胜负;type:text_area"`
	WolfCount int    `json:"wolf_count" widget:"name:狼人数量;type:integer;unit:人"`
	GoodCount int    `json:"good_count" widget:"name:好人数量;type:integer;unit:人"`
	Message   string `json:"message" widget:"name:提示信息;type:text_area"`
}

// ================ 查看房间状态 ================

// RoomStatus 查看房间状态入口
func RoomStatus(ctx *app.Context, resp response.Response) error {
	var req RoomStatusReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoRoomStatus(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoRoomStatus 查看房间状态业务逻辑
func DoRoomStatus(ctx *app.Context, req *RoomStatusReq) (*RoomStatusResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoRoomStatus] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoRoomStatus]： 数据库连接失败, req: %+v", req)
	}

	var room GameRoom
	if err := db.Where("room_no = ?", req.RoomNo).First(&room).Error; err != nil {
		return nil, fmt.Errorf("房间 %s 不存在", req.RoomNo)
	}

	var players []Player
	if err := db.Where("room_no = ? AND deleted_at IS NULL", req.RoomNo).Find(&players).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoRoomStatus] 查询玩家失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoRoomStatus]： 查询玩家失败, req: %+v, err: %w", req, err)
	}

	var survivorNames []string
	var wolfCount, goodCount int
	for _, p := range players {
		if p.Status == "存活" {
			survivorNames = append(survivorNames, p.PlayerName)
			if p.Role == "狼人" {
				wolfCount++
			} else {
				goodCount++
			}
		}
	}

	survivors := strings.Join(survivorNames, "、")
	if survivors == "" {
		survivors = "无"
	}

	result := "游戏进行中"
	if room.Status == "结算" {
		if wolfCount == 0 {
			result = "好人胜利！狼人全部死亡"
		} else if wolfCount >= goodCount {
			result = "狼人胜利！狼人数量等于或超过好人数量"
		} else {
			result = "好人胜利！"
		}
	} else if room.Status == "等待开始" {
		result = "等待开始"
	} else {
		result = "游戏进行中"
	}

	return &RoomStatusResp{
		Status:    room.Status,
		Round:     room.CurrentRound,
		Survivors: survivors,
		Result:    result,
		WolfCount: wolfCount,
		GoodCount: goodCount,
		Message:   fmt.Sprintf("房间 %s 当前状态查询成功", req.RoomNo),
	}, nil
}

// RoomStatusTemplate 查看房间状态配置
var RoomStatusTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "查看房间状态",
		Desc:     `查看当前房间状态、游戏历史和结果`,
		Tags:     []string{"狼人杀", "游戏查询"},
		Request:  &RoomStatusReq{},
		Response: &RoomStatusResp{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("room_status.form", RoomStatus, RoomStatusTemplate)
}
