# 代码体量与重构体检记录

本文记录当前删减后的代码体量判断、代码为什么仍然偏多、以及后续可按优先级推进的重构方向。

统计口径：

- 基于当前工作区实际存在的 Git 跟踪文件和未跟踪新文件。
- 不包含 `.gitignore` 排除的依赖、构建产物等。
- 二进制图片不计入行数。
- Swagger、lockfile、自动生成声明、内置 bundle 等单独视为“生成物或数据型文件”，不直接用于评价手写业务代码质量。

## 当前体量判断

当前项目仍属于中大型平台型工程。

粗略数据：

| 类别 | 文件数 | 行数 |
| --- | ---: | ---: |
| 源码 | 约 1,138 | 约 202,396 |
| 文本文档和配置总计 | 约 1,385 | 约 257,298 |
| 去掉明显生成物后的手写源码 | 约 1,131 | 约 193,572 |

主要语言和文件类型：

| 类型 | 文件数 | 行数 |
| --- | ---: | ---: |
| Go | 538 | 约 89,599 |
| Vue | 119 | 约 52,978 |
| TypeScript | 411 | 约 50,076 |
| Markdown | 139 | 约 28,991 |
| JSON | 27 | 约 18,598 |
| YAML | 27 | 约 5,312 |
| Shell | 27 | 约 2,106 |
| SQL | 29 | 约 2,106 |

主要目录分布：

| 目录 | 文件数 | 行数 |
| --- | ---: | ---: |
| `web/src` | 约 533 | 约 104,945 |
| `core/agent-server` | 约 175 | 约 30,291 |
| `core/app-server` | 约 89 | 约 27,610 |
| `sdk/agent-app` | 约 104 | 约 17,364 |
| `core/app-storage` | 约 32 | 约 10,550 |
| `core/app-runtime` | 约 48 | 约 8,323 |

去掉 Swagger、lockfile、生成声明、prompt embed 等明显生成物后，主要手写源码集中在：

| 目录 | 文件数 | 行数 |
| --- | ---: | ---: |
| `web/src` | 约 521 | 约 103,831 |
| `core/agent-server` | 约 109 | 约 16,632 |
| `sdk/agent-app` | 约 94 | 约 14,938 |
| `core/app-server` | 约 83 | 约 14,011 |
| `core/app-runtime` | 约 47 | 约 8,227 |

## 为什么功能删减后代码仍然多

这不是普通 CRUD 项目，而是平台型产品。一个看起来很小的功能，在代码里通常会展开为一整条链路：

- 后端 API、DTO、Service、Repository、模型和测试。
- 前端页面、组件、composable、store、router 状态、类型定义和测试。
- Agent 工作台的会话、LLM 调用、工具循环、SSE、结果投影、文件上传和历史记录。
- 应用运行时、SDK、工具库、表单/表格/图表协议、回调机制。
- Swagger、配置模板、部署脚本、内置种子数据和能力包。
- 兼容旧接口、旧字段、旧路由和历史数据格式的保留逻辑。

所以“产品功能看起来不多”和“代码仍然二十万行级别”并不冲突。删功能之后，往往还会留下通用架构、兼容层、生成物、运行时和前端状态管理代码。

## 当前是否属于屎山

当前判断：不是屎山，但有明显的复杂度堆积和删减后的遗留结构。

不是屎山的原因：

- 能看到清晰的技术分层：`core`、`pkg`、`dto`、`sdk`、`web`。
- 后端有 Service、Repository、DTO、SDK 等边界。
- 前端有 architecture/domain/infrastructure/presentation/shared 等分层意识。
- 有一定测试体量，测试代码约 2 万行级别。
- 代码体量集中在少数主链路模块，不是完全无序地散落增长。

值得警惕的信号：

- 大文件较多。约 81 个代码文件超过 500 行，约 34 个超过 800 行，约 7 个超过 1200 行。
- 前端 `web/src` 仍然超过 10 万行，是当前最大复杂度来源。
- 工作台、表格、表单、选择器、文件上传等组件承载了较多 UI、状态、兼容和业务流程。
- 后端的工作台会话编排层承担了会话、消息、LLM、工具循环、SSE、取消控制、结果投影等多种职责。
- `兼容`、`临时`、`legacy`、`deprecated` 等痕迹仍然存在，删减后需要二次清理。
- `map[string]interface{}`、`interface{}`、`any` 在动态协议边界较多，需要确认动态性是否被限制在边界层。

更准确的描述是：当前代码不是不可救的泥团，而是一个曾经目标较大的平台工程，在产品收敛之后还没有完成工程结构的第二轮收敛。

## 明显生成物和数据型文件

以下类型会显著增加行数，但不应直接算作业务复杂度：

- `*/docs/docs.go`
- `*/docs/swagger.json`
- `*/docs/swagger.yaml`
- `web/package-lock.json`
- `web/components.d.ts`
- `*.capability-bundle.json`
- `core/agent-server/prompt/system_prompt_source.go`
- `core/agent-server/prompt/embed.go`

本轮粗看这些文件约 18 个，约 28,634 行。

建议后续统计代码体量时至少给出两组数据：

- 全量文本行数。
- 排除生成物和数据型文件后的手写源码行数。

## 高优先级重构方向

### 1. 前端大组件瘦身

优先关注：

- `web/src/architecture/presentation/components/MiniWorkstation.vue`
- `web/src/architecture/presentation/views/TableView.vue`
- `web/src/architecture/presentation/views/FormView.vue`
- `web/src/architecture/presentation/views/WorkspaceView.vue`
- `web/src/architecture/presentation/widgets/SelectWidget.vue`
- `web/src/architecture/presentation/widgets/MultiSelectWidget.vue`
- `web/src/architecture/presentation/widgets/FilesWidget.vue`

目标不是为了拆而拆，而是把以下职责分离：

- 纯 UI 结构。
- 状态派生和 computed。
- 路由/URL 参数同步。
- 后端 API 调用。
- 兼容旧数据的转换逻辑。
- 复杂交互流程。
- 文件、附件、预览、下载等副作用。

建议做法：

- 大 SFC 保留为组装层。
- 复杂状态移动到 composable。
- 纯转换函数移动到 `utils` 并补测试。
- 可复用 UI 拆成更小的 presentation component。
- 每次只拆一条业务线，避免大面积重排。

### 2. 工作台会话编排拆分

优先关注：

- `core/agent-server/service/workspace_chat_service.go`

当前该类文件承担了较多中枢职责：

- 会话创建和查询。
- 消息入库和历史拼装。
- 用户文件引用拼装。
- LLM 请求构造。
- 工具调用循环。
- SSE 事件发送。
- 取消控制。
- 工具结果和 artifact/display fields 投影。
- 运行时状态写入。

建议拆分方向：

- `WorkspaceChatSessionService`：会话生命周期。
- `WorkspaceChatMessageBuilder`：历史消息、文件引用、上下文策略。
- `WorkspaceToolLoopRunner`：工具循环和 LLM round 编排。
- `WorkspaceEventPublisher`：SSE/后台任务事件统一出口。
- `WorkspaceResultProjector`：工具结果、artifact、display fields、状态推导。
- `WorkspaceRuntimeStateUpdater`：运行时状态落库和更新。

拆分时要保持外部 API 不变，优先抽纯函数和内部小服务，配合现有测试逐步推进。

### 3. 兼容层和临时逻辑二次清理

删功能后，兼容逻辑容易成为虚胖来源。

建议建立清单：

- 仍被当前前端调用的兼容字段。
- 只为历史数据迁移保留的兼容字段。
- 已无调用方的临时接口。
- 已无产品路径的旧路由、旧 query 参数、旧 callback。
- 旧企业版、权限、License、消息、备份相关残留。

处理策略：

- 无调用方且无数据迁移价值的直接删除。
- 仍需兼容历史数据的集中到 adapter/normalizer。
- 对外协议字段保留，但内部尽早转换成新结构。
- 新代码避免继续向旧字段写逻辑。

### 4. 动态类型边界收紧

项目里动态 JSON 是合理存在的，尤其是表单、表格、工具调用和 LLM 协议。但需要确认动态性只存在于边界层。

建议：

- API 入参出参尽量使用明确 DTO。
- 工具调用参数在进入核心逻辑前完成 schema 校验和类型转换。
- 前端 `any` 优先替换为领域类型、`unknown` 加 type guard、或局部协议类型。
- `map[string]interface{}` 只用于协议边界，进入服务层后转换为明确结构。
- 对高频转换函数补测试，避免重构时破坏兼容。

### 5. 统计和治理工具化

建议新增一个脚本，定期输出：

- 全量文本行数。
- 手写源码行数。
- 生成物/数据型文件行数。
- 大于 500/800/1200 行的文件列表。
- 按目录的源码行数。
- `TODO/FIXME/兼容/临时/legacy/deprecated` 数量趋势。

这比单纯看总行数更有用，可以避免被 Swagger、lockfile、bundle 或文档误导。

## 建议的优化顺序

第一阶段：建立口径，不急着大拆。

- 固化代码体量统计脚本。
- 排除生成物，形成每次优化前后的可比数据。
- 列出当前大文件 Top 50。
- 标记“保留生成物”“候选拆分”“候选删除”“候选类型化”四类。

第二阶段：先清虚胖。

- 删除已无调用方的兼容接口、旧 DTO、旧路由、旧配置。
- 清理删减后不再可达的前端页面、API 封装、store、composable。
- 把 seed/bundle/prompt embed 从业务统计里剥离。

第三阶段：拆主链路大文件。

- 先拆 `MiniWorkstation.vue` 和 `workspace_chat_service.go`。
- 每次拆分都保持行为不变，先抽纯逻辑，再移动副作用。
- 拆完补少量关键测试，不追求一次性全覆盖。

第四阶段：收紧协议和类型。

- 对工具调用、表单/表格协议、工作区路径、文件引用等核心对象建立更明确的类型。
- 前后端对同一协议字段形成一致命名和转换位置。
- 减少跨层传递裸 `any`/`interface{}`。

## 判断标准

后续重构不是为了把行数压到很低，而是为了降低修改成本。

可以用这些结果判断是否变好：

- 新功能改动涉及文件数减少。
- 大组件行数下降，但抽出的模块职责清晰。
- 兼容逻辑集中，而不是散在业务流程里。
- 动态类型只出现在协议边界。
- 删除一个功能时能顺着模块边界删干净。
- 工作台主链路可以单独测试消息构造、工具循环、结果投影。
- 前端页面组件更像组装层，而不是状态和副作用的混合体。

## 一句话结论

当前项目不是屎山，而是一个瘦身后的中大型平台工程。真正的问题不是“代码多”本身，而是产品收敛之后，工程结构、兼容层和主链路大文件还需要同步收敛。
