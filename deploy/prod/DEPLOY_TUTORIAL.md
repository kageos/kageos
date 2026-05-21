# 一分钟部署

只看这份就够了。

前提：

- Linux
- 已安装 `podman compose` 或 `docker compose`
- `/data` 可写
- `80` 端口空闲；如果容器自己做 HTTPS，`443` 也要空闲

## 最短路径

```bash
git clone <your-repo-url>
cd kageos

go run ./cmd/kagectl init --base-url http://your-ip-or-domain
go run ./cmd/kagectl doctor --config .kageos/prod/kage.yaml
go run ./cmd/kagectl up --config .kageos/prod/kage.yaml
go run ./cmd/kagectl verify --config .kageos/prod/kage.yaml
```

`kagectl init` 会生成 `.kageos/prod/kage.yaml`，里面包含数据库密码、NATS 密码、JWT 密钥、system 初始密码等敏感配置，默认不入库。

## 改公网地址或 TLS

编辑 `.kageos/prod/kage.yaml`：

```yaml
site:
  base_url: "http://your-ip-or-domain"
  tls_mode: "http"
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
go run ./cmd/kagectl up --config .kageos/prod/kage.yaml
```

渲染后证书会落到 `.kageos/prod/generated/tls/`，最终注入容器的环境变量会落到 `.kageos/prod/generated/env/kageos.env`，便于后续运维查看和备份。

改完执行：

```bash
go run ./cmd/kagectl up --config .kageos/prod/kage.yaml
```

## 常用命令

```bash
go run ./cmd/kagectl status --config .kageos/prod/kage.yaml
go run ./cmd/kagectl logs --config .kageos/prod/kage.yaml main
go run ./cmd/kagectl verify --config .kageos/prod/kage.yaml
go run ./cmd/kagectl down --config .kageos/prod/kage.yaml
go run ./cmd/kagectl uninstall --config .kageos/prod/kage.yaml
go run ./cmd/kagectl uninstall --config .kageos/prod/kage.yaml --purge-data --force
```

`down` 只停服务；`uninstall` 会移除 Compose 栈和可再生的 `.kageos/prod/generated/`。来回测试想清数据库/对象存储/业务数据但不重建用户应用基础镜像，用 `--purge-data --force`，它默认保留 `/data/kageos/podman_storage`。

## 生成物

生产部署只维护 `kagectl` 入口；如需变更配置，修改 `.kageos/prod/kage.yaml` 后重新执行 `kagectl up`。
