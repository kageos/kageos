# 客户主站一键部署（Compose：宿主机 Docker 或 Podman）

**范围**：主站 + 中间件（MySQL / NATS / MinIO）+ 内置 Nginx(8080) + **容器内 Podman**（跑用户应用）。**不包含 Hub**（Hub 为公司自建单独运维）。

## 前置

- **宿主机 Podman**：Podman 4+，**`podman compose`** 或 **`podman-compose`**（与 `embedding.sh` 中 `PODMAN_COMPOSE_PROVIDER` 说明一致）。
- **或 Docker**：Docker Engine + `docker compose`。
- **配置**：只在 **`deploy/customer/.env`** 里写键值；**无 YAML、无渲染脚本**（后续你们用任意方式把内容**输出/下发到这个文件**即可）。
- 本机构建需足够 CPU/内存；`main` 需 **`privileged: true`**。

## 快速开始

```bash
cd deploy/customer
cp .env.example .env
# 编辑 .env：全部必填；SMTP 不用则 SMTP_PASSWORD 留空

# 可选：自检必填项
chmod +x check-env.sh && ./check-env.sh

podman compose up -d --build
# Docker：docker compose up -d --build
```

浏览器访问：**`CANONICAL_BASE_URL`**（DNS 指向本机，放行 **`HTTP_PUBLISH_PORT`**）。

## 存储与公网地址

- **`CANONICAL_BASE_URL`** 为唯一主站真值。
- `app-storage` 里 **`cdn_domain` 空**时由进程用 **`CANONICAL_BASE_URL`** 补全（`pkg/config/app_storage.go`）。
- Nginx：**`www.<host>`** 301 到与真值一致的 scheme + 裸域。

## 文件说明

| 文件 | 说明 |
|------|------|
| `.env` | **Compose 唯一环境配置**（勿提交；由人工或你们自己的系统写入本目录） |
| `.env.example` | 键名清单 |
| `check-env.sh` | 可选：校验必填非空（纯 bash） |
| `docker-compose.yaml` | 服务定义，变量来自 `.env` |
| `Dockerfile` / `entrypoint-main.sh` / `nginx/` / `config/prod/` | 镜像与内置模板 |

## 升级

- 应用：`podman compose build main && podman compose up -d main`。

## 已知限制

- 边缘当前为容器内 **8080 HTTP**；对外 HTTPS 可在前加 LB，并同步 **`CANONICAL_BASE_URL`** 为 `https://`。
