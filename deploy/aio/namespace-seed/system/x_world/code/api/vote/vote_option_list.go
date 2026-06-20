package vote

import (
	"fmt"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// VoteOption 投票选项表
type VoteOption struct {
	ID         int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:选项ID;type:ID" hide:"create,update"`
	CreatedAt  types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	TopicID    int            `json:"topic_id" gorm:"column:topic_id;comment:主题ID;index" widget:"name:投票主题;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Content    string         `json:"content" gorm:"column:content;comment:选项内容" widget:"name:选项内容;type:input" validate:"required,min=1,max=100"`
	VoteCount  int            `json:"vote_count" gorm:"column:vote_count;comment:得票人数;default:0" widget:"name:得票人数;type:integer;unit:人" hide:"create,update"`
	Percentage float64        `json:"percentage" gorm:"column:percentage;comment:得票率;default:0;type:decimal(5,2)" widget:"name:得票率;type:progress;min:0;max:100;unit:%" hide:"create,update"`
	TopicTitle string         `json:"topic_title" gorm:"-" widget:"name:投票主题标题;type:text" hide:"create,update"`
}

func (VoteOption) TableName() string {
	return "vote_option"
}

// ================ 投票选项管理 ================

// VoteOptionListReq 投票选项列表请求
type VoteOptionListReq struct {
	TopicID    int    `json:"topic_id" form:"topic_id" widget:"name:投票主题;type:select" callback:"OnSelectFuzzy"`
	TopicTitle string `json:"topic_title" form:"topic_title" gorm:"-" widget:"name:投票标题;type:input"`
	Content    string `json:"content" form:"content" widget:"name:选项内容;type:input"`

	query.PageSortReq `widget:"-"`
}

// VoteOptionList 投票选项管理
func VoteOptionList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req VoteOptionListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&VoteOption{})
	if req.TopicID > 0 {
		queryDB = queryDB.Where("topic_id = ?", req.TopicID)
	}
	if req.TopicTitle != "" {
		var topicIDs []int
		if err := db.Model(&VoteTopic{}).
			Where("title LIKE ?", "%"+req.TopicTitle+"%").
			Pluck("id", &topicIDs).Error; err == nil && len(topicIDs) > 0 {
			queryDB = queryDB.Where("topic_id IN ?", topicIDs)
		} else {
			queryDB = queryDB.Where("1 = 0")
		}
	}
	if req.Content != "" {
		queryDB = queryDB.Where("content LIKE ?", "%"+req.Content+"%")
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var options []VoteOption
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&options).Error; err != nil {
		return err
	}

	// 批量获取 topicID 对应的标题
	topicIDs := make([]int, 0)
	topicIDSet := make(map[int]bool)
	for _, opt := range options {
		if !topicIDSet[opt.TopicID] {
			topicIDs = append(topicIDs, opt.TopicID)
			topicIDSet[opt.TopicID] = true
		}
	}
	topicTitles := make(map[int]string)
	if len(topicIDs) > 0 {
		var topics []VoteTopic
		db.Where("id IN ?", topicIDs).Find(&topics)
		for _, t := range topics {
			topicTitles[t.ID] = t.Title
		}
	}

	for i := range options {
		if title, ok := topicTitles[options[i].TopicID]; ok {
			options[i].TopicTitle = title
		}
	}

	return resp.Table(response.TableResult{
		Items:      options,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// voteOnSelectFuzzyTopicForOptionList 选项管理里选择主题的回调
func voteOnSelectFuzzyTopicForOptionList(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	var topics []VoteTopic
	db := ctx.GetGormDB()

	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		db = db.Where("title LIKE ? OR description LIKE ?", "%"+req.Keyword()+"%", "%"+req.Keyword()+"%").Limit(20)
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

	return &callback.OnSelectFuzzyResp{Items: items}, nil
}

// VoteOptionListTemplate 投票选项管理配置
var VoteOptionListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "投票选项管理",
		Desc:         `维护每个投票主题下的可选项和得票统计`,
		Tags:         []string{"投票系统", "选项管理"},
		Request:      &VoteOptionListReq{},
		CreateTables: []interface{}{&VoteOption{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"topic_id": voteOnSelectFuzzyTopicForOptionList,
		},
	},
	AutoCrudTable: &VoteOption{},

	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var option VoteOption
		if err := ctx.ShouldBindValidate(&option); err != nil {
			return nil, err
		}

		var topic VoteTopic
		if err := db.Where("id = ?", option.TopicID).First(&topic).Error; err != nil {
			return nil, fmt.Errorf("投票主题不存在")
		}

		status := getTopicStatus(topic.StartTime, topic.EndTime)
		if status != "未开始" {
			return nil, fmt.Errorf("投票主题 '%s' 状态为 '%s'，不能添加选项。只有未开始的投票才能添加选项", topic.Title, status)
		}

		// 去重检查：同一主题下不能存在相同内容的选项
		var existingCount int64
		db.Model(&VoteOption{}).Where("topic_id = ? AND content = ?", option.TopicID, option.Content).Count(&existingCount)
		if existingCount > 0 {
			return nil, fmt.Errorf("投票主题 '%s' 下已存在选项 '%s'，请勿重复创建", topic.Title, option.Content)
		}

		err := db.Create(&option).Error
		if err != nil {
			logger.Errorf(ctx, "Create vote option err: %v", err)
			return nil, err
		}

		return &callback.OnTableAddRowResp{Data: &option}, nil
	},

	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()

		var updateFields VoteOption
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}

		var topic VoteTopic
		if err := db.Where("id = ?", updateFields.TopicID).First(&topic).Error; err != nil {
			return &callback.OnTableUpdateRowResp{}, nil
		}

		status := getTopicStatus(topic.StartTime, topic.EndTime)
		if status != "未开始" {
			return nil, fmt.Errorf("投票主题 '%s' 状态为 '%s'，不能修改选项。只有未开始的投票才能修改选项", topic.Title, status)
		}

		updates := req.ChangedFields()
		err := db.Model(&VoteOption{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "Update vote option err: %v", err)
			return nil, err
		}

		return &callback.OnTableUpdateRowResp{}, nil
	},

	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()

		var options []VoteOption
		if err := db.Where("id IN ?", req.GetIds()).Find(&options).Error; err != nil {
			return nil, fmt.Errorf("查询投票选项失败: %v", err)
		}

		for _, option := range options {
			var topic VoteTopic
			if err := db.Where("id = ?", option.TopicID).First(&topic).Error; err != nil {
				continue
			}
			status := getTopicStatus(topic.StartTime, topic.EndTime)
			if status != "未开始" {
				return nil, fmt.Errorf("投票主题 '%s' 状态为 '%s'，不能删除选项。只有未开始的投票才能删除选项", topic.Title, status)
			}
		}

		err := db.Where("id IN ?", req.GetIds()).Delete(&VoteOption{}).Error
		if err != nil {
			logger.Errorf(ctx, "Delete vote option err: %v", err)
			return nil, err
		}

		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("vote_option_list.table", VoteOptionList, VoteOptionListTemplate)
}
