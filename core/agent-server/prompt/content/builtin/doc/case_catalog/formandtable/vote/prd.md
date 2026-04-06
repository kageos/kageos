# 案例：投票系统（Table + Form）

## 一、项目概要

- **类型**：多表（主题 + 选项 + 记录）+ 两个 POST Form（提交投票、查看结果）。
- **GET Table**：`vote_topic_list.table`（投票主题管理）、`vote_option_list.table`（投票选项管理）、`vote_record_list.table`（投票记录查询）。
- **POST Form**：`vote_submit.form`（选择主题 + 选择选项 **multiselect；depend_on:topic_id** + 备注）、`vote_result.form`（选择主题 → 返回结果：选项得票数、得票率 table）。
- **关系**：主题 1:N 选项，主题 1:N 记录；选项选主题用 OnSelectFuzzy；提交时选主题用 OnSelectFuzzy（仅「进行中」），选选项用 **multiselect + depend_on:topic_id**（选项列表随主题变化）。
- **状态**：主题状态不落库，按开始/结束时间**实时计算**（未开始/进行中/已结束）；列表可筛「投票状态」时在 Handler 里用时间条件过滤。
- **link**：主题列表带「选项列表」link（跳转 vote_option_list）、「投票操作」link（进行中且未投→vote_submit，已投或已结束→vote_result）。
- **适合参考**：主从多表（主题-选项-记录）、**multiselect + depend_on**、OnSelectFuzzy、link 按状态切换、时间状态计算、POST 提交与得票率更新。

---

## 二、PRD 要点（表格格式）

### 1. 主题表（vote_topic_list）

**表单字段（新增/编辑）**

| 字段           | 类型     | 必填 | 默认值 | 说明 |
|----------------|----------|------|--------|------|
| 投票标题       | 文本输入 | ✓   | —      | 2–100 字 |
| 投票描述       | 多行文本 | ✓   | —      | 5–500 字 |
| 投票类型       | 下拉选择 | ✓   | 单选   | 单选/多选 |
| 最多选择数     | 数字输入 | ✓   | 1      | 多选时 1–10，单选时固定 1 |
| 是否匿名投票   | 开关     | ✗   | 否     | — |
| 是否显示实时结果 | 开关   | ✗   | 是     | — |
| 开始时间       | 时间选择 | ✓   | —      | — |
| 结束时间       | 时间选择 | ✓   | —      | 必须晚于开始时间 |
| 投票选项       | 表格     | ✓   | —      | type:table，至少 2 条；每行「选项内容」input |
| 详细内容       | 富文本   | ✗   | —      | type:richtext |
| 创建人         | 用户选择 | ✗   | —      | 只读，可自动带当前用户 |

**列表模式**（可筛：投票状态；状态、总选择次数为仅列表展示/计算）

| ID | 创建时间 | 更新时间 | 创建人 | 投票标题 | 投票描述 | 投票类型 | 最多选择 | 开始时间 | 结束时间 | 状态 | 总选择次数 | 操作 |
|----|----------|----------|--------|----------|----------|----------|----------|----------|----------|------|------------|------|
| 1 | 2025-01-18 09:00 | 2025-01-18 09:00 | 张三 | 年度优秀员工评选 | — | 单选 | 1 | 2025-01-20 00:00 | 2025-01-25 23:59 | 进行中 | 42 | 删除、查看选项列表、点击参与投票 |
| 2 | 2025-01-10 14:00 | 2025-01-10 14:00 | 李四 | 团建地点投票 | — | 多选 | 3 | 2025-01-08 00:00 | 2025-01-12 23:59 | 已结束 | 28 | 删除、查看选项列表、查看投票结果 |

**说明**：状态按开始/结束时间计算（未开始/进行中/已结束）。**操作列**统一在列表右侧，包含删除与 link：查看选项列表（跳转 vote_option_list）、投票操作按状态为「点击参与投票」（vote_submit）或「查看投票结果」（vote_result）。

---

### 2. 选项表（vote_option_list）

**表单字段（新增/编辑）**

| 字段     | 类型     | 必填 | 默认值 | 说明 |
|----------|----------|------|--------|------|
| 投票主题 | 下拉选择 | ✓   | —      | OnSelectFuzzy，可选仅当前用户创建的主题 |
| 选项内容 | 文本输入 | ✓   | —      | 1–100 字 |

**列表模式**（得票人数、得票率% 为仅列表展示、后端计算）

| ID | 创建时间 | 更新时间 | 创建人 | 投票主题 | 选项内容 | 得票人数 | 得票率% |
|----|----------|----------|--------|----------|----------|----------|---------|
| 1 | 2025-01-18 09:05 | 2025-01-18 09:05 | — | 年度优秀员工评选 | 小王 | 15 | 35.7 |
| 2 | 2025-01-18 09:05 | 2025-01-18 09:05 | — | 年度优秀员工评选 | 小刘 | 12 | 28.6 |

**说明**：新增/编辑选项时校验：只有主题创建人可增改；只有「未开始」的主题可增改选项。

---

### 3. 记录表（vote_record_list）

仅列表查询，无可编辑表单（记录由 vote_submit 产生）。

**列表模式**（可筛：投票标题、选项内容）

| 投票时间 | 投票标题 | 选项内容 | 投票人 | 是否匿名 | 投票备注 |
|----------|----------|----------|--------|----------|----------|
| 2025-01-21 10:30 | 年度优秀员工评选 | 小王 | 赵六 | 否 | — |

---

### 4. 提交投票 Form（vote_submit.form，POST）

**请求**：选择投票主题（OnSelectFuzzy，仅「进行中」）、选择投票选项（multiselect + depend_on:topic_id）、投票备注（可选）。

**响应**：是否成功、处理结果、投票标题、已选选项、投票时间、查看结果 link。

**业务校验**：主题存在且状态为「进行中」；每人每个主题只能投一次；选项属于该主题；单选选 1 个，多选不超过 max_selections。

---

### 5. 查看结果 Form（vote_result.form，POST）

**请求**：选择投票主题（OnSelectFuzzy）。

**响应**：投票标题、描述、类型、状态、总选择次数、投票选项统计（选项内容、得票人数、得票率%）、开始/结束时间。

**说明**：若主题设置「不显示实时结果」，则仅「已结束」后可查看；否则进行中、已结束均可查看。

---

## 三、业务校验与逻辑要点

- **主题**：开始时间 < 结束时间；新增时至少 2 个选项；单选时 max_selections 固定为 1。
- **选项**：仅主题创建人可增删改；仅「未开始」的主题可增删改选项。
- **提交投票**：主题状态为「进行中」；当前用户在该主题下未投过；选项 ID 属于该主题；单选选 1 个，多选 1～max_selections；提交后写 vote_record、更新主题 total_votes、更新选项 vote_count 与 percentage。
- **主题列表筛「投票状态」**：未开始 / 进行中 / 已结束 用 start_time、end_time 与 now 比较。
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

代码随本案例一起提供；read_doc 本案例路径（如 `/builtin/doc/case_catalog/formandtable/vote`）即获得 PRD 与代码，无需再调用 read_go_file。


---

## 代码实现

以下为本案目录下 Go 源码，供 read_doc 时一并参考。

### vote_option_list.go

```go
package vote

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/statistics"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// VoteOption 投票选项表
type VoteOption struct {
	ID         int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:选项ID;type:ID" permission:"read" search:"eq"`
	CreatedAt  int64          `json:"created_at" gorm:"autoCreateTime:milli;column:created_at" widget:"name:创建时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" permission:"read"`
	DeletedAt  gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	TopicID    int            `json:"topic_id" gorm:"column:topic_id;comment:主题ID;index" widget:"name:投票主题ID;type:select" search:"in" validate:"required" callback:"OnSelectFuzzy"`
	Content    string         `json:"content" gorm:"column:content;comment:选项内容" widget:"name:选项内容;type:input" search:"like" validate:"required"`
	VoteCount  int            `json:"vote_count" gorm:"column:vote_count;comment:得票人数;default:0" widget:"name:得票人数;type:number;unit:人" search:"gte,lte" permission:"read"`
	Percentage float64        `json:"percentage" gorm:"column:percentage;comment:得票率;default:0;type:decimal(5,2)" widget:"name:得票率%;type:progress;min:0;max:100;unit:%" search:"gte,lte" permission:"read"`
	Topic      *VoteTopic     `json:"-" widget:"-" gorm:"foreignKey:TopicID"`
	TopicTitle string         `json:"topic_title" gorm:"-" widget:"name:投票主题;type:text" permission:"read"`
}

func (VoteOption) TableName() string {
	return "vote_option"
}

// ================ 投票选项管理 ================

// VoteOptionListReq 投票选项列表请求
type VoteOptionListReq struct {
	query.SearchFilterPageReq `widget:"-"`
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

	var options []VoteOption
	builder := resp.Table(&options).AutoSearchFilterPaged(queryDB, &VoteOption{}, &req.SearchFilterPageReq)

	if err := builder.Build(); err != nil {
		return err
	}

	for i := range options {
		if options[i].Topic != nil && options[i].Topic.ID > 0 {
			options[i].TopicTitle = options[i].Topic.Title
		}
	}

	return nil
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
		db = db.Where("title LIKE ? OR description LIKE ?", "%"+req.Keyword()+"%", "%"+req.Keyword()+"%").
			Limit(20)
	}

	db = db.Where("create_by = ?", currentUser)
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
			"投票描述": statistics.Value("投票描述"),
		},
	}, nil
}

// VoteOptionListTemplate 投票选项管理配置
var VoteOptionListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "投票选项管理",
		Desc:         "查看和管理投票选项，包括得票数、百分比等统计信息",
		Tags:         []string{"投票系统", "选项管理"},
		Request:      &VoteOptionListReq{},
		Response:     query.PaginatedTable[[]VoteOption]{},
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

		if topic.CreateBy != currentUser {
			return nil, fmt.Errorf("权限不足：只有投票主题 '%s' 的创建人 '%s' 才能添加选项，当前用户为 '%s'", topic.Title, topic.CreateBy, currentUser)
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
		if err := req.BindUpdates(&updateFields); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}

		currentUser := ctx.GetRequestUser()
		if currentUser == "" {
			return nil, fmt.Errorf("获取用户信息失败，请重新登录")
		}

		var topic VoteTopic
		if err := db.Where("id = ?", updateFields.TopicID).First(&topic).Error; err != nil {
			return &callback.OnTableUpdateRowResp{}, nil
		}

		if topic.CreateBy != currentUser {
			return nil, fmt.Errorf("权限不足：只有投票主题 '%s' 的创建人 '%s' 才能修改选项，当前用户为 '%s'", topic.Title, topic.CreateBy, currentUser)
		}

		status := getTopicStatus(topic.StartTime, topic.EndTime)
		if status != "未开始" {
			return nil, fmt.Errorf("投票主题 '%s' 状态为 '%s'，不能修改选项。只有未开始的投票才能修改选项", topic.Title, status)
		}

		updates := req.GetUpdates()
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

			if option.Topic.CreateBy != currentUser {
				return nil, fmt.Errorf("权限不足：只有投票主题 '%s' 的创建人 '%s' 才能删除选项，当前用户为 '%s'", option.Topic.Title, option.Topic.CreateBy, currentUser)
			}

			status := getTopicStatus(option.Topic.StartTime, option.Topic.EndTime)
			if status != "未开始" {
				return nil, fmt.Errorf("投票主题 '%s' 状态为 '%s'，不能删除选项。只有未开始的投票才能删除选项", option.Topic.Title, status)
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
```

### vote_record_list.go

```go
package vote

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// VoteRecord 投票记录表
type VoteRecord struct {
	ID            int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:记录ID;type:ID" permission:"read" search:"eq"`
	CreatedAt     int64          `json:"created_at" gorm:"autoCreateTime:milli;column:created_at" widget:"name:投票时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	TopicID       int            `json:"topic_id" gorm:"column:topic_id;comment:主题ID;index" widget:"name:投票主题ID;type:select" search:"in" callback:"OnSelectFuzzy" validate:"required" permission:"read"`
	TopicTitle    string         `json:"topic_title" gorm:"-" widget:"name:投票标题;type:input" permission:"read"`
	OptionID      int            `json:"option_id" gorm:"column:option_id;comment:选项ID;index" widget:"name:选项ID;type:number" permission:"read"`
	OptionContent string         `json:"option_content" gorm:"-" widget:"name:选项内容;type:input" permission:"read"`
	VoterName     string         `json:"voter_name" gorm:"column:voter_name;comment:投票人" widget:"name:投票人;type:user" search:"in" permission:"read"`
	IsAnonymous   bool           `json:"is_anonymous" gorm:"column:is_anonymous;comment:是否匿名;default:false" widget:"name:是否匿名;type:switch" permission:"read"`
	Remark        string         `json:"remark" gorm:"column:remark;comment:投票备注" widget:"name:投票备注;type:text_area" search:"like" permission:"read"`
	Topic         *VoteTopic     `json:"-" widget:"-" gorm:"foreignKey:TopicID"`
	Option        *VoteOption    `json:"-" widget:"-" gorm:"foreignKey:OptionID"`
}

func (VoteRecord) TableName() string {
	return "vote_record"
}

// ================ 投票记录管理 ================

// VoteRecordListReq 投票记录列表请求
type VoteRecordListReq struct {
	TopicTitle                string `json:"topic_title" form:"topic_title" widget:"name:投票标题;type:input"`
	OptionContent             string `json:"option_content" form:"option_content" widget:"name:选项内容;type:input"`
	query.SearchFilterPageReq `widget:"-"`
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

	if req.TopicTitle != "" {
		var topicIDs []int
		if err := db.Model(&VoteTopic{}).
			Where("title LIKE ?", "%"+req.TopicTitle+"%").
			Pluck("id", &topicIDs).Error; err == nil && len(topicIDs) > 0 {
			queryDB = queryDB.Where("topic_id IN ?", topicIDs)
		} else {
			return resp.Table(&[]VoteRecord{}).Build()
		}
	}

	if req.OptionContent != "" {
		var optionIDs []int
		if err := db.Model(&VoteOption{}).
			Where("content LIKE ?", "%"+req.OptionContent+"%").
			Pluck("id", &optionIDs).Error; err == nil && len(optionIDs) > 0 {
			queryDB = queryDB.Where("option_id IN ?", optionIDs)
		} else {
			return resp.Table(&[]VoteRecord{}).Build()
		}
	}

	queryDB = queryDB.Preload("Topic").Preload("Option")

	var records []*VoteRecord
	builder := resp.Table(&records).AutoSearchFilterPaged(queryDB, &VoteRecord{}, &req.SearchFilterPageReq)

	if err := builder.Build(); err != nil {
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

	return nil
}

// VoteRecordListTemplate 投票记录管理配置
var VoteRecordListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "投票记录查询",
		Desc:         "投票记录查询管理，支持按主题、选项、投票人等条件筛选",
		Tags:         []string{"投票系统", "记录管理"},
		Request:      &VoteRecordListReq{},
		Response:     query.PaginatedTable[[]VoteRecord]{},
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

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
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
	VoteCount  int     `json:"vote_count" widget:"name:得票人数;type:number;unit:人"`
	Percentage float64 `json:"percentage" widget:"name:得票率%;type:progress;min:0;max:100;unit:%"`
}

// VoteResultResp 查看投票结果响应
type VoteResultResp struct {
	Success     bool                `json:"success" widget:"name:是否成功;type:switch;true_label:成功;false_label:失败"`
	Message     string              `json:"message" widget:"name:处理结果;type:text_area"`
	TopicTitle  string              `json:"topic_title" widget:"name:投票标题;type:input"`
	Description string              `json:"description" widget:"name:投票描述;type:text_area"`
	VoteType    string              `json:"vote_type" widget:"name:投票类型;type:input"`
	Status      string              `json:"status" widget:"name:投票状态;type:select;options:未开始,进行中,已结束;options_colors:info,primary,success"`
	TotalVotes  int                 `json:"total_votes" widget:"name:总选择次数;type:number;unit:次"`
	Options     []*VoteOptionResult `json:"options" widget:"name:投票选项统计;type:table"`
	StartTime   string              `json:"start_time" widget:"name:开始时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss"`
	EndTime     string              `json:"end_time" widget:"name:结束时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss"`
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
		Description: topic.Description,
		VoteType:    topic.VoteType,
		Status:      status,
		TotalVotes:  topic.TotalVotes,
		Options:     optionResults,
		StartTime:   time.UnixMilli(topic.StartTime).Format("2006-01-02 15:04:05"),
		EndTime:     time.UnixMilli(topic.EndTime).Format("2006-01-02 15:04:05"),
	}, nil
}

// VoteResultTemplate 查看投票结果配置
var VoteResultTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "投票结果查询",
		Desc:     "查看投票统计结果，包括各选项得票数、百分比等信息",
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

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/statistics"
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
	VoteTime        string `json:"vote_time" widget:"name:投票时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss"`
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
	now := time.Now().UnixMilli()

	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		db = db.Where("(title LIKE ? OR description LIKE ?) AND start_time <= ? AND end_time > ?",
			"%"+keyword+"%", "%"+keyword+"%", now, now).
			Limit(20)
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
			"投票描述": statistics.Value("投票描述"),
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
		Desc:     "选择投票主题和选项，提交投票，支持单选/多选，防止重复投票",
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
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/statistics"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// VoteTopic 投票主题表
type VoteTopic struct {
	ID          int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:主题ID;type:ID" permission:"read" search:"eq"`
	CreatedAt   int64          `json:"created_at" gorm:"autoCreateTime:milli;column:created_at" widget:"name:创建时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"`
	UpdatedAt   int64          `json:"updated_at" gorm:"autoUpdateTime:milli;column:updated_at" widget:"name:更新时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	Title       string         `json:"title" gorm:"column:title;comment:投票标题" widget:"name:投票标题;type:input" search:"like" validate:"required,min=2,max=100"`
	Description string         `json:"description" gorm:"column:description;comment:投票描述" widget:"name:投票描述;type:text_area" search:"like" validate:"required,min=5,max=500"`
	// select 须配 options_colors，与 options 一一对应，前端用颜色区分选项
	VoteType        string           `json:"vote_type" gorm:"column:vote_type;comment:投票类型" widget:"name:投票类型;type:select;options:单选,多选;options_colors:primary,success;default:单选" search:"in" validate:"required,oneof=单选 多选"`
	MaxSelections   int              `json:"max_selections" gorm:"column:max_selections;comment:最多选择数" widget:"name:最多选择数;type:number;unit:个;default:1" validate:"required_if=VoteType 多选,min=1,max=10"`
	IsAnonymous     bool             `json:"is_anonymous" gorm:"column:is_anonymous;comment:是否匿名;default:false" widget:"name:是否匿名投票;type:switch"`
	ShowResult      bool             `json:"show_result" gorm:"column:show_result;comment:是否显示结果;default:true" widget:"name:是否显示实时结果;type:switch"`
	StartTime       int64            `json:"start_time" gorm:"column:start_time;comment:开始时间;index" widget:"name:开始时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" validate:"required"`
	EndTime         int64            `json:"end_time" gorm:"column:end_time;comment:结束时间;index" widget:"name:结束时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" validate:"required,gtfield=StartTime"`
	Options         []VoteOptionItem `json:"options" gorm:"-" widget:"name:投票选项;type:table" permission:"create" validate:"required,min=2"`
	Content         string           `json:"content" gorm:"column:content;type:text" widget:"name:详细内容;type:richtext;height:420" search:"like"`
	OptionsLink     string           `json:"options_link" gorm:"-" widget:"name:选项列表;type:link;target:_blank" permission:"read"`
	Status          string           `json:"status" gorm:"-" widget:"name:状态;type:select;options:未开始,进行中,已结束;options_colors:info,primary,success" permission:"read"`
	TotalVotes      int              `json:"total_votes" gorm:"column:total_votes;comment:总选择次数;default:0" widget:"name:总选择次数;type:number;unit:次" permission:"read"`
	CreateBy        string           `json:"create_by" gorm:"column:create_by;comment:创建人" widget:"name:创建人;type:user" search:"in" permission:"read"`
	VoteActionLink  string           `json:"vote_action_link" gorm:"-" widget:"name:投票操作;type:link;target:_blank" permission:"read"`
	UserVoteRecords []*VoteRecord    `json:"-" widget:"-" gorm:"foreignKey:TopicID"`
}

func (VoteTopic) TableName() string {
	return "vote_topic"
}

// VoteOptionItem 投票选项项（用于 list/table 组件）
type VoteOptionItem struct {
	Content string `json:"content" widget:"name:选项内容;type:input" validate:"required,min=1,max=100"`
}

// ================ 辅助函数 ================

// getTopicStatus 获取投票状态（计算属性）
func getTopicStatus(startTime, endTime int64) string {
	now := time.Now().UnixMilli()
	if now < startTime {
		return "未开始"
	} else if now > endTime {
		return "已结束"
	}
	return "进行中"
}

// ================ 模糊搜索回调 ================

// voteOnSelectFuzzyTopic 投票主题模糊搜索回调
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
		db = db.Where("title LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%").
			Limit(20)
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
				"最多选择": func() string {
					if topic.VoteType == "单选" {
						return "1个"
					}
					return fmt.Sprintf("%d个", topic.MaxSelections)
				}(),
				"时间范围": fmt.Sprintf("%s - %s",
					time.UnixMilli(topic.StartTime).Format("2006-01-02 15:04"),
					time.UnixMilli(topic.EndTime).Format("2006-01-02 15:04")),
				"是否匿名": func() string {
					if topic.IsAnonymous {
						return "匿名投票"
					}
					return "实名投票"
				}(),
				"创建人": topic.CreateBy,
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
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
	Status                    string `json:"status" form:"status" widget:"name:投票状态;type:select;options:未开始,进行中,已结束;options_colors:info,primary,success"`
	query.SearchFilterPageReq `widget:"-"`
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

	now := time.Now().UnixMilli()
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

	var topics []VoteTopic
	builder := resp.Table(&topics).AutoSearchFilterPaged(queryDB, &VoteTopic{}, &req.SearchFilterPageReq)

	if err := builder.Build(); err != nil {
		return err
	}

	for i := range topics {
		topics[i].Status = getTopicStatus(topics[i].StartTime, topics[i].EndTime)

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

	return nil
}

// VoteTopicListTemplate 投票主题管理配置
var VoteTopicListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "投票主题管理",
		Desc:         "投票主题的增删改查管理，支持创建投票、设置选项、时间控制",
		Tags:         []string{"投票系统", "主题管理"},
		Request:      &VoteTopicListReq{},
		Response:     query.PaginatedTable[[]VoteTopic]{},
		CreateTables: []interface{}{&VoteTopic{}, &VoteOption{}, &VoteRecord{}},
	},
	AutoCrudTable: &VoteTopic{},

	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var topic VoteTopic
		if err := ctx.ShouldBindValidate(&topic); err != nil {
			return nil, err
		}

		topic.CreateBy = ctx.GetRequestUser()
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

		updates := req.GetUpdates()

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
```
