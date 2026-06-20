package vote

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/statistics"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// VoteOptionItem 投票选项子表行（仅用于创建主题时内嵌新增选项）
type VoteOptionItem struct {
	Content string `json:"content" widget:"name:选项内容;type:input;placeholder:请输入选项内容" validate:"required,min=1,max=100"`
}

// VoteTopic 投票主题表
type VoteTopic struct {
	ID              int              `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:主题ID;type:ID" hide:"create,update"`
	CreatedAt       types.Time       `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt       types.Time       `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt       gorm.DeletedAt   `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	Title           string           `json:"title" gorm:"column:title;comment:投票标题" widget:"name:投票标题;type:input" validate:"required,min=2,max=100"`
	Description     string           `json:"description" gorm:"column:description;comment:投票描述" widget:"name:投票描述;type:text_area" validate:"required,min=5,max=500"`
	VoteType        string           `json:"vote_type" gorm:"column:vote_type;comment:投票类型" widget:"name:投票类型;type:select;options:单选,多选;options_colors:409EFF,67C23A;render_default:单选" validate:"required,oneof=单选 多选"`
	MaxSelections   int              `json:"max_selections" gorm:"column:max_selections;comment:最多选择数" widget:"name:最多选择数;type:integer;min:1;max:10;step:1;render_default:1;unit:个" validate:"required_if=VoteType 多选,min=1,max=10"`
	IsAnonymous     bool             `json:"is_anonymous" gorm:"column:is_anonymous;comment:是否匿名;default:false" widget:"name:是否匿名投票;type:switch"`
	ShowResult      bool             `json:"show_result" gorm:"column:show_result;comment:是否显示实时结果;default:true" widget:"name:是否显示实时结果;type:switch;render_default:true"`
	StartTime       types.Time       `json:"start_time" gorm:"column:start_time;type:datetime;comment:开始时间;index" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required"`
	EndTime         types.Time       `json:"end_time" gorm:"column:end_time;type:datetime;comment:结束时间;index" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required,gtfield=StartTime"`
	TotalVotes      int              `json:"total_votes" gorm:"column:total_votes;comment:总选择次数;default:0" widget:"name:总选择次数;type:integer;unit:次" hide:"create,update"`
	CreatedBy       string           `json:"created_by" gorm:"column:created_by;comment:创建人" widget:"name:创建人;type:user" hide:"create,update"`
	Status          string           `json:"status" gorm:"-" widget:"name:状态;type:select;options:未开始,进行中,已结束;options_colors:909399,409EFF,67C23A" hide:"create,update"`
	Options         []VoteOptionItem `json:"options" gorm:"-" widget:"name:投票选项;type:table" hide:"list,update" validate:"required,min=2"`
	VoteActionLink  string           `json:"vote_action_link" gorm:"-" widget:"name:投票操作;type:link;target:_blank" hide:"create,update"`
	OptionsLink     string           `json:"options_link" gorm:"-" widget:"name:选项列表;type:link;target:_blank" hide:"create,update"`
	UserVoteDisplay string           `json:"user_vote_display" gorm:"-" widget:"name:是否已投;type:text" hide:"create,update"`
}

func (VoteTopic) TableName() string {
	return "vote_topic"
}

// ================ 辅助函数 ================

func getTopicStatus(startTime, endTime types.Time) string {
	now := time.Now()
	if now.Before(startTime.Time()) {
		return "未开始"
	} else if now.After(endTime.Time()) {
		return "已结束"
	}
	return "进行中"
}

// ================ 模糊搜索回调 ================

func voteOnSelectFuzzyTopic(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}

	var topics []VoteTopic
	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		db = db.Where("title LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%").Limit(20)
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
				"时间范围": fmt.Sprintf("%s - %s",
					topic.StartTime.Time().Format("2006-01-02 15:04"),
					topic.EndTime.Time().Format("2006-01-02 15:04")),
				"是否匿名": func() string {
					if topic.IsAnonymous {
						return "匿名投票"
					}
					return "实名投票"
				}(),
				"创建人": topic.CreatedBy,
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
		Items: items,
		Statistics: map[string]interface{}{
			"选中标题":  statistics.Value("投票标题"),
			"投票类型":  statistics.Value("投票类型"),
			"最多选择数": statistics.Value("最多选择数"),
			"投票状态":  statistics.Value("投票状态"),
			"是否匿名":  statistics.Value("是否匿名"),
			"创建人":   statistics.Value("创建人"),
		},
	}, nil
}

// ================ 投票主题管理 ================

// VoteTopicListReq 投票主题列表请求
type VoteTopicListReq struct {
	Title  string `json:"title" form:"title" widget:"name:投票标题;type:input"`
	Status string `json:"status" form:"status" gorm:"-" widget:"name:投票状态;type:select;options:未开始,进行中,已结束;options_colors:909399,409EFF,67C23A"`

	query.PageSortReq `widget:"-"`
}

// VoteTopicList 投票主题管理
func VoteTopicList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req VoteTopicListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	userInfo := ctx.GetRequestUser()

	queryDB := db.Model(&VoteTopic{})
	if req.Title != "" {
		queryDB = queryDB.Where("title LIKE ?", "%"+req.Title+"%")
	}

	now := time.Now()
	if req.Status != "" {
		switch req.Status {
		case "未开始":
			queryDB = queryDB.Where("start_time > ?", now)
		case "进行中":
			queryDB = queryDB.Where("start_time <= ? AND end_time > ?", now, now)
		case "已结束":
			queryDB = queryDB.Where("end_time <= ?", now)
		}
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var topics []VoteTopic
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&topics).Error; err != nil {
		return err
	}

	for i := range topics {
		topics[i].Status = getTopicStatus(topics[i].StartTime, topics[i].EndTime)

		var userVoteCount int64
		db.Model(&VoteRecord{}).Where("topic_id = ? AND voter_name = ?", topics[i].ID, userInfo).Count(&userVoteCount)
		if userVoteCount > 0 {
			topics[i].UserVoteDisplay = "已投票"
		} else {
			topics[i].UserVoteDisplay = "未投票"
		}

		if topics[i].Status == "已结束" || topics[i].Status == "未开始" {
			params := VoteResultReq{TopicID: topics[i].ID}
			topics[i].VoteActionLink, _ = ctx.BuildFunctionUrlWithText("vote_result.form", params, "查看投票结果")
		} else if topics[i].Status == "进行中" {
			if userVoteCount > 0 {
				params := VoteResultReq{TopicID: topics[i].ID}
				topics[i].VoteActionLink, _ = ctx.BuildFunctionUrlWithText("vote_result.form", params, "查看投票结果")
			} else {
				params := VoteSubmitReq{TopicID: topics[i].ID}
				topics[i].VoteActionLink, _ = ctx.BuildFunctionUrlWithText("vote_submit.form", params, "点击参与投票")
			}
		}

		params := VoteOption{}
		params.TopicID = topics[i].ID
		topics[i].OptionsLink, _ = ctx.BuildFunctionUrlWithText("vote_option_list.table", params, "查看选项列表")
	}

	return resp.Table(response.TableResult{
		Items:      topics,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// VoteTopicListTemplate 投票主题管理配置
var VoteTopicListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "投票主题管理",
		Desc:         `维护投票活动的标题、描述、规则和状态，支持创建、编辑、删除投票主题`,
		Tags:         []string{"投票系统", "主题管理"},
		Request:      &VoteTopicListReq{},
		CreateTables: []interface{}{&VoteTopic{}, &VoteOption{}, &VoteRecord{}},
	},
	AutoCrudTable: &VoteTopic{},

	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var topic VoteTopic
		if err := ctx.ShouldBindValidate(&topic); err != nil {
			return nil, err
		}

		if len(topic.Options) < 2 {
			return nil, fmt.Errorf("投票选项至少需要2个")
		}

		topic.CreatedBy = ctx.GetRequestUser()
		topic.TotalVotes = 0
		if topic.VoteType == "单选" {
			topic.MaxSelections = 1
		}

		var createdTopic *VoteTopic
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&topic).Error; err != nil {
				return err
			}
			createdTopic = &topic
			for _, opt := range topic.Options {
				option := VoteOption{
					TopicID: topic.ID,
					Content: opt.Content,
				}
				if err := tx.Create(&option).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			logger.Errorf(ctx, "Create vote topic and options err: %v", err)
			return nil, err
		}

		return &callback.OnTableAddRowResp{Data: createdTopic}, nil
	},

	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		updates := req.ChangedFields()

		if voteType, ok := updates["vote_type"].(string); ok && voteType == "单选" {
			updates["max_selections"] = 1
		}

		err := db.Model(&VoteTopic{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "Update vote topic err: %v", err)
			return nil, err
		}

		return &callback.OnTableUpdateRowResp{}, nil
	},

	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()

		err := db.Transaction(func(tx *gorm.DB) error {
			tx.Where("topic_id IN ?", req.GetIds()).Delete(&VoteRecord{})
			tx.Where("topic_id IN ?", req.GetIds()).Delete(&VoteOption{})
			tx.Where("id IN ?", req.GetIds()).Delete(&VoteTopic{})
			return nil
		})
		if err != nil {
			logger.Errorf(ctx, "Delete vote topic err: %v", err)
			return nil, err
		}

		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("vote_topic_list.table", VoteTopicList, VoteTopicListTemplate)
}
