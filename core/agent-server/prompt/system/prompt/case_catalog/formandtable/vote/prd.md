# 案例：投票系统（Table + Form）

## 一、项目概要

- **类型**：多表（主题 + 选项 + 记录）+ 两个 POST Form（提交投票、查看结果）。
- **GET Table**：`vote_topic_list.table`（投票主题管理）、`vote_option_list.table`（投票选项管理）、`vote_record_list.table`（投票记录查询）。
- **POST Form**：`vote_submit.form`（选择主题 + 选择选项 **multiselect；depend_on:topic_id** + 备注）、`vote_result.form`（选择主题 → 返回结果：选项得票数、得票率 table）。
- **关系**：主题 1:N 选项，主题 1:N 记录；选项选主题用 OnSelectFuzzy；提交时选主题用 OnSelectFuzzy（仅「进行中」），选选项用 **multiselect + depend_on:topic_id**（选项列表随主题变化）；主题只通过 `RichText` 展示唯一的富文本说明，选项通过 `Files` 展示候选图片。
- **状态**：主题状态不落库，按开始/结束时间**实时计算**（未开始/进行中/已结束）；列表可筛「投票状态」时在 Handler 里用时间条件过滤。
- **link**：主题列表带「选项列表」link（跳转 vote_option_list）、「投票操作」link（进行中且未投→vote_submit，已投或已结束→vote_result）。
- **适合参考**：主从多表（主题-选项-记录）、**multiselect + depend_on**、OnSelectFuzzy、link 按状态切换、时间状态计算、POST 提交与得票率更新。

---

## 二、结构化 PRD

本案例的产品经理输出样例统一维护在同目录 `prd.json`，使用 PRD v2：`project/tables/forms/charts/rules`。本 Markdown 只保留实现参考、SDK 写法和注意事项，不再承载旧 PRD 表格。

## 三、业务校验与逻辑要点

- **主题**：开始时间 < 结束时间；新增时至少 2 个选项；单选时 max_selections 固定为 1。
- **选项**：仅主题创建人可增删改；仅「未开始」的主题可增删改选项。
- **提交投票**：主题状态为「进行中」；当前用户在该主题下未投过；选项 ID 属于该主题；单选选 1 个，多选 1～max_selections；提交后写 vote_record、更新主题 total_votes、更新选项 vote_count 与 percentage。
- **主题列表筛「投票状态」**：未开始 / 进行中 / 已结束 用 start_time、end_time 与 now 比较。
- **选项/记录列表筛「投票主题」**：Request 显式声明 `topic_id`，链接跳转或下拉筛选时用 `topic_id = ?` 精准过滤。
- **记录列表筛「投票标题」「选项内容」**：用 Topic.title LIKE / Option.content LIKE 查出 id 列表，再 topic_id IN / option_id IN 过滤。

---

## 四、文件与路由

| 文件               | 说明           | 注册路由              |
|--------------------|----------------|-----------------------|
| vote_topic_list.go | 投票主题管理   | GET vote_topic_list.table  |
| vote_option_list.go| 投票选项管理   | GET vote_option_list.table |
| vote_record_list.go| 投票记录查询   | GET vote_record_list.table  |
| vote_submit.go     | 提交投票       | POST vote_submit.form     |
| vote_result.go     | 查看结果       | POST vote_result.form     |

**OnSelectFuzzyMap**：选项表 topic_id、vote_submit 主题与选项；选项 multiselect + depend_on:topic_id。

---

代码随本案例一起提供；read_doc 本案例路径（如 `/system/prompt/case_catalog/formandtable/vote`）即获得 PRD 与代码，无需再调用 read_file。


---

## 代码实现

以下为本案目录下 Go 源码，供 read_doc 时一并参考。

### vote_option_list.go

```go
package vote

import (
	"fmt"
	"time"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/statistics"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// VoteOption 投票选项表
type VoteOption struct {
	ID         int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:选项ID;type:ID" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt  types.Time          `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	DeletedBy  string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`
	TopicID    int            `json:"topic_id" gorm:"column:topic_id;comment:主题ID;index" widget:"name:投票主题ID;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Content    string         `json:"content" gorm:"column:content;comment:选项内容" widget:"name:选项内容;type:input" validate:"required"`
	Image      string         `json:"image" gorm:"column:image;type:text;comment:选项图片" widget:"name:选项图片;type:files;accept:image/*;max_count:1;thumbnail:true;list_preview:true"`
	VoteCount  int            `json:"vote_count" gorm:"column:vote_count;comment:得票人数;default:0" widget:"name:得票人数;type:integer;unit:人"`
	Percentage float64        `json:"percentage" gorm:"column:percentage;comment:得票率;default:0;type:decimal(5,2)" widget:"name:得票率%;type:progress;min:0;max:100;unit:%"`
	Topic      *VoteTopic     `json:"-" widget:"-" gorm:"foreignKey:TopicID"`
	TopicTitle string         `json:"topic_title" gorm:"-" widget:"name:投票主题;type:text" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
}

func (VoteOption) TableName() string {
	return "vote_option"
}

// ================ 投票选项管理 ================

// VoteOptionListReq 投票选项列表请求
type VoteOptionListReq struct {
	TopicID    int    `json:"topic_id" form:"topic_id" widget:"name:投票主题;type:select" callback:"OnSelectFuzzy"`
	TopicTitle string `json:"topic_title" form:"topic_title" widget:"name:投票标题关键词;type:input"`
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

	queryDB := db.Model(&VoteOption{}).Preload("Topic")
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

	for i := range options {
		if options[i].Topic != nil && options[i].Topic.ID > 0 {
			options[i].TopicTitle = options[i].Topic.Title
		}
	}

	return resp.Table(response.TableResult{
		Items:      options,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// voteOnSelectFuzzyTopicForOptionList 选项管理里选择主题的回调（只显示当前用户创建的）
func voteOnSelectFuzzyTopicForOptionList(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	var topics []VoteTopic
	db := ctx.GetGormDB()

	currentUser := ctx.GetRequestUser()
	if currentUser == "" {
		return nil, fmt.Errorf("获取用户信息失败，请重新登录")
	}

	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		db = db.Where("title LIKE ?", "%"+req.Keyword()+"%").
			Limit(20)
	}

	db = db.Where("created_by = ?", currentUser)
	db.Find(&topics)

	items := make([]*callback.SelectFuzzyItem, 0)
	for _, topic := range topics {
		status := getTopicStatus(topic.StartTime, topic.EndTime)
		items = append(items, &callback.SelectFuzzyItem{
			Value:    topic.ID,
			Label:    fmt.Sprintf("%s - %s", topic.Title, status),
			RichText: topicRichText(topic),
			DisplayInfo: map[string]interface{}{
				"投票标题": topic.Title,
				"投票状态": status,
				"投票类型": topic.VoteType,
				"最多选择": func() string {
					if topic.VoteType == "单选" {
						return "1个"
					}
					return fmt.Sprintf("%d个", topic.MaxSelections)
				}(),
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
		Items: items,
		Statistics: map[string]interface{}{
			"选中标题": statistics.Value("投票标题"),
			"投票类型": statistics.Value("投票类型"),
			"最多选择": statistics.Value("最多选择"),
			"投票状态": statistics.Value("投票状态"),
		},
	}, nil
}

// VoteOptionListTemplate 投票选项管理配置
var VoteOptionListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "投票选项管理",
		Desc:         `查看和管理投票选项，包括得票数、百分比等统计信息`,
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

		currentUser := ctx.GetRequestUser()
		if currentUser == "" {
			return nil, fmt.Errorf("获取用户信息失败，请重新登录")
		}

		var topic VoteTopic
		if err := db.Where("id = ?", option.TopicID).First(&topic).Error; err != nil {
			return nil, fmt.Errorf("投票主题不存在，无法添加选项")
		}

		if topic.CreatedBy != currentUser {
			return nil, fmt.Errorf("操作不允许：只有投票主题 '%s' 的创建人 '%s' 才能添加选项，当前用户为 '%s'", topic.Title, topic.CreatedBy, currentUser)
		}

		status := getTopicStatus(topic.StartTime, topic.EndTime)
		if status != "未开始" {
			return nil, fmt.Errorf("投票主题 '%s' 状态为 '%s'，不能添加选项。只有未开始的投票才能添加选项", topic.Title, status)
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

		currentUser := ctx.GetRequestUser()
		if currentUser == "" {
			return nil, fmt.Errorf("获取用户信息失败，请重新登录")
		}

		var currentOption VoteOption
		if err := db.Where("id = ?", req.GetId()).First(&currentOption).Error; err != nil {
			return nil, fmt.Errorf("查询待更新投票选项失败: %w", err)
		}

		topicID := currentOption.TopicID
		if req.IsFieldUpdated("topic_id") {
			if updateFields.TopicID <= 0 {
				return nil, fmt.Errorf("投票主题不能为空")
			}
			topicID = updateFields.TopicID
		}

		var topic VoteTopic
		if err := db.Where("id = ?", topicID).First(&topic).Error; err != nil {
			return nil, fmt.Errorf("投票主题不存在或查询失败: %w", err)
		}

		if topic.CreatedBy != currentUser {
			return nil, fmt.Errorf("操作不允许：只有投票主题 '%s' 的创建人 '%s' 才能修改选项，当前用户为 '%s'", topic.Title, topic.CreatedBy, currentUser)
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

		currentUser := ctx.GetRequestUser()
		if currentUser == "" {
			return nil, fmt.Errorf("获取用户信息失败，请重新登录")
		}

		var options []VoteOption
		if err := db.Preload("Topic").Where("id IN ?", req.GetIds()).Find(&options).Error; err != nil {
			return nil, fmt.Errorf("查询投票选项失败: %v", err)
		}

		for _, option := range options {
			if option.Topic == nil || option.Topic.ID == 0 {
				continue
			}

			if option.Topic.CreatedBy != currentUser {
				return nil, fmt.Errorf("操作不允许：只有投票主题 '%s' 的创建人 '%s' 才能删除选项，当前用户为 '%s'", option.Topic.Title, option.Topic.CreatedBy, currentUser)
			}

			status := getTopicStatus(option.Topic.StartTime, option.Topic.EndTime)
			if status != "未开始" {
				return nil, fmt.Errorf("投票主题 '%s' 状态为 '%s'，不能删除选项。只有未开始的投票才能删除选项", option.Topic.Title, status)
			}
		}

		err := db.Model(&VoteOption{}).Where("id IN ?", req.GetIds()).Updates(map[string]interface{}{
			"deleted_at": time.Now(),
			"deleted_by": currentUser,
		}).Error
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
```

### vote_record_list.go

```go
package vote

import (
	"fmt"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// VoteRecord 投票记录表
type VoteRecord struct {
	ID            int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:记录ID;type:ID" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt     types.Time          `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:投票时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	DeletedBy     string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`
	TopicID       int            `json:"topic_id" gorm:"column:topic_id;comment:主题ID;index" widget:"name:投票主题ID;type:select" callback:"OnSelectFuzzy" validate:"required"`
	TopicTitle    string         `json:"topic_title" gorm:"-" widget:"name:投票标题;type:input" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	OptionID      int            `json:"option_id" gorm:"column:option_id;comment:选项ID;index" widget:"name:选项ID;type:integer"`
	OptionContent string         `json:"option_content" gorm:"-" widget:"name:选项内容;type:input" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	VoterName     string         `json:"voter_name" gorm:"column:voter_name;comment:投票人" widget:"name:投票人;type:user"`
	IsAnonymous   bool           `json:"is_anonymous" gorm:"column:is_anonymous;comment:是否匿名;default:false" widget:"name:是否匿名;type:switch"`
	Remark        string         `json:"remark" gorm:"column:remark;comment:投票备注" widget:"name:投票备注;type:text_area"`
	Topic         *VoteTopic     `json:"-" widget:"-" gorm:"foreignKey:TopicID"`
	Option        *VoteOption    `json:"-" widget:"-" gorm:"foreignKey:OptionID"`
}

func (VoteRecord) TableName() string {
	return "vote_record"
}

// ================ 投票记录管理 ================

// VoteRecordListReq 投票记录列表请求
type VoteRecordListReq struct {
	TopicID       int    `json:"topic_id" form:"topic_id" widget:"name:投票主题;type:select" callback:"OnSelectFuzzy"`
	TopicTitle    string `json:"topic_title" form:"topic_title" widget:"name:投票标题;type:input"`
	OptionContent string `json:"option_content" form:"option_content" widget:"name:选项内容;type:input"`
	VoterName     string `json:"voter_name" form:"voter_name" widget:"name:投票人;type:user"`

	query.PageSortReq `widget:"-"`
}

// VoteRecordList 投票记录管理
func VoteRecordList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req VoteRecordListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&VoteRecord{})

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

	if req.OptionContent != "" {
		var optionIDs []int
		if err := db.Model(&VoteOption{}).
			Where("content LIKE ?", "%"+req.OptionContent+"%").
			Pluck("id", &optionIDs).Error; err == nil && len(optionIDs) > 0 {
			queryDB = queryDB.Where("option_id IN ?", optionIDs)
		} else {
			queryDB = queryDB.Where("1 = 0")
		}
	}
	if req.VoterName != "" {
		queryDB = queryDB.Where("voter_name = ?", req.VoterName)
	}

	queryDB = queryDB.Preload("Topic").Preload("Option")

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var records []*VoteRecord
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&records).Error; err != nil {
		return err
	}

	for i := range records {
		if records[i].Topic != nil && records[i].Topic.ID > 0 {
			records[i].TopicTitle = records[i].Topic.Title
		}
		if records[i].Option != nil && records[i].Option.ID > 0 {
			records[i].OptionContent = records[i].Option.Content
		}

		if records[i].IsAnonymous {
			records[i].VoterName = "匿名用户"
		}
	}

	return resp.Table(response.TableResult{
		Items:      records,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// VoteRecordListTemplate 投票记录管理配置
var VoteRecordListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "投票记录查询",
		Desc:         `投票记录查询管理，支持按主题、选项、投票人等条件筛选`,
		Tags:         []string{"投票系统", "记录管理"},
		Request:      &VoteRecordListReq{},
		CreateTables: []interface{}{&VoteRecord{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"topic_id": voteOnSelectFuzzyTopic,
		},
	},
	AutoCrudTable: &VoteRecord{},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("vote_record_list.table", VoteRecordList, VoteRecordListTemplate)
}
```

### vote_result.go

```go
package vote

import (
	"errors"
	"fmt"
	"time"

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

// VoteOptionResult 投票选项结果
type VoteOptionResult struct {
	Content    string  `json:"content" widget:"name:选项内容;type:input"`
	VoteCount  int     `json:"vote_count" widget:"name:得票人数;type:integer;unit:人"`
	Percentage float64 `json:"percentage" widget:"name:得票率%;type:progress;min:0;max:100;unit:%"`
}

// VoteResultResp 查看投票结果响应
type VoteResultResp struct {
	Success     bool                `json:"success" widget:"name:是否成功;type:switch"`
	Message     string              `json:"message" widget:"name:处理结果;type:text_area"`
	TopicTitle  string              `json:"topic_title" widget:"name:投票标题;type:input"`
	Content     string              `json:"content" widget:"name:主题说明;type:richtext"`
	VoteType    string              `json:"vote_type" widget:"name:投票类型;type:input"`
	Status      string              `json:"status" widget:"name:投票状态;type:select;options:未开始,进行中,已结束;options_colors:909399,409EFF,67C23A"`
	TotalVotes  int                 `json:"total_votes" widget:"name:总选择次数;type:integer;unit:次"`
	Options     []*VoteOptionResult `json:"options" widget:"name:投票选项统计;type:table"`
	StartTime   string              `json:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime     string              `json:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

// ================ 查看投票结果 ================

// VoteResult 查看投票结果入口（SDK 注册用）：解析请求 → 调 DoVoteResult → 写响应
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

// DoVoteResult 查看投票结果业务逻辑：(ctx, req) → (res, err)，便于单测与复用。
// 仅需智能体介入的错误加 [系统错误] 前缀（由 SDK 区分）；此类错误打日志时须带足上下文（req/model %+v）便于排查。
func DoVoteResult(ctx *app.Context, req *VoteResultReq) (*VoteResultResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoVoteResult] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoVoteResult]： 数据库连接失败, req: %+v", req)
	}

	var topic VoteTopic
	if err := db.Where("id = ?", req.TopicID).First(&topic).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("投票主题不存在")
		}
		// [系统错误] 需智能体介入；打足上下文便于排查
		logger.Errorf(ctx, "[系统错误]-[DoVoteResult] 查询投票主题失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoVoteResult]： 查询投票主题失败, req: %+v, err: %w", req, err)
	}

	var latestTotalVotes int
	if err := db.Model(&VoteTopic{}).Where("id = ?", req.TopicID).Select("total_votes").Scan(&latestTotalVotes).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoVoteResult] 查询总票数失败, req: %+v, topic_id: %d, err: %v", req, req.TopicID, err)
		return nil, fmt.Errorf("[系统错误]-[DoVoteResult]： 查询总票数失败, req: %+v, err: %w", req, err)
	}
	topic.TotalVotes = latestTotalVotes

	status := getTopicStatus(topic.StartTime, topic.EndTime)
	if !topic.ShowResult && status != "已结束" {
		return nil, fmt.Errorf("该投票不允许查看实时结果，请等待投票结束")
	}

	var options []*VoteOption
	if err := db.Where("topic_id = ?", req.TopicID).Order("percentage DESC, id ASC").Find(&options).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoVoteResult] 查询投票选项失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoVoteResult]： 查询投票选项失败, req: %+v, err: %w", req, err)
	}

	optionResults := make([]*VoteOptionResult, 0)
	for _, option := range options {
		optionResults = append(optionResults, &VoteOptionResult{
			Content:    option.Content,
			VoteCount:  option.VoteCount,
			Percentage: option.Percentage,
		})
	}

	return &VoteResultResp{
		Success:     true,
		Message:     "查询成功",
		TopicTitle:  topic.Title,
		Content:     topicRichText(topic),
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
		Name:     "投票结果查询",
		Desc:     `查看投票统计结果，包括各选项得票数、百分比等信息`,
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
```

### vote_submit.go

```go
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
	TopicID int `json:"topic_id" widget:"name:选择投票主题;type:select" validate:"required" callback:"OnSelectFuzzy"`
	// OptionIDs 投票选项ID列表
	// depend_on:topic_id 表示该字段依赖于 topic_id 字段
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
	IsAnonymous     bool   `json:"is_anonymous" widget:"name:投票类型;type:switch"`
	FunctionLink    string `json:"function_link" widget:"name:查看结果;type:link;target:_blank"`
}

// ================ 辅助函数 ================

// checkCanVote 检查是否可以投票（供 DoVoteSubmit 使用，不依赖 resp）。
// 数据库等异常用 [系统错误] 返回，便于区分需智能体介入的错误；业务校验（主题不存在、已投过等）不加标签。
func checkCanVote(db *gorm.DB, topicID int, voterName string) error {
	var topic VoteTopic
	if err := db.Where("id = ?", topicID).First(&topic).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("投票主题不存在")
		}
		return fmt.Errorf("[系统错误]-[checkCanVote]： 查询投票主题失败, topic_id: %d, voter: %s, err: %v", topicID, voterName, err)
	}

	status := getTopicStatus(topic.StartTime, topic.EndTime)
	if status != "进行中" {
		return fmt.Errorf("投票状态为 %s，无法投票", status)
	}

	var count int64
	if err := db.Model(&VoteRecord{}).Where("topic_id = ? AND voter_name = ?", topicID, voterName).Count(&count).Error; err != nil {
		return fmt.Errorf("[系统错误]-[checkCanVote]： 查询投票记录失败, topic_id: %d, voter: %s, err: %v", topicID, voterName, err)
	}
	if count > 0 {
		return fmt.Errorf("您已经投过票了，每人每个主题只能投一次")
	}

	return nil
}

// calculatePercentage 计算投票百分比
func calculatePercentage(voteCount int, totalVotes int) float64 {
	if totalVotes == 0 {
		return 0
	}
	percentage := float64(voteCount) * 100 / float64(totalVotes)
	return float64(int(percentage*100+0.5)) / 100
}

// updateOptionsPercentage 更新指定主题下所有选项的得票率
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

// voteOnSelectFuzzyTopicForSubmit 投票主题模糊搜索回调（只显示进行中的投票）
func voteOnSelectFuzzyTopicForSubmit(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[voteOnSelectFuzzyTopicForSubmit] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[voteOnSelectFuzzyTopicForSubmit]： 数据库连接失败, req: %+v", req)
	}

	var topics []VoteTopic
	now := time.Now()

	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		db = db.Where("title LIKE ? AND start_time <= ? AND end_time > ?",
			"%"+keyword+"%", now, now).
			Limit(20)
	}

	db.Find(&topics)

	items := make([]*callback.SelectFuzzyItem, 0)
	for _, topic := range topics {
		status := getTopicStatus(topic.StartTime, topic.EndTime)
		items = append(items, &callback.SelectFuzzyItem{
			Value:    topic.ID,
			Label:    fmt.Sprintf("%s - %s", topic.Title, status),
			RichText: topicRichText(topic),
			DisplayInfo: map[string]interface{}{
				"投票标题": topic.Title,
				"投票状态": status,
				"投票类型": topic.VoteType,
				"最多选择": func() string {
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
			"选中标题": statistics.Value("投票标题"),
			"投票类型": statistics.Value("投票类型"),
			"最多选择": statistics.Value("最多选择"),
			"投票状态": statistics.Value("投票状态"),
		},
	}, nil
}

// voteOnSelectFuzzyOption 投票选项模糊搜索回调
// 选项依赖「投票主题」：需在回调中通过 req.BindCurrentFormData 获取当前表单已填数据（含 TopicID），
// 因此请求结构体里要把依赖字段放上面（TopicID 在上、OptionIDs 在下），顺序很重要。详见 SDK 文档「OnSelectFuzzy → 回调中获取当前表单数据」。
func voteOnSelectFuzzyOption(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[voteOnSelectFuzzyOption] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[voteOnSelectFuzzyOption]： 数据库连接失败, req: %+v", req)
	}

	var currentData VoteSubmitReq

	// 在回调中获取当前用户已填写的表单数据；依赖的字段需放在上面先填写（如先选投票主题再选选项），顺序很重要
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
			Order("id ASC").
			Limit(20)
	}
	db.Find(&options)

	items := make([]*callback.SelectFuzzyItem, 0)
	for _, o := range options {
		items = append(items, &callback.SelectFuzzyItem{
			Value: o.ID,
			Label: o.Content,
			Files: o.Image,
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

// VoteSubmit 提交投票入口（SDK 注册用）：解析请求 → 调 DoVoteSubmit → 写响应
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

// DoVoteSubmit 提交投票业务逻辑：(ctx, req) → (res, err)，便于单测与复用。
// 仅需智能体介入的错误加 [系统错误] 前缀（由 SDK 区分）；此类错误打日志时须带足上下文（req/model %+v）便于排查。
func DoVoteSubmit(ctx *app.Context, req *VoteSubmitReq) (*VoteSubmitResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoVoteSubmit] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoVoteSubmit]： 数据库连接失败, req: %+v", req)
	}

	userInfo := ctx.GetRequestUser()

	var topic VoteTopic
	if err := db.Where("id = ?", req.TopicID).First(&topic).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("投票主题不存在")
		}
		logger.Errorf(ctx, "[系统错误]-[DoVoteSubmit] 查询投票主题失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoVoteSubmit]： 查询投票主题失败, req: %+v, err: %w", req, err)
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
		return nil, fmt.Errorf("[系统错误]-[DoVoteSubmit]： 查询投票选项失败, req: %+v, err: %w", req, err)
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
		// [系统错误] 需智能体介入；打足上下文（req、topic）便于排查
		logger.Errorf(ctx, "[系统错误]-[DoVoteSubmit] 事务失败, req: %+v, topic: %+v, err: %v", req, topic, err)
		return nil, fmt.Errorf("[系统错误]-[DoVoteSubmit]： 事务失败, req: %+v, topic: %+v, err: %w", req, topic, err)
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
		Name:     "参与投票",
		Desc:     `选择投票主题和选项，提交投票，支持单选/多选，防止重复投票`,
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
```

### vote_topic_list.go

```go
//<文件名>vote_topic_list.go</文件名>
// 投票主题管理：数据模型、列表 Handler、Template，以及主题模糊搜索回调

package vote

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/statistics"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// VoteTopic 投票主题表
type VoteTopic struct {
	ID          int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:主题ID;type:ID" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt   types.Time          `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	UpdatedAt   types.Time          `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	DeletedBy   string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`
	Title       string         `json:"title" gorm:"column:title;comment:投票标题" widget:"name:投票标题;type:input" validate:"required,min=2,max=100"`
	Content     string         `json:"content" gorm:"column:content;type:text;comment:投票主题说明" widget:"name:主题说明;type:richtext;height:420" validate:"required,min=1"`
	LegacyDescription string   `json:"-" gorm:"column:description;type:text;comment:旧投票描述，仅兼容历史数据" widget:"-"`
	// select 须配 options_colors，与 options 一一对应，前端用颜色区分选项
	VoteType        string           `json:"vote_type" gorm:"column:vote_type;comment:投票类型" widget:"name:投票类型;type:select;options:单选,多选;options_colors:409EFF,67C23A;render_default:单选" validate:"required,oneof=单选 多选"`
	// required_if 不只是后端校验；前端也会按条件动态处理：
	// 当 VoteType=多选 时，显示 MaxSelections 且标记为必填；否则隐藏该字段。
	// 同类场景还可用 required_unless、required_with、required_without、excluded_* 等规则，详见 SDK 文档的 validate 标签说明。
	MaxSelections   int              `json:"max_selections" gorm:"column:max_selections;comment:最多选择数" widget:"name:最多选择数;type:integer;unit:个;render_default:1" validate:"required_if=VoteType 多选,min=1,max=10"`
	IsAnonymous     bool             `json:"is_anonymous" gorm:"column:is_anonymous;comment:是否匿名;default:false" widget:"name:是否匿名投票;type:switch"`
	ShowResult      bool             `json:"show_result" gorm:"column:show_result;comment:是否显示结果;default:true" widget:"name:是否显示实时结果;type:switch"`
	StartTime       types.Time            `json:"start_time" gorm:"column:start_time;type:datetime;comment:开始时间;index" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required"`
	EndTime         types.Time            `json:"end_time" gorm:"column:end_time;type:datetime;comment:结束时间;index" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required,gtfield=StartTime"`
	Options         []VoteOptionItem `json:"options" gorm:"-" widget:"name:投票选项;type:table" hide:"list,update" validate:"required,min=2"` // 前端仅在新增表单展示，列表和编辑不展示。
	OptionsLink     string           `json:"options_link" gorm:"-" widget:"name:选项列表;type:link;target:_blank" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	Status          string           `json:"status" gorm:"-" widget:"name:状态;type:select;options:未开始,进行中,已结束;options_colors:909399,409EFF,67C23A" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	TotalVotes      int              `json:"total_votes" gorm:"column:total_votes;comment:总选择次数;default:0" widget:"name:总选择次数;type:integer;unit:次"`
	CreatedBy        string           `json:"created_by" gorm:"column:created_by;comment:创建人" widget:"name:创建人;type:user" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	VoteActionLink  string           `json:"vote_action_link" gorm:"-" widget:"name:投票操作;type:link;target:_blank" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	UserVoteRecords []*VoteRecord    `json:"-" widget:"-" gorm:"foreignKey:TopicID"`
}

func (VoteTopic) TableName() string {
	return "vote_topic"
}

// VoteOptionItem 投票选项项（用于 list/table 组件）
type VoteOptionItem struct {
	Content string `json:"content" widget:"name:选项内容;type:input" validate:"required,min=1,max=100"`
	Image   string `json:"image" widget:"name:选项图片;type:files;accept:image/*;max_count:1;thumbnail:true;list_preview:true"`
}

// ================ 辅助函数 ================

// getTopicStatus 获取投票状态（计算属性）
func getTopicStatus(startTime, endTime types.Time) string {
	now := time.Now()
	if now.Before(startTime.Time()) {
		return "未开始"
	} else if now.After(endTime.Time()) {
		return "已结束"
	}
	return "进行中"
}

func topicRichText(topic VoteTopic) string {
	if strings.TrimSpace(topic.Content) != "" {
		return topic.Content
	}
	legacy := strings.TrimSpace(topic.LegacyDescription)
	if legacy == "" {
		return ""
	}
	return "<p>" + html.EscapeString(legacy) + "</p>"
}

// ================ 模糊搜索回调 ================

// voteOnSelectFuzzyTopic 投票主题模糊搜索回调
func voteOnSelectFuzzyTopic(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[voteOnSelectFuzzyTopic] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[voteOnSelectFuzzyTopic]：数据库连接失败")
	}

	var topics []VoteTopic
	queryDB := db.Model(&VoteTopic{})

	if req.IsByValue() {
		queryDB = queryDB.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		queryDB = queryDB.Where("id in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		queryDB = queryDB.Where("title LIKE ?", "%"+keyword+"%").
			Limit(20)
	}

	if err := queryDB.Order("id DESC").Find(&topics).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[voteOnSelectFuzzyTopic] 查询投票主题失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[voteOnSelectFuzzyTopic]：查询投票主题失败，请确认应用数据库迁移已完成: %w", err)
	}

	items := make([]*callback.SelectFuzzyItem, 0)
	for _, topic := range topics {
		status := getTopicStatus(topic.StartTime, topic.EndTime)
		items = append(items, &callback.SelectFuzzyItem{
			Value:    topic.ID,
			Label:    fmt.Sprintf("%s - %s", topic.Title, status),
			RichText: topicRichText(topic),
			DisplayInfo: map[string]interface{}{
				"投票标题": topic.Title,
				"投票状态": status,
				"投票类型": topic.VoteType,
				"最多选择": func() string {
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
		MaxSelections: 1,
		Statistics: map[string]interface{}{
			"选中标题": statistics.Value("投票标题"),
			"投票类型": statistics.Value("投票类型"),
			"最多选择": statistics.Value("最多选择"),
			"投票状态": statistics.Value("投票状态"),
			"是否匿名": statistics.Value("是否匿名"),
			"创建人":  statistics.Value("创建人"),
		},
		Items: items,
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

	queryDB := db.Model(&VoteTopic{}).Preload("UserVoteRecords", "voter_name = ?", userInfo)
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
		topics[i].Content = topicRichText(topics[i])

		hasUserVoted := len(topics[i].UserVoteRecords) > 0

		if topics[i].Status == "已结束" || topics[i].Status == "未开始" {
			params := VoteResultReq{TopicID: topics[i].ID}
			topics[i].VoteActionLink, _ = ctx.BuildFunctionUrlWithText("vote_result.form", params, "查看投票结果")
		} else if topics[i].Status == "进行中" {
			if hasUserVoted {
				params := VoteResultReq{TopicID: topics[i].ID}
				topics[i].VoteActionLink, _ = ctx.BuildFunctionUrlWithText("vote_result.form", params, "查看投票结果")
			} else {
				params := VoteSubmitReq{TopicID: topics[i].ID}
				topics[i].VoteActionLink, _ = ctx.BuildFunctionUrlWithText("vote_submit.form", params, "点击参与投票")
			}
		}

		params := VoteOption{
			TopicID: topics[i].ID,
		}
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
		Desc:         `投票主题的增删改查管理，支持创建投票、设置选项、时间控制`,
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

		topic.CreatedBy = ctx.GetRequestUser()
		topic.TotalVotes = 0

		if topic.VoteType == "单选" {
			topic.MaxSelections = 1
		}

		if len(topic.Options) < 2 {
			return nil, fmt.Errorf("投票 %s 至少需要2个选项", topic.Title)
		}

		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&topic).Error; err != nil {
				return fmt.Errorf("创建投票主题失败: %v", err)
			}

			optionList := make([]*VoteOption, 0, len(topic.Options))
			for _, opt := range topic.Options {
				optionList = append(optionList, &VoteOption{
					TopicID:    topic.ID,
					Content:    opt.Content,
					Image:      opt.Image,
					VoteCount:  0,
					Percentage: 0,
				})
			}
			if len(optionList) > 0 {
				if err := tx.Create(&optionList).Error; err != nil {
					return fmt.Errorf("创建投票选项失败: %v", err)
				}
			}
			return nil
		})

		if err != nil {
			logger.Errorf(ctx, "Create vote topic err: %v", err)
			return nil, err
		}

		return &callback.OnTableAddRowResp{Data: &topic}, nil
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
		deletedBy := ctx.GetRequestUser()
		if deletedBy == "" {
			return nil, fmt.Errorf("获取用户信息失败，请重新登录")
		}
		deletedAt := time.Now()
		updates := map[string]interface{}{
			"deleted_at": deletedAt,
			"deleted_by": deletedBy,
		}

		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&VoteRecord{}).Where("topic_id IN ?", req.GetIds()).Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.Model(&VoteOption{}).Where("topic_id IN ?", req.GetIds()).Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.Model(&VoteTopic{}).Where("id IN ?", req.GetIds()).Updates(updates).Error; err != nil {
				return err
			}
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
```
