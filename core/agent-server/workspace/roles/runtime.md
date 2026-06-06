# 工作台角色运行时协议

## 目标

工作台主流程只负责路由和上下文组装，具体执行从角色出发。每个角色声明自己的工具、文档、进入条件、禁止条件、SOP、完成标准和生命周期 Hook，避免把所有规则堆进主提示词。

## 角色对象

每个角色由 `Spec` 定义：

- `Docs`：切换到角色时直接加载的稳定文档。
- `AllowedTools` / `ForbiddenTools`：工具边界。
- `Runtime.EntryConditions`：什么时候应该进入该角色。
- `Runtime.ForbiddenConditions`：什么时候不应该进入该角色。
- `Runtime.SOP`：该角色标准作业流程。
- `Runtime.DoneWhen`：该角色何时算完成。
- `Runtime.HandoffRequired`：标准交接字段，当前固定为 `execute_directory/task_context/key_information/references`。
- `Runtime.Hooks`：生命周期 Hook 声明。

## Hook 阶段

第一阶段 Hook 先声明、可展示、可测试；其中高价值链路逐步实现为可执行函数。

- `before_enter`：进入角色前补齐角色所需上下文，例如应用操作员获取当前应用函数 schema。
- `after_tool`：关键工具执行后整理产物或诊断，例如 build 失败生成错误诊断。
- `before_handoff`：交接下一角色前把产物整理成下一角色的执行视图，例如 PRD 转开发 Markdown 表格。

Hook 必须产出结构化数据，不能偷偷拼隐藏长提示词；必须声明读取内容和产出内容；必须受 `execute_directory` 限制。

## 已执行 Hook

- `product_manager.to_app_developer`：在用户确认 `agent_app_prd` 并交接给 `app_developer` 前执行，把结构化 PRD 渲染成 `PRD_EXECUTION_MARKDOWN`，同时在 handoff context 写入 `executed_hooks`，用于前端和日志观察本次交接到底产出了什么。

后续 Hook 应按同一方式接入：主流程只调用 HookRunner，具体角色逻辑放在对应 Hook 内；Hook 输出必须可测试、可展示，不依赖旧会话完整历史。

## 上下文边界

- 给模型：角色文档、运行契约、交接四块、结构化产物、必要执行视图。
- 给前端：角色交接卡、运行契约、产物卡、诊断卡。
- 给数据库：完整历史、artifact、handoff packet、hook 输出。

旧会话不应在阶段交接后继续透传给新模型；新模型只接收交接包和结构化产物。
