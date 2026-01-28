# 智能工作台 — 产品需求文档（PRD）

> 版本：v0.1 | 状态：草稿 | 基于：以目录为边界的 Cursor 式对话框、开发与编排为主、MCP+插件可搜

---

## 一、产品定位与目标

### 1.1 产品定位

**智能工作台** 是一个 **以目录为边界、以开发与编排为核心** 的 AI 对话工作台。用户在**指定目录**下输入需求，系统结合**知识库**与 **MCP 工具**（含插件、搜索、生成等）进行分析与编排，通过**多轮对话**和**工具调用**完成：从需求/文件/网络信息到**生成应用、package、函数**的完整链路。

- **类似 Cursor**：在对话框用自然语言描述需求，支持工具调用、多轮对话；**不是**无根的「全局助手」。
- **必须指定目录**：能明确、显式就不要隐式；**一个目录 = 一个领域 = 一个系统**；指定目录 = 可访问**该目录 + 子节点**。
- **点根 = 上帝视角**：支持，但**鼓励在具体目录开**，省大模型推理。
- **开发与编排为主**：聚焦从需求到生成系统、编排多工具；**执行业务**（如「帮我写一条记录」）弱化、后置。

### 1.2 目标用户

- **业务/产品**：有 Excel、 PDF、描述文档，希望快速生成 CRM、审批、表单列表等系统。
- **开发/配置**：在已有目录下增量生成函数、查文档、查网上资料再写 demo。
- **运维/集成**：查该目录下结构、读文件、触发构建。

### 1.3 核心目标

| 目标 | 说明 |
|------|------|
| **需求→生成** | 在对话框描述需求（可带文件），经多轮与工具调用，生成应用/package/函数并落盘。 |
| **可编排** | LLM 根据需求**选择工具**（插件、搜索、文档、add_functions 等）及顺序，而非固定流水线。 |
| **可扩展** | 新**插件**登记即可被 **MCP search** 搜到；常用可写 prompt，减少 search。 |
| **边界清晰** | 所有读、写、建、搜均受 **tree_id** 约束，知识库来自该目录及子节点。 |

---

## 二、用户故事与场景

### 2.1 开发与编排（核心）

| ID | 用户故事 | 验收要点 |
|----|----------|----------|
| U1 | 作为用户，我**在某个目录下打开工作台**，输入「我有一份 Excel 客户名单，帮我生成一个 CRM」并上传文件，系统能先解析 Excel，再根据解析结果和知识库生成 CRM 并落盘到该目录。 | 必须选目录；可上传文件；自动选 Excel 解析插件 + add_functions；落盘到当前 tree 下。 |
| U2 | 作为用户，我输入「用 2024 年某某 API 的最新用法写一个 demo」，系统能先搜网上信息，再结合知识库生成 demo 函数并落盘。 | 自动选 web_search + add_functions；生成在当 tree 下。 |
| U3 | 作为用户，我上传 PDF 说明 + 表截图，说「按这个生成表单和列表」，系统能按需调用 PDF 解析、表格识别（若有）、add_functions 等多工具，并落盘。 | 多插件 + add_functions 编排；均受 tree 约束。 |
| U4 | 作为用户，当需求不清晰时，系统能**多轮澄清**（如「要不要审批？」「表格要哪些筛选？」），在确认后再生成。 | 多轮对话；可中途修改需求再生成。 |
| U5 | 作为用户，当生成或工具调用失败时，系统能给出可理解提示，并支持我修改后**重试或换方案**。 | 错误可读；可续聊、重试。 |

### 2.2 目录与知识

| ID | 用户故事 | 验收要点 |
|----|----------|----------|
| U6 | 作为用户，我**点击某目录**进入工作台，系统只在该目录及子节点范围内读文档、搜函数、建 package、落盘；**不依赖隐式上下文**。 | tree_id 必填；工具作用域=该目录+子节点。 |
| U7 | 作为用户，我**点根目录**进入时，可在整个 app 内操作（上帝视角）；但我若只改某个 package，我**更希望在该 package 下开**，以节省大模型推理。 | 根=上帝视角支持；产品上可提示「在具体目录下更精准」。 |

### 2.3 插件与发现

| ID | 用户故事 | 验收要点 |
|----|----------|----------|
| U8 | 作为用户，当我需求涉及「解析 Excel」「查网上」等能力时，系统能**自动选择**对应插件或工具，无需我手动选。 | LLM 通过 list_tools / search_tools 或 prompt 选工具。 |
| U9 | 作为管理员，我**新增一个插件**（实现 Form + 在 Plugin 表登记）后，该插件能在工作台的 **MCP search** 里被 keyword/描述 搜到，从而被 LLM 选用。 | Plugin→MCP tool 映射；search_tools 覆盖 Plugin 的 name、description、keywords。 |

### 2.4 与现有入口的关系

| ID | 用户故事 | 验收要点 |
|----|----------|----------|
| U10 | 作为用户，我仍可在 **AIChatPanel**（包级、选某智能体）做**单一插件 + 生成**的简化流程；**工作台**与 AIChatPanel 并存，我按场景选。 | 工作台与 AIChat 都需 tree_id；工作台=多工具/编排，AIChat=包级/单插件+生成。 |

---

## 三、功能需求

### 3.1 入口与上下文

| 需求 ID | 描述 | 优先级 |
|---------|------|--------|
| F1 | **入口**：在 Workspace 的**服务树点击某目录**（含根）后，提供「打开工作台」入口，进入以该目录为边界的工作台。 | P0 |
| F2 | **tree_id 必传**：工作台 API、会话、工具执行 均**显式带 tree_id**；不提供「无目录」模式。 | P0 |
| F3 | **目录即边界**：指定目录 = 可访问**该目录 + 所有子节点**；点根 = 整个 app（上帝视角）。 | P0 |
| F4 | **鼓励具体**：在 UI 或引导上提示「在具体目录下打开更精准」，不禁止上帝视角。 | P2 |

### 3.2 工作台对话

| 需求 ID | 描述 | 优先级 |
|---------|------|--------|
| F5 | **多轮对话**：支持多轮 user/assistant；支持**澄清、确认、修改需求后继续**。 | P0 |
| F6 | **Tool Call 循环**：LLM 可返回 tool_calls → 后端执行 → 结果回填 → 再调 LLM，直至最终回复或需用户输入。 | P0 |
| F7 | **输入**：文本 + 可选文件（如 Excel、PDF）；文件参与插件或生成上下文。 | P0 |
| F8 | **会话 persist**：会话与消息可持久化，支持刷新后恢复、切目录可建新会话。 | P1 |
| F9 | **Tool 调用可视化**：可展示「调用了哪个工具、参数概览、成功/失败」，可折叠。 | P1 |

### 3.3 知识库

| 需求 ID | 描述 | 优先级 |
|---------|------|--------|
| F10 | **绑定知识库**：工作台所用 Agent 绑定 **DocsPaths**（或该 tree 下 docs）；加载该 tree 内文档作为需求理解与生成的上下文。 | P0 |
| F11 | **知识库来源**：以 **tree_id** 为范围，从 **service_tree 的 docs 节点** 及 Doc 表加载；与现有 DocsPaths、function_gen 知识加载方式兼容或统一。 | P0 |
| F12 | **docs 类工具**：在该 tree 下提供 **docs_search、docs_get、docs_list_outline**，供 LLM 按需查更多文档。 | P1 |

### 3.4 MCP 与工具

| 需求 ID | 描述 | 优先级 |
|---------|------|--------|
| F13 | **MCP 服务**：在 agent-server 实现 MCP 形态的工具层，提供 **list_tools、search_tools(keyword)、call_tool(name, arguments)**。 | P0 |
| F14 | **search_tools 语义**：search 针对**能力/插件**（如 excel、解析、搜索），不是搜平台内 table/form 函数；返回匹配的 tool 列表供 LLM 选择。 | P0 |
| F15 | **常用写 prompt**：将 Excel 解析、web_search、add_functions、docs_search 等常用工具的名称与用法写进 **System Prompt**，LLM 可不必每次 search 也能选。 | P0 |
| F16 | **插件 = MCP tool**：Plugin 表每条记录映射为一个 MCP tool（name、description、FormPath、keywords）；**call_tool(插件)** → `CallFormAPI(FormPath, {Content, InputFiles})`。 | P0 |
| F17 | **search 覆盖插件**：search_tools 在 **Plugin 的 name、description、keywords** 中检索，新插件登记即可被搜到。 | P0 |

### 3.5 工具清单（开发与编排优先）

| 工具 | 类型 | 说明 | 优先级 |
|------|------|------|--------|
| **add_functions** | 内置 | 将生成代码落盘到该 tree 下某 package；入参：tree_id、目标 package、code；走现有 add_functions。 | P0 |
| **插件（Excel 解析等）** | 插件 | 每个 Plugin 一个 tool；调用=CallFormAPI；search 可搜到。 | P0 |
| **web_search** | 内置 | 搜网上信息；入参：keyword；常用写 prompt。 | P0 |
| **docs_search** | 内置 | 在该 tree 下按 keyword 搜文档；需 SearchDocs 支持 full_code_path_prefix 或等同。 | P1 |
| **docs_get** | 内置 | 按 path 或 id 取文档内容；path 须在该 tree 下。 | P1 |
| **docs_list_outline** | 内置 | 该 tree 下文档大纲/列表。 | P2 |
| **create_package** | 内置 | 在该目录下建 package；需封装现有或新接口，并做 tree 校验。 | P1 |
| **query_service_tree** | 内置 | 查该节点及子树结构。 | P1 |
| **read_file** | 内置 | 读该树下文件；路径与权限受 tree 约束。 | P1 |
| **run_build** | 内置 | 触发该 app 的编译/重启；可选。 | P2 |
| **table_list / form_submit / table_create 等** | 内置 | 执行业务；**弱化、后置**，可 Phase 2 或按需。 | P2 |

### 3.6 与现有组件的关系

| 需求 ID | 描述 | 优先级 |
|---------|------|--------|
| F18 | **与 AIChatPanel 并存**：AIChatPanel 保留，**必须 tree_id**，包级、单插件+function_gen；工作台=目录级、多工具、编排。两者可共用会话/消息模型或 Agent 引擎，按 chat_type 或配置区分工具集。 | P0 |
| F19 | **与 Plugin、RunPlugin 兼容**：Plugin 表、FormPath、CallFormAPI 不变；工作台通过 MCP 的 call_tool 调插件，不再要求 Agent 绑死 PluginFunctionPath。plugin 型 Agent 仍可服务于 AIChatPanel 等单插件场景。 | P0 |
| F20 | **与 function_gen、add_functions 复用**：add_functions 作为 MCP tool 暴露，内部走现有流程；生成时的文档=知识库+可选 docs_search+插件 Result。 | P0 |

---

## 四、非功能需求

| ID | 类型 | 描述 |
|----|------|------|
| NF1 | 安全 | 所有工具在执行前做 **tree 内校验** 与**权限校验**；read_file、write_file、add_functions 等须限定在该 tree 对应 app 与路径下。 |
| NF2 | 性能 | 单轮若有多步 tool call，可**流式**返回 assistant 内容；tool 执行结果可异步回填后再续跑，超时与重试有上限。 |
| NF3 | 可观测 | 记录会话、消息、tool 调用（name、参数摘要、成功/失败、耗时），便于排查与迭代。 |
| NF4 | 兼容 | 与现有 Agent、Plugin、DocsPaths、service_tree、app-server 接口兼容；通过扩展参数（如 tree_id、full_code_path_prefix）或新接口实现，不破坏现有行为。 |

---

## 五、接口与数据（概要）

### 5.1 工作台对话 API

- **POST** `/agent/api/v1/workspace/chat/stream`（流式）
- **请求**：`tree_id`（必填）, `message`（含 content、可选 files）, `session_id`（可选）, `agent_id`（可选）  
- **响应**：流式或非流式；含 `content`、`tool_calls`（若有）、`session_id`；多步 tool 时可由服务端连续推 assistant+tool 结果，或客户端轮询/SSE。

### 5.2 MCP 形态（内部或对外）

- **list_tools**：返回全部 tool 定义（内置 + 插件映射）。  
- **search_tools(keyword)**：在 name、description、keywords 中检索，返回匹配列表。  
- **call_tool(name, arguments)**：执行；插件类转 `CallFormAPI(FormPath, req)`，内置类路由到对应实现；**隐式带 tree_id**（从会话/请求上下文传入）。

### 5.3 会话 / 消息

- **会话**：`session_id`, `tree_id`, `agent_id`, `status`, `created_at`, `updated_at`。  
- **消息**：`role`（user/assistant/tool）, `content`, `tool_calls`（ assistant ）, `tool_call_id`（ tool ）；支持多轮与 tool 结果回填。

---

## 六、前端（工作台 UI）

| 模块 | 描述 |
|------|------|
| **入口** | 服务树节点（含根）的上下文菜单或面板上的「打开工作台」；点击后打开工作台视图，并**传入当前 tree_id**。 |
| **布局** | 对话区（消息列表）+ 输入区（文本框 + 附件）；可选：右侧或折叠的「本会话 Tool 调用」列表。 |
| **消息** | 支持 user / assistant 展示；assistant 若含 tool_calls，可展示为可折叠的「调用 xxx(…) → 成功/失败」卡片。 |
| **输入** | 支持粘贴/上传文件；发送时带 `tree_id`、`session_id`（若有）。 |
| **标题/会话** | 可展示当前目录名；支持「新建会话」；切换目录时提示新建或切换会话，保证 **tree_id 与当前目录一致**。 |

---

## 七、依赖与约束

- **依赖**：  
  - 现有：service_tree、Doc、Plugin、FormPath、CallFormAPI、add_functions、Agent、DocsPaths、app-server 相关 API；  
  - 需扩展：SearchDocs 的 `full_code_path_prefix`（或 `tree_id`）、SearchFunctions 的 `tree_id`/`full_code_path_prefix`（若工作台需要搜平台内函数，可后置）；docs_list_outline 所需接口或拼装逻辑。  
- **约束**：  
  - 首版工具以**开发与编排**为主，**table_*、form_submit 等执行类**可后置；  
  - MCP 可先在 agent-server 内**自建 Tool Registry + LLM Tool Call 循环**，协议形态对齐 MCP 便于后续对外；  
  - 插件需在 **Plugin 表** 登记且 **FormPath** 有效，MCP 才能正确 call。

---

## 八、成功指标

| 指标 | 目标 |
|------|------|
| **核心路径** | 用户在某目录下通过「Excel + 一句需求」生成并落盘一个可用 table/form，**无需离开工作台、无需手选插件**。 |
| **多轮** | 支持至少 3 轮以上的澄清与再生成，且会话不丢失。 |
| **可扩展** | 新增 1 个 Plugin 并登记后，能在 search_tools 中被合理 keyword 搜到，并在工作台被 LLM 选用。 |
| **边界** | 任意工具（add_functions、docs_search、read_file 等）均**不**操作到当前 tree 之外的资源（根=整 app 除外）。 |

---

## 九、实施阶段建议

| 阶段 | 范围 | 目标 |
|------|------|------|
| **Phase 1** | 目录入口 + 工作台 UI + 对话 API + **MCP（list_tools、call_tool）** + **add_functions、1 个插件（如 Excel）、web_search** + 知识库加载（DocsPaths/tree） | 在**某目录下**完成「Excel+需求→解析→生成→落盘」；多轮与 tool 可视化可简化。 |
| **Phase 2** | **search_tools** + **docs_search、docs_get、create_package、query_service_tree、read_file**；Plugin 全量映射；常用写 prompt | 支持「先搜文档/先 search_tools 再选插件」；目录内 doc 与结构可查。 |
| **Phase 3** | **docs_list_outline、run_build**；**table_list、form_submit** 等执行类（可选）；Tool 调用流式与折叠展示；与 AIChatPanel 的引擎共用与配置化 | 开发与编排能力补全；执行类按需；体验与复用优化。 |

---

## 十、附录：术语与引用

- **tree_id**：service_tree 主键；表示「该目录」，其范围=该节点+所有子节点；根=整 app。  
- **DocsPaths**：Agent 上的逗号分隔 full_code_path，用于加载文档；工作台可结合 tree 映射为该 tree 的 docs。  
- **Plugin / FormPath**：Plugin 表、FormPath（full-code-path），CallFormAPI 调 Form 函数，返回 Result 文本。  
- **MCP**：Model Context Protocol 或本项目中实现的 list_tools、search_tools、call_tool 的协议形态。  
- **参考文档**：`工作台从执行为主到开发编排为主-思路分析.md`、`智能工作台与当前架构-综合分析与对齐.md`、`类似Cursor的全局AI对话框-需求与方案分析.md`。
