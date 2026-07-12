# `deploy/prod/config/`

本目录保留镜像内置模板与运行时落盘目录：

- `template/`：镜像内置模板源。`deploy/prod/Dockerfile` 会把它复制到镜像内，作为无外部挂载时的兜底模板。
- `runtime/`：生产运行时生效目录。容器启动时由 `/app/config.prod.template` 渲染生成。

标准生产入口是 `kagectl`。它会把配置渲染到 `.kageos/prod/generated/config/`，再由 Compose 挂载到容器内 `/app/config.prod.template`。

常见模板变量：

- `${MYSQL_ROOT_PASSWORD}`
- `${JWT_SECRET}`
- `${KAGEOS_CONTROL_PLANE_SECRET}`
- `${SYSTEM_USER_PASSWORD}`
- `${MINIO_ROOT_PASSWORD}`
- `${NATS_URL}`
- `${KAGEOS_APP_BASE_IMAGE}`
- `${KAGEOS_REGISTRATION_MODE}`
- `${SMTP_MODE}`
- `${SMTP_HOST}`
- `${SMTP_PORT}`
- `${SMTP_USERNAME}`
- `${SMTP_PASSWORD}`
- `${SMTP_FROM}`
- `${SMTP_FROM_NAME}`

说明：

- `hr-server.yaml` 会消费 `${SYSTEM_USER_PASSWORD}` 初始化 `system` / `test_user` 的密码；标准入口由 `kage.yaml` 的 `system_user.password` 渲染。
- `hr-server.yaml` 会消费 `${KAGEOS_REGISTRATION_MODE}` 控制自助注册策略，生产默认 `admin_only`。
- `app-server.yaml` 已不再消费 SMTP 变量；这组变量现在主要供 `hr-server.yaml` 的邮件验证码链路使用。
- `${KAGEOS_APP_BASE_IMAGE}` 用于渲染 `app-runtime.yaml` 里的 `container.image.base_image`；不传时默认 `kagebase:latest`。`kagectl` 也支持用 `KAGEOS_APP_BASE_IMAGE` 临时覆盖 `images.app_base`。

常见服务端口模板字段：

- `server.listen_host`：监听地址。生产模板默认 `127.0.0.1`，用于把内部服务收口到宿主机本地。
- `server.enable_pprof`：是否启用 `/debug/pprof`。生产模板默认 `false`。
- `runtime.listen_host`：`app-runtime` 的监听地址。生产模板默认 `127.0.0.1`。
- `timer-scheduler.yaml`：独立定时调度服务配置，默认监听 `127.0.0.1:9098`，数据库名为 `timer-scheduler`。
- `message-server.yaml`：独立消息服务配置，默认监听 `127.0.0.1:9099`，数据库名为 `message-server`，MVP 只写站内信。

## 本次控制面加固的升级注意事项

- 新部署必须为 `KAGEOS_CONTROL_PLANE_SECRET` 使用独立、至少 32 字节的随机值；旧配置暂时回退到 `JWT_SECRET` 仅用于平滑升级。
- 旧版曾由 `hr-server` 与 `app-server` 竞争设置 OpenAPI token store，历史 token 记录可能分散在两个数据库。最安全的升级方式是让旧 OpenAPI token 全部失效并重新签发。若必须合库，应按 `token_hash` 去重，任一副本已撤销则保持撤销，过期时间取更早值，且不要沿用旧记录 ID。
- 使用新版 SDK 重新编译 App 后，NATS 凭据才会从 App 环境变量迁移到只读 runtime secret；未重编译的旧 App 暂时保留兼容路径。
- 旧 SDK 曾把带 userinfo 的 NATS URL 写入连接日志；升级完成后应轮换一次现有 NATS 凭据，并按日志保留策略清理历史副本。
- runtime secret 只减少凭据在环境变量、进程参数和日志中的暴露，不构成 NATS 租户隔离；同进程恶意代码仍可能读取自身凭据，共享 NATS 上的跨 App 订阅/抢占仍需后续用 Accounts/ACL（或物理隔离）解决。
