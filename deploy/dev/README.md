# 本地开发入口

本目录是 **本地开发** 的官方入口。

目标：

- 本地起基础设施
- 本地起后端
- 本地起前端
- 不再要求开发同学去记 `customer`、`embedding`、根目录 compose 等历史路径

## 官方入口

### 1. 一键启动开发后端

推荐首次开发直接用 `kagectl bootstrap --dev`：

```bash
go run ./cmd/kagectl bootstrap --dev
```

该命令会初始化开发模式、启动 MySQL / NATS / MinIO、确保本地用户应用运行时基础镜像存在，并以前台方式启动后端主进程。停止后端用 `Ctrl-C`；停止 MySQL / NATS / MinIO 用：

```bash
go run ./cmd/kagectl down
```

### 1.1 只初始化开发模式

如需只初始化，不启动后端主进程：

```bash
go run ./cmd/kagectl init --dev
```

该命令会写入 `.kageos/kageos.env`：

```dotenv
KAGEOS_MODE=dev
KAGEOS_DEV_ENGINE=podman
```

后续 `kagectl up/status/down/logs/doctor/verify` 会根据这个模式自动走本地开发链路，不需要再手工设置运行模式。

该命令默认使用 Podman 启动 MySQL / NATS / MinIO，执行幂等数据库初始化 SQL，并确保本地用户应用运行时基础镜像 `kagebase:latest` 存在；如果镜像已存在会跳过构建。
首次执行时会生成 `.kageos/dev/env/kageos.env`，里面包含 MySQL、NATS、MinIO、JWT、system user 等本地随机密钥；后续重复执行会复用已有值，避免把已有本地数据库密码刷掉。
初始化结束后会在终端打印英文表格，列出本地开发需要记录的路径、账号、密码和 key。
本地开发默认 `SMTP_MODE=log`，发送验证码时不会调用真实邮箱服务；验证码会写入后端日志，并通过 `send_email_code` 接口的 `debug_code` 返回，填这个验证码即可注册。生产环境需要真实发信时再改成 `SMTP_MODE=smtp` 并配置 `SMTP_*`。
本地 dev MySQL 默认只绑定宿主机 `127.0.0.1:3318`，容器内部仍是 `3306`，这样不容易和你机器上的旧 MySQL 或其它项目冲突。
幂等规则：一旦 `.kageos/dev/env/kageos.env` 存在，`init --dev` 只复用里面的值，不会隐式生成替换密码。

若你想显式指定容器引擎：

```bash
go run ./cmd/kagectl init --dev --engine docker
go run ./cmd/kagectl init --dev --engine podman
```

如只想初始化 MySQL / NATS / MinIO，不处理基础镜像：

```bash
go run ./cmd/kagectl init --dev --skip-base
```

如需重新生成本地密钥：

```bash
go run ./cmd/kagectl init --dev --regen-secrets
```

注意：如需重新生成本地密钥，请先清理本地开发数据 volume，否则已有 MySQL/MinIO 数据仍会使用原密码。

如需初始化时使用自定义基础镜像 tag：

```bash
go run ./cmd/kagectl init --dev --base-image "registry.example.com/kagebase:stable"
```

下面是排障时才需要看的底层 compose 命令，不是日常入口。

Docker 本地开发：

```bash
docker compose --env-file .kageos/dev/env/kageos.env -f deploy/dev/compose/docker-compose.dev.yml up -d
```

Podman 本地开发：

```bash
podman compose --env-file .kageos/dev/env/kageos.env -f deploy/dev/compose/docker-compose.infra.yml up -d
```

两份 compose 都保留的原因很简单：

- `docker-compose.dev.yml` 给 Docker 本地开发用，容器名和 volume 名更偏“项目隔离”，避免和你机器上别的容器混淆。
- `docker-compose.infra.yml` 给 Podman 本地开发用，容器名必须固定为 `mysql8 / nats-server / minio`，这样 `app-runtime` 的基础设施探测逻辑才能直接识别。
- 它们的服务内容基本一致，主要差异就是容器名和 volume 名；不要试图为了“只保留一份文件”把这层约束藏起来。

### 1.1 开发配置

本地开发的 canonical 配置目录是：

```text
.kageos/dev/config/
```

当前服务配置加载会读取 `.kageos/dev/config/*.yaml`。`message-server` 和 `timer-scheduler` 已经是主线平台服务，本地开发配置里应保留对应 YAML；本地基础设施仍只需要 MySQL、NATS 和 MinIO。历史备份控制面和旧控制面服务不属于当前本地开发必需项。

### 2. 单独起后端

命令行启动：

```bash
go run ./cmd/kagectl up
```

`kagectl up` 会先确保本地基础设施运行，再以前台方式启动 `core/cmd/main`。停止后端用 `Ctrl-C`；停止 MySQL / NATS / MinIO 用：

```bash
go run ./cmd/kagectl down
```

如需从 GoLand 启动平台主进程：

- Run target：`core/cmd/main/main.go`
- Working directory：仓库根目录

说明：

- GoLand 启动前先执行 `go run ./cmd/kagectl init --dev`，让 `.kageos/kageos.env` 记录开发模式。
- 服务配置会按 `.kageos/kageos.env` 自动读取 `.kageos/dev/config/*.yaml`。

### 2.1 构建用户应用运行时基础镜像

如果本地缺少用户应用运行时基础镜像，或想单独重建，请执行 canonical 脚本：

```bash
go run ./cmd/kagectl build-app-base
```

如需即使 tag 已存在也重建：

```bash
go run ./cmd/kagectl build-app-base --force
```

如需自定义 tag，可临时指定：

```bash
KAGEOS_APP_BASE_IMAGE="kagebase:latest" go run ./cmd/kagectl build-app-base
# 或：
go run ./cmd/kagectl build-app-base --image "kagebase:latest"
```

如需强制重建且不复用缓存：

```bash
go run ./cmd/kagectl build-app-base --force --no-cache
```

dev 默认配置 `.kageos/dev/config/app-runtime.yaml` 里的 `container.image.base_image` 默认也是 `kagebase:latest`。如需临时切换运行时镜像，可在启动 app-runtime / core-server 时设置 `KAGEOS_APP_BASE_IMAGE`。

### 3. 起前端

```bash
cd web
npm run dev
```

若只跑前端、连线上后端，请在 `web/.env.development.local` 中配置 `VITE_PROXY_TARGET`。

本地开发相关资源的 canonical 位置已收敛到本目录及 `deploy/base/`。

## 开发配置约定

- `.kageos/dev/config/` 和 `.kageos/dev/env/` 里的密钥、SMTP、系统用户密码都应视为本机私有值，不要提交。
- 如果你本地确实要联通真实 SMTP / 其他外部服务，请改你自己的本地配置，不要把真实账号密码写回仓库。
