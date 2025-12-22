# Docker 部署指南

本项目支持两种部署方式：

1. **单实例部署**（测试环境）- 所有服务在一个容器中运行
2. **微服务部署**（生产环境）- 每个服务独立容器

---

## 快速开始

### 单实例部署（推荐用于测试环境）

```bash
# 构建并启动所有服务（单容器）
docker-compose -f docker-compose.single.yml up -d --build

# 查看日志
docker-compose -f docker-compose.single.yml logs -f

# 停止服务
docker-compose -f docker-compose.single.yml down
```

### 微服务部署（推荐用于生产环境）

```bash
# 构建并启动所有服务（多容器）
docker-compose -f docker-compose.microservices.yml up -d --build

# 查看日志
docker-compose -f docker-compose.microservices.yml logs -f

# 停止服务
docker-compose -f docker-compose.microservices.yml down
```

---

## 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| API Gateway | 9090 | API 网关入口 |
| App Server | 9091 | 主业务服务 |
| App Storage | 9092 | 存储服务 |
| App Runtime | 9093 | 运行时服务 |
| Agent Server | 9095 | 智能体服务 |
| Control Service | 9096 | 控制服务 |
| MySQL | 3306 | 数据库 |
| NATS | 4222 | 消息中间件 |
| MinIO API | 9000 | 对象存储 API |
| MinIO Console | 9001 | 对象存储控制台 |

---

## 配置说明

### 配置文件位置

配置文件位于 `configs/` 目录，Docker 容器会挂载此目录：

- `configs/global.yaml` - 全局共享配置
- `configs/app-server.yaml` - App Server 配置
- `configs/agent-server.yaml` - Agent Server 配置
- `configs/app-runtime.yaml` - App Runtime 配置
- `configs/app-storage.yaml` - App Storage 配置
- `configs/api-gateway.yaml` - API Gateway 配置
- `configs/control-service.yaml` - Control Service 配置

### 容器环境配置

Docker Compose 会通过环境变量覆盖配置文件中的主机名：

- `GATEWAY_HOST=api-gateway` - 网关主机名（容器内使用服务名）
- `NATS_URL=nats://nats:4222` - NATS 地址（容器内使用服务名）
- `MYSQL_HOST=mysql` - MySQL 主机名（容器内使用服务名）
- `MINIO_ENDPOINT=minio:9000` - MinIO 端点（容器内使用服务名）

**注意**：配置文件中的 `localhost` 或 `127.0.0.1` 在容器环境中会被环境变量覆盖。

---

## 数据持久化

Docker Compose 使用 volumes 持久化数据：

- `mysql-data` - MySQL 数据
- `minio-data` - MinIO 数据
- `app-runtime-data` - App Runtime 的应用数据（`namespace/` 目录）

数据存储在 Docker volumes 中，即使删除容器也不会丢失数据。

---

## 健康检查

所有服务都配置了健康检查，确保服务启动顺序正确：

1. **基础设施服务**（MySQL、NATS、MinIO）先启动并健康
2. **Control Service** 启动
3. **App Runtime、App Storage、Agent Server** 启动
4. **App Server** 启动
5. **API Gateway** 最后启动

---

## 日志查看

### 查看所有服务日志

```bash
# 单实例部署
docker-compose -f docker-compose.single.yml logs -f

# 微服务部署
docker-compose -f docker-compose.microservices.yml logs -f
```

### 查看特定服务日志

```bash
# 单实例部署
docker-compose -f docker-compose.single.yml logs -f ai-agent-os

# 微服务部署
docker-compose -f docker-compose.microservices.yml logs -f app-server
docker-compose -f docker-compose.microservices.yml logs -f api-gateway
```

### 日志文件位置

容器内的日志文件位于 `/app/logs/`，已挂载到主机的 `./logs/` 目录。

---

## 常见问题

### 1. 服务启动失败

**问题**：服务启动失败，日志显示连接数据库失败。

**解决**：
- 检查 MySQL 容器是否正常启动：`docker-compose ps`
- 检查 MySQL 健康检查是否通过：`docker-compose logs mysql`
- 等待 MySQL 完全启动后再启动业务服务（已配置 `depends_on`）

### 2. 端口冲突

**问题**：端口已被占用。

**解决**：
- 修改 `docker-compose.*.yml` 中的端口映射
- 或停止占用端口的服务

### 3. 配置文件不生效

**问题**：修改配置文件后，服务没有使用新配置。

**解决**：
- 重启服务：`docker-compose restart <service-name>`
- 或重新构建：`docker-compose up -d --build`

### 4. 容器内无法访问其他服务

**问题**：容器内使用 `localhost` 无法访问其他服务。

**解决**：
- 容器环境应使用服务名（如 `mysql`、`nats`）而不是 `localhost`
- 确保服务在同一个 Docker 网络中（已配置 `ai-agent-os-network`）

---

## 生产环境建议

1. **使用微服务部署**：`docker-compose.microservices.yml`
2. **配置环境变量**：使用 `.env` 文件管理敏感信息
3. **使用 Docker Secrets**：管理密码、密钥等敏感信息
4. **配置资源限制**：为每个服务设置 CPU 和内存限制
5. **配置日志收集**：使用 ELK、Loki 等日志收集系统
6. **配置监控**：使用 Prometheus、Grafana 等监控系统
7. **配置备份**：定期备份 MySQL 和 MinIO 数据

---

## 开发环境建议

1. **使用单实例部署**：`docker-compose.single.yml`，简单快速
2. **挂载源代码**：开发时可以挂载源代码目录，支持热重载
3. **使用本地数据库**：可以直接使用本地 MySQL，不启动 MySQL 容器

---

## 扩展阅读

- [Docker Compose 官方文档](https://docs.docker.com/compose/)
- [项目配置说明](../configs/README.md)
- [服务架构文档](../blueprint/)

