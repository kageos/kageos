# Embedding 部署（一机一实例 · Linux + Podman）

**目标**：在一台 Linux 上跑 **一套** AI Agent OS：后端 **裸机进程**，依赖 **容器** 只承载 MySQL / NATS / MinIO。

| 组件 | 形态 |
|------|------|
| **core-server** | 单进程，内嵌 gateway、app-server、agent-server、app-storage、app-runtime、hr、control（见 `core/cmd/main/main.go`） |
| **hub-server** | 单独二进制 |
| **MySQL / NATS / MinIO** | `podman compose` + 根目录 `docker-compose.dev.yml` |
| **用户应用** | 由 **app-runtime** 调本机 **Podman** 起容器（需 **podman socket**） |

---

## 环境要求

- Linux x86_64 / arm64（与 Go 构建目标一致）
- **Go 1.24+**（与 `go.mod` 一致）、**Podman 4+**（含 `podman compose`）
- **`podman-compose` 包（强烈建议）**：`podman compose` 在检测到本机有 `docker-compose` 时会**优先调用后者**，镜像由 Docker 拉取，国内常直连 Docker Hub 超时；`embedding.sh` 会尽量设置 `PODMAN_COMPOSE_PROVIDER` 指向 `podman-compose`，使拉镜像走 Podman（可用 `/etc/containers/registries.conf.d/` 镜像加速）。安装示例：`sudo apt-get install -y podman-compose`
- **`init` 全量部署**还需 **Node.js / npm**（构建 Web + Hub 前端）、**Nginx**（脚本可 `apt` 安装）、**sudo**（写 `/etc/nginx`、reload）
- 构建 core-server 需本机 **CGO 依赖**（如 `libgpgme-dev` 等；Debian/Ubuntu 上 `embedding.sh build` 会尝试自动安装）
- 内存建议 **≥ 8GB**，磁盘 **≥ 40GB**（含镜像与数据）

---

## 端口约定（默认）

| 服务 | 端口 |
|------|------|
| API Gateway | **9090** |
| MinIO API | **9000** |
| MinIO 控制台 | **9001** |
| NATS | **4222** |
| MySQL | **3306** |
| Hub HTTP | **9094**（见 `deploy/config/prod/hub.yaml`） |

防火墙按需放行；**公网访问**时务必改默认密码与 JWT 密钥。

---

## 首次部署（命令顺序）

在 **仓库根目录** 克隆代码后执行（路径含空格请自行加引号）：

### 推荐：`init`（对齐 `deploy/server-deploy.sh init`）

含：**中间件** → **local 覆盖（若有）** → **建库** → **Go 编译** → **前后端构建** → **Nginx** → **Podman socket** → **运行时镜像** → **后台启动 core-server / hub-server**（日志在 `./logs/`，PID 在 `./run/`）。

```bash
cd /path/to/ai-agent-os
bash deploy/embedding/scripts/embedding.sh init
```

### 仅栈、不启 Nginx/前端/进程：`all`

```bash
bash deploy/embedding/scripts/embedding.sh all
# = infra → local（若有 deploy/config/local/）→ dbs → build → runtime
```

### 分步（便于排错）

```bash
bash deploy/embedding/scripts/embedding.sh infra     # MySQL / NATS / MinIO
bash deploy/embedding/scripts/embedding.sh local    # 可选：local/*.yaml → prod/
bash deploy/embedding/scripts/embedding.sh dbs        # MySQL 库
bash deploy/embedding/scripts/embedding.sh build      # bin/core-server、hub-server
bash deploy/embedding/scripts/embedding.sh frontend   # Web + Hub 静态资源
bash deploy/embedding/scripts/embedding.sh nginx      # 安装/更新站点并 reload（sudo）
bash deploy/embedding/scripts/embedding.sh runtime    # 镜像 ai-agent-os:latest
# 再手动 ./bin/core-server & ./bin/hub-server，或用: embedding.sh restart
```

### 启动 Podman API（app-runtime 必需）

`init` / `update` / `restart` 内会尝试启动 **`/run/podman/podman.sock`**（若尚不存在）。也可手动：

```bash
sudo podman system service --time=0 unix:///run/podman/podman.sock &
```

生产建议使用 **systemd** 常驻（见 Podman 文档 *podman.socket*）。

### 手动启动进程（未用 `init` 时）

```bash
export AI_AGENT_OS_ROOT=/绝对路径/ai-agent-os   # 可选
./bin/core-server
./bin/hub-server
```

---

## 配置说明

- **默认**：**不设置 `APP_ENV` 即使用 prod**，读取 **`deploy/config/prod/*.yaml`**（已按本机 `127.0.0.1` / `localhost` 连中间件）；仅开发时设 **`APP_ENV=dev`**。
- **根目录**：优先 **`AI_AGENT_OS_ROOT`**，否则向上查找 **`.ai-agent-os-root` / `deploy/config` / `go.mod`** 等。详见 **[deploy/config/README.md](../config/README.md)**。
- **按机覆盖、不进 Git**：将 yaml 放入 **`deploy/config/local/`**，执行 **`bash deploy/embedding/scripts/embedding.sh local`**；说明见 [embedding/config/README.md](config/README.md)。
- **必改项（公网）**：至少 **`app-storage` 的 `cdn_domain`**、**`hub.yaml` 的 `os.base_url`**，以及 **JWT / 数据库 / MinIO / 系统账号** 等密钥类字段。

---

## 与 `deploy/config/prod` 里 `sdk.*` 的说明

用户应用容器访问宿主机 Gateway/NATS 时，配置里可能为 **`host.containers.internal`**（Podman/Docker 场景）。若在你环境中不解析，请在本机 **`deploy/config/local/global.yaml`** 中改为实际可达地址（如宿主机内网 IP），再执行 **`embedding.sh local`**。

---

## Nginx 配置

模板与说明与 Embedding 放同一目录：**[nginx/nginx-server.conf](nginx/nginx-server.conf)**、**[nginx/README.md](nginx/README.md)**。  
**可选域名反代**（`geeleo.com` → 8999、`hub.*` → 8998）：见 **[nginx/DOMAIN_PROXY.md](nginx/DOMAIN_PROXY.md)** 与示例 **[nginx/nginx-domain-proxy.example.conf](nginx/nginx-domain-proxy.example.conf)**（DNS 未指过来时可不启用）。

**静态目录**：`embedding.sh nginx` / `init` 会把 `web/dist`、`hub-frontend/dist` **rsync 到 `/opt/ai-agent-os/`** 再让 Nginx 读（避免仓库在 **`/root`** 时 `www-data` 无权限 → **500**）。更新前端后请再执行 **`embedding.sh nginx`** 或跑 **`update`**（已内含 nginx 步骤）。

**可选域名（如 geeleo.com → 8999、hub.geeleo.com → 8998）**：仅当 DNS 指向本机时启用，默认可不装；见 **[nginx/DOMAIN_PROXY.md](nginx/DOMAIN_PROXY.md)** 与 **`nginx/nginx-domain-proxy.example.conf`**。

---

## 脚本

**`deploy/embedding/scripts/embedding.sh`** 单入口（逻辑参考 **`deploy/server-deploy.sh`**，中间件为 Podman）：

| 命令 | 作用 |
|------|------|
| **`init`** | 首次完整上线：栈 + 前后端 + Nginx + 运行时镜像 + 启动 core/hub |
| **`update`** | **`git pull`** 后 **与 `init` 同序全量**：`infra` → `local`（若有）→ `dbs` → `build` → `frontend` → **`nginx`** → **`runtime`** → 重启 core/hub（日常只跑这一条即可） |
| `restart` / `stop` / `status` / `logs` | 与 `server-deploy.sh` 同类（`logs` 默认 `core-server`，可跟 `hub-server`） |
| `all` | 仅：`infra` → `local`（若有）→ `dbs` → `build` → `runtime` |
| `infra` | Podman Compose 起 MySQL/NATS/MinIO |
| `local` | `deploy/config/local/*.yaml` → `deploy/config/prod/` |
| `dbs` | 幂等创建 MySQL 库 |
| `build` | 编译 `bin/core-server`、`bin/hub-server` |
| `frontend` | 构建 Web + Hub（`npm`） |
| `nginx` | 部署 `deploy/embedding/nginx/nginx-server.conf` 并 reload |
| `runtime` | `podman build` → `ai-agent-os:latest`（`build/Dockerfile`） |

`bash deploy/embedding/scripts/embedding.sh --help` 查看用法。

---

## 升级代码

```bash
bash deploy/embedding/scripts/embedding.sh update
```

（仓库须为 git clone；若仍用 `deploy/config/local/`，改完 yaml 后另执行 `embedding.sh local` 再 `restart`。）

---

更多部署索引见 **[../README.md](../README.md)**。
