# `deploy/prod/config/`

本目录现在分两层：

- `template/`：**官方生产模板源**。`deploy/prod/Dockerfile` 会把它复制到镜像内，再由 entrypoint 做 `envsubst` 渲染。
- `runtime/`：**生产运行时生效目录**。容器启动时由 `template/` 渲染生成。

兼容说明：

- 旧路径 `deploy/config/prod/` 仍由 entrypoint 建立软链，指向 `deploy/prod/config/runtime/`。
- 裸机 / 历史路径若仍直接读取 `deploy/config/prod/`，当前不会被打断。

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
