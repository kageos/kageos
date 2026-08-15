# AI 原生工作空间应用建模模式

只适用于已经证明“一张主 Table + 筛选 + 少量状态/提醒”不够用的 kageos 工作空间应用。不要因为业务里出现状态、提醒、附件、负责人、到期日、备注或 AI 辅助，就自动进入多表工作流模式。

本模式定义的是 kageos 复杂应用原型：先证明简单表格不够，再建模业务对象、主工作台、状态机、人工决策点、系统/AI 动作和平台能力边界。目标不是显得完整，而是让用户看到的入口足够少、足够好用。

## 复杂度分级

先选择最低可用级别，再考虑升级：

| 级别 | 形态 | 适用场景 |
|---|---|---|
| L0 | 1 个主 Table | 合同台账、客户台账、证书台账、设备台账、简单提醒 |
| L1 | 1 个主 Table + 回调/定时提醒 | 到期提醒、附件提取、状态自动更新 |
| L2 | 2-3 个 Table/Form | 确有独立生命周期的对象，例如任务和结果、主档和流水 |
| L3 | 多表 + AgentTask + Chart | 多角色协同、外部动作、独立审计、跨对象风险运营 |

MVP 默认从 L0 或 L1 开始。升级到 L2/L3 必须写清楚为什么一张表不够。

## 开工前置检查

写代码前必须先输出：

1. 最小可用形态：一张主 Table 能否解决？如果不能，原因是什么。
2. 主流程：用户输入什么，AI/系统生产什么，人在哪里确认。
3. 主工作台：用户每天最常打开的一个 Table/List 是什么。
4. 主对象状态机：状态如何自动流转，失败状态如何可见。
5. 路由预算：MVP 默认 1 个入口，最多 2 个；超过要逐个说明不可替代原因。
6. 对象边界：哪些是独立对象，哪些只是当前对象的字段或派生产物。
7. 后端补全：用户选择上游对象后，哪些字段由后端查询补齐。
8. 拒绝路由：明确哪些 Form/Table 不创建，以及为什么。

没有这 8 项，不要开始建表或写函数。

## 模型高频误区

AI 写 kageos 应用最容易犯这些错误。实现前逐项自检。

1. 从“完整系统”出发，而不是从用户每天要填的一张表出发。结果是表很多，用户不知道去哪工作。
2. 每个动作都创建 Form。AI 写入、状态更新、审核、当前预览生成都应优先走 Table 回调。
3. 把派生产物拆成独立对象。当前对象的 HTML/PDF/图片预览默认是主对象字段，不是新表。
4. 裸露内部 ID。不要让用户理解“账号 ID 1、任务 ID 2”；使用业务选择器和展示字段。
5. 用选择器掩盖错误边界。选了上游对象后能查出的字段，由后端补齐，不要继续让用户选。
6. 重造平台已有能力。操作日志、Table 筛选、OnTableUpdateRow、AgentTask、runbook 不要重复造。
7. 提前做完整大系统。MVP 默认 1 个主入口，发布包、发布记录、复盘等按独立生命周期再扩。
8. 把 AI 后台复杂度暴露给人。人看业务对象、状态、结果文件和确认动作；AI 看 runbook 和后台动作。
9. 看到“合同、审批、提醒、风险”就自动拆申请表、节点表、日志表和图表。先问一张合同表能不能用。

## Table / Form 边界

优先使用 Table 承载长期对象和状态流转：

- 账号、配置、任务、草稿、发布记录、复盘结果。
- AI 写草稿走主对象表 `OnTableAddRow`。
- 审核通过、退回、废弃走主对象表 `OnTableUpdateRow`。
- 当前对象的 HTML/PDF/图片预览能自动生成时，放在主对象字段并由 Table 回调触发。

只在这些情况使用 Form：

- 需要人工参数或确认；
- 有外部副作用，例如发布到公众号、抖音、小红书；
- 生成发布包并冻结版本；
- 跑一次复盘报告或外部系统同步。

注意：外部副作用不自动等于 Form。如果外部动作可以被主对象状态更新清楚表达，且动作参数已经在表行里，就优先使用 `OnTableUpdateRow`。例如合同节点从 `提醒草稿已生成` 更新为 `已提醒` 时发送提醒。

不要为普通新增、编辑、查询、审核、当前预览生成创建 Form。

## 对象边界

派生产物默认不拆表。

例如当前草稿的 HTML 预览、封面图、导出 PDF，优先是草稿字段：

```go
type Draft struct {
    Status              string
    PreviewFile         string
    PreviewRenderStatus string
    PreviewErrorMessage string
    ReviewComment       string
}
```

只有满足以下任一条件，才拆成结果表：

- 需要多版本历史；
- 需要独立审计；
- 需要跨主对象聚合；
- 有独立权限；
- 生命周期和主对象不同。

## ID 和选择器

不要裸露内部 ID：

```go
// Bad
ProfileID int `widget:"name:公众号ID;type:integer"`
TopicID   int `widget:"name:任务ID;type:integer"`
```

使用业务选择器和展示字段：

```go
// Good
TopicID     int    `json:"topic_id" gorm:"column:topic_id;index" widget:"name:生成任务;type:select" callback:"OnSelectFuzzy" hide:"list"`
ProfileID   int    `json:"profile_id" gorm:"column:profile_id;index" widget:"-"`
TopicTitle  string `json:"topic_title" gorm:"-" widget:"name:任务主题;type:text" hide:"create,update"`
AccountName string `json:"account_name" gorm:"-" widget:"name:公众号名称;type:text" hide:"create,update"`
```

只让用户选择真正需要判断的上游对象。选了任务后能查到公众号，就不要再让用户选公众号。

`OnSelectFuzzy` 必须用 `Label`、`DisplayInfo`、`Statistics` 展示业务名称和上下文，不要只显示 `#1` 或 `ID 1`。

## 公众号内容工厂反例

不要把 MVP 写成：

```text
setup.form
create_task.form
ai_write_draft.form
generate_sample_draft.form
drafts.table
render_preview.form
previews.table
review_preview.form
review_queue.table
reviews.table
today_queue.form
batch_render_previews.form
```

问题：

- AI 写草稿不该是 Form；
- 当前预览不该拆表；
- 生成预览不该让人点；
- 审核不该单独 Form；
- 平台操作日志已覆盖基础审核留痕；
- 今日待审可由草稿表筛选；
- 示例和初始化入口不是生产能力。

推荐 MVP：

```text
<./profiles.table>
<./topics.table>
<./drafts.table>
```

草稿表包含正文、预览文件、渲染状态、审核状态和审核意见。新增待预览草稿后自动生成预览，正文或预览样式变化后自动重新生成。

## 必须拒绝的坏味道

- 从数据库表数量倒推产品。
- 每个动作都创建 Form。
- 每个中间状态都创建结果表。
- 用今日清单、工作台 Form、审核队列表补救主工作台设计不清。
- 用户选择上游对象后还要求手填下游 ID。
- 为了列表展示名称声明 GORM 物理外键关联。
- 用示例数据、初始化入口、批量演示入口充当生产能力。

## 输出格式

实现前，在回复或设计注释中给出：

```text
最小可用形态：
升级理由：
主流程：
主工作台：
状态机：
路由预算：
保留入口：
拒绝入口：
后端补全：
自动动作：
```

确认这份建模后再写代码。
