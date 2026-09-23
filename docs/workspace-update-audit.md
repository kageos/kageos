# 工作空间更新日志

`workspace.updated` 继续复用 `operate_logs`，无需新增表或迁移。新日志在 `details_json` 保存运行时返回的功能定义差异：名称、完整路径、类型的快照，按新增、修改、删除分组。删除项使用 SDK 返回的历史 API 信息，不依赖当前 Service Tree。列表摘要带版本和数量，展开可查看明细、变更说明、构建 trace、提交哈希和告警。

`changes.mode=diff` 表示 SDK API 差异，`added/updated/deleted` 为去重并按路径排序的轻量数组。全量 `Packages` 不属于新增目录；API 差异也不代表源码逻辑、业务数据、文档或定时任务的完整变化。日志不保存完整源码或 schema。

`ForceDiff` 会清空 SDK 比较基准，因此日志使用 `changes.mode=resync` 和 `synced` 清单，不把全量注册误标为新增，也不据此推断删除。仅写文件使用 `outcome=write_only`，不展示发布变更。缺少 Diff 或历史日志未记录明细时显示“功能变更明细不可用”，与空 Diff 的“未检测到功能定义变更”区分。

`outcome` 区分 `completed`、`write_only`、`runtime_failed`、`finalization_failed` 和 `metadata_warning`。`warnings` 保留运行时及元数据同步告警，`error` 保留失败原因。现有成功/失败状态保持兼容；前端遇到成功但带告警的更新日志显示告警状态。版本记录失败或元数据同步告警时，展示的是运行时报告的差异，不承诺所有平台元数据已经生效。runtime 返回失败且未提供版本信息时，不推断部署或回滚已经完成。

验证覆盖日志实际持久化、删除快照、路径回退、去重、强制同步、缺失与空差异、失败/告警/仅写入状态，以及前端中英文文案和分组展示。历史日志不做推测性回填。
