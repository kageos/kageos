# SDK 按需参考文档

本目录只放低频或专项知识。不要默认注入这些文档；只有当前身份 SOP 明确命中某个问题时，模型才用 `read_doc` 读取对应路径。

## 参考路径

- `/system/prompt/sdk/reference/runtime-capabilities`：程序里发消息、取当前用户/部门、调用平台 API、事务、副作用顺序、Table 回调高级能力、Python 运行时。
- `/system/prompt/sdk/reference/build-validation`：`build_workspace`、启动期 `CompileAndValidate()`、schema/widget/路由/未定义 SDK API 报错排查。
- `/system/prompt/sdk/reference/platform-api`：SDK 代码中用 `ctx.APICall` 调平台 Web API，或用 `/system/openapi` 包装平台接口。

## 使用方式

- 创建/修改主链路优先读 `/system/prompt/sdk/agent-app-sdk-readme` 和匹配案例。
- 发现上下文缺少专项细节时，再读取本目录明确路径。
- 读取后只使用文档中确认过的 SDK API、字段、参数和调用方式；仍不确定时读取源码。
