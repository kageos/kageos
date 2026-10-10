# kageos All-in-One Image

`deploy/aio` 是单容器部署镜像。外部只运行一个 kageos 容器，容器内部会用 Podman 拉起 MySQL、NATS、MinIO，并启动 kageos 主服务。

用户应用运行时依赖单独发布为 `qiayanai/kagebase`。AIO 首次启动时会先让平台达到可登录状态，再在后台拉取匹配版本的 `kagebase`；如果拉取失败，会 fallback 到本地构建。平台登录不再被重型工具镜像阻塞，但首次构建工作空间前仍需等待 `kagebase` 就绪。

> AIO 必须使用 Docker/Podman 的 bridge/slirp 网络，通过 `-p` 映射入口端口。不要使用 `--network host`，否则容器内部的 `9093`、`13306`、`14222`、`19000` 等监听会直接占用宿主机端口。

## Build

在仓库根目录执行：

```bash
docker build \
  -f deploy/aio/Dockerfile \
  -t kageos:latest \
  -t qiayanai/kageos:latest \
  .
```

国内网络构建默认使用项目里的 Debian / npm / Go 镜像源配置；如果在海外构建，可以关闭国内 APT mirror：

```bash
docker build \
  --build-arg APT_USE_MIRROR=0 \
  --build-arg GOPROXY=https://proxy.golang.org,direct \
  --build-arg GOSUMDB=sum.golang.org \
  --build-arg NPM_REGISTRY=https://registry.npmjs.org \
  --build-arg USE_CN_REGISTRY_MIRROR=0 \
  -f deploy/aio/Dockerfile \
  -t kageos:latest \
  -t qiayanai/kageos:latest \
  .
```

## Run With Docker

```bash
docker volume create kageos-data

docker run -d \
  --name kageos \
  --privileged \
  --restart unless-stopped \
  -p 8080:80 \
  -v kageos-data:/var/lib/kageos \
  -e CANONICAL_BASE_URL=http://localhost:8080 \
  qiayanai/kageos:latest
```

不要加 `--network host`。AIO 镜像启动时会默认检查外层网络是否隔离；检测到疑似 host 网络会直接退出，避免内部服务污染宿主机端口。

Ubuntu VM 里如果希望局域网其他机器访问，把 `CANONICAL_BASE_URL` 改成 VM 的 IP：

```bash
docker run -d \
  --name kageos \
  --privileged \
  --restart unless-stopped \
  -p 8080:80 \
  -v kageos-data:/var/lib/kageos \
  -e CANONICAL_BASE_URL=http://192.168.1.100:8080 \
  qiayanai/kageos:latest
```

通过 `install-prod.sh` 安装时，安装器先检查共享归档缓存、宿主机以及已有运行实例中的同名同版本镜像；缺失时才下载，MySQL、NATS、MinIO 默认使用 `m.daocloud.io`，kageos 自有镜像在 `--cn` 下使用阿里云仓库。下载显示容器引擎的逐层字节进度，失败立即终止。安装器将镜像保存到 `<install-dir>/image-cache` 并只读挂载到新实例，避免跨实例重复下载；不共享各实例的数据库或可写 Podman 存储。`latest` 也优先使用本地缓存，升级请使用实例管理器的更新命令。安装器会复用已有 kagebase 缓存；缓存缺失时保持后台准备，避免重型应用镜像下载阻塞平台登录。通过 `sudo kageos logs <instance-id>` 查看后台下载的逐层进度。

直接运行 AIO 容器时，首次启动会在容器内部拉取并初始化 MySQL、NATS、MinIO。平台 API 和登录页就绪后，用户应用基础镜像会继续在后台准备。
如果使用官方发布镜像，AIO 会优先拉取 `docker.io/qiayanai/kagebase:<version>`，通常比现场构建快很多；只有拉取失败时才会 fallback 到本地构建。

日志出现 `kageos started successfully` 后即可登录。需要创建或构建工作空间时，先等待日志出现 `后台用户应用基础镜像已就绪`。后台拉取失败不会关闭平台；修复网络后重启外层实例即可复用已下载层并重试。

MinIO 默认使用经 DaoCloud 下载验证的 Bitnami 历史固定版本 `2025.7.23-debian-12-r5`，直接以 `minio` 为入口运行，保持 `/data` 路径和原容器的 root 文件属主兼容。旧实例已有的 MinIO 容器优先复用，不自动替换镜像或迁移数据。官方 `minio/minio` 镜像源已不能匿名完整拉取，不能将缓存的 manifest list 当作可下载证明。

基础服务启动分为镜像准备、容器启动和协议就绪三个阶段。先启动 MySQL、NATS、MinIO 的全部容器，再等待就绪，让数据库初始化和其他服务启动重叠。MySQL 用真实登录和 `SELECT 1` 验证并幂等初始化所需数据库；NATS 验证鉴权及 PING/PONG；MinIO 验证 `/minio/health/ready`。三个容器均配置 `unless-stopped` 重启策略。容器退出立即报错，诊断包含状态、退出码、OOM 信息及脱敏日志。默认就绪时限分别为 MySQL 600 秒、NATS 120 秒、MinIO 180 秒，可通过运行环境变量 `KAGEOS_AIO_MYSQL_READY_TIMEOUT`、`KAGEOS_AIO_NATS_READY_TIMEOUT`、`KAGEOS_AIO_MINIO_READY_TIMEOUT` 调整（安装器使用 `--env-file` 传入）。网络探测和数据库查询都有单次时限。Podman API 必须通过实际 `_ping` 验证，core-server 的就绪时限按真实耗时计算。

本地回归检查：`python3 scripts/test-aio-image-cache.py`。使用本地已有镜像进行临时容器验收：`python3 scripts/test-aio-services.py`，不拉取镜像、不挂载业务数据，结束后清理准确的测试容器名。可通过 `TEST_MYSQL_IMAGE`、`TEST_NATS_IMAGE`、`TEST_MINIO_IMAGE` 指定本地测试镜像。安装器支持复用 DaoCloud 路径对应的同版本上游镜像缓存，不用旧的 `latest` 代替固定版本。

启动日志会用统一的 `==> [耗时] 阶段: 秒数 (状态)` 格式记录关键阶段，包括 MySQL、NATS、MinIO、Podman API、core-server 各服务、用户应用基础镜像，以及 AIO 容器启动到可登录的总耗时。排查慢启动时可执行 `docker logs kageos 2>&1 | grep '\[耗时\]'` 汇总查看；耗时日志不包含密码或密钥。

查看日志：

```bash
docker logs -f kageos
```

启动成功后日志会输出最终成功块：

```text
kageos started successfully
Access URL:
  http://localhost:8080

Login:
  Username: system
  Password: (hidden; see /var/lib/kageos/secrets/SYSTEM_USER_PASSWORD)
```

查看初始密码也可以直接读持久化文件：

```bash
docker exec kageos cat /var/lib/kageos/secrets/SYSTEM_USER_PASSWORD
```

访问：

```text
http://localhost:8080
```

默认初始用户是 `system`。

## 安装为本地桌面 Web 应用

容器就绪后，在 Chromium 系浏览器打开且始终使用同一个地址：

```text
http://localhost:8080
```

确认登录和接口正常，再使用浏览器的“安装应用”功能。`localhost` 属于浏览器认可的
本地安全上下文；不要在 `localhost`、`127.0.0.1` 和局域网 IP 之间切换后重复安装，
因为每个 origin 都会被识别成不同应用。Docker 示例已配置
`--restart unless-stopped`，Docker 服务恢复后会重新拉起 kageos；桌面图标本身不会
启动容器。

本地安装与 `https://app.kageos.com` 等线上安装可以共存。PWA 身份由 origin 和
manifest `id` 共同决定，API 始终通过各自 origin 的同源代理访问；Service Worker
只缓存前端静态资源，不缓存 API、认证或文件请求。

## Run With Podman

```bash
podman volume create kageos-data

podman run -d \
  --name kageos \
  --privileged \
  -p 8080:80 \
  -v kageos-data:/var/lib/kageos \
  -e CANONICAL_BASE_URL=http://localhost:8080 \
  qiayanai/kageos:latest
```

同样不要使用 `--network host`。如果需要宿主机 80/443 端口，使用 `-p 80:80` / `-p 443:443`。

## Network Model

AIO 有两层网络：

- 外层 Docker/Podman 容器必须是 bridge/slirp 网络，宿主机只通过 `-p` 暴露入口端口。
- 内层 Podman 容器共享外层 kageos 容器的网络命名空间，所以 MySQL/NATS/MinIO/用户 App 可以用 `127.0.0.1` 互通。

这意味着官方部署方式是：

```bash
docker run ... -p 8080:80 qiayanai/kageos:latest
```

而不是：

```bash
docker run ... --network host qiayanai/kageos:latest
```

如果误用 host 网络，`app-runtime` 的 `127.0.0.1:9093` 会变成宿主机的 `127.0.0.1:9093`，同一台机器上再次启动或残留旧进程时就会出现端口冲突。

## Push To Docker Hub

### Automatic Multi-Arch Release

正式发布推荐走 GitHub Actions。它会把 `linux/amd64` 和 `linux/arm64` 合成同一个 Docker Hub tag，用户不用选择架构。

在 GitHub 仓库的 `Settings` -> `Secrets and variables` -> `Actions` 配置：

```text
Variable:
  DOCKERHUB_USERNAME

Secret:
  DOCKERHUB_TOKEN
```

然后打版本 tag：

```bash
git tag v0.2.0
git push origin v0.2.0
```

同一个 tag 会触发两条发布：

```text
qiayanai/kagebase:0.2.0
qiayanai/kageos:0.2.0
```

`qiayanai/kageos:0.2.0` 内置默认值会指向 `docker.io/qiayanai/kagebase:0.2.0`。用户运行 `kageos` 时不需要手动选择架构，也不需要单独指定 `kagebase`。

发布完成后，用户只需要：

```bash
docker run -d \
  --name kageos \
  --privileged \
  --restart unless-stopped \
  -p 8080:80 \
  -v kageos-data:/var/lib/kageos \
  -e CANONICAL_BASE_URL=http://localhost:8080 \
  qiayanai/kageos:0.2.0
```

Docker 会自动按机器选择 `linux/amd64` 或 `linux/arm64`。

### Local Fallback

本地临时发布可以用：

```bash
scripts/release-docker.sh 0.2.0
```

这个脚本需要 Docker Buildx，并会推送：

```text
qiayanai/kageos:0.2.0
qiayanai/kageos:latest
```

### Manual Single-Arch Push

如果只是临时推当前机器架构，先登录：

```bash
docker login
```

再推当前本地镜像。注意这种方式不是多架构，只适合内部临时测试：

```bash
docker tag kageos:latest qiayanai/kageos:0.2.0-arm64
docker push qiayanai/kageos:0.2.0-arm64
```

多架构发布完成后，用户可以直接运行：

```bash
docker run -d \
  --name kageos \
  --privileged \
  --restart unless-stopped \
  -p 8080:80 \
  -v kageos-data:/var/lib/kageos \
  -e CANONICAL_BASE_URL=http://localhost:8080 \
  qiayanai/kageos:0.2.0
```

Docker Hub 上的公开镜像需要带命名空间，所以对外推荐 `qiayanai/kageos:latest`。如果只写 `docker run kageos`，Docker 会去找 Docker Official Images 里的 `library/kageos`，一般不是我们的仓库。

本地自己 build 之后也会有 `kageos:latest` 这个短 tag，所以在同一台机器上可以把最后一行改成 `kageos:latest`。

## Reset

只删容器，保留数据：

```bash
docker rm -f kageos
```

彻底重置：

```bash
docker rm -f kageos
docker volume rm kageos-data
```

## Useful Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `CANONICAL_BASE_URL` | `http://localhost:8080` | Browser-facing site URL. Set this to the Ubuntu VM IP when testing over LAN. |
| `KAGEOS_AIO_DATA_DIR` | `/var/lib/kageos` | Persistent data root inside the container. |
| `KAGEOS_AIO_RECREATE_INFRA` | `0` | Reuse existing inner MySQL/NATS/MinIO containers on restart. Set to `1` only when an explicit infrastructure recreation is required. |
| `KAGEOS_AIO_REQUIRE_BRIDGE` | `1` | Refuse to start when the outer container does not look like bridge/slirp networking. |
| `KAGEOS_AIO_ALLOW_HOST_NETWORK` | `0` | Emergency override for the network guard. Setting it to `1` allows host networking but may expose internal ports on the host. |
| `KAGEOS_AIO_CORE_READY_TIMEOUT` | `600` | Seconds to wait for the platform API after infrastructure startup. |
| `KAGEOS_AIO_MYSQL_IMAGE` | `m.daocloud.io/docker.io/library/mysql:8.0.45` | Inner MySQL image. |
| `KAGEOS_AIO_NATS_IMAGE` | `m.daocloud.io/docker.io/library/nats:2.10.29-alpine` | Inner NATS image. |
| `KAGEOS_AIO_MINIO_IMAGE` | `m.daocloud.io/docker.io/bitnamilegacy/minio:2025.7.23-debian-12-r5` | Inner MinIO image. |
| `KAGEOS_APP_BASE_IMAGE` | `docker.io/qiayanai/kagebase:<version>` in release images, `docker.io/qiayanai/kagebase:latest` in local builds | User app runtime base image. |
| `KAGEOS_APP_BASE_ACTION` | `ensure` | Use `rebuild` to rebuild the user app base image. |
| `KAGEOS_APP_BASE_PULL` | `1` | Pull `KAGEOS_APP_BASE_IMAGE` before falling back to local build. |
| `KAGEOS_APP_BASE_PULL_FALLBACK_BUILD` | `1` | Build locally if pulling the base image fails. Set to `0` to fail fast. |
| `KAGEOS_APP_BASE_BACKGROUND` | `1` | Start the platform first and prepare `kagebase` in the background. Set to `0` to retain the blocking startup behavior. |
| `KAGEOS_AIO_PRINT_SECRETS` | `0` | Hide plaintext secrets in the final success log. Set to `1` only for disposable local testing. |
