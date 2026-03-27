# 本地开发入口（Official Dev）

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

### 1.1 开发配置

本地开发的 canonical 配置目录是：

```text
deploy/dev/config/
```

当前服务配置加载会读取 `deploy/dev/config/*.yaml`。

### 2. 起后端

推荐在仓库根目录：

```bash
bash deploy/dev/scripts/run-backend.sh
```

说明：

- 脚本会默认设置 `APP_ENV=dev`
- 脚本会默认设置 `AI_AGENT_OS_DEV_SKIP_EMBEDDING_INFRA=1`
- 脚本会默认设置 `AI_AGENT_OS_ROOT=<仓库根目录>`
- 如需手动控制，仍可直接执行 `go run ./core/cmd/main`

### 3. 起前端

```bash
cd web
npm run dev
```

若只跑前端、连线上后端，请在 `web/.env.development.local` 中配置 `VITE_PROXY_TARGET`。

本地开发相关资源的 canonical 位置已收敛到本目录及 `deploy/base/`。
