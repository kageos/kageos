# 生产单机一键部署：core-server + 内置 Nginx(8080) + Podman（需 --privileged）
# 构建（仓库根）：docker build -f deploy/prod/Dockerfile -t ai-agent-os-main:latest .

# ========== Go 编译 ==========
FROM golang:1.24-bookworm AS gobuilder
WORKDIR /src
# 国内构建：APT / Go 模块 / npm 可加速见 README；海外构建传 --build-arg APT_USE_MIRROR=0
ARG APT_USE_MIRROR=1
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn
ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=${GOSUMDB}
RUN set -eux; \
    if [ "${APT_USE_MIRROR}" = "1" ]; then \
      if [ -f /etc/apt/sources.list.d/debian.sources ]; then \
        sed -i \
          -e 's|http://deb.debian.org/debian|http://mirrors.aliyun.com/debian|g' \
          -e 's|https://deb.debian.org/debian|http://mirrors.aliyun.com/debian|g' \
          -e 's|http://security.debian.org/debian-security|http://mirrors.aliyun.com/debian-security|g' \
          -e 's|https://security.debian.org/debian-security|http://mirrors.aliyun.com/debian-security|g' \
          /etc/apt/sources.list.d/debian.sources; \
      elif [ -f /etc/apt/sources.list ]; then \
        sed -i 's|deb.debian.org|mirrors.aliyun.com|g; s|security.debian.org|mirrors.aliyun.com/debian-security|g' /etc/apt/sources.list; \
      fi; \
    fi
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc libc6-dev libgpgme-dev libdevmapper-dev pkg-config \
    && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build \
    -tags "exclude_graphdriver_btrfs" \
    -ldflags="-s -w" \
    -o /out/core-server \
    ./core/cmd/main/main.go
RUN CGO_ENABLED=1 GOOS=linux go build \
    -tags "exclude_graphdriver_btrfs" \
    -ldflags="-s -w" \
    -o /out/backup-server \
    ./core/backup-service/cmd/app/main.go

# ========== Web 静态 ==========
FROM node:20-bookworm AS webbuilder
WORKDIR /src/web
ARG NPM_REGISTRY=https://registry.npmmirror.com
ENV NPM_CONFIG_REGISTRY=${NPM_REGISTRY}
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build-only

# ========== 运行 ==========
FROM debian:bookworm-slim

ARG APT_USE_MIRROR=1
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai
ENV LANG=C.UTF-8
ENV APP_ENV=prod

RUN set -eux; \
    if [ "${APT_USE_MIRROR}" = "1" ]; then \
      if [ -f /etc/apt/sources.list.d/debian.sources ]; then \
        sed -i \
          -e 's|http://deb.debian.org/debian|http://mirrors.aliyun.com/debian|g' \
          -e 's|https://deb.debian.org/debian|http://mirrors.aliyun.com/debian|g' \
          -e 's|http://security.debian.org/debian-security|http://mirrors.aliyun.com/debian-security|g' \
          -e 's|https://security.debian.org/debian-security|http://mirrors.aliyun.com/debian-security|g' \
          /etc/apt/sources.list.d/debian.sources; \
      elif [ -f /etc/apt/sources.list ]; then \
        sed -i 's|deb.debian.org|mirrors.aliyun.com|g; s|security.debian.org|mirrors.aliyun.com/debian-security|g' /etc/apt/sources.list; \
      fi; \
    fi
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata curl netcat-openbsd \
    nginx gettext-base default-mysql-client restic \
    libgpgme11 libdevmapper1.02.1 git \
    podman buildah fuse-overlayfs slirp4netns crun \
    && case "$(dpkg --print-architecture)" in \
         amd64) mc_arch="linux-amd64" ;; \
         arm64) mc_arch="linux-arm64" ;; \
         *) echo "ERROR: unsupported architecture for MinIO mc: $(dpkg --print-architecture)" >&2; exit 1 ;; \
       esac \
    && curl --fail --location --output /usr/local/bin/mc "https://dl.min.io/client/mc/release/${mc_arch}/mc" \
    && chmod +x /usr/local/bin/mc \
    && /usr/local/bin/mc --help >/dev/null \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && rm -rf /var/lib/apt/lists/*

ARG USE_CN_REGISTRY_MIRROR=1
COPY deploy/prod/containers/registries.conf.d/000-docker-io-mirror.conf /tmp/000-docker-io-mirror.conf
RUN mkdir -p /etc/ai-agent-os /etc/containers /var/lib/containers/storage /run/podman /run/containers/storage \
    && : > /etc/ai-agent-os/prod-compose-bundle \
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

COPY --from=gobuilder /out/core-server /app/core-server
COPY --from=gobuilder /out/backup-server /app/backup-server
COPY --from=gobuilder /src/go.mod /src/go.sum /app/
COPY --from=gobuilder /src/sdk/ /app/sdk/
COPY --from=gobuilder /src/pkg/ /app/pkg/
COPY --from=gobuilder /src/dto/ /app/dto/
COPY --from=gobuilder /src/core/ /app/core/
COPY --from=gobuilder /src/enterprise_impl/ /app/enterprise_impl/
COPY --from=gobuilder /src/enterprise/ /app/enterprise/
COPY --from=gobuilder /go/pkg/mod /go/pkg/mod

COPY deploy/base/images/app-base/Dockerfile /app/app-base/Dockerfile
COPY deploy/base/images/app-base/start.sh /app/app-base/start.sh

# 配置模板（每次启动从模板复制再 sed，避免重启破坏占位符）
COPY deploy/prod/config/template/ /app/config.prod.template/
COPY deploy/prod/nginx/default.conf.template /app/deploy/prod/nginx/default.conf.template
COPY deploy/prod/entrypoint-main.sh /app/entrypoint-main.sh
COPY deploy/prod/entrypoint-backup.sh /app/entrypoint-backup.sh
RUN chmod +x /app/entrypoint-main.sh /app/entrypoint-backup.sh

RUN mkdir -p /app/deploy /app/logs /app/namespace \
    /app/data/runtime/app-runtime /app/data/license /app/data/backup/repo \
    /app/data/backup/state /app/data/backup/staging /app/data/tmp \
    && mkdir -p /var/www/web \
    && rm -f /etc/nginx/sites-enabled/default

COPY --from=webbuilder /src/web/dist /var/www/web/dist

EXPOSE 8080 9090 19088

ENTRYPOINT ["/app/entrypoint-main.sh"]
