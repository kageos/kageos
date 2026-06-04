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
go run ./cmd/kagectl doctor
go run ./cmd/kagectl up
go run ./cmd/kagectl verify
```

`init` 会在终端打印需要保存的账号、密码、JWT、MinIO、NATS 等英文表格。默认只有 `system` 能登录；需要开放注册时，先进入 `System settings` 配置 SMTP 并发送测试邮件，再开启邮箱验证码注册。

访问：

```text
http://your-ip-or-domain
```

## 常用命令

```bash
go run ./cmd/kagectl status
go run ./cmd/kagectl logs main
go run ./cmd/kagectl down
go run ./cmd/kagectl uninstall --purge-data --force
```

需要后台执行时可用 `./prod-up.sh` / `./prod-stop.sh`，它们只是 `kagectl up/down` 的 wrapper，并把输出写入 `.kageos/prod/kagectl-up.log`；此时可用 `tail -f .kageos/prod/kagectl-up.log` 查看后台部署日志。

`uninstall --purge-data --force` 用于测试重置数据，默认保留 `/data/kageos/podman_storage`，避免每次重新构建用户应用基础镜像。

## 生成物

生成物位于 `.kageos/prod/generated/`，不要手工编辑；需要变更时修改 `.kageos/prod/kage.yaml` 后重新执行 `kagectl up`。
