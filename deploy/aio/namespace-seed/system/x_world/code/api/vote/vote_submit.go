package vote

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/statistics"
	"gorm.io/gorm"
)

// ================ 请求/响应结构 ================

// VoteSubmitReq 提交投票请求
type VoteSubmitReq struct {
	TopicID   int    `json:"topic_id" widget:"name:选择投票主题;type:select" validate:"required" callback:"OnSelectFuzzy"`
	OptionIDs []int  `json:"option_ids" widget:"name:选择投票选项;type:multiselect;depend_on:topic_id" validate:"required,min=1" callback:"OnSelectFuzzy"`
	Remark    string `json:"remark" widget:"name:投票备注;type:text_area;placeholder:请输入您的建议或说明（可选）" validate:"max=500"`
}

// VoteSubmitResp 提交投票响应
type VoteSubmitResp struct {
	Success         bool   `json:"success" widget:"name:是否成功;type:switch"`
	Message         string `json:"message" widget:"name:处理结果;type:text_area"`
	TopicTitle      string `json:"topic_title" widget:"name:投票标题;type:input"`
	SelectedOptions string `json:"selected_options" widget:"name:已选选项;type:text_area"`
	VoteTime        string `json:"vote_time" widget:"name:投票时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	IsAnonymous     bool   `json:"is_anonymous" widget:"name:是否匿名;type:switch"`
	FunctionLink    string `json:"function_link" widget:"name:查看结果;type:link;target:_blank"`
}

// ================ 辅助函数 ================

func checkCanVote(db *gorm.DB, topicID int, voterName string) error {
	var topic VoteTopic
	if err := db.Where("id = ?", topicID).First(&topic).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("投票主题不存在")
		}
		return fmt.Errorf("[系统错误]-[checkCanVote]：查询投票主题失败, topic_id: %d, voter: %s, err: %v", topicID, voterName, err)
	}

	status := getTopicStatus(topic.StartTime, topic.EndTime)
	if status != "进行中" {
		return fmt.Errorf("投票状态为 %s，无法投票", status)
	}

	var count int64
	if err := db.Model(&VoteRecord{}).Where("topic_id = ? AND voter_name = ?", topicID, voterName).Count(&count).Error; err != nil {
		return fmt.Errorf("[系统错误]-[checkCanVote]：查询投票记录失败, topic_id: %d, voter: %s, err: %v", topicID, voterName, err)
	}
	if count > 0 {
		return fmt.Errorf("您已经投过票了，每人每个主题只能投一次")
	}

	return nil
}

func calculatePercentage(voteCount int, totalVotes int) float64 {
	if totalVotes == 0 {
		return 0
	}
	percentage := float64(voteCount) * 100 / float64(totalVotes)
	return float64(int(percentage*100+0.5)) / 100
}

func updateOptionsPercentage(tx *gorm.DB, topicID int) error {
	var topic VoteTopic
	if err := tx.Where("id = ?", topicID).First(&topic).Error; err != nil {
		return fmt.Errorf("查询投票主题失败: %v", err)
	}

	var options []VoteOption
	if err := tx.Where("topic_id = ?", topicID).Find(&options).Error; err != nil {
		return fmt.Errorf("查询投票选项失败: %v", err)
	}

	if len(options) == 0 {
		return nil
	}

	var caseWhenBuilder strings.Builder
	var args []interface{}
	optionIDs := make([]int, 0, len(options))

	caseWhenBuilder.WriteString("CASE id")
	for _, option := range options {
		percentage := calculatePercentage(option.VoteCount, topic.TotalVotes)
		caseWhenBuilder.WriteString(" WHEN ? THEN ?")
		args = append(args, option.ID, percentage)
		optionIDs = append(optionIDs, option.ID)
	}
	caseWhenBuilder.WriteString(" ELSE percentage END")
	voteOption := VoteOption{}
	sql := "UPDATE " + voteOption.TableName() + " SET percentage = " + caseWhenBuilder.String() + " WHERE id IN ?"
	args = append(args, optionIDs)

	if err := tx.Model(&VoteOption{}).Exec(sql, args...).Error; err != nil {
		return fmt.Errorf("更新选项得票率失败: %v", err)
	}

	return nil
}

// ================ 模糊搜索回调 ================

// voteOnSelectFuzzyTopicForSubmit 提交投票时主题模糊搜索回调（只显示进行中的投票）
func voteOnSelectFuzzyTopicForSubmit(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[voteOnSelectFuzzyTopicForSubmit] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[voteOnSelectFuzzyTopicForSubmit]：数据库连接失败")
	}

	var topics []VoteTopic
	now := time.Now()

	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		db = db.Where("(title LIKE ? OR description LIKE ?) AND start_time <= ? AND end_time > ?",
			"%"+keyword+"%", "%"+keyword+"%", now, now).Limit(20)
	}
	db.Find(&topics)

	items := make([]*callback.SelectFuzzyItem, 0)
	for _, topic := range topics {
		status := getTopicStatus(topic.StartTime, topic.EndTime)
		items = append(items, &callback.SelectFuzzyItem{
			Value: topic.ID,
			Label: fmt.Sprintf("%s - %s", topic.Title, status),
			DisplayInfo: map[string]interface{}{
				"投票标题": topic.Title,
				"投票描述": topic.Description,
				"投票状态": status,
				"投票类型": topic.VoteType,
				"最多选择数": func() string {
					if topic.VoteType == "单选" {
						return "1个"
					}
					return fmt.Sprintf("%d个", topic.MaxSelections)
				}(),
			},
		})
	}

	maxSelections := 1
	if len(topics) > 0 {
		if topics[0].VoteType == "单选" {
			maxSelections = 1
		} else {
			maxSelections = topics[0].MaxSelections
		}
	}

	return &callback.OnSelectFuzzyResp{
		MaxSelections: maxSelections,
		Items:         items,
		Statistics: map[string]interface{}{
			"选中标题":  statistics.Value("投票标题"),
			"投票类型":  statistics.Value("投票类型"),
			"最多选择数": statistics.Value("最多选择数"),
			"投票状态":  statistics.Value("投票状态"),
		},
	}, nil
}

// voteOnSelectFuzzyOption 投票选项模糊搜索回调
func voteOnSelectFuzzyOption(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[voteOnSelectFuzzyOption] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[voteOnSelectFuzzyOption]：数据库连接失败")
	}

	var currentData VoteSubmitReq
	err := req.BindCurrentFormData(&currentData)
	if err != nil {
		return nil, fmt.Errorf("表单解析失败，请刷新选择投票主题后再重试")
	}

	if currentData.TopicID == 0 {
		return nil, fmt.Errorf("请先选择投票主题，再选择投票选项")
	}

	var topic VoteTopic
	if err := db.Where("id = ?", currentData.TopicID).First(&topic).Error; err != nil {
		return nil, fmt.Errorf("投票主题不存在")
	}

	var options []*VoteOption
	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		db = db.Where("content LIKE ?", "%"+req.Keyword()+"%").
			Where("topic_id = ?", currentData.TopicID).
			Order("id ASC").Limit(20)
	}
	db.Find(&options)

	items := make([]*callback.SelectFuzzyItem, 0)
	for _, o := range options {
		items = append(items, &callback.SelectFuzzyItem{
			Value: o.ID,
			Label: o.Content,
			DisplayInfo: map[string]interface{}{
				"选项内容": o.Content,
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
		MaxSelections: topic.MaxSelections,
		Items:         items,
		Statistics: map[string]interface{}{
			"选中选项": statistics.Value("选项内容"),
			"选项数量": statistics.Count("选项内容"),
		},
	}, nil
}

// ================ 提交投票 ================

// VoteSubmit 提交投票入口
func VoteSubmit(ctx *app.Context, resp response.Response) error {
	var req VoteSubmitReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoVoteSubmit(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoVoteSubmit 提交投票业务逻辑
func DoVoteSubmit(ctx *app.Context, req *VoteSubmitReq) (*VoteSubmitResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoVoteSubmit] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoVoteSubmit]：数据库连接失败")
	}

	userInfo := ctx.GetRequestUser()

	var topic VoteTopic
	if err := db.Where("id = ?", req.TopicID).First(&topic).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("投票主题不存在")
		}
		logger.Errorf(ctx, "[系统错误]-[DoVoteSubmit] 查询投票主题失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoVoteSubmit]：查询投票主题失败")
	}

	if err := checkCanVote(db, req.TopicID, userInfo); err != nil {
		if strings.Contains(err.Error(), "[系统错误]") {
			logger.Errorf(ctx, "[系统错误]-[DoVoteSubmit] checkCanVote 失败, req: %+v, err: %v", req, err)
		}
		return nil, err
	}

	if topic.VoteType == "单选" && len(req.OptionIDs) != 1 {
		return nil, fmt.Errorf("单选投票只能选择1个选项")
	}

	if topic.VoteType == "多选" && len(req.OptionIDs) > topic.MaxSelections {
		return nil, fmt.Errorf("多选投票最多选择%d个选项", topic.MaxSelections)
	}

	var options []*VoteOption
	if err := db.Where("id IN ? AND topic_id = ?", req.OptionIDs, req.TopicID).Find(&options).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoVoteSubmit] 查询投票选项失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoVoteSubmit]：查询投票选项失败")
	}

	if len(options) != len(req.OptionIDs) {
		return nil, fmt.Errorf("部分投票选项不存在或不属于该主题")
	}

	var selectedOptions string
	err := db.Transaction(func(tx *gorm.DB) error {
		records := make([]*VoteRecord, 0, len(options))
		optionIDs := make([]int, 0, len(options))
		for _, option := range options {
			records = append(records, &VoteRecord{
				TopicID:     req.TopicID,
				OptionID:    option.ID,
				VoterName:   userInfo,
				IsAnonymous: topic.IsAnonymous,
				Remark:      req.Remark,
			})
			optionIDs = append(optionIDs, option.ID)
			if selectedOptions != "" {
				selectedOptions += "、"
			}
			selectedOptions += option.Content
		}
		if len(records) > 0 {
			if err := tx.Create(&records).Error; err != nil {
				return fmt.Errorf("创建投票记录失败: %v", err)
			}
		}

		if len(optionIDs) > 0 {
			if err := tx.Model(&VoteOption{}).Where("id IN ?", optionIDs).
				Update("vote_count", gorm.Expr("vote_count + ?", 1)).Error; err != nil {
				return fmt.Errorf("更新选项得票数失败: %v", err)
			}
		}

		selectionCount := len(options)
		if err := tx.Model(&VoteTopic{}).Where("id = ?", req.TopicID).
			Update("total_votes", gorm.Expr("total_votes + ?", selectionCount)).Error; err != nil {
			return fmt.Errorf("更新总投票数失败: %v", err)
		}

		if err := updateOptionsPercentage(tx, req.TopicID); err != nil {
			return fmt.Errorf("更新选项得票率失败: %v", err)
		}

		return nil
	})

	if err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoVoteSubmit] 事务失败, req: %+v, topic: %+v, err: %v", req, topic, err)
		return nil, fmt.Errorf("[系统错误]-[DoVoteSubmit]：事务失败")
	}

	params := VoteResultReq{TopicID: req.TopicID}
	functionLink, _ := ctx.BuildFunctionUrlWithText("vote_result.form", params, "查看投票结果")

	return &VoteSubmitResp{
		Success:         true,
		Message:         "投票成功！",
		TopicTitle:      topic.Title,
		SelectedOptions: selectedOptions,
		VoteTime:        time.Now().Format("2006-01-02 15:04:05"),
		IsAnonymous:     topic.IsAnonymous,
		FunctionLink:    functionLink,
	}, nil
}

// VoteSubmitTemplate 提交投票配置
var VoteSubmitTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "提交投票",
		Desc:     `用户选择进行中的投票主题和选项后提交投票，生成投票记录并更新统计`,
		Tags:     []string{"投票系统", "投票参与"},
		Request:  &VoteSubmitReq{},
		Response: &VoteSubmitResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"topic_id":   voteOnSelectFuzzyTopicForSubmit,
			"option_ids": voteOnSelectFuzzyOption,
		},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("vote_submit.form", VoteSubmit, VoteSubmitTemplate)
}
