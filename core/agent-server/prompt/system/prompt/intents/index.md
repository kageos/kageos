# 工作台意图索引

本文档是工作台意图路由入口。模型处理非闲聊任务时，先根据用户需求和下方路由表判断一级意图；意图不确定时先用简短问题澄清。

## 一级意图

| 意图 ID | 场景 | 必读文档 |
| --- | --- | --- |
| `app.plan` | 新建系统、目录、Form/Table/Chart 的 PRD 设计和确认 | `/system/prompt/intents/app-plan` |
| `app.create` | 已确认 PRD 后创建目录、写代码、build | `/system/prompt/intents/app-create` |
| `app.modify` | 修改已有应用、字段、回调、图表、业务逻辑 | `/system/prompt/intents/app-modify` |
| `app.operate_test` | 操作和测试已有函数 | `/system/prompt/intents/app-operate-test` |
| `app.build_fix` | 构建、启动、schema、widget 校验排错 | `/system/prompt/intents/app-build-fix` |
| `temp.task` | 文件、媒体、数据的一次性杂活 | `/system/prompt/intents/temp-task` |
| `schedule.task` | 定时任务、周期任务、定时智能体 | `/system/prompt/intents/schedule-task` |
| `platform.openapi` | 平台接口、消息、权限、审计、应用市场目录复用 | `/system/prompt/intents/platform-openapi` |
| `app.explain_review` | 解释项目、review、只读分析 | `/system/prompt/intents/app-explain-review` |

## 通用闭环

```text
根据路由表识别意图
-> 调用 change_role 获取当前身份文档包
-> 执行当前意图
-> 输出结果
-> 给出下一步意图建议
```

切换意图时，先把目录、路由、文件、build 状态、已验证路径、已知问题和下一步建议压成极简摘要，再调用 `change_role` 进入新身份。模型只在上下文不足时按明确路径 `read_doc`，不继续依赖旧 PRD、旧错误和长工具输出。
