// day_break.go
// 天亮结算表单 - 处理夜晚到白天的过渡，自动结算死亡

package werewolf

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"gorm.io/gorm"
)

// ================ 请求/响应结构 ================

// DayBreakReq 天亮结算请求
type DayBreakReq struct {
	RoomNo string `json:"room_no" widget:"name:房间号;type:input" validate:"required"`
}

// DayBreakResp 天亮结算响应
type DayBreakResp struct {
	Result    string `json:"result" widget:"name:结果;type:text_area"`
	DeathInfo string `json:"death_info" widget:"name:死亡信息;type:text_area"`
	Survivors string `json:"survivors" widget:"name:存活玩家;type:text_area"`
	Winner    string `json:"winner" widget:"name:胜负;type:text_area"`
	Message   string `json:"message" widget:"name:提示信息;type:text_area"`
}

// ================ 天亮结算 ================

// DayBreak 天亮结算入口
func DayBreak(ctx *app.Context, resp response.Response) error {
	var req DayBreakReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoDayBreak(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoDayBreak 天亮结算业务逻辑
func DoDayBreak(ctx *app.Context, req *DayBreakReq) (*DayBreakResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoDayBreak] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoDayBreak]： 数据库连接失败, req: %+v", req)
	}

	var room GameRoom
	if err := db.Where("room_no = ?", req.RoomNo).First(&room).Error; err != nil {
		return nil, fmt.Errorf("房间 %s 不存在", req.RoomNo)
	}

	if room.Status != "夜晚" {
		return nil, fmt.Errorf("当前不是夜晚阶段，无法执行天亮结算")
	}

	// 查询本轮狼人击杀记录
	var killRecord GameRecord
	if err := db.Where("room_no = ? AND round = ? AND phase = ? AND content LIKE ?",
		req.RoomNo, room.CurrentRound, "夜晚", "%选择击杀%").Order("created_at DESC").First(&killRecord).Error; err != nil {
		// 没有击杀记录，说明平安夜
	}

	// 查询女巫救人记录
	var saveRecord GameRecord
	if err := db.Where("room_no = ? AND round = ? AND phase = ? AND content LIKE ?",
		req.RoomNo, room.CurrentRound, "夜晚", "%使用解药%").First(&saveRecord).Error; err != nil {
		// 没有救人记录
	}

	// 查询女巫毒人记录
	var poisonRecord GameRecord
	if err := db.Where("room_no = ? AND round = ? AND phase = ? AND content LIKE ?",
		req.RoomNo, room.CurrentRound, "夜晚", "%使用毒药%").First(&poisonRecord).Error; err != nil {
		// 没有毒人记录
	}

	var deathPlayerName string
	var deathCause string

	// 处理死亡逻辑
	err := db.Transaction(func(tx *gorm.DB) error {
		// 狼人击杀死亡
		if killRecord.Content != "" && saveRecord.Content == "" {
			// 从击杀记录中提取目标
			var targetName string
			fmt.Sscanf(killRecord.Content, "狼人 选择击杀 %s", &targetName)
			if targetName != "" {
				// 检查是否有女巫救人
				if saveRecord.Content == "" {
					// 执行击杀
					if err := tx.Model(&Player{}).
						Where("room_no = ? AND player_name = ? AND status = ? AND deleted_at IS NULL",
							req.RoomNo, targetName, "存活").
						Updates(map[string]interface{}{
							"status":      "死亡",
							"death_cause": "狼人击杀",
						}).Error; err != nil {
						return err
					}
					deathPlayerName = targetName
					deathCause = "狼人击杀"
				}
			}
		}

		// 女巫毒人
		if poisonRecord.Content != "" {
			var targetName string
			fmt.Sscanf(poisonRecord.Content, "女巫使用毒药毒杀了 %s", &targetName)
			if targetName != "" {
				if err := tx.Model(&Player{}).
					Where("room_no = ? AND player_name = ? AND status = ? AND deleted_at IS NULL",
						req.RoomNo, targetName, "存活").
					Updates(map[string]interface{}{
						"status":      "死亡",
						"death_cause": "女巫毒杀",
					}).Error; err != nil {
					return err
				}
				if deathPlayerName != "" {
					deathPlayerName += "、" + targetName
					deathCause += "、女巫毒杀"
				} else {
					deathPlayerName = targetName
					deathCause = "女巫毒杀"
				}
			}
		}

		// 更新房间状态为白天
		if err := tx.Model(&GameRoom{}).Where("room_no = ?", req.RoomNo).
			Update("status", "白天").Error; err != nil {
			return err
		}

		// 添加天亮公告记录
		announcement := "天亮了！"
		if deathPlayerName != "" {
			announcement = fmt.Sprintf("天亮了！昨晚 %s 倒下了。", deathPlayerName)
		} else {
			announcement = "天亮了！昨晚是平安夜。"
		}

		record := &GameRecord{
			RecordID:   fmt.Sprintf("REC%s系统%d", req.RoomNo, time.Now().Unix()),
			RoomNo:     req.RoomNo,
			Round:      room.CurrentRound,
			Phase:      "白天发言",
			PlayerName: "系统",
			Content:    announcement,
		}
		return tx.Create(record).Error
	})

	if err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoDayBreak] 天亮结算失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoDayBreak]： 天亮结算失败, req: %+v, err: %w", req, err)
	}

	// 查询存活玩家
	var survivors []Player
	if err := db.Where("room_no = ? AND status = ? AND deleted_at IS NULL", req.RoomNo, "存活").
		Find(&survivors).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoDayBreak] 查询存活玩家失败, req: %+v, err: %v", req, err)
	}

	var survivorNames []string
	var wolfCount, goodCount int
	for _, p := range survivors {
		survivorNames = append(survivorNames, p.PlayerName)
		if p.Role == "狼人" {
			wolfCount++
		} else {
			goodCount++
		}
	}

	// 检查胜负
	winner := ""
	if wolfCount == 0 {
		winner = "好人胜利！狼人全部死亡"
	} else if wolfCount >= goodCount {
		winner = "狼人胜利！狼人数量等于或超过好人数量"
		db.Model(&GameRoom{}).Where("room_no = ?", req.RoomNo).Update("status", "结算")
	}

	result := "天亮成功，进入白天阶段"
	if winner != "" {
		result = winner
	}

	deathInfo := "无人死亡（平安夜）"
	if deathPlayerName != "" {
		deathInfo = fmt.Sprintf("%s 被处决（%s）", deathPlayerName, deathCause)
	}

	return &DayBreakResp{
		Result:    result,
		DeathInfo: deathInfo,
		Survivors: fmt.Sprintf("%d人存活：%v", len(survivors), survivorNames),
		Winner:    winner,
		Message:   fmt.Sprintf("房间 %s 天亮结算完成", req.RoomNo),
	}, nil
}

// DayBreakTemplate 天亮结算配置
var DayBreakTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "天亮结算",
		Desc:     `夜晚阶段结束后执行天亮结算，处理狼人击杀、女巫用药等死亡判定，并切换到白天阶段`,
		Tags:     []string{"狼人杀", "游戏操作"},
		Request:  &DayBreakReq{},
		Response: &DayBreakResp{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("day_break.form", DayBreak, DayBreakTemplate)
}
