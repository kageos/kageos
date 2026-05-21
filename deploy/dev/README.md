# 本地开发入口

本目录是 **本地开发** 的官方入口。

目标：

- 本地起基础设施
- 本地起后端
- 本地起前端
- 不再要求开发同学去记 `customer`、`embedding`、根目录 compose 等历史路径

## 官方入口

### 1. 起基础设施

推荐直接用官方脚本：

```bash
go run ./cmd/kagectl init-dev
```

该命令默认使用 Podman 启动 MySQL / NATS / MinIO，执行幂等数据库初始化 SQL，并确保本地用户应用运行时基础镜像 `kagebase:latest` 存在；如果镜像已存在会跳过构建。
首次执行时会生成 `.kageos/dev/env/kageos.env`，里面包含 MySQL、NATS、MinIO、JWT、system user 等本地随机密钥；后续重复执行会复用已有值，避免把已有本地数据库密码刷掉。
初始化结束后会在终端打印英文表格，列出本地开发需要记录的路径、账号、密码和 key。
本地 dev MySQL 默认只绑定宿主机 `127.0.0.1:3318`，容器内部仍是 `3306`，这样不容易和你机器上的旧 MySQL 或其它项目冲突。
幂等规则：只有 `.kageos` 不存在时才初始化生成 secrets；一旦 `.kageos` 存在，`init-dev` 只应用 `.kageos/dev/env/kageos.env` 里的值，不会隐式生成替换密码。

若你想显式指定容器引擎：

```bash
go run ./cmd/kagectl init-dev --engine docker
go run ./cmd/kagectl init-dev --engine podman
```

如只想初始化 MySQL / NATS / MinIO，不处理基础镜像：

```bash
go run ./cmd/kagectl init-dev --skip-base
```

如需重新生成本地密钥：

```bash
go run ./cmd/kagectl init-dev --regen-secrets
```

注意：如果 MySQL/MinIO 已经有旧 volume，重新生成密码后需要清理旧本地 volume，否则容器里的历史密码不会自动变化。
当前 dev MySQL 使用独立 volume `kageos-dev-mysql3318-data`；历史 `3306` 时代的 volume 不会被自动复用。

如需初始化时使用自定义基础镜像 tag：

```bash
go run ./cmd/kagectl init-dev --base-image "registry.example.com/kagebase:stable"
```

等价的原始命令如下。

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

当前服务配置加载会读取 `.kageos/dev/config/*.yaml`。备份、消息和控制面服务已从 MVP 删除，本地开发不再需要它们的独立配置。

### 2. 起后端

本地开发后端从 GoLand 启动平台主进程：

- Run target：`core/cmd/main/main.go`
- Working directory：仓库根目录
- Environment：`APP_ENV=dev`

说明：

- `APP_ENV=dev` 时启动预检会自动按本地 compose 基础设施模式处理，无需额外环境变量
- GoLand 本地开发直接启动 `core/cmd/main/main.go` 也需要设置 `APP_ENV=dev`
- 如需命令行临时启动，等价命令是 `APP_ENV=dev go run ./core/cmd/main`

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

dev 默认配置 [app-runtime.yaml](config/app-runtime.yaml) 里的 `container.image.base_image` 默认也是 `kagebase:latest`。如需临时切换运行时镜像，可在启动 app-runtime / core-server 时设置 `KAGEOS_APP_BASE_IMAGE`。

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
