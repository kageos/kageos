# Kageos All-in-One Image

`deploy/aio` 是本地体验/单机演示镜像。外部只需要运行一个容器，容器内部会用 Podman 拉起 MySQL、NATS、MinIO，并启动 Kageos 主服务。

用户应用运行时依赖单独发布为 `qiayanai/kagebase`。AIO 首次启动时会优先拉取匹配版本的 `kagebase`；如果拉取失败，会 fallback 到本地构建。

> 这个镜像用于 Ubuntu VM、本地试用和演示，不建议作为正式生产部署入口。生产环境优先使用 `install.sh` / `kagectl` / Compose。

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

首次启动会在容器内部拉取并初始化 MySQL、NATS、MinIO，并准备用户应用基础镜像。
如果使用官方发布镜像，AIO 会优先拉取 `docker.io/qiayanai/kagebase:<version>`，通常比现场构建快很多；只有拉取失败时才会 fallback 到本地构建。

查看日志：

```bash
docker logs -f kageos
```

启动成功后日志会输出最终成功块：

```text
Kageos started successfully
Access URL:
  http://localhost:8080

Login:
  Username: system
  Password: <generated password>
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
| `KAGEOS_AIO_RECREATE_INFRA` | `0` | Set to `1` to recreate inner MySQL/NATS/MinIO containers on next boot. |
| `KAGEOS_AIO_MYSQL_IMAGE` | `docker.io/library/mysql:8.0` | Inner MySQL image. |
| `KAGEOS_AIO_NATS_IMAGE` | `docker.io/library/nats:2.10-alpine` | Inner NATS image. |
| `KAGEOS_AIO_MINIO_IMAGE` | `docker.io/minio/minio:RELEASE.2025-09-07T16-13-09Z` | Inner MinIO image. |
| `KAGEOS_APP_BASE_IMAGE` | `docker.io/qiayanai/kagebase:<version>` in release images, `docker.io/qiayanai/kagebase:latest` in local builds | User app runtime base image. |
| `KAGEOS_APP_BASE_ACTION` | `ensure` | Use `rebuild` to rebuild the user app base image. |
| `KAGEOS_APP_BASE_PULL` | `1` | Pull `KAGEOS_APP_BASE_IMAGE` before falling back to local build. |
| `KAGEOS_APP_BASE_PULL_FALLBACK_BUILD` | `1` | Build locally if pulling the base image fails. Set to `0` to fail fast. |
| `KAGEOS_AIO_PRINT_SECRETS` | `1` | Print generated credentials in the final success log. Set to `0` to hide plaintext secrets and only print file paths. |
