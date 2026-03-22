# AI Agent OS - Docker Compose 部署指南

## Compose 文件位置

- **全栈定义**：[`../compose/docker-compose.yml`](../compose/docker-compose.yml)（说明见 [`../compose/README.md`](../compose/README.md)）。
- 仓库根 [`docker-compose.yml`](../../docker-compose.yml) 通过 **`include`** 引入上述文件，在根目录仍可直接执行 `docker compose up -d`。
- 下文命令若在**仓库根**执行，统一使用 **`-f deploy/compose/docker-compose.yml`**（与先 `cd deploy/compose` 再执行等价）。

## 一、环境要求

| 项目 | 最低要求 | 推荐配置 |
|---|---|---|
| 操作系统 | Linux (x86_64) | Ubuntu 22.04 / Debian 12 |
| CPU | 4 核 | 8 核+ |
| 内存 | 8GB | 16GB+ |
| 磁盘 | 40GB | 100GB+（SSD） |
| Docker | 24.0+ | 最新版 |
| Docker Compose | V2 (2.20+) | 最新版 |

## 二、快速部署

### 1. 安装 Docker（如未安装）

```bash
# 一键安装 Docker（包含 Docker Compose V2）
curl -fsSL https://get.docker.com | sh

# 将当前用户加入 docker 组（避免 sudo）
sudo usermod -aG docker $USER
newgrp docker

# 验证安装
docker --version
docker compose -f deploy/compose/docker-compose.yml version
```

### 2. 克隆项目

```bash
git clone <your-repo-url> ai-agent-os
cd ai-agent-os
```

### 3. 配置域名/IP

编辑 `deploy/config/compose/app-storage.yaml`，将 `cdn_domain` 改为你的服务器外网域名或 IP：

```yaml
storage:
  minio:
    cdn_domain: "http://your-domain.com"    # 有域名
    # cdn_domain: "http://192.168.1.100"    # 无域名用 IP
```

> Nginx 已配置 `/ai-agent-os/` 路径反向代理到 MinIO，浏览器通过主域名 80 端口即可访问所有文件，无需暴露 MinIO 9000 端口。
> `endpoint` 和 `server_endpoint` 保持 `minio:9000` 不需要改。

### 4. 一键部署

```bash
bash deploy/podman/deploy.sh
```

这个脚本会依次：
1. 构建 Go 后端镜像（内含 Podman 容器引擎）
2. 构建前端镜像
3. 启动所有服务
4. 首次启动时，后端容器内部会自动用 Podman 构建用户应用基础镜像（约 10-20 分钟）

### 5. 访问

| 服务 | 地址 |
|---|---|
| Web 主前端 | http://你的域名或IP |
| Hub 前端 | http://你的域名或IP:81 |
| API Gateway | http://你的域名或IP:9090 |
| MinIO 控制台 | http://你的域名或IP:9001 |

## 三、服务架构

```
┌─────────────┐  ┌──────────────┐
│ Web 前端:80  │  │ Hub 前端:81  │
│   (Nginx)   │  │   (Nginx)    │
└──────┬──────┘  └──────┬───────┘
       │                │
       ├── /api/* ──────┤──→ backend:9090 (API Gateway)
       └── /ai-agent-os/* ──→ minio:9000  (文件直传)
       
┌──────────────────────────────────────┐
│     backend 容器 (privileged)         │
│  ┌──────────┬──────────────────┐     │
│  │ Gateway  │  app-server      │     │
│  │  :9090   │  :9091           │     │
│  ├──────────┼──────────────────┤     │
│  │ storage  │  agent-server    │     │
│  │  :9092   │  :9095           │     │
│  ├──────────┼──────────────────┤     │
│  │ runtime  │  control-service │     │
│  │  :9093   │  :9096           │     │
│  ├──────────┼──────────────────┤     │
│  │ hub      │  hr-server       │     │
│  │  :9094   │  :9097           │     │
│  └──────────┴──────────────────┘     │
│  ┌─────────────────────────────┐     │
│  │ Podman (内嵌容器引擎)        │     │
│  │  管理用户应用容器             │     │
│  └─────────────────────────────┘     │
└──────────────────────────────────────┘
       │         │         │
  ┌────┴──┐ ┌───┴──┐ ┌───┴───┐
  │ MySQL │ │ NATS │ │ MinIO │
  │ :3306 │ │:4222 │ │:9000  │
  └───────┘ └──────┘ └───────┘
```

### 完全自包含

项目使用 **Podman-in-Docker** 架构：容器引擎（Podman）内嵌在 backend 容器中，不依赖宿主机的 Docker daemon 来管理用户应用容器。拉代码 → 配 IP → `docker compose -f deploy/compose/docker-compose.yml up` → 搞定。

## 四、配置说明

### 配置覆盖机制

Docker 环境使用 `deploy/config/compose/` 下的配置文件覆盖默认配置：

| 配置文件 | 用途 | 关键变更 |
|---|---|---|
| `global.yaml` | 全局配置 | NATS 地址改为 `nats:4222`，SDK 使用 `host.docker.internal` |
| `app-server.yaml` | 应用服务 | MySQL 地址改为 `mysql:3306` |
| `app-storage.yaml` | 存储服务 | endpoint 改为 `minio:9000`，**cdn_domain 改为域名/IP** |
| `app-runtime.yaml` | 运行时 | 容器引擎改为 Podman（内嵌），socket 自动连接 |
| `agent-server.yaml` | Agent 服务 | MySQL 地址改为 `mysql:3306` |
| `hr-server.yaml` | HR 服务 | MySQL 地址改为 `mysql:3306` |
| `hub.yaml` | Hub 服务 | MySQL 地址改为 `mysql:3306` |

### 关键配置：文件访问域名

`deploy/config/compose/app-storage.yaml` 中的 `cdn_domain` 是**唯一需要根据部署环境修改的配置**：

```yaml
storage:
  minio:
    endpoint: "minio:9000"                  # 后端直连 MinIO（不需要改）
    server_endpoint: "minio:9000"           # 容器内部访问（不需要改）
    cdn_domain: "http://your-domain.com"    # 浏览器通过此地址访问文件
```

原理：Nginx 的 `/ai-agent-os/` location 反向代理到 MinIO，所以浏览器通过 `http://your-domain.com/ai-agent-os/文件路径` 即可访问文件，上传的 presigned URL 也走同域名，无 CORS 问题。

### 系统账号

初始系统账号密码在 `deploy/config/compose/hr-server.yaml` 中配置：

```yaml
system_user:
  password: "Admin@123456"
```

### 邮箱配置

如需邮箱验证码功能，编辑 `deploy/config/compose/app-server.yaml` 和 `deploy/config/compose/hr-server.yaml` 中的 `email.smtp` 配置。

## 五、常用命令

```bash
# 查看服务状态
docker compose -f deploy/compose/docker-compose.yml ps

# 查看日志
docker compose -f deploy/compose/docker-compose.yml logs -f              # 所有服务
docker compose -f deploy/compose/docker-compose.yml logs -f backend      # 仅后端
docker compose -f deploy/compose/docker-compose.yml logs -f mysql        # 仅 MySQL

# 重启服务
docker compose -f deploy/compose/docker-compose.yml restart              # 全部重启
docker compose -f deploy/compose/docker-compose.yml restart backend      # 仅重启后端

# 停止服务
docker compose -f deploy/compose/docker-compose.yml stop                 # 停止（保留数据）
docker compose -f deploy/compose/docker-compose.yml down                 # 停止并删除容器
docker compose -f deploy/compose/docker-compose.yml down -v              # 停止并删除容器和数据卷（⚠️ 危险）

# 重新构建（代码更新后）
docker compose -f deploy/compose/docker-compose.yml build backend        # 重建后端
docker compose -f deploy/compose/docker-compose.yml build web            # 重建前端
docker compose -f deploy/compose/docker-compose.yml up -d                # 重启

# 进入容器调试
docker compose -f deploy/compose/docker-compose.yml exec backend bash
docker compose -f deploy/compose/docker-compose.yml exec mysql mysql -uroot -proot

# 查看内嵌 Podman 状态
docker compose -f deploy/compose/docker-compose.yml exec backend podman info
docker compose -f deploy/compose/docker-compose.yml exec backend podman ps -a
docker compose -f deploy/compose/docker-compose.yml exec backend podman images
```

## 六、数据持久化

| 数据卷 | 用途 | 说明 |
|---|---|---|
| `ai-agent-os-mysql-data` | MySQL 数据 | 所有业务数据 |
| `ai-agent-os-minio-data` | MinIO 数据 | 上传的文件 |
| `ai-agent-os-podman-storage` | Podman 存储 | 用户应用基础镜像和容器（首次构建后缓存） |
| `ai-agent-os-namespace-data` | 用户应用代码 | 用户创建的应用 |
| `ai-agent-os-app-runtime-data` | 运行时数据 | 应用运行状态 |
| `ai-agent-os-backend-logs` | 日志 | 服务日志 |
| `ai-agent-os-license-data` | License | 授权文件 |

## 七、故障排查

### MySQL 启动失败

```bash
docker compose -f deploy/compose/docker-compose.yml logs mysql

# 如果是权限问题
docker compose -f deploy/compose/docker-compose.yml down
docker volume rm ai-agent-os-mysql-data
docker compose -f deploy/compose/docker-compose.yml up -d
```

### 后端连接数据库失败

```bash
docker compose -f deploy/compose/docker-compose.yml exec mysql mysqladmin ping -uroot -proot
docker compose -f deploy/compose/docker-compose.yml exec mysql mysql -uroot -proot -e "SHOW DATABASES;"
```

### 前端无法访问 API

```bash
docker compose -f deploy/compose/docker-compose.yml ps backend
curl http://localhost:9090/swagger/index.html
```

### 用户应用容器无法启动

```bash
# 检查内嵌 Podman 是否正常
docker compose -f deploy/compose/docker-compose.yml exec backend podman info

# 检查 ai-agent-os:latest 基础镜像是否已构建
docker compose -f deploy/compose/docker-compose.yml exec backend podman images | grep ai-agent-os

# 如果镜像不存在，手动触发构建
docker compose -f deploy/compose/docker-compose.yml exec backend podman build -t ai-agent-os:latest -f /app/app-base/Dockerfile /app/app-base/

# 查看用户应用容器日志
docker compose -f deploy/compose/docker-compose.yml exec backend podman logs <container-name>
```

### 文件/图片无法显示

```bash
# 确认 MinIO 是否正常运行
docker compose -f deploy/compose/docker-compose.yml ps minio

# 确认 Nginx 代理是否工作（应返回 MinIO 错误页面或文件内容）
curl -v http://localhost/ai-agent-os/

# 确认 endpoint 配置是否正确（应为浏览器可访问的域名/IP）
cat deploy/config/compose/app-storage.yaml | grep endpoint
```

## 八、升级

```bash
git pull
docker compose -f deploy/compose/docker-compose.yml build
docker compose -f deploy/compose/docker-compose.yml up -d
```

## 九、安全注意事项

1. **修改默认密码**：MySQL root 密码、MinIO 密码、系统账号密码
2. **JWT 密钥**：修改 `deploy/config/compose/global.yaml` 中的 `jwt.secret`
3. **privileged 模式**：backend 容器运行在 privileged 模式（Podman 需要），自用服务器可接受，公网环境建议配合防火墙
4. **端口暴露**：生产环境建议使用反向代理统一入口，MinIO 9000/9001 端口可不暴露（已通过 Nginx 代理）
