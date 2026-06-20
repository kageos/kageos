// vote.go
// 投票表单

package werewolf

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// ================ 请求/响应结构 ================

// VoteReq 投票请求
type VoteReq struct {
	RoomNo   string `json:"room_no" widget:"name:房间号;type:input" validate:"required"`
	Voter    string `json:"voter" widget:"name:投票者;type:input" validate:"required"`
	VotedFor string `json:"voted_for" widget:"name:被投票者;type:input" validate:"required"`
}

// VoteResp 投票响应
type VoteResp struct {
	Result    string `json:"result" widget:"name:结果;type:text_area"`
	VoteCount string `json:"vote_count" widget:"name:票数统计;type:text_area"`
	Message   string `json:"message" widget:"name:提示信息;type:text_area"`
}

// ================ 投票 ================

// Vote 投票入口
func Vote(ctx *app.Context, resp response.Response) error {
	var req VoteReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoVote(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoVote 投票业务逻辑
func DoVote(ctx *app.Context, req *VoteReq) (*VoteResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoVote] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoVote]： 数据库连接失败, req: %+v", req)
	}

	var room GameRoom
	if err := db.Where("room_no = ?", req.RoomNo).First(&room).Error; err != nil {
		return nil, fmt.Errorf("房间 %s 不存在", req.RoomNo)
	}

	if room.Status != "投票" && room.Status != "白天" {
		return nil, fmt.Errorf("当前不是投票阶段，无法投票")
	}

	var voter Player
	if err := db.Where("room_no = ? AND player_name = ? AND status = ? AND deleted_at IS NULL", req.RoomNo, req.Voter, "存活").First(&voter).Error; err != nil {
		return nil, fmt.Errorf("投票者 '%s' 不存在或已死亡", req.Voter)
	}

	var votedPlayer Player
	if err := db.Where("room_no = ? AND player_name = ? AND status = ? AND deleted_at IS NULL", req.RoomNo, req.VotedFor, "存活").First(&votedPlayer).Error; err != nil {
		return nil, fmt.Errorf("被投票者 '%s' 不存在或已死亡", req.VotedFor)
	}

	var existingVote GameRecord
	if err := db.Where("room_no = ? AND round = ? AND phase = ? AND player_name = ? AND content LIKE ?",
		req.RoomNo, room.CurrentRound, "投票", req.Voter, "%投票给%").First(&existingVote).Error; err == nil {
		return nil, fmt.Errorf("玩家 %s 已经投过票了", req.Voter)
	}

	recordID := fmt.Sprintf("REC%s%s%d", req.RoomNo, req.Voter, time.Now().UnixNano()/1e6)
	record := &GameRecord{
		RecordID:   recordID,
		RoomNo:     req.RoomNo,
		Round:      room.CurrentRound,
		Phase:      "投票",
		PlayerName: req.Voter,
		Content:    fmt.Sprintf("投票给%s", req.VotedFor),
	}

	if err := db.Create(record).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoVote] 记录投票失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoVote]： 记录投票失败, req: %+v, err: %w", req, err)
	}

	var voteCounts []struct {
		Content string
		Count   int
	}
	db.Model(&GameRecord{}).
		Select("content, COUNT(*) as count").
		Where("room_no = ? AND round = ? AND phase = ?", req.RoomNo, room.CurrentRound, "投票").
		Group("content").
		Scan(&voteCounts)

	var voteCountInfo string
	for _, vc := range voteCounts {
		voteCountInfo += fmt.Sprintf("%s - %d票; ", vc.Content, vc.Count)
	}

	return &VoteResp{
		Result:    fmt.Sprintf("投票成功！%s 投票给了 %s", req.Voter, req.VotedFor),
		VoteCount: voteCountInfo,
		Message:   "投票已记录",
	}, nil
}

// VoteTemplate 投票配置
var VoteTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "投票",
		Desc:     `白天投票环节，选择要放逐的玩家`,
		Tags:     []string{"狼人杀", "游戏操作"},
		Request:  &VoteReq{},
		Response: &VoteResp{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("vote.form", Vote, VoteTemplate)
}
