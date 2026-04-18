# 一分钟部署

只看这份就够了。

前提：

- Linux
- 有 `sudo` 或 root 权限；`bash build.sh init` 缺少宿主机 `podman` 时会自动安装，Ubuntu 腾讯云源 404 时会自动切官方源重试一次
- `/data` 可写
- `80` 端口空闲；如果容器自己做 HTTPS，`443` 也要空闲

## 最短路径

```bash
git clone <your-repo-url>
cd ai-agent-os/deploy/prod

bash build.sh init          # 已有发布镜像就改成: bash build.sh init --image
# 编辑 .env，只看 CANONICAL_BASE_URL 和 TLS_MODE

bash build.sh up
bash build.sh verify
```

## `.env` 最小示例

先用 HTTP 跑起来：

```env
CANONICAL_BASE_URL="http://your-ip-or-domain"
TLS_MODE="http"
```

前面已有 LB / CDN / Ingress 做 HTTPS：

```env
CANONICAL_BASE_URL="https://your-domain"
TLS_MODE="external"
```

容器自己做 HTTPS：

```env
CANONICAL_BASE_URL="https://your-domain"
TLS_MODE="redirect"
TLS_CERTS_HOST_DIR="./certs"
TLS_CERT_FILE="/app/tls/fullchain.pem"
TLS_KEY_FILE="/app/tls/privkey.pem"
```

证书默认放这里：

```text
deploy/prod/certs/fullchain.pem
deploy/prod/certs/privkey.pem
```

## 最常用命令

```bash
bash build.sh init
bash build.sh init --image
bash build.sh up
bash build.sh verify
bash build.sh logs main
bash build.sh update
bash build.sh update --image
```

## 出问题先看这几个

- `bash build.sh doctor`
- `/data/ai-agent-os` 建不起来
- 直接跑了 `up`，但没先跑 `init`
- `TLS_MODE=redirect`，但 `CANONICAL_BASE_URL` 还是 `http://`
- 证书文件不在 `deploy/prod/certs/`

想看完整说明，再去看 [README.md](README.md)。
