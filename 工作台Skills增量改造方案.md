# 工作台 Skills 主链路说明

当前工作台已经切到 **Skills 主链路**。旧的 `/system/prompt/workspace/*` SOP 文档链路已下线，不再作为模型入口、seed 文档或 `read_doc` 可读路径。

## 当前执行链路

```text
用户需求
  ↓
意图识别
  ↓
按 Skills 目录直接 read_skill("<skill id>")
  ↓
按 skill 的 required_docs / recommended_demos 读取 SDK、平台总览或案例文档
  ↓
调用核心 tools、/system/tools 或 /system/openapi 工作空间函数
  ↓
构建、运行、验证、总结
```

## 已下线内容

- 不再支持 `AGENT_WORKSPACE_SKILLS_MODE=off|shadow` 这类旧链路回滚开关。
- 不再 seed `core/agent-server/prompt/system/prompt/workspace/**`。
- `read_doc("/system/prompt/workspace/...")` 会返回错误，提示改用 `read_skill`。
- mode 不再支持 `operation_prompt_file` / `OperationPrompt` 额外操作提示词。
- `scripts/sync-case-catalog` 不再回写旧 create-project SOP。

## 当前有效入口

- 创建项目：`sop.create-project`
- 修改项目：`sop.modify-project`
- 执行已有函数：`sop.execute-function`
- 解释项目：`sop.explain-project`
- SDK 写法：`sdk.*`
- 官方工具工作空间：`system.tools` / `system.tools.*`
- 平台接口工作空间：`system.openapi` / `system.openapi.*`

## 文档来源

- SDK 文档：`/system/prompt/sdk/*`
- 案例文档：`/system/prompt/case_catalog/*`
- 平台总览：`/system/prompt/platform-overview`
- 平台能力边界：`/system/prompt/platform-capability-boundaries`
- 平台横切能力：`/system/prompt/platform-cross-cutting-capabilities`

## 维护原则

- 新增或修改 SOP，只改 `core/agent-server/skills/**/SKILL.md`。
- 长篇稳定规则放在 SDK / 平台顶层文档中，由 skill 显式引用。
- 不恢复旧 workspace SOP 目录；如果需要历史内容，从 git 历史查。
