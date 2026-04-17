# `deploy/prod/config/`

本目录现在分两层：

- `template/`：**官方生产模板源**。`deploy/prod/Dockerfile` 会把它复制到镜像内，再由 entrypoint 做 `envsubst` 渲染。
- `runtime/`：**生产运行时生效目录**。容器启动时由 `template/` 渲染生成。

常见模板变量：

- `${MYSQL_ROOT_PASSWORD}`
- `${JWT_SECRET}`
- `${CONTROL_ENC_KEY}`
- `${MINIO_ROOT_USER}`
- `${MINIO_ROOT_PASSWORD}`
- `${APP_BASE_IMAGE}`
- `${SMTP_HOST}`
- `${SMTP_PORT}`
- `${SMTP_USERNAME}`
- `${SMTP_PASSWORD}`
- `${SMTP_FROM}`
- `${SMTP_FROM_NAME}`

说明：

- `app-server.yaml` 已不再消费 SMTP 变量；这组变量现在主要供 `hr-server.yaml` 的邮件验证码链路使用。
- `${APP_BASE_IMAGE}` 用于渲染 `app-runtime.yaml` 里的 `container.image.base_image`。

常见服务端口模板字段：

- `server.listen_host`：监听地址。生产模板默认 `127.0.0.1`，用于把内部服务收口到宿主机本地。
- `server.enable_pprof`：是否启用 `/debug/pprof`。生产模板默认 `false`。
- `runtime.listen_host`：`app-runtime` 的监听地址。生产模板默认 `127.0.0.1`。
