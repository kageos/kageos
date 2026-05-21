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
bash deploy/dev/scripts/infra.sh up
```

若你想显式指定容器引擎：

```bash
bash deploy/dev/scripts/infra.sh docker up -d
bash deploy/dev/scripts/infra.sh podman up -d
```

等价的原始命令如下。

Docker 本地开发：

```bash
docker compose -f deploy/dev/compose/docker-compose.dev.yml up -d
```

Podman 本地开发：

```bash
podman compose -f deploy/dev/compose/docker-compose.infra.yml up -d
```

两份 compose 都保留的原因很简单：

- `docker-compose.dev.yml` 给 Docker 本地开发用，容器名和 volume 名更偏“项目隔离”，避免和你机器上别的容器混淆。
- `docker-compose.infra.yml` 给 Podman 本地开发用，容器名必须固定为 `mysql8 / nats-server / minio`，这样 `app-runtime` 的基础设施探测逻辑才能直接识别。
- 它们的服务内容基本一致，主要差异就是容器名和 volume 名；不要试图为了“只保留一份文件”把这层约束藏起来。

### 1.1 开发配置

本地开发的 canonical 配置目录是：

```text
deploy/dev/config/
```

当前服务配置加载会读取 `deploy/dev/config/*.yaml`。备份、消息和控制面服务已从 MVP 删除，本地开发不再需要它们的独立配置。

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
bash deploy/base/scripts/build-app-base-image.sh
```

如需即使 tag 已存在也重建：

```bash
bash deploy/base/scripts/build-app-base-image.sh --force
```

如需自定义 tag，可临时指定：

```bash
APP_BASE_IMAGE="kagebase:latest" bash deploy/base/scripts/build-app-base-image.sh
```

如需强制重建且不复用缓存：

```bash
bash deploy/base/scripts/build-app-base-image.sh --force --no-cache
```

但注意：dev 默认配置 [app-runtime.yaml](config/app-runtime.yaml) 里的 `container.image.base_image` 默认也是 `kagebase:latest`。如果你真要用自定义 tag，记得同时改这里。

### 3. 起前端

```bash
cd web
npm run dev
```

若只跑前端、连线上后端，请在 `web/.env.development.local` 中配置 `VITE_PROXY_TARGET`。

本地开发相关资源的 canonical 位置已收敛到本目录及 `deploy/base/`。

## 开发配置约定

- `deploy/dev/config/` 里的密钥、SMTP、系统用户密码都应视为**占位值**，不要提交真实凭据。
- 如果你本地确实要联通真实 SMTP / 其他外部服务，请改你自己的本地配置，不要把真实账号密码写回仓库。
