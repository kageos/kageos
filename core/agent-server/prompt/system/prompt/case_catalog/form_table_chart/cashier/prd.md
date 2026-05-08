# 案例：收银台（Table + Form + Chart）

## 一、项目概要

- **类型**：多 Table（商品、会员、支付记录）+ 收银台 Form（请求里 table 子项 + 会员 select）+ 三类统计 Chart（折线、柱状、饼图）。适合单店、小门店、理发店这类轻量收银场景。
- **路由**：POST `cashier_desk.form`，GET 多个 list.table + 多个 statistics.chart；路由组 `/form_table_chart/cashier`。
- **适合参考**：FormTemplate 请求中 table 子组件、OnSelectFuzzy、主从表、统计/图表。

### 图形化展示

**模块与数据流（纯文本示意，任意环境可见）**

```
  [商品表] ──┐
            ├──► [收银台 Form] ──► [支付记录表] ──┬──► 销售趋势（折线图）
  [会员表] ──┘    POST cashier_desk.form          ├──► 每日销售（柱状图）
                                                 └──► 分类销售（饼图）
```

**图表类型一览**

| 图表     | 类型        | 路由名 |
|----------|-------------|--------|
| 销售趋势 | line 折线图 | cashier_sales_trend_statistics |
| 每日销售 | bar 柱状图  | cashier_sales_bar_statistics |
| 分类销售 | pie 饼图    | cashier_category_sales_statistics |

**图表效果简述**（供大模型理解每个图长什么样，写其他项目 PRD 时可照此用文本描述）

- **销售趋势（折线图）**：横轴为日期（按日聚合），纵轴两条线——销售额(元)、订单数；下方 Metadata 展示总销售额、总订单数、统计天数、平均日销售额。
- **每日销售（柱状图）**：横轴为日期，柱子展示销售额和订单数，适合对比每天经营表现。
- **分类销售（饼图）**：四类扇区——饮料/零食/日用品/其他，数值为各类销售额；摘要指标展示总销售额。

> 本 PRD 标准样例只展示 line、bar、pie 三种图表；除非用户明确要求，不要默认输出其他图表类型。

---

## 二、结构化 PRD JSON（app.plan 输出格式）

以下 JSON 是 `write_prd` 的标准结构示例。Table/Form 字段统一使用 `name/type/required/desc`，复杂选项和默认值写进 `desc`；Chart 使用 `filters/preview_data/summary`，不要回退到其他字段写法。

```json
{
  "kind": "agent_app_prd",
  "project": {
    "name": "会员收银系统",
    "code": "cashier",
    "create_new_directory": true,
    "parent_directory": "",
    "target_directory": "",
    "summary": "管理商品、会员、收银提交、支付记录和经营统计。",
    "reason": "收银、库存、会员余额和支付统计属于独立业务域，需要单独目录承载多表和事务逻辑。"
  },
  "models": [
    {
      "name": "商品",
      "description": "维护可售商品、价格、库存、折扣和上架状态。",
      "fields": [
        {"name":"商品名称","widget":"name:商品名称;type:input","validate":"required,min=2,max=50"},
        {"name":"商品分类","widget":"name:商品分类;type:select;options:饮料,零食,日用品,其他;options_colors:409EFF,E6A23C,67C23A,909399","validate":"required,oneof=饮料 零食 日用品 其他"},
        {"name":"售价","widget":"name:售价;type:float;precision:2;unit:元","validate":"required,gt=0"},
        {"name":"库存","widget":"name:库存;type:number;step:1;unit:件","validate":"required,min=0"},
        {"name":"折扣率","widget":"name:折扣率;type:float;precision:2;render_default:0.90","validate":"min=0,max=1"},
        {"name":"状态","widget":"name:状态;type:select;options:上架,下架;options_colors:67C23A,909399;render_default:上架","validate":"required,oneof=上架 下架"}
      ]
    },
    {
      "name": "会员",
      "description": "维护会员卡号、姓名、余额和状态。",
      "fields": [
        {"name":"会员卡号","widget":"name:会员卡号;type:input","validate":"required,min=6,max=20","description":"业务上要求唯一。"},
        {"name":"客户姓名","widget":"name:客户姓名;type:input","validate":"required,min=2,max=20"},
        {"name":"余额","widget":"name:余额;type:float;precision:2;unit:元","validate":"min=0"},
        {"name":"状态","widget":"name:状态;type:select;options:正常,冻结;options_colors:67C23A,F56C6C;render_default:正常","validate":"required,oneof=正常 冻结"}
      ]
    },
    {
      "name": "支付记录",
      "description": "收银台提交后生成的只读支付流水。",
      "fields": [
        {"name":"订单号","widget":"name:订单号;type:input","hide":"create,update","description":"后端生成。"},
        {"name":"会员卡号","widget":"name:会员卡号;type:input","hide":"create,update"},
        {"name":"会员姓名","widget":"name:会员姓名;type:input","hide":"create,update"},
        {"name":"消费明细","widget":"name:消费明细;type:text_area","hide":"create,update"},
        {"name":"商品总额","widget":"name:商品总额;type:float;precision:2;unit:元","hide":"create,update"},
        {"name":"折扣金额","widget":"name:折扣金额;type:float;precision:2;unit:元","hide":"create,update"},
        {"name":"实付金额","widget":"name:实付金额;type:float;precision:2;unit:元","hide":"create,update"},
        {"name":"状态","widget":"name:状态;type:select;options:支付成功,支付失败;options_colors:67C23A,F56C6C;render_default:支付成功","hide":"create,update"},
        {"name":"备注","widget":"name:备注;type:text_area","hide":"create,update"}
      ]
    }
  ],
  "functions": [
    {
      "title": "商品表",
      "type": "table",
      "route": "cashier_product_list.table",
      "method": "GET",
      "model": "商品",
      "description": "维护商品基础信息，供收银台选择商品。",
      "table": {
        "capability": "支持商品列表查询、筛选、新增、编辑、删除。",
        "readonly": false,
        "operations": [
          "列表查询",
          "新增",
          "编辑",
          "删除"
        ],
        "request_fields": [
          {"type":"input","required":false,"name":"商品名称","desc":"按商品名称模糊搜索。"},
          {"type":"select","required":false,"name":"商品分类","desc":"按分类筛选。"},
          {"type":"select","required":false,"name":"状态","desc":"按上架/下架筛选。"}
        ],
        "columns": [
          "商品名称",
          "商品分类",
          "售价",
          "库存",
          "折扣率",
          "状态"
        ],
        "sample_rows": [
          {"商品名称":"可口可乐","商品分类":"饮料","售价":"3.50","库存":"100","折扣率":"0.90","状态":"上架"},
          {"商品名称":"薯片","商品分类":"零食","售价":"8.00","库存":"50","折扣率":"0.85","状态":"上架"}
        ]
      }
    },
    {
      "title": "会员表",
      "type": "table",
      "route": "cashier_member_list.table",
      "method": "GET",
      "model": "会员",
      "description": "维护会员卡信息，供收银台选择会员并扣减余额。",
      "table": {
        "capability": "支持会员列表查询、筛选、新增、编辑、删除。",
        "readonly": false,
        "operations": [
          "列表查询",
          "新增",
          "编辑",
          "删除"
        ],
        "request_fields": [
          {"type":"input","required":false,"name":"会员卡号","desc":"按会员卡号模糊搜索。"},
          {"type":"input","required":false,"name":"客户姓名","desc":"按客户姓名模糊搜索。"},
          {"type":"select","required":false,"name":"状态","desc":"按正常/冻结筛选。"}
        ],
        "columns": [
          "会员卡号",
          "客户姓名",
          "余额",
          "状态"
        ],
        "sample_rows": [
          {"会员卡号":"M001","客户姓名":"张三","余额":"200.00","状态":"正常"},
          {"会员卡号":"M002","客户姓名":"李四","余额":"0.00","状态":"正常"}
        ]
      }
    },
    {
      "title": "支付记录表",
      "type": "table",
      "route": "cashier_payment_record_list.table",
      "method": "GET",
      "model": "支付记录",
      "description": "只读查看支付流水；记录由收银台 Form 产生，不允许手工新增、编辑、删除。",
      "table": {
        "capability": "仅列表查询和筛选，禁止手工新增、编辑、删除。",
        "readonly": true,
        "operations": [
          "列表查询"
        ],
        "request_fields": [
          {"type":"input","required":false,"name":"订单号","desc":"按订单号搜索。"},
          {"type":"input","required":false,"name":"会员卡号","desc":"按会员卡号搜索。"},
          {"type":"select","required":false,"name":"状态","desc":"按支付状态筛选。"}
        ],
        "columns": [
          "创建时间",
          "订单号",
          "会员卡号",
          "会员姓名",
          "消费明细",
          "商品总额",
          "折扣金额",
          "实付金额",
          "状态"
        ],
        "sample_rows": [
          {"创建时间":"2025-01-20 11:30","订单号":"ORD202501200001","会员卡号":"M001","会员姓名":"张三","消费明细":"可口可乐×2, 薯片×1","商品总额":"15.00","折扣金额":"1.50","实付金额":"13.50","状态":"支付成功"}
        ]
      }
    },
    {
      "title": "收银台 Form",
      "type": "form",
      "route": "cashier_desk.form",
      "method": "POST",
      "model": "支付记录",
      "description": "选择商品和会员后完成支付，写支付记录、扣减库存和会员余额。",
      "form": {
        "request_fields": [
          {"type":"table","required":true,"name":"商品清单","desc":"type:table，至少 1 行；每行包含商品（OnSelectFuzzy 从商品表选）和数量（数字，≥1）。"},
          {"type":"select","required":true,"name":"会员卡","desc":"OnSelectFuzzy 从会员表选择正常会员。"},
          {"type":"text_area","required":false,"name":"备注","desc":"可选备注，写入支付记录。"}
        ],
        "response_fields": [
          {"type":"text_area","example":"支付成功！订单号：ORD202501200001，实付金额：¥13.50","name":"支付结果","desc":"提交后的处理结果。"},
          {"type":"input","example":"ORD202501200001","name":"订单号","desc":"后端生成的支付流水号。"},
          {"type":"number","example":"15.00","name":"商品总额","desc":"打折前金额。"},
          {"type":"number","example":"1.50","name":"折扣金额","desc":"折扣优惠金额。"},
          {"type":"number","example":"13.50","name":"实付金额","desc":"最终支付金额。"},
          {"type":"table","example":"商品名称、单价、数量、小计、折扣率、折扣后金额","name":"商品清单","desc":"展示本次支付的商品明细。"},
          {"type":"form","example":"会员卡号、姓名、扣减后余额","name":"会员信息","desc":"展示支付后的会员信息。"}
        ]
      }
    },
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
        "metrics": [
          "销售额",
          "订单数"
        ],
        "filters": [
          {"name":"开始时间","type":"datetime","required":false,"desc":"统计开始时间。"},
          {"name":"结束时间","type":"datetime","required":false,"desc":"统计结束时间。"}
        ],
        "preview_data": [
          {"日期":"2025-01-18","销售额":860.5,"订单数":28},
          {"日期":"2025-01-19","销售额":1024,"订单数":34},
          {"日期":"2025-01-20","销售额":1280.5,"订单数":41}
        ],
        "summary": [
          {"name":"总销售额","value":"3165.00","desc":"当前筛选范围内的销售额合计。"},
          {"name":"总订单数","value":"103","desc":"当前筛选范围内的订单总数。"}
        ]
      }
    },
    {
      "title": "每日销售柱状图",
      "type": "chart",
      "route": "cashier_sales_bar_statistics.chart",
      "method": "GET",
      "model": "支付记录",
      "description": "按日期展示每日销售额和订单数柱状图。",
      "chart": {
        "chart_type": "bar",
        "dimension": "日期",
        "metrics": [
          "销售额",
          "订单数"
        ],
        "filters": [
          {"name":"开始时间","type":"datetime","required":false,"desc":"统计开始时间。"},
          {"name":"结束时间","type":"datetime","required":false,"desc":"统计结束时间。"}
        ],
        "preview_data": [
          {"日期":"2025-01-18","销售额":860.5,"订单数":28},
          {"日期":"2025-01-19","销售额":1024,"订单数":34},
          {"日期":"2025-01-20","销售额":1280.5,"订单数":41}
        ],
        "summary": [
          {"name":"最高销售日","value":"2025-01-20","desc":"当前筛选范围内销售额最高的日期。"}
        ]
      }
    },
    {
      "title": "分类销售占比",
      "type": "chart",
      "route": "cashier_category_sales_statistics.chart",
      "method": "GET",
      "model": "支付记录",
      "description": "按商品分类统计销售额占比。",
      "chart": {
        "chart_type": "pie",
        "dimension": "商品分类",
        "metrics": [
          "销售额"
        ],
        "filters": [
          {"name":"开始时间","type":"datetime","required":false,"desc":"统计开始时间。"},
          {"name":"结束时间","type":"datetime","required":false,"desc":"统计结束时间。"}
        ],
        "preview_data": [
          {"name":"饮料","value":520},
          {"name":"零食","value":390},
          {"name":"日用品","value":260},
          {"name":"其他","value":95}
        ],
        "summary": [
          {"name":"总销售额","value":"1265.00","desc":"当前筛选范围内的分类销售额合计。"}
        ]
      }
    }
  ],
  "acceptance_cases": [
    {"name":"维护商品","action":"进入商品表，新增商品并修改库存。","expected":"商品保存成功，列表展示最新价格、库存和状态。"},
    {"name":"维护会员","action":"进入会员表，新增会员卡并设置余额。","expected":"会员保存成功，收银台可以通过会员卡选择该会员。"},
    {"name":"完成收银","action":"进入收银台 Form，选择商品、数量和会员后提交。","expected":"支付成功，库存和会员余额扣减，生成支付记录。"},
    {"name":"查看支付记录","action":"进入支付记录表，按订单号或会员卡号搜索。","expected":"列表展示匹配支付流水，不能手工新增、编辑、删除。"},
    {"name":"查看经营统计","action":"打开销售趋势折线图、每日销售柱状图和分类销售饼图。","expected":"每个图表独立渲染一张图，并展示对应摘要指标。"}
  ],
  "confirmation": {
    "question": "请确认是否按以上 PRD 创建目录和生成代码：\n- 是否创建新目录：是\n- 目录名称：会员收银系统\n- 目录 code：cashier\n- 将创建/修改的功能：cashier_product_list.table、cashier_member_list.table、cashier_payment_record_list.table、cashier_desk.form、cashier_sales_trend_statistics.chart、cashier_sales_bar_statistics.chart、cashier_category_sales_statistics.chart\n\n确认后我再进入开发阶段，按 PRD 直接创建目录、写 Go 文件并 build。"
  }
}
```

---

## 三、文件与路由

| 文件                         | 说明           | 注册路由                    |
|------------------------------|----------------|-----------------------------|
| cashier_desk.go              | 收银台 Form    | POST cashier_desk.form      |
| cashier_product_list.go      | 商品列表       | GET cashier_product_list.table  |
| cashier_member_list.go       | 会员列表       | GET cashier_member_list.table   |
| cashier_payment_record_list.go | 支付记录列表 | GET cashier_payment_record_list.table |
| cashier_statistics.go        | 统计/图表      | GET 多个 xxx_statistics.chart   |

---

本案例只保留结构化 PRD 和实现要点；具体 SDK API 以主文档和专项参考为准。


---

## 四、实现参考要点

- PRD 标准示例只包含 `line`、`bar`、`pie` 三种图表；不要在 app.plan 默认输出其他图表类型。
- Table/Form 字段只用 `name/type/required/desc`；选项、默认值、OnSelectFuzzy、嵌套 table 说明都写进 `desc`。
- Table 的 `columns` 只是列表表头；`request_fields` 只是搜索条件；`sample_rows` 用用户可见列名做 key。
- Form 的 `request_fields` 是提交字段；`response_fields` 是提交后展示字段，可额外写 `example`。
- Chart 的 `filters` 是查询条件；`preview_data` 是前端预览和实现数据形态参考；`summary` 后续实现映射到 Metadata。
- 具体 Go 实现以 SDK 主文档和当前目录匹配案例为准，不要从本 PRD 文档复制代码。
