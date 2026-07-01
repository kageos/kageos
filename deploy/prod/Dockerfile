# 生产单机镜像：core-server + 内置 Nginx + Podman
# 构建（仓库根）：docker build -f deploy/prod/Dockerfile -t localhost/kageos-main:latest .

# ========== Go 编译 ==========
FROM docker.io/library/golang:1.25.10-bookworm AS gobuilder
WORKDIR /src
# 国内构建：APT / Go 模块 / npm 可加速见 README；海外构建传 --build-arg APT_USE_MIRROR=0
ARG APT_USE_MIRROR=1
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn
ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=${GOSUMDB}
# 共享 helper：把 Debian 官方源切到阿里云镜像，避免国内构建首包慢或 TLS 校验卡住。
COPY deploy/base/scripts/configure-debian-apt-mirror.sh /usr/local/bin/configure-debian-apt-mirror
RUN chmod +x /usr/local/bin/configure-debian-apt-mirror && /usr/local/bin/configure-debian-apt-mirror
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc libc6-dev libgpgme-dev libdevmapper-dev pkg-config \
    && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOCACHE=/tmp/go-build CGO_ENABLED=1 GOOS=linux go build \
    -tags "exclude_graphdriver_btrfs" \
    -ldflags="-s -w" \
    -o /out/core-server \
    ./core/cmd/main/main.go \
    && rm -rf /tmp/go-build /root/.cache/go-build

# ========== Web 静态 ==========
FROM docker.io/library/node:20-bookworm AS webbuilder
WORKDIR /src/web
ARG NPM_REGISTRY=https://registry.npmmirror.com
ENV NPM_CONFIG_REGISTRY=${NPM_REGISTRY}
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build-only

# ========== 运行 ==========
FROM docker.io/library/debian:bookworm-slim

ARG APT_USE_MIRROR=1
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai
ENV LANG=C.UTF-8

# 共享 helper：运行阶段依然沿用同一份 APT mirror 配置逻辑。
COPY deploy/base/scripts/configure-debian-apt-mirror.sh /usr/local/bin/configure-debian-apt-mirror
RUN chmod +x /usr/local/bin/configure-debian-apt-mirror && /usr/local/bin/configure-debian-apt-mirror
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata curl netcat-openbsd \
    nginx gettext-base \
    libgpgme11 libdevmapper1.02.1 git \
    podman buildah fuse-overlayfs slirp4netns crun \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && rm -rf /var/lib/apt/lists/*

ARG USE_CN_REGISTRY_MIRROR=1
COPY deploy/prod/containers/registries.conf.d/000-docker-io-mirror.conf /tmp/000-docker-io-mirror.conf
RUN mkdir -p /etc/kageos /etc/containers /var/lib/containers/storage /run/podman /run/containers/storage \
    && : > /etc/kageos/prod-compose-bundle \
    && printf '[containers]\nnetns = "host"\n' > /etc/containers/containers.conf \
    && printf '%s\n' '[storage]' 'driver = "overlay"' 'runroot = "/run/containers/storage"' 'graphroot = "/var/lib/containers/storage"' > /etc/containers/storage.conf \
    && printf 'unqualified-search-registries = ["docker.io"]\n' > /etc/containers/registries.conf \
    && mkdir -p /etc/containers/registries.conf.d \
    && if [ "${USE_CN_REGISTRY_MIRROR}" = "1" ]; then \
         cp /tmp/000-docker-io-mirror.conf /etc/containers/registries.conf.d/; \
       fi

COPY --from=gobuilder /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV GOCACHE="/tmp/go-cache"

WORKDIR /app

# Go 二进制与运行时源码
COPY --from=gobuilder /out/core-server /app/core-server
COPY --from=gobuilder /src/go.mod /src/go.sum /app/
COPY --from=gobuilder /go/pkg/mod /go/pkg/mod

# 共享 canonical 资源：用户应用运行时基础镜像构建上下文
COPY deploy/base/images/app-base/ /app/app-base/

# 内置系统能力种子：启动时用于初始化 /system/tools，不是运行期 namespace 数据。
COPY core/app-server/system-seed/ /app/core/app-server/system-seed/

# prod 专属资源：配置模板、healthcheck、Nginx、entrypoint
COPY deploy/prod/config/template/ /app/config.prod.template/
COPY deploy/prod/health/ /app/health/
COPY deploy/prod/nginx/ /app/deploy/prod/nginx/
COPY deploy/prod/entrypoint-common.sh /app/entrypoint-common.sh
COPY deploy/prod/entrypoint-main.sh /app/entrypoint-main.sh
COPY deploy/prod/entrypoint-app-base.sh /app/entrypoint-app-base.sh
RUN chmod +x /app/entrypoint-common.sh /app/entrypoint-main.sh /app/entrypoint-app-base.sh /app/health/*.sh

# 运行期目录骨架与静态站点
RUN mkdir -p /app/deploy /app/logs /app/namespace \
    /app/data/runtime/app-runtime /app/data/tmp \
    && mkdir -p /var/www/web \
    && test -f "/app/core/app-server/system-seed/system/tools/官方工具.capability-bundle.json" \
    && test -z "$(find /app/logs -mindepth 1 -print -quit)" \
    && test -z "$(find /app/namespace -mindepth 1 -print -quit)" \
    && rm -f /etc/nginx/sites-enabled/default

COPY --from=webbuilder /src/web/dist /var/www/web/dist

EXPOSE 80 443 9090

ENTRYPOINT ["/app/entrypoint-main.sh"]
