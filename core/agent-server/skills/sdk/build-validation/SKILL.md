---
id: sdk.build-validation
name: sdk-build-validation
description: 分析或修复 build_workspace、app.Run、CompileAndValidate、widget validator、schema compile failed 等启动期校验错误时使用。
triggers:
  - build_workspace
  - 编译失败
  - 启动失败
  - CompileAndValidate
  - SDK schema compile failed
  - widget validator
  - schema 校验
  - 启动校验
modes:
  - qa
  - dev
  - modify
  - agent
required_docs:
  - /system/prompt/sdk/build-validation-reference
  - /system/prompt/sdk/widget-reference
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - read_go_file_lines
  - read_app_log
  - write_go_file
  - search_replace_file
  - build_workspace
completion:
  - 已区分 Go 编译错误、SDK schema 错误、runtime 启动错误和业务执行错误
  - 已一次性收集并解释聚合错误
  - 已按 widget/template/schema 规则修复根因
  - 已重新 build_workspace 验证
---

# SDK 构建与启动校验

## 使用条件

当 `build_workspace` 失败、启动失败、schema 校验失败、widget 参数错误、路由后缀不匹配时使用本 skill。

## 当前机制

`build_workspace` 会触发工作空间构建和部署。Go 编译成功后，新版本启动时 `app.Run()` 会先执行 `CompileAndValidate()`：

1. 校验路由后缀与 Template 类型。
2. 调 `getApis()` 解析全部 API schema。
3. 调 widget validator 校验 Go 类型、widget 参数、search/validate/display 等标签。
4. 调 `functionschema.Validate` 校验最终 schema。
5. 多个错误通过 `errors.Join` 聚合返回。
6. 失败时发布 startup failed，新版本不能认为启动成功。

## 修复方式

1. 先读完整错误，不要只修第一条。
2. 判断错误层级：Go 编译、SDK schema、runtime、业务执行。
3. widget 参数错误优先查 `/system/prompt/sdk/widget-reference`。
4. 路由/Template 错误查 `/system/prompt/sdk/form-table-chart-reference`。
5. 修复后重新 `build_workspace`，直到启动校验通过。
