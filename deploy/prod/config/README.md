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
- `${SMTP_HOST}`
- `${SMTP_PORT}`
- `${SMTP_USERNAME}`
- `${SMTP_PASSWORD}`
- `${SMTP_FROM}`
- `${SMTP_FROM_NAME}`

常见服务端口模板字段：

- `server.listen_host`：监听地址。生产模板默认 `127.0.0.1`，用于把内部服务收口到宿主机本地。
- `server.enable_pprof`：是否启用 `/debug/pprof`。生产模板默认 `false`。
- `runtime.listen_host`：`app-runtime` 的监听地址。生产模板默认 `127.0.0.1`。
