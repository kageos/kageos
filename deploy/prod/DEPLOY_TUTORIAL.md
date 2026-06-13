# 一分钟部署

只看这份就够了。

前提：

- Linux
- 已安装 `podman compose` 或 `docker compose`
- 默认数据目录 `~/.kageos/storage/prod` 可写，或在 `.kageos/prod/kage.yaml` 中配置其他可写的绝对路径
- 默认需要 `80` 端口空闲；如果该端口不可用，可以配置 `site.http_port` 或安装时传 `--http-port`。如果容器自己做 HTTPS，对应的 `site.https_port` 也要空闲，默认 `443`。

## 最短路径

```bash
git clone <your-repo-url>
cd kageos

go run ./cmd/kagectl init --base-url http://your-ip-or-domain
go run ./cmd/kagectl doctor
go run ./cmd/kagectl up
go run ./cmd/kagectl verify
```

如果宿主机 `80` 被占用，首次部署可以直接指定监听端口：

```bash
sudo ./install.sh --base-url http://your-ip-or-domain:8080 --http-port 8080
```

`kagectl init` 会生成 `.kageos/prod/kage.yaml`，里面包含数据库密码、NATS 密码、JWT 密钥、system 初始密码等敏感配置，默认不入库。

## 改公网地址或 TLS

编辑 `.kageos/prod/kage.yaml`：

```yaml
site:
  base_url: "http://your-ip-or-domain"
  tls_mode: "http"
```

宿主机 80 被占用时，可改用其他端口：

```yaml
site:
  base_url: "http://your-ip-or-domain:8080"
  tls_mode: "http"
  http_port: 8080
```

前面已有 LB / CDN / Ingress 做 HTTPS：

```yaml
site:
  base_url: "https://your-domain"
  tls_mode: "external"
```

容器自己做 HTTPS：

```yaml
site:
  base_url: "https://your-domain"
  tls_mode: "redirect"
  cert_file: "/app/tls/fullchain.pem"
  key_file: "/app/tls/privkey.pem"
  tls_cert_pem_b64: "base64-fullchain-pem"
  tls_key_pem_b64: "base64-privkey-pem"
```

也可以用环境变量传证书内容：

```bash
KAGEOS_TLS_CERT_PEM_B64="$(base64 < fullchain.pem | tr -d '\n')" \
KAGEOS_TLS_KEY_PEM_B64="$(base64 < privkey.pem | tr -d '\n')" \
go run ./cmd/kagectl up
```

渲染后证书会落到 `.kageos/prod/generated/tls/`，最终注入容器的环境变量会落到 `.kageos/prod/generated/env/kageos.env`，便于后续运维查看和备份。

改完执行：

```bash
go run ./cmd/kagectl up
```

## 常用命令

```bash
go run ./cmd/kagectl status
go run ./cmd/kagectl logs main
go run ./cmd/kagectl verify
go run ./cmd/kagectl down
go run ./cmd/kagectl uninstall
go run ./cmd/kagectl uninstall --purge-data --force
```

`down` 只停服务；`uninstall` 会移除 Compose 栈和可再生的 `.kageos/prod/generated/`。来回测试想清数据库/对象存储/业务数据但不重建用户应用基础镜像，用 `--purge-data --force`，它默认保留 `<storage.root>/podman_storage`。

## 生成物

生产部署只维护 `kagectl` 入口；如需变更配置，修改 `.kageos/prod/kage.yaml` 后重新执行 `kagectl up`。
