# SDK Table/Form 组合任务包

本文档用于 Table + Form 场景：Table 管理一个长期对象或记录集合，Form 承担另一个用户侧一次性提交动作，但暂时不需要 Chart。它是一个闭环任务包：说明什么时候拆 Table/Form、前端长什么样、函数如何连接、事务和只读记录怎么处理、如何验证。字段组件统一读取 `/system/prompt/sdk/widget-system`，基础 Table CRUD 读 `/system/prompt/sdk/table-crud-basic`，单 Form 写法读 `/system/prompt/sdk/form-submit-basic`。

## 什么时候用 Table + Form

用户需求同时包含以下两类能力时，使用 Table + Form：

- 需要长期保存、搜索、分页和维护的一批业务记录。
- 需要一个独立动作入口：提交评价、提交投票、报名、导入、快速跟进、复杂派单、批量处理。

前端形态：

- Table：Element Plus `el-table` 风格的数据表格，展示长期记录、搜索、分页、列和操作入口。
- Form：Element Plus `el-form` 风格的提交界面，收集一次动作所需输入，提交后展示结果、文件或 link。

如果只是单表新增/编辑，优先用 Table 回调，不要为了新增一条记录额外建 Form。简单状态流转也优先放在 Table update 回调中；只有动作面向不同使用者、需要独立入口、复杂校验、跨表写入、消息通知、文件导入或批量处理时，才拆 Form。

## 典型结构

```text
/用户/应用/evaluation
  evaluation_object_list.table   管理被评价对象
  evaluation_submit.form         用户提交评价
  evaluation_record_list.table   查看评价记录
```

常见命名：

- 主记录：`xxx_list.table`
- 用户侧提交：`xxx_submit.form`
- 动作入口：`xxx_import.form`、`xxx_followup.form`、`xxx_assign.form`
- 动作记录：`xxx_record_list.table`，通常只读

## 职责边界

Table 负责：

- 长期业务记录的列表、搜索、分页、详情展示。
- 简单新增、编辑、删除。
- 主数据维护，例如客户、商品、主题、工单、导入记录。

Form 负责：

- 一次业务动作，例如提交评价、提交投票、快速跟进、导入 Excel。
- 复杂校验，例如对象是否开放评价、是否重复提交、时间窗口、余额/库存校验。
- 跨表写入，例如写主记录 + 写流水 + 更新状态。
- 返回结果、文件或跳转 link。

不要让 Table 承担复杂动作，也不要让 Form 承担分页列表。

## 组合拆分流程

1. 找主对象：哪些记录需要长期保存，先建 Table。
2. 找动作入口：哪些操作不是简单新增/编辑/状态更新，而是用户侧一次性提交，拆成 Form。
3. 找动作结果：是否需要写入记录表或只读流水表。
4. 找连接方式：Table 行上是否需要 link 跳到 Form，Form 结果是否需要 link 回 Table。
5. 找验证路径：至少验证一个 Table 查询和一个 Form 提交。

## 组合必备 SDK 能力

Table + Form 不是“一个表格加一个表单”这么简单。必须把对象选择、记录查询、跳转和展示场景一起设计，否则用户无法从列表进入提交动作，也无法在提交后看到记录。

### Table 的自定义搜索参数

对象表和记录表都可能需要超出主表字段的搜索：

- 评价记录按评价对象筛选：优先让 `object_id` 使用 `select + OnSelectFuzzy`，前端显示“评价对象”，用户按对象名称搜索，实际提交 ID。
- 跟进记录按客户行业搜索：Request 增加 `CustomerIndustry`，Build 前跨表过滤。
- 导入历史按文件名、状态、导入人搜索：筛选字段写在 Request，Build 前手写 Where；复杂统计字段同样用显式参数。

写法：Request 嵌入 `query.PageSortReq`，业务筛选字段全部显式声明；Build 前手写 `Where` / `Joins` / `Preload`，再交给 `Table`。Table 新增/编辑也需要外键选择时，外键字段必须在 Table Model 上是 `type:select` / `type:multiselect`，并在同一个 `TableTemplate.BaseConfig.OnSelectFuzzyMap` 注册同名 key；如果只用于列表筛选，也可以放在 Request 中并注册对应的筛选控件。

### GORM Preload 和后置关联填充

记录表通常要展示关联对象名称，但数据库里只存 ID：

```go
queryDB := db.Model(&EvaluationRecord{}).Preload("Object")
var records []EvaluationRecord
if err := resp.Table(response.TableResult{Items: records, TotalCount: total, PageInfo: &req.PageSortReq}).Build(); err != nil {
    return err
}
for i := range records {
    if records[i].Object != nil {
        records[i].ObjectName = records[i].Object.Name
    }
}
```

如果要展示平均分、评价次数、最近提交时间等聚合字段，在 Build 后收集当前页对象 ID，批量查询统计 map，再填充 `gorm:"-"` 字段。不要在循环里每行查一次。

### hide 标签

组合场景更需要 `hide` 控制前端：

- 对象表：ID、创建时间、评价次数、平均分、提交评价 link 用 `hide:"create,update"`。
- 评价 Form：Request 只放用户要填的对象、评分、内容、附件，不放列表字段。
- 记录表：评价内容、评分、提交人、状态、管理员回复可列表展示；GORM 关联对象用 `widget:"-"`。
- 只读计算字段、关联名称、link 都必须 `gorm:"-"` + `hide:"create,update"`。

### link 串联 Table 和 Form

对象 Table 行上放 link 进入提交 Form：

```go
objects[i].SubmitLink, _ = ctx.BuildFunctionUrlWithText(
    "evaluation_submit.form",
    EvaluationSubmitReq{ObjectID: objects[i].ID},
    "提交评价",
)
```

Form 提交成功后返回 link 到记录 Table：

```go
recordLink, _ := ctx.BuildFunctionUrlWithText(
    "evaluation_record_list.table",
    EvaluationRecord{ObjectID: req.ObjectID},
    "查看评价记录",
)
```

不要把 link 字段落库；它是前端导航字段，Build 后或 Form Response 中填充。

### Form 选择 Table 数据

Form 选择对象时，用 OnSelectFuzzy，而不是静态 options：

```go
ObjectID int `json:"object_id" widget:"name:评价对象;type:select" validate:"required" callback:"OnSelectFuzzy"`
```

`OnSelectFuzzyMap` 的 key 必须等于字段 `json` 名。回调要支持：

- keyword：用户输入关键词时搜索候选项。
- by_value：表单预填或回显单个已选值。
- by_values：multiselect 回显多个已选值。

联动多选用 `depend_on`，例如投票主题决定可选投票选项。

当前请求上下文能确定的身份字段不要放进 Form Request。比如竞拍出价的出价人、投票提交的投票人、评价提交人、跟进记录创建人等，应在 Handler 中用 `ctx.GetRequestUser()` 赋值；需要部门时用 `ctx.GetRequestUserDept()`。只有业务明确允许代填/指定他人时，才把这类字段暴露给用户填写。

Table 记录筛选也可以在 Request 字段上使用 OnSelectFuzzy。例如评价记录表中：

```go
ObjectID int `json:"object_id" form:"object_id" widget:"name:评价对象;type:select" callback:"OnSelectFuzzy"`
ObjectName string `json:"object_name" gorm:"-" widget:"name:评价对象;type:text" hide:"create,update"`
```

注意：筛选回调和提交回调的业务过滤可能不同。`evaluation_submit.form` 只返回开放评价的对象；`evaluation_record_list.table` 查询历史记录时应返回所有对象，包括已关闭对象，否则历史记录无法按已关闭对象筛选。

### 文件上传下载链路

Table + Form 场景常见两类文件：

- Form 输入文件：用户提交评价附件、上传 Excel 导入、上传简历。
- Table 长期附件：评价记录附件、导入原始文件、导入结果文件、客户资料附件。

规则：

1. Form Request 的文件字段用 `type:files` + `string`，通过 `fs.DownloadFiles(req.InputFiles)` 下载到本地处理。
2. Form 处理后生成文件，写入 `fs.GetTraceOutputDir()`，再用 `fs.ResponseFiles` / `fs.ResponseDirFiles` 返回或落库。
3. 如果文件只是作为附件保存到记录表，不需要下载，直接把前端提交的 refs 字符串写入 Table Model 的 files 字段。
4. Table 中长期附件字段也用 `type:files` + `string`，列表和详情由前端提供下载/预览。
5. 导入类 Form 通常还要写 `import_record_list.table`，保存原始文件 refs、结果文件 refs、成功数、失败数、错误摘要。

```go
type EvaluationSubmitReq struct {
    ObjectID   int    `json:"object_id" widget:"name:评价对象;type:select" validate:"required" callback:"OnSelectFuzzy"`
    Attachment string `json:"attachment" widget:"name:评价附件;type:files;accept:image/*,.pdf;max_size:20MB;max_count:5"`
}

type EvaluationRecord struct {
    Attachment string `json:"attachment" gorm:"column:attachment;type:text" widget:"name:评价附件;type:files;accept:image/*,.pdf;max_size:20MB;max_count:5"`
}
```

只有需要解析附件内容时才下载；普通附件保存直接存 refs。

### 记录表默认只读

Table + Form 组合里，很多 Table 是 Form 动作产生的事实记录，例如评价记录、投票记录、导入历史、跟进流水。默认规则：

- Form 负责写入事实记录。
- Table 负责搜索、分页、查看和必要的只读展示。
- 只读记录表仍然建议显式配置 `AutoCrudTable`，前端需要它来渲染 Element 表格 schema、搜索、分页和列表列。
- 不配置 `OnTableAddRow`、`OnTableUpdateRow`、`OnTableDeleteRows` 时，前端不会出现新增、编辑、删除入口。
- 只有用户明确要求“管理员可修正/补录/删除记录”，才添加对应回调，并在 PRD 里说明风险。
- PRD 不能一边写“默认只读”，一边在列表样例里承诺“通过/隐藏/回复/删除”等操作；如果要做这些受控修改，PRD 必须明确 `evaluation_record_list.table` 会配置 `OnTableUpdateRow`，实现时也必须写该回调。

例如 `evaluation_record_list.table` 默认只读；用户提交评价通过 `evaluation_submit.form` 写入记录，管理员不应在表格里手工新增评价记录。审核、隐藏、回复如果只是内容处理状态，可以按业务要求配置 `OnTableUpdateRow`；但新增和删除仍应谨慎。

## 评价系统

评价系统是最典型的 Table + Form：后台管理被评价对象，用户前台提交一次评价，管理员再查看评价记录。

- `evaluation_object_list.table`：评价对象管理。前端是 Element 表格，管理员维护课程、商品、服务、活动或人员等被评价对象，展示名称、分类、负责人、状态、开放评价时间、评价次数等列；支持按状态、分类、关键词搜索。它承载“哪些东西可以被评价”。
- `evaluation_submit.form`：提交评价。前端是 Element 表单，用户选择评价对象，填写评分、评价标签、评价内容、附件、匿名标记后提交；后端校验对象是否开放评价、是否允许重复评价，然后写入评价记录。它是一次用户侧提交动作，所以是 Form，不是 Table 新增。
- `evaluation_record_list.table`：评价记录管理。前端是 Element 表格，管理员查看评价对象、评分、内容、标签、提交人、提交时间、状态、管理员回复等；可按对象、评分、状态、时间搜索。审核状态、管理员回复这类简单修改可以直接走 Table update 回调。

职责边界：

- Table 管评价对象和评价记录这些长期数据。
- Form 只执行“提交一次评价”。
- 简单审核、隐藏、回复可以用 `evaluation_record_list.table` 的更新回调，不需要额外拆 Form。
- 如果用户要求评分趋势、评分分布、标签占比，再升级到 Table/Form/Chart，增加 Chart。

## 投票系统（无 Chart 版本）

当用户只要求管理投票并提交投票，但没有明确要求可视化图表时，也可以做 Table + Form：

- `vote_topic_list.table`：投票主题管理。前端是 Element 表格，管理员维护主题标题、说明、开始时间、结束时间、状态、匿名/多选规则；支持按状态、时间和关键词搜索。
- `vote_option_list.table`：投票选项管理。前端是 Element 表格，按主题筛选选项，展示选项内容、排序、启用状态和当前票数。
- `vote_submit.form`：用户投票动作。前端是 Element 表单，用户选择主题和选项后提交；后端校验投票时间、重复投票、多选数量，然后写入投票记录。
- `vote_record_list.table`：投票记录查询。前端是只读 Element 表格，展示用户、主题、选项、投票时间，用于审计和排查。
- `vote_result.form`：普通结果查看。前端是表单或结果页，按主题返回文本化统计、选项票数和跳转链接；如果用户要柱状图/饼图，再升级成 `vote_result.chart`。

职责边界：

- Table 管主题、选项、记录。
- Form 执行“投一票”和“查看普通结果”。
- 不需要图表时不要硬加 Chart；需要可视化时再读 Table/Form/Chart 组合包。

## 客户管理（客户 + 跟进）

- `customer_list.table`：客户档案。前端表格展示客户名称、行业、等级、联系人、负责人、状态和下次跟进时间。
- `contact_record_list.table`：跟进记录。前端表格展示客户、跟进方式、内容、下次跟进时间和创建人。
- `followup_create.form`：快速跟进。前端表单从客户列表 link 进入，填写跟进内容、方式和下次跟进时间；后端写入跟进记录并更新客户最近跟进字段。

职责边界：

- 客户和跟进记录长期存在，用 Table。
- “快速跟进”是一次动作，用 Form。
- 客户列表可以用 link 跳到快速跟进 Form，并带上客户 ID。

## 导入记录（Form + Table）

- `excel_import.form`：上传 Excel 并导入业务数据。前端表单上传文件、选择导入模式；后端解析、校验、写入数据。
- `import_record_list.table`：导入历史。前端只读表格展示文件名、导入人、成功数、失败数、状态、错误摘要和时间。

职责边界：

- 导入动作是 Form。
- 导入历史是 Table。
- 如果导入生成了业务主数据，再额外建对应主数据 Table。

## link 连接规则

Table/Form 之间通过 `link` 字段连接：

- Table 行跳 Form：例如“提交评价”“提交投票”“快速跟进”“重新导入”。
- Form 结果跳 Table：例如“查看评价记录”“查看导入记录”“查看投票记录”。

实现时使用 `ctx.BuildFunctionUrlWithText`，不要手拼 URL。link 字段通常 `gorm:"-"`，只在 Response 或列表展示中填充。

## 事务规则

Form 如果写多张表或影响关键状态，必须事务化：

- 提交评价：写评价记录 + 更新评价对象的评价次数或平均分缓存。
- 投票提交：写投票记录 + 更新选项票数。
- 快速跟进：写跟进记录 + 更新客户最近跟进时间。
- Excel 导入：批量写业务记录 + 写导入历史。

事务失败时返回错误，不要只写一半数据。

## PRD 中怎么描述 Table + Form

创建类 PRD 必须写“函数类型判断”：

```text
函数类型判断：
- 评价对象需要长期维护，选择 `evaluation_object_list.table`，前端以 Element 表格展示对象列表、状态筛选、分页和新增/编辑入口。
- 用户提交评价是一次性动作，选择 `evaluation_submit.form`，前端以 Element 表单收集评价对象、评分、标签、内容和附件，提交后写入评价记录。
- 评价记录需要后台查看和处理，选择 `evaluation_record_list.table`，前端以 Element 表格展示评价内容、评分、提交人、状态和管理员回复；简单审核/回复走 Table update 回调。
- 当前需求没有统计看板，因此暂不创建 Chart。
```

不要只列函数名，要写清楚每个函数在前端呈现什么、承担什么职责、如何连接。

PRD 还必须写“落地目录和函数清单”，让用户知道确认后会创建什么：

```text
落地目录和函数清单：
- 创建目录：/用户/应用/evaluation
- evaluation_object_list.table：评价对象管理，前端是 Element 表格，支持搜索、分页、新增、编辑、关闭评价。
- evaluation_submit.form：提交评价，前端是 Element 表单，用户填写评分、评价内容、附件并提交。
- evaluation_record_list.table：评价记录管理，前端是 Element 表格，默认只读展示评价记录；如需审核/回复，只开放受控 update 回调。
```

PRD 必须包含示例数据；示例数据要覆盖对象表、提交表单和提交后生成的记录表：

| 表/表单 | 样例 |
| --- | --- |
| `evaluation_object_list.table` | 对象名称：课程 A；分类：课程；负责人：张三；状态：开放；开放评价时间：2026-05-04 09:00；评价次数：12；平均分：4.8 |
| `evaluation_submit.form` | 评价对象：课程 A；评分：5；评价标签：讲解清晰、响应及时；评价内容：老师讲得很清楚，资料也完整；附件：课堂截图 |
| `evaluation_record_list.table` | 评价对象：课程 A；评分：5；评价内容：老师讲得很清楚；提交人：李四；状态：待审核；管理员回复：空；提交时间：2026-05-04 10:00 |

结尾确认语前必须写清下一步：

```text
确认后我将创建目录：/用户/应用/evaluation，并生成：
- evaluation_object_list.table
- evaluation_submit.form
- evaluation_record_list.table
```

## 验证

写完后必须：

1. `build_workspace`。
2. `run_table_search` 验证核心 Table 能查询。
3. `run_form_submit` 验证核心 Form 能提交。
4. 如果 Form 写入 Table，再次 `run_table_search` 验证数据变化。
5. 如果有 link 或 OnSelectFuzzy，使用对应 run 工具验证。

复杂动作要至少验证一条完整链路，例如“评价对象列表 -> 提交评价 -> 评价记录出现”。

## 推荐案例

- 投票/提交类系统：`/system/prompt/case_catalog/formandtable/vote`
- 招聘主从表和 link：`/system/prompt/case_catalog/tables/hr`
- Excel/CSV 导入处理：`/system/prompt/case_catalog/form/excelorcsv`
- Table 基础 CRUD：`/system/prompt/case_catalog/table/ticket`

## 常见错误

- 用户只要简单 CRUD，却拆出多余 Form。
- 有用户侧提交/导入动作，却硬塞进 Table 新增或编辑。
- 简单状态修改本来可以走 Table update 回调，却额外拆出 Form。
- Form 写入数据后没有记录表或没有 link 回列表。
- 只建 Form，不建长期记录 Table，导致用户看不到历史数据。
- 把统计图表放进 Form；需要图表时应升级到 Table/Form/Chart。
- 没有事务，跨表写入只成功一半。
- 半成品阶段过早 build，浪费构建成本并制造误导错误。
