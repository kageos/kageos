# 单机生产部署教程

这份文档只讲“怎么在一台新 Linux 服务器上把 `deploy/prod` 跑起来”。

如果你想看依赖关系和启动链路，去看 [DEPLOYMENT_FLOW.md](DEPLOYMENT_FLOW.md)。  
如果你想看完整参数说明，去看 [README.md](README.md)。

## 0. 先确认前提

- 宿主机必须是 Linux。
- 已安装 `podman compose` 或 `docker compose`。
- 当前执行部署的用户需要能写 `/data`，因为持久化目录固定为 `/data/ai-agent-os`。
- `80` 端口必须空闲；如果你要容器自己做 HTTPS，`443` 也必须空闲。
- 如果宿主机已有 `nginx`，`bash build.sh up` 会尝试停它，最好使用有 `sudo` 权限的账号。

## 1. 拉代码并进入目录

```bash
git clone <your-repo-url>
cd ai-agent-os/deploy/prod
```

## 2. 选择初始化方式

你只需要先决定一件事：主镜像是“目标机构建”还是“目标机只拉镜像”。

### 方式 A：目标机本地构建主镜像

适合还没有正式镜像发布链、或者你就准备在目标机本地编译的场景。

```bash
bash build.sh init
```

这一步会做：

- 初始化 `.env`
- 自动生成缺失的随机密钥
- 预拉取 `mysql / nats / minio`
- 本地构建 `MAIN_IMAGE`
- 在和 `main` 相同的 Podman 存储里初始化 `APP_BASE_IMAGE`

### 方式 B：目标机只拉已发布主镜像

适合正式生产、固定 tag、可回滚场景。

先在 `.env` 里加上你的主镜像 tag，例如：

```env
MAIN_IMAGE="registry.example.com/agentos/agentos-main:v1.2.3"
```

然后执行：

```bash
bash build.sh init --image
```

这一步不会在目标机构建主镜像，只会：

- 初始化 `.env`
- 预拉取 `mysql / nats / minio`
- 拉取 `MAIN_IMAGE`
- 在和 `main` 相同的 Podman 存储里初始化 `APP_BASE_IMAGE`

## 3. 填写 `.env`

初始化后，最少要看这两个配置：

- `CANONICAL_BASE_URL`
- `TLS_MODE`

### 场景 1：先用 HTTP 跑起来

```env
CANONICAL_BASE_URL="http://your-ip-or-domain"
TLS_MODE="http"
```

### 场景 2：前面已有 LB / CDN / Ingress 做 HTTPS

```env
CANONICAL_BASE_URL="https://your-domain"
TLS_MODE="external"
```

### 场景 3：容器自己做正式 HTTPS

```env
CANONICAL_BASE_URL="https://your-domain"
TLS_MODE="redirect"
TLS_CERTS_HOST_DIR="./certs"
TLS_CERT_FILE="/app/tls/fullchain.pem"
TLS_KEY_FILE="/app/tls/privkey.pem"
```

如果是 `TLS_MODE=https` 或 `redirect`，要把证书文件放到宿主机证书目录里。默认就是：

```text
deploy/prod/certs/fullchain.pem
deploy/prod/certs/privkey.pem
```

## 4. 启动

```bash
bash build.sh up
```

`up` 现在只负责启动，不再构建镜像。它会：

- 校验 `.env`
- 停宿主机 `nginx`（如果有）
- 检查端口
- 执行 `compose up -d --no-build`

如果你想在启动前先看一份显式预检报告，再手动执行：

```bash
bash build.sh doctor
```

它会额外把 Linux / Compose、`/data/ai-agent-os` 写权限、TLS 文件、镜像准备状态等都列出来。主路径不强制依赖它。

## 5. 启动后验证

```bash
bash build.sh verify
bash build.sh status
bash build.sh logs main
```

如果你只是先验首页，访问：

- `http://your-ip-or-domain`
- 或 `https://your-domain`

## 6. 后续常用操作

### 只改了 `.env`

```bash
bash build.sh up
```

### 本地重建并更新主服务

```bash
bash build.sh update
```

### 拉新主镜像并更新主服务

```bash
bash build.sh update --image
```

### 只重建用户应用基础镜像

```bash
bash build.sh build-app-base --no-cache
```

这个命令会在和 `main` 相同的运行环境里重建 `APP_BASE_IMAGE`。

## 7. 最容易卡住的地方

- `/data` 不可写，导致 `/data/ai-agent-os` 建不起来。
- 你直接跑了 `up`，但没先执行 `init` / `init --image`。
- `TLS_MODE=redirect`，但 `CANONICAL_BASE_URL` 还是 `http://`。
- 证书文件路径写对了，但宿主机目录下没有 `fullchain.pem` / `privkey.pem`。
- 你以为宿主机提前 `podman build` 一次 `APP_BASE_IMAGE` 就够了，但它没进入 `main` 那套 Podman 存储。

## 8. 推荐命令顺序

### 本地构建主镜像

```bash
cd deploy/prod
bash build.sh init
# 编辑 .env
bash build.sh up
bash build.sh verify
```

### 拉已发布主镜像

```bash
cd deploy/prod
# 可先编辑 .env，写入 MAIN_IMAGE
bash build.sh init --image
# 继续完善 .env
bash build.sh up
bash build.sh verify
```
