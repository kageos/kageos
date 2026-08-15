# packageContext 目录自动对账

kageos 支持“代码即目录结构”。本地开发时不需要先去页面创建目录；写好 package 和函数后，build/update 会自动让平台侧补齐目录。

## 链路

1. Go package 通过 `packageContext.GET/POST` 注册函数，或通过 `packageContext.AddDocs/AddAgentTask` 声明 package 级 manifest。
2. SDK 在 update callback 中收集全量 package 信息。
3. diff 里返回 `packages`。
4. app-server 先 `reconcilePackages` 幂等创建缺失目录。
5. app-server 再为默认文档种子创建缺失 docs 节点，为默认定时会话创建缺失 Agent 任务，并为新增函数创建 function 和 service_tree 节点。

## 成立条件

- 新 package 被 `code/cmd/app/main.go` blank import。
- package 内至少注册了一个函数，或通过 `packageContext.AddDocs(...)` / `packageContext.AddAgentTask(...)` 显式登记 package 级 manifest。
- 执行了真正的 build/update，新版本启动并完成 SDK update callback。
- 不是 write-only。只写文件不会触发 diff，也不会触发目录对账。

## 自动创建什么

如果代码注册了 `RouterGroup: "/<package>/<child>"`，平台会补：

```text
/<user>/<app>/<package>
/<user>/<app>/<package>/<child>
```

父目录会先创建，子目录后创建。

如果在 package 的 `kageos_manifest.go` 中声明：

```go
func init() {
	packageContext.AddDocs(app.DocManifest{
		Code:    "runbook.docs",
		Name:    "运行手册",
		Content: "...",
		Format:  "markdown",
	})
}
```

平台会在 update 时幂等创建：

```text
/<user>/<app>/<package>/runbook.docs
```

默认策略是缺失才创建；如果树上已经存在 `runbook.docs`，平台不会覆盖已有文档内容。运行态权威内容以 service tree 上的 docs 为准，`kageos_manifest.go` 只是本地开发时的一次性种子声明。

同一 package 需要子目录文档时，不创建第二个 `PackageContext`，直接声明安全的相对多级路径：

```go
packageContext.AddDocs(app.DocManifest{
	Code:    "./docs/readme.docs",
	Name:    "文档/目录说明",
	Content: "# 文档目录说明\n",
	Format:  "markdown",
})
```

SDK 会把缺失的 `docs` 中间目录加入对账，并创建 `<package>/docs/readme.docs`。新代码统一写 `.docs` 后缀；绝对路径、反斜杠、空路径段、`.` 和 `..` 会被拒绝。`Name` 只表示文档名称，不表示目录名称。

如果在 package 的 `kageos_manifest.go` 中声明：

```go
func init() {
	packageContext.AddAgentTask(app.AgentTask{
		Code:     "daily_report",
		Title:    "每日复盘报告",
		Message:  "...",
		CronExpr: "0 8 * * *",
		Timezone: "Asia/Shanghai",
		Enabled:  false,
	})
}
```

平台会在 update 时创建缺失的默认 Agent 任务。不要填写 `Policy`；默认就是 `create_if_missing`。如果任务已经存在，平台不会覆盖它的 message、cron、启停状态、模型配置或附件配置。后续真实任务以 timer-scheduler / 平台运行态为准，`kageos_manifest.go` 只是首次安装或首次 update 的默认模板。

## 不自动做什么

- 不自动删除已经消失的空目录。
- 不保证改 `packageContext.Name/Desc` 后覆盖已有目录元数据。
- 不为没有注册函数、默认文档或默认定时会话的空 package 创建目录。
- 不用 `AddDocs` 覆盖已经存在的 docs 内容。
- 不用 `AddAgentTask` 覆盖已经存在的默认 Agent 任务。
- 不替代业务数据迁移和真实功能测试。

## 对 Agent 的要求

新增业务目录时，优先让模型直接创建 SDK package 和函数，不要先要求用户去页面点“新建目录”。完成代码后必须 build，目录会随 build 对账。
