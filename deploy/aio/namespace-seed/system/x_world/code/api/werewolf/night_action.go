// night_action.go
// 夜间行动表单

package werewolf

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// ================ 请求/响应结构 ================

// NightActionReq 夜间行动请求
type NightActionReq struct {
	RoomNo string `json:"room_no" widget:"name:房间号;type:input" validate:"required"`
	Role   string `json:"role" widget:"name:角色;type:select;options:狼人,预言家,女巫;options_colors:F56C6C,409EFF,9C27B0" validate:"required"`
	Target string `json:"target" widget:"name:目标;type:input" validate:"required"`
	Action string `json:"action" widget:"name:行动;type:select;options:击杀,查验,救人,毒人;options_colors:F56C6C,409EFF,67C23A,F56C6C" validate:"required"`
}

// NightActionResp 夜间行动响应
type NightActionResp struct {
	Result      string `json:"result" widget:"name:结果;type:text_area"`
	CheckResult string `json:"check_result" widget:"name:查验结果;type:text_area"`
	Message     string `json:"message" widget:"name:提示信息;type:text_area"`
}

// ================ 夜间行动 ================

// NightAction 夜间行动入口
func NightAction(ctx *app.Context, resp response.Response) error {
	var req NightActionReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoNightAction(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoNightAction 夜间行动业务逻辑
func DoNightAction(ctx *app.Context, req *NightActionReq) (*NightActionResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoNightAction] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoNightAction]： 数据库连接失败, req: %+v", req)
	}

	var room GameRoom
	if err := db.Where("room_no = ?", req.RoomNo).First(&room).Error; err != nil {
		return nil, fmt.Errorf("房间 %s 不存在", req.RoomNo)
	}

	if room.Status != "夜晚" {
		return nil, fmt.Errorf("当前不是夜晚阶段，无法执行夜间行动")
	}

	var targetPlayer Player
	if err := db.Where("room_no = ? AND player_name = ? AND status = ? AND deleted_at IS NULL", req.RoomNo, req.Target, "存活").First(&targetPlayer).Error; err != nil {
		return nil, fmt.Errorf("目标玩家 '%s' 不存在或已死亡", req.Target)
	}

	var executorPlayer Player
	if err := db.Where("room_no = ? AND role = ? AND status = ? AND deleted_at IS NULL", req.RoomNo, req.Role, "存活").First(&executorPlayer).Error; err != nil {
		return nil, fmt.Errorf("当前没有存活的 %s", req.Role)
	}

	recordID := fmt.Sprintf("REC%s%s%d", req.RoomNo, req.Role, time.Now().UnixNano()/1e6)

	var recordContent string
	var resultMsg string
	var checkResult string

	switch req.Action {
	case "击杀":
		recordContent = fmt.Sprintf("%s 选择击杀 %s", req.Role, req.Target)
		resultMsg = fmt.Sprintf("狼人选择了 %s 作为击杀目标", req.Target)
	case "查验":
		var targetRole string
		if targetPlayer.Role == "" {
			targetRole = "未分配"
		} else {
			targetRole = targetPlayer.Role
		}
		isWolf := (targetRole == "狼人")
		if isWolf {
			checkResult = fmt.Sprintf("%s 是狼人！", req.Target)
		} else {
			checkResult = fmt.Sprintf("%s 是好人", req.Target)
		}
		recordContent = fmt.Sprintf("%s 查验 %s，结果：%s", req.Role, req.Target, checkResult)
		resultMsg = fmt.Sprintf("查验结果：%s", checkResult)
	case "救人":
		recordContent = fmt.Sprintf("女巫使用解药救活了 %s", req.Target)
		resultMsg = fmt.Sprintf("女巫使用解药，今晚不会有人死亡")
	case "毒人":
		recordContent = fmt.Sprintf("女巫使用毒药毒杀了 %s", req.Target)
		resultMsg = fmt.Sprintf("女巫使用毒药，%s 被毒杀", req.Target)
	default:
		return nil, fmt.Errorf("无效的行动类型: %s", req.Action)
	}

	record := &GameRecord{
		RecordID:   recordID,
		RoomNo:     req.RoomNo,
		Round:      room.CurrentRound,
		Phase:      "夜晚",
		PlayerName: req.Role,
		Content:    recordContent,
	}

	if err := db.Create(record).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoNightAction] 记录夜间行动失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoNightAction]： 记录夜间行动失败, req: %+v, err: %w", req, err)
	}

	return &NightActionResp{
		Result:      resultMsg,
		CheckResult: checkResult,
		Message:     fmt.Sprintf("夜间行动记录成功"),
	}, nil
}

// NightActionTemplate 夜间行动配置
var NightActionTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "夜间行动",
		Desc:     `夜晚阶段执行角色行动：狼人选择击杀目标、预言家查验身份、女巫用药`,
		Tags:     []string{"狼人杀", "游戏操作"},
		Request:  &NightActionReq{},
		Response: &NightActionResp{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("night_action.form", NightAction, NightActionTemplate)
}
