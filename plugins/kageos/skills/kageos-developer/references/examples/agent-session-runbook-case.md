# 定时会话提示词最佳实践案例

适用：用户要创建无人值守的智能体定时会话，例如每日分析数据、跨多个工具搜集情报、生成 Markdown 报告并推送。此类案例不是 Go 代码，而是给 agent-server 执行的 runbook。

## 代码内置默认定时会话

如果一个工作台应用希望“安装后自带一条可选定时会话”，优先在 package 下用 `packageContext.AddAgentTask(...)` 声明。默认任务可以和默认 runbook 一起放在 `kageos_manifest.go`；不要为了暴露提示词再做一个 `daily_report_runbook.form`，除非用户明确需要手动生成/复制 runbook。

```go
package sample

import "github.com/kageos/kageos-sdk/agent-app/app"

// 这里是应用出厂推荐的默认定时会话模板。
// 平台安装或 update 后会在任务缺失时创建它；用户后续在平台启用、暂停、修改、删除后，以平台运行态配置为准。
// 因此实际定时会话可能和这里不同，这是符合预期的。
func init() {
	packageContext.AddDocs(app.DocManifest{
		Code:    "runbook.docs",
		Name:    "运行手册",
		Content: sampleDailyReportRunbook,
		Format:  "markdown",
	})

	packageContext.AddAgentTask(app.AgentTask{
		Code:               "sample_daily_report",
		Title:              "每日复盘报告",
		Description:        "每天读取业务数据并生成复盘报告。",
		CronExpr:           "0 8 * * *",
		Timezone:           "Asia/Shanghai",
		Enabled:            false,
		Message:            sampleDailyReportRunbook,
		MaxDurationSeconds: 900,
	})
}
```

约定：

- `Code` 必须稳定且在当前 package 内唯一，用于平台幂等同步。
- 不要填写 `Policy`；默认就是 `create_if_missing`，已有任务不会被代码 seed 覆盖。
- 默认 runbook 使用 `packageContext.AddDocs(app.DocManifest{Code: "runbook.docs", ...})`；不要使用 `AddRunbook` 或额外封装。
- 子目录文档继续使用原 `packageContext.AddDocs(app.DocManifest{Code: "./docs/readme.docs", ...})`；不要为 docs 新建 `PackageContext`。
- `CronExpr` 和 `EverySeconds` 必须二选一；国内业务建议显式写 `Timezone: "Asia/Shanghai"`。
- `Enabled:false` 表示安装后默认暂停，让用户在平台里选择性开启；定时会话会启动 Agent 会话，成本明显高于固定定时函数，不要默认开启。
- 代码声明只是“出厂默认设置”；首次创建后，后续平台里新增、暂停、修改、删除后的真实任务以平台运行态为准。
- `Message` 里引用当前 package 的函数、表格、图表和文档时必须用 `<./xxx.form>`、`<./xxx.table>`、`<./xxx.chart>`、`<./runbook.docs>` 这类带 `./` 的尖括号资源引用，禁止写 `./xxx.table`、`<xxx.table>` 或 `/<user>/<app>/...` 绝对工作台路径；这样导出、fork、安装到别的 workspace 后仍可用。只有明确调用外部服务、跨 workspace 或跨 package 的资源时，才写绝对路径并说明原因。

## 推荐结构

```markdown
## 执行身份

你正在执行一个无人值守的定时会话。运行时用户不在线，不要向用户提问，不要等待确认。遇到可恢复错误时降级输出；遇到不可恢复错误时发送失败通知并结束。

## 长期目标

<用一句话说明每天/每小时要完成的业务结果>

## 绑定范围

- 目标资源：
  - `<./items.table>`
  - `<./submit.form>`
  - `<./summary.chart>`
  - `<./runbook.docs>`
  - <外部系统或消息通道名称>
  - <必要的数据字段或时间范围>

## 预期工具

- <tool_a>：<用途>
- <tool_b>：<用途>
- <notification_tool>：发送最终报告

## 执行步骤

1. 计算本轮时间窗口，使用绝对时间边界，避免重复或漏取。
2. 查询或读取全部必要数据，分页 page_size 设足够大；超过上限时继续翻页。
3. 清洗字段并归一化关键维度，例如状态、楼层、部门、负责人、来源。
4. 从基础统计、分组对比、异常检测、趋势和行动建议几个维度分析。
5. 生成适合移动端阅读的 Markdown 报告。
6. 发送站内信、群机器人或目标消息通道。
7. 在会话日志输出一行摘要：分析日期、记录数、覆盖维度数、异常数、发送状态。

## 质量控制

- 时间范围必须严格限定本轮窗口。
- 没有数据也要正常输出空报告，不要报错。
- 所有外部消息只发送一次；失败时在日志记录错误。
- 报告里区分事实、计算结论和建议，不要把推测写成事实。

## 失败处理

- 关键数据读取失败：发送“数据获取失败，请检查数据源状态”的通知。
- 非关键数据读取失败：继续输出报告，并在报告注明数据不完整。
- 通知发送失败：输出到会话日志，不重复尝试无限循环。
```

## 写作要点

- 第一段必须说明“无人值守，不向用户提问”。
- 绑定范围写带 `./` 的尖括号资源引用，例如 `<./orders.table>`、`<./send_notice.form>`、`<./runbook.docs>`；不要写本机文件路径、`./xxx.table`、`<xxx.table>` 或同 package 的绝对工作台路径。
- 工具调用顺序写清楚，但不要把密钥、真实个人联系方式、一次性 token 写进提示词。
- 输出格式要固定，便于长期追踪和回放。
