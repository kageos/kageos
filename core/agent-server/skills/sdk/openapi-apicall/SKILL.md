---
id: sdk.openapi-apicall
name: sdk-openapi-apicall
description: 在 SDK 函数或 /system/openapi 工作空间中调用平台 Web API 时使用。约束统一走 ctx.APICall，携带 token、trace、request_user、source 信息，不自建 HTTP 客户端。
triggers:
  - APICall
  - ctx.APICall
  - 平台 API
  - OpenAPI
  - 调用平台接口
  - token
  - trace
  - 权限校验
modes:
  - dev
  - modify
  - agent
required_docs:
  - /system/prompt/sdk/platform-api-reference
  - /system/prompt/platform-cross-cutting-capabilities
capabilities:
  - /system/openapi
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - read_go_file_lines
  - search_tools
  - write_go_file
  - search_replace_file
  - build_workspace
  - run_form_submit
completion:
  - 已确认这是平台接口调用，不是普通文件/媒体工具
  - 已统一使用 ctx.APICall
  - 未硬编码 token、用户身份或平台内网地址
  - 已确认权限与审计由平台 Web API 承接
  - 已 build_workspace 并验证核心路径
---

# SDK APICall 写法

## 使用条件

当 SDK 代码需要访问 AgentOS 平台能力，或编写 `/system/openapi` 官方平台接口包装函数时使用本 skill。

## 规则

1. 只有一个入口：`ctx.APICall(method, path, reqBody, respData)`。
2. `path` 使用平台网关路径，例如 `/hub/api/v1/directories`。
3. SDK 自动传递 token、trace、request_user、department、client_source、source_type、source_ref。
4. 平台服务端负责权限、审计和数据边界。
5. 不要裸写 HTTP client，不要硬编码 token，不要直连平台数据库。

## 流程

1. 读 `/system/prompt/sdk/platform-api-reference`。
2. 若是平台工作空间函数，再读具体 `system.openapi.*` skill。
3. 在对应 domain 目录下实现 wrapper，目录按平台领域隔离。
4. 用 `ctx.APICall` 调平台 API。
5. `build_workspace` 并用 `run_form_submit` 或只读工具验证。
