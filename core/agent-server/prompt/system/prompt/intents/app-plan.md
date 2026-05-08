# 意图：app.plan 应用设计 SOP

## 目标

把用户的新建需求整理成专业、可确认、可渲染的 PRD artifact。只负责设计和确认，不创建目录、不写 Go 文件、不 build。

## 适用场景

用户要求新建系统、新建目录、新建 Form/Table/Chart，或表达“搞个系统 / 做个后台 / 创建工具”，但还没有确认结构化 PRD。

## 执行步骤

1. 理解需求：提取业务对象、使用角色、核心操作、字段、状态、统计指标和验收路径。
2. 判断形态：管理一批记录用 Table；一次提交、导入、计算、生成用 Form；趋势、分布、占比、仪表盘指标用 Chart。Table 自带列表查询、新增、编辑、删除能力；普通单表 CRUD 不要再单独设计“提交/新增/创建” Form。
3. 选择案例：开干前读取 1 到多个匹配案例，优先看案例里的 `prd.json`；非常简单且 SDK 主文档已足够时才可跳过。
4. 做必要前置检查：结合当前目录信息判断是否创建新目录；不确定目录归属时可 `read_dir`，不要读取大量源码。
5. 输出 PRD：必须调用 `write_prd`，用结构化 JSON 参数描述目录、models、functions、Table/Form Request/Response、Chart filters/preview_data/summary 和验收用例。
6. 等用户确认：用户可点击前端的「确认 PRD」按钮，也可直接回复确认。未确认前不创建目录、不写 Go 文件、不 build。

## 结构化 PRD 规则

PRD 只通过 `write_prd` 工具参数输出。`write_prd` 成功就是本轮的主要输出；助手正文最多 1 句话提醒用户通过前端「确认 PRD」按钮或回复确认。不要再复制 PRD 表格、字段清单、功能清单或确认问题，避免前端预览和文本重复。

禁止把 PRD 写成 Markdown 表格。Markdown 表格只允许存在于旧案例说明或实现参考里，不作为 app.plan 的输出格式。

`functions` 必须直接传 JSON 数组，首层值就是数组本身。数组顺序必须就是用户理解业务的顺序，每个 `functions[]` 都写 `order`，从 1 开始。常见顺序：

1. 基础资料、配置、主数据维护 Table。
2. 核心业务提交/处理 Form，仅在它不是 Table 自带新增/编辑时出现。
3. 由 Form 产生的记录查询 Table。
4. 统计分析 Chart，放最后。

每个 `functions[]` 只填一种结构：

```json
{
  "type": "table",
  "table": {}
}
```

标准写法：

```json
{
  "functions": [
    {
      "order": 1,
      "type": "table",
      "route": "ticket_list.table",
      "model": "工单",
      "table": {
        "request_fields": [],
        "columns": ["工单标题", "工单状态"],
        "sample_rows": [
          {
            "工单标题": "打印机无法连接",
            "工单状态": "待处理"
          }
        ]
      }
    }
  ]
}
```

`type=table` 只填 `table`，`type=form` 只填 `form`，`type=chart` 只填 `chart`，不要在一个功能里混入多个类型。

## project 写法

`project` 负责目录确认，不负责业务字段。

```json
{
  "name": "客户工单管理",
  "code": "customer_ticket",
  "create_new_directory": true,
  "parent_directory": "/liubeiluo/ccc",
  "target_directory": "",
  "summary": "管理客户工单的提交、处理和统计。",
  "reason": "客户工单是独立业务域，需要单独目录承载。"
}
```

要求：

1. 新应用默认 `create_new_directory=true`。
2. `code` 使用小写下划线，语义清晰。
3. 如果放入现有目录，必须写 `target_directory` 并说明原因。

## models 写法

`models` 是 PRD 预览和后续实现的字段底座。这里只描述用户看到什么、怎么填、在哪些界面展示。

每个字段只写：

```json
{
  "name": "工单状态",
  "widget": "name:工单状态;type:select;options:待处理,处理中,已完成;options_colors:909399,E6A23C,67C23A;render_default:待处理",
  "validate": "required",
  "hide": "",
  "description": "状态流转字段。"
}
```

要求：

1. `models[].name` 是中文业务模型名，例如 `工单`、`商品`、`支付记录`。
2. 不写字段 code，不写 `go_source`，不写 Go struct，不写 `go_name/json_name/go_type/gorm/example`。
3. `widget` 是字段个性化入口：控件类型、选项、颜色、默认值、format、OnSelectFuzzy 等都写在 widget 里。
4. `validate` 决定是否必填、长度、范围。
5. `hide:"create,update"` 表示只在列表展示；`hide:"list"` 表示只在新增/编辑展示；不设则列表和表单都展示。
6. 不需要列出 `ID`、`CreatedAt`、`UpdatedAt`、`DeletedAt` 等系统字段，开发阶段自动补齐。

## Table 写法

Table 是业务列表。Table 必须把搜索请求、列表列、示例行分开写。

```json
{
  "title": "支付记录表",
  "type": "table",
  "route": "cashier_payment_record_list.table",
  "method": "GET",
  "model": "支付记录",
  "description": "只读查看支付流水；记录由收银台 Form 产生。",
  "table": {
    "capability": "仅列表查询和筛选，禁止手工新增、编辑、删除。",
    "readonly": true,
    "operations": ["列表查询"],
    "request_fields": [
      {
        "name": "状态",
        "type": "select",
        "required": false,
        "desc": "按支付状态筛选；可选：支付成功、支付失败；默认全部。"
      }
    ],
    "columns": ["创建时间", "订单号", "会员卡号", "会员姓名", "消费明细", "商品总额", "折扣金额", "实付金额", "状态"],
    "sample_rows": [
      {
        "创建时间": "2025-01-20 11:30",
        "订单号": "ORD202501200001",
        "会员卡号": "M001",
        "会员姓名": "张三",
        "消费明细": "可口可乐×2, 薯片×1",
        "商品总额": "15.00",
        "折扣金额": "1.50",
        "实付金额": "13.50",
        "状态": "支付成功"
      }
    ]
  }
}
```

要求：

1. `request_fields` 是 Table 搜索/筛选请求，不是列表列；没有搜索条件时填空数组。
2. `columns` 是用户看到的业务列。
3. `sample_rows` 的 key 必须和 `columns` 的用户可见列名一致。
4. 只读表必须 `readonly=true`，并在 `capability/description` 里说明不允许手工新增、编辑、删除。
5. `request_fields` 每项只写 `name/type/required/desc`；默认值、选项、OnSelectFuzzy 等复杂信息写进 `desc`。

## Form 写法

Form 是一次提交、导入、计算或生成。Form 必须明确 Request 和 Response。

先判断是否真的需要 Form：

1. 如果 Table 已经是同一个 model 的管理表，并允许 `新增/编辑/删除`，普通“新建记录、提交记录、编辑记录”都通过 Table 自带操作完成，不要再加独立 Form。
2. 只有存在明确差异时才设计 Form，例如：外部/匿名/客户自助提交、批量导入、文件解析、计算生成、支付结算、审批流、跨多表事务、只提交不允许编辑、提交后返回专门结果。
3. 有独立 Form 时，必须在 `description` 里说明它和 Table 新增/编辑的差异；否则应删除 Form，使用 Table 操作。

```json
{
  "title": "收银台 Form",
  "type": "form",
  "route": "cashier_desk.form",
  "method": "POST",
  "model": "支付记录",
  "description": "选择商品和会员后完成支付。",
  "form": {
    "request_fields": [
      {
        "name": "商品清单",
        "type": "table",
        "required": true,
        "desc": "type:table，至少 1 行；每行包含商品（OnSelectFuzzy 从商品表选）和数量（数字，≥1）。"
      },
      {
        "name": "会员卡",
        "type": "select",
        "required": true,
        "desc": "OnSelectFuzzy 从会员表选择正常会员。"
      }
    ],
    "response_fields": [
      {
        "name": "支付结果",
        "type": "text_area",
        "example": "支付成功！订单号：ORD202501200001",
        "desc": "提交后的处理结果。"
      },
      {
        "name": "订单号",
        "type": "input",
        "example": "ORD202501200001",
        "desc": "后端生成的支付流水号。"
      }
    ]
  }
}
```

要求：

1. `request_fields` 是用户提交字段，必须能被前端预览。
2. `response_fields` 是提交后展示给用户看的结果；没有展示结果时填空数组。
3. Form 字段同样只写 `name/type/required/desc`；响应字段可额外写 `example`。
4. 嵌套表格、OnSelectFuzzy、文件上传等复杂形态写在字段 `desc` 和对应 model 的 `widget` 里。

## Chart 写法

Chart 使用自己的结构，不要强行套 Form 的 Request/Response。一个 Chart 功能只对应一个 `.chart` 路由和一张图。多张图必须拆成多个 `functions[]`。

Chart 字段：

- `chart_type`：推荐写 `line`、`bar`、`pie`，也兼容 SDK 名 `LineChart`、`BarChart`、`PieChart`。
- `dimension`：横轴/分组/扇区维度，例如 `日期`、`状态`、`商品分类`。
- `metrics`：指标列表，例如 `销售额`、`订单数`、`工单数量`。
- `filters`：查询条件，每项只写 `name/type/required/desc`。时间范围不要写 `datetime_range`，必须拆成 `开始时间`、`结束时间` 两个 `datetime` 字段。
- `preview_data`：前端图表预览数据。折线/柱状按行写维度和指标；饼图按 `name/value` 写扇区。
- `summary`：摘要指标，例如总数、占比、平均值、NPS 分数；后续实现时放 Metadata。

折线图：

```json
{
  "title": "销售趋势统计",
  "type": "chart",
  "route": "cashier_sales_trend_statistics.chart",
  "method": "GET",
  "model": "支付记录",
  "description": "按日期统计销售额和订单数。",
  "chart": {
    "chart_type": "line",
    "dimension": "日期",
    "metrics": ["销售额", "订单数"],
    "filters": [
      {"name": "开始时间", "type": "datetime", "required": false, "desc": "统计开始时间。"},
      {"name": "结束时间", "type": "datetime", "required": false, "desc": "统计结束时间。"}
    ],
    "preview_data": [
      {"日期": "2025-01-18", "销售额": 860.5, "订单数": 28},
      {"日期": "2025-01-19", "销售额": 1024.0, "订单数": 34},
      {"日期": "2025-01-20", "销售额": 1280.5, "订单数": 41}
    ],
    "summary": [
      {"name": "总销售额", "value": "3165.00", "desc": "当前筛选范围内的销售额合计。"},
      {"name": "总订单数", "value": "103", "desc": "当前筛选范围内的订单数。"}
    ]
  }
}
```

柱状图：

```json
{
  "title": "分类销售统计",
  "type": "chart",
  "route": "cashier_category_sales_bar.chart",
  "method": "GET",
  "model": "支付记录",
  "description": "按商品分类统计销售额。",
  "chart": {
    "chart_type": "bar",
    "dimension": "商品分类",
    "metrics": ["销售额", "订单数"],
    "filters": [
      {"name": "开始时间", "type": "datetime", "required": false, "desc": "统计开始时间。"},
      {"name": "结束时间", "type": "datetime", "required": false, "desc": "统计结束时间。"}
    ],
    "preview_data": [
      {"商品分类": "饮料", "销售额": 520.0, "订单数": 80},
      {"商品分类": "零食", "销售额": 390.0, "订单数": 45},
      {"商品分类": "日用品", "销售额": 260.0, "订单数": 20}
    ],
    "summary": [
      {"name": "最高分类", "value": "饮料", "desc": "销售额最高的商品分类。"}
    ]
  }
}
```

饼图：

```json
{
  "title": "工单状态分布",
  "type": "chart",
  "route": "ticket_status_distribution.chart",
  "method": "GET",
  "model": "工单",
  "description": "按工单状态展示数量占比。",
  "chart": {
    "chart_type": "pie",
    "dimension": "工单状态",
    "metrics": ["工单数量"],
    "filters": [
      {"name": "开始时间", "type": "datetime", "required": false, "desc": "创建时间开始。"},
      {"name": "结束时间", "type": "datetime", "required": false, "desc": "创建时间结束。"}
    ],
    "preview_data": [
      {"name": "待处理", "value": 12},
      {"name": "处理中", "value": 8},
      {"name": "已完成", "value": 35}
    ],
    "summary": [
      {"name": "总工单数", "value": 55, "desc": "当前筛选范围内的工单总量。"}
    ]
  }
}
```

要求：

1. `line/bar/pie` 三种图都必须能按上面结构写清楚；除非用户明确要仪表盘，否则不要为了炫技使用 `gauge`。
2. 总数、占比、NPS 值、平均值等汇总指标放到 `summary`，后续实现映射到 Chart Metadata。
3. 不要编造 `resp.Charts`、`resp.Chart(chart1, chart2)` 或多图响应。

## 验收和确认

`acceptance_cases` 写用户能执行的验收路径：

```json
[
  {
    "name": "完成收银",
    "action": "进入收银台 Form，选择商品、数量和会员后提交。",
    "expected": "支付成功，库存和会员余额扣减，生成支付记录。"
  }
]
```

`confirmation.question` 必须明确询问：

```text
请确认是否按以上 PRD 创建目录和生成代码：
- 是否创建新目录：是/否
- 目录名称：xxx
- 目录 code：xxx
- 将创建/修改的功能：xxx.table、xxx.form、xxx.chart

确认后我再进入开发阶段，按 PRD 直接创建目录、写 Go 文件并 build。
```

用户确认前，禁止调用 `create_directory`、`write_go_file`、`build_workspace`。

## 案例参考

开干前按需求读取 1 到多个案例。案例里的 `prd.json` 是结构化 PRD 标准样例；`prd.md` 和 Go 文件只作为实现参考，不要模仿旧 Markdown 表格 PRD。

- 单表 CRUD、列表筛选、新增编辑删除：`read_doc("/system/prompt/case_catalog/table/ticket")`
- Form + Table + Chart、库存、经营统计、图表组合：`read_doc("/system/prompt/case_catalog/form_table_chart/cashier")`
- 多表关联、预约、资源占用、明细展示：`read_doc("/system/prompt/case_catalog/tables/meeting")`
- 问卷、投票、表单提交后进入列表统计：`read_doc("/system/prompt/case_catalog/formandtable/vote")`
- Excel/CSV、PDF、图片、视频、文本处理：按 `/system/prompt/case_catalog/form/...` 选择对应案例。
