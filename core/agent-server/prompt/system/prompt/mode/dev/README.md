# 开发模式

本目录下**只维护两个文件**，规则与用法都在这里说清。

## 文件与用途

| 文件 | 用途 | 何时、怎么用 |
|------|------|--------------|
| **config.json** | 模式配置：name、description、tool_names、引用的 md 文件名 | 代码 init 时 `loadModeProvider("dev")` 读取；`ToolNames()` 决定本模式可用工具；详细 SOP 已下沉到 `core/agent-server/skills`。 |
| **system_prompt.md** | **开发模式的系统级合同**（角色、skill 路由、边界、执行闭环） | 拼入发给模型的 **system**：在「身份 + 环境块」之后。这里只保留模式级约束，详细规则通过 Skills 目录 + `read_skill` 承接，不确定时再 `search_skills`；长文档按 skill 的 `required_docs` 读取。 |
| **first_assistant.md** | 会话开始时的**首条 assistant 消息**（可留空） | 留空则不注入首条 assistant，规则已在 system_prompt 中；若填写内容则在 system 之后、历史消息之前插入一条 assistant（如「会话开幕词」）。 |

## 发给模型的 system 实际顺序

1. Agent 身份（或默认「智能工作台的助手」）
2. **环境块**（用户、目录、子节点、可读文件、可读目录列表等）
3. **system_prompt.md 全文**（开发模式身份 + 路由 + 系统级约束）
4. Skills 工作规则（运行时追加）：先按 Skills 目录直接 `read_skill`，不确定时再 `search_skills`，然后按 skill 的 `required_docs` 读取文档

## 首条消息顺序

1. 一条 **system**（上面 1～3）
2. 若 **first_assistant.md** 非空，则插入一条 **assistant**（其内容）；否则不插入
3. 之后才是历史消息（user / assistant / tool …）

## 维护方式

- **改模式级约束、任务路由、风格**：改 **system_prompt.md**。
- **改创建/修改/执行 SOP**：只改 `core/agent-server/skills/**/SKILL.md`；旧 `workspace/` SOP 目录已下线，不再作为运行时入口。
- **改 SDK 规则、标签、模式说明**：改 `sdk/` 子目录。
- **改会话开幕词**（可选）：若希望注入首条 assistant，在 **first_assistant.md** 填写；留空则不注入。
- 无需再维护单独的 operation prompt 文件。

代码入口：`prompt/mode_provider.go`（加载）、`workspace_chat_service.buildLLMMessages`（拼接模式提示词与 Skills 工作规则）。

---

## 可用文档的维护逻辑

**核心原则**：`system_prompt.md` 只保留路由与总约束；案例、SDK、workspace 细节全部以下游文档为准，保证「说的」和「跑的」一致。

### 示例项目（实例）位置

- **默认源路径**：`namespace/luobei/demos/code/api`（相对项目根目录）。
- 该目录是案例同步工具的输入源，用于从真实 demos 应用同步案例 PRD、摘要和 Go 代码。
- 如果当前仓库没有这个目录，说明本地只保留了已同步后的 `case_catalog` seed；此时不要运行同步工具，直接维护 `core/agent-server/prompt/system/prompt/case_catalog/` 下的文档，或先恢复 demos 源目录。

### 每个案例目录的 3 文件结构（解耦设计）

每个案例目录（如 `api/tables/meeting/`）建议只放 **3 类内容**，便于维护和工具同步：

| 类型 | 文件 | 用途 |
|------|------|------|
| **1. Go 代码** | `*.go`、`init_.go` | 业务实现；init_.go 由系统生成。 |
| **2. PRD** | `prd.md` | 该案例的完整 PRD（表单字段表、列表列、关系说明等）；工具可同步到 builtin case_catalog 供 read_doc 使用。 |
| **3. 摘要** | `summary.md` | **仅用于文档大纲**：一两句话说明「案例名 + 何时用」；在「可读的目录」/文档大纲里直接展示这条摘要，模型一眼可知该案例适合什么场景。 |

**解耦效果**：

- **更新项目时**：只改代码 + 改 `prd.md` + 改 `summary.md`，**无需**去改 `system_prompt.md` 或额外维护目录索引文件。
- **文档大纲**：维护工具会把案例摘要同步到各目录的 `readme.md`；运行时再根据树上的目录元数据动态生成「可读的目录」，与示例项目一致。
- **流程**：前端验证 → 改代码 / prd.md / summary.md（只在案例目录内）→ 运行工具 → 工具把案例目录、PRD、摘要同步到提示词与文档大纲；提示词与文档不再手改，实现解耦。

参考示例目录：`namespace/luobei/demos/code/api/tables/meeting/`（内含 go、prd.md、summary.md）。

### 文档工具说明（sync-case-catalog）

为降低人工同步成本，**专门提供了文档维护工具**，后续维护以本说明为准。

| 项目 | 说明 |
|------|------|
| **工具路径** | `scripts/sync-case-catalog/main.go`，在**项目根目录**执行。 |
| **命令** | `go run ./scripts/sync-case-catalog` |
| **输入** | 示例项目 `namespace/luobei/demos/code/api` 下各案例目录：**prd.md**、**summary.md** 及该目录下业务 **.go**（不含 init_.go）。源目录不存在时工具会直接失败且不改 seed 文件。 |
| **输出** | 见下方「执行后产出」。 |

**执行后产出**：

1. **system/prompt/case_catalog/**（`core/agent-server/prompt/system/prompt/case_catalog/`）
   - **目录与示例项目对齐**：如 `form/excelorcsv/`、`table/ticket/`、`tables/meeting/` 等，与 `api/` 下结构一致。
   - **文档名统一为 prd.md**：每个案例目录下生成 **prd.md**（内容 = 该案例 prd.md 全文 + 该目录下业务 .go 代码）。
   - read_doc 时：传目录路径如 `/system/prompt/case_catalog/form/excelorcsv`，后端按 `core/agent-server/prompt/system/prompt/case_catalog/form/excelorcsv/prd.md` 初始化到树上，并按该逻辑路径读取。

2. **目录元数据**
   - 各案例目录的 **name**、**description** 来自同目录 `readme.md`；启动时会同步到树上。
   - 系统消息里的「可读的目录」运行时根据树上的目录元数据动态生成，模型据此调用 `read_doc`。

3. 案例目录由 `scripts/sync-case-catalog` 同步到 `case_catalog/`，创建类入口由 `sop.create-project` 维护推荐案例。

**推荐流程**：

1. 在前端验证 demos 项目（跑真实功能、看列表/表单/路由等）。
2. 发现问题 → 在示例项目**对应案例目录**内改 **代码**、**prd.md**、**summary.md**（摘要可含「本案例有 X 个模块，分别是 xxx」等说明）。
3. 确认 `namespace/luobei/demos/code/api` 存在后，**运行文档工具**：`go run ./scripts/sync-case-catalog`。
4. `case_catalog/` 文档立即反映最新示例，`sop.create-project` 只维护稳定推荐路径。

这样形成闭环：**示例项目是唯一事实来源**，每个案例目录 **3 类（go + prd.md + summary.md）** → 只改这 3 类 → 工具同步到 `/system/prompt/case_catalog` 与各目录 `readme`/`prd` → 运行时动态生成目录索引 → **解耦**，无需手改提示词或额外索引文件。
