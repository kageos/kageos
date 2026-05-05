# 工作台、工具、文档与 SDK 的关系说明

本文用于让新同事或大模型快速理解 AI-Agent-OS 当前的生成与运行闭环：工作台负责对话和操作入口，skills 是任务 SOP 和能力路由，docs 是长文档和案例上下文，tools 是大模型的受控动作集合，SDK 是应用协议与编译校验层，runtime/app-server/front-end 根据 SDK 产出的 schema 完成部署、注册、渲染和调用。

一句话概括：

> 大模型不是直接“随便写一个网站”，而是在工作台中先按 Skills 目录读取匹配 SOP，再按 skill 读取文档、案例和当前目录上下文，通过 tools 修改 `namespace` 中的 Go 代码；SDK 把 Go 结构体、路由、模板和 widget tag 编译成函数 schema；runtime 等待新版本启动并校验成功后，app-server 保存函数资产；前端再按 schema 渲染 Form/Table/Chart，Agent 也可继续通过 run_* tools 调用这些函数。

## 1. 总体分层

```text
用户需求
  ↓
工作台 UI / Mini Workstation
  ↓
agent-server 会话、prompt、工具循环
  ↓
Skills：sop.* / sdk.* / system.tools.* / system.openapi.*
  ↓
文档与环境上下文：skill.required_docs + /system/prompt + 当前 full_code_path
  ↓
tools：read/write/build/run/search/publish
  ↓
namespace/{user}/{app}/code/api/... 中的 Go 代码
  ↓
sdk/agent-app：Template + widget + handler + compile validation
  ↓
app-runtime：编译、启动新版本、等待 SDK 启动通知
  ↓
app-server：同步 function/service_tree/schema 元数据
  ↓
web：按 schema 渲染函数，或 Agent 用 run_* tools 调用函数
```

各层职责如下：

| 层 | 主要位置 | 职责 |
|---|---|---|
| 工作台前端 | `web/src/architecture/presentation` | 目录树、函数渲染、Mini 工作台、SSE 展示、文件/结果展示 |
| Agent 服务 | `core/agent-server` | 会话、LLM 调用、stream loop、ToolRegistry、prompt/env 注入 |
| Skills | `core/agent-server/skills` | 按任务场景承接 SOP、文档索引、示例索引、可用能力和完成标准 |
| Prompt/文档 | `core/agent-server/prompt/system/prompt` | 模式边界、平台说明、SDK 长文档、案例库、工作环境模板 |
| Tools | `core/agent-server/service/tool_*.go` | 给模型提供受控读写、构建、执行、搜索、发布能力 |
| SDK | `sdk/agent-app` | Go 应用协议，注册 Form/Table/Chart，解析 widget tag，产出 schema，启动校验 |
| Runtime | `core/app-runtime` | 源码写盘、go build、容器版本、启动等待、生命周期通知 |
| App Server | `core/app-server` | app/function/service_tree 元数据，标准 API，权限/日志/任务等平台能力 |
| 前端组件协议 | `web/src/core/constants/widget.ts`、`widgetRegistry` | 把 SDK schema 中的 widget.type 映射成实际 Vue 组件 |

## 2. 工作台做什么

工作台不是一个普通聊天框，它是“当前目录 + 当前应用 + 当前函数树 + LLM + tools”的操作面。

前端侧主要能力：

- 左侧 `ServiceTreePanel` 展示应用、目录、函数、docs、board。
- 中间 `WorkspaceFunctionRenderer` 根据函数类型渲染 Form/Table/Chart。
- 右侧或浮窗 `MiniWorkstation` 发起 Agent 会话，展示 content、tool_call、工具结果和输出文件。
- 表单组件通过 `widgetComponentFactory` 按 `widget.type` 渲染。
- 表单提交通过 `FieldExtractorRegistry` 提取各字段 `raw` 值，确保 schema 到请求体一致。

后端侧 `WorkspaceChatService` 做几件关键事：

- 根据 `full_code_path` 获取当前目录上下文。
- 根据 mode 加载对应 prompt provider。
- 默认追加 Skills 工作规则和 Skills 目录，引导模型能判断意图时直接 `read_skill`。
- 把用户消息、历史消息、工具定义、环境信息拼成 LLM messages。
- 通过 `streamloop.RunStreamLoop` 执行“模型输出 tool_calls -> 调工具 -> 保存 tool 消息 -> 继续下一轮”。

因此，工作台本质是一个带环境和工具权限的 Agent runtime，而不是单纯的问答 UI。

## 3. Skills、文档和 prompt 做什么

当前主链路是 **skills 优先**。prompt 负责最小身份、模式边界和安全约束；skills 负责按场景路由 SOP；docs 负责承接长文档、SDK 细节和案例。

模型默认流程：

```text
先识别意图
  ↓
按 Skills 目录直接 read_skill("<skill id>")
  ↓
按 skill.required_docs 读取 /system/prompt 长文档
  ↓
按 skill.allowed_tools / completion 执行和自检
```

只有无法从目录判断应该读取哪个 skill，或用户需求超出目录大纲时，才使用 `search_skills` 兜底。

Skill 分层：

- `sop.*`：创建、修改、解释、执行等工作台任务 SOP。
- `sdk.*`：SDK 写法、组件选择、构建校验、平台 API 调用。
- `system.tools.*`：`/system/tools` 官方工具工作空间能力。
- `system.openapi.*`：`/system/openapi` 平台接口工作空间能力。

文档/prompt 是模型的长上下文和模式约束。

主要来源：

- `platform-overview.md`：告诉模型平台是什么，Form/Table/Chart 怎么选。
- `platform-cross-cutting-capabilities.md`：说明权限、审批、评论、日志、消息等平台能力，不让业务代码重复造。
- `sdk/agent-app-sdk-readme.md`：SDK 主文档，说明模板、tag、组件、回调、DB、文件、图表等写法。
- `case_catalog/*`：具体业务案例，给模型参考完整 PRD 和代码风格。
- `mode/*/config.json` 与 `system_prompt.md`：不同模式可用工具、模式边界和 skill 路由提示。
- `doc/workspace-env-template.md`：注入当前用户、当前目录、函数 schema、可读文档目录、运行时工具等环境信息。

加载逻辑：

- agent-server 优先从 `/system/prompt` 服务树读取文档，缺失时回退到本地 seed。
- 每次会话都会根据当前 `full_code_path` 构造环境块。
- skills 默认从本地 `core/agent-server/skills` 内置加载，运行时 prompt 中会给出按 mode 过滤后的 Skills 目录。
- 环境块里会包含当前目录下函数的 schema 摘要，因此模型不需要猜字段名。
- `read_doc` 可以按文档路径读取 SDK 文档或案例；`read_go_file` 读代码时会从 runtime 磁盘实时读取，保证不是旧快照。

文档的价值不是“写给人看”这么简单，而是把平台约束变成模型可以遵守的上下文。

## 4. Tools 做什么

tools 是大模型唯一可靠的动作边界。模型不能直接访问生产系统，而是通过工具做有限动作。

当前工具大致分三类：

| 类型 | 示例 | 作用 |
|---|---|---|
| 工作区工具 | `read_go_file`、`read_dir`、`write_go_file`、`search_replace_file`、`delete_file`、`build_workspace` | 读写当前应用源码并触发构建 |
| 运行时工具 | `run_form_submit`、`run_table_search`、`run_table_create`、`run_table_update`、`run_chart_query`、`run_on_select_fuzzy` | 执行已注册函数，验证效果 |
| Skills 工具 | `read_skill`、`search_skills` | 读取本地 skill；`search_skills` 只做兜底 |
| 平台/检索工具 | `search_tools`、`search_hub_directory`、`copy_directory`、`publish_to_hub`、`record_workspace_event`、`web_search` | 搜索复用、发布复用、记录事件、外部信息 |

关键规则：

- `write_go_file` 只落盘，不编译。
- `build_workspace` 只编译部署，不写文件。
- `read_skill` 是 skills 主入口，能判断意图时直接按 Skills 目录读取；`search_skills` 只在不确定时兜底。
- `search_tools` 返回内置工具和 `system` 用户下已注册函数摘要，执行前应看字段摘要。
- 低频官方工具优先沉淀到 `/system/tools`，平台接口优先沉淀到 `/system/openapi`。
- `run_form_submit`、`run_table_search`、`run_chart_query` 都走 app-server 标准 API，再通过 NATS 调到 SDK app。

这个拆分很重要：模型可以连续写多个文件，然后一次 `build_workspace`；构建失败时错误回流给模型，模型再继续修复。

## 5. SDK 做什么

`sdk/agent-app` 是整个系统的应用协议层。

业务代码只需要做三件事：

1. 用 `PackageContext` 注册路由。
2. 用 `FormTemplate`、`TableTemplate`、`ChartTemplate` 描述函数。
3. 用 handler 执行业务逻辑并通过 `response` 返回结构化结果。

SDK 负责把这些转换成平台协议：

- 路由：`xxx.form`、`xxx.table`、`xxx.chart`。
- 类型：Form/Table/Chart。
- 字段：从 Go struct tag 解析 `json`、`widget`、`search`、`validate`、`display`、`data`。
- UI：生成 `widget.Field`，包含 `widget.type`、`config`、`data.type`、children 等。
- 回调：Table CRUD、OnSelectFuzzy、OnApiCreate 等。
- 运行：连接 NATS，监听 invoke/control/discovery。
- 上报：启动时和 onAppUpdate 时生成 API diff，并把 schema 回给 runtime/app-server。

最核心链路：

```text
Go struct
  ↓ ParseModelWithType
FieldTags
  ↓ ValidateFieldTags
Field
  ↓ functionschema.NewForm/NewTable/NewChart
FunctionSchema
  ↓ functionschema.Validate
ApiInfo
  ↓ onAppUpdate diff
app-server function/service_tree
```

SDK 当前已经有启动级编译校验：

- `app.Run()` 里先执行 `CompileAndValidate()`。
- 校验 route 后缀和 Template 类型是否匹配。
- 调用 `getApis()` 解析所有 schema。
- widget 级 validator 校验组件参数和 Go 类型是否匹配。
- `functionschema.Validate()` 再校验最终 schema。
- 失败时 SDK 发布 `startup failed` 生命周期事件，runtime 等待启动时会收到失败并让 `build_workspace` 返回错误。

这意味着现在的 `build_workspace` 不只是 Go 编译成功就算完成，还会等待新版本启动结果。模型写错 widget、路由后缀、schema 等问题，会在启动阶段暴露出来。

## 6. Runtime 和 App Server 做什么

`build_workspace` 的实际后端路径是：

```text
agent-server build_workspace
  ↓ apicall.UpdateAppBuild
app-server /workspace/api/v1/app/update
  ↓ appcall.UpdateApp
app-runtime UpdateApp
  ↓ go mod tidy + go build
  ↓ 创建新版本 runtime 容器
  ↓ 等待 SDK app 发布 startup running/failed
  ↓ 成功后停止旧版本
  ↓ 触发/等待 onAppUpdate，拿 API diff
app-server 同步 function 和 service_tree
```

runtime 的关键职责：

- 写源码和回滚失败写入。
- 编译二进制。
- 用版本号启动独立容器。
- 注入 `APP_VERSION`、SDK 环境变量。
- 等待 SDK 生命周期通知。
- 成功后再切换版本，避免坏版本直接覆盖旧版本。

app-server 的关键职责：

- 保存 app 当前版本。
- 根据 SDK diff 创建或更新 function 记录。
- 根据 SDK 返回的 package 全量列表做目录对账。
- 保存 service_tree 节点，让工作台和搜索可见。
- 提供标准 API：`table/search`、`form/submit`、`chart/query`、callback、定时任务等。

## 7. 前端如何消费 SDK schema

前端不理解 Go 代码，只理解 app-server 返回的 function schema。

核心约定：

```text
widget.Field
  code        字段编码，来自 json tag
  name        展示名，来自 widget:name
  data.type   数据类型，来自 Go 类型推断
  widget.type 组件类型，来自 widget:type
  config      组件配置，来自 widget tag
  search      搜索能力
  display     list/create/update 展示场景
  children    嵌套 form/table 子字段
```

前端消费方式：

- `WidgetType` 常量要和 SDK `widget.Type*` 对齐。
- `widget-configs.ts` 要和 SDK 每个 widget struct 的 JSON 字段对齐。
- `widgetRegistry` 把 `widget.type` 注册到 Vue 组件。
- `FieldExtractorRegistry` 决定表单提交时怎么从 `FieldValue` 提取 raw。
- Table/Form/Chart 渲染器根据 schema 决定搜索栏、列表列、新增/编辑表单、响应展示。

例如 `type:list;item_type:number` 这次新增后，对应链路是：

```text
SDK TypeList + List struct + validateListWidget
  ↓
schema widget.type=list, config.item_type=number
  ↓
前端 WidgetType.LIST + ListWidgetConfig + ListWidget.vue
  ↓
FieldExtractorRegistry 用 BasicFieldExtractor 提交 number[]
```

这说明每个组件都必须同时有 SDK 定义、SDK validator、前端类型、前端组件、提交提取逻辑和文档说明。缺一个就容易漂移。

## 8. 一次完整开发闭环

下面是一条标准“生成新功能”的闭环：

```text
1. 用户在某个目录打开工作台，说出需求
2. agent-server 注入当前目录、函数、文件、可读文档
3. 模型按 Skills 目录直接读取匹配 skill
4. 模型按 skill 读取 platform/SDK/case docs
5. 模型判断应该创建 Table、Form 还是 Chart
6. 模型用 write_go_file 或 search_replace_file 落盘 Go 代码
7. 模型调用 build_workspace
8. runtime 编译 Go
9. SDK 启动并 CompileAndValidate
10. SDK onAppUpdate 生成 schema diff
11. app-server 同步 function/service_tree
12. 前端刷新后可见新函数
13. 模型用 run_* 工具执行验证
14. 用户得到可运行、可复用、可继续迭代的函数资产
```

这条链路里有三层约束：

| 约束 | 作用 |
|---|---|
| skills/docs/prompt | skills 负责 SOP 路由，docs 承接长上下文，prompt 保留模式边界 |
| tools | 限制模型只能通过受控动作修改、构建、执行 |
| SDK/runtime validation | 把错误从运行时或用户使用阶段提前到 build/startup 阶段 |

这三层一起工作，才是系统真正的优势。

## 9. 当前设计判断

我认为这个思路是对的，而且方向比较高级。它不是传统低代码，也不是普通 AI 编程工具，而是把 AI 生成、应用协议、运行时验证、统一 UI、工具调用和资产沉淀放在一条闭环里。

更准确地说，它像下面几类系统的组合：

- 低代码平台的 schema/UI 渲染能力。
- MCP/Function Calling 的工具协议思想。
- AI IDE 的代码生成和修复循环。
- 企业应用平台的权限、服务树、日志、Hub、版本管理。
- SDK-first 的应用协议。

真正有价值的地方不是“模型能写 Go”，而是：

- 写出来的东西必须是平台可理解的 Form/Table/Chart。
- 字段和组件必须进入统一 schema。
- 函数能被前端渲染，也能被 Agent 当工具调用。
- build/startup 能阻断错误 schema。
- 最终沉淀为 service_tree/function/Hub 资产，而不是一次性代码。

这套设计比“AI 生成一个页面”更像“AI 生成一个可治理的企业应用能力”。

## 10. 主要风险

当前最大的风险不是方向，而是契约分散。

### 10.1 widget 契约分散

一个组件需要同时改：

- SDK `widget/*.go`
- SDK validator
- `widget.go` 支持列表
- 前端 `WidgetType`
- 前端 `widget-configs.ts`
- 前端 Vue 组件
- 前端 registry
- FieldExtractor
- SDK 文档和 prompt
- 测试

这套流程可控，但如果没有强规则，后续组件多了会漏。

### 10.2 skills/docs/prompt 与代码可能漂移

skills 和文档告诉模型怎么写，但如果 SDK 已改而 skill/docs 没改，模型会继续生成旧写法。当前已把 SOP 收敛到 `core/agent-server/skills`，把长文档放到 `/system/prompt`，这是对的，但还需要把 skill 和文档变成版本化、可测试的资产。

### 10.3 tag 字符串能力强，但也脆

`widget:"name:标题;type:input"` 对模型友好，但字符串约定天然容易拼错。现在 validator 能挡住很多错误，下一步可以继续把高频错误前移到 lint 或 typed builder。

### 10.4 search_tools 范围有意收窄

`search_tools` 当前主要搜内置工具和 `system` 用户已注册函数，不搜所有用户目录。这有利于安全和稳定，但模型需要清楚：当前目录用环境块/read_dir，公共工具用 search_tools，市场复用用 search_hub_directory。

## 11. 建议

### 11.1 建一个组件能力清单

建议为每个 widget 建立统一 manifest，至少包含：

- widget type
- Go 支持类型
- config 字段
- search 支持
- 前端组件名
- extractor
- 示例 tag
- validator 是否存在

短期可以是 Markdown 表，长期可以是 JSON/YAML，再生成前后端类型和文档。

### 11.2 继续强化 SDK 编译校验

现在 `CompileAndValidate()` 是正确方向。后续建议继续加：

- route 重名和后缀错误定位到具体路由。
- widget config 未知字段提醒。
- select/options 与 validate oneof 一致性检查。
- display scene、search operator 与 Go 类型的更完整组合校验。
- OnSelectFuzzyMap 字段类型与 widget 类型匹配检查。

目标是让 `build_workspace` 一次返回尽可能完整的错误列表，方便模型一轮修复。

### 11.3 给错误增加“给模型看的修复建议”

例如：

```text
field Numbers (numbers): list widget requires item_type:number or item_type:text
建议：[]int 使用 widget:"name:数字列表;type:list;item_type:number"
```

模型修复效率会明显提高。

### 11.4 文档也要有测试

可以做几类轻量测试：

- prompt 中列出的 widget type 必须都在 SDK 支持列表里。
- SDK 支持列表必须在前端 `WidgetType` 中存在。
- 每个 SDK widget 必须有 validator。
- 每个非容器 widget 必须有前端 registry 或明确只后端使用。
- 文档示例代码可以做最小编译测试。

这能防止“文档指导模型写出已废弃代码”。

### 11.5 把案例库当成训练集维护

案例文档比抽象说明更影响模型产出。建议每类能力保留 1 到 3 个高质量黄金案例：

- 单表 CRUD。
- 多表关联。
- Form 文件处理。
- Python/CLI 工具。
- Chart 多系列统计。
- 定时任务。
- OnSelectFuzzy 联动。

模型生成质量很大程度取决于案例质量。

## 12. 最终结论

这套架构的核心逻辑可以压缩成一句话：

> 文档告诉模型规则，tools 限制模型动作，SDK 把代码编译成平台 schema，runtime 确认新版本能启动，app-server 把 schema 沉淀为函数资产，前端和 Agent 再基于同一份 schema 渲染与调用。

这个思路是成立的，而且比单纯 AI 写代码更有平台价值。下一阶段最值得投入的是“契约收敛”：把 skills、组件、schema、文档、前端注册、validator 这些分散约定变成可生成、可校验、可测试的一套平台协议。
