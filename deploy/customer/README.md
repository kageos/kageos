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

## 存储与公网地址

- **`CANONICAL_BASE_URL`**（写在 `env.yaml` → 生成进 `.env`）为唯一主站真值。
- `cdn_domain` 空时由进程用该 URL 补全；Nginx **`www` → 裸域 301** 与真值 scheme 一致。

## 文件说明

| 文件 | 说明 |
|------|------|
| **`env.yaml.example`** | 结构模板；用户复制为 **`env.yaml`** 并手写 |
| **`env.yaml`** | 用户真实配置（**勿提交**，见 `.gitignore`） |
| **`render-env.sh`** | `env.yaml` → **`.env`**（校验必填，无 Python） |
| **`.env`** | Compose 读取（**勿手改**、勿提交） |
| `docker-compose.yaml` | 服务定义 |
| `Dockerfile` / `entrypoint-main.sh` / `nginx/` / `config/prod/` | 镜像与内置模板 |

## 升级

`podman compose build main && podman compose up -d main`

## 已知限制

- 边缘为容器内 **8080 HTTP**；对外 HTTPS 前加 LB 并同步 **`CANONICAL_BASE_URL`** 为 `https://`。
