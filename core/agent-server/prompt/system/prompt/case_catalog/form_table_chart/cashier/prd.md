# 案例：收银台（Table + Form + Chart）

## 一、项目概要

- **类型**：多 Table（商品、会员、支付记录）+ 收银台 Form + 三类统计 Chart。
- **适合参考**：多表管理、独立提交入口、只读结果表、line/bar/pie 图表。
- **数据流**：商品和会员提供基础资料；收银台提交后生成支付记录；支付记录驱动销售趋势、支付状态和分类销售统计。

```
[商品] ──┐
        ├──► [收银台] ──► [支付记录] ──┬──► 销售趋势统计
[会员] ──┘                            ├──► 支付状态分布
                                      └──► 分类销售统计
```

## 二、结构化 PRD JSON

`product_manager` 只输出轻量 PRD v2：`project/tables/forms/charts/workflow/rules`。
字段只写 `name/widget/required/desc/hide`，`widget` 只保留组件类型；选项、默认值、范围、数据来源和计算规则写进自然语言 `desc`。
完整标准样例见同目录 `prd.json`。

```json
{
  "kind": "agent_app_prd",
  "schema_version": "prd.v2",
  "project": {
    "name": "会员收银系统",
    "code": "cashier",
    "summary": "管理商品、会员、收银支付记录，并统计销售趋势、支付状态和分类销售。"
  },
  "tables": [
    {
      "name": "商品",
      "title": "商品管理",
      "desc": "维护门店商品资料和库存。",
      "fields": [
        {"name": "商品名称", "widget": "input", "required": true, "desc": "商品显示名称。"},
        {"name": "商品分类", "widget": "select", "required": true, "desc": "例如饮料、零食、日用品、其他。"},
        {"name": "单价", "widget": "number", "required": true, "desc": "商品销售单价，单位元。"},
        {"name": "库存数量", "widget": "number", "required": true, "desc": "当前库存数量，必须为非负整数。"},
        {"name": "上架状态", "widget": "select", "required": true, "desc": "有上架、下架两个选项，默认上架。"}
      ],
      "search_fields": [
        {"name": "商品名称", "widget": "input", "required": false, "desc": "按商品名称模糊搜索。"},
        {"name": "商品分类", "widget": "select", "required": false, "desc": "按商品分类筛选。"}
      ],
      "handlers": ["OnTableAddRow", "OnTableUpdateRow", "OnTableDeleteRow"],
      "examples": [
        {"商品名称": "可口可乐", "商品分类": "饮料", "单价": 3.5, "库存数量": 120, "上架状态": "上架"}
      ]
    },
    {
      "name": "支付记录",
      "title": "支付记录",
      "desc": "只读查看收银台产生的支付流水。",
      "fields": [
        {"name": "订单号", "widget": "input", "required": true, "desc": "系统生成的支付订单号。"},
        {"name": "会员卡号", "widget": "input", "required": false, "desc": "本次支付关联的会员卡号。"},
        {"name": "消费明细", "widget": "text_area", "required": true, "desc": "商品名称、数量和单价组成的消费明细。"},
        {"name": "实付金额", "widget": "number", "required": true, "desc": "实际支付金额。"},
        {"name": "支付状态", "widget": "select", "required": true, "desc": "有支付成功、支付失败两个状态。"}
      ],
      "search_fields": [
        {"name": "会员卡号", "widget": "input", "required": false, "desc": "按会员卡号搜索支付记录。"},
        {"name": "支付状态", "widget": "select", "required": false, "desc": "按支付成功或支付失败筛选。"}
      ],
      "handlers": [],
      "examples": [
        {"订单号": "ORD202605090001", "会员卡号": "M001", "消费明细": "可口可乐×2，薯片×1", "实付金额": 13.5, "支付状态": "支付成功"}
      ]
    }
  ],
  "forms": [
    {
      "name": "收银台",
      "desc": "选择商品、数量和会员后完成收银支付，生成支付记录并扣减库存。",
      "target_table": "支付记录",
      "request_fields": [
        {"name": "商品清单", "widget": "table", "required": true, "desc": "至少 1 行，每行选择商品并填写数量；商品从商品表中选择。"},
        {"name": "会员卡号", "widget": "select", "required": false, "desc": "选择正常状态会员；不选择时按非会员原价结算。"},
        {"name": "支付方式", "widget": "select", "required": true, "desc": "有现金、微信、支付宝、会员余额四个选项。"}
      ],
      "response_fields": [
        {"name": "订单号", "widget": "input", "required": false, "desc": "支付成功后返回系统生成订单号。"},
        {"name": "支付结果", "widget": "input", "required": false, "desc": "展示支付成功或失败原因。"},
        {"name": "实付金额", "widget": "number", "required": false, "desc": "本次订单实际支付金额。"}
      ],
      "example": {
        "request": {"商品清单": "可口可乐×2，薯片×1", "会员卡号": "M001", "支付方式": "会员余额"},
        "response": {"订单号": "ORD202605090001", "支付结果": "支付成功", "实付金额": 13.5}
      }
    }
  ],
  "charts": [
    {
      "name": "销售趋势统计",
      "desc": "按日期统计销售额和订单数。",
      "source_table": "支付记录",
      "chart_type": "line",
      "dimension": "日期",
      "metrics": ["销售额", "订单数"],
      "filters": [
        {"name": "开始时间", "widget": "datetime", "required": false, "desc": "统计开始时间。"},
        {"name": "结束时间", "widget": "datetime", "required": false, "desc": "统计结束时间。"}
      ],
      "examples": [
        {"dimension": "2026-05-07", "metrics": {"销售额": 860.5, "订单数": 28}},
        {"dimension": "2026-05-08", "metrics": {"销售额": 1024, "订单数": 34}}
      ]
    }
  ],
  "workflow": [
    {"type": "table", "ref": "商品"},
    {"type": "form", "ref": "收银台"},
    {"type": "table", "ref": "支付记录"},
    {"type": "chart", "ref": "销售趋势统计"}
  ],
  "rules": [
    "收银台提交成功后生成支付记录并扣减商品库存。",
    "支付记录只允许查询和筛选，不允许手工新增、编辑、删除。"
  ]
}
```

## 三、实现参考要点

- `tables[].handlers` 决定是否生成新增、编辑、删除回调；只读查询表使用空数组。
- `forms[].target_table` 表示表单提交后写入或影响的主表。
- `charts[].examples` 推荐使用模型自然结构，例如 `{"dimension":"2026-05-07","metrics":{"销售额":860.5,"订单数":28}}`；工具会归一成前端预览行。
- `workflow` 是用户展示和操作顺序，也是后续代码生成顺序，例如先商品/会员，再收银台，再支付记录，最后统计图。
- 具体 Go 实现以 SDK 主文档和当前目录源码为准。
