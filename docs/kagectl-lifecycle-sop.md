# Kageos 生命周期 SOP

Kageos 生命周期只有一个官方入口：`kagectl`。

当前运行模式记录在：

```text
.kageos/kageos.env
```

## 配置归属

不要把所有参数都塞进同一个 env 文件。

- `.kageos/kageos.env`：本机工作区状态，只放 `KAGEOS_MODE`、`KAGEOS_DEV_ENGINE` 这类 `kagectl` 启动前就要读取的选择器。
- `.kageos/dev/env/kageos.env`：本地开发依赖的账号、密码、JWT、system user、SMTP 等环境变量。
- `.kageos/prod/generated/env/kageos.env`：生产渲染后的容器环境变量和密钥。
- `.kageos/prod/kage.yaml`：生产部署声明，保留结构化配置，不用 env 扁平化。

## 本地开发

一键启动开发后端：

```bash
go run ./cmd/kagectl bootstrap --dev
```

该命令会初始化开发模式、启动 MySQL / NATS / MinIO、确保本地用户应用运行时基础镜像存在，并以前台方式运行 `core/cmd/main`。停止后端用 `Ctrl-C`。

只初始化开发模式：

```bash
go run ./cmd/kagectl init --dev
```

该命令会写入：

```dotenv
KAGEOS_MODE=dev
KAGEOS_DEV_ENGINE=podman
```

单独启动后端：

```bash
go run ./cmd/kagectl up
```

dev 模式下，`up` 会先启动 MySQL / NATS / MinIO，再以前台方式运行 `core/cmd/main`。

停止本地基础设施：

```bash
go run ./cmd/kagectl down
```

常用开发命令：

```bash
go run ./cmd/kagectl status
go run ./cmd/kagectl logs main
go run ./cmd/kagectl logs infra
go run ./cmd/kagectl doctor
go run ./cmd/kagectl verify
```

GoLand 仍然可以直接启动 `core/cmd/main/main.go`，但必须先执行 `init --dev`，让 `.kageos/kageos.env` 写入开发模式。

## 生产部署

生产是默认模式。

初始化生产模式：

```bash
go run ./cmd/kagectl init --base-url http://your-ip-or-domain
```

该命令会把 `.kageos/kageos.env` 写成 `KAGEOS_MODE=prod`，并创建 `.kageos/prod/kage.yaml`。

启动或更新生产：

```bash
go run ./cmd/kagectl doctor
go run ./cmd/kagectl up
go run ./cmd/kagectl verify
```

常用生产命令：

```bash
go run ./cmd/kagectl status
go run ./cmd/kagectl logs main
go run ./cmd/kagectl down
go run ./cmd/kagectl uninstall --dry-run
```

`prod-up.sh` 和 `prod-stop.sh` 只是后台 shell 场景的 wrapper，不是独立生命周期入口。

## 内部排障入口

下面这些不是官方生命周期入口：

- 直接执行 `docker compose` / `podman compose`
- 直接执行 `deploy/dev/scripts/infra.sh`
- 单独启动 `core/*/cmd/app/main.go`
- 绕过 `kagectl` 手写临时环境变量启动后端

这些入口只用于调试、测试和事故修复。
