# 定时任务 SDK 与异步执行改造方案

## 1. 架构结论

定时任务能力应该收敛成一个独立部署的中心服务：

```text
timer-scheduler
```

所有需要定时任务的服务都通过平台内部 SDK 连接这个服务，完成任务创建、更新、暂停、恢复、取消、手动触发、查询执行记录、消费触发事件、回写执行状态等操作。

这个架构下，`timer-scheduler` 只负责通用调度能力，不负责具体业务逻辑：

- 负责保存任务定义和调度计划。
- 负责计算 `next_run_at`。
- 负责到点创建 execution。
- 负责派发触发事件。
- 负责维护 execution 状态机。
- 负责接收业务服务回写的 started/finished 状态。
- 负责通知、审计、基础分析字段。

它不应该知道这些业务细节：

- Agent 如何创建会话。
- 表单如何提交。
- 表格如何查询。
- 报表如何生成。
- 某个业务动作需要哪些私有参数。

这些由业务服务自己实现。定时任务侧只做触发和状态流转。

## 2. 服务边界

### 2.1 timer-scheduler 的职责

`timer-scheduler` 是唯一的调度服务实例或唯一的调度服务集群。集群部署时也要通过租约、分片或数据库 claim 保证同一个 due task 只被一个 scheduler 派发。

核心职责：

- Control Plane：任务 CRUD、查询、手动触发、暂停恢复、取消。
- Schedule Engine：计算到期任务、生成 execution、推进下一次运行时间。
- Dispatch Plane：向执行器服务发布通用触发事件。
- State Plane：接收执行器服务回写 started/finished，维护 execution 和 task 状态。
- Observability：提供 title、category、tags、metadata 等通用分析字段。

### 2.2 业务服务的职责

业务服务只消费自己能处理的任务。

例如：

- `agent-server` 订阅 `executor_key=agent.session`。
- `app-server` 订阅 `executor_key=app.function`。
- 未来报表服务订阅 `executor_key=report.generate`。

业务服务收到触发事件后：

1. 根据 `execution_id` 做幂等处理。
2. 读取 `executor_payload`。
3. 从自己的业务表加载具体配置。
4. 执行业务逻辑。
5. 调用 SDK 回写 execution started/finished。
6. 可选发布或消费完成事件做后续业务处理。

`timer-scheduler` 不调用业务实现，不 import 业务 service，不解析业务 payload。

## 3. 触发和状态流转

### 3.1 创建任务

```text
business service
  -> scheduledsdk.CreateTask
  -> timer-scheduler
  -> save task
  -> return task
```

创建请求里必须有：

- `title`：任务标题，用于展示。
- `executor_key`：执行器路由 key。
- `schedule`：调度计划。
- `executor_payload`：执行器私有 payload，调度侧不解析。

可以有：

- `description`：描述。
- `category`：分类。
- `tags`：标签。
- `metadata`：轻量描述元数据。
- `source_type/source_ref`：来源定位。
- `notify_users/notify_departments`：通知对象，落库用逗号分隔字符串，不使用 JSON。

### 3.2 到点触发

```text
timer-scheduler tick
  -> claim due task
  -> create execution(status=queued)
  -> set task.inflight_execution_id
  -> publish scheduled.execution.requested
```

触发事件需要包含：

- `event_id`
- `task_id`
- `execution_id`
- `executor_key`
- `scheduled_at`
- `source_type`
- `source_ref`
- `metadata`
- `executor_payload`

业务服务按 `executor_key` 消费事件。消费方式可以是 JetStream subject filter、queue group、HTTP pull 或 SDK worker 封装，但协议语义要保持一致。

### 3.3 执行开始

```text
business worker
  -> MarkExecutionStarted(task_id, execution_id, worker_id, executor_run_id)
  -> timer-scheduler: queued -> running
```

`executor_run_id` 是业务执行实例 ID。Agent 场景可以放 session id，表单场景可以放操作日志 ID，报表场景可以放 report job id。调度侧只保存，不解释。

### 3.4 执行完成

```text
business worker
  -> MarkExecutionFinished(status=success/failed/timeout/cancelled)
  -> timer-scheduler:
       update execution
       clear task.inflight_execution_id
       compute next_run_at
       update task status
       publish scheduled.execution.finished
```

完成事件可以被前端通知服务、审计服务或业务服务继续消费。

### 3.5 超时和恢复

`timer-scheduler` 需要定期扫描 stale execution：

- `queued` 太久未被 worker started：重新派发或标记 timeout。
- `running` heartbeat 过期：标记 timeout 或重新进入 queued。
- `inflight_execution_id` 指向终态 execution：清理 inflight。

这些是调度平台能力，不应该分散在每个业务服务里。

## 4. SDK 定位

`pkg/scheduledsdk` 是平台内部 SDK。它定义稳定类型、Client 和 Adapter interface。业务服务通过 SDK 调用 `timer-scheduler`，不直接操作调度表、NATS subject 或状态机。

```go
payload := json.RawMessage(`{"business_ref":"agent_scheduled_task:123"}`)

task, err := client.CreateTask(ctx, scheduledsdk.CreateTaskRequest{
    Title:           "每日项目巡检",
    Description:     "检查项目风险并生成摘要",
    Category:        "inspection",
    Tags:            []string{"project", "daily"},
    ExecutorKey:     "agent.session",
    ExecutorPayload: payload,
    Metadata: map[string]string{
        "app_code": "agent",
        "team":     "platform",
    },
    Schedule:   scheduledsdk.Every(3600),
    SourceType: "workspace",
    SourceRef:  "/liubeiluo/work/project",
})
```

业务服务消费触发事件时再解析 `executor_payload`：

```go
type AgentTaskPayload struct {
    BusinessRef string `json:"business_ref"`
}
```

推荐 payload 只放稳定引用，例如 `business_ref`。具体 prompt、表单参数、报表配置等大而易变的业务配置，应该存业务自己的表。

## 5. 核心 API 草案

### 5.1 Client

```go
func NewClient(opts Options) *Client

func (c *Client) CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error)
func (c *Client) UpdateTask(ctx context.Context, taskID int64, req UpdateTaskRequest) (*Task, error)
func (c *Client) PauseTask(ctx context.Context, taskID int64) error
func (c *Client) ResumeTask(ctx context.Context, taskID int64) error
func (c *Client) CancelTask(ctx context.Context, taskID int64) error
func (c *Client) RunNow(ctx context.Context, taskID int64) (*Execution, error)

func (c *Client) GetTask(ctx context.Context, taskID int64) (*Task, error)
func (c *Client) ListTasks(ctx context.Context, req ListTasksRequest) (*ListTasksResponse, error)
func (c *Client) GetExecution(ctx context.Context, taskID, executionID int64) (*Execution, error)
func (c *Client) ListExecutions(ctx context.Context, taskID int64, req ListExecutionsRequest) (*ListExecutionsResponse, error)

func (c *Client) PublishExecutionRequested(ctx context.Context, event ExecutionRequestedEvent) error
func (c *Client) MarkExecutionStarted(ctx context.Context, req MarkExecutionStartedRequest) error
func (c *Client) MarkExecutionFinished(ctx context.Context, req MarkExecutionFinishedRequest) error
```

`PublishExecutionRequested` 通常只给 `timer-scheduler` 内部使用。普通业务服务主要使用 `CreateTask/UpdateTask/...` 和 `MarkExecutionStarted/Finished`。

### 5.2 Adapter

```go
type Adapter interface {
    CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error)
    UpdateTask(ctx context.Context, taskID int64, req UpdateTaskRequest) (*Task, error)
    PauseTask(ctx context.Context, taskID int64) error
    ResumeTask(ctx context.Context, taskID int64) error
    CancelTask(ctx context.Context, taskID int64) error
    RunNow(ctx context.Context, taskID int64) (*Execution, error)
    GetTask(ctx context.Context, taskID int64) (*Task, error)
    ListTasks(ctx context.Context, req ListTasksRequest) (*ListTasksResponse, error)
    GetExecution(ctx context.Context, taskID, executionID int64) (*Execution, error)
    ListExecutions(ctx context.Context, taskID int64, req ListExecutionsRequest) (*ListExecutionsResponse, error)
}
```

第一阶段 Adapter 可以是 HTTP、JetStream command，也可以是测试用 fake adapter。不要再做 agent/function 专属 adapter 接口。

## 6. 核心数据模型

### 6.1 Task

```go
type Task struct {
    ID                  int64
    Title               string
    Description         string
    Category            string
    Tags                []string
    ExecutorKey         string
    Status              TaskStatus
    Schedule            Schedule
    NextRunAt           *time.Time
    RunCount            int
    InflightExecutionID int64
    SourceType          string
    SourceRef           string
    Metadata            map[string]string
    ExecutorPayload     json.RawMessage
}
```

### 6.2 Execution

```go
type Execution struct {
    ID             int64
    TaskID         int64
    ExecutorKey    string
    Status         ExecutionStatus
    ExecutorRunID  string
    ScheduledAt    time.Time
    StartedAt      *time.Time
    FinishedAt     *time.Time
    WorkerID       string
    LeaseUntil     *time.Time
    HeartbeatAt    *time.Time
    Attempt        int
    DurationMillis int64
    OutputSummary  string
    ResultPayload  json.RawMessage
    ErrorMessage   string
    TraceID        string
}
```

`ResultPayload` 也是执行器私有结果。调度侧可以保存和返回，但不解析。

### 6.3 状态机

```text
task:
  pending -> paused
  pending -> done
  pending -> failed
  pending -> cancelled
  paused  -> pending

execution:
  queued -> running
  queued -> timeout
  queued -> cancelled
  running -> success
  running -> failed
  running -> timeout
  running -> cancelled
```

状态更新必须做条件更新，例如 `queued -> running` 只能在当前状态仍为 queued 时成功，避免重复消费导致状态回退。

## 7. metadata 是否合适

结论：可以保留 `metadata`，但必须限定语义。它不应该承载业务执行参数。

### 7.1 metadata 适合放什么

`metadata` 适合放调度平台、前端列表、分析报表可能会读取的轻量描述信息：

- `app_code`
- `team`
- `env`
- `priority`
- `owner`
- `cost_center`

建议 SDK 类型使用 `map[string]string`，而不是任意 JSON。这样可以减少业务方把复杂参数塞进 metadata 的冲动，也便于索引、过滤和展示。

### 7.2 metadata 不适合放什么

不要在 metadata 里放：

- prompt。
- SQL。
- 表单提交参数。
- Agent 工具策略。
- 报表生成配置。
- 大 JSON。
- 业务状态快照。

这些都属于业务执行参数，应该放在业务服务自己的存储里，然后通过 `executor_payload.business_ref` 引用。

### 7.3 为什么需要 executor_payload

如果只用 `metadata`，字段含义会很快失控。不同业务会把它当成参数、上下文、展示信息、过滤字段混用，最后调度侧无法判断哪些字段能索引、哪些字段能展示、哪些字段需要保密。

所以需要两个字段：

- `metadata`：调度侧可读，轻量、描述性、可展示、可过滤。
- `executor_payload`：调度侧不可解释，原样投递给业务执行器。

这个命名比单纯叫 `payload` 更清楚，因为它明确属于 executor，不属于 scheduler。

### 7.4 可选替代名

可选名包括：

- `payload`：太泛，容易被理解成 scheduler 也要理解。
- `context`：容易和 request context、执行上下文混淆。
- `business_payload`：语义也可以，但调度平台更关心执行器路由，`executor_payload` 更准确。
- `metadata`：适合描述信息，不适合执行参数。

推荐最终使用：

```text
metadata          调度侧可读描述信息
executor_payload  执行器私有输入
result_payload    执行器私有输出
```

## 8. 事件协议

第一版可以先定义通用 subject：

```text
scheduled.execution.requested
scheduled.execution.started
scheduled.execution.finished
scheduled.execution.failed
```

事件里必须带 `executor_key`。如果后续需要减少消费者过滤成本，可以扩展成：

```text
scheduled.execution.requested.{executor_key}
scheduled.execution.finished.{executor_key}
```

subject 命名不应该出现 `agent`、`function` 这种业务类型。业务类型只体现在 `executor_key` 上。

## 9. 一致性策略

`CreateExecution + set inflight_execution_id + publish event` 不是天然原子操作。

推荐长期使用 outbox：

```text
DB transaction:
  insert execution
  update task.inflight_execution_id
  insert outbox event

publisher:
  scan unpublished outbox
  publish event
  mark published
```

第一版如果不做 outbox，至少需要补偿：

- publish 失败时把 execution 标记 failed，并清空 inflight。
- 定时扫描 queued execution，如果长时间没有 worker started，就重新投递或 timeout。

## 10. 落地步骤

### 阶段一：收敛 SDK 协议

- `pkg/scheduledsdk` 改为通用 `CreateTask`。
- 移除旧的业务专属创建方法。
- 移除 SDK 内的 Agent/Function 专属字段。
- 增加 `executor_key`、`metadata`、`executor_payload`。
- subject 常量改成通用 execution subject。

### 阶段二：建设 timer-scheduler 服务

- 新增独立 `timer-scheduler` 服务。
- 建立 task、execution、outbox 表。
- 提供控制面 API 或 JetStream command adapter。
- 实现 due task 扫描、claim、execution 创建、事件投递。
- 实现 started/finished 状态回写。

### 阶段三：业务服务接入执行器

- agent-server 订阅 `executor_key=agent.session`。
- app-server 订阅 `executor_key=app.function`。
- 各服务只解析自己的 `executor_payload`。
- 各服务通过 SDK 回写 execution 状态。

### 阶段四：迁移存量任务

- 存量业务任务统一补偿创建对应的 timer task，并回填 `timer_task_id`。
- 业务表只保存业务配置和执行记录，timer task 只保存 `business_ref` / `executor_payload`。
- 前端统一按中心 timer task 和业务执行记录展示。

## 11. 当前实现进展

当前已经先完成第一版协议层和中心服务骨架：

- SDK 控制面已经转向通用任务。
- 调度侧协议不再暴露 Agent/Function 专属 create API。
- `scheduledsdk` 支持 HTTP adapter，业务服务可用 `BaseURL` 指向独立服务。
- 新增 `core/timer-scheduler`，包含 task、execution、outbox 表模型。
- 新增调度派发事务：claim due task、创建 queued execution、设置 inflight、写 outbox。
- 新增 outbox publisher，把 pending outbox 发布到 NATS subject，成功后标记 published，失败记录错误并保留待重试。
- 新增 started/finished 状态回写，完成后清理 inflight 并计算下一次运行时间。
- 新增 `/timer/api/v1` HTTP 控制面。
- 新增 `core/timer-scheduler/cmd/app` 独立启动入口。
- 新增 `scheduledsdk.Worker`，业务服务可按 `executor_key` 消费触发事件并自动回写 started/finished。
- agent-server 已先接入中心调度：
  - `scheduled_agent_task` 继续保存 Agent 业务配置。
  - 新增 `timer_task_id` 映射中心任务。
  - 创建 Agent 定时任务时同步创建 `executor_key=agent.session` 的 timer task。
  - agent-server 不再自行扫描 due task，改为 worker 消费 `scheduled.execution.requested` 后按 `executor_payload.business_ref` 执行业务逻辑。
- app-server 已接入中心调度：
  - `scheduled_task` 继续保存表单/表格等业务执行配置。
  - 新增 `timer_task_id` 映射中心任务。
  - 创建业务定时任务时同步创建 `executor_key=app.function` 的 timer task。
  - app-server 启动后作为业务执行器消费 `scheduled.execution.requested`，按 `executor_payload.task_id` 回查本地业务任务并执行。
  - 本地执行记录仍写入 `scheduled_task_execution`，用于兼容现有 API 和前端查询。
- 旧本地轮询调度入口和历史兼容开关已移除；不存在业务服务本地调度模式。
- `timer-scheduler` 是唯一调度服务，必须单独部署；业务服务只通过 SDK 注册任务并消费触发事件。
- app-server/agent-server 的 timer worker 订阅失败会在启动阶段直接返回错误，避免服务启动成功但无人消费触发事件。
- 生产 scheduler 容器已改为启动 `timer-scheduler`，健康检查改为探测 `http://127.0.0.1:9108/health`。
- 文档明确 `timer-scheduler` 是唯一调度服务。
- 文档明确业务逻辑由业务服务订阅执行，不放在调度侧。

SDK 连接示例：

```go
client := scheduledsdk.NewClient(scheduledsdk.Options{
    BaseURL: "http://127.0.0.1:9108/timer/api/v1",
})
```

Worker 连接示例：

```go
worker, err := scheduledsdk.NewWorker(scheduledsdk.WorkerOptions{
    Client:      client,
    NATSConn:    natsConn,
    ExecutorKey: "agent.session",
    Handler: func(ctx context.Context, event scheduledsdk.ExecutionRequestedEvent) (*scheduledsdk.ExecutionResult, error) {
        // 业务侧只解析自己的 executor_payload
        return &scheduledsdk.ExecutionResult{Status: scheduledsdk.ExecutionStatusSuccess}, nil
    },
})
```

后续还需要继续补：

- JetStream 化：当前 publisher 先使用 NATS subject，后续把 outbox 投递升级为 JetStream ack。
- app-server 和 agent-server 存量任务补偿迁移脚本：为历史本地任务批量创建对应 timer task 并回填 `timer_task_id`。
- finished 事件订阅侧：按需把 `scheduled.execution.finished` 接到通知、审计或前端聚合视图。
- 生产文档与 `aosctl` 模板继续补齐 timer-scheduler 的描述和运维命令。
