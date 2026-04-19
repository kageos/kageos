# 生产单机一键部署（Compose：宿主机 Docker 或 Podman）

> 官方生产入口：`deploy/prod/`

如果你只想一眼看懂并直接部署，直接看：

- [QUICK_START.md](QUICK_START.md)

如果你还想看一分钟版本，再看：

- [DEPLOY_TUTORIAL.md](DEPLOY_TUTORIAL.md)

如果你要看启动顺序和依赖链，再看：

- [DEPLOYMENT_FLOW.md](DEPLOYMENT_FLOW.md)

**范围**：主站 + 独立定时任务调度器 + 中间件（MySQL / NATS / MinIO）+ 内置 Nginx（默认 80，可选 443）+ **容器内 Podman**（跑用户应用）。**不包含 Hub**。

## 架构

```
公网 :80 / :443（可选）
  └─ main 容器（network_mode: host）
       ├─ Nginx :80 / :443  → 静态文件 / SPA fallback
       │             → proxy_pass API Gateway :9090
       │             → proxy_pass MinIO 127.0.0.1:9000
       └─ API Gateway :9090
              ├─ MySQL  127.0.0.1:3306
              ├─ NATS   127.0.0.1:4222
              └─ MinIO  127.0.0.1:9000

host 网络独立容器
  └─ scheduler 容器
       └─ app-scheduler（仅执行定时任务调度与投递）
```

`main` 使用 `network_mode: host`，容器内 Nginx 默认直接监听宿主机 80 端口；开启 HTTPS 后会额外监听 443。`scheduler` 也使用 `network_mode: host`，通过数据库租约 claim 防止重复执行，并直接依赖 `app-runtime` 的 `http://127.0.0.1:9093/health` 就绪；自身会在 `http://127.0.0.1:9098/health` 暴露进程内健康探针。中间件容器通过 `127.0.0.1` 暴露端口供 `main` / `scheduler` 访问，无需额外宿主机 Nginx。

## 前置

- Linux 宿主机；缺少宿主机 `podman` / `podman compose` 时，`bash build.sh init` 会自动尝试安装。
- Ubuntu 如果命中腾讯云 APT 镜像 `404`，`bash build.sh init` 会自动切到官方源并重试一次安装。
- 如果宿主机没有 root，请使用有 `sudo` 权限的账号执行 `bash build.sh init`。
- `main` 服务 **`privileged: true`**。
- 如果是普通用户运行 rootless `podman compose`，`bash build.sh up` 会自动将 `net.ipv4.ip_unprivileged_port_start` 调整为 `80`，保证容器内 Nginx 能绑定宿主机 `80` 端口。
- 宿主机 **80 端口未被占用**；如果开启 HTTPS，**443 端口也必须空闲**（`build.sh` 会自动检测并停用宿主机 nginx）。

## 一分钟部署

```bash
cd deploy/prod
bash build.sh init     # 已有发布镜像就改成: bash build.sh init --image
# 编辑 .env，只看 CANONICAL_BASE_URL 和 TLS_MODE
bash build.sh up
bash build.sh verify
```

默认数据目录固定是 `/data/ai-agent-os`。  
`build.sh init` 负责准备 `.env`、中间件镜像、`MAIN_IMAGE`、`APP_BASE_IMAGE`；`build.sh up` 只负责启动。  
如果宿主机缺少 `podman` / `podman compose`，`build.sh init` 会先尝试自动安装。  
如果 `/data/ai-agent-os` 还不存在，`build.sh init` 会按需用 `sudo` 自动创建。  
想看显式预检，再手动执行 `bash build.sh doctor`。

最小 `.env`：

```env
CANONICAL_BASE_URL="http://your-ip-or-domain"
TLS_MODE="http"
```

如果前面已有 LB / CDN 做 HTTPS：

```env
CANONICAL_BASE_URL="https://your-domain"
TLS_MODE="external"
```

如果容器自己做 HTTPS：

```env
CANONICAL_BASE_URL="https://your-domain"
TLS_MODE="redirect"
TLS_CERTS_HOST_DIR="./certs"
TLS_CERT_FILE="/app/tls/fullchain.pem"
TLS_KEY_FILE="/app/tls/privkey.pem"
```

改配置：

- 只改运行参数：直接 `bash build.sh up`
- 改了 `MAIN_IMAGE` / `APP_BASE_IMAGE`：重新执行 `bash build.sh init` 或 `bash build.sh init --image`

## HTTP / HTTPS 模式

默认模式是 **HTTP**，只填 `CANONICAL_BASE_URL` 就能启动。

```env
CANONICAL_BASE_URL="http://your-domain-or-ip"
TLS_MODE="http"
```

如需启用容器内 HTTPS，补齐证书目录并切到对应模式：

```env
CANONICAL_BASE_URL="https://your-domain"
TLS_MODE="redirect"
TLS_CERTS_HOST_DIR="./certs"
TLS_CERT_FILE="/app/tls/fullchain.pem"
TLS_KEY_FILE="/app/tls/privkey.pem"
```

四种模式：

- `TLS_MODE=http`：仅监听 `80`，保持当前 HTTP 部署方式。
- `TLS_MODE=https`：同时监听 `80` 和 `443`，HTTP / HTTPS 都可访问。
- `TLS_MODE=redirect`：`80` 统一 `301 -> 443`，`443` 提供 HTTPS。
- `TLS_MODE=external`：容器只跑 `80`，由外部 LB / CDN / 网关终止 TLS。

证书路径说明：

- `TLS_CERTS_HOST_DIR` 是宿主机证书目录，会整体挂载到容器内 `/app/tls`。
- `TLS_CERTS_HOST_DIR` 支持绝对路径；如果写相对路径，则相对 `deploy/prod/` 目录解析。
- `TLS_CERT_FILE` / `TLS_KEY_FILE` 必须是容器内路径，且位于 `/app/tls/` 下。
- `build.sh` 只会在 `TLS_MODE=https` / `redirect` 时检查证书文件是否真实存在；缺文件直接失败，不会带着坏配置启动。

如果你是云厂商 LB / CDN 做 TLS 终止，直接使用 `TLS_MODE=external` 即可；这时 `CANONICAL_BASE_URL` 通常写成 `https://`。

生产配置说明：

- 版本库中的官方模板源在 `deploy/prod/config/template/`
- 容器启动后会渲染到 `deploy/prod/config/runtime/`
- 定时任务固定由独立 `scheduler` 容器执行；`main` 不再内嵌调度器

> 构建 Go 依赖默认使用 `GOPROXY=https://goproxy.cn,direct` 与 `GOSUMDB=sum.golang.google.cn`；如需覆盖，可在构建时传 `--build-arg GOPROXY=... --build-arg GOSUMDB=...`。

## 常用命令

```bash
bash build.sh init [--image]   # 初始化 .env，并准备中间件 / MAIN_IMAGE / APP_BASE_IMAGE
bash build.sh up               # 启动已初始化服务（默认，不再构建镜像）
bash build.sh update [--image] # 更新 main / scheduler / backup；加 --image 时直接拉取 MAIN_IMAGE
bash build.sh verify           # 执行生产健康检查（mysql / main / scheduler / backup）
bash build.sh doctor           # 显式预检；主路径可省略
bash build.sh pull-update   # git pull --ff-only 后执行 update
bash build.sh restart-main  # 仅重启 main，不重建镜像
bash build.sh restart-scheduler  # 仅重启 scheduler，不重建镜像
bash build.sh build-app-base --no-cache  # 在与 main 相同的运行环境里重建 APP_BASE_IMAGE
bash build.sh logs main     # 查看 main 日志
bash build.sh status        # 查看服务状态
bash build.sh down          # 停止服务（保留数据卷）
```

推荐升级路径：

- 首次机器准备：`bash build.sh init`
- 已发布固定镜像的首次机器准备：`bash build.sh init --image`
- 只升级主站代码：`bash build.sh update`
- 已发布固定镜像更新：`bash build.sh update --image`
- 先拉最新代码再升级：`bash build.sh pull-update`
- 只改 `.env` 或想让主进程重启生效：`bash build.sh restart-main`
- 只想重启独立调度器：`bash build.sh restart-scheduler`
- 部署完成后补一轮健康检查：`bash build.sh verify`
- 只想排查用户应用基础镜像：`bash build.sh build-app-base --no-cache`
- 想在启动前看显式预检报告：`bash build.sh doctor`

## 安全默认值

- 生产模板里的内部 Go 服务现在默认监听 **`127.0.0.1`**，不再直接绑宿主机所有网卡。
- 生产模板里的 **`pprof`** 默认关闭；如需临时排障，必须显式在对应服务配置里把 `enable_pprof` 打开。
- 公网入口仍然只应通过容器内 Nginx 暴露；生产 Nginx 不再转发 `/swagger/`。
- `build.sh` 现在会校验：`CANONICAL_BASE_URL` 必须以 `http://` 或 `https://` 开头、`JWT_SECRET` 至少 32 字符、`CONTROL_ENC_KEY` 必须正好 32 字符、`BACKUP_BASIC_AUTH_PASSWORD` 至少 16 字符。
- 基础设施镜像（MySQL / NATS / MinIO）已经固定写死；普通部署只在确实需要时才覆盖 `MAIN_IMAGE` / `APP_BASE_IMAGE` 或 SMTP 行为。
- 如果 `TLS_MODE=https` / `redirect`，`build.sh` 还会校验证书路径、宿主机证书文件和 `80/443` 端口占用；其中 `TLS_MODE=redirect` 还要求 `CANONICAL_BASE_URL` 必须是 `https://`。

## 构建加速（依赖与源，默认偏国内）

`Dockerfile` 已内置（可按需关闭或覆盖）：

| 环节 | 默认行为 | 关闭 / 覆盖 |
|------|----------|-------------|
| **Debian APT**（Go 阶段、运行阶段） | `APT_USE_MIRROR=1` 时把官方源换成 **阿里云**（**`http://`**，避免 `bookworm-slim` 首包 `ca-certificates` 未就绪时 HTTPS 校验失败） | 海外构建：`--build-arg APT_USE_MIRROR=0` |
| **Go 模块** | `GOPROXY` / `GOSUMDB` 见上文 | `--build-arg GOPROXY=direct` 等 |
| **npm**（前端 `npm ci`） | `NPM_REGISTRY=https://registry.npmmirror.com` | 官方源：`--build-arg NPM_REGISTRY=https://registry.npmjs.org` |

示例（仅重建 `main`）：

```bash
podman compose build --build-arg APT_USE_MIRROR=0 --build-arg NPM_REGISTRY=https://registry.npmjs.org main
```

**说明**：拉取 **`golang` / `node` / `debian` 层镜像**仍走宿主机配置的容器 registry（可在 **`/etc/containers/registries.conf.d/`** 为 `docker.io` 配镜像加速，与 Dockerfile 无关）。

**容器内 `podman build` 拉 `ubuntu:22.04`（docker.io）超时**：胖镜像默认安装 **`deploy/prod/containers/registries.conf.d/000-docker-io-mirror.conf`**，为 `docker.io` 配置 **DaoCloud 镜像**（`docker.m.daocloud.io`）。海外构建：`podman compose build --build-arg USE_CN_REGISTRY_MIRROR=0 main`。

## 存储与公网地址

- **`CANONICAL_BASE_URL`**（写在 `.env`）为唯一主站真值。
- `cdn_domain` 空时由进程用该 URL 补全；Nginx **`www` → 裸域 301** 与真值 scheme 一致。

### 持久卷（勿误删）

生产环境统一使用 **`/data/ai-agent-os` 宿主机固定目录挂载**，避免核心数据落在容器层。

| 宿主机目录 | 容器挂载点 | 用途 |
|------|--------|------|
| `/data/ai-agent-os/mysql` | `/var/lib/mysql` | MySQL 数据目录 |
| `/data/ai-agent-os/minio` | `/data` | MinIO 数据目录 |
| `/data/ai-agent-os/namespace` | `/app/namespace` | **用户应用空间**（`namespace/{user}/{app}/...` 等工作区；`app-runtime` 默认固定使用这里作为工作区根目录） |
| `/data/ai-agent-os/data` | `/app/data` | 应用侧其他本地数据目录（当前已用于 `app-runtime` SQLite、License、backup repo/state/tmp） |
| `/data/ai-agent-os/logs` | `/app/logs` | 主站与 backup-service 日志 |
| `/data/ai-agent-os/podman_storage` | `/var/lib/containers` | 容器内 Podman 存储 |

`build.sh` 会自动创建以下目录结构：

- `/data/ai-agent-os/mysql`
- `/data/ai-agent-os/minio`
- `/data/ai-agent-os/podman_storage`
- `/data/ai-agent-os/logs`
- `/data/ai-agent-os/namespace`
- `/data/ai-agent-os/data`

`/app/data` 当前推荐子目录：

- `/app/data/runtime/app-runtime/app_runtime.db`
- `/app/data/license/license.json`
- `/app/data/license/license.key`
- `/app/data/backup/repo`
- `/app/data/backup/state`
- `/app/data/backup/staging`
- `/app/data/tmp`

**备份 / 升级**：

- 直接备份 `/data/ai-agent-os` 下对应目录即可。
- 切勿对 `/data/ai-agent-os` 做 `rm -rf` 或未经验证的清理。
- `backup` 服务默认监听 `127.0.0.1:19088`，作为后续恢复控制面的独立入口。

### Backup 控制面

当前 `backup-service` 已经是独立控制面，不依赖主站 MySQL：

- 本地控制台：`http://127.0.0.1:19088/backup`
- 服务说明：见 `core/backup-service/README.md`
- 值班清单：见 `RECOVERY_CHECKLIST.md`
- 恢复操作手册：见 `RECOVERY.md`
- 健康检查：`GET /health`
- 状态总览：`GET /backup/api/v1/status`
- 最近任务：`GET /backup/api/v1/tasks`
- 单任务详情：`GET /backup/api/v1/tasks/:id`
- 快照列表：`GET /backup/api/v1/snapshots?resource_type=namespace`
- 单快照详情：`GET /backup/api/v1/snapshots/:id`
- 手工预检：`POST /backup/api/v1/precheck`
- 维护模式：`POST /backup/api/v1/maintenance`
- 创建 namespace 快照：`POST /backup/api/v1/namespace/snapshots`
- 基于快照恢复 namespace：`POST /backup/api/v1/namespace/restore`
- 创建 MySQL 快照：`POST /backup/api/v1/mysql/snapshots`
- 基于快照恢复 MySQL：`POST /backup/api/v1/mysql/restore`
- 创建 MinIO 快照：`POST /backup/api/v1/minio/snapshots`
- 基于快照恢复 MinIO：`POST /backup/api/v1/minio/restore`

当前阶段已落地：

- SQLite 状态库存放在 `/app/data/backup/state/backup-service.db`
- 维护模式状态持久化，并同步生成 `/app/data/backup/state/maintenance.flag` 与 `/app/data/backup/state/maintenance.html`
- 预检任务持久化与历史查询
- `namespace` 本地快照归档与快照列表查询
- `namespace` 基于快照的恢复能力，恢复前自动创建 `pre-restore` 快照
- `MySQL` 逻辑全量快照归档与快照列表查询
- `MySQL` 基于快照的整库恢复能力，恢复前自动创建 `pre-restore` dump
- `MinIO` 对象级快照归档与快照列表查询
- `MinIO` 基于快照的整仓恢复能力，恢复前自动创建 `pre-restore` 对象快照
- 对 `namespace / data / mysql / minio / podman_storage` 的路径、MySQL/MinIO 连通性和工具链预检

注意：

- `namespace` / `MySQL` / `MinIO` 恢复目前都要求先开启维护模式。
- 开启维护模式后，主站 Nginx 会自动对外返回 `503` 维护页，入口页内容来自 `/app/data/backup/state/maintenance.html`。
- 如果你是在宿主机本地直接启动 `backup-service`，没有手工导出环境变量也没关系；它会优先读取 `deploy/prod/.env` 里的 `MYSQL_ROOT_PASSWORD / MINIO_ROOT_PASSWORD / BACKUP_BASIC_AUTH_PASSWORD`，并把容器内 `/app/...` 路径自动映射到宿主机 `/data/ai-agent-os`。
- 当前 `namespace` 快照仓库存放在 `/app/data/backup/repo/namespace/`。
- 当前 `MySQL` 快照仓库存放在 `/app/data/backup/repo/mysql/`。
- 当前 `MinIO` 快照仓库存放在 `/app/data/backup/repo/minio/`。

后续再在此基础上接真正的备份执行器与恢复编排。

## 文件说明

| 文件 | 说明 |
|------|------|
| **`.env.example`** | 配置模板；复制为 **`.env`** 后填写 |
| **`.env`** | 唯一配置源；Compose 与 `build.sh` 直接读取（**勿提交**） |
| **`build.sh`** | 运维入口：外部命令保持不变，内部通过 `scripts/*.sh` 分拆校验、健康检查和命令实现 |
| `scripts/` | `build.sh` 的内部实现拆分：`lib / validate / health / commands` |
| `RECOVERY_CHECKLIST.md` | 值班恢复速查清单 |
| `RECOVERY.md` | 误删数据后的恢复操作手册 |
| `docker-compose.yaml` | 服务定义（main 使用 host 网络） |
| `../base/infra/mysql/init-db.sql` | MySQL 首次启动建库的 canonical SQL（由 `docker-compose.yaml` 挂载） |
| `../base/infra/nats/nats-server.conf` | NATS canonical 配置（由 `docker-compose.yaml` 挂载） |
| `Dockerfile` / `entrypoint-common.sh` / `entrypoint-main.sh` / `entrypoint-scheduler.sh` / `entrypoint-backup.sh` / `nginx/` / `config/template/` | 镜像与内置模板 |

补充：

- `.env` 中的 `MAIN_IMAGE` 只是主镜像 tag；`init` 会在目标机构建它，`init --image` 则会直接拉取它。
- `APP_BASE_IMAGE` 不再由 `main` 启动时隐式构建；现在由 `init` / `init --image` 显式准备，`up` 只负责启动。
- Compose 文件名实际为 **`docker-compose.yaml`**，不是 `docker-compose.yml`。

## 升级

```bash
bash build.sh update
```

## 构建说明（避免踩坑）

- 用户应用基础镜像的 canonical 构建资源已迁到 **`deploy/base/images/app-base/`**；若本地构建报启动脚本找不到，请先确认 **`deploy/base/images/app-base/start.sh`** 存在且最新。
- **Podman `runroot must be set`**：镜像内 **`/etc/containers/storage.conf`** 已写 `runroot` / `graphroot`；**`entrypoint-main.sh`** 会创建 **`/run/containers/storage`**。生产 Compose 预检走 **Compose 路径**由镜像内空标记文件 **`/etc/ai-agent-os/prod-compose-bundle`** 自动识别，**线上无需为此设环境变量**。
- **本机开发**：`APP_ENV=dev` 时启动预检默认按本地 compose 基础设施处理，无需额外 `AI_AGENT_OS_DEV_SKIP_EMBEDDING_INFRA`。

## 故障排查

### `panic: nats: no servers available for connection`

`app_db` 里 **`nats` 表** 的 **`host`** 须能被 `main` 容器解析。由于 `main` 使用 `network_mode: host`，中间件端口通过 `127.0.0.1` 暴露，进程会在连接前把仍为 **`localhost` / `127.0.0.1`** 的行自动改为 **`nats`**（Compose 服务名），**无需额外配置环境变量**。

**本机开发**：连本机 NATS 时请设 **`APP_ENV=dev`**（与 `deploy/dev/config` 约定一致）。

若仍异常，可手工：

```sql
USE app_db;
UPDATE nats SET host = 'nats' WHERE host IN ('localhost', '127.0.0.1');
```

### 外网无法访问

`main` 容器使用 `network_mode: host`，容器内 Nginx 直接绑定宿主机 80 端口；开启 HTTPS 后也会绑定 443，不依赖 Podman/Docker 的端口映射。如果仍无法访问：

1. 确认宿主机 80 端口无其他进程占用：`ss -tlnp | grep :80`
2. 如果开启 HTTPS，再检查 443：`ss -tlnp | grep :443`
3. 确认云平台安全组 / 防火墙允许 TCP 80/443 入站
4. `build.sh` 已自动停用宿主机 systemd nginx，若手动启动了其他 HTTP / HTTPS 服务请先关闭

## 已知限制

- 如果你挂的是 Let’s Encrypt `live/` 目录，里面通常包含符号链接；请确认挂载目录内能解析到真实证书文件，否则容器内会校验失败。
- `www` 到裸域的 HTTPS 跳转会复用同一张证书；如果证书不覆盖 `www`，浏览器会在握手阶段先报证书不匹配。
