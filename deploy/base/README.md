# 部署共享资源（Canonical Base）

本目录存放 **dev / prod 共享** 的部署与构建资源。

原则：

- **共享的** Dockerfile、基础镜像启动脚本、Nginx 模板、初始化 SQL、通用脚本放这里。
- **环境入口** 不放这里；本地开发看 `deploy/dev/`，线上部署看 `deploy/prod/`。
- **业务源码** 不迁入本目录；这里只收敛交付与运行资源。

当前收拢内容：

- `images/app-base/`：用户应用基础镜像（`ai-agent-os:latest`）的 canonical Dockerfile 与启动脚本
- `images/backend/`：后端大镜像 Dockerfile 与 entrypoint
- `images/web/`：Web 前端镜像 Dockerfile 与 Nginx 配置
- `images/hub-frontend/`：Hub 前端镜像 Dockerfile 与 Nginx 配置
- `infra/mysql/`：MySQL 初始化 SQL
- `infra/nats/`：NATS 配置
- `scripts/`：共享构建脚本

兼容说明：

- 仓库根的 `build/`、`docker-compose.dev.yml`、`docker-compose.infra.yml` 目前仍保留为兼容入口。
- 新增和维护时，请优先修改 `deploy/base/` 下的 canonical 资源。
