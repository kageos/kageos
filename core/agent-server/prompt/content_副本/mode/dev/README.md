# 开发模式（dev）提示词说明

本目录下**只维护两个文件**，规则与用法都在这里说清。

## 文件与用途

| 文件 | 用途 | 何时、怎么用 |
|------|------|--------------|
| **config.json** | 模式配置：name、description、tool_names、引用的 md 文件名 | 代码 init 时 `loadModeProvider("dev")` 读取；`ToolNames()` 决定本模式可用工具；**不配置** operation_prompt_file（规则全部在 system_prompt 里） |
| **system_prompt.md** | **开发模式下的完整规则**（定位、风格、PRD、SOP、工具、禁止、示例） | 拼入发给模型的 **system**：在「身份 + 环境块」之后、**不再**追加其他操作提示词。**维护规则只改本文件**。 |
| **first_assistant.md** | 会话开始时的**首条 assistant 消息**（可留空） | 留空则不注入首条 assistant，规则已在 system_prompt 中；若填写内容则在 system 之后、历史消息之前插入一条 assistant（如「会话开幕词」）。 |

## 发给模型的 system 实际顺序

1. Agent 身份（或默认「智能工作台的助手」）
2. **环境块**（用户、目录、子节点、可读文件、可读目录列表等）
3. **system_prompt.md 全文**（开发模式身份 + 工具 + 完整规则与示例）
4. **不再**追加「工作台操作提示词」或 doc 下的长文（规则已全部在 system_prompt 中）

## 首条消息顺序

1. 一条 **system**（上面 1～3）
2. 若 **first_assistant.md** 非空，则插入一条 **assistant**（其内容）；否则不插入
3. 之后才是历史消息（user / assistant / tool …）

## 维护方式

- **改规则、PRD、SOP、示例、禁止**：只改 **system_prompt.md**。
- **改会话开幕词**（可选）：若希望注入首条 assistant，在 **first_assistant.md** 填写；留空则不注入。
- 无需再维护单独的「工作台操作提示词」文件。

代码入口：`prompt/mode_provider.go`（加载）、`workspace_chat_service.buildLLMMessages`（拼接；当 mode 未提供 OperationPrompt 时不再追加 doc/工作台操作提示词）。

---

## 可用文档的维护逻辑

**核心原则**：`system_prompt.md` 里的「可用文档」、案例分类等，**都以本仓库的示例项目为准**；文档是围着这个项目转的，保证「说的」和「跑的」一致。

### 示例项目（实例）位置

- **仓库内路径**：`namespace/luobei/demos/code/api`（相对项目根目录）
- 该目录即 demos 应用的 API 代码，也是单 Table / 单 Form / 多 Table / Table+Form / Table+Form+Chart 等案例的真实实现；`system_prompt.md` 中的案例说明、代码路径、注入的目录树都应对齐此处。

### 每个案例目录的 3 文件结构（解耦设计）

每个案例目录（如 `api/tables/meeting/`）建议只放 **3 类内容**，便于维护和工具同步：

| 类型 | 文件 | 用途 |
|------|------|------|
| **1. Go 代码** | `*.go`、`init_.go` | 业务实现；init_.go 由系统生成。 |
| **2. PRD** | `prd.md` | 该案例的完整 PRD（表单字段表、列表列、关系说明等）；工具可同步到 builtin case_catalog 供 read_doc 使用。 |
| **3. 摘要** | `summary.md` | **仅用于文档大纲**：一两句话说明「案例名 + 何时用」；在「可读的目录」/文档大纲里直接展示这条摘要，模型一眼可知该案例适合什么场景。 |

**解耦效果**：

- **更新项目时**：只改代码 + 改 `prd.md` + 改 `summary.md`，**无需**去改 `system_prompt.md` 或 `文档目录.json`。
- **文档大纲**：由维护工具根据各案例目录下的 **summary.md** 自动生成（或更新 `文档目录.json`）；模型看到的「可读的目录」来自摘要，与示例项目一致。
- **流程**：前端验证 → 改代码 / prd.md / summary.md（只在案例目录内）→ 运行工具 → 工具把案例目录、PRD、摘要同步到提示词与文档大纲；提示词与文档不再手改，实现解耦。

参考示例目录：`namespace/luobei/demos/code/api/tables/meeting/`（内含 go、prd.md、summary.md）。

### 文档工具说明（sync-case-catalog）

为降低人工同步成本，**专门提供了文档维护工具**，后续维护以本说明为准。

| 项目 | 说明 |
|------|------|
| **工具路径** | `scripts/sync-case-catalog/main.go`，在**项目根目录**执行。 |
| **命令** | `go run ./scripts/sync-case-catalog` |
| **输入** | 示例项目 `namespace/luobei/demos/code/api` 下各案例目录：**prd.md**、**summary.md** 及该目录下业务 **.go**（不含 init_.go）。 |
| **输出** | 见下方「执行后产出」。 |

**执行后产出**：

1. **builtin/doc/case_catalog/**（`core/agent-server/prompt/content/builtin/doc/case_catalog/`）
   - **目录与示例项目对齐**：如 `form/excelorcsv/`、`table/ticket/`、`tables/meeting/` 等，与 `api/` 下结构一致。
   - **文档名统一为 prd.md**：每个案例目录下生成 **prd.md**（内容 = 该案例 prd.md 全文 + 该目录下业务 .go 代码）。
   - read_doc 时：传目录路径如 `/builtin/doc/case_catalog/form/excelorcsv`，后端按 `content/builtin/doc/case_catalog/form/excelorcsv/prd.md` 读取。

2. **文档目录.json**（`core/agent-server/prompt/content/doc/文档目录.json`）
   - 各案例的 **name**、**full_code_path**（与目录对齐，如 `/builtin/doc/case_catalog/table/ticket`）、**when_to_use**（来自 summary.md 的「适合参考」）。
   - 供系统消息「可读的目录」注入，模型据此调用 read_doc。

3. **create-project/01-create-project.md** 末尾一段由工具生成，**请勿手改**：
   - **案例按类型归类**（`<!-- BEGIN CASE CATALOG -->` … `<!-- END CASE CATALOG -->`）：按 summary.md 的名称、类型、适合参考生成，只保留 read_doc 路径。
   **system_prompt.md** 中仅保留一句说明：案例分类见 read_doc("/builtin/doc/workspace/create-project") 文档末尾，由工具生成。

**推荐流程**：

1. 在前端验证 demos 项目（跑真实功能、看列表/表单/路由等）。
2. 发现问题 → 在示例项目**对应案例目录**内改 **代码**、**prd.md**、**summary.md**（摘要可含「本案例有 X 个模块，分别是 xxx」等说明）。
3. **运行文档工具**：`go run ./scripts/sync-case-catalog`。
4. 文档与 create-project/01-create-project.md 末尾案例目录段立即反映最新示例。

这样形成闭环：**示例项目是唯一事实来源**，每个案例目录 **3 类（go + prd.md + summary.md）** → 只改这 3 类 → 工具同步到 builtin、文档目录、system_prompt → **解耦**，无需手改提示词或文档目录。
