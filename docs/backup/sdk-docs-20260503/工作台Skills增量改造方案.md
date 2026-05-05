# 工作台 Skills 增量改造方案

本文说明如何把当前工作台从“prompt + docs + tools”增量升级为“prompt + skills + docs + tools/functions”，同时保留旧实现，支持灰度、无缝切换和快速回滚。

核心原则：

> 只新增能力，不删除旧链路；先双轨运行，再切默认；所有旧 prompt/docs/tools 保留，出问题能一键切回。

当前已落地：

- 本地 `core/agent-server/skills` 包和 4 个 SOP skills；
- `search_skills` / `read_skill` 两个只读工具；
- `AGENT_WORKSPACE_SKILLS_MODE=off|shadow` 回滚/灰度开关；默认不配置，直接走 skills 主路径；
- `qa` 问答模式；
- 工作台工具执行入口的 mode allowlist 校验。
- 当前用户轮次成功读取的 skill 会参与 `allowed_tools` 二次校验。
- 当前默认已切到 skills 主路径：未显式设置环境变量时走 skills；只有需要回滚/灰度观测时才设置 `AGENT_WORKSPACE_SKILLS_MODE=off|shadow`。
- `on` 模式下，skills 只做 prompt/目录/推荐，不做后端硬拦截，避免普通搜索和临时问答被误伤。
- `/system/openapi` 第一阶段脚手架、`openapi.platform` skill、平台 OpenAPI SOP 和提示词路由。
- `/system/tools` 已有 `tools.official` skill，用于官方工具工作空间和一次性文件/媒体/数据处理任务。
- SDK 侧新增 `ctx.APICall(...)`，让 `/system/openapi` 函数复用 Web API 调用链路并自动透传 token、trace、用户、部门和 client_source。
- app-server 启动时会初始化 `system/official`、`system/tools`、`system/openapi`、`system/prompt` 四个系统工作空间。
- `/system/openapi` 已落地 Hub 函数：`/system/openapi/hub/search.form`、`/system/openapi/hub/detail.form`、`/system/openapi/hub/publish.form`、`/system/openapi/hub/push.form`、`/system/openapi/hub/push_info.form`、`/system/openapi/hub/copy.form`。
- `/system/openapi` 已落地消息发送函数：`/system/openapi/message/send.form`，底层走 message-server 的 HTTP API。
- `/system/openapi` 已落地定时任务函数：`/system/openapi/scheduled_task/create.form`、`/system/openapi/scheduled_task/list.form`、`/system/openapi/scheduled_task/cancel.form`、`/system/openapi/scheduled_task/executions.form`。
- `/system/openapi` 已落地操作日志、目录变更记录和权限申请/查询/审批函数。

## 1. 改造目标

当前工作台主要依赖：

```text
mode system prompt
  + workspace env
  + /system/prompt 文档
  + mode config 中的一批 tools
```

这套方式已经能工作，但随着 SOP、SDK 文档、案例、工具越来越多，会出现：

- system prompt 越来越长；
- tools 常驻列表越来越大；
- 不同任务需要不同 SOP，但现在区分不够明确；
- 模型可能没读对文档就开始写代码或执行；
- 后续官方工具、OpenAPI、案例、用户函数都需要更好的组织方式。

改造后目标：

```text
Base Prompt：只保留最小规则、模式、意图识别和安全边界
Skills：按任务场景承接 SOP、文档索引、示例索引、可用能力和完成标准
Docs：继续保留 SDK 长文档、案例、设计说明
Tools：只保留核心工作台操作工具，低频能力逐步外置到 system 工作空间函数
```

## 2. 不做什么

第一阶段不做大重构：

- 不删除现有 `/system/prompt` 文档。
- 不删除现有 `mode/*/system_prompt.md`。
- 不删除现有 tools。
- 不立刻把 skills 做成数据库或服务树工作空间。
- 不立刻合并 `run_form_submit`、`run_table_search`、`run_chart_query`。
- 不立刻迁移 Python/PDF/视频/图片等官方工具。
- 不立刻实现用户函数自动生成 skill。

第一阶段只做：

- 新增本地 skills 目录；
- 新增 skill 读取和搜索能力；
- 修改工作台 prompt 装配逻辑，支持开关切换；
- 先迁移几个 SOP skill；
- 旧逻辑完整保留。

## 3. 新目录设计

新增本地目录：

```text
core/agent-server/skills/
  sop/
    create-project/
      SKILL.md
    modify-project/
      SKILL.md
    explain-project/
      SKILL.md
    execute-function/
      SKILL.md
  tools/
    pdf/
      SKILL.md
    python/
      SKILL.md
  openapi/
    hub/
      SKILL.md
```

第一阶段只需要落地 `sop/*` 四个：

```text
sop/create-project      创建新应用、新模块、新 Form/Table/Chart
sop/modify-project      修改已有应用
sop/explain-project     解释项目、分析代码、回答问题
sop/execute-function    调用已有函数、查数据、提交表单、查图表
```

后续再逐步增加：

```text
tools/pdf
tools/python
tools/video
tools/image
openapi/hub
openapi/message
openapi/changelog
```

## 4. Skill 文件格式

采用兼容业界的 `SKILL.md`，但扩展 AgentOS 需要的字段。

示例：

```markdown
---
id: sop.create-project
name: create-project
description: 创建新应用、新模块、新 Form/Table/Chart 时使用。用户提出“帮我做一个系统/应用/管理后台/工具”时优先使用。
triggers:
  - 创建系统
  - 新建应用
  - 生成管理后台
  - 做一个工具
modes:
  - execute
required_docs:
  - /system/prompt/platform-overview
  - /system/prompt/sdk/agent-app-sdk-readme
recommended_demos:
  - /system/prompt/case_catalog/table/ticket
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - write_go_file
  - search_replace_file
  - build_workspace
  - run_form_submit
  - run_table_search
  - run_chart_query
completion:
  - 已输出 PRD 并获得用户确认后才写代码
  - build_workspace 必须通过
  - 至少验证一个核心函数
---

# 创建项目 SOP

...
```

字段说明：

| 字段 | 说明 |
|---|---|
| `id` | 全局唯一 ID，建议 `sop.create-project`、`tools.pdf` |
| `name` | 短名称，兼容通用 skills 格式 |
| `description` | 用于检索和触发判断，必须写清何时使用 |
| `triggers` | 中文/英文触发词，供搜索和 intent router 使用 |
| `modes` | 允许在哪些模式使用，如 `qa`、`execute` |
| `required_docs` | skill 执行前必须读取的文档 |
| `recommended_demos` | 推荐读取的案例或示例 |
| `capabilities` | 后续可引用 system 工作空间函数 |
| `allowed_tools` | 本 skill 建议/允许使用的工具 |
| `completion` | 完成标准，模型最终自检用 |

## 5. 新增包设计

新增 Go package：

```text
core/agent-server/skills/
  embed.go
  model.go
  parser.go
  registry.go
  search.go
```

职责：

| 文件 | 职责 |
|---|---|
| `embed.go` | `go:embed skills`，嵌入本地 skills |
| `model.go` | `SkillMeta`、`Skill`、`SearchSkillReq` 等结构 |
| `parser.go` | 解析 `SKILL.md` frontmatter 和正文 |
| `registry.go` | 启动时建立 skill catalog，并按 id/name/path 读取 skill |
| `search.go` | 按 keyword、mode、trigger 搜索 skill |

不要直接复用 `prompt` 包，避免 prompt 和 skill 概念耦合过深。`prompt` 可以调用 `skills` 包，但 `skills` 包不依赖 `prompt`。

## 6. 新增工具

新增两个内置 tools：

```text
search_skills
read_skill
```

### 6.1 search_skills

输入：

```json
{
  "keyword": "创建工单系统",
  "mode": "execute",
  "limit": 5
}
```

输出只返回短 metadata：

```text
匹配到 2 个 skill：
1. sop.create-project
   name: create-project
   description: 创建新应用、新模块、新 Form/Table/Chart 时使用
   modes: execute
   required_docs: /system/prompt/platform-overview, /system/prompt/sdk/agent-app-sdk-readme

2. sop.modify-project
   ...
```

不要返回完整正文，避免 token 膨胀。

### 6.2 read_skill

输入：

```json
{
  "id": "sop.create-project"
}
```

输出：

- skill metadata；
- `SKILL.md` 正文；
- required docs 列表；
- recommended demos 列表；
- allowed tools；
- completion 标准。

是否自动读取 `required_docs`：第一阶段不自动读取，只提示模型必须调用 `read_doc`。这样更可控，也更容易观察模型行为。

## 7. Prompt 装配改造

当前链路大概是：

```text
WorkspaceChatService.buildLLMMessages
  -> BuildWorkspaceEnvData
  -> modeProvider.SystemPrompt
  -> modeProvider.OperationPrompt
  -> ListTools(mode tool_names)
```

改造后做双轨：

```text
if skills enabled:
  使用新版短 prompt + skill 指令
else:
  使用旧 mode prompt
```

建议新增配置：

```go
type SkillsMode string

const (
    SkillsModeOff    SkillsMode = "off"    // 完全旧逻辑
    SkillsModeShadow SkillsMode = "shadow" // 加载 skills catalog，但 prompt 仍旧逻辑，可观测
    SkillsModeOn     SkillsMode = "on"     // 新 prompt + skills
)
```

当前不要求常规部署配置该环境变量。环境变量只用于回滚或灰度观测：

```text
AGENT_WORKSPACE_SKILLS_MODE=off|shadow
```

默认行为：

```text
未设置 AGENT_WORKSPACE_SKILLS_MODE => skills 主路径
```

需要灰度观测但不强制拦截时：

```text
shadow
```

需要快速回滚旧链路时：

```text
off
```

当前已经进入默认 skills 阶段：常规部署不要显式配置 `AGENT_WORKSPACE_SKILLS_MODE`；保留 `off` 作为一键回滚，`shadow` 作为灰度观测。

## 8. Base Prompt 新旧切换

### 8.1 旧模式 off

完全保持现状：

- 继续使用 `mode/*/system_prompt.md`；
- 继续注入现有 workspace env；
- 继续使用 mode config 的 tool_names；
- 不要求模型读取 skill。

### 8.2 shadow 模式

shadow 模式不改变模型行为，只做观测：

- system prompt 仍用旧版本；
- 暴露 `search_skills/read_skill`，便于手动或模型自发读取；
- 不强制模型使用 skill。

目的：

- 验证 skill catalog 是否能正常解析；
- 观察 intent 到 skill 的匹配是否准确；
- 不影响线上行为。

### 8.3 on 模式

on 模式第一阶段不删除旧 mode prompt，而是在旧 prompt 后追加 skills 工作规则：

```text
你是 AI-Agent-OS 工作台助手。

当前模式：{{MODE}}
当前环境：{{WORKSPACE_ENV}}

工作规则：
1. 先判断用户意图。
2. 复杂任务优先根据 Skills 目录直接 read_skill；不确定该读哪个 skill 时，再 search_skills 兜底。
3. 建议读取对应 skill 后再写代码、构建、发布或执行副作用操作，但不做后端硬拦截。
4. skill 中 required_docs 建议 read_doc 后再继续。
5. 执行时优先使用当前模式和 skill 建议的工具；普通搜索/问答可直接使用合适工具。
6. 问答模式不得写文件、构建、删除、发布或调用有副作用函数。
```

后续 skills 稳定后，再把旧 mode prompt 逐步缩短为新版短 prompt。第一阶段先保证无缝切换和可回滚。

## 9. 模式权限硬约束

问答/执行模式不能只靠 prompt，需要 ToolRegistry 后端校验。

建议新增：

```text
mode -> allowed tool names
```

在 `ToolRegistry.CallTool` 或 WorkspaceChatService 调用工具前校验：

```text
if tool not allowed by session mode:
  return error
```

当前策略：

- 工作台对话执行 tool_call 前，根据当前 mode 的 `tool_names` 生成 allowlist；
- `shadow/on` 时自动追加 `search_skills/read_skill`；
- 空 allowlist 保留旧的 allow-all 语义，避免影响历史兜底路径；
- `on` 模式下，skills 只做推荐，不做后端硬拦截；
- `on` 模式下，如果当前用户轮次已经成功 `read_skill`，模型按这些 active skills 的 `allowed_tools` 做自我约束；
- active skills 只统计最新一条用户消息之后成功读取的 skills，不跨用户任务无限累积；
- `search_skills/read_skill` 是启动阶段工具：优先根据 Skills 目录直接 `read_skill`，不确定时才 `search_skills`。
- 未读取 active skill 前，`read_doc/read_dir/read_go_file/search_tools/web_search` 等工具不会被 skills 机制拦截。
- 读取 active skill 后，`required_docs` 和 `allowed_tools` 作为执行建议，不作为工具放行条件。

内置模式策略：

### qa 模式

允许：

```text
search_skills
read_skill
read_doc
read_dir
read_go_file
read_go_file_lines
search_tools
web_search
fetch_url_content
```

禁止：

```text
write_go_file
search_replace_file
delete_file
build_workspace
run_form_submit
run_table_create
run_table_update
run_table_delete
publish_to_hub
push_to_hub
copy_directory
record_workspace_event
```

### execute 模式

允许旧执行模式已有工具，再新增：

```text
search_skills
read_skill
```

后续可以再按 skill allowed_tools 进一步收紧。

## 10. 现有 SOP 迁移方式

不要一次性删旧 SOP。采用复制迁移：

```text
旧：
core/agent-server/prompt/system/prompt/workspace/create-project/01-create-project.md

新：
core/agent-server/skills/sop/create-project/SKILL.md
```

迁移规则：

1. 从旧 SOP 提炼核心流程到 `SKILL.md`。
2. 长篇 SDK 说明不复制，改用 `required_docs` 引用。
3. 完整案例不复制，改用 `recommended_demos` 引用。
4. 旧文档继续保留，read_doc 仍可读取。
5. 新 skill 先只覆盖最核心路径，不追求一次迁完整。

第一批迁移：

| 旧文档 | 新 skill |
|---|---|
| `/system/prompt/workspace/create-project` | `sop.create-project` |
| `/system/prompt/workspace/modify-project` | `sop.modify-project` |
| `/system/prompt/workspace/explain-project` | `sop.explain-project` |
| `/system/prompt/workspace/execute` | `sop.execute-function` |

## 11. Tools 收敛策略

这次改造先不删 tools。

第一阶段只新增：

```text
search_skills
read_skill
```

第二阶段再开始减少默认暴露 tools：

- mode config 中不再给所有场景暴露全部工具；
- 低频工具迁到 `/system/tools` 工作空间；
- 平台 API 类能力迁到 `/system/openapi` 工作空间；
- 用 `search_tools` 或后续 `search_capabilities` 查找；
- 用现有 `run_form_submit` 等调用。

第三阶段再考虑：

```text
run_form_submit
run_table_search
run_chart_query
  -> run_capability
```

不要在第一阶段合并执行工具，避免同时动 prompt、skills、tools、runtime 调用链。

## 12. system 工作空间规划

skills 第一阶段先放本地目录，不放工作空间。

但后续可以规划几个 system 工作空间：

```text
/system/demos
/system/tools
/system/openapi
```

### 12.1 /system/demos

存高质量示例应用：

- 单表 CRUD；
- 多表关联；
- Form 文件处理；
- Chart 多系列；
- Python 工具；
- OnSelectFuzzy；
- 定时任务。

Skill 只引用 demo，不复制 demo 内容。

### 12.2 /system/tools

存官方外挂工具：

- PDF；
- Python；
- 图片；
- 视频；
- Excel；
- 文档转换；
- 数据分析。

这些是可运行 Form/Table/Chart，不是内置 tool。

### 12.3 /system/openapi

存平台能力封装：

- Hub 搜索；
- Hub 发布；
- 推送到 Hub；
- 读取资源变更日志；
- 发送消息通知；
- 创建/查询/取消定时任务；
- 查询权限；
- 查询操作日志。

当前已落地入口、规则和第一批可执行接口：

- `namespace/system/openapi/README.md`
- `namespace/system/openapi/code/api/README.md`
- `openapi.platform` skill
- `read_doc("/system/prompt/workspace/platform-openapi")`
- dev/agent/execute/misc/overview 提示词中说明 `/system/openapi` 与 `/system/tools` 的分工

具体 Form/Table/Chart 接口函数按需逐步加入。SDK 侧统一使用 `ctx.APICall(...)`，权限、审计和幂等由平台 Web API 侧承接，不要为了占位伪造可执行接口。当前已落地 Hub 搜索/详情/发布/推送/复制、消息通知发送、定时任务创建/查询/取消/执行记录、操作日志查询、目录变更记录查询、权限申请/查询/审批。资源元信息、定时智能体会话等继续按领域逐步补。

## 13. SDK 调平台 API 的后续设计

`/system/openapi` 函数如果用 SDK 写，就需要安全调用平台接口。

建议 SDK 侧保持一个薄入口：

```go
var resp HubSearchResp
err := ctx.APICall(http.MethodPost, "/hub/api/v1/directories/search", req, &resp)
```

原则：

- 默认使用当前请求用户 token。
- 传递 trace_id、request_user、department、client_source。
- app-server 继续做权限校验。
- system/openapi 函数不天然拥有超级权限。
- 如需系统权限，后续由平台 Web API 侧统一设计。
- 所有副作用操作进入平台侧操作日志。

这部分不进入第一阶段。

## 14. 回滚策略

必须支持一键回滚。

### 14.1 配置回滚

设置：

```text
AGENT_WORKSPACE_SKILLS_MODE=off
```

效果：

- 不走新版短 prompt；
- 不要求模型读取 skill；
- 旧 mode prompt 生效；
- 旧 docs/tools 全部可用。

### 14.2 工具回滚

即使新增 `search_skills/read_skill`，也不影响旧工具。

如果新工具有问题：

- 从 mode config 的 `tool_names` 移除；
- 或在 ToolRegistry 中通过配置禁用；
- 旧工具继续工作。

### 14.3 文档回滚

旧 `/system/prompt/workspace/*` 文档不删除。

如果某个 skill 写得不好：

- 修改 skill；
- 或禁用该 skill；
- 或切回旧 prompt；
- 不影响旧文档读取。

## 15. 灰度步骤

### 阶段 0：准备

- 新增 `core/agent-server/skills` 包；
- 新增 4 个 SOP SKILL.md；
- 新增解析测试；
- 早期曾默认 `AGENT_WORKSPACE_SKILLS_MODE=off`，旧工作台行为不变。
- 已完成。

### 阶段 1：shadow

设置：

```text
AGENT_WORKSPACE_SKILLS_MODE=shadow
```

行为：

- 旧 prompt 继续使用；
- 工作台 tools 列表追加 `search_skills/read_skill`；
- 不强制模型使用。

验证：

- skill catalog 正常加载；
- search 结果合理；
- read_skill 输出可读；
- 旧工作台行为不变。
- 已具备开关和工具能力，可直接进入 shadow 验证。

### 阶段 2：默认 skills

不设置 `AGENT_WORKSPACE_SKILLS_MODE`，直接使用默认 skills 主路径。

验证：

- 创建项目会先读 `sop.create-project`；
- 修改项目会先读 `sop.modify-project`；
- 问答模式不会写文件；
- build_workspace 行为不变；
- 失败时显式设置 `AGENT_WORKSPACE_SKILLS_MODE=off` 切回旧链路。
- `qa` 模式和 mode allowlist 已落地，可用于验证“问答无副作用”。
- skill `allowed_tools` 二次校验已落地，可验证“未读 skill 不能调用副作用工具；读了 create skill 后不能直接调用 Python 工具，读了 execute/tools skill 后可以调用执行类工具”。

### 阶段 3：保持默认 skills

当前已进入默认 skills：

- 未设置 `AGENT_WORKSPACE_SKILLS_MODE` 时默认启用 skills；
- 常规部署不显式配置 `AGENT_WORKSPACE_SKILLS_MODE`；
- 保留 `off` 回滚开关；
- 保留 `shadow` 灰度观测开关；
- 旧 prompt 继续保留一段时间。

### 阶段 4：tools 收敛

skills 稳定后再进行：

- 缩短默认 tool_names；
- 官方工具迁到 `/system/tools`；
- 平台 API 迁到 `/system/openapi`；
- 设计 `run_capability`。

## 16. 测试计划

### 16.1 单元测试

新增测试：

- skill frontmatter 解析；
- skill catalog 构建；
- search_skills keyword 匹配；
- read_skill 读取正文；
- mode 过滤；
- 缺失 skill 报错；
- malformed SKILL.md 报错但不影响其他 skill。

### 16.2 prompt 测试

- `go test ./core/agent-server/prompt`
- 新增 `go test ./core/agent-server/skills`
- `go test ./core/agent-server/service` 中覆盖两个新工具。

### 16.3 行为测试

人工或自动化测试：

- 问答模式：问“这个项目是什么”，确认没有写操作。
- 执行模式：创建一个简单 Form，确认先读 skill，再写代码，再 build。
- 修改模式：改一个字段，确认先读 modify skill。
- 回滚：切 `AGENT_WORKSPACE_SKILLS_MODE=off` 后旧流程可用。

## 17. 首批文件改动建议

第一批代码改动：

```text
core/agent-server/skills/embed.go
core/agent-server/skills/model.go
core/agent-server/skills/parser.go
core/agent-server/skills/registry.go
core/agent-server/skills/search.go

core/agent-server/skills/sop/create-project/SKILL.md
core/agent-server/skills/sop/modify-project/SKILL.md
core/agent-server/skills/sop/explain-project/SKILL.md
core/agent-server/skills/sop/execute-function/SKILL.md

core/agent-server/service/tool_search_skills.go
core/agent-server/service/tool_read_skill.go
core/agent-server/service/tool_skills_test.go
core/agent-server/service/workspace_skills_mode.go
core/agent-server/service/tool_mode_policy.go
core/agent-server/service/tool_mode_policy_test.go
core/agent-server/service/workspace_skill_policy.go
core/agent-server/service/workspace_skill_policy_test.go
core/agent-server/service/tool_defs_platform.go
core/agent-server/service/workspace_chat_service.go
core/agent-server/service/workspace_stream_loop_deps.go

core/agent-server/prompt/system/prompt/mode/qa/config.json
core/agent-server/prompt/system/prompt/mode/qa/system_prompt.md
core/agent-server/prompt/system/prompt/mode/qa/first_assistant.md
core/agent-server/prompt/mode_provider.go
core/agent-server/model/workspace_mode.go

工作台Skills增量改造方案.md
```

第二批再考虑：

```text
core/agent-server/prompt/system/prompt/mode/*/config.json
core/agent-server/prompt/system/prompt/mode/*/system_prompt.md
system/tools 和 system/openapi 工作空间
用户函数自动生成 skill
平台 Web API 的权限、审计和幂等规则
```

## 18. 最终形态

稳定后的工作台应该是：

```text
Base Prompt 很短
  ↓
识别用户意图
  ↓
search_skills
  ↓
read_skill
  ↓
按 skill 读取 docs/demos
  ↓
按 skill 使用 tools/functions
  ↓
build/run/verify
```

而不是：

```text
一次性塞入超长 prompt
  + 全部 SOP
  + 全部 tools
  + 全部案例说明
```

## 19. 结论

这次改造应该按“新增目录 + 双轨实现 + 开关切换 + 旧链路保留”的方式做。

第一阶段只建立 skills 基础设施和 SOP 迁移，不动核心运行链路，不删除旧 prompt，不缩减旧 tools。等模型在新流程下稳定做到“先识别意图 -> 读 skill -> 按 SOP 执行”后，再逐步收敛内置 tools，把低频能力迁到 `/system/tools` 和 `/system/openapi`。

这样改造风险低、可灰度、可回滚，也能逐步把工作台从“长 prompt 驱动”升级成“按场景加载 skill 驱动”。
