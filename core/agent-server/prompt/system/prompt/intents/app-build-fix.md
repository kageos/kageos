# 意图：app.build_fix 构建排错

## 使用条件

`build_workspace` 失败、Go 编译失败、schema compile failed、widget 校验失败、路由后缀错误、SDK API 未定义。

## 按需参考

- 构建失败、启动期 schema/widget/路由校验、`CompileAndValidate()` 细节不够：`read_doc("/system/prompt/sdk/reference/build-validation")`
- SDK API、Template、widget 参数不确定：`read_doc("/system/prompt/sdk/agent-app-sdk-readme")`
- 仍不确定真实符号或方法签名：读取对应 SDK 源码，不要继续猜。

## 流程

1. 读取完整错误，不截断关键路由和字段。
2. 按错误类型归类。
3. 同类问题批量修复，不一次只修一行。
4. 未确认的 SDK API 不继续猜，读取完整 SDK 或源码。
5. 修复后先处理写入/替换工具返回的文件级非阻断诊断，然后重新 `build_workspace`。跨文件/schema 问题以 build/startup 的完整错误为准，不再走独立预检流程。

## 常见错误

- `options_colors` 不是 RRGGBB。
- widget type 不在白名单。
- Table Request 和 Model 字段 code 冲突。
- OnSelectFuzzyMap 指向非 select/multiselect 字段。
- `types.Time` 没有直接 `Format`，应 `Time().Format(...)`。
- 编造未存在的 `chart.ComboChart`、`app.GET`、`app.POST`。
