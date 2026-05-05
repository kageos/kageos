# 工作台、工具、文档与 SDK 的关系

这份文档用于给工作台大模型快速建立全局心智模型：工作台不是普通聊天框，tools 不是普通函数列表，SDK 也不是普通 Go 工具库。它们共同组成一条“生成代码 -> 编译启动 -> schema 注册 -> 前端渲染 -> Agent 调用 -> 资产沉淀”的闭环。

## 一句话

> 文档告诉模型规则，tools 限制模型动作，SDK 把 Go 代码编译成平台 schema，runtime 确认新版本能启动，app-server 把 schema 沉淀为函数资产，前端和 Agent 再基于同一份 schema 渲染与调用。

## 1. 总体链路

```text
用户需求
  ↓
工作台 / Mini Workstation
  ↓
agent-server：LLM 会话、prompt、tool loop
  ↓
文档：platform、SDK、case、当前环境
  ↓
tools：read/write/build/run/search/publish
  ↓
namespace/{user}/{app}/code/api/... Go 代码
  ↓
sdk/agent-app：Template、widget、handler、CompileAndValidate
  ↓
app-runtime：go build、启动新版本、等待 startup running/failed
  ↓
app-server：同步 function/service_tree/schema
  ↓
web：按 schema 渲染 Form/Table/Chart
  ↓
Agent 或用户通过 run_* / 页面继续使用
```

## 2. 各层职责

| 层 | 位置 | 职责 |
|---|---|---|
| 工作台前端 | `web/src/architecture/presentation` | 展示目录树、函数页面、Mini 工作台、SSE、工具结果、输出文件 |
| Agent 服务 | `core/agent-server` | 组织会话、加载 prompt/env、注册 tools、执行 stream loop |
| Prompt/文档 | `core/agent-server/prompt/system/prompt` | 说明平台边界、SDK 写法、案例、模式规则和当前环境 |
| Tools | `core/agent-server/service/tool_*.go` | 给模型提供受控读写、构建、执行、搜索、发布能力 |
| SDK | `sdk/agent-app` | 定义应用协议，注册 Form/Table/Chart，解析 widget tag，产出 schema，启动校验 |
| Runtime | `core/app-runtime` | 写盘、编译、容器版本、生命周期通知、启动等待 |
| App Server | `core/app-server` | app/function/service_tree 元数据、标准 API、日志、权限、定时任务 |
| 前端组件协议 | `WidgetType`、`widgetRegistry`、`widget-configs.ts` | 把 SDK schema 映射成 Vue 组件和提交值 |

## 3. 工作台是什么

工作台是“当前目录 + 当前函数树 + 当前文件 + 当前用户 + LLM + tools”的操作面。

它会把这些信息注入给模型：

- 当前用户、部门、时间。
- 当前 `full_code_path`。
- 当前目录下的子目录、函数、schema 摘要。
- 当前目录下的代码文件列表。
- 可读文档目录。
- 运行时预装 CLI 能力。
- 当前目录的 `init_.go` 内容。

所以模型生成代码时，不应该靠猜，而应该：

1. 看当前环境。
2. 必要时 `read_doc` 读 SDK 和案例。
3. `read_go_file` 或 `read_dir` 看已有代码。
4. 用 tools 修改代码。
5. `build_workspace` 编译部署。
6. 用 run_* 工具执行验证。

## 4. 文档和 prompt 的作用

文档/prompt 是模型的操作规程。

关键文档：

- `/system/prompt/platform-overview`：平台是什么，Form/Table/Chart 怎么选。
- `/system/prompt/platform-cross-cutting-capabilities`：平台横切能力，业务代码不要重造权限、审批、评论、日志、消息。
- `/system/prompt/sdk/agent-app-sdk-readme`：SDK 主文档。
- `/system/prompt/case_catalog/...`：业务案例和完整代码。
- `/system/prompt/mode/...`：不同模式的系统提示词和可用工具。
- `/system/prompt/doc/workspace-env-template`：当前工作环境模板。

文档不是可选参考，而是模型生成代码的约束来源。平台质量依赖文档、工具、SDK 校验三者一起工作。

## 5. Tools 的作用

tools 是模型的动作边界。

| 类型 | 示例 | 作用 |
|---|---|---|
| 工作区工具 | `read_go_file`、`read_dir`、`write_go_file`、`search_replace_file`、`delete_file`、`build_workspace` | 读写源码和编译部署 |
| 运行时工具 | `run_form_submit`、`run_table_search`、`run_table_create`、`run_table_update`、`run_chart_query`、`run_on_select_fuzzy` | 调用已注册函数并验证效果 |
| 平台工具 | `search_tools`、`search_hub_directory`、`copy_directory`、`publish_to_hub`、`record_workspace_event`、`web_search` | 搜索复用、Hub、发布、事件记录和外部信息 |

重要约定：

- `write_go_file` 只落盘，不编译。
- `build_workspace` 只构建部署，不写文件。
- 连续写多个文件后再统一 `build_workspace`。
- `search_tools` 主要搜内置工具和 `system` 用户下已注册函数；当前目录看环境或 `read_dir`，Hub 复用用 `search_hub_directory`。
- 执行函数前要确认 schema 字段，不要猜字段名。

## 6. SDK 的作用

`sdk/agent-app` 是平台应用协议层。

业务代码通过 SDK 声明：

- 目录：`PackageContext`。
- 函数：`.form`、`.table`、`.chart`。
- 类型：`FormTemplate`、`TableTemplate`、`ChartTemplate`。
- 字段：Go struct tag。
- UI：`widget:"name:标题;type:input"` 等。
- 搜索：`search:"like"`、`search:"in"` 等。
- 校验：`validate:"required"` 等。
- 回调：Table CRUD、OnSelectFuzzy、OnApiCreate。
- 处理逻辑：handler。
- 响应：`resp.Form`、`resp.Table`、`resp.Chart`。

SDK 转换链路：

```text
Go struct tag
  ↓ ParseModelWithType
FieldTags
  ↓ ValidateFieldTags
widget.Field
  ↓ functionschema.NewForm/NewTable/NewChart
FunctionSchema
  ↓ functionschema.Validate
ApiInfo
  ↓ onAppUpdate diff
app-server function/service_tree
```

## 7. SDK 启动校验

`app.Run()` 启动时先执行 `CompileAndValidate()`：

- 校验路由后缀与 Template 类型匹配。
- 调用 `getApis()` 解析所有 schema。
- 组件级 validator 校验 widget 参数和 Go 类型。
- `functionschema.Validate()` 校验最终 schema。
- 多个错误会聚合返回。
- 失败时 SDK 发布 `startup failed` 生命周期事件。

因此 `build_workspace` 不只是 Go 编译成功，还会等待新版本启动结果。如果 SDK schema compile 失败，新版本不会被认为启动成功，模型能拿到错误继续修。

## 8. Runtime 和 App Server 的关系

`build_workspace` 的真实链路：

```text
agent-server build_workspace
  ↓
app-server /workspace/api/v1/app/update
  ↓
app-runtime UpdateApp
  ↓
go mod tidy + go build
  ↓
创建新版本容器
  ↓
等待 SDK startup running/failed
  ↓
成功后停止旧版本
  ↓
SDK onAppUpdate 返回 API diff
  ↓
app-server 同步 function/service_tree/schema
```

runtime 负责源码、构建、容器和生命周期。

app-server 负责平台元数据、标准 API 和治理能力。

## 9. 前端如何消费 schema

前端只消费 schema，不理解 Go 代码。

每个字段主要包含：

```text
code        json 字段名
name        展示名
data.type   Go 类型推断出的数据类型
widget.type 组件类型
config      widget 配置
search      搜索能力
display     list/create/update 展示场景
children    嵌套 form/table 子字段
```

一个组件要完整可用，至少需要：

- SDK widget 类型。
- SDK widget config struct。
- SDK validator。
- SDK 支持列表。
- 前端 `WidgetType`。
- 前端 `widget-configs.ts`。
- 前端 Vue 组件。
- 前端 registry 注册。
- FieldExtractor 提交逻辑。
- SDK 文档示例。
- 测试。

例如自由输入数组：

```go
Numbers []int    `json:"numbers" widget:"name:数字列表;type:list;item_type:number"`
Names   []string `json:"names" widget:"name:文本列表;type:list;item_type:text"`
```

对应链路：

```text
SDK TypeList + List + validateListWidget
  ↓
schema widget.type=list, config.item_type=number/text
  ↓
前端 WidgetType.LIST + ListWidgetConfig + ListWidget.vue
  ↓
FieldExtractorRegistry 提交 number[] / string[]
```

## 10. 这套设计的本质

它不是“AI 生成一个页面”，而是“AI 生成一个平台可治理的企业应用能力”。

价值点：

- 生成结果必须落到 Form/Table/Chart 这三类标准资产。
- 字段和组件必须进入统一 schema。
- 同一份 schema 同时服务前端渲染和 Agent 调用。
- build/startup 会挡住错误 schema。
- 函数会沉淀到 service_tree/function/Hub，而不是一次性代码。

## 11. 主要风险和建议

最大风险是契约分散：

- SDK widget。
- 前端 WidgetType。
- 前端组件。
- extractor。
- 文档。
- prompt。
- 测试。

这些地方必须保持一致，否则模型会生成前端不支持或 SDK 不接受的代码。

建议：

1. 为每个 widget 建统一 manifest，记录 type、Go 支持类型、config、validator、前端组件、extractor 和示例。
2. 继续强化 `CompileAndValidate()`，让错误尽量在 build/startup 阶段暴露。
3. 错误消息加“给模型看的修复建议”。
4. 给文档加测试：文档里的 widget type 必须存在于 SDK 和前端。
5. 把案例库当成黄金训练集维护，每类能力保留高质量完整案例。

## 12. 结论

当前方向是对的。真正的核心不是模型会写代码，而是平台把模型写出的代码约束成可编译、可渲染、可调用、可治理、可复用的标准应用资产。下一阶段应重点把组件协议、文档、前端注册和 SDK validator 继续收敛成可测试的统一契约。
