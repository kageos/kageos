# 案例：投票系统（Table + Form）

## 一、项目概要

- **类型**：多表（主题 + 选项 + 记录）+ 两个 POST Form（提交投票、查看结果）。
- **GET Table**：`vote_topic_list`（投票主题管理）、`vote_option_list`（投票选项管理）、`vote_record_list`（投票记录查询）。
- **POST Form**：`vote_submit`（选择主题 + 选择选项 **multiselect；depend_on:topic_id** + 备注）、`vote_result`（选择主题 → 返回结果：选项得票数、得票率 table）。
- **关系**：主题 1:N 选项，主题 1:N 记录；选项选主题用 OnSelectFuzzy；提交时选主题用 OnSelectFuzzy（仅「进行中」），选选项用 **multiselect + depend_on:topic_id**（选项列表随主题变化）。
- **状态**：主题状态不落库，按开始/结束时间**实时计算**（未开始/进行中/已结束）；列表可筛「投票状态」时在 Handler 里用时间条件过滤。
- **link**：主题列表带「选项列表」link（跳转 vote_option_list）、「投票操作」link（进行中且未投→vote_submit，已投或已结束→vote_result）。
- **适合参考**：主从多表（主题-选项-记录）、**multiselect + depend_on**、OnSelectFuzzy、link 按状态切换、时间状态计算、POST 提交与得票率更新。

---

## 二、PRD 要点（表格格式）

### 1. 主题表（vote_topic_list）

**表单字段（新增/编辑）**

| 字段           | 类型     | 必填 | 说明 |
|----------------|----------|------|------|
| 投票标题       | 文本输入 | ✓   | 2–100 字 |
| 投票描述       | 多行文本 | ✓   | 5–500 字 |
| 投票类型       | 下拉选择 | ✓   | 单选/多选，默认单选 |
| 最多选择数     | 数字输入 | ✓   | 多选时 1–10，单选时固定 1 |
| 是否匿名投票   | 开关     | ✗   | 默认否 |
| 是否显示实时结果 | 开关   | ✗   | 默认是 |
| 开始时间       | 时间选择 | ✓   | 必填 |
| 结束时间       | 时间选择 | ✓   | 必须晚于开始时间 |
| 投票选项       | 表格     | ✓   | type:table，至少 2 条；每行「选项内容」input |
| 详细内容       | 富文本   | ✗   | type:richtext |
| 创建人         | 用户选择 | ✗   | 可自动带当前用户 |

**列表模式**（可筛：投票状态）

| 创建时间 | 更新时间 | 投票标题 | 投票类型 | 最多选择 | 开始时间 | 结束时间 | 状态 | 总选择次数 | 创建人 | 选项列表 | 投票操作 |
|----------|----------|----------|----------|----------|----------|----------|------|------------|--------|----------|----------|

**说明**：
- 状态按开始/结束时间计算（未开始/进行中/已结束）。
- 「选项列表」link：`BuildFunctionUrlWithText("vote_option_list", params, "查看选项列表")`，params 为 `VoteOption{TopicID: ID}`。
- 「投票操作」link：进行中且当前用户未投 → vote_submit（点击参与投票）；进行中已投或未开始/已结束 → vote_result（查看投票结果）。

---

### 2. 选项表（vote_option_list）

**表单字段（新增/编辑）**

| 字段     | 类型     | 必填 | 说明 |
|----------|----------|------|------|
| 投票主题 | 下拉选择 | ✓   | OnSelectFuzzy，可选仅当前用户创建的主题 |
| 选项内容 | 文本输入 | ✓   | 1–100 字 |

**列表模式**

| 创建时间 | 选项ID | 投票主题 | 选项内容 | 得票人数 | 得票率% |
|----------|--------|----------|----------|----------|---------|

**说明**：新增/编辑选项时校验：只有主题创建人可增改；只有「未开始」的主题可增改选项。

---

### 3. 记录表（vote_record_list）

仅列表查询，无可编辑表单（记录由 vote_submit 产生）。

**列表模式**（可筛：投票标题、选项内容）

| 投票时间 | 投票标题 | 选项内容 | 投票人 | 是否匿名 | 投票备注 |
|----------|----------|----------|--------|----------|----------|

**说明**：投票标题、选项内容来自 Preload 的 Topic/Option；匿名时投票人展示为「匿名用户」。

---

### 4. 提交投票 Form（vote_submit，POST）

**请求**

| 字段         | 类型     | 必填 | 说明 |
|--------------|----------|------|------|
| 选择投票主题 | 下拉选择 | ✓   | OnSelectFuzzy，仅「进行中」的主题 |
| 选择投票选项 | 多选下拉 | ✓   | **depend_on:topic_id**；选项列表随主题变化；单选主题下最多选 1 个，多选主题下最多选 max_selections |
| 投票备注     | 多行文本 | ✗   | 最多 500 字 |

**响应**

| 字段       | 类型     | 说明 |
|------------|----------|------|
| 是否成功   | 开关     | 成功/失败 |
| 处理结果   | 多行文本 | 提示信息 |
| 投票标题   | 文本     | 所选主题标题 |
| 已选选项   | 多行文本 | 所选选项内容 |
| 投票时间   | 时间     | 提交时间 |
| 投票类型   | 开关     | 是否匿名 |
| 查看结果   | link     | 跳转 vote_result |

**业务校验**：主题存在且状态为「进行中」；每人每个主题只能投一次；选项属于该主题；单选选 1 个，多选不超过 max_selections。

---

### 5. 查看结果 Form（vote_result，POST）

**请求**

| 字段         | 类型     | 必填 | 说明 |
|--------------|----------|------|------|
| 选择投票主题 | 下拉选择 | ✓   | OnSelectFuzzy |

**响应**

| 字段           | 类型     | 说明 |
|----------------|----------|------|
| 是否成功       | 开关     | 成功/失败 |
| 处理结果       | 多行文本 | 提示信息 |
| 投票标题       | 文本     | 主题标题 |
| 投票描述       | 多行文本 | 主题描述 |
| 投票类型       | 文本     | 单选/多选 |
| 投票状态       | 下拉     | 未开始/进行中/已结束 |
| 总选择次数     | 数字     | 总票数 |
| 投票选项统计   | 表格     | 选项内容、得票人数、得票率% |
| 开始时间/结束时间 | 时间   | 主题时间范围 |

**说明**：若主题设置「不显示实时结果」，则仅「已结束」后可查看；否则进行中、已结束均可查看。

---

## 三、业务校验与逻辑要点

- **主题**：开始时间 < 结束时间；新增时至少 2 个选项；单选时 max_selections 固定为 1。
- **选项**：仅主题创建人可增删改；仅「未开始」的主题可增删改选项。
- **提交投票**：主题状态为「进行中」；当前用户在该主题下未投过；选项 ID 属于该主题；单选选 1 个，多选 1～max_selections；提交后写 vote_record、更新主题 total_votes、更新选项 vote_count 与 percentage。
- **主题列表筛「投票状态」**：未开始 `start_time > now`，进行中 `start_time <= now AND end_time > now`，已结束 `end_time <= now`。
- **记录列表筛「投票标题」「选项内容」**：用 Topic.title LIKE / Option.content LIKE 查出 id 列表，再 topic_id IN / option_id IN 过滤。

---

## 四、文件与路由

| 文件               | 说明           | 注册 |
|--------------------|----------------|------|
| vote_topic_list.go | 投票主题管理   | GET `vote_topic_list` |
| vote_option_list.go| 投票选项管理   | GET `vote_option_list` |
| vote_record_list.go| 投票记录查询   | GET `vote_record_list` |
| vote_submit.go     | 提交投票       | POST `vote_submit` |
| vote_result.go     | 查看结果       | POST `vote_result` |
| init_.go           | 路由组         | RouterGroup: `/data/vote` |

**OnSelectFuzzyMap**：
- 主题表：无（列表筛状态用时间条件）。
- 选项表：`"topic_id": voteOnSelectFuzzyTopicForOptionList`（可选仅当前用户创建的主题）。
- 记录表：`"topic_id": voteOnSelectFuzzyTopic`。
- vote_submit：主题与选项均用 OnSelectFuzzy；选项 **multiselect + depend_on:topic_id**，选项回调根据 topic_id 拉该主题下选项。

---

## 五、参考代码（namespace/luobei/operations/code/api/data/vote）

以下代码对应路径 `namespace/luobei/operations/code/api/data/vote`，便于大模型学习实现细节。

init_.go
```go
package vote

import (
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/data/vote",
	Name:        "vote",
	Desc:        "投票系统：主题、选项、记录管理及提交投票、查看结果",
}
```

vote_topic_list.go
```go
// 投票主题管理：VoteTopic 数据模型、列表 Handler、Template、状态计算与 link（选项列表、投票操作）

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

// ================ 投票主题管理 ================

// VoteTopic 投票主题表
type VoteTopic struct {
	ID                 int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:主题ID;type:ID" permission:"read" search:"eq"`
	CreatedAt          int64          `json:"created_at" gorm:"autoCreateTime:milli;column:created_at" widget:"name:创建时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"`
	UpdatedAt          int64          `json:"updated_at" gorm:"autoUpdateTime:milli;column:updated_at" widget:"name:更新时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"`
	DeletedAt          gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	Title               string `json:"title" gorm:"column:title;comment:投票标题" widget:"name:投票标题;type:input" search:"like" validate:"required,min=2,max=100"`
	Description         string `json:"description" gorm:"column:description;type:text;comment:投票描述" widget:"name:投票描述;type:text_area" search:"like" validate:"required,min=5,max=500"`
	VoteType            string `json:"vote_type" gorm:"column:vote_type;comment:投票类型;default:单选" widget:"name:投票类型;type:select;options:单选,多选;options_colors:primary,success;default:单选" search:"in" validate:"required,oneof=单选 多选"`
	MaxSelections       int    `json:"max_selections" gorm:"column:max_selections;comment:最多选择数;default:1" widget:"name:最多选择数;type:number" search:"gte,lte" validate:"required,min=1,max=10"`
	IsAnonymous         bool   `json:"is_anonymous" gorm:"column:is_anonymous;comment:是否匿名;default:0" widget:"name:是否匿名投票;type:switch;default:false"`
	ShowRealtimeResult  bool   `json:"show_realtime_result" gorm:"column:show_realtime_result;comment:是否显示实时结果;default:1" widget:"name:是否显示实时结果;type:switch;default:true"`
	StartTime           int64  `json:"start_time" gorm:"column:start_time;comment:开始时间;index" widget:"name:开始时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" validate:"required"`
	EndTime             int64  `json:"end_time" gorm:"column:end_time;comment:结束时间;index" widget:"name:结束时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" validate:"required"`
	DetailContent       string `json:"detail_content" gorm:"column:detail_content;type:text;comment:详细内容" widget:"name:详细内容;type:richtext"`
	Creator             string `json:"creator" gorm:"column:creator;comment:创建人" widget:"name:创建人;type:user" search:"in"`
	TotalVotes          int    `json:"total_votes" gorm:"column:total_votes;comment:总选择次数;default:0" widget:"name:总选择次数;type:number" permission:"read"`

	Status    string `json:"status" gorm:"-" widget:"name:状态;type:select;options:未开始,进行中,已结束;options_colors:info,primary,success" search:"-" permission:"read"`
	OptionLink string `json:"option_link" gorm:"-" widget:"name:选项列表;type:link;target:_blank" permission:"read"`
	VoteLink  string `json:"vote_link" gorm:"-" widget:"name:投票操作;type:link;target:_blank" permission:"read"`
}

func (VoteTopic) TableName() string {
	return "vote_topic"
}

// VoteTopicListReq 主题列表请求
type VoteTopicListReq struct {
	Status string `json:"status" form:"status" widget:"name:投票状态;type:select;options:未开始,进行中,已结束;options_colors:info,primary,success"`
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

	queryDB := db.Model(&VoteTopic{})

	if req.Status != "" {
		now := time.Now().UnixMilli()
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
		topics[i].Status = calculateTopicStatus(topics[i].StartTime, topics[i].EndTime)
		topics[i].OptionLink, _ = ctx.BuildFunctionUrlWithText("vote_option_list", &VoteOption{TopicID: topics[i].ID}, "查看选项列表")
		topics[i].VoteLink = buildVoteOperationLink(ctx, &topics[i])
	}

	return nil
}

func calculateTopicStatus(startTime, endTime int64) string {
	now := time.Now().UnixMilli()
	if now < startTime {
		return "未开始"
	}
	if now >= startTime && now < endTime {
		return "进行中"
	}
	return "已结束"
}

// buildVoteOperationLink 根据主题状态与当前用户是否已投，返回「投票操作」link：进行中且未投→vote_submit，否则→vote_result
func buildVoteOperationLink(ctx *app.Context, topic *VoteTopic) string {
	status := calculateTopicStatus(topic.StartTime, topic.EndTime)
	if status != "进行中" {
		url, _ := ctx.BuildFunctionUrlWithText("vote_result", map[string]interface{}{"topic_id": topic.ID}, "查看投票结果")
		return url
	}
	db := ctx.GetGormDB()
	if db == nil {
		return ""
	}
	user := ctx.GetRequestUser()
	var count int64
	db.Model(&VoteRecord{}).Where("topic_id = ? AND voter = ?", topic.ID, user).Count(&count)
	if count > 0 {
		url, _ := ctx.BuildFunctionUrlWithText("vote_result", map[string]interface{}{"topic_id": topic.ID}, "查看投票结果")
		return url
	}
	url, _ := ctx.BuildFunctionUrlWithText("vote_submit", map[string]interface{}{"topic_id": topic.ID}, "参与投票")
	return url
}

// VoteTopicListTemplate 主题管理配置
var VoteTopicListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "投票主题管理",
		Desc:         "投票主题的增删改查，包括标题、描述、类型、时间、选项等",
		Tags:         []string{"投票系统", "主题管理"},
		Request:      &VoteTopicListReq{},
		Response:     query.PaginatedTable[[]VoteTopic]{},
		CreateTables: []interface{}{&VoteTopic{}, &VoteOption{}, &VoteRecord{}},
	},
	AutoCrudTable: &VoteTopic{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row VoteTopic
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		if row.VoteType == "单选" {
			row.MaxSelections = 1
		}
		if row.StartTime >= row.EndTime {
			return nil, fmt.Errorf("结束时间必须晚于开始时间")
		}
		err := db.Create(&row).Error
		if err != nil {
			logger.Errorf(ctx, "Create vote_topic err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		var updateFields VoteTopic
		if err := req.BindUpdates(&updateFields); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}
		updates := req.GetUpdates()
		err := db.Model(&VoteTopic{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "Update vote_topic err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		_ = db.Where("topic_id IN ?", req.GetIds()).Delete(&VoteRecord{}).Error
		_ = db.Where("topic_id IN ?", req.GetIds()).Delete(&VoteOption{}).Error
		if err != nil {
			logger.Errorf(ctx, "Delete vote_option err: %v", err)
		}
		err = db.Delete(&VoteTopic{}, "id in ?", req.GetIds()).Error
		if err != nil {
			logger.Errorf(ctx, "Delete vote_topic err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

// voteOnSelectFuzzyTopic 主题模糊搜索（用于选项表、记录表、vote_submit、vote_result）
func voteOnSelectFuzzyTopic(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}
	var topics []VoteTopic
	if req.IsByValue() {
		db = db.Model(&VoteTopic{}).Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Model(&VoteTopic{}).Where("id in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		db = db.Model(&VoteTopic{}).Where("title LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%").Limit(20)
	}
	db.Find(&topics)
	items := make([]*callback.SelectFuzzyItem, 0)
	for _, t := range topics {
		items = append(items, &callback.SelectFuzzyItem{
			Value: t.ID,
			Label: fmt.Sprintf("%s (%s)", t.Title, calculateTopicStatus(t.StartTime, t.EndTime)),
			DisplayInfo: map[string]interface{}{
				"投票标题": t.Title,
				"投票类型": t.VoteType,
				"开始时间": t.StartTime,
				"结束时间": t.EndTime,
			},
		})
	}
	return &callback.OnSelectFuzzyResp{
		MaxSelections: 1,
		Items:         items,
		Statistics: map[string]interface{}{
			"投票标题": statistics.Value("投票标题"),
			"投票类型": statistics.Value("投票类型"),
		},
	}, nil
}

func init() {
	packageContext.GET("vote_topic_list", VoteTopicList, VoteTopicListTemplate)
}
```
