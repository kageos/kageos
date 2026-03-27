# Docker / Podman-in-Docker 构建与全栈辅助

本目录为 **Compose 全栈** 使用的镜像定义与脚本（与 **`deploy/compose/docker-compose.yml`** 配套）。

| 内容 | 说明 |
|------|------|
| **`Dockerfile.backend`** | 统一后端大镜像（内嵌 Podman） |
| **`Dockerfile.web` / `Dockerfile.hub-frontend`** | 主站 / Hub 前端 Nginx 镜像 |
| **`Dockerfile.app-base`** | 历史用户应用基础镜像定义；官方 canonical 已迁到 `deploy/base/images/app-base/` |
| **`entrypoint-backend.sh`** | 后端容器入口 |
| **`init-db.sql` / `nats-server.conf`** | 开发/Embedding 与 `deploy/dev/compose/` 下的 compose 文件共用 |
| **`deploy.sh`** | 一键 `docker compose` 构建并启动 |
| **`DEPLOY.md`** | 详细部署与运维说明 |

业务 YAML 不在此目录，见 **`../config/compose/`**。
