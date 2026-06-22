# Kageos All-in-One local image.
# Build from repository root:
#   docker build -f deploy/aio/Dockerfile -t kageos:latest -t qiayanai/kageos:latest .
#
# This image keeps the outside UX to one `docker run`, then uses the embedded
# Podman runtime to start MySQL / NATS / MinIO inside the same outer container.
# It is intended for local evaluation and single-VM demos, not production.

# ========== Go build ==========
FROM docker.io/library/golang:1.25-bookworm AS gobuilder
WORKDIR /src

ARG APT_USE_MIRROR=1
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn
ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=${GOSUMDB}

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

# ========== Web static build ==========
FROM docker.io/library/node:20-bookworm AS webbuilder
WORKDIR /src/web

ARG NPM_REGISTRY=https://registry.npmmirror.com
ENV NPM_CONFIG_REGISTRY=${NPM_REGISTRY}

COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build-only

# ========== Runtime ==========
FROM docker.io/library/debian:bookworm-slim

ARG APT_USE_MIRROR=1
ARG KAGEOS_DEFAULT_APP_BASE_IMAGE=docker.io/qiayanai/kagebase:latest
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai
ENV LANG=C.UTF-8
ENV KAGEOS_DEFAULT_APP_BASE_IMAGE=${KAGEOS_DEFAULT_APP_BASE_IMAGE}

COPY deploy/base/scripts/configure-debian-apt-mirror.sh /usr/local/bin/configure-debian-apt-mirror
RUN chmod +x /usr/local/bin/configure-debian-apt-mirror && /usr/local/bin/configure-debian-apt-mirror
RUN apt-get update && apt-get install -y --no-install-recommends \
    bash ca-certificates tzdata curl netcat-openbsd \
    nginx gettext-base \
    libgpgme11 libdevmapper1.02.1 git \
    podman buildah fuse-overlayfs slirp4netns crun \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && rm -rf /var/lib/apt/lists/*

ARG USE_CN_REGISTRY_MIRROR=1
COPY deploy/prod/containers/registries.conf.d/000-docker-io-mirror.conf /tmp/000-docker-io-mirror.conf
RUN mkdir -p /etc/kageos /etc/containers /var/lib/containers/storage /run/podman /run/containers/storage \
    && : > /etc/kageos/aio-bundle \
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
COPY --from=gobuilder /src/go.mod /src/go.sum /app/
COPY --from=gobuilder /go/pkg/mod /go/pkg/mod

COPY deploy/base/images/app-base/ /app/app-base/

COPY deploy/prod/config/template/ /app/config.prod.template/
COPY deploy/prod/health/ /app/health/
COPY deploy/prod/nginx/ /app/deploy/prod/nginx/
COPY deploy/prod/entrypoint-common.sh /app/entrypoint-common.sh
COPY deploy/prod/entrypoint-main.sh /app/entrypoint-main.sh
COPY deploy/prod/entrypoint-app-base.sh /app/entrypoint-app-base.sh
COPY deploy/aio/entrypoint-aio.sh /app/entrypoint-aio.sh
RUN chmod +x \
    /app/entrypoint-common.sh \
    /app/entrypoint-main.sh \
    /app/entrypoint-app-base.sh \
    /app/entrypoint-aio.sh \
    /app/health/*.sh

RUN mkdir -p /app/deploy /app/logs /app/namespace \
    /app/data/runtime/app-runtime /app/data/tmp \
    /var/lib/kageos \
    && mkdir -p /var/www/web \
    && test -z "$(find /app/namespace -mindepth 1 -print -quit)" \
    && rm -f /etc/nginx/sites-enabled/default

COPY --from=webbuilder /src/web/dist /var/www/web/dist

VOLUME ["/var/lib/kageos"]
EXPOSE 80 443 9090

ENTRYPOINT ["/app/entrypoint-aio.sh"]
