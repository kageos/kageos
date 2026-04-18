# Prod Quick Start

只看这份就够了。

## 你只需要记住 2 个命令

```bash
bash build.sh init
bash build.sh up
```

## 首次部署

```bash
git clone <your-repo-url>
cd ai-agent-os/deploy/prod

bash build.sh init
```

编辑 `.env`，先只填这两个：

```env
CANONICAL_BASE_URL="http://your-ip-or-domain"
TLS_MODE="http"
```

然后启动：

```bash
bash build.sh up
bash build.sh verify
```

访问：

```text
http://your-ip-or-domain
```

## 如果你已经有发布镜像

```bash
bash build.sh init --image
bash build.sh up
```

## 以后升级

```bash
git pull
cd deploy/prod
bash build.sh update
```

如果中间件没在运行，直接用：

```bash
bash build.sh up
```

## 最常用

```bash
bash build.sh status
bash build.sh logs main
bash build.sh verify
bash build.sh down
```

## 就这几个前提

- Linux
- 有 `sudo` 或 root 权限
- `80` 端口空闲
- `/data` 可写

想看完整说明，再看 [README.md](README.md)。
