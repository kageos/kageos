# 工作台单 Dev 模式与角色状态机架构设计

> 状态：执行口径
> 更新时间：2026-05-17
> 负责人窗口：事项 8 / codex/document-archive-cleanup

## 核心结论

工作台只保留一个 `dev` 模式，但 `dev` 不是单一大角色。每轮请求必须先通过 `change_role` 选择或沿用标准角色，再由后端根据角色返回文档包并执行工具门禁。

## 角色状态机

```text
router
  -> product_manager
  -> app_developer
  -> qa_engineer
  -> maintenance_engineer
  -> build_engineer
  -> platform_engineer
  -> data_operator
  -> reviewer
```

标准角色 ID：

- `router`
- `product_manager`
- `app_developer`
- `maintenance_engineer`
- `qa_engineer`
- `build_engineer`
- `data_operator`
- `platform_engineer`
- `reviewer`

## Dev System Prompt 职责

`dev/system_prompt.md` 只负责：

1. 要求每轮先调用 `change_role`。
2. 说明标准角色 ID 和路由规则。
3. 说明角色不匹配时必须切换。
4. 说明目标不明确时只问最少必要问题。
5. 说明阶段切换时要压缩上下文。

SDK 细节、案例、构建规则和专项操作规则都放在角色文档、SDK 主文档和案例目录中按需读取。

## `change_role`

`change_role` 是角色状态机入口：

- 输入：`current_role`、`target_role`、`task_summary`、`directory`、`reset_context`。
- 输出：`role_id`、`display_name`、`required_docs`、`loaded_docs`、`allowed_next_tools`、`next_roles`。
- 非法 `target_role` 直接报错，不做旧 ID 兼容。

## 工具门禁

工具权限由后端根据当前 `role_id` 执行：

- 产品经理禁止写代码、build、运行业务工具。
- 应用开发工程师禁止重新输出 PRD。
- 测试工程师禁止写代码和 build。
- 代码审查分析师只读。

## 交接策略

阶段交接使用结构化 handoff packet，目标会话只接收：

- artifact
- 任务摘要
- 目录和关键文件
- 路由清单
- 构建结果
- 测试结论
- 已知问题和下一步目标

历史消息继续保留给 UI 展示，但不作为下一角色的完整模型上下文。
