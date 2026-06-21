// vote_settle.go
// 投票结算表单 - 投票结束后自动结算放逐结果

package werewolf

import (
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"gorm.io/gorm"
)

// ================ 请求/响应结构 ================

// VoteSettleReq 投票结算请求
type VoteSettleReq struct {
	RoomNo string `json:"room_no" widget:"name:房间号;type:input" validate:"required"`
}

// VoteSettleResp 投票结算响应
type VoteSettleResp struct {
	Result     string `json:"result" widget:"name:结果;type:text_area"`
	VoteResult string `json:"vote_result" widget:"name:投票结果;type:text_area"`
	Eliminated string `json:"eliminated" widget:"name:被放逐者;type:text_area"`
	Survivors  string `json:"survivors" widget:"name:存活玩家;type:text_area"`
	Winner     string `json:"winner" widget:"name:胜负;type:text_area"`
	Message    string `json:"message" widget:"name:提示信息;type:text_area"`
}

// ================ 投票结算 ================

// VoteSettle 投票结算入口
func VoteSettle(ctx *app.Context, resp response.Response) error {
	var req VoteSettleReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoVoteSettle(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoVoteSettle 投票结算业务逻辑
func DoVoteSettle(ctx *app.Context, req *VoteSettleReq) (*VoteSettleResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoVoteSettle] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoVoteSettle]： 数据库连接失败, req: %+v", req)
	}

	var room GameRoom
	if err := db.Where("room_no = ?", req.RoomNo).First(&room).Error; err != nil {
		return nil, fmt.Errorf("房间 %s 不存在", req.RoomNo)
	}

	if room.Status != "投票" {
		return nil, fmt.Errorf("当前不是投票阶段，无法执行投票结算")
	}

	// 查询本轮所有投票记录
	var voteRecords []GameRecord
	if err := db.Where("room_no = ? AND round = ? AND phase = ? AND content LIKE ?",
		req.RoomNo, room.CurrentRound, "投票", "投票给%").
		Find(&voteRecords).Error; err != nil {
		return nil, fmt.Errorf("查询投票记录失败")
	}

	// 统计每个玩家的得票数
	voteCountMap := make(map[string]int)
	for _, r := range voteRecords {
		target := strings.TrimPrefix(r.Content, "投票给")
		if target != r.Content {
			voteCountMap[target]++
		}
	}

	// 找出得票最多的玩家
	var eliminatedPlayer string
	var maxVotes int
	var voteDetails []string

	for player, count := range voteCountMap {
		voteDetails = append(voteDetails, fmt.Sprintf("%s:%d票", player, count))
		if count > maxVotes {
			maxVotes = count
			eliminatedPlayer = player
		}
	}

	var winner string
	var finalEliminated string
	finalVoteResult := "平票，无人出局"

	err := db.Transaction(func(tx *gorm.DB) error {
		if eliminatedPlayer != "" && maxVotes > 0 {
			// 检查是否有平票
			tieCount := 0
			for _, count := range voteCountMap {
				if count == maxVotes {
					tieCount++
				}
			}

			if tieCount == 1 {
				// 唯一的最高票，放逐该玩家
				if err := tx.Model(&Player{}).
					Where("room_no = ? AND player_name = ? AND status = ? AND deleted_at IS NULL",
						req.RoomNo, eliminatedPlayer, "存活").
					Updates(map[string]interface{}{
						"status":      "死亡",
						"death_cause": "投票放逐",
					}).Error; err != nil {
					return err
				}

				// 添加遗言记录
				var playerRole string
				tx.Model(&Player{}).Where("room_no = ? AND player_name = ?", req.RoomNo, eliminatedPlayer).
					Pluck("role", &playerRole)

				lastWords := fmt.Sprintf("投票结束！%s 被放逐！（%s）", eliminatedPlayer, playerRole)
				record := &GameRecord{
					RecordID:   fmt.Sprintf("REC%s系统%d", req.RoomNo, time.Now().Unix()),
					RoomNo:     req.RoomNo,
					Round:      room.CurrentRound,
					Phase:      "遗言",
					PlayerName: "系统",
					Content:    lastWords,
				}
				if err := tx.Create(record).Error; err != nil {
					return err
				}

				finalEliminated = eliminatedPlayer
				finalVoteResult = fmt.Sprintf("%s 被放逐（%d票）", eliminatedPlayer, maxVotes)
			} else {
				// 平票，无人出局
				lastWords := "投票结束！出现平票，无人出局，进入夜晚。"
				record := &GameRecord{
					RecordID:   fmt.Sprintf("REC%s系统%d", req.RoomNo, time.Now().Unix()),
					RoomNo:     req.RoomNo,
					Round:      room.CurrentRound,
					Phase:      "投票",
					PlayerName: "系统",
					Content:    lastWords,
				}
				if err := tx.Create(record).Error; err != nil {
					return err
				}
			}
		}

		// 查询存活玩家数量，判断胜负
		var survivors []Player
		if err := tx.Where("room_no = ? AND status = ? AND deleted_at IS NULL", req.RoomNo, "存活").
			Find(&survivors).Error; err != nil {
			return err
		}

		var wolfCount, goodCount int
		var survivorNames []string
		for _, p := range survivors {
			survivorNames = append(survivorNames, p.PlayerName)
			if p.Role == "狼人" {
				wolfCount++
			} else {
				goodCount++
			}
		}

		// 判断胜负
		if wolfCount == 0 {
			winner = "好人胜利！狼人全部死亡"
		} else if wolfCount >= goodCount {
			winner = "狼人胜利！狼人数量等于或超过好人数量"
		}

		// 如果有胜负，结算游戏
		if winner != "" {
			if err := tx.Model(&GameRoom{}).Where("room_no = ?", req.RoomNo).
				Update("status", "结算").Error; err != nil {
				return err
			}
		} else {
			// 进入下一夜
			if err := tx.Model(&GameRoom{}).Where("room_no = ?", req.RoomNo).Updates(map[string]interface{}{
				"status":        "夜晚",
				"current_round": gorm.Expr("current_round + 1"),
			}).Error; err != nil {
				return err
			}

			// 添加新夜晚公告
			var nextRound int
			tx.Model(&GameRoom{}).Where("room_no = ?", req.RoomNo).Pluck("current_round", &nextRound)
			announcement := fmt.Sprintf("第%d夜，请狼人选择击杀目标。", nextRound)
			record := &GameRecord{
				RecordID:   fmt.Sprintf("REC%s系统%d", req.RoomNo, time.Now().Unix()),
				RoomNo:     req.RoomNo,
				Round:      nextRound,
				Phase:      "夜晚",
				PlayerName: "系统",
				Content:    announcement,
			}
			if err := tx.Create(record).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoVoteSettle] 投票结算失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoVoteSettle]： 投票结算失败, req: %+v, err: %w", req, err)
	}

	// 查询最终存活玩家
	var survivors []Player
	db.Where("room_no = ? AND status = ? AND deleted_at IS NULL", req.RoomNo, "存活").
		Find(&survivors)

	var survivorNames []string
	for _, p := range survivors {
		survivorNames = append(survivorNames, p.PlayerName)
	}

	result := "投票结算完成"
	if winner != "" {
		result = winner
	}

	return &VoteSettleResp{
		Result:     result,
		VoteResult: finalVoteResult,
		Eliminated: finalEliminated,
		Survivors:  fmt.Sprintf("%d人存活：%v", len(survivors), survivorNames),
		Winner:     winner,
		Message:    fmt.Sprintf("房间 %s 投票结算完成", req.RoomNo),
	}, nil
}

// VoteSettleTemplate 投票结算配置
var VoteSettleTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "投票结算",
		Desc:     `投票阶段结束后自动结算放逐结果，判断胜负，或进入下一夜`,
		Tags:     []string{"狼人杀", "游戏操作"},
		Request:  &VoteSettleReq{},
		Response: &VoteSettleResp{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("vote_settle.form", VoteSettle, VoteSettleTemplate)
}
