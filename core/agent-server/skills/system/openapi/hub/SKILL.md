---
id: system.openapi.hub
name: hub-openapi
description: 通过 /system/openapi/hub 操作 AgentOS Hub，包括搜索资源、读取详情、发布目录、推送更新、复制 Hub 或本地目录。
triggers:
  - Hub
  - Hub 搜索
  - 搜索 Hub
  - 发布到 Hub
  - 推送 Hub
  - 复制 Hub
  - 资源复用
  - 应用中心
modes:
  - execute
  - dev
  - agent
required_docs:
  - /system/prompt/platform-overview
  - /system/prompt/sdk/platform-api-reference
capabilities:
  - /system/openapi/hub/search.form
  - /system/openapi/hub/detail.form
  - /system/openapi/hub/publish.form
  - /system/openapi/hub/push.form
  - /system/openapi/hub/push_info.form
  - /system/openapi/hub/copy.form
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - read_go_file_lines
  - search_tools
  - search_hub_directory
  - copy_directory
  - publish_to_hub
  - push_to_hub
  - run_form_submit
  - record_workspace_event
completion:
  - 已确认任务属于 Hub 搜索、详情、复制、发布或推送
  - 只读查询已优先使用搜索或详情接口
  - 发布、推送、复制等副作用操作已获得用户明确授权
  - 已确认目标目录、Hub 路径、版本和远程 Hub 地址等关键字段
---

# Hub OpenAPI SOP

## 使用条件

用户要搜索 Hub、查看 Hub 详情、复制复用资源、发布目录到 Hub、推送已发布目录更新时，使用本 skill。

## 标准流程

1. 读取 `required_docs`。旧 `/system/prompt/workspace/*` SOP 已下线，不再读取。
2. 用 `search_tools` 搜索 `/system/openapi/hub` 下的函数；如果只是快速查找复用资源，也可以用内置 `search_hub_directory`。
3. 只读任务优先用 search/detail；写入任务必须先确认目标目录、版本、远程 Hub 地址和影响范围。
4. 发布、推送、复制都属于副作用操作，必须得到用户明确授权后再执行。
5. 如果 Hub 函数返回资源路径或版本，最终回复中给出可追踪的路径和版本。

## 当前函数

- `/system/openapi/hub/search.form`：搜索 Hub 资源。
- `/system/openapi/hub/detail.form`：读取 Hub 资源详情。
- `/system/openapi/hub/publish.form`：发布目录到 Hub。
- `/system/openapi/hub/push.form`：推送已发布目录到 Hub。
- `/system/openapi/hub/push_info.form`：查询 Hub 推送预填信息。
- `/system/openapi/hub/copy.form`：复制 Hub 或本地目录。
