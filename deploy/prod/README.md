# 生产单机部署

> 官方生产入口：`deploy/prod/`

AI-Agent-OS 的生产部署由 `aosctl` 统一控制。Compose 仍是底层容器执行器，但用户不需要手工维护生成后的 Compose 文件。

快速入口：

- [QUICK_START.md](QUICK_START.md)
- [DEPLOY_TUTORIAL.md](DEPLOY_TUTORIAL.md)

**范围**：主站、中间件（MySQL / NATS / MinIO）、内置 Nginx、容器内 Podman 用户应用运行时。不包含企业 License、控制面、消息中心或备份控制面。

## 部署分层

完整定义见 [部署分层模型](../../docs/deployment-layers.md)。

```mermaid
flowchart TD
  L0["L0 部署控制层<br/>aosctl / Compose / 配置生成"]
  L1["L1 基础设施层<br/>MySQL / NATS / MinIO / 数据目录"]
  L2["L2 入口接入层<br/>Nginx / TLS / 静态前端 / API 反代"]
  L3["L3 平台服务层<br/>core-server"]
  L4["L4 运行时管理层<br/>app-runtime / Podman API / app-base / namespace"]
  L5["L5 用户应用层<br/>用户 App 容器 / SDK / 业务代码"]

  L0 --> L1
  L0 --> L2
  L0 --> L3
  L0 --> L4
  L2 --> L3
  L3 --> L1
  L3 --> L4
  L4 --> L5
```

当前物理拓扑：

```text
Compose
  ├─ mysql / nats / minio（bundled 时启用）
  └─ main 容器（network_mode: host）
       ├─ Nginx :80 / :443
       ├─ core-server
       └─ Podman API
```

## 前置条件

- Linux 宿主机。
- 已安装 `podman compose` 或 `docker compose`。
- `/data` 可写；`aosctl up` 会创建 `/data/kageos`。
- `main` 服务需要 `privileged: true`。
- 宿主机 80 端口未被占用；如果启用容器内 HTTPS，443 端口也必须空闲。

## 一分钟部署

```bash
go run ./cmd/aosctl bootstrap --base-url http://your-ip-or-domain
```

最小 `aos.yaml`：

```yaml
site:
  base_url: "http://your-ip-or-domain"
  tls_mode: "http"
```

前面已有 LB / CDN 做 HTTPS：

```yaml
site:
  base_url: "https://your-domain"
  tls_mode: "external"
```

容器自己做 HTTPS：

```yaml
site:
  base_url: "https://your-domain"
  tls_mode: "redirect"
  certs_host_dir: "./certs"
  cert_file: "/app/tls/fullchain.pem"
  key_file: "/app/tls/privkey.pem"
```

## 常用命令

```bash
go run ./cmd/aosctl init --base-url http://your-ip-or-domain
go run ./cmd/aosctl doctor --config deploy/prod/aos.yaml
go run ./cmd/aosctl up --config deploy/prod/aos.yaml
go run ./cmd/aosctl verify --config deploy/prod/aos.yaml
go run ./cmd/aosctl status --config deploy/prod/aos.yaml
go run ./cmd/aosctl logs --config deploy/prod/aos.yaml --layer L3
go run ./cmd/aosctl down --config deploy/prod/aos.yaml
go run ./cmd/aosctl uninstall --config deploy/prod/aos.yaml --dry-run
go run ./cmd/aosctl uninstall --config deploy/prod/aos.yaml --purge-data --force
```

后台启动脚本：

```bash
./prod-up.sh
tail -f deploy/prod/aosctl-up.log
./prod-stop.sh
```

## 安全默认值

- 内部 Go 服务默认监听 `127.0.0.1`。
- 生产 `pprof` 默认关闭。
- 公网入口只通过容器内 Nginx 暴露；生产 Nginx 不转发 `/swagger/`。
- `deploy/prod/aos.yaml` 和 `deploy/prod/.generated/` 是本机敏感配置/生成物，默认不应入库。
- NATS 生产默认开启账号密码认证；`aosctl init` 会自动生成 `nats.password`。
- 如果 `site.tls_mode=redirect`，`site.base_url` 必须是 `https://`。

## 持久卷

生产环境统一使用 `/data/kageos` 宿主机固定目录挂载。

| 宿主机目录 | 容器挂载点 | 用途 |
|------|--------|------|
| `/data/kageos/mysql` | `/var/lib/mysql` | MySQL 数据目录 |
| `/data/kageos/minio` | `/data` | MinIO 数据目录 |
| `/data/kageos/namespace` | `/app/namespace` | 用户应用空间 |
| `/data/kageos/data` | `/app/data` | 应用侧本地数据目录 |
| `/data/kageos/logs` | `/app/logs` | 平台日志 |
| `/data/kageos/podman_storage` | `/var/lib/containers` | 容器内 Podman 存储 |

备份时直接备份 `/data/kageos` 下对应目录。不要对该目录做未经验证的清理。

## 文件说明

| 文件 | 说明 |
|------|------|
| `cmd/aosctl` | Go 部署控制入口，负责生成配置、调用 Compose、启停和验证 |
| `aos.example.yaml` | 无密钥示例配置 |
| `aos.yaml` | 本机私有配置，包含密钥，默认不入库 |
| `.generated/` | `aosctl` 渲染出的 Compose、配置和中间件文件，默认不入库 |
| `Dockerfile` / `entrypoint-common.sh` / `entrypoint-main.sh` / `nginx/` / `config/template/` | 镜像与内置模板 |

## 升级

```bash
git pull --ff-only
go run ./cmd/aosctl up --config deploy/prod/aos.yaml
go run ./cmd/aosctl verify --config deploy/prod/aos.yaml
```
