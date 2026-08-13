# kageos 验证报告：/system/democase/minimax&&#35;95;h3

**状态：BLOCKED**

迁移并验证 MiniMax H3 V2 文生视频闭环；仅执行一次经用户批准的约 ¥2 真实付费测试

## 运行信息

| 项目 | 内容 |
| --- | --- |
| 目录 | /system/democase/minimax&&#35;95;h3 |
| 目标 | — |
| 认证模式 | access&&#35;95;token |
| 来源标识 | minimax-h3-v2-paid-test-20260813 |
| Trace ID | kageos-operator-minimax-h3-20260813-1134 |
| 开始时间 | 2026-08-13T11:20:00+08:00 |
| 完成时间 | 2026-08-13T11:43:00+08:00 |

## 验证检查

| 操作 | 路径 | 状态 | 证据 |
| --- | --- | --- | --- |
| directory.discovery | /system/democase/minimax&&#35;95;h3 | passed | 发现 config.form、generate.form、tasks.table、refresh.form 与使用说明，H3 目录结构可操作。 |
| config.form.submit | /system/democase/minimax&&#35;95;h3/config.form | passed | 从既有思源机密记录恢复 MiniMax API Key 到单例配置表；未在报告、聊天、任务表或截图中暴露密钥。 |
| generate.form.submit | /system/democase/minimax&&#35;95;h3/generate.form | passed | 仅提交一次 MiniMax-H3 V2 付费请求：4 秒、768P、16:9；目录预估费用 ¥2，服务端接受任务。 |
| tasks.table.search | /system/democase/minimax&&#35;95;h3/tasks.table | passed | 用唯一测试标记回读到 1 条记录；模型 MiniMax-H3，状态成功，预计费用 ¥2，生成文件已挂载。 |
| storage.resolve | /system/democase/minimax&&#35;95;h3/tasks.table | passed | 工作空间持久化对象可解析，文件名 minimax-h3-task-1.mp4，大小 2,136,079 字节，SHA-256 为 dfe94cf052fab05ae831c8bfdde25f8314743696b4c9db4914d3721b68e10f88。 |
| artifact.ffprobe | /system/democase/minimax&&#35;95;h3/tasks.table | passed | 下载后的 MP4 可解析：H.264 1344×768，AAC 音轨，时长约 4.46 秒。 |
| ui.agreement | /system/democase/minimax&&#35;95;h3/tasks.table | passed | 浏览器详情页显示成功、MiniMax-H3、4 秒、768P、16:9、2 元与 2.04 MB 可预览文件；生成表单字段与后端 schema 一致。 |

## 自动化证据

| 代码 | 类型 | 状态 | 证据或原因 |
| --- | --- | --- | --- |
| — | — | passed | 发现每 60 秒刷新任务；异步任务从已提交状态更新为成功并持久化成片，随后刷新返回无待处理任务。 |

## 清理结果

| 资源或操作 | 路径 | 状态 | 证据 |
| --- | --- | --- | --- |
| retain.test.deliverable | — | skipped | 按用户要求保留成功记录和成片供查看，因此未删除本次测试业务记录与对象。 |

## 问题与注意事项

- 严格 operator 验证状态只能标记 blocked：所有功能检查已通过，但用户要求保留测试成片，清理门禁未满足。
- 旧 MiniMax 海螺 2.3 目录仍在服务树中；发布 Hub 前应单独退役，避免与 H3 混淆。

## 敏感字段（仅字段名）

- config.form.api&&#35;95;key

---

由 `kageos-operator/scripts/render_report.py` 从机器可校验 JSON 生成。
