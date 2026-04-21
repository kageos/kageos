# 函数 Schema 统一模型重构方案

## 文档目的

本文档用于在正式开干前盘清当前 `request/response` 模型的问题，明确为什么要改、改成什么、影响面在哪里，以及后续执行 TODO。

当前结论：如果我们新开分支处理，并且不需要兼容历史用户数据，那么值得把函数配置从 `request/response` 双字段模型升级为统一 `schema` 模型。

## 背景

当前函数配置主要由 `function` 表中的两个 JSON 字段承载：

```go
Request  json.RawMessage `json:"request" gorm:"type:json"`
Response json.RawMessage `json:"response" gorm:"type:json"`
```

前端 `FunctionDetail` 也延续了同样结构：

```ts
interface FunctionDetail {
  template_type?: string
  request?: FieldConfig[]
  response?: FieldConfig[]
}
```

这个模型在早期只支持简单表单或简单接口时是够用的：`request` 表示入参字段，`response` 表示出参字段。

但现在函数类型已经扩展到 `form`、`table`，并且还有定时任务、执行记录、消息通知、AI 工具 schema、Hub 包、权限、表格增删改查、回调等能力。`request/response` 开始承载太多语义，已经不适合作为长期函数配置模型。

## 当前问题

### 1. `request/response` 语义过载

`form` 场景里：

```text
request = 表单输入字段
response = 函数输出字段
```

这个语义基本合理。

但 `table` 场景里：

```text
request = 查询条件？新增字段？更新字段？回调参数？
response = 表格列？接口响应？详情字段？
```

同一个字段在不同上下文含义不同，后续代码只能靠约定猜。

### 2. Table 类型被迫套 Form 的字段模型

Table 实际上至少有这些配置：

```text
查询条件
表格列
新增场景字段
编辑场景字段
删除动作
批量动作
回调能力
```

现在这些配置无法被 `request/response` 准确表达，导致很多逻辑默认 `response = 表格列`。这个约定短期能跑，长期会让执行记录、字段展示策略、AI 工具生成都变复杂。

同时也不应该把 Table 字段拆成多份完整数组，例如 `fields/create_fields/update_fields/detail_fields`。这样会让 schema 变大，同一个字段配置重复出现，字段名称、widget、校验规则、选项等改动时容易不一致。

更合理的方式是：字段只定义一份，特殊字段通过轻量 `display.scenes` 白名单控制在哪些场景出现。

### 3. 定时任务执行记录不好做只读渲染

执行记录现在保存的是：

```text
request_payload
response_payload
status
trace_id
duration_millis
error_message
```

如果只有 `request/response`，前端渲染时无法稳定判断：

```text
这条记录是 form 执行还是 table 动作？
table_create 应该按哪些字段展示？
table_update 应该展示 id、updates 还是完整行？
table_delete 应该展示 ids 还是删除结果？
response_payload.result 应该按哪个字段集合渲染？
```

结果就是容易退回到 JSON 展示，体验差，也不利于后续通知跳转后的详情页。

### 4. AI 工具 schema 和运行时 schema 容易不一致

AI 工具需要清楚知道函数的输入输出结构。当前 `request/response` 只能表达一组入参和一组出参，无法准确表达 Table 的搜索、创建、更新、删除等不同动作。

这会导致工具 schema 生成逻辑越来越依赖特殊判断。

### 5. 前端业务组件与底层字段结构耦合

当前很多地方直接读：

```ts
functionDetail.request
functionDetail.response
```

这导致底层模型一变，业务组件就会大面积受影响。我们需要把字段读取收敛到 selector 层，而不是让各组件理解 schema 内部结构。

### 6. 后续扩展类型会继续放大问题

如果后面支持：

```text
chart
workflow
agent
report
dashboard
file processor
```

继续使用 `request/response` 会让每种类型都去“借用”这两个字段，模型会越来越脏。

## 目标

把函数配置改成统一 `schema` 模型：

```text
function.schema = 函数配置唯一来源
```

`schema` 需要满足：

```text
按函数类型组织配置
有 version 便于后续升级
form/table/chart 等类型可以拥有不同结构
前端通过 selector 读取字段，不直接读深层结构
定时任务和执行记录可以基于 schema 判断如何渲染
AI 工具可以基于 schema 生成更准确的 input/output schema
```

## 最佳方案

### 1. 数据模型改为 schema 单字段

新分支里建议直接把 `request/response` 从主模型里移除，改成：

```go
type Function struct {
    Schema       json.RawMessage `json:"schema" gorm:"type:json"`
    TemplateType string          `json:"template_type"`
    Method       string          `json:"method"`
    Router       string          `json:"router"`
    // 其他元信息保留
}
```

说明：

```text
schema 是配置唯一来源
template_type 可以保留为索引/查询/展示字段，但其值必须与 schema.type 一致
callbacks 建议进入 schema 顶层，避免 form/table 结构不一致
create_tables 是否进入 schema 需要开干前再确认
```

如果开发库需要迁移，可以用脚本从旧 `request/response/template_type/callbacks/create_tables` 生成新 `schema`。如果可以重建开发库，则直接按新模型初始化。

### 数据库存储结构

当前 `function` 表核心字段：

```text
id
app_id
tree_id
method
router
request        JSON
response       JSON
callbacks      varchar/text，逗号分隔
template_type  varchar，当前 model 里映射到 widget 列
create_tables  varchar/text，逗号分隔
has_config
created_by
created_at
updated_at
```

新 `function` 表建议结构：

```text
id
app_id
tree_id
method
router
schema         JSON，函数配置唯一来源
template_type  varchar，冗余摘要字段，必须与 schema.type 一致
create_tables  varchar/text，暂时保留；是否迁入 schema 待确认
has_config
created_by
created_at
updated_at
```

建议删除或废弃的旧字段：

```text
request   删除或不再读写
response  删除或不再读写
callbacks 删除或不再作为配置源，改为 schema.callbacks
```

数据库层 `schema` 存储示例，Table：

```json
{
  "version": 1,
  "type": "table",
  "table": {
    "request": [],
    "fields": [
      {
        "code": "id",
        "name": "ID",
        "field_name": "ID",
        "search": "eq",
        "data": {
          "type": "int"
        },
        "widget": {
          "type": "ID",
          "config": {}
        },
        "display": {
          "scenes": ["list"]
        }
      },
      {
        "code": "title",
        "name": "工单标题",
        "field_name": "Title",
        "search": "like",
        "data": {
          "type": "string"
        },
        "widget": {
          "type": "input",
          "config": {}
        },
        "validation": "required,min=2,max=100"
      }
    ],
    "actions": {
      "create": true,
      "batch_create": true,
      "update": true,
      "batch_delete": true
    }
  },
  "callbacks": [
    "OnTableAddRow",
    "OnTableCreateInBatches",
    "OnTableUpdateRow",
    "OnTableDeleteRows"
  ]
}
```

数据库层 `schema` 存储示例，Form：

```json
{
  "version": 1,
  "type": "form",
  "form": {
    "request": [
      {
        "code": "member_id",
        "name": "会员卡",
        "field_name": "MemberID",
        "data": {
          "type": "int"
        },
        "widget": {
          "type": "select",
          "config": {
            "creatable": false
          }
        },
        "callbacks": ["OnSelectFuzzy"],
        "validation": "required"
      }
    ],
    "response": [
      {
        "code": "order_number",
        "name": "订单号",
        "field_name": "OrderNumber",
        "data": {
          "type": "string"
        },
        "widget": {
          "type": "input",
          "config": {}
        }
      }
    ]
  },
  "callbacks": []
}
```

数据库迁移策略：

```text
新分支开发阶段优先直接改表结构或重建开发库
如需要保留开发库数据，用一次性脚本把 request/response/callbacks 迁入 schema
迁移后后端读写只认 schema
template_type 作为冗余字段保留时，每次写入必须校验 template_type == schema.type
callbacks 如果顶层字段暂时保留，只能作为列表摘要，不允许作为配置源
```

### 2. Form Schema

推荐结构：

```json
{
  "version": 1,
  "type": "form",
  "form": {
    "request": [],
    "response": []
  },
  "callbacks": []
}
```

语义：

```text
form.request = 表单提交输入字段
form.response = 函数执行输出字段
callbacks = 表单级回调能力
```

### 3. Table Schema

推荐结构：

```json
{
  "version": 1,
  "type": "table",
  "table": {
    "request": [],
    "fields": [],
    "actions": {
      "create": true,
      "update": true,
      "delete": true,
      "batch_delete": true
    }
  },
  "callbacks": []
}
```

语义：

```text
table.request = 查询/筛选字段
table.fields = 表格字段唯一配置源
table.actions = 表格支持的动作
callbacks = 函数回调能力，例如 OnTableUpdateRow、OnTableDeleteRows、OnSelectFuzzy
```

字段场景控制使用 `display.scenes`：

```json
{
  "code": "internal_note",
  "name": "内部备注",
  "widget": { "type": "textarea" },
  "display": {
    "scenes": ["list", "update"]
  }
}
```

规则：

```text
display 不存在：不做场景限制，按默认规则展示/使用
display.scenes 存在：白名单，仅在列出的场景展示/使用
display.scenes 必须是非空数组
display.scenes 空数组是非法配置，不支持
display.scenes 只允许固定场景名：list/create/update
display.scenes 出现未知场景名是非法配置
字段完全不在任何场景展示：不要进入 schema，生成时直接排除
不要使用 permissions 表达这个含义，避免和用户/角色/资源权限混淆
这里的 display 是字段展示/使用策略，不是权限控制
搜索不进入 display.scenes，继续使用字段已有 search 标签/配置判断
详情不进入 display.scenes，能 list 的字段自然可以 detail 展示
```

场景定义：

```text
list：表格列表列展示
create：新增表单、table_create 定时任务输入展示
update：编辑表单、table_update 定时任务输入展示
```

默认场景建议：

```text
list：默认使用 table.fields
detail：不单独作为 scene，默认复用 list 可见字段
create：默认使用 table.fields 中可编辑、非只读字段
update：默认使用 table.fields 中可编辑、非只读字段
search：不走 display.scenes，使用 table.request 和字段已有 search 标签/配置
actions 未配置时按后端能力或模板默认值处理
```

selector 逻辑：

```ts
function visibleInScene(field, scene) {
  const scenes = field.display?.scenes
  if (!Array.isArray(scenes)) {
    return true
  }
  return scenes.includes(scene)
}
```

注意：

```text
不要引入 display.scenes = [] 表达“任何场景都不展示”
如果字段所有场景都不用，类似 json:"-" / widget:"-"，它不应该出现在返回给前端的 schema 里
```

### 4. Chart / 其他类型预留

后续类型可以独立扩展：

```json
{
  "version": 1,
  "type": "chart",
  "chart": {
    "request": [],
    "data_fields": [],
    "chart_config": {}
  }
}
```

这样不会污染 form/table 的语义。

## 最新 Schema 完整示例

下面示例按当前接口返回结构完整改造，不再只给字段片段。

约定：

```text
data.request / data.response 不再作为函数配置主字段返回
data.schema 是函数配置唯一来源
data.template_type 可以保留为冗余摘要字段，但必须与 schema.type 一致
data.callbacks 不再作为完整配置字段；如列表接口需要，可以返回 callbacks 摘要，详情配置以 schema.callbacks 为准
data.create_tables 暂时保留在顶层，是否进入 schema 留作开干前确认
```

## 旧结构 vs 新结构对比

### 顶层结构对比

旧结构：

```json
{
  "template_type": "table",
  "request": [],
  "response": [],
  "callbacks": "OnTableAddRow,OnTableUpdateRow",
  "create_tables": "ticket"
}
```

新结构：

```json
{
  "template_type": "table",
  "create_tables": "ticket",
  "schema": {
    "version": 1,
    "type": "table",
    "table": {
      "request": [],
      "fields": []
    },
    "callbacks": ["OnTableAddRow", "OnTableUpdateRow"]
  }
}
```

核心变化：

```text
request/response 不再是顶层配置字段
schema 成为配置唯一来源
callbacks 从逗号字符串变成 schema.callbacks 数组
template_type 可继续作为摘要字段，但必须等于 schema.type
create_tables 暂时保留顶层，是否进入 schema 另行确认
```

### Table 字段迁移对比

旧结构：

```json
{
  "template_type": "table",
  "request": [],
  "response": [
    {
      "code": "id",
      "name": "ID",
      "search": "eq",
      "widget": { "type": "ID", "config": {} },
      "table_permission": "read"
    },
    {
      "code": "title",
      "name": "工单标题",
      "search": "like",
      "widget": { "type": "input", "config": {} },
      "validation": "required,min=2,max=100"
    }
  ]
}
```

新结构：

```json
{
  "template_type": "table",
  "schema": {
    "version": 1,
    "type": "table",
    "table": {
      "request": [],
      "fields": [
        {
          "code": "id",
          "name": "ID",
          "search": "eq",
          "widget": { "type": "ID", "config": {} },
          "display": {
            "scenes": ["list"]
          }
        },
        {
          "code": "title",
          "name": "工单标题",
          "search": "like",
          "widget": { "type": "input", "config": {} },
          "validation": "required,min=2,max=100"
        }
      ]
    },
    "callbacks": []
  }
}
```

迁移规则：

```text
旧 table.request -> 新 schema.table.request
旧 table.response -> 新 schema.table.fields
旧 callbacks 字符串 -> 新 schema.callbacks 数组
旧 table_permission: read -> 新 display.scenes: ["list"]
旧 search 原样保留，不进入 display.scenes
旧 validation/data/widget/children/callbacks 等字段级配置原样保留
```

### Form 字段迁移对比

旧结构：

```json
{
  "template_type": "form",
  "request": [
    {
      "code": "member_id",
      "name": "会员卡",
      "widget": { "type": "select", "config": { "creatable": false } },
      "callbacks": ["OnSelectFuzzy"],
      "validation": "required"
    }
  ],
  "response": [
    {
      "code": "order_number",
      "name": "订单号",
      "widget": { "type": "input", "config": {} },
      "table_permission": "read"
    }
  ],
  "callbacks": ""
}
```

新结构：

```json
{
  "template_type": "form",
  "schema": {
    "version": 1,
    "type": "form",
    "form": {
      "request": [
        {
          "code": "member_id",
          "name": "会员卡",
          "widget": { "type": "select", "config": { "creatable": false } },
          "callbacks": ["OnSelectFuzzy"],
          "validation": "required"
        }
      ],
      "response": [
        {
          "code": "order_number",
          "name": "订单号",
          "widget": { "type": "input", "config": {} }
        }
      ]
    },
    "callbacks": []
  }
}
```

迁移规则：

```text
旧 form.request -> 新 schema.form.request
旧 form.response -> 新 schema.form.response
旧 callbacks 空字符串 -> 新 schema.callbacks = []
旧 response 内 table_permission: read 删除，因为 form.response 天然只读
字段级 callbacks 继续保留在字段内部
字段级 children 原样保留
```

### 语义变化对比

| 旧字段 | 旧含义 | 新字段 | 新含义 |
| --- | --- | --- | --- |
| `request` | form 入参 / table 查询条件，语义混用 | `schema.form.request` / `schema.table.request` | 按类型拆分后的请求字段 |
| `response` | form 出参 / table 列，语义混用 | `schema.form.response` / `schema.table.fields` | 按类型拆分后的输出字段或表格字段 |
| `callbacks` | 顶层逗号字符串 | `schema.callbacks` | 顶层数组 |
| `table_permission: read` | 只读/只展示的混合标记 | `display.scenes: ["list"]` 或删除 | table list-only 字段用 display；form response 直接删除 |
| `search` | 字段搜索标签 | `search` | 原样保留，不进入 display |
| `permissions` | 当前用户资源权限 | `permissions` | 原样保留，不进入 schema |

### Table 完整示例

```json
{
  "code": 0,
  "data": {
    "id": 53,
    "app_id": 2,
    "tree_id": 0,
    "method": "GET",
    "router": "/liubeiluo/work/ticket_system/ticket_list.table",
    "has_config": false,
    "create_tables": "ticket",
    "template_type": "table",
    "schema": {
      "version": 1,
      "type": "table",
      "table": {
        "request": [],
        "fields": [
          {
            "code": "id",
            "name": "ID",
            "field_name": "ID",
            "search": "eq",
            "data": {
              "type": "int"
            },
            "widget": {
              "type": "ID",
              "config": {}
            },
            "display": {
              "scenes": ["list"]
            }
          },
          {
            "code": "created_at",
            "name": "创建时间",
            "field_name": "CreatedAt",
            "search": "gte,lte",
            "data": {
              "type": "int"
            },
            "widget": {
              "type": "timestamp",
              "config": {
                "format": "YYYY-MM-DD HH:mm:ss"
              }
            },
            "display": {
              "scenes": ["list"]
            }
          },
          {
            "code": "updated_at",
            "name": "更新时间",
            "field_name": "UpdatedAt",
            "search": "gte,lte",
            "data": {
              "type": "int"
            },
            "widget": {
              "type": "timestamp",
              "config": {
                "format": "YYYY-MM-DD HH:mm:ss"
              }
            },
            "display": {
              "scenes": ["list"]
            }
          },
          {
            "code": "title",
            "name": "工单标题",
            "field_name": "Title",
            "search": "like",
            "data": {
              "type": "string"
            },
            "widget": {
              "type": "input",
              "config": {}
            },
            "validation": "required,min=2,max=100"
          },
          {
            "code": "description",
            "name": "问题描述",
            "field_name": "Description",
            "search": "like",
            "data": {
              "type": "string"
            },
            "widget": {
              "type": "richtext",
              "config": {
                "height": 400
              }
            },
            "validation": "required"
          },
          {
            "code": "priority",
            "name": "优先级",
            "field_name": "Priority",
            "search": "in",
            "data": {
              "type": "string"
            },
            "widget": {
              "type": "select",
              "config": {
                "creatable": false,
                "default": "中",
                "options": ["低", "中", "高"],
                "options_colors": ["success", "warning", "danger"]
              }
            },
            "validation": "required,oneof=低 中 高"
          },
          {
            "code": "status",
            "name": "工单状态",
            "field_name": "Status",
            "search": "in",
            "data": {
              "type": "string"
            },
            "widget": {
              "type": "select",
              "config": {
                "creatable": false,
                "default": "待处理",
                "options": ["待处理", "处理中", "已完成"],
                "options_colors": ["info", "warning", "success"]
              }
            },
            "validation": "required,oneof=待处理 处理中 已完成"
          },
          {
            "code": "handler",
            "name": "处理人",
            "field_name": "Handler",
            "search": "in",
            "data": {
              "type": "string"
            },
            "widget": {
              "type": "user",
              "config": {
                "default": "Me()"
              }
            }
          },
          {
            "code": "deadline",
            "name": "截止时间",
            "field_name": "Deadline",
            "search": "gte,lte",
            "data": {
              "type": "int"
            },
            "widget": {
              "type": "timestamp",
              "config": {
                "format": "YYYY-MM-DD HH:mm:ss"
              }
            }
          },
          {
            "code": "remark",
            "name": "备注",
            "field_name": "Remark",
            "search": "like",
            "data": {
              "type": "string"
            },
            "widget": {
              "type": "text_area",
              "config": {}
            }
          },
          {
            "code": "cc_departments",
            "name": "抄送部门",
            "field_name": "CcDepartments",
            "search": "contains",
            "data": {
              "type": "string"
            },
            "widget": {
              "type": "departments",
              "config": {}
            }
          },
          {
            "code": "attachment",
            "name": "附件",
            "field_name": "Attachment",
            "data": {
              "type": "string"
            },
            "widget": {
              "type": "files",
              "config": {
                "max_count": 5
              }
            }
          },
          {
            "code": "remaining_time",
            "name": "剩余时间",
            "field_name": "RemainingTime",
            "data": {
              "type": "string"
            },
            "widget": {
              "type": "text",
              "config": {}
            },
            "display": {
              "scenes": ["list"]
            }
          }
        ],
        "actions": {
          "create": true,
          "batch_create": true,
          "update": true,
          "batch_delete": true
        }
      },
      "callbacks": [
        "OnTableAddRow",
        "OnTableCreateInBatches",
        "OnTableUpdateRow",
        "OnTableDeleteRows"
      ]
    },
    "created_at": "2026-04-04T12:43:59Z",
    "updated_at": "2026-04-20T17:43:27Z",
    "created_by": "liubeiluo",
    "full_code_path": "/liubeiluo/work/ticket_system/ticket_list.table",
    "permissions": {
      "app:admin": true,
      "table:admin": true,
      "table:delete": true,
      "table:read": true,
      "table:update": true,
      "table:write": true
    }
  },
  "msg": "成功",
  "metadata": null
}
```

Table 迁移说明：

```text
当前 response 全量迁移到 schema.table.fields
当前 request 迁移到 schema.table.request
当前 callbacks 字符串拆分为 schema.callbacks
当前 table_permission: read 不再保留，read-only/list-only 字段改成 display.scenes: ["list"]
没有 display 的字段表示不限制，默认可用于 list/create/update，再由 readonly/disabled/validation 等配置决定是否可编辑
search 字段原样保留，搜索逻辑继续根据 search 标签判断
```

### Form 完整示例

```json
{
  "code": 0,
  "data": {
    "id": 98,
    "app_id": 2,
    "tree_id": 0,
    "method": "POST",
    "router": "/liubeiluo/work/member_cashier/cashier_desk.form",
    "has_config": false,
    "create_tables": "member_cashier_product,member_cashier_member,member_cashier_payment_record,member_cashier_payment_record_item",
    "template_type": "form",
    "schema": {
      "version": 1,
      "type": "form",
      "form": {
        "request": [
          {
            "code": "member_id",
            "name": "会员卡",
            "field_name": "MemberID",
            "data": {
              "type": "int"
            },
            "widget": {
              "type": "select",
              "config": {
                "creatable": false
              }
            },
            "callbacks": ["OnSelectFuzzy"],
            "validation": "required"
          },
          {
            "code": "product_quantities",
            "name": "商品清单",
            "field_name": "ProductQuantities",
            "data": {
              "type": "[]struct"
            },
            "widget": {
              "type": "table"
            },
            "children": [
              {
                "code": "product_id",
                "name": "商品",
                "field_name": "ProductID",
                "data": {
                  "type": "int"
                },
                "widget": {
                  "type": "select",
                  "config": {
                    "creatable": false
                  }
                },
                "callbacks": ["OnSelectFuzzy"],
                "validation": "required"
              },
              {
                "code": "quantity",
                "name": "数量",
                "field_name": "Quantity",
                "data": {
                  "type": "int"
                },
                "widget": {
                  "type": "number",
                  "config": {
                    "default": 1
                  }
                },
                "validation": "required,min=1"
              }
            ],
            "validation": "required,min=1"
          },
          {
            "code": "remarks",
            "name": "备注",
            "field_name": "Remarks",
            "data": {
              "type": "string"
            },
            "widget": {
              "type": "text_area",
              "config": {}
            }
          }
        ],
        "response": [
          {
            "code": "payment_result",
            "name": "支付结果",
            "field_name": "PaymentResult",
            "data": {
              "type": "string"
            },
            "widget": {
              "type": "text_area",
              "config": {}
            }
          },
          {
            "code": "order_number",
            "name": "订单号",
            "field_name": "OrderNumber",
            "data": {
              "type": "string"
            },
            "widget": {
              "type": "input",
              "config": {}
            }
          },
          {
            "code": "total_amount",
            "name": "商品总额",
            "field_name": "TotalAmount",
            "data": {
              "type": "float"
            },
            "widget": {
              "type": "float",
              "config": {
                "precision": "2",
                "unit": "元"
              }
            }
          },
          {
            "code": "discount_amount",
            "name": "折扣金额",
            "field_name": "DiscountAmount",
            "data": {
              "type": "float"
            },
            "widget": {
              "type": "float",
              "config": {
                "precision": "2",
                "unit": "元"
              }
            }
          },
          {
            "code": "final_amount",
            "name": "实付金额",
            "field_name": "FinalAmount",
            "data": {
              "type": "float"
            },
            "widget": {
              "type": "float",
              "config": {
                "precision": "2",
                "unit": "元"
              }
            }
          },
          {
            "code": "product_list",
            "name": "商品清单",
            "field_name": "ProductList",
            "data": {
              "type": "[]struct"
            },
            "widget": {
              "type": "table"
            },
            "children": [
              {
                "code": "product_id",
                "name": "商品ID",
                "field_name": "ProductID",
                "data": {
                  "type": "int"
                },
                "widget": {
                  "type": "ID",
                  "config": {}
                }
              },
              {
                "code": "product_name",
                "name": "商品名称",
                "field_name": "ProductName",
                "data": {
                  "type": "string"
                },
                "widget": {
                  "type": "input",
                  "config": {}
                }
              },
              {
                "code": "price",
                "name": "单价",
                "field_name": "Price",
                "data": {
                  "type": "float"
                },
                "widget": {
                  "type": "float",
                  "config": {
                    "precision": "2",
                    "unit": "元"
                  }
                }
              },
              {
                "code": "quantity",
                "name": "数量",
                "field_name": "Quantity",
                "data": {
                  "type": "int"
                },
                "widget": {
                  "type": "number",
                  "config": {}
                }
              },
              {
                "code": "total_price",
                "name": "小计",
                "field_name": "TotalPrice",
                "data": {
                  "type": "float"
                },
                "widget": {
                  "type": "float",
                  "config": {
                    "precision": "2",
                    "unit": "元"
                  }
                }
              },
              {
                "code": "discount_rate",
                "name": "折扣率",
                "field_name": "DiscountRate",
                "data": {
                  "type": "float"
                },
                "widget": {
                  "type": "float",
                  "config": {
                    "precision": "2"
                  }
                }
              },
              {
                "code": "discount_price",
                "name": "折扣后金额",
                "field_name": "DiscountPrice",
                "data": {
                  "type": "float"
                },
                "widget": {
                  "type": "float",
                  "config": {
                    "precision": "2",
                    "unit": "元"
                  }
                }
              }
            ]
          },
          {
            "code": "member_info",
            "name": "会员信息",
            "field_name": "MemberInfo",
            "data": {
              "type": "struct"
            },
            "widget": {
              "type": "form"
            },
            "children": [
              {
                "code": "card_number",
                "name": "会员卡号",
                "field_name": "CardNumber",
                "data": {
                  "type": "string"
                },
                "widget": {
                  "type": "text",
                  "config": {}
                }
              },
              {
                "code": "customer_name",
                "name": "客户姓名",
                "field_name": "CustomerName",
                "data": {
                  "type": "string"
                },
                "widget": {
                  "type": "text",
                  "config": {}
                }
              },
              {
                "code": "balance",
                "name": "余额",
                "field_name": "Balance",
                "data": {
                  "type": "float"
                },
                "widget": {
                  "type": "float",
                  "config": {
                    "precision": "2",
                    "unit": "元"
                  }
                }
              },
              {
                "code": "status",
                "name": "状态",
                "field_name": "Status",
                "data": {
                  "type": "string"
                },
                "widget": {
                  "type": "text",
                  "config": {}
                }
              }
            ]
          }
        ]
      },
      "callbacks": []
    },
    "created_at": "2026-04-09T18:13:31Z",
    "updated_at": "2026-04-20T17:43:27Z",
    "created_by": "liubeiluo",
    "full_code_path": "/liubeiluo/work/member_cashier/cashier_desk.form",
    "permissions": {
      "app:admin": true,
      "form:admin": true,
      "form:read": true,
      "form:write": true
    }
  },
  "msg": "成功",
  "metadata": null
}
```

Form 迁移说明：

```text
当前 request 全量迁移到 schema.form.request
当前 response 全量迁移到 schema.form.response
当前 callbacks 字符串拆分为 schema.callbacks；空字符串变成 []
form.response 中原 table_permission: read 不再保留，因为 response 天然是只读展示配置
字段级 callbacks 继续保留在字段内部，例如 OnSelectFuzzy
```

Table 规则：

```text
table.request = 独立查询条件
table.fields = 表格字段唯一来源
display.scenes 不存在 = 不限制，按默认规则使用
display.scenes = 非空白名单，只允许 list/create/update
search 不进入 display.scenes，继续使用字段已有 search 标签/配置
detail 不进入 display.scenes，默认复用 list 可见字段
完全不展示的字段不进入 schema
```

### Table 字段选择逻辑

```text
列表字段 = table.fields 中 visibleInScene(field, "list") 的字段
详情字段 = 列表字段
新增字段 = table.fields 中 visibleInScene(field, "create") 且字段非只读/非禁用的字段
编辑字段 = table.fields 中 visibleInScene(field, "update") 且字段非只读/非禁用的字段
搜索字段 = table.request + table.fields 中带 search 标签/配置的字段
```

说明：

```text
display 只控制字段是否进入 list/create/update 场景
display 不控制搜索
display 不控制权限
display 不控制是否可编辑
是否可编辑仍然看字段已有 readonly/disabled/widget config 等配置
```

## 前端访问方式

不要让业务组件直接读：

```ts
functionDetail.schema.form.request
functionDetail.schema.table.fields
```

应该统一通过 selector：

```ts
getFunctionSchema(functionDetail)
getFunctionType(functionDetail)
getFormRequestFields(functionDetail)
getFormResponseFields(functionDetail)
getTableRequestFields(functionDetail)
getTableFields(functionDetail)
getTableCreateFields(functionDetail)
getTableUpdateFields(functionDetail)
getFunctionCallbacks(functionDetail)
```

说明：

```text
getTableCreateFields / getTableUpdateFields 只是 selector 名称
它们都从 table.fields 按 display.scenes 和默认规则过滤出来
schema 内不再存在 create_fields / update_fields / detail_fields 三份字段数组
详情展示使用 getTableFields 的 list 可见结果
```

这样后续 schema 内部结构调整时，业务组件不用大面积改。

## 定时任务执行记录渲染方案

schema 改完后，执行记录只读渲染会更清晰。

Form 执行：

```text
输入区域 = form.request + request_payload
输出区域 = form.response + response_payload.result
摘要区域 = trace_id、duration_millis、status、error_message
```

Table 创建：

```text
动作摘要 = 新增行
输入区域 = table.fields 按 create 场景过滤 + request_payload
输出区域 = table.fields 按 list 场景过滤 + response_payload.result
```

Table 更新：

```text
动作摘要 = 更新行
输入区域 = id + updates + old_values
字段渲染 = table.fields 按 update 场景过滤
输出区域 = response_payload.result
```

Table 删除：

```text
动作摘要 = 删除行
输入区域 = ids
输出区域 = 删除结果或原始响应
```

兜底规则：

```text
schema 缺失时展示格式化 JSON
字段匹配失败时展示字段卡片 + 原始 JSON
失败记录优先展示错误信息、trace_id、耗时
```

## 最新渲染逻辑

### 通用字段过滤

```ts
type DisplayScene = 'list' | 'create' | 'update'

function visibleInScene(field, scene: DisplayScene) {
  const scenes = field.display?.scenes
  if (!Array.isArray(scenes)) {
    return true
  }
  return scenes.includes(scene)
}
```

规则：

```text
display 不存在：返回 true
display.scenes 非空：只在命中的 scene 返回 true
display.scenes 为空或包含未知 scene：schema 校验失败
```

### Table 页面渲染

```text
列表页：getTableFields = fields 按 list 过滤
详情页：复用 getTableFields
新增页：getTableCreateFields = fields 按 create 过滤，再排除只读/禁用字段
编辑页：getTableUpdateFields = fields 按 update 过滤，再排除只读/禁用字段
搜索区：getTableSearchFields = table.request + fields 中带 search 标签/配置的字段
```

### 执行记录只读渲染

```text
form 执行：form.request 渲染 request_payload，form.response 渲染 response_payload.result
table_create：table.fields 按 create 过滤渲染 request_payload，按 list 过滤渲染响应
table_update：展示 id、updates、old_values；updates 用 table.fields 按 update 过滤渲染
table_delete：展示 ids 和删除结果
异常/字段不匹配：展示错误摘要 + 格式化 JSON 兜底
```

### 接口返回策略

```text
函数详情接口返回完整 schema
服务树、函数搜索、列表接口默认不返回完整 schema，只返回 template_type、callbacks、必要摘要
需要完整 schema 的页面再按 full_code_path/id 拉取详情
```

这样避免 table.fields 很大时拖慢目录树、搜索列表和工作台初始化。

## 影响范围

### 后端

需要调整：

```text
core/app-server/model/function.go
dto/function.go
dto/service_tree.go
dto/app_runtime_namespace.go
core/app-server/service/app_service.go
core/app-server/service/function_service.go
core/app-server/api/v1/function.go
core/app-server/api/v1/service_tree.go
core/app-server/api/v1/standard_api.go
core/app-server/service/service_tree_workspace_service.go
core/app-server/service/service_tree_hub_copy_helpers.go
core/agent-server/service 中依赖 request/response 的工具
Hub bundle 导入导出逻辑
测试用例
```

重点风险点：

```text
创建/更新函数时 schema 写入是否完整
标准 API 是否能从 schema 取到正确字段
table 搜索、新增、编辑、删除是否仍然正常
AI tool schema 生成是否仍然准确
Hub 包结构是否需要同步升级
```

### 前端

需要调整：

```text
FunctionDetail 类型
函数详情加载与保存
FormView
TableView
FormDialog
TableRowDetailDrawer
FormOperateLogSection
ScheduledTaskList
FunctionExecutionResultReadonly
TableDomainService
FormDomainService
WorkspaceDomainService
所有直接读取 functionDetail.request/response 的组件和 composable
相关测试
```

重点风险点：

```text
不要在业务组件里散落 schema 深层访问
先建 selector，再迁移业务逻辑
table 的新增/编辑字段要有默认推导规则
执行记录渲染要保留 JSON 兜底
```

## 是否应该现在改

建议现在改。

理由：

```text
当前没有历史用户，兼容压力低
定时任务执行记录已经暴露出 request/response 模型问题
后续 AI tool、通知跳转、Hub、更多函数类型都会依赖清晰 schema
越晚改，直接读 request/response 的代码会越多
现在开新分支集中处理，成本最低
```

不建议继续在旧模型上补丁式扩展。

如果继续保留 `request/response`：

```text
短期改动小
长期每个新功能都要做特殊判断
table 语义会继续混乱
执行记录很难优雅渲染
schema 快照后续也会很别扭
```

## 改造收益与代价

### 收益

```text
函数配置语义更清晰，不再让 request/response 承担所有类型的含义
table 字段只保留一份 fields，避免 create/update/detail 多份字段重复
display.scenes 规则很轻，只解决 list/create/update 的展示差异
搜索继续使用已有 search 标签，不引入额外复杂度
详情复用 list 字段，不额外增加 detail 场景
定时任务执行记录可以按 schema 渲染，不再只能展示 JSON
AI 工具可以从 schema 生成更准确的入参结构
后续 chart/workflow/agent 等类型可以独立扩展，不污染 table/form
selector 成为唯一入口，后续 schema 内部调整不会让业务组件大面积改
```

### 代价

```text
这是函数模型重构，不是小改，会影响前后端核心链路
后端创建、更新、查询函数都要改成 schema
前端所有直接读取 functionDetail.request/response 的地方都要迁移到 selector
agent prompt、函数生成、Hub 导入导出、AI tool schema 都要同步
短期测试成本会上升，尤其是 form/table/定时任务/执行记录/回调链路
开发阶段可以不兼容历史数据，但开发库如果保留旧数据仍需要一次性迁移或重建
```

### 总体判断

```text
短期成本：中等偏大
长期收益：高
当前时机：适合，因为没有历史用户包袱
推荐策略：新分支集中改，不在旧 request/response 上继续补丁式扩展
```

## 改动规模评估

整体评估：中等偏大，但可控。

不是小 UI 改动，也不是单纯加字段。它是函数配置模型重构，会影响前后端多个核心链路。

推荐按 1 到 2 天级别的集中重构来预估第一版，不建议穿插在其他大功能中做。

如果先定义 schema 和 selector，再逐层迁移，风险可控。

如果全仓直接搜索替换 `request/response`，风险较高。

## 非目标

本轮不做：

```text
执行记录 schema 快照
历史数据兼容
复杂版本迁移框架
UI 大改版
运行时 payload 协议重构
权限模型重构
```

说明：

```text
执行记录 schema 快照后续再加
当前重点是函数配置源头从 request/response 切到 schema
运行时 request_payload/response_payload 仍然保持现有 JSON 结构
```

## 推荐实施路径

### 阶段 1：冻结 schema 结构

- [ ] 确认 `form` schema 字段名。
- [ ] 确认 `table` schema 字段名。
- [ ] 确认 callbacks 统一放在 schema 顶层。
- [ ] 确认 `template_type` 是否继续作为冗余字段保留。
- [ ] 确认 table 只保留单 `fields` 数组。
- [ ] 确认字段级 `display.scenes` 白名单语义。
- [ ] 确认 `display.scenes` 空数组作为非法配置处理。
- [ ] 确认 `display.scenes` 只允许 `list/create/update`。
- [ ] 确认搜索继续走字段已有 search 标签/配置，不进入 `display.scenes`。
- [ ] 确认详情默认复用 list 可见字段，不进入 `display.scenes`。
- [ ] 确认完全隐藏字段不进入 schema。

### 阶段 2：新增类型和 selector

- [ ] 后端定义 `FunctionSchema`、`FormFunctionSchema`、`TableFunctionSchema`。
- [ ] 前端定义 `FunctionSchema` TypeScript 类型。
- [ ] 前端新增 `functionSchemaSelectors.ts`。
- [ ] 用 selector 替代直接读取 `functionDetail.request/response`。
- [ ] 补 selector 单元测试。

### 阶段 3：后端模型重构

- [ ] `model.Function` 使用 `Schema` 字段作为配置唯一来源。
- [ ] 数据库 `function` 表新增/保留 `schema` JSON 字段。
- [ ] 移除或废弃数据库 `request/response` 字段。
- [ ] 移除或废弃顶层 `callbacks` 配置字段；如暂时保留，只作为列表摘要。
- [ ] 准备开发库重建或一次性迁移脚本，把旧 `request/response/callbacks` 迁入 `schema`。
- [ ] 创建函数时写入 schema。
- [ ] 更新函数时写入 schema。
- [ ] 获取函数详情时返回 schema。
- [ ] 服务树搜索结果返回必要 schema 摘要。
- [ ] 标准 API 从 schema 读取 form/table 字段。
- [ ] 新增 schema normalize/validate，校验 `display.scenes` 非空且只使用 `list/create/update`。

### 阶段 4：前端业务迁移

- [ ] Form 视图改为读取 `form.request/form.response`。
- [ ] Table 视图改为读取 `table.request/table.fields`。
- [ ] 新增弹窗读取 `table.fields` 并按 `create` 场景过滤。
- [ ] 编辑抽屉读取 `table.fields` 并按 `update` 场景过滤。
- [ ] 详情抽屉读取 `table.fields` 的 list 可见结果。
- [ ] 操作日志读取 schema selector。
- [ ] 定时任务创建和执行记录读取 schema selector。

### 阶段 5：AI 工具和 Hub 同步

- [ ] list_tools input_schema 从 schema 生成。
- [ ] form 工具使用 `form.request`。
- [ ] table 搜索工具使用 `table.request`。
- [ ] table 新增/编辑/删除工具使用 table action schema。
- [ ] Hub bundle schema 版本升级。
- [ ] 导入导出使用新 schema。

### 阶段 6：执行记录只读渲染

- [ ] Form 执行记录按 `form.request/form.response` 渲染。
- [ ] Table create 执行记录按 `table.fields` 的 `create` 场景渲染。
- [ ] Table update 执行记录按 `table.fields` 的 `update` 场景渲染。
- [ ] Table delete 执行记录展示 ids 和删除结果。
- [ ] 所有执行记录保留 JSON 兜底。
- [ ] 通知跳转到执行详情时打开只读渲染视图。

### 阶段 7：测试和验收

- [ ] 新建 Form 函数。
- [ ] Form 提交正常。
- [ ] Form 执行记录正常。
- [ ] 新建 Table 函数。
- [ ] Table 查询正常。
- [ ] Table 新增正常。
- [ ] Table 编辑正常。
- [ ] Table 删除正常。
- [ ] Table 定时任务正常。
- [ ] Table 执行记录正常。
- [ ] OnSelectFuzzy 正常。
- [ ] OnTableUpdateRow 正常。
- [ ] AI 工具列表正常。
- [ ] Hub 导入导出正常。
- [ ] 前端 type-check 通过。
- [ ] 后端相关 go test 通过。

## 验收标准

第一版完成后应满足：

```text
function 配置唯一来源是 schema
前端业务组件不直接依赖 request/response
form/table 基础链路能正常使用
定时任务执行记录可以按 schema 只读渲染
旧 request/response 语义不再继续扩散
后续 chart/workflow 等类型可以独立扩展 schema
```

## 开放问题

需要开干前确认：

```text
完全隐藏字段由哪一层排除？建议 schema 生成层直接排除。
table.update 场景是否需要专门展示 old_values？
未来是否需要增加更多 display scene？本轮先不做，避免模型膨胀。
create_tables 是否进入 schema？
template_type 是否保留为数据库冗余字段？
Hub bundle 是否本轮同步升级？
agent prompt 中生成函数的格式是否同步改为 schema？
```

## 最终建议

建议新开分支做。

建议不要保留历史兼容包袱，也不要继续在 `request/response` 上打补丁。

正确做法是：

```text
先冻结 schema
再建立 selector
再迁移后端模型
再迁移前端业务
最后做执行记录只读渲染
```

这样改完后，函数模型会更清晰，定时任务执行记录会更好渲染，AI 工具 schema 会更准确，后续扩展新函数类型也不会继续污染 `request/response`。
