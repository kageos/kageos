# `deploy/prod/config/`

本目录现在分两层：

- `template/`：**官方生产模板源**。`deploy/prod/Dockerfile` 会把它复制到镜像内，再由 entrypoint 做 `envsubst` 渲染。
- `runtime/`：**生产运行时生效目录**。容器启动时由 `template/` 渲染生成。

常见模板变量：

- `${MYSQL_ROOT_PASSWORD}`
- `${JWT_SECRET}`
- `${CONTROL_ENC_KEY}`
- `${MINIO_ROOT_PASSWORD}`
- `${APP_BASE_IMAGE}`
- `${SMTP_HOST}`
- `${SMTP_PORT}`
- `${SMTP_USERNAME}`
- `${SMTP_PASSWORD}`
- `${SMTP_FROM}`
- `${SMTP_FROM_NAME}`
- `${AOS_SCHEDULER_HEALTH_PORT}`

说明：

- MinIO 管理员用户名固定为 `minioadmin`，backup Basic Auth 用户名固定为 `admin`，不再作为标准部署配置项暴露。
- `app-server.yaml` 已不再消费 SMTP 变量；这组变量现在主要供 `hr-server.yaml` 的邮件验证码链路使用。
- `app-server.yaml` 的 `scheduler_db.name` 默认是 `app-scheduler`，与 `db` 复用同一个 MySQL 实例，只隔离定时任务表。
- `${APP_BASE_IMAGE}` 用于渲染 `app-runtime.yaml` 里的 `container.image.base_image`。
- `${AOS_SCHEDULER_HEALTH_PORT}` 用于同步渲染 `scheduler.health_port` 与容器 healthcheck 探测端口，默认 `9098`。

常见服务端口模板字段：

- `server.listen_host`：监听地址。生产模板默认 `127.0.0.1`，用于把内部服务收口到宿主机本地。
- `server.enable_pprof`：是否启用 `/debug/pprof`。生产模板默认 `false`。
- `runtime.listen_host`：`app-runtime` 的监听地址。生产模板默认 `127.0.0.1`。
- `scheduler.health_port`：`app-scheduler` 的本地健康探针端口。生产默认 `9098`，应与 healthcheck 使用同一个 `${AOS_SCHEDULER_HEALTH_PORT}`。
