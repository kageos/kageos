# 本地启动参考

## 目录

- 环境要求
- 启动决策
- 容器引擎处理
- 官方启动命令
- 验证与登录
- 常见故障

## 环境要求

以目标源码仓库当前 `README.md` 和 `CONTRIBUTING.md` 为最终事实。当前最低要求：

| 依赖 | 要求 |
|---|---|
| Git | 能访问 GitHub 并克隆仓库 |
| Go | 1.25 或更新版本 |
| Node.js | 20.19+，或 22.12+ |
| npm | 随 Node.js 提供 |
| 容器引擎 | Docker Compose 或 Podman Compose，任选一个 |

MySQL、NATS、MinIO 不需要在宿主机单独安装，由 `kagectl` 管理。

安装命令容易随操作系统和包管理器变化。执行安装前先检测 macOS/Linux、CPU 架构、现有包管理器和官方文档；向用户展示将执行的准确命令并获得确认。不要修改 shell 启动文件来自动启动 Podman。

## 启动决策

```text
已有健康 Docker + Compose → 使用 Docker
否则已有健康 Podman + Compose → 使用 Podman
否则只有 Docker 但 daemon 未运行 → 帮助启动或提示打开 Docker Desktop
否则只有 Podman → 安全诊断 machine 和连接
否则 → 经用户确认后安装一种容器引擎
```

用户没有偏好时，复用现有健康环境优先于安装新软件。

## 容器引擎处理

### Docker

至少验证：

```bash
docker compose version
docker info
```

CLI 存在但 `docker info` 失败时，通常是 daemon/Desktop 没启动或当前 context 不可用。先恢复现有 Docker，不直接安装第二套引擎。

### Podman

至少验证：

```bash
podman compose version
podman info
podman machine list
podman system connection list
```

macOS/Windows 的 Podman 依赖虚拟机。只有在 `podman machine list` 明确显示不存在 machine、机器属于全新安装且用户确认后，才执行初始化。已存在 machine、连接异常、磁盘空间不足或命令卡住时，先诊断和保护数据，禁止直接 init/rm/reset。

## 官方启动命令

在仓库根目录一键启动完整开发环境：

```bash
./scripts/dev.sh
```

脚本会同时启动基础设施、后端和前端，并持续显示两边输出。需要分开调试时再使用下面的命令。

在仓库根目录启动后端及基础设施：

```bash
go run ./cmd/kagectl bootstrap --dev
```

强制使用 Docker：

```bash
go run ./cmd/kagectl bootstrap --dev --engine docker
```

强制使用 Podman：

```bash
go run ./cmd/kagectl bootstrap --dev --engine podman
```

另一个终端启动前端：

```bash
cd web
npm install
npm run dev
```

只初始化并通过 IDE 启动后端：

```bash
go run ./cmd/kagectl init --dev
```

然后从仓库根目录运行 `core/cmd/main/main.go`。

## 验证与登录

```bash
go run ./cmd/kagectl doctor
go run ./cmd/kagectl status
go run ./cmd/kagectl verify
```

默认入口是 `http://localhost:5173`，前端默认代理到 `http://localhost:9090`。管理员用户名是 `system`；密码由 `bootstrap --dev` 的初始化摘要显示。摘要丢失时运行：

```bash
go run ./cmd/kagectl init --dev
```

本地 `.kageos/dev/env/kageos.env` 含私密密码和密钥，不输出、不提交。

停止前端和后端前台进程后，停止基础设施：

```bash
go run ./cmd/kagectl down
```

`down` 默认保留本地数据。

## 常见故障

| 现象 | 首要检查 |
|---|---|
| `compose is required` | `docker compose version` 或 `podman compose version` |
| Docker CLI 存在但不可用 | Docker daemon/Desktop 与当前 context |
| Podman 连接失败 | machine、connection、磁盘空间；不要立即 init |
| 后端退出 | `kagectl doctor` 与 `kagectl logs main` |
| MySQL/NATS/MinIO 不健康 | `kagectl status`、`verify`、`logs infra` |
| 前端打不开 | `npm run dev` 输出、5173 端口、Node 版本 |
| API 请求失败 | 9090 健康状态、Vite proxy、后端日志 |
| 首次启动很慢 | 本地用户应用基础镜像可能正在构建 |

默认开发端口包括 5173、9090–9099、3318、4222、9000 和 9001。遇到占用时先识别占用进程，不擅自结束用户的其他服务。
