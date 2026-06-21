package vote

import (
	"errors"
	"fmt"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"gorm.io/gorm"
)

// ================ 请求/响应结构 ================

// VoteResultReq 查看投票结果请求
type VoteResultReq struct {
	TopicID int `json:"topic_id" widget:"name:选择投票主题;type:select" validate:"required" callback:"OnSelectFuzzy"`
}

// VoteOptionResultItem 投票选项统计项
type VoteOptionResultItem struct {
	Content    string  `json:"content" widget:"name:选项内容;type:input"`
	VoteCount  int     `json:"vote_count" widget:"name:得票人数;type:integer;unit:人"`
	Percentage float64 `json:"percentage" widget:"name:得票率;type:progress;min:0;max:100;unit:%"`
}

// VoteResultResp 查看投票结果响应
type VoteResultResp struct {
	Success     bool                    `json:"success" widget:"name:是否成功;type:switch"`
	Message     string                  `json:"message" widget:"name:处理结果;type:text_area"`
	TopicTitle  string                  `json:"topic_title" widget:"name:投票标题;type:input"`
	Description string                  `json:"description" widget:"name:投票描述;type:text_area"`
	VoteType    string                  `json:"vote_type" widget:"name:投票类型;type:input"`
	Status      string                  `json:"status" widget:"name:投票状态;type:select;options:未开始,进行中,已结束;options_colors:909399,409EFF,67C23A"`
	TotalVotes  int                     `json:"total_votes" widget:"name:总选择次数;type:integer;unit:次"`
	Options     []*VoteOptionResultItem `json:"options" widget:"name:投票选项统计;type:table"`
	StartTime   string                  `json:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime     string                  `json:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

// ================ 查看投票结果 ================

// VoteResult 查看投票结果入口
func VoteResult(ctx *app.Context, resp response.Response) error {
	var req VoteResultReq
	if err := ctx.ShouldBind(&req); err != nil {
		return fmt.Errorf("参数解析失败")
	}
	res, err := DoVoteResult(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoVoteResult 查看投票结果业务逻辑
func DoVoteResult(ctx *app.Context, req *VoteResultReq) (*VoteResultResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoVoteResult] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoVoteResult]：数据库连接失败")
	}

	var topic VoteTopic
	if err := db.Where("id = ?", req.TopicID).First(&topic).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("投票主题不存在")
		}
		logger.Errorf(ctx, "[系统错误]-[DoVoteResult] 查询投票主题失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoVoteResult]：查询投票主题失败")
	}

	var latestTotalVotes int
	if err := db.Model(&VoteTopic{}).Where("id = ?", req.TopicID).Select("total_votes").Scan(&latestTotalVotes).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoVoteResult] 查询总票数失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoVoteResult]：查询总票数失败")
	}
	topic.TotalVotes = latestTotalVotes

	status := getTopicStatus(topic.StartTime, topic.EndTime)
	if !topic.ShowResult && status != "已结束" {
		return nil, fmt.Errorf("该投票不允许查看实时结果，请等待投票结束")
	}

	var options []*VoteOption
	if err := db.Where("topic_id = ?", req.TopicID).Order("percentage DESC, id ASC").Find(&options).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoVoteResult] 查询投票选项失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoVoteResult]：查询投票选项失败")
	}

	optionResults := make([]*VoteOptionResultItem, 0)
	for _, option := range options {
		optionResults = append(optionResults, &VoteOptionResultItem{
			Content:    option.Content,
			VoteCount:  option.VoteCount,
			Percentage: option.Percentage,
		})
	}

	return &VoteResultResp{
		Success:     true,
		Message:     "查询成功",
		TopicTitle:  topic.Title,
		Description: topic.Description,
		VoteType:    topic.VoteType,
		Status:      status,
		TotalVotes:  topic.TotalVotes,
		Options:     optionResults,
		StartTime:   topic.StartTime.Time().Format("2006-01-02 15:04:05"),
		EndTime:     topic.EndTime.Time().Format("2006-01-02 15:04:05"),
	}, nil
}

// VoteResultTemplate 查看投票结果配置
var VoteResultTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "查看投票结果",
		Desc:     `选择投票主题后查看选项得票人数和得票率`,
		Tags:     []string{"投票系统", "结果统计"},
		Request:  &VoteResultReq{},
		Response: &VoteResultResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"topic_id": voteOnSelectFuzzyTopic,
		},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("vote_result.form", VoteResult, VoteResultTemplate)
}
