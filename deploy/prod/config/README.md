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
