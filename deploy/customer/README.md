# 客户主站一键部署（Compose：宿主机 Docker 或 Podman）

**范围**：主站 + 中间件（MySQL / NATS / MinIO）+ 内置 Nginx(8080) + **容器内 Podman**（跑用户应用）。**不包含 Hub**。

## 配置约定（只认用户手写的 env.yaml）

1. **`env.yaml` 必须由用户手写**（从 **`env.yaml.example`** 复制改名后填写）。
2. **不要求、不期望用户编辑 `.env`**：`.env` 仅由 **`./render-env.sh`** 从 `env.yaml` **自动生成**，供 Docker/Podman Compose 读取。
3. **`render-env.sh` 无 Python**：宿主机只需 **`bash` + `awk`**（一般系统自带）。

## 前置

- Podman 4+（`podman compose`）或 Docker（`docker compose`）。
- `main` 服务 **`privileged: true`**。

## 快速开始

```bash
cd deploy/customer
cp env.yaml.example env.yaml
# 手写填写 env.yaml（全部必填项见 example 注释；smtp_password 可留空）

chmod +x render-env.sh
./render-env.sh

podman compose up -d --build
# Docker：docker compose up -d --build
```

> 构建 Go 依赖默认使用 `GOPROXY=https://goproxy.cn,direct` 与 `GOSUMDB=sum.golang.google.cn`；如需覆盖，可在构建时传 `--build-arg GOPROXY=... --build-arg GOSUMDB=...`。

改配置：只改 **`env.yaml`**，再执行 **`./render-env.sh`**，然后按需 **`compose up -d`**。

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

**容器内 `podman build` 拉 `ubuntu:22.04`（docker.io）超时**：胖镜像默认安装 **`deploy/customer/containers/registries.conf.d/000-docker-io-mirror.conf`**，为 `docker.io` 配置 **DaoCloud 镜像**（`docker.m.daocloud.io`）。海外构建：`podman compose build --build-arg USE_CN_REGISTRY_MIRROR=0 main`。爱你呦。

## 存储与公网地址

- **`CANONICAL_BASE_URL`**（写在 `env.yaml` → 生成进 `.env`）为唯一主站真值。
- `cdn_domain` 空时由进程用该 URL 补全；Nginx **`www` → 裸域 301** 与真值 scheme 一致。

### 持久卷（勿误删）

Compose 为 **`main`** 挂载命名卷，避免重建容器后丢失数据：

| 卷名 | 挂载点 | 用途 |
|------|--------|------|
| `namespace_data` | `/app/namespace` | **用户应用空间**（`namespace/{user}/{app}/...` 等工作区，与配置里 `app_dir.base_path: namespace` 对应） |
| `app_data` | `/app/data` | 应用侧其他本地数据目录（与镜像内约定一致） |
| `app_logs` | `/app/logs` | 主站日志 |
| `podman_storage` | `/var/lib/containers` | 容器内 Podman 存储 |

**备份 / 升级**：不要用 **`compose down -v`**，否则会删掉上述命名卷。**`docker volume`** / **`podman volume`** 需单独备份 `*_data`、`*_logs`、`podman_storage` 等卷（按运维策略做快照或 `volume inspect` 后拷宿主机路径）。

## 文件说明

| 文件 | 说明 |
|------|------|
| **`env.yaml.example`** | 结构模板；用户复制为 **`env.yaml`** 并手写 |
| **`env.yaml`** | 用户真实配置（**勿提交**，见 `.gitignore`） |
| **`render-env.sh`** | `env.yaml` → **`.env`**（校验必填，无 Python） |
| **`.env`** | Compose 读取（**勿手改**、勿提交） |
| `docker-compose.yaml` | 服务定义 |
| `init-db.sql` | MySQL 首次启动建库（挂载 `docker-entrypoint-initdb.d`，**仅本目录**） |
| `nats-server.conf` | NATS 容器配置（**仅本目录**） |
| `Dockerfile` / `entrypoint-main.sh` / `nginx/` / `config/prod/` | 镜像与内置模板 |

## 升级

`podman compose build main && podman compose up -d main`

## 构建说明（避免踩坑）

- 用户应用基础镜像上下文里的 **`scripts/start.sh`** 来自仓库根目录的 **`build/start.sh`**（胖镜像构建时复制到 `/app/app-base/scripts/`）。若本地构建报 `scripts/start.sh` / `start.sh` 找不到，先 **`git pull`** 确保 Dockerfile 与 `build/start.sh` 一致。
- **Podman `runroot must be set`**：镜像内 **`/etc/containers/storage.conf`** 已写 `runroot` / `graphroot`；**`entrypoint-main.sh`** 会创建 **`/run/containers/storage`**。客户预检走 **Compose 路径**由镜像内空标记文件 **`/etc/ai-agent-os/customer-compose-bundle`** 自动识别，**线上无需为此设环境变量**。
- **开发特殊**：本机中间件已由 compose 提供、不想走 `podman start mysql8` 那套时，可设 **`AI_AGENT_OS_DEV_SKIP_EMBEDDING_INFRA=1`**（仅 dev 使用）。

## 已知限制

- 边缘为容器内 **8080 HTTP**；对外 HTTPS 前加 LB 并同步 **`CANONICAL_BASE_URL`** 为 `https://`。

## 故障排查

### `panic: nats: no servers available for connection`

`app_db` 里 **`nats` 表** 的 **`host`** 须能被 `main` 容器解析。交付/线上（**未设 `APP_ENV=dev`**）时，进程会在连接前把仍为 **`localhost` / `127.0.0.1`** 的行自动改为 **`nats`**（与 Compose 服务名一致），**无需在 Compose 里再配 NATS 相关环境变量**。

**本机开发**：连本机 NATS 时请设 **`APP_ENV=dev`**（与 `deploy/config/dev` 约定一致）。若在开发环境用 Compose 里的 NATS 服务而非本机端口，可显式设 **`NATS_SEED_HOST=nats`**。

若仍异常，可手工：

```sql
USE app_db;
UPDATE nats SET host = 'nats' WHERE host IN ('localhost', '127.0.0.1');
```
