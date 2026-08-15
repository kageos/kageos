# 合同履约节点工作台骨架案例（高级版）

用途：提供一个非内容场景的 AI 原生工作空间应用高级骨架。只有当合同确实存在多类履约节点、独立负责人、独立截止日和独立完成状态时，才使用本案例。

普通合同管理不要默认套用本案例。默认先做一张合同台账表，让用户能录入、筛选、提醒、关闭。

本案例只给建模和路由骨架，不实现具体发送、AI 生成、权限、通知渠道适配等核心代码。

## 先做简单版

合同管理 V1 默认是一张 `contracts.table`：

| 字段 | 类型 | 说明 | 示例 |
|---|---|---|---|
| `contract_no` | input | 合同编号 | HT-2026-001 |
| `contract_name` | input | 合同名称 | 年度服务合同 |
| `counterparty` | input | 合同对方 | 上海某某科技有限公司 |
| `amount` | amount | 合同金额 | 300000 |
| `start_date` | date | 开始日期 | 2026-01-01 |
| `end_date` | date | 到期日期 | 2026-12-31 |
| `owner` | user | 负责人 | 张三 |
| `status` | select | 履约中/快到期/已完成/已归档/已逾期 | 履约中 |
| `next_action` | input | 下一步动作 | 12 月前确认续约 |
| `remind_at` | date | 提醒日期 | 2026-11-30 |
| `files` | files | 合同附件 | signed.pdf |
| `remark` | textarea | 备注 | 客户预计续约 |

只有出现以下情况，才升级到 `milestones.table`：

- 一个合同有多个付款/开票/交付/验收/续约节点，且每个节点有不同截止日。
- 节点有不同负责人或需要独立关闭。
- 需要分别统计节点逾期金额、节点类型风险或多次履约历史。
- 一张合同表里的 `next_action` 已经无法表达实际工作。

## 这个案例优秀在哪里

它展示了一个容易误判的边界：

> 有外部副作用不自动等于必须做 Form。

如果发送提醒需要的参数都已经在节点行里，且人工确认可以通过状态更新清楚表达，那么发送动作优先放在 `milestones.table` 的 `OnTableUpdateRow` 中完成。

只有当发送动作需要临时附件、临时收件人、一次性模板选择、强二次确认、冻结通知包或复杂预览时，才考虑额外 Form。

## 开工前置建模

### 主流程

```text
录入合同 -> 拆分付款/开票/续约/验收节点 -> 系统按到期时间标记待提醒 -> AI 生成提醒草稿 -> 人在节点表确认状态 -> 系统发送提醒并记录结果 -> 节点完成或逾期 -> 汇总风险
```

### 主工作台

```text
milestones.table 合同节点
```

在复杂履约场景里，用户每天主要看合同节点，而不是合同主档。合同主档只是基础台账，真正推动工作的是付款、开票、续约、验收等节点。普通合同台账不要套用这个判断。

### 状态机

```text
未开始
-> 待提醒
-> 提醒草稿已生成
-> 已提醒
-> 已完成
-> 已逾期
-> 已取消
-> 提醒失败
```

状态含义：

- `未开始`：节点还未进入提醒窗口。
- `待提醒`：到达提醒窗口，系统或人要求生成提醒。
- `提醒草稿已生成`：AI 或模板已生成提醒内容，等待人确认。
- `已提醒`：人确认后系统已发送提醒。
- `已完成`：付款、开票、续约或验收已经完成。
- `已逾期`：超过截止日且未完成。
- `已取消`：节点不再需要处理。
- `提醒失败`：发送失败，错误写入本行。

### 路由预算

复杂版 MVP 推荐：

```text
contracts.table       合同主档
milestones.table      主工作台：付款/开票/续约/验收节点
risk_report.chart     可选：风险统计
```

简单版 MVP 只保留：

```text
contracts.table       合同台账：到期、负责人、提醒、下一步动作
```

默认不要加：

```text
send_notice.form
today_due.form
reminder_queue.table
reminder_logs.table
setup_demo.form
batch_send_notice.form
```

### 保留入口

`contracts.table`

维护合同基础信息。它是配置/主档，不是日常推进主工作台。

`milestones.table`

日常主工作台。用户在这里筛选待提醒、提醒草稿已生成、提醒失败、已逾期等状态，并通过更新状态确认发送或完成。

`risk_report.chart`

可选。展示逾期金额、近 30 天到期节点、负责人风险分布等，不承载主操作。

### 拒绝入口

`send_notice.form`

默认拒绝。提醒发送可以通过节点状态从 `提醒草稿已生成` 更新为 `已提醒` 触发。只有需要额外临时参数、附件、模板选择或冻结通知包时才加 Form。

`today_due.form`

拒绝。今日待提醒、今日逾期、未来 7 天到期都应由 `milestones.table` 筛选完成。

`reminder_queue.table`

拒绝。队列是主工作台设计不清的补救。待提醒节点本身就是队列。

`reminder_logs.table`

默认拒绝。基础留痕优先依赖平台操作日志和节点行的发送结果字段。只有需要独立外部回执、审计报表或多次发送历史时再拆事实表。

`setup_demo.form`

拒绝。演示数据不是生产入口。默认数据用 manifest/seed，或由用户直接录入真实合同。

`batch_send_notice.form`

默认拒绝。批量扫描由 AgentTask 或状态扫描完成；人工批量确认应先看是否能通过表格批量更新状态表达。

### 后端补全

- 用户选择合同时，后端补合同名称、客户名称、负责人、默认收件人。
- 用户创建节点时，后端根据节点类型补默认提醒渠道、默认提醒提前天数、默认提醒对象。
- 用户把状态改为 `待提醒` 时，后端生成提醒草稿。
- 用户把状态改为 `已提醒` 时，后端发送提醒，写入发送时间、发送人、错误信息。
- 用户只选合同，不再手填客户 ID、负责人 ID、合同编号等可查询字段。

### 自动动作

- AgentTask 定时扫描临近到期节点，把符合条件的节点标记为 `待提醒`。
- `OnTableUpdateRow` 发现状态变为 `待提醒`，生成提醒草稿并转为 `提醒草稿已生成`。
- `OnTableUpdateRow` 发现状态变为 `已提醒`，发送提醒并写发送结果。
- AgentTask 定时扫描逾期未完成节点，标记 `已逾期`。

## Go 骨架

### PackageContext

```go
var packageContext = &app.PackageContext{
    RouterGroup: "/contract_ops",
    Name:        "合同履约工作台",
    Desc:        "维护合同主档和付款/开票/续约/验收节点；节点状态流转触发提醒草稿生成、人工确认发送和风险统计。",
}
```

### 状态常量

```go
const (
    contractStatusActive   = "履约中"
    contractStatusDone     = "已完成"
    contractStatusArchived = "已归档"

    milestoneStatusPending       = "未开始"
    milestoneStatusNeedNotice    = "待提醒"
    milestoneStatusDraftReady    = "提醒草稿已生成"
    milestoneStatusNoticed       = "已提醒"
    milestoneStatusDone          = "已完成"
    milestoneStatusOverdue       = "已逾期"
    milestoneStatusCanceled      = "已取消"
    milestoneStatusNoticeFailed  = "提醒失败"
)
```

### 合同主档

```go
type Contract struct {
    ID        int        `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:合同ID;type:ID" hide:"create,update"`
    CreatedAt types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
    UpdatedAt types.Time `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`

    ContractNo   string     `json:"contract_no" gorm:"column:contract_no;type:varchar(120);index" widget:"name:合同编号;type:input" validate:"required"`
    ContractName string     `json:"contract_name" gorm:"column:contract_name;type:varchar(240);index" widget:"name:合同名称;type:input" validate:"required"`
    CustomerName string     `json:"customer_name" gorm:"column:customer_name;type:varchar(180);index" widget:"name:客户名称;type:input" validate:"required"`
    Amount       float64    `json:"amount" gorm:"column:amount" widget:"name:合同金额;type:amount"`
    StartDate    types.Time `json:"start_date" gorm:"column:start_date;type:date" widget:"name:开始日期;type:date"`
    EndDate      types.Time `json:"end_date" gorm:"column:end_date;type:date" widget:"name:结束日期;type:date"`
    Status       string     `json:"status" gorm:"column:status;type:varchar(40);index" widget:"name:状态;type:select;options:履约中,已完成,已归档;render_default:履约中"`
    Owner        string     `json:"owner" gorm:"column:owner;type:varchar(120);index" widget:"name:负责人;type:user;render_default:Me()"`

    DefaultNoticeTo      string `json:"default_notice_to" gorm:"column:default_notice_to;type:varchar(500)" widget:"name:默认提醒对象;type:input"`
    DefaultNoticeChannel string `json:"default_notice_channel" gorm:"column:default_notice_channel;type:varchar(40)" widget:"name:默认提醒渠道;type:select;options:站内通知,邮件,企微,钉钉;render_default:站内通知"`
    Files                string `json:"files" gorm:"column:files;type:text" widget:"name:合同附件;type:files"`
}

func (Contract) TableName() string { return "contract_ops_contracts" }
```

### 合同节点

```go
type ContractMilestone struct {
    ID        int        `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:节点ID;type:ID" hide:"create,update"`
    CreatedAt types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
    UpdatedAt types.Time `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`

    ContractID   int    `json:"contract_id" gorm:"column:contract_id;index" widget:"name:所属合同;type:select" callback:"OnSelectFuzzy" hide:"list"`
    ContractName string `json:"contract_name" gorm:"-" widget:"name:合同名称;type:text" hide:"create,update"`
    CustomerName string `json:"customer_name" gorm:"-" widget:"name:客户名称;type:text" hide:"create,update"`

    NodeType    string     `json:"node_type" gorm:"column:node_type;type:varchar(40);index" widget:"name:节点类型;type:select;options:付款,开票,续约,交付,验收,其他;render_default:付款"`
    NodeName    string     `json:"node_name" gorm:"column:node_name;type:varchar(240);index" widget:"name:节点名称;type:input" validate:"required"`
    DueDate     types.Time `json:"due_date" gorm:"column:due_date;type:date;index" widget:"name:截止日期;type:date" validate:"required"`
    Amount      float64    `json:"amount" gorm:"column:amount" widget:"name:节点金额;type:amount"`
    Priority    string     `json:"priority" gorm:"column:priority;type:varchar(20);index" widget:"name:优先级;type:select;options:高,中,低;render_default:中"`
    Status      string     `json:"status" gorm:"column:status;type:varchar(40);index" widget:"name:状态;type:select;options:未开始,待提醒,提醒草稿已生成,已提醒,已完成,已逾期,已取消,提醒失败;render_default:未开始"`
    Owner       string     `json:"owner" gorm:"column:owner;type:varchar(120);index" widget:"name:负责人;type:user;render_default:Me()"`

    NoticeTo       string     `json:"notice_to" gorm:"column:notice_to;type:varchar(500)" widget:"name:提醒对象;type:input"`
    NoticeChannel  string     `json:"notice_channel" gorm:"column:notice_channel;type:varchar(40)" widget:"name:提醒渠道;type:select;options:站内通知,邮件,企微,钉钉;render_default:站内通知"`
    NoticeDraft    string     `json:"notice_draft" gorm:"column:notice_draft;type:text" widget:"name:提醒草稿;type:text_area;rows:5"`
    NoticeSentAt   types.Time `json:"notice_sent_at" gorm:"column:notice_sent_at;type:datetime" widget:"name:提醒时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
    NoticeSentBy   string     `json:"notice_sent_by" gorm:"column:notice_sent_by;type:varchar(120)" widget:"name:提醒人;type:user" hide:"create,update"`
    NoticeError    string     `json:"notice_error" gorm:"column:notice_error;type:text" widget:"name:提醒错误;type:text_area;rows:3" hide:"create,update"`
    ReviewComment string     `json:"review_comment" gorm:"column:review_comment;type:text" widget:"name:处理备注;type:text_area;rows:3"`
}

func (ContractMilestone) TableName() string { return "contract_ops_milestones" }
```

### 查询请求

```go
type ContractListReq struct {
    Keyword string `json:"keyword" form:"keyword" widget:"name:关键词;type:input;placeholder:合同编号、合同名称、客户"`
    Status  string `json:"status" form:"status" widget:"name:状态;type:select;options:,履约中,已完成,已归档"`
    Owner   string `json:"owner" form:"owner" widget:"name:负责人;type:user"`
    query.PageSortReq `widget:"-"`
}

type MilestoneListReq struct {
    Keyword    string `json:"keyword" form:"keyword" widget:"name:关键词;type:input;placeholder:合同、客户、节点"`
    ContractID int    `json:"contract_id" form:"contract_id" widget:"name:所属合同;type:select" callback:"OnSelectFuzzy"`
    NodeType   string `json:"node_type" form:"node_type" widget:"name:节点类型;type:select;options:,付款,开票,续约,交付,验收,其他"`
    Status     string `json:"status" form:"status" widget:"name:状态;type:select;options:,未开始,待提醒,提醒草稿已生成,已提醒,已完成,已逾期,已取消,提醒失败"`
    Owner      string `json:"owner" form:"owner" widget:"name:负责人;type:user"`
    query.PageSortReq `widget:"-"`
}
```

## 回调职责骨架

### 选择合同

`contract_id` 使用 `OnSelectFuzzy`，但列表显示合同名称和客户，不显示裸 ID。

```go
func onSelectFuzzyContract(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
    // 查询 Contract，Label 形如：合同名称 / 客户名称（履约中，负责人）
    // DisplayInfo 至少包含：合同名称、客户名称、合同编号、状态、负责人、默认提醒渠道。
    // Statistics 展示：选中合同、客户、状态、负责人。
    return &callback.OnSelectFuzzyResp{}, nil
}
```

### 新增节点

```go
func onAddMilestoneRow(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
    // 1. 绑定 ContractMilestone。
    // 2. 根据 ContractID 查询 Contract。
    // 3. 后端补 NoticeTo、NoticeChannel、Owner 等默认值。
    // 4. 创建节点。
    // 5. 返回时补 ContractName、CustomerName 展示字段。
    return &callback.OnTableAddRowResp{}, nil
}
```

### 更新节点

```go
func onUpdateMilestoneRow(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
    // 1. 只处理 ChangedFields。
    // 2. 校验 Status 是否允许。
    // 3. 如果状态变为 待提醒，生成 NoticeDraft，并把状态更新为 提醒草稿已生成。
    // 4. 如果状态变为 已提醒，使用本行 NoticeDraft、NoticeTo、NoticeChannel 发送提醒。
    // 5. 发送成功写 NoticeSentAt、NoticeSentBy、清空 NoticeError。
    // 6. 发送失败写 NoticeError，并把状态改为 提醒失败。
    // 7. 如果状态变为 已完成/已取消，只更新节点，不额外发送。
    return &callback.OnTableUpdateRowResp{}, nil
}
```

### 定时扫描

```go
type ScanMilestonesForNoticeResp struct {
    Message            string `json:"message" widget:"name:处理结果;type:text"`
    DraftReadyCount    int    `json:"draft_ready_count" widget:"name:生成草稿数;type:integer"`
    NeedNoticeCount    int    `json:"need_notice_count" widget:"name:待提醒数;type:integer"`
    MarkedOverdueCount int    `json:"marked_overdue_count" widget:"name:标记逾期数;type:integer"`
}

func scanMilestonesForNotice(ctx *app.Context, resp response.Response) error {
    // AgentTask 或定时函数调用：
    // 1. 找到未来 N 天到期且状态为 未开始 的节点。
    // 2. 标记为 待提醒，或直接生成提醒草稿后标记 提醒草稿已生成。
    // 3. 找到已经超过 DueDate 且未完成/未取消的节点，标记 已逾期。
    return resp.Form(&ScanMilestonesForNoticeResp{
        Message:            "扫描完成",
        DraftReadyCount:    0,
        NeedNoticeCount:    0,
        MarkedOverdueCount: 0,
    }).Build()
}
```

## 路由注册骨架

```go
func init() {
    packageContext.GET("contracts.table", ContractList, &app.TableTemplate{
        BaseConfig: app.BaseConfig{
            Name:         "合同主档",
            Desc:         "维护合同编号、客户、金额、周期、负责人和默认提醒信息；日常推进请看合同节点。",
            Request:      &ContractListReq{},
            CreateTables: []interface{}{&Contract{}},
        },
        AutoCrudTable:     &Contract{},
        OnTableAddRow:     onAddContractRow,
        OnTableUpdateRow:  onUpdateContractRow,
    })

    packageContext.GET("milestones.table", MilestoneList, &app.TableTemplate{
        BaseConfig: app.BaseConfig{
            Name:             "合同节点",
            Desc:             "主工作台：筛选付款、开票、续约、验收等节点；状态更新会触发提醒草稿生成和人工确认发送。",
            Request:          &MilestoneListReq{},
            CreateTables:     []interface{}{&ContractMilestone{}, &Contract{}},
            OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{"contract_id": onSelectFuzzyContract},
        },
        AutoCrudTable:    &ContractMilestone{},
        OnTableAddRow:    onAddMilestoneRow,
        OnTableUpdateRow: onUpdateMilestoneRow,
    })

    packageContext.GET("risk_report.chart", RiskReport, &app.ChartTemplate{
        BaseConfig: app.BaseConfig{
            Name:         "履约风险统计",
            Desc:         "按负责人、节点类型和状态观察逾期风险；只做观察，不做主操作入口。",
            Request:      &RiskReportReq{},
            Response:     &chart.BarChart{},
            CreateTables: []interface{}{&ContractMilestone{}, &Contract{}},
        },
        ChartType: app.ChartTypeBar,
    })
}
```

注意：Table 不写 `Response: &[]ContractMilestone{}`。表格列 schema 来自 `AutoCrudTable`，列表响应由 Handler 返回 `response.TableResult`。Chart 则要写与 `resp.Chart(...)` 一致的 chart 类型，并声明 `ChartType`。

## Bad Case 对照

不要这样建：

```text
create_contract.form
create_milestone.form
send_notice.form
today_due.form
reminder_queue.table
reminder_logs.table
batch_send_notice.form
setup_demo.form
```

原因：

- 合同和节点新增是表格新增，不是 Form。
- 今日待办是节点表筛选，不是查询 Form。
- 当前提醒草稿是节点字段，不是单独提醒草稿表。
- 发送提醒默认由节点状态更新触发，不默认做 Form。
- 基础发送结果写回节点行，基础留痕依赖平台操作日志。
- 批量扫描由 AgentTask 做，不做演示按钮。
- 示例数据不是生产入口。
- 合同、节点、账号这类生产对象先确认删除语义：需要保留生命周期时用已归档、已取消、已废弃或启用开关；开放删除入口时用 `OnTableDeleteRows` 做软删除并写 `deleted_at/deleted_by`。

## 什么时候可以加 Form

只有出现下面需求时，才考虑新增 Form：

- 发送前需要临时上传附件；
- 发送前需要临时选择多个收件人或抄送人；
- 需要选择一次性模板并预览；
- 需要生成冻结通知包；
- 要调用外部系统且必须有人二次确认；
- 同一节点需要多次发送历史和外部回执报表。

即使加 Form，也要说明它为什么不能由 `milestones.table` 的状态更新表达。

## AgentTask 骨架

```go
func init() {
    packageContext.AddAgentTask(app.AgentTask{
        Code: "scan_contract_milestones",
        Name: "合同节点每日扫描",
        Desc: "每天扫描临近到期和逾期的合同节点，生成提醒草稿或标记风险。",
        // 不主动填写 Policy；默认 create_if_missing。
        Message: `读取合同节点表：
- 对未来 7 天到期且未开始的节点，生成提醒草稿并标记为提醒草稿已生成。
- 对已超过截止日期且未完成/未取消的节点，标记为已逾期。
- 不直接发送外部提醒；发送需要人把状态更新为已提醒。`,
    })
}
```

注意：AgentTask 负责无人值守扫描和草稿生成，不直接绕过人发送外部提醒。

## 实现检查清单

- `milestones.table` 是否是主工作台？
- 是否没有裸露 `contract_id` 给用户手填？
- 选了合同后，客户、负责人、提醒渠道是否后端补齐？
- 是否没有默认创建 `send_notice.form`？
- 状态改为 `已提醒` 是否能表达人工确认发送？
- 发送失败是否写回同一节点行？
- 今日待办是否能用节点表筛选？
- 基础留痕是否依赖平台操作日志？
- 是否没有为了展示合同名称声明 GORM 物理外键？
