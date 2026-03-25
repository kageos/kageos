# 部署说明（入口）

本目录收敛 **部署方式与辅助资源**；具体步骤请进入对应子目录。

| 方式 | 目录 | 说明 |
|------|------|------|
| **客户主站一键（Compose）** | [customer/](customer/) | **不含 Hub**：**`deploy/customer/.env`** 为唯一配置（无 YAML/渲染）；可选 **`./check-env.sh`**；**`podman compose`** / **`docker compose`** 起栈；说明见 [customer/README.md](customer/README.md)。 |
| **Embedding（推荐先做）** | [embedding/](embedding/) | 一机一实例：**Podman Compose** 起中间件；脚本 **`embedding/scripts/embedding.sh`** 支持 **`init`**（含前后端+Nginx+启进程）、**`update`**（`git pull` 后编译+构建+重启）、`restart`/`stop`/`status`/`logs`。 |
| **Docker Compose 全栈** | [compose/](compose/) + [podman/](podman/) | 根目录 **`docker-compose.yml`** `include` **[compose/docker-compose.yml](compose/docker-compose.yml)**；镜像与 **`deploy.sh`** 在 **[podman/](podman/)**，说明 **[podman/DEPLOY.md](podman/DEPLOY.md)**。 |
| **Distributed（规划）** | [distributed/](distributed/) | 各服务独立进程/容器、可水平扩展；文档与脚本后续补充。 |

其他：

- **本机只跑前端、连线上网关**：[前端开发-本地与连线上.md](前端开发-本地与连线上.md)
- **Nginx 站点配置**（裸机 Web + Hub + 反代）：[embedding/nginx/nginx-server.conf](embedding/nginx/nginx-server.conf)（说明见 [embedding/nginx/README.md](embedding/nginx/README.md)）
- **历史脚本**（Docker 起中间件 + 本机二进制）：[server-deploy.sh](server-deploy.sh) — 与 `embedding` 目标一致时可逐步迁到 **`embedding/scripts/embedding.sh`**。

**应用 YAML**：开发与 Compose 挂载在 **`deploy/config/`**（**`dev` / `prod` / `compose`**），说明见 **[deploy/config/README.md](deploy/config/README.md)**。客户主站一键部署的 prod 模板在 **`deploy/customer/config/prod/`**（与 **`deploy/customer`** 同包）。可选 **`AI_AGENT_OS_ROOT`** 指定项目根。
