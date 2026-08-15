# 通知 SOP

kageos 业务通知使用平台消息能力，不要在业务应用里自造通知表、通知队列，也不要硬连飞书、钉钉、企微或邮件。业务代码只表达通知对象、标题、正文和平台文件引用；具体站内信和外部 webhook 触达由平台 message-service 处理。

## 什么时候发通知

通知是打断用户的异步触达，不是每个 Form 成功后的默认动作。实时提交后页面已经返回结果的场景，默认不要再发通知。

适合发通知：

| 场景 | 示例 |
|---|---|
| 到期/风险/失败需要人处理 | 证书即将到期、网站连续失败、合同逾期、库存低于阈值 |
| 后台定时任务发现异常 | 每分钟巡检、每日扫描、自动续期失败 |
| 异步任务完成，用户不在页面等待 | 长报告生成完成、批处理完成、文件产物已生成 |
| 需要负责人跟进 | 新候选人投递、客户跟进到期、面试即将开始 |
| AgentTask 输出最终报告 | 每日情报、周报、异常归因报告 |

不适合默认发通知：

| 场景 | 做法 |
|---|---|
| 用户实时提交 Form 并马上看到结果 | 直接在 `resp.Form(...)` 里返回结果和文件 |
| 用户打开 Table 手动新增/编辑一行 | 让表格结果可见，不额外通知本人 |
| Chart 查询完成 | 直接展示图表 |
| 普通成功状态 | 不刷屏；只有异常、风险、异步完成或明确要求时通知 |

## Go 代码发送通知

在 Form、Table 回调或定时函数里使用 `ctx.SendNotification`：

```go
err := ctx.SendNotification(&app.SendNotificationOpts{
    ToUsers: "zhangsan,lisi",
    Title:   "会议即将开始提醒",
    Message: "您预约/参与的会议将在 10 分钟后开始，请提前准备。",
})
if err != nil {
    logger.Errorf(ctx, "send notification failed: %v", err)
}
```

字段说明：

| 字段 | 说明 |
|---|---|
| `ToUsers` | 接收人 username，多个用逗号分隔；为空时 SDK 会尝试默认通知请求用户 |
| `Title` | 通知标题 |
| `Message` | 通知正文，默认按 markdown 发送 |
| `ContentType` | 可选，默认 `markdown` |
| `Level` | 可选：`info`、`warning`、`critical` |
| `Files` | 可选，平台文件引用，多个用逗号分隔 |

## 文件附件

`files` 组件的值可以直接作为通知附件传给 `Files`。这通常用于异步完成或风险告警场景；实时 Form 已经返回文件时，默认直接在页面返回，不要再发通知。

```go
type AsyncReportReq struct {
    NotifyUsers string `json:"notify_users" widget:"name:通知人;type:users"`
}

type AsyncReportResp struct {
    Summary     string `json:"summary" widget:"name:执行结果;type:text"`
    ReportFiles string `json:"report_files" widget:"name:报告附件;type:files"`
}

func GenerateReport(ctx *app.Context, resp response.Response) error {
    var req AsyncReportReq
    if err := ctx.ShouldBindValidate(&req); err != nil {
        return err
    }

    // 生成报告并得到平台文件引用，不是本地路径。
    reportFiles := "bucket/report.xlsx"

    err := ctx.SendNotification(&app.SendNotificationOpts{
        ToUsers: req.NotifyUsers,
        Title:   "报告已生成",
        Message: "后台报告已生成，请查看附件。",
        Files:   reportFiles, // 平台文件引用可直接传，不要改成本地路径，不要重新上传。
    })
    if err != nil {
        logger.Errorf(ctx, "send report notification failed: %v", err)
    }

    return resp.Form(&AsyncReportResp{
        Summary:     "报告已生成",
        ReportFiles: reportFiles,
    }).Build()
}
```

注意：

- `Files` 传平台文件引用，不传本地磁盘路径。
- 请求里的 `files` 组件值可以直接传，例如 `Files: req.ReportFiles`。
- Form 返回的 output files、平台文件字段值也可以直接传。
- 多个附件用逗号、换行或中文顿号分隔都可以，SDK 会规范化。
- 外部 webhook 卡片通常只展示附件摘要，完整查看/下载回到 kageos 详情。

## 失败处理

- 普通业务流程不要因为通知失败而回滚主业务，记录日志或在响应里提示即可。
- 需要强通知保证的业务，要把通知状态写入业务表，例如 `notice_status`、`notice_error`，并提供重试入口。
- 定时提醒要幂等：发送前先 claim 或检查冷却时间，发送成功后写 `notified_at`/`notify_count`，失败时释放 claim 或记录错误。

## AgentTask 通知

AgentTask 的 `Message` 里引用内置通知工具时写 `<tool:send_notification>`，真实工具名仍是 `send_notification`。不要写 `<send_notification>`。

AgentTask 适合发送最终报告或异常摘要；固定业务提醒优先写 Go Form + `FormTemplate.Schedules`，在 Go 里调用 `ctx.SendNotification`。

## 推荐案例

- 会前提醒：`references/case_catalog/tables/meeting/prd.md`
- 默认定时函数：`references/examples/scheduled-function-case.md`
- AgentTask 通知报告：`references/examples/agent-session-runbook-case.md`
