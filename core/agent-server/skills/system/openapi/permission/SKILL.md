---
id: system.openapi.permission
name: permission-openapi
description: 通过 /system/openapi/permission 查询工作空间或资源权限，提交权限申请，并审批通过或拒绝权限申请。
triggers:
  - 权限
  - 权限查询
  - 申请权限
  - 审批权限
  - 授权
  - 拒绝权限
  - workspace permission
  - resource permission
modes:
  - execute
  - dev
  - agent
required_docs:
  - /system/prompt/platform-cross-cutting-capabilities
  - /system/prompt/sdk/platform-api-reference
capabilities:
  - /system/openapi/permission/apply.form
  - /system/openapi/permission/workspace.form
  - /system/openapi/permission/resource.form
  - /system/openapi/permission/requests.form
  - /system/openapi/permission/approve.form
  - /system/openapi/permission/reject.form
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - read_go_file_lines
  - search_tools
  - run_form_submit
  - record_workspace_event
completion:
  - 已确认资源路径、主体、角色或申请 ID
  - 查询类操作已返回权限状态或申请列表
  - 申请、审批、拒绝等副作用操作已获得用户明确授权
  - 未绕过平台权限校验或硬编码身份
---

# Permission OpenAPI SOP

## 使用条件

用户要查询工作空间权限、查询资源权限、提交权限申请、审批通过或拒绝权限申请时，使用本 skill。业务应用不要自己实现权限表或审批流。

## 标准流程

1. 读取 `required_docs`。旧 `/system/prompt/workspace/*` SOP 已下线，不再读取。
2. 用 `search_tools` 搜索 `/system/openapi/permission`。
3. 查询权限时确认资源路径、主体类型和主体标识。
4. 提交权限申请时确认资源路径、角色 ID、主体和申请理由。
5. 审批通过或拒绝时确认申请 ID 和处理意见；这类操作必须得到用户明确授权。
6. 不要假设 `/system/openapi` 有超级权限，平台服务端仍按当前 token 校验。

## 当前函数

- `/system/openapi/permission/apply.form`：提交权限申请。
- `/system/openapi/permission/workspace.form`：查询工作空间权限。
- `/system/openapi/permission/resource.form`：查询资源权限。
- `/system/openapi/permission/requests.form`：查询权限申请列表。
- `/system/openapi/permission/approve.form`：审批通过权限申请。
- `/system/openapi/permission/reject.form`：审批拒绝权限申请。
