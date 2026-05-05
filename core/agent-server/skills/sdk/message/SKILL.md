---
id: sdk.message
name: sdk-message
description: 在业务 SDK 函数中发送消息通知时使用。约束使用 ctx.SendMessage 和 MessageSendEnvelope 元数据，不自建消息表、消息通道或伪造发送人。
triggers:
  - SendMessage
  - ctx.SendMessage
  - 发送消息
  - 消息通知
  - 邮件
  - 站内信
  - 通知用户
  - 通知部门
modes:
  - dev
  - modify
  - agent
required_docs:
  - /system/prompt/sdk/platform-api-reference
  - /system/prompt/platform-cross-cutting-capabilities
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - read_go_file_lines
  - write_go_file
  - search_replace_file
  - build_workspace
  - run_form_submit
completion:
  - 已确认通知是关键业务节点，不是每次普通 CRUD 都发送
  - 已使用 ctx.SendMessage
  - 已确认接收用户或接收部门字段来源
  - 已确认 content_type，默认 markdown
  - 已 build_workspace 并验证消息触发路径或说明无法验证原因
---

# SDK 消息发送

## 使用条件

业务函数需要给用户或部门发送提醒、通知、邮件、站内信时使用本 skill。

## 规则

1. 业务代码使用 `ctx.SendMessage(&app.SendMessageOpts{...})`。
2. 不要自建消息表、消息通道、SMTP/Webhook 客户端。
3. 发送人、请求用户、目录、trace、client_source 等元数据从 `ctx` 生成，不让用户正文决定。
4. `ContentType` 默认 `markdown`；只有明确需要 HTML 模板时用 `html`。
5. 避免在每次增删改都发消息，只在状态流转、到期提醒、异常告警等关键节点发送。

## 与 OpenAPI 的区别

- 业务函数内部发送：用 `ctx.SendMessage`。
- 工作台直接操作平台发消息：读 `system.openapi.message`，调用 `/system/openapi/message/send.form`。
