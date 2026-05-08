# SDK Table/Form/Chart 组合任务包

本文档用于复杂业务系统建模：一个系统中同时出现长期记录管理、一次性业务动作和统计分析。它不替代单个函数写法文档，而是说明如何拆目录、拆职责、建立函数之间的协作关系。字段组件统一读 `/system/prompt/sdk/widget-system`，基础 Table CRUD 读 `/system/prompt/sdk/table-crud-basic`。

## 总原则

复杂系统不要写成一个大函数，也不要手写独立页面。拆成一组函数：

```text
Table 管长期数据
Form 做一次动作
Chart 看统计结果
```

前端表现：

- Table：Element 表格列表，展示、搜索、分页和维护长期记录。
- Form：Element 表单提交，执行一次业务动作。
- Chart：筛选条件 + ECharts 图表，展示统计结果。

## 什么时候使用组合包

使用本文档的典型需求：

- “做一个投票系统，可以维护投票、用户投票、查看结果”
- “做一个收银系统，有商品会员、收银结算、销售统计”
- “做一个评价系统，有评价对象、提交评价、评价记录和评分统计”
- “做一个客户管理系统，有客户档案、跟进记录、销售漏斗”
- “做一个库存系统，有商品库存、入库出库动作、库存预警统计”

如果用户只需要“管理一批记录”，先用 `/system/prompt/sdk/table-crud-basic`；如果只需要“上传文件处理一次”，先用 `/system/prompt/sdk/form-submit-basic`；如果是长期记录 + 独立动作但不需要统计，先用 `/system/prompt/sdk/combo-table-form`；如果只需要“统计一张图”，先按 Chart 短参考实现。

## 组合拆分流程

1. 找主业务对象：哪些东西需要长期保存，先建 Table。
2. 找一次性动作：哪些操作会改变多张表、触发校验、生成流水或发送通知，建 Form。
3. 找统计问题：哪些指标需要趋势、占比、汇总，建 Chart。
4. 找跳转关系：Table 列表是否需要 link 到 Form 或 Chart。
5. 找权限和横切能力：权限、审批、消息、操作日志等优先用平台能力，不在业务代码里重造。
6. 完整实现后统一构建：先写完 Table 主数据、Form 动作、Chart 统计和联动代码，再统一 build 和验证。

## 组合必备 SDK 能力

Table/Form/Chart 组合包必须同时覆盖表格、表单、图表之间的数据流。不要只建三个函数名，要把搜索、关联展示、跳转、动态选择和图表筛选都写出来。

复杂系统通常会拆多个 `.go` 文件。共享数据库 Model 在同一个 package 中只能定义一次，后续文件直接复用；不要在列表文件和表单文件里重复定义同名 struct。Handler 函数名也不要和 Model 类型名完全相同，例如有 `type AuctionBid struct` 时，提交处理函数应命名为 `AuctionBidSubmit` 或 `AuctionBidHandler`。

### Table Request 自定义搜索参数

主数据表、流水表、记录表都要支持真实查询路径：

- 主表字段：在 Request 中显式声明筛选字段，Build 前手写 `Where`。
- 关联字段：在 Request 中新增筛选参数，Build 前用 `Joins` / `Where`。
- 统计缓存字段：如果是 `gorm:"-"` 计算字段，必须在 Request 中自定义筛选字段并手写 SQL。

Request 一般这样组织：

```go
type PaymentRecordListReq struct {
    MemberName string `json:"member_name" form:"member_name" widget:"name:会员名称;type:input"`
    Status     string `json:"status" form:"status" widget:"name:状态;type:select;options:成功,失败"`
    query.PageSortReq `widget:"-"`
}
```

`query.PageSortReq` 只负责分页和排序；`Table` 只处理 Count、排序、Offset、Limit、Find 和分页信息。需要按外键筛选时，可以在 Request 中声明外键筛选字段并手写 Where；如果新增/编辑弹窗也需要选择该外键，则 Model 外键字段仍要配置 `widget:"type:select"` 或 `type:multiselect`、`callback:"OnSelectFuzzy"`，并在 `OnSelectFuzzyMap` 注册该字段。

### GORM Preload 和后置关联填充

Table 列表先负责拿到当前页数据，再填充前端展示字段：

- Build 前 `Preload("Member")`、`Preload("Product")`，用于标准 GORM 关联。
- Build 后把关联名称、link、计算状态、平均值、占比等填到 `gorm:"-"` 字段。
- 聚合字段用当前页 ID 批量查询 map 后回填，避免 N+1。

这一步对 Chart 也有价值：Chart 聚合通常直接从主表/流水表查，不依赖 Table Response；但 Table 中展示的“查看统计” link 要在 Build 后填充。

### hide 标签

复杂系统字段多，必须用 hide 控制界面复杂度：

- 系统字段、link、计算字段、关联展示名：`hide:"create,update"`。
- Form 专用明细输入，例如收银商品明细子表：放在 Form Request，不要塞进 Table 列表。
- Table 新增时才需要的嵌套子表：可用 `hide:"list,update"`。
- GORM 关联对象、内部中间字段：`widget:"-"`。

没有 hide 规划时，前端会把大量不该编辑的字段暴露在新增/编辑表单里。

### link 串联 Table、Form、Chart

复杂系统必须有导航关系：

- 主数据 Table 行跳 Form：参与投票、收银结算、快速跟进。
- 主数据 Table 行跳 Chart：查看该主题结果、查看客户统计。
- Form 结果跳 Table：查看流水、查看记录。
- Form 结果跳 Chart：查看本次动作相关统计。

统一用 `ctx.BuildFunctionUrlWithText`。跳 Table 时参数会按目标 Table 筛选规则转换；跳 Form/Chart 时参数会按目标 Request 预填。

### Form 动态选择和事务

Form 中选择商品、会员、投票主题、评价对象、客户时，使用 OnSelectFuzzy。Table 筛选区按外键筛选记录时也优先使用 OnSelectFuzzy，例如支付流水按会员、评价记录按评价对象筛选；前端显示业务名，实际提交 ID。依赖其他字段时用 `depend_on`。

当前请求上下文能确定的身份字段不要放进 Form Request。比如竞拍出价的出价人、投票提交的投票人、评价提交人、跟进记录创建人、收银员等，应在 Handler 中用 `ctx.GetRequestUser()` 赋值；需要部门时用 `ctx.GetRequestUserDept()`。只有业务明确允许代填/指定他人时，才把这类字段暴露给用户填写。

复杂 Form 只要写多张表或影响余额、库存、票数、状态，必须事务化。事务内部做写入和计数更新，事务外再构建 Response link。

OnSelectFuzzy 的模板 key 必须对应真实的 `select` / `multiselect` 字段。不要给 `type:ID`、`type:input` 或已经移除的 Request 字段注册回调；这类错误会在 schema 启动校验阶段失败。

### 文件上传下载链路

复杂系统里的文件能力要按数据生命周期设计：

- Form 临时输入文件：导入 Excel、上传凭证、上传评价附件。需要处理内容时用 `fs.DownloadFiles` 下载，处理完 `fs.RemoveFiles`。
- Table 长期附件字段：商品图片、会员附件、评价附件、支付凭证、导入原始文件。直接保存前端提交的 files refs，Go 类型 `string`。
- 输出文件：导入错误报告、销售报表导出、处理结果文件。写到 `fs.GetTraceOutputDir()`，再用 `fs.ResponseFiles` / `fs.ResponseDirFiles` 得到 refs。
- Chart 不返回文件；如果用户要导出报表文件，应新增 Form 导出动作或 Table 记录，不要让 Chart 混入 files Response。

```go
type ImportResp struct {
    ResultFiles string `json:"result_files" widget:"name:导入结果文件;type:files"`
    RecordLink  string `json:"record_link" widget:"name:查看导入记录;type:link;target:_blank"`
}

func buildImportOutput(ctx *app.Context, outputPath string) string {
    return ctx.GetFS().ResponseFiles([]string{outputPath})
}
```

事务只包数据库写入；文件上传/下载失败要返回错误，输出文件 refs 生成后再写入记录或 Response。

### Chart Request 和聚合

Chart 的 Request 是筛选条件，不嵌入分页请求：

```go
type SalesTrendReq struct {
    StartTime types.Time `json:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
    EndTime   types.Time `json:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
    MemberID  int        `json:"member_id" widget:"name:会员;type:select" callback:"OnSelectFuzzy"`
}
```

图表 Handler 里手写聚合 SQL 或 GORM 查询，组装当前已读文档或源码中确认存在的图表响应结构，再 `resp.Chart(c).Build()`。业务代码只填该结构真实存在的字段，不要给 `Series`、图表类型或 Template 增加未确认字段。

一个 `.chart` 路由只返回一张图；多个图表拆多个路由。

如果一个系统要展示成交趋势、热门排行、状态占比这类多张图，必须拆成多个 `.chart` 路由；不要为了把多张图塞进一个路由而编造组合图表类型或 Template 枚举。

`types.Time` 是 SDK 时间类型，做格式化、比较或日期循环时先调用 `Time()` 方法：`req.StartTime.Time().Format(...)`、`deadline.Time().After(now)`。不要写 `req.StartTime.Format(...)` 或 `req.StartTime.Time.Format(...)`。

需要时间处理、空请求、图表结构、分页结构或文件处理时，只能使用当前已读知识点里出现过的写法；没读到就先读对应文档或源码，不能按命名规则拼 SDK API。

## PRD 必写的函数类型判断

复杂系统 PRD 开头必须写：

```text
函数类型判断：
- 需要长期保存和管理的业务对象：xxx，使用 Table，前端以 Element 表格展示列表、搜索、分页和操作入口。
- 需要一次性执行的业务动作：xxx，使用 Form，前端以表单收集输入，后端校验并写入/更新数据。
- 需要统计展示的指标：xxx，使用 Chart，前端以筛选条件 + 图表展示结果。
- 函数之间通过 link 或参数跳转连接，不手写独立页面。
```

PRD 还必须写：

- 落地目录和函数清单：例如“确认后创建 `/用户/应用/cashier`”，并列出每个 `.table`、`.form`、`.chart` 的前端形态和职责。
- 示例数据：至少覆盖一个主数据 Table 样例、一条业务动作 Form 输入样例、一条事实记录 Table 样例、一组 Chart 聚合结果样例。
- 确认后创建内容：在确认语前写清“确认后我将创建目录：xxx，并生成：a.table、b.form、c.chart”。

## Table + Form 组合

适用于“后台管理 + 独立动作入口”：

- Table 管长期记录。
- Form 执行动作，可能创建记录、更新状态、写流水、发送消息。
- Table 列表中可以放 link 跳到 Form，并带上当前行参数。

示例：评价系统

- `evaluation_object_list.table`：评价对象列表，前端表格展示对象名称、分类、状态、开放时间、评价次数等。
- `evaluation_submit.form`：提交评价表单，选择评价对象、评分、标签、内容和附件后提交。
- `evaluation_record_list.table`：评价记录表，前端表格展示对象、评分、内容、提交人、状态和管理员回复。
- `evaluation_object_list.table` 的行 link 可以跳 `evaluation_submit.form` 并预填对象 ID。

什么时候不用 Form：

- 只是直接新增/编辑/审核/回复单条记录时，用 Table 的新增/编辑回调即可。
- 只有当动作面向不同使用者、需要独立入口、复杂校验、跨表事务、消息通知或批量处理时，才拆 Form。

## Table + Chart 组合

适用于“后台管理 + 统计分析”：

- Table 提供长期业务数据。
- Chart 基于 Table 数据聚合。
- Table 列表或目录中可提供 link 跳到 Chart，并传筛选参数。

示例：工单统计

- `ticket_list.table`：工单表格。
- `ticket_status_chart.chart`：按状态统计工单数量。
- `ticket_duration_chart.chart`：按时间统计平均处理时长。

规则：

- 一个 Chart 路由只返回一张图。
- 多个图表拆多个 `.chart`。
- Chart 不负责 CRUD。
- 临时给一份 Excel 画图不是 Chart 系统函数，优先用 Form 工具。

## Table + Form + Chart 组合

适用于完整业务系统：

```text
主数据 Table
业务动作 Form
流水/记录 Table
统计分析 Chart
```

### 投票系统

- `vote_topic_list.table`：投票主题管理。前端是 Element 表格，管理员维护主题、时间、状态、匿名/多选规则；支持按状态、时间和关键词搜索。
- `vote_option_list.table`：投票选项管理。前端是表格，按主题筛选选项，展示选项名称、排序、票数、占比、启用状态。
- `vote_submit.form`：用户投票动作。前端是表单，用户选择主题和选项后提交；后端校验投票时间、重复投票、多选数量，然后写入投票记录。
- `vote_record_list.table`：投票记录查询。前端是只读表格，展示用户、主题、选项、投票时间，用于审计和排查。
- `vote_result.chart`：投票结果图表。前端按主题筛选后展示选项票数、占比或趋势。

职责边界：

- Table 不执行“投票”动作，只管理主题、选项、记录。
- Form 不维护选项列表，只提交一次投票。
- Chart 不写数据，只读投票记录并聚合。
- 主题列表可以用 link 跳到 `vote_submit.form` 或 `vote_result.chart`。

### 收银系统

- `product_list.table`：商品管理。前端表格展示名称、分类、售价、库存、条码、状态，支持新增和编辑。
- `member_list.table`：会员管理。前端表格展示会员、手机号、等级、余额、积分、状态。
- `cashier_checkout.form`：收银结算。前端表单包含商品明细子表、会员选择、优惠、支付方式、实收金额；后端事务化扣库存、更新会员余额/积分、生成流水。
- `payment_record_list.table`：支付流水。前端只读表格展示订单号、会员、金额、支付方式、支付时间、收银员。
- `sales_trend.chart`：销售趋势图。前端按时间范围筛选，展示折线图或柱状图。
- `category_sales.chart`：品类销售占比。前端展示饼图或柱状图。
- `member_consumption.chart`：会员消费分析。前端展示消费排行或复购趋势。

职责边界：

- 商品和会员是主数据，用 Table。
- 收银结算是一次复杂动作，用 Form，不要写成 Table 新增。
- 支付流水、收银记录是事实记录，默认只读 Table；不配置 `OnTableAddRow`、`OnTableUpdateRow`、`OnTableDeleteRows`，前端就不会出现新增、编辑、删除入口。
- Chart 基于流水做经营分析。

### 客户管理系统

- `customer_list.table`：客户档案，Element 表格展示客户名称、行业、等级、联系人、负责人、状态。
- `contact_record_list.table`：跟进记录，Element 表格展示客户、跟进方式、内容、下次跟进时间。
- `followup_create.form`：快速跟进动作，表单提交一次跟进内容，可从客户列表 link 进入。
- `customer_level_chart.chart`：客户等级分布。
- `sales_funnel.chart`：销售漏斗。

职责边界：

- 客户和跟进记录长期存在，用 Table。
- 快速跟进是动作入口，用 Form。
- 等级分布和销售漏斗是统计，用 Chart。

## link 连接规则

Table/Form/Chart 之间通过 `link` 字段连接：

- Table 行跳 Form：例如“提交评价”“参与投票”“快速跟进”。
- Table 行跳 Chart：例如“查看该主题结果”“查看客户统计”。
- Form 结果跳 Table：例如提交后返回“查看记录”。

实现时用 `ctx.BuildFunctionUrlWithText`，不要手拼 URL。

## 事实记录表只读规则

完整业务系统里通常同时存在主数据表和事实记录表：

- 主数据表：商品、会员、客户、评价对象、投票主题。通常可以按业务要求配置新增、编辑、删除回调。
- 事实记录表：收银记录、支付流水、消费记录、投票记录、评价记录、导入历史、审计记录。默认只读。

事实记录表的写入来源应是业务动作 Form 或系统流程，而不是表格手工新增。例如收银系统中：

- `cashier_checkout.form` 事务化扣库存、更新会员、生成收银记录和支付流水。
- `payment_record_list.table` 只负责展示和搜索流水。
- `payment_record_list.table` 仍然建议显式配置 `AutoCrudTable`，让前端按支付流水 Model 渲染列表、筛选和分页。
- `payment_record_list.table` 不配置 `OnTableAddRow`、`OnTableUpdateRow`、`OnTableDeleteRows`；前端不会显示新增、编辑、删除入口。

只有用户明确要求人工补录或冲正，才单独设计补录/冲正 Form 或受控 Table update；不要默认开放流水删除。

PRD 和实现必须一致：事实记录表如果写“默认只读”，列表样例里不要承诺“编辑/删除/审核/退款/冲正”等操作；如果业务确实需要某个受控动作，必须在 PRD 中说明用独立 Form、link，或明确配置哪个 Table 回调实现。

## 事务规则

Form 如果执行跨表写入，必须考虑事务：

- 收银结算：扣库存 + 写支付流水 + 更新会员余额/积分。
- 投票提交：写投票记录 + 写投票明细 + 更新计数。
- 提交评价：写评价记录 + 更新评价次数或平均分缓存。

这类动作不要放在 Table 新增回调里硬做，除非它本质就是单表新增。复杂动作拆独立 Form 更清晰。

## 完整实现和验证

复杂系统不要写一部分就提前 build。建议顺序：

1. 建主数据 Table、动作 Form、统计 Chart、link 和联动所需的全部 Go 文件。
2. 写入/替换工具如果失败，先修工具参数或源码，不要继续假设文件已存在。
3. 全部落盘后统一 `build_workspace`。
4. build 成功后用 `run_table_search`、`run_form_submit`、`run_chart_query` 验证核心路径。
5. 若涉及写操作，至少验证一条完整业务链路。

## 推荐案例

- Table + Form：`/system/prompt/case_catalog/formandtable/vote`
- Table + Form + Chart：`/system/prompt/case_catalog/form_table_chart/cashier`
- 多 Table 和 link：`/system/prompt/case_catalog/tables/meeting`
- 招聘主从表：`/system/prompt/case_catalog/tables/hr`

## 常见错误

- 用户要一个系统，模型只建一个大 Form，没有列表和长期记录管理。
- 用户要收银结算，模型写成 Table 新增，缺少事务和明细处理。
- 用户要统计，模型把 chart 结构塞进 Form Response。
- 多个图表塞进一个 Chart 路由。
- 编造未在已读 SDK 文档或源码中确认存在的图表类型、Template 枚举或 `Series` 字段。
- 在多个文件重复定义同一个数据库 Model，或让 Handler 函数名和 Model 类型名冲突。
- Table Request 未嵌入 `query.PageSortReq` 导致字段冲突诊断；筛选字段应写在 Request 中，Handler 里显式转成真实查询条件。
- Table、Form、Chart 都建了，但 PRD 没说明各自前端呈现和职责边界。
- 没有 link，导致函数之间孤立。
- 半成品阶段过早 build，浪费构建成本并制造误导错误。
