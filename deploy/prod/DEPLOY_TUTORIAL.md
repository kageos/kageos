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

go run ./cmd/aosctl init --base-url http://your-ip-or-domain
go run ./cmd/aosctl doctor --config deploy/prod/aos.yaml
go run ./cmd/aosctl up --config deploy/prod/aos.yaml
go run ./cmd/aosctl verify --config deploy/prod/aos.yaml
```

`aosctl init` 会生成 `deploy/prod/aos.yaml`，里面包含数据库密码、NATS 密码、JWT 密钥、system 初始密码等敏感配置，默认不入库。

## 改公网地址或 TLS

编辑 `deploy/prod/aos.yaml`：

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
  certs_host_dir: "./certs"
  cert_file: "/app/tls/fullchain.pem"
  key_file: "/app/tls/privkey.pem"
```

改完执行：

```bash
go run ./cmd/aosctl up --config deploy/prod/aos.yaml
```

## 常用命令

```bash
go run ./cmd/aosctl status --config deploy/prod/aos.yaml
go run ./cmd/aosctl logs --config deploy/prod/aos.yaml main
go run ./cmd/aosctl verify --config deploy/prod/aos.yaml
go run ./cmd/aosctl down --config deploy/prod/aos.yaml
go run ./cmd/aosctl uninstall --config deploy/prod/aos.yaml
go run ./cmd/aosctl uninstall --config deploy/prod/aos.yaml --purge-data --force
```

`down` 只停服务；`uninstall` 会移除 Compose 栈和可再生的 `.generated/`。来回测试想清数据库/对象存储/业务数据但不重建用户应用基础镜像，用 `--purge-data --force`，它默认保留 `/data/kageos/podman_storage`。

## 生成物

生产部署只维护 `aosctl` 入口；如需变更配置，修改 `deploy/prod/aos.yaml` 后重新执行 `aosctl up`。
