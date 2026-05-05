---
id: system.openapi
name: system-openapi
description: 操作 AgentOS 平台接口的总览和兜底 skill。具体 Hub、消息、定时任务、权限、审计任务应优先读取 system.openapi.hub、system.openapi.message、system.openapi.scheduled-task、system.openapi.permission、system.openapi.audit。
triggers:
  - 平台接口
  - openapi
  - system openapi
  - Hub 操作
  - 发送消息
  - 通知
  - 定时任务
  - 创建定时任务
  - 资源变更日志
  - 操作日志
  - 权限查询
modes:
  - execute
  - dev
  - agent
required_docs:
  - /system/prompt/platform-overview
  - /system/prompt/platform-cross-cutting-capabilities
  - /system/prompt/sdk/platform-api-reference
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - read_go_file_lines
  - search_tools
  - run_form_submit
  - run_table_search
  - run_chart_query
  - run_on_select_fuzzy
  - record_workspace_event
completion:
  - 已确认需求属于平台接口而不是业务应用逻辑
  - 已读取本 skill 或更具体的 system.openapi.* skill
  - 已优先直接读取更具体的 system.openapi.* skill；没有具体 skill 时才使用本总览兜底
  - 已通过 search_tools 找到 /system/openapi 或 system 用户下的已注册函数
  - 已确认函数 schema、字段、必填项、枚举和副作用
  - 副作用操作已得到用户明确授权
  - 未假设 system/openapi 拥有超级权限
---

# 平台 OpenAPI SOP

## 使用条件

用户要操作 AgentOS 平台自身能力但暂时无法判断具体领域时使用本 skill。能判断领域时优先读取更具体的 skill：

- Hub、发布、推送、复制：`system.openapi.hub`
- 消息、通知、邮件：`system.openapi.message`
- 创建/查询/取消定时任务：`system.openapi.scheduled-task`
- 权限查询、申请、审批：`system.openapi.permission`
- 操作日志、审计、变更记录：`system.openapi.audit`

如果用户只是要处理文件、图片、视频、PDF、Excel、图表生成等通用工具任务，应优先使用 `/system/tools` 相关能力，而不是本 skill。

## 流程

1. 先读取 `required_docs`。旧 `/system/prompt/workspace/*` SOP 已下线，不再读取。
2. 判断是否为平台接口任务：操作对象是平台资源、Hub、消息、权限、审计、日志等。
3. 如果能匹配具体领域，继续读取对应 `system.openapi.*` skill，并以具体 skill 为准。
4. 用 `search_tools` 搜 system 用户下已注册函数，关键词带上平台领域词，例如 `hub|发布|推送`、`消息|通知`、`变更日志|资源`。
5. 优先复用 `/system/openapi` 下的 Form/Table/Chart 函数；搜索结果里如果没有明确来源或能力摘要，继续确认 schema。
6. 执行前必须确认字段、必填项、枚举、文件字段、权限要求和副作用。
7. 有副作用的操作，例如发布、推送、发送消息、修改权限，必须得到用户明确授权后再调用。
8. 没有合适的 openapi 函数时，明确告诉用户当前未注册该平台接口，不要伪造调用结果。

当前已落地函数：

- `/system/openapi/hub/search.form`：搜索 Hub 资源。
- `/system/openapi/hub/detail.form`：读取 Hub 资源详情。
- `/system/openapi/hub/publish.form`：发布目录到 Hub。
- `/system/openapi/hub/push.form`：推送已发布目录到 Hub。
- `/system/openapi/hub/push_info.form`：查询 Hub 推送预填信息。
- `/system/openapi/hub/copy.form`：复制 Hub 或本地目录。
- `/system/openapi/message/send.form`：通过 message-server 发送用户/部门消息通知。
- `/system/openapi/scheduled_task/create.form`：创建定时任务。
- `/system/openapi/scheduled_task/list.form`：查询定时任务。
- `/system/openapi/scheduled_task/cancel.form`：取消定时任务。
- `/system/openapi/scheduled_task/executions.form`：查询定时任务执行记录。
- `/system/openapi/operate_log/table.form`：查询表格操作日志。
- `/system/openapi/operate_log/form.form`：查询表单操作日志。
- `/system/openapi/directory_history/app_version.form`：查询应用版本变更记录。
- `/system/openapi/directory_history/directory.form`：查询目录变更记录。
- `/system/openapi/permission/apply.form`：提交权限申请。
- `/system/openapi/permission/workspace.form`：查询工作空间权限。
- `/system/openapi/permission/resource.form`：查询资源权限。
- `/system/openapi/permission/requests.form`：查询权限申请列表。
- `/system/openapi/permission/approve.form`：审批通过权限申请。
- `/system/openapi/permission/reject.form`：审批拒绝权限申请。

## 权限约束

- `/system/openapi` 函数默认代表“可复用的平台接口封装”，不代表超级权限。
- 调用应使用当前请求用户身份，平台服务端继续做权限校验。
- SDK 函数必须使用 `ctx.APICall(...)` 调用平台 Web API，并透传当前请求 token 和 trace。
- 需要系统权限的接口由平台 Web API 侧统一控制；SDK 侧不要绕过。
- 不要在普通业务工作区里用裸 HTTP、裸 SQL 或硬编码 token 临时绕过平台接口。

## 完成标准

满足 frontmatter `completion` 中所有项目后，才认为任务完成。
