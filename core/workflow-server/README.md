# Workflow Server

`workflow-server` 是 AI-Agent-OS 的标准能力编排服务。

它的第一阶段目标不是做通用自动化平台，而是把平台内已有的 `Form` 串起来，让一个 Form 的输出成为下一个 Form 的输入，并把整个过程沉淀为可复用、可运行、可发布的 workflow。

## 1. 与 flow-server 的边界

`workflow-server` 和 `flow-server` 必须保持边界清晰。

| 服务 | 负责什么 | 不负责什么 |
| --- | --- | --- |
| `workflow-server` | 自动化编排、节点执行、输入输出映射、运行状态 | 审批策略、审批人、审批流转 |
| `flow-server` | 审批控制、动作拦截、审批通过后执行 | 多 Form 自动化编排 |

后续如果 workflow 需要人工审批，应通过一个 `approval.wait` 节点调用 `flow-server`，而不是在 `workflow-server` 里重做审批系统。

## 2. MVP 目标

第一版只支持：

- 手动触发。
- 顺序执行。
- 节点类型：`form.submit`。
- 表达式：`$ref` 和 `$const`。
- 运行记录：workflow run 与 step run。
- 失败即停止。

暂不支持：

- 画布。
- 并行。
- 条件。
- 循环。
- 审批。
- 定时触发。
- 外部 HTTP 节点。

## 当前实现状态

当前 `codex/workflow-mvp` 分支已经落地后端 MVP 骨架：

- `pkg/workflowexpr`：支持 `$ref` / `$const` 的表达式解析和校验。
- `definition`：支持 `workflow.v1`、`nodes + edges`、sequence DAG 校验。
- `executor`：支持 executor registry，已注册 `form.submit`。
- `service`：支持 workflow 创建、更新、发布、运行、run 查询、step 查询、取消。
- `server`：提供 `/workflow/api/v1` HTTP API。
- `model/repository`：持久化 definition、version、run、step run、run event。
- `core/cmd/main`：统一启动入口已接入 workflow-server。
- `api-gateway`：已新增 `/workflow` 路由。

## 3. 推荐目录结构

```text
core/workflow-server/
  api/v1/
  cmd/app/
  config/
  dto/
  model/
  repository/
  service/
  runner/
  executor/
  README.md
```

建议公共表达式能力放在：

```text
pkg/workflowexpr/
```

这样后续 `agent-server`、Hub 校验、导入导出工具也能复用同一套表达式解析和校验逻辑。

## 4. 核心模块

### 4.1 Definition Service

负责 workflow 定义的 CRUD、发布和版本管理。

关键规则：

- 草稿可以编辑。
- 发布后生成不可变 version。
- 每次 run 绑定一个具体 version。
- 禁用 workflow 后不能新建 run，但历史 run 可查询。

### 4.2 Graph Validator

负责校验定义是否合法。

MVP 校验：

- `schema_version` 必须支持。
- node ID 唯一。
- edge 引用的 node 必须存在。
- 只允许单链路 DAG。
- 不允许环。
- `form.submit` 节点必须有 `ref`。
- 输入表达式只能使用 `$ref` 和 `$const`。

后续扩展：

- 条件分支校验。
- 并行 merge 校验。
- foreach 子图校验。
- 子工作流引用校验。

### 4.3 Expression Engine

负责把 workflow 上下文映射成节点输入。

MVP 上下文：

```json
{
  "input": {},
  "steps": {
    "node_id": {
      "input": {},
      "output": {}
    }
  }
}
```

支持：

```json
{ "$const": "fixed value" }
{ "$ref": "input.file" }
{ "$ref": "steps.extract.output.text" }
```

### 4.4 Node Executor Registry

负责按节点类型找到执行器。

建议接口：

```go
type NodeExecutor interface {
	Type() string
	Validate(ctx context.Context, node Node, definition Definition) error
	Execute(ctx context.Context, input NodeInput, runtime RuntimeContext) (NodeOutput, error)
}
```

MVP：

```text
form.submit -> FormSubmitExecutor
```

后续：

```text
table.search
table.create
table.update
chart.query
agent.run
approval.wait
condition
foreach
merge
subworkflow.run
```

### 4.5 Runner

负责执行 workflow。

MVP 流程：

```text
create run
load version
validate graph
build context
for each node:
  resolve input
  create step run
  execute node
  persist output
  update context
compute workflow output
finish run
```

### 4.6 State Machine

run 状态预留：

```text
pending
running
waiting
success
failed
cancelled
timeout
```

step 状态预留：

```text
pending
running
waiting
success
failed
skipped
cancelled
```

MVP 不一定全部用到，但数据库和 DTO 应该提前按这些状态设计。

## 5. app-server 调用边界

`workflow-server` 不直接调用 `app-runtime`。

`form.submit` executor 应通过 `app-server` 的标准 API 或内部 client 调用 Form，原因：

- 保留现有权限链路。
- 保留现有文件字段处理。
- 保留现有 Function/Form schema 行为。
- 保留现有执行日志和运行时转发逻辑。

建议抽象：

```go
type FormSubmitClient interface {
	Submit(ctx context.Context, req FormSubmitRequest) (FormSubmitResponse, error)
}
```

## 6. API 草案

```text
POST   /workflow/api/v1/workflows
GET    /workflow/api/v1/workflows
GET    /workflow/api/v1/workflows/:id
PUT    /workflow/api/v1/workflows/:id
POST   /workflow/api/v1/workflows/:id/publish
POST   /workflow/api/v1/workflows/:id/run
GET    /workflow/api/v1/runs/:run_id
GET    /workflow/api/v1/runs/:run_id/steps
POST   /workflow/api/v1/runs/:run_id/cancel
```

## 7. MVP 验收标准

一个合格的第一版必须能完成：

1. 用户创建 workflow 草稿。
2. 用户选择两个 Form，配置 A 输出到 B 输入。
3. 用户发布 workflow。
4. 用户手动运行 workflow。
5. 系统成功执行 A，再执行 B。
6. 用户能看到每一步 input/output/status/duration。
7. A 或 B 失败时，run 状态为 failed，并能定位失败节点。

## 8. 后续扩展原则

- 控制流扩展通过 graph validator 和 runner 扩展。
- 节点能力扩展通过 executor registry 扩展。
- 数据映射扩展通过 expression engine 函数白名单扩展。
- 定时触发复用 `timer-scheduler`。
- 人工审批复用 `flow-server`。
- 商业分发复用 Hub。
