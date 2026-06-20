// day_speech.go
// 白天发言表单

package werewolf

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// ================ 请求/响应结构 ================

// DaySpeechReq 白天发言请求
type DaySpeechReq struct {
	RoomNo     string `json:"room_no" widget:"name:房间号;type:input" validate:"required"`
	PlayerName string `json:"player_name" widget:"name:玩家名;type:input" validate:"required"`
	Speech     string `json:"speech" widget:"name:发言;type:text_area" validate:"required"`
}

// DaySpeechResp 白天发言响应
type DaySpeechResp struct {
	Result  string `json:"result" widget:"name:结果;type:text_area"`
	Message string `json:"message" widget:"name:提示信息;type:text_area"`
}

// ================ 白天发言 ================

// DaySpeech 白天发言入口
func DaySpeech(ctx *app.Context, resp response.Response) error {
	var req DaySpeechReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoDaySpeech(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoDaySpeech 白天发言业务逻辑
func DoDaySpeech(ctx *app.Context, req *DaySpeechReq) (*DaySpeechResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoDaySpeech] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoDaySpeech]： 数据库连接失败, req: %+v", req)
	}

	var room GameRoom
	if err := db.Where("room_no = ?", req.RoomNo).First(&room).Error; err != nil {
		return nil, fmt.Errorf("房间 %s 不存在", req.RoomNo)
	}

	if room.Status != "白天" && room.Status != "投票" {
		return nil, fmt.Errorf("当前不是白天阶段，无法发言")
	}

	// 系统发言不需要验证玩家存活状态
	if req.PlayerName != "系统" {
		var player Player
		if err := db.Where("room_no = ? AND player_name = ? AND status = ? AND deleted_at IS NULL", req.RoomNo, req.PlayerName, "存活").First(&player).Error; err != nil {
			return nil, fmt.Errorf("玩家 '%s' 不存在或已死亡", req.PlayerName)
		}
	}

	recordID := fmt.Sprintf("REC%s%s%d", req.RoomNo, req.PlayerName, time.Now().UnixNano()/1e6)
	phase := "白天发言"
	if room.Status == "投票" {
		phase = "投票"
	}

	record := &GameRecord{
		RecordID:   recordID,
		RoomNo:     req.RoomNo,
		Round:      room.CurrentRound,
		Phase:      phase,
		PlayerName: req.PlayerName,
		Content:    req.Speech,
	}

	if err := db.Create(record).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoDaySpeech] 记录发言失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoDaySpeech] 记录发言失败, req: %+v, err: %w", req, err)
	}

	return &DaySpeechResp{
		Result:  fmt.Sprintf("发言已记录：%s", req.Speech),
		Message: fmt.Sprintf("玩家 %s 发言记录成功", req.PlayerName),
	}, nil
}

// DaySpeechTemplate 白天发言配置
var DaySpeechTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "白天发言",
		Desc:     `白天阶段玩家提交发言，支持系统公告`,
		Tags:     []string{"狼人杀", "游戏操作"},
		Request:  &DaySpeechReq{},
		Response: &DaySpeechResp{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("day_speech.form", DaySpeech, DaySpeechTemplate)
}
