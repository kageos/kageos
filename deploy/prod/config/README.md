# `deploy/prod/config/`

本目录保留镜像内置模板与运行时落盘目录：

- `template/`：镜像内置模板源。`deploy/prod/Dockerfile` 会把它复制到镜像内，作为无外部挂载时的兜底模板。
- `runtime/`：生产运行时生效目录。容器启动时由 `/app/config.prod.template` 渲染生成。

标准生产入口是 `aosctl`。它会把配置渲染到 `deploy/prod/.generated/config/`，再由 Compose 挂载到容器内 `/app/config.prod.template`。

常见模板变量：

- `${MYSQL_ROOT_PASSWORD}`
- `${JWT_SECRET}`
- `${CONTROL_ENC_KEY}`
- `${MINIO_ROOT_PASSWORD}`
- `${NATS_URL}`
- `${APP_BASE_IMAGE}`
- `${SMTP_HOST}`
- `${SMTP_PORT}`
- `${SMTP_USERNAME}`
- `${SMTP_PASSWORD}`
- `${SMTP_FROM}`
- `${SMTP_FROM_NAME}`

说明：

- MinIO 管理员用户名固定为 `minioadmin`，backup Basic Auth 用户名固定为 `admin`，不再作为标准部署配置项暴露。
- `app-server.yaml` 已不再消费 SMTP 变量；这组变量现在主要供 `hr-server.yaml` 的邮件验证码链路和 `message-server.yaml` 的系统通知/业务消息链路使用。
- `timer-scheduler.yaml` 的 `db.name` 默认是 `timer-scheduler`，中心调度服务独立保存通用 task、execution、outbox。
- `app-server.yaml` 的 `scheduled_task_db.name` 默认是 `app-scheduled-task`，只保存 app-server 侧业务任务和执行记录；调度状态统一在 `timer-scheduler`。
- `${APP_BASE_IMAGE}` 用于渲染 `app-runtime.yaml` 里的 `container.image.base_image`。
- 生产 scheduler 容器现在启动 `timer-scheduler`，healthcheck 默认探测 `127.0.0.1:9108/health`。

常见服务端口模板字段：

- `server.listen_host`：监听地址。生产模板默认 `127.0.0.1`，用于把内部服务收口到宿主机本地。
- `server.enable_pprof`：是否启用 `/debug/pprof`。生产模板默认 `false`。
- `runtime.listen_host`：`app-runtime` 的监听地址。生产模板默认 `127.0.0.1`。
- `timer_scheduler.base_url`：业务服务通过 SDK 连接中心调度服务的内部地址，生产默认 `http://127.0.0.1:9108/timer/api/v1`。
