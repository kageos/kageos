---
id: system.openapi.message
name: message-openapi
description: 通过 /system/openapi/message 调用 AgentOS message-server 发送用户或部门消息通知。适用于发送站内信、邮件、企微/钉钉等统一消息入口。
triggers:
  - 发送消息
  - 消息通知
  - 通知用户
  - 通知部门
  - message-server
  - 邮件
  - 站内信
  - webhook 通知
modes:
  - execute
  - dev
  - agent
required_docs:
  - /system/prompt/platform-cross-cutting-capabilities
  - /system/prompt/sdk/agent-app-sdk-readme
  - /system/prompt/sdk/platform-api-reference
capabilities:
  - /system/openapi/message/send.form
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - read_go_file_lines
  - search_tools
  - run_form_submit
  - record_workspace_event
completion:
  - 已确认这是平台消息通知需求，不是在业务代码里自建消息通道
  - 已确认接收用户或接收部门
  - 已确认消息标题、内容和 content_type
  - 已确认发送动作会产生副作用，并获得用户明确授权
  - 已通过 /system/openapi/message/send.form 执行或说明当前不可执行原因
---

# Message OpenAPI SOP

## 使用条件

用户要发送消息、邮件、站内信、用户通知或部门通知时，优先使用本 skill。普通业务代码里需要发消息时，使用 SDK `ctx.SendMessage(...)`；工作台直接执行平台消息发送时，使用 `/system/openapi/message/send.form`。

## 关键规则

1. 消息发送由 message-server 承接，不要调用旧服务入口。
2. `/system/openapi/message/send.form` 只暴露业务消息字段：接收人、接收部门、标题、内容、内容类型。
3. 发送人、来源路径、当前用户、token、trace 等元数据从 SDK `ctx` 和平台请求上下文自动传递，不让模型或用户手填。
4. 发送前必须明确接收对象和正文内容；群发、跨部门发送、外部渠道发送必须先向用户确认。
5. `content_type` 默认 `markdown`；只有用户明确要求精确 HTML 邮件模板时才用 `html`。
6. 不要把通知逻辑写进业务表字段或自建消息表；业务应用需要通知时调用 SDK `ctx.SendMessage(...)`。

## 标准流程

1. 读取 `required_docs`。旧 `/system/prompt/workspace/*` SOP 已下线，不再读取。
2. 用 `search_tools` 搜索 `message send /system/openapi/message`。
3. 确认函数 schema，目标函数应为 `/system/openapi/message/send.form`。
4. 整理参数：
   - `to_users`：用户列表，允许为空但不能和 `to_departments` 同时为空。
   - `to_departments`：部门列表，允许为空但不能和 `to_users` 同时为空。
   - `title`：可选标题。
   - `content`：必填正文。
   - `content_type`：`markdown`、`html` 或 `text`，默认 `markdown`。
5. 得到用户明确授权后调用 `run_form_submit`。
6. 返回发送结果中的 `from`、`full_code_path`、接收对象和 summary，便于审计。

## 当前函数

- `/system/openapi/message/send.form`：通过 message-server 发送用户/部门消息通知。
