# 客户主站一键部署（Compose：宿主机 Docker 或 Podman）

**范围**：主站 + 中间件（MySQL / NATS / MinIO）+ 内置 Nginx(8080) + **容器内 Podman**（跑用户应用）。**不包含 Hub**（Hub 为公司自建单独运维）。

## 前置

- **宿主机 Podman（你们现状）**：安装 **Podman 4+**，并具备 **`podman compose`**（推荐）或 **`podman-compose`**；镜像加速见 `/etc/containers/registries.conf.d/`。若 `podman compose` 误用 `docker-compose` 拉镜像，请安装 **`podman-compose`** 并设 **`PODMAN_COMPOSE_PROVIDER`**（与 `embedding.sh` 说明一致）。
- **或宿主机 Docker**：Docker Engine + `docker compose`。
- **无需 Python**：环境只用 **`.env`**（shell 校验），不依赖 `pip` / PyYAML。
- 本机构建需：CPU/内存足够（Go + Node 构建、容器内首次 `podman build ai-agent-os:latest` 较慢）。
- `main` 服务需 **`privileged: true`**（嵌套 Podman）。

## 快速开始

```bash
cd deploy/customer
cp .env.example .env
# 编辑 .env：逐项必填（无默认值）；SMTP 不用可留空 SMTP_PASSWORD=

chmod +x check-env.sh
./check-env.sh

# 宿主机 Podman：
podman compose up -d --build

# 宿主机 Docker：
# docker compose up -d --build
```

浏览器访问：`.env` 里的 **`CANONICAL_BASE_URL`**（需 DNS 指向本机，且放行 **`HTTP_PUBLISH_PORT`**）。

## 存储与公网地址

- 环境变量 **`CANONICAL_BASE_URL`**（如 `https://geeleo.com`）为**唯一主站真值**。
- `app-storage.yaml` 中 **`cdn_domain` 留空**时，进程会用 **`CANONICAL_BASE_URL`** 自动补全（见 `pkg/config/app_storage.go`）。
- `www` 与裸域：内置 Nginx 将 **`www.<host>` 301 到 `canonical_base_url`**（scheme 与裸域一致）。

## 文件说明

| 文件 | 说明 |
|------|------|
| `docker-compose.yaml` | mysql / nats / minio / main（变量均来自 `.env`，无 `:-` 默认） |
| `Dockerfile` | 主镜像：core-server + web/dist + Nginx + Podman |
| `.env.example` | 必填项清单模板；复制为 **`.env`**（**`.env` 勿提交**） |
| `check-env.sh` | 校验 `.env` 必填非空（纯 bash） |
| `entrypoint-main.sh` | 等待依赖、注入密钥、起 Nginx + Podman + core |
| `nginx/default.conf.template` | 边缘模板（envsubst） |
| `config/prod/*.yaml` | 主站应用 YAML 模板（含占位符，构建进镜像；启动时复制到容器内 `deploy/config/prod` 再 `sed`） |

## 升级

- 应用：`podman compose build main && podman compose up -d main`（Docker 则把 `podman` 换成 `docker`）。
- MySQL / MinIO / NATS：按需单独升级镜像，**注意备份 volume**。

## 已知限制（后续迭代）

- HTTPS 证书：当前模板为 **8080 HTTP**；对外 HTTPS 可在前面再加一层宿主机或云 LB 终结 TLS，并同步 `CANONICAL_BASE_URL` 为 `https://`。
- `minio.use_ssl` 与 HTTPS 预签名细节需按环境微调（见 `core/app-storage/storage/minio.go`）。
