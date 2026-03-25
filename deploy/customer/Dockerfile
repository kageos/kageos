# 客户主站一键部署：core-server + 内置 Nginx(8080) + Podman（需 --privileged）
# 构建（仓库根）：docker build -f deploy/customer/Dockerfile -t ai-agent-os-main:latest .

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
ENV AI_AGENT_OS_ROOT=/app

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
    nginx gettext-base \
    libgpgme11 libdevmapper1.02.1 git \
    podman buildah fuse-overlayfs slirp4netns crun \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /etc/containers /var/lib/containers/storage /run/podman \
    && printf '[containers]\nnetns = "host"\n' > /etc/containers/containers.conf \
    && printf '[storage]\ndriver = "overlay"\n' > /etc/containers/storage.conf \
    && printf 'unqualified-search-registries = ["docker.io"]\n' > /etc/containers/registries.conf

COPY --from=gobuilder /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV GOCACHE="/tmp/go-cache"

WORKDIR /app

COPY --from=gobuilder /out/core-server /app/core-server
COPY --from=gobuilder /src/go.mod /src/go.sum /app/
COPY --from=gobuilder /src/sdk/ /app/sdk/
COPY --from=gobuilder /src/pkg/ /app/pkg/
COPY --from=gobuilder /src/dto/ /app/dto/
COPY --from=gobuilder /src/core/ /app/core/
COPY --from=gobuilder /src/enterprise_impl/ /app/enterprise_impl/
COPY --from=gobuilder /src/enterprise/ /app/enterprise/
COPY --from=gobuilder /go/pkg/mod /go/pkg/mod

COPY deploy/podman/Dockerfile.app-base /app/app-base/Dockerfile
COPY build/start.sh /app/app-base/scripts/start.sh

# 配置模板（每次启动从模板复制再 sed，避免重启破坏占位符）
COPY deploy/customer/config/prod/ /app/config.prod.template/
COPY deploy/customer/nginx/default.conf.template /app/deploy/customer/nginx/default.conf.template
COPY deploy/customer/entrypoint-main.sh /app/entrypoint-main.sh
RUN chmod +x /app/entrypoint-main.sh

RUN mkdir -p /app/deploy /app/logs /app/namespace /app/data \
    && mkdir -p /var/www/web \
    && rm -f /etc/nginx/sites-enabled/default

COPY --from=webbuilder /src/web/dist /var/www/web/dist

EXPOSE 8080 9090

ENTRYPOINT ["/app/entrypoint-main.sh"]
