# Prod Quick Start

生产部署入口是 Go 部署器 `kagectl`。Compose 仍是底层容器执行器，但用户不需要直接维护 Compose 文件。

## 前提

- Linux
- 已安装 `podman compose` 或 `docker compose`
- 默认数据目录 `~/.kageos/storage/prod` 可写，或在 `.kageos/prod/kage.yaml` 中配置其他可写的绝对路径
- 默认需要 `80` 端口空闲；如果该端口不可用，可以配置 `site.http_port` 或安装时传 `--http-port`。如果本机终止 HTTPS，对应的 `site.https_port` 也要空闲，默认 `443`。

## 首次部署（一键）

在仓库根目录执行：

```bash
sudo ./install.sh --base-url http://your-ip-or-domain
tail -f .kageos/prod/kagectl-up.log
```

宿主机 `80` 被占用时：

```bash
sudo ./install.sh --base-url http://your-ip-or-domain:8080 --http-port 8080
```

`install.sh` 会选择 sudo 调用者作为部署用户，自动处理 rootless Podman 生产环境需要的 linger，并在缺少生产配置时创建 `.kageos/prod/kage.yaml`，最后调用 `./prod-up.sh` 后台启动部署。

`init` 会在终端打印需要保存的账号、密码、JWT、MinIO、NATS 等英文表格。默认只有 `system` 能登录；需要开放注册时，先进入 `System settings` 配置 SMTP 并发送测试邮件，再开启邮箱验证码注册。

访问：

```text
http://your-ip-or-domain
```

如果安装时用了自定义端口，就访问对应端口，例如 `http://your-ip-or-domain:8080`。

## 常用命令

```bash
go run ./cmd/kagectl status
go run ./cmd/kagectl logs main
go run ./cmd/kagectl down
go run ./cmd/kagectl uninstall --purge-data --force
```

需要后台执行时可用 `./prod-up.sh` / `./prod-stop.sh`，它们只是 `kagectl up/down` 的 wrapper，并把输出写入 `.kageos/prod/kagectl-up.log`；此时可用 `tail -f .kageos/prod/kagectl-up.log` 查看后台部署日志。

`uninstall --purge-data --force` 用于测试重置数据，默认保留 `<storage.root>/podman_storage`，避免每次重新构建用户应用基础镜像。

## 生成物

生成物位于 `.kageos/prod/generated/`，不要手工编辑；需要变更时修改 `.kageos/prod/kage.yaml` 后重新执行 `kagectl up`。
