# 本地开发入口

本目录是 **本地开发** 的官方入口。

目标：

- 本地起基础设施
- 本地起后端
- 本地起前端
- 不再要求开发同学去记 `customer`、`embedding`、根目录 compose 等历史路径

## 给贡献者的推荐路径

外部贡献者本地开发应优先使用 `kagectl`。本目录下的 compose 文件仍然保留，但它们是开发栈的底层实现和排障入口，不是日常上手入口。

源码仓库里不要求贡献者提前安装独立的 `kagectl` 二进制。文档中的 `kagectl` 日常命令，在贡献者本地都可以写成：

```bash
go run ./cmd/kagectl <command>
```

如果是第一次从 GitHub 拿代码：

```bash
git clone https://github.com/kageos/kageos.git
cd kageos
```

按开发目标选择路径：

| 目标 | 推荐做法 |
| --- | --- |
| 第一次把完整产品跑起来 | `go run ./cmd/kagectl bootstrap --dev`，再另开终端跑 `cd web && npm run dev` |
| 用 GoLand / VS Code 调试后端 | 先执行 `go run ./cmd/kagectl init --dev`，再从仓库根目录运行 `core/cmd/main/main.go` |
| 只改前端 | 跑 `web` 的 dev server；需要连接已有后端时配置 `web/.env.development.local` 里的 `VITE_PROXY_TARGET` |
| 排查本地基础设施 | 先用 `kagectl status`、`kagectl doctor`、`kagectl verify`、`kagectl logs`，再看底层 compose |

`bootstrap --dev` 已经包含初始化步骤，会先完成 `init --dev` 对应的开发模式准备，再启动本地基础设施和后端主进程。只有在你想用 IDE 自己启动后端，或者只想初始化不启动后端时，才需要单独执行 `go run ./cmd/kagectl init --dev`。

本地开发默认优先使用 Podman。若贡献者本机更习惯 Docker，可在首次启动时显式选择：

```bash
go run ./cmd/kagectl bootstrap --dev --engine docker
```

首次启动可能会因为构建本地用户应用基础镜像而比较慢。本地验证码默认走日志模式，不会调用真实 SMTP；验证码可从后端日志或接口返回的 debug code 中查看。

## 启动后如何登录

前后端都启动后，浏览器打开：

```text
http://localhost:5173
```

首次执行 `bootstrap --dev` 或 `init --dev` 时，终端会打印 `kageos dev initialization summary`。本地默认管理员：

```text
Username: system
Password: 使用 summary 里的 Admin password
```

如果终端输出已经被刷掉，可以再次执行下面命令重新打印 summary，不会自动重置已有本地密钥：

```bash
go run ./cmd/kagectl init --dev
```

也可以从 `.kageos/dev/env/kageos.env` 读取 `SYSTEM_USER_PASSWORD`。该文件包含本地随机密钥和密码，只能作为本机私有运行状态，不要提交。

如果想注册新账号，本地默认 `SMTP_MODE=log`，不会真的发邮件。验证码会写入后端日志，并通过 `send_email_code` 接口返回的 `debug_code` 暴露给本地开发使用。

## 环境变量说明

普通本地启动不需要手动 export 环境变量，也不需要自己准备 `.env`。`kagectl` 会生成并维护本地运行所需的 env 文件：

| 文件 | 作用 |
| --- | --- |
| `.kageos/kageos.env` | 记录当前仓库的运行模式，例如 `KAGEOS_MODE=dev` 和 `KAGEOS_DEV_ENGINE=podman`。 |
| `.kageos/dev/env/kageos.env` | 保存本机私有的 MySQL、NATS、MinIO、JWT、SMTP log 模式、`SYSTEM_USER_PASSWORD` 等随机密钥和连接信息。 |

这些文件属于本机私有运行状态，不要提交到仓库。

本地后端场景下，前端也不需要创建 `web/.env.development.local`。不创建该文件，或不设置 `VITE_PROXY_TARGET`，Vite 会默认把 API 请求代理到：

```text
http://localhost:9090
```

只有下面这些情况才需要额外配置：

| 场景 | 配置方式 |
| --- | --- |
| 强制使用 Docker 而不是 Podman | `go run ./cmd/kagectl bootstrap --dev --engine docker` |
| 只跑前端并连接远程后端 | 在 `web/.env.development.local` 设置 `VITE_PROXY_TARGET` |
| 远程后端需要 WebSocket 地址 | 在 `web/.env.development.local` 设置 `VITE_WS_URL` |
| 需要真实发邮件 | 将本地配置改为 `SMTP_MODE=smtp`，并配置 `SMTP_*` |
| 需要使用 AI 工作台 | 登录后配置 LLM；LLM API Key 不是启动平台和登录系统的前置条件 |

## 本地端口

常用端口如下。若启动失败，先检查这些端口是否被其它进程占用。

| 端口 | 服务 |
| --- | --- |
| `5173` | Vite 前端开发服务器 |
| `9090` | API Gateway，前端默认代理目标 |
| `9091` | `app-server` |
| `9092` | `app-storage` |
| `9093` | `app-runtime` |
| `9095` | `agent-server` |
| `9096` | `connector-server` |
| `9097` | `hr-server` |
| `9098` | `timer-scheduler` |
| `9099` | `message-server` |
| `3318` | 本地 dev MySQL 宿主机端口，容器内仍是 `3306` |
| `4222` | NATS |
| `9000` | MinIO API |
| `9001` | MinIO Console |

## 常见排障

建议按这个顺序排查：

```bash
go run ./cmd/kagectl doctor
go run ./cmd/kagectl status
go run ./cmd/kagectl verify
go run ./cmd/kagectl logs main
go run ./cmd/kagectl logs infra
```

- `doctor` 检查当前仓库是否已经初始化为 dev 模式、配置目录是否存在、compose 命令是否可用。
- `status` 查看本地 MySQL / NATS / MinIO 容器状态。
- `verify` 检查基础设施和各平台服务健康状态。
- `logs main` 查看后端主进程日志。
- `logs infra` 查看 MySQL / NATS / MinIO 日志。

停止本地基础设施：

```bash
go run ./cmd/kagectl down
```

`down` 默认不会删除本地数据。如果确实需要彻底重置，请先备份 `.kageos/dev/namespace/` 里的本地用户应用数据，再清理 `.kageos/dev/` 状态和当前容器引擎对应的本地 volume。不要在没有备份的情况下直接删除本地数据。

## 官方入口

### 1. 一键启动开发后端

推荐首次开发直接用 `kagectl bootstrap --dev`：

```bash
go run ./cmd/kagectl bootstrap --dev
```

该命令会初始化开发模式、启动 MySQL / NATS / MinIO、确保本地用户应用运行时基础镜像存在，并以前台方式启动后端主进程。停止后端用 `Ctrl-C`；停止 MySQL / NATS / MinIO 用：

```bash
go run ./cmd/kagectl down
```

### 1.1 只初始化开发模式

如需只初始化，不启动后端主进程：

```bash
go run ./cmd/kagectl init --dev
```

该命令会写入 `.kageos/kageos.env`：

```dotenv
KAGEOS_MODE=dev
KAGEOS_DEV_ENGINE=podman
```

后续 `kagectl up/status/down/logs/doctor/verify` 会根据这个模式自动走本地开发链路，不需要再手工设置运行模式。

该命令默认使用 Podman 启动 MySQL / NATS / MinIO，执行幂等数据库初始化 SQL，并确保本地用户应用运行时基础镜像 `kagebase:latest` 存在；如果镜像已存在会跳过构建。
首次执行时会生成 `.kageos/dev/env/kageos.env`，里面包含 MySQL、NATS、MinIO、JWT、system user 等本地随机密钥；后续重复执行会复用已有值，避免把已有本地数据库密码刷掉。
初始化结束后会在终端打印英文表格，列出本地开发需要记录的路径、账号、密码和 key。
本地开发默认 `SMTP_MODE=log`，发送验证码时不会调用真实邮箱服务；验证码会写入后端日志，并通过 `send_email_code` 接口的 `debug_code` 返回，填这个验证码即可注册。生产环境需要真实发信时再改成 `SMTP_MODE=smtp` 并配置 `SMTP_*`。
本地 dev MySQL 默认只绑定宿主机 `127.0.0.1:3318`，容器内部仍是 `3306`，这样不容易和你机器上的旧 MySQL 或其它项目冲突。
幂等规则：一旦 `.kageos/dev/env/kageos.env` 存在，`init --dev` 只复用里面的值，不会隐式生成替换密码。

若你想显式指定容器引擎：

```bash
go run ./cmd/kagectl init --dev --engine docker
go run ./cmd/kagectl init --dev --engine podman
```

如只想初始化 MySQL / NATS / MinIO，不处理基础镜像：

```bash
go run ./cmd/kagectl init --dev --skip-base
```

如需重新生成本地密钥：

```bash
go run ./cmd/kagectl init --dev --regen-secrets
```

注意：如需重新生成本地密钥，请先清理本地开发数据 volume，否则已有 MySQL/MinIO 数据仍会使用原密码。

如需初始化时使用自定义基础镜像 tag：

```bash
go run ./cmd/kagectl init --dev --base-image "registry.example.com/kagebase:stable"
```

下面是排障时才需要看的底层 compose 命令，不是日常入口。

Docker 本地开发：

```bash
docker compose --env-file .kageos/dev/env/kageos.env -f deploy/dev/compose/docker-compose.dev.yml up -d
```

Podman 本地开发：

```bash
podman compose --env-file .kageos/dev/env/kageos.env -f deploy/dev/compose/docker-compose.infra.yml up -d
```

两份 compose 都保留的原因很简单：

- `docker-compose.dev.yml` 给 Docker 本地开发用，容器名和 volume 名更偏“项目隔离”，避免和你机器上别的容器混淆。
- `docker-compose.infra.yml` 给 Podman 本地开发用，容器名必须固定为 `mysql8 / nats-server / minio`，这样 `app-runtime` 的基础设施探测逻辑才能直接识别。
- 它们的服务内容基本一致，主要差异就是容器名和 volume 名；不要试图为了“只保留一份文件”把这层约束藏起来。

### 1.1 开发配置

本地开发的 canonical 配置目录是：

```text
.kageos/dev/config/
```

当前服务配置加载会读取 `.kageos/dev/config/*.yaml`。`message-server` 和 `timer-scheduler` 已经是主线平台服务，本地开发配置里应保留对应 YAML；本地基础设施仍只需要 MySQL、NATS 和 MinIO。历史备份控制面和旧控制面服务不属于当前本地开发必需项。

用户应用源码、构建产物和运行元数据默认位于 `.kageos/dev/namespace/`。仓库根目录下的 `namespace/` 不是当前本地开发的 canonical 位置，只应视为旧运行数据或临时排障数据。

### 2. 单独起后端

命令行启动：

```bash
go run ./cmd/kagectl up
```

`kagectl up` 会先确保本地基础设施运行，再以前台方式启动 `core/cmd/main`。停止后端用 `Ctrl-C`；停止 MySQL / NATS / MinIO 用：

```bash
go run ./cmd/kagectl down
```

如需从 GoLand 启动平台主进程：

- Run target：`core/cmd/main/main.go`
- Working directory：仓库根目录

说明：

- GoLand 启动前先执行 `go run ./cmd/kagectl init --dev`，让 `.kageos/kageos.env` 记录开发模式。
- 服务配置会按 `.kageos/kageos.env` 自动读取 `.kageos/dev/config/*.yaml`。

### 2.1 构建用户应用运行时基础镜像

如果本地缺少用户应用运行时基础镜像，或想单独重建，请执行 canonical 脚本：

```bash
go run ./cmd/kagectl build-app-base
```

如需即使 tag 已存在也重建：

```bash
go run ./cmd/kagectl build-app-base --force
```

如需自定义 tag，可临时指定：

```bash
KAGEOS_APP_BASE_IMAGE="kagebase:latest" go run ./cmd/kagectl build-app-base
# 或：
go run ./cmd/kagectl build-app-base --image "kagebase:latest"
```

如需强制重建且不复用缓存：

```bash
go run ./cmd/kagectl build-app-base --force --no-cache
```

dev 默认配置 `.kageos/dev/config/app-runtime.yaml` 里的 `container.image.base_image` 默认也是 `kagebase:latest`。如需临时切换运行时镜像，可在启动 app-runtime / core-server 时设置 `KAGEOS_APP_BASE_IMAGE`。

### 3. 起前端

```bash
cd web
npm run dev
```

若只跑前端、连线上后端，请在 `web/.env.development.local` 中配置 `VITE_PROXY_TARGET`。

本地开发相关资源的 canonical 位置已收敛到本目录及 `deploy/base/`。

## 开发配置约定

- `.kageos/dev/config/` 和 `.kageos/dev/env/` 里的密钥、SMTP、系统用户密码都应视为本机私有值，不要提交。
- `.kageos/dev/namespace/` 是本地用户应用运行目录，必要时先备份再清理。
- 如果你本地确实要联通真实 SMTP / 其他外部服务，请改你自己的本地配置，不要把真实账号密码写回仓库。
