# SDK 公共运行能力

本文档是按需参考，只在 SDK 主文档和匹配案例不足以确认横切运行能力时读取。它覆盖平台 API、当前用户上下文、Table 回调高级能力、事务和副作用、日志错误处理、Python 运行时。具体 Form/Table/Chart 写法以 `/system/prompt/sdk/agent-app-sdk-readme` 和匹配案例为准。

## 使用原则

业务代码只写业务规则，不重造平台能力：

- 平台接口：用 `ctx.APICall(...)`；确需沉淀平台接口包装函数时放在 `/system/tools/openapi`，不要裸写 HTTP client、硬编码 token、直连平台库。
- 消息通知：需要通知用户时使用 SDK `ctx.SendNotification`，由 message-service 异步写站内信，并由平台外部 webhook 渠道做摘要触达；业务代码不要直接耦合飞书、邮件、钉钉、企业微信等渠道，也不要让普通业务等待通知投递完成。
- 通用权限、审批、评论、收藏不属于 MVP 应用侧能力，不在每个业务系统自造。
- Table 更新日志由平台记录；业务上确实需要流水、操作记录、支付记录、投票记录时，可以建只读业务 Table。
- 运行上下文：从 `ctx` 取当前用户、部门、trace、full_code_path，不让用户表单伪造。
- 定时任务和后台调度由 timer-scheduler 独立服务承载；业务应用不要自造调度器，也不要引用 `/scheduled_tasks`、`/scheduled_agent_tasks` 等旧平台 API。Form 自带默认定时执行时，在 `FormTemplate.Schedules` 声明，由平台发布后同步到 timer-scheduler。

## 平台 API

业务函数调用 kageos 平台能力使用：

```go
var out SomeResp
if err := ctx.APICall(http.MethodGet, "/workspace/api/v1/operate_log/general?"+query.Encode(), nil, &out); err != nil {
    logger.Errorf(ctx, "[OperateLog] APICall failed, err=%v", err)
    return nil, fmt.Errorf("[系统错误]-[OperateLog] 调用平台接口失败: %w", err)
}
```

规则：

- `method` 使用 `http.MethodGet`、`http.MethodPost` 等。
- `path` 使用平台网关路径，例如 `/workspace/api/v1/operate_log/general`。
- `reqBody` 是请求体；GET 可传 `nil`。
- `respData` 是响应 data 对应结构体指针。
- SDK 会带上 token、trace、request_user、department、client_source、source_type、source_ref。

禁止：

- 裸写 `http.Client` 调平台内部接口。
- 硬编码 token。
- 伪造 request_user。
- 直连平台数据库。
- 伪造平台运行上下文。

操作日志等平台领域，优先走 `platform_engineer` 角色和 `ctx.APICall`；确需复用包装函数时使用 `/system/tools/openapi` 下的系统工具函数。

## 消息通知

应用函数需要通知用户时，使用 SDK 的 `ctx.SendNotification`。message-service 负责收件箱存储和后续渠道分发，应用侧只表达“发给哪些用户、标题、内容、附件”，不要关心飞书、钉钉、企业微信等具体渠道。`ctx.SendNotification` 是异步投递命令：成功只表示通知命令已发布到 NATS，不等待 message-service 落库或渠道投递完成。组织架构通知暂不暴露，不要使用部门作为消息接收方。

```go
err := ctx.SendNotification(&app.SendNotificationOpts{
    ToUsers:     "alice,bob",
    Title:       "工单处理完成",
    Message:     "工单 #123 已处理完成，请查看详情。",
    ContentType: "markdown",
    Files:       "kageos/reports/work-order-123.pdf",
})
if err != nil {
    logger.Errorf(ctx, "[Notify] send notification failed: %v", err)
}
```

规则：

- `ToUsers` 推荐显式填写；通知当前请求用户时可省略，由 message-service 兜底到真实请求用户。没有真实请求用户时必须显式填写；多个用户用逗号分隔。
- `Message` 和 `Files` 至少填写一个；普通通知推荐写一条简短 `Message`，纯文件交付可只传 `Files`。
- `ContentType` 默认 `markdown`，也可用 `text` 或 `html`。
- `Files` 可选，填写平台文件引用字符串，格式 `bucket/object_key`，多个用逗号分隔。站内信详情和移动端处理页会展示完整附件；外部 webhook 卡片只展示附件数量和前几个文件名，并引导用户回 kageos 详情查看/下载，不做飞书/钉钉/企业微信原生文件上传。
- 对普通业务函数，通知失败只记录日志，不要阻塞主业务返回；只有函数本身就是“发送通知/消息”时，才把发布失败作为业务错误返回。
- 发送人、部门、trace、full_code_path 等上下文由 SDK 自动带上，不要让用户表单填写。
- 组织架构功能隐藏期间，不要使用部门作为消息接收方。
- 需要“定时通知”时，优先把通知逻辑放在被定时执行的函数里，或让 Agent 任务在 message 中明确完成后如何通知；不要在业务表里自造通知队列。

## 当前用户和上下文

用户、部门、调用来源必须从 `ctx` 取：

```go
user := ctx.GetRequestUser()
dept := ctx.GetRequestUserDept()
traceID := ctx.GetTraceId()
fullCodePath := ctx.GetFullCodePath()
clientSource := ctx.GetClientSource()
```

常见用法：

- 创建人、提交人、评价人、收银员：后端用 `ctx.GetRequestUser()` 赋值。
- 当前部门默认值、部门过滤：用 `ctx.GetRequestUserDept()`。
- 日志串联：错误日志带 `ctx.GetTraceId()` 或直接用 `logger.Errorf(ctx, ...)`。
- 平台 API 来源：SDK 会从 `ctx` 生成 full_code_path 和 source_ref。

不要把创建人、提交人、操作人作为普通 Request 字段让用户填写；需要展示时放在 Table Model 中，通常 `hide:"create,update"`。

## SDK API 使用契约

只使用本轮已经读取过的 SDK 文档、案例或 SDK 源码中真实存在的类型、函数、方法和常量。不要根据命名直觉编造 `app.X`、`types.X`、`chart.X`、`callback.X`、`response.X` 等 SDK API。

规则：

- 如果需要使用新的 SDK 包、类型、函数、常量或结构体字段，必须先读取对应知识点文档、案例或 SDK 源码确认真实签名。
- 如果 build 报 `undefined: <sdk package>.<symbol>`，说明代码用了未确认的 SDK API；停止继续猜替代名字，回到对应文档或源码确认。
- 时间字段、空请求、图表类型、分页结构、文件处理、用户上下文等都属于独立知识点；当前已读文档没有覆盖时，不要自行推断。
- 示例代码只能复用已经验证过的 SDK 调用形态；不要把自然语言里的概念直接拼成 Go 选择器。

## 平台治理边界

通用权限、审批、评论、收藏和备份控制面不属于应用侧能力。业务代码不要为了补齐平台治理而自造通用模块。

- 不要默认给每个业务表加审批状态、审批人、审批意见、审批记录表。
- 不要在业务代码里自己造审批按钮、审批流、审批权限。
- 如果用户只是要管理“审批单据”这种业务对象，可以按普通 Table/Form 建模；这不等于平台通用审批。
- 如果用户要求“新增/修改/删除必须审批后执行”，应说明这是平台侧流程控制能力，MVP 暂不内置。
- 如果用户要求“每天/每周自动执行”，不要在业务应用里自造 cron；已明确是某个 Form 的内置运营逻辑时，可用 `FormTemplate.Schedules` 声明默认调度，否则交给 `automation_operator` 使用 timer-scheduler 创建函数任务或 Agent 任务。默认按平台部署时区执行，不要主动填写 `Timezone`。
- 如果用户要求“发消息/通知某人”，不要自建通知表或硬连具体渠道；应使用 `ctx.SendNotification` 交给 message-service。按部门通知暂不暴露。

操作日志：

- Table 更新链路已有平台记录。
- 业务上确实需要流水、操作记录、支付记录、投票记录时，可以建只读业务 Table。
- 不要把平台通用操作日志功能重复塞进每个业务系统。

评论、点赞、收藏属于平台统一交互能力，不要在业务系统里默认自造通用评论表或点赞表。

## Table 回调高级能力

Table 写操作由回调控制：

- `OnTableAddRow`：新增。
- `OnTableUpdateRow`：编辑和简单状态变更。
- `OnTableDeleteRows`：删除。
- 不配置某个回调，前端就没有对应写操作入口；三个回调都不配置，前端就不会出现新增、编辑、删除入口。

事实记录表默认只读：

- 收银记录、支付流水、消费记录、投票记录、评价记录、导入历史、操作记录等通常由 Form 或系统流程写入。
- 只读 Table 仍然建议显式配置 `AutoCrudTable`；它负责告诉前端用哪个 Model 渲染列表、搜索、分页和字段 schema。
- 这类表默认不配置 `OnTableAddRow`、`OnTableUpdateRow`、`OnTableDeleteRows`。
- 不要为了“看起来完整”给流水表补 CRUD；除非用户明确要求人工补录、修正或删除，并且 PRD 说明风险和约束。

更新回调要点：

```go
func UpdateRecord(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
    id := req.GetId()
    updates := req.ChangedFields()
    if req.IsFieldUpdated("status") {
        // 只在状态字段变更时做状态相关逻辑
    }
    oldValues := req.GetOldValues()
    _ = oldValues
    if err := ctx.GetGormDB().Model(&Record{}).Where("id = ?", id).Updates(updates).Error; err != nil {
        return nil, err
    }
    return &callback.OnTableUpdateRowResp{}, nil
}
```

规则：

- 简单审核、隐藏、回复、状态更新优先走 `OnTableUpdateRow`，不要额外拆 Form。
- `ChangedFields()` 只包含变化字段，适合直接传给 GORM `Updates`。
- `IsFieldUpdated(field)` 用于判断某字段是否真的变更。
- `BindChangedFields(&target)` 可把原始更新值绑定到结构体；未更新字段为零值，需要旧数据就查库或读 `GetOldValues()`。
- 删除和批量写入前要确认业务是否允许；流水、操作记录、事实记录类表通常只读。

## 事务和副作用顺序

涉及多表写入、余额、库存、票数、评分缓存、流水时必须事务化：

```go
err := db.Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&record).Error; err != nil {
        return err
    }
    if err := tx.Model(&Object{}).Where("id = ?", req.ObjectID).
        Updates(map[string]interface{}{"review_count": gorm.Expr("review_count + ?", 1)}).Error; err != nil {
        return err
    }
    return nil
})
if err != nil {
    logger.Errorf(ctx, "[SubmitReview] transaction failed, req=%+v, err=%v", req, err)
    return nil, fmt.Errorf("[系统错误]-[SubmitReview] 保存评价失败: %w", err)
}
```

副作用顺序：

- 数据库事务内：只做必须一致的数据库写入和计数更新。
- 事务后：构建 link、调用非强一致外部 API。
- 文件上传/下载失败要返回错误；输出文件 refs 生成后再写入记录或 Response。
- 如果外部 API 必须和数据库强一致，业务上要设计补偿或状态表，不要假装一次事务能覆盖外部系统。

## 日志和错误处理

错误分两类：

- 用户输入或业务规则错误：返回可读文案，例如“评价对象未开放评价”“库存不足”。
- 系统错误：需要排查，日志要带 req、关键 model、err；返回错误可带 `[系统错误]` 前缀。

示例：

```go
if err := db.Where("id = ?", req.ObjectID).First(&object).Error; err != nil {
    logger.Errorf(ctx, "[EvaluationSubmit] 查询评价对象失败, req=%+v, err=%v", req, err)
    return nil, fmt.Errorf("[系统错误]-[EvaluationSubmit] 查询评价对象失败: %w", err)
}
if object.Status != "开放" {
    return nil, fmt.Errorf("评价对象当前未开放评价")
}
```

规则：

- 不要吞错。
- 不要只返回“失败”。
- 系统错误日志要有足够上下文。
- 不要把敏感 token、完整大文件内容、隐私正文打进日志。
- `sensitive:"true"` 只表示平台操作日志会移除该字段；它不加密用户业务表。密码、token、API key 等如需落业务库，默认按普通字段明文存储；需要密文存储时由业务代码自行加密后写入。
- 不要在 widget 标签中生成 `password:true` 配置，SDK 不支持业务密码输入组件。

## Python 和外部处理

文件处理、图片、PDF、Excel、NLP 可能需要 Python 运行时。只有要沉淀成业务 SDK 函数时才写进应用；临时一次性脚本优先用工具链。

规则：

- 固定入口函数是 `kageos_entry(args, output_dir)`。
- Go 侧使用 `pythonRuntime.NewExecutor(...)` 后必须 `defer executor.Close()`。
- 需要输出文件时，Python 返回 `output_files`，Go 侧用 `ExecuteJSONWithResult`、`OutputFilePaths()` 校验，再用 `ctx.GetFS().ResponseFiles(...)` 下发。
- Go 和 Python 是同机子进程，但 cwd 可能不同；文件路径尽量用绝对路径。
- 不要用 base64 大字段返回文件内容，输出文件走 files。

## 构建和启动校验

写完必须验证：

1. `build_workspace`。
2. 启动期 `CompileAndValidate()` 会检查路由后缀、Template 类型、schema、widget、validate 和筛选字段。
3. 根据函数类型继续验证：
   - Form：`run_form_submit`
   - Table：`run_table_search`，有写能力时验证 create/update/delete/
   - Chart：`run_chart_query`
   - OnSelectFuzzy：`run_on_select_fuzzy`

不要只落盘不编译；build 失败时先读完整错误，按 Go 编译错误、SDK schema 错误、runtime 启动错误、业务执行错误的顺序修。
