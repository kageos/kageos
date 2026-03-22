# Docker Compose 全栈（backend + web + hub + 中间件）

Compose 文件：**`docker-compose.yml`**（本目录）。

## 用法

在**仓库根目录**执行（推荐，与文档一致）：

```bash
docker compose -f deploy/compose/docker-compose.yml up -d --build
```

或进入本目录：

```bash
cd deploy/compose
docker compose up -d --build
```

配置挂载仍使用 **`../config/compose/*.yaml`**，说明见 **[../config/compose/README.md](../config/compose/README.md)**。

总部署索引：**[../README.md](../README.md)** · 详细运维：**[../podman/DEPLOY.md](../podman/DEPLOY.md)**（其中 `docker compose` 命令请加 `-f deploy/compose/docker-compose.yml`，或在 `deploy/compose` 下执行）。
