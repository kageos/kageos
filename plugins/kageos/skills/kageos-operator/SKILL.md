---
name: kageos-operator
description: 通过真实鉴权 HTTP 请求、实时 schema 发现、Form/Table/Chart 和 OnSelectFuzzy 调用、文件上传、用户选择、富文本资源、业务读回、测试数据清理、自动化证据和可选 UI 检查，操作并验收一个已经开发和构建的 kageos 工作空间目录。在 kageos-developer 之后、发布之前，用户要求测试、操作、验收、走通或证明目录真实可用时使用。
---

# kageos 真实验收

像真实业务操作员一样验证已经安装和构建的目录。本 Skill 不重新设计、不修改代码、不执行构建，也不发布。

## 输入

收集或发现：

- kageos gateway 基础地址。
- 工作空间目录 `full_code_path`，不是本地磁盘路径。
- `KAGEOS_OPENAPI_TOKEN`，或仅在本地/测试环境临时使用 `KAGEOS_ACCESS_TOKEN`。
- 预期业务场景和安全的合成测试数据。
- 可选的浏览器地址，用于视觉一致性检查。

凭证只保存在环境变量中。不得打印，不得写进命令文件、报告或源码。优先使用权限最小的专用测试身份；无人值守或生产自动化不得使用管理员访问 JWT。

## 工作流

1. 读取 `references/http-operation-contract.md`、`references/complex-input-contract.md` 和 `references/verification-contract.md`。
2. 为本次验收生成一个稳定的来源引用和 trace ID，每个请求都发送要求的审计 headers。
3. 直接调用 discovery 接口，检查当前函数 schema、Table 回调、字段级 `OnSelectFuzzy`、定时函数和 AgentTask。不得猜测路径、字段、枚举或回调能力。
4. 检查每个输入组件。提交前解析动态值：上传 `files` 和富文本资源，搜索并补全 `user/users`，查询 `OnSelectFuzzy`。不得猜测 ref、username 或外键 ID。
5. 直接执行只读操作。每次都先检查真实响应，再决定下一步；不得假设响应一定包含 `data.list` 或 `data.total`。
6. 第一次写入前，用用户能看懂的语言展示准确函数、本地文件、合成标记、预期业务影响、上传对象清理和记录清理动作，并获得一次明确授权。不要创建 JSON 执行计划。
7. 对已授权的写操作逐个执行。写入和上传不得自动重试。返回的 ID 和文件 ref 只保存在当前执行上下文，并通过只读操作证明保存或更新后的业务状态。
8. 每个发现的 `OnSelectFuzzy` 都调用 `by_keyword`。返回值再次使用前，标量调用 `by_value`，数组调用 `by_values`；不得猜测外键 ID。
9. 只测试真实业务链路需要的操作。不能因为发现了 CRUD 回调就机械地全部调用。
10. 始终尝试确定性清理本次创建的合成记录和上传对象。不得删除预先存在或属于客户的数据。
11. 如果提供了浏览器地址，使用可用的 Browser 或 Chrome 控制能力走同一场景。不得读取 Cookie、本地存储或隐藏状态。
12. 验收属于发布或完整交付时，必须落盘 `kageos.operator-report.v1` JSON；单独探索操作时，在用户要求制品时生成。场景和证据使用用户语言，操作名、路径、状态和 schema 字段保持稳定。JSON 只记录机器可校验证据，不驱动执行。
13. 每个保存的 JSON 报告都用 `python3 scripts/render_report.py <report.json>` 渲染。向用户返回 JSON、Markdown 和自包含 HTML；不得手工修改渲染报告来改变验收状态或证据。

## 鉴权

只能选择一种模式：

- OpenAPI：发送 `Authorization: Bearer $KAGEOS_OPENAPI_TOKEN`。
- 临时本地/测试访问 JWT：发送 `X-Token: $KAGEOS_ACCESS_TOKEN`。

访问 JWT 不得作为 Bearer 发送，OpenAPI token 不得作为 `X-Token` 发送。证据中只记录 `auth_mode`，不记录凭证。

## 自动化

盘点 discovery 返回的每个定时函数和 AgentTask。发现定义只证明已经打包；只有真实读取或轮询观察到输出，才能标记为运行时已验证。无法安全触发或当前停用时，报告为 `blocked` 并写清原因。

## 门禁

只有所有必要业务断言通过、清理成功、每个发现的自动化都有运行证据，并且提供 UI 时前后端结果一致，才能返回 `verified`。否则返回 `blocked`，列出准确缺失或失败证据。

## 交接

交给 Publisher：目录 `full_code_path`、JSON/Markdown/HTML 报告路径、浏览器地址和关键页面、已知敏感字段、简洁发布事实；完整交付还要传递当前 `kageos.delivery-run.v1` 路径。
