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
sudo ./install.sh --base-url app.example.com
tail -f .kageos/prod/kagectl-up.log
```

如果 `--base-url` 是域名，安装器默认启用 HTTPS pending：`site.base_url` 会写成 `https://app.example.com`，`tls_mode` 为 `redirect`，没有正式证书时先生成临时自签证书。浏览器会提示证书不受信任；后续把正式证书写入 `<storage.root>/tls/fullchain.pem` 和 `<storage.root>/tls/privkey.pem` 后，执行 `go run ./cmd/kagectl reload-tls` 即可热加载可信 HTTPS，不需要重装。

只想用 IP 或临时 HTTP 尝鲜时：

```bash
sudo ./install.sh --base-url http://your-ip-or-domain --tls-mode http
```

宿主机 `80` 被占用时：

```bash
sudo ./install.sh --base-url http://your-ip-or-domain:8080 --http-port 8080
```

`install.sh` 会选择 sudo 调用者作为部署用户，自动处理 rootless Podman 生产环境需要的 linger，并在缺少生产配置时创建 `.kageos/prod/kage.yaml`，最后调用 `./prod-up.sh` 后台启动部署。

`init` 会在终端打印需要保存的账号、密码、JWT、MinIO、NATS 等英文表格。默认只有 `system` 能登录；需要开放注册时，先进入 `System settings` 配置 SMTP 并发送测试邮件，再开启邮箱验证码注册。

访问：

```text
https://app.example.com
```

如果安装时用了自定义端口，就访问对应端口，例如 `http://your-ip-or-domain:8080`。

## 安装为桌面 Web 应用

先通过最终、长期不变的 HTTPS 域名访问 kageos，确认登录和接口正常，再使用浏览器的
“安装应用”功能。不要先用临时 IP 或测试域名安装后再切换到正式域名；PWA 会绑定安装
时的 origin，域名变化会被浏览器视为另一个应用。

线上安装可以和本机 `http://localhost:8080` 的 AIO 安装同时存在。两者共享同一份
前端代码，但分别通过自己的同源 Nginx 路由访问 API。Service Worker 不缓存 API、
认证或上传请求，因此离线时只能加载前端壳，不能离线执行 kageos 业务能力。

## 常用命令

```bash
go run ./cmd/kagectl status
go run ./cmd/kagectl logs main
go run ./cmd/kagectl reload-tls
go run ./cmd/kagectl down
go run ./cmd/kagectl uninstall --purge-data --force
```

需要后台执行时可用 `./prod-up.sh` / `./prod-stop.sh`，它们只是 `kagectl up/down` 的 wrapper，并把输出写入 `.kageos/prod/kagectl-up.log`；此时可用 `tail -f .kageos/prod/kagectl-up.log` 查看后台部署日志。

`uninstall --purge-data --force` 用于测试重置数据，默认保留 `<storage.root>/podman_storage`，避免每次重新构建用户应用基础镜像。

## 生成物

生成物位于 `.kageos/prod/generated/`，不要手工编辑；TLS 证书位于 `<storage.root>/tls/`，默认是 `~/.kageos/storage/prod/tls/`。需要变更普通配置时修改 `.kageos/prod/kage.yaml` 后重新执行 `kagectl up`；只替换证书时执行 `kagectl reload-tls`。
