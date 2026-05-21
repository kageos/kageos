# Prod Quick Start

生产部署入口是 Go 部署器 `kagectl`。Compose 仍是底层容器执行器，但用户不需要直接维护 Compose 文件。

## 前提

- Linux
- 已安装 `podman compose` 或 `docker compose`
- `/data` 可写
- `80` 端口空闲；如果本机终止 HTTPS，`443` 也要空闲

## 首次部署

在仓库根目录执行：

```bash
go run ./cmd/kagectl init --base-url http://your-ip-or-domain
go run ./cmd/kagectl doctor --config .kageos/prod/kage.yaml
./prod-up.sh
go run ./cmd/kagectl verify --config .kageos/prod/kage.yaml
```

`init` 会在终端打印需要保存的账号、密码、JWT、MinIO、NATS 等英文表格。默认只有 `system` 能登录；需要开放注册时，先进入 `System settings` 配置 SMTP 并发送测试邮件，再开启邮箱验证码注册。

访问：

```text
http://your-ip-or-domain
```

## 常用命令

```bash
tail -f .kageos/prod/kagectl-up.log
go run ./cmd/kagectl status --config .kageos/prod/kage.yaml
go run ./cmd/kagectl logs --config .kageos/prod/kage.yaml main
./prod-stop.sh
go run ./cmd/kagectl uninstall --config .kageos/prod/kage.yaml --purge-data --force
```

`./prod-up.sh` 会在后台执行 `kagectl up`，并把输出写入 `.kageos/prod/kagectl-up.log`，SSH 或终端会话关闭后部署流程不会被当前 shell 带停。需要传递 `kagectl up` 参数时直接追加，例如 `./prod-up.sh --image` 或 `./prod-up.sh --wait-timeout 10m`。

`uninstall --purge-data --force` 用于测试重置数据，默认保留 `/data/kageos/podman_storage`，避免每次重新构建用户应用基础镜像。

## 生成物

生成物位于 `.kageos/prod/generated/`，不要手工编辑；需要变更时修改 `.kageos/prod/kage.yaml` 后重新执行 `kagectl up`。
