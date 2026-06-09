# 一次性用户应用基础镜像构建器。
# kagectl up 会先启动基础设施，再运行该镜像把 kagebase 构建进共享 Podman 存储，
# 最后才构建并启动 main。

FROM docker.io/library/debian:bookworm-slim

ARG APT_USE_MIRROR=1
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai
ENV LANG=C.UTF-8

COPY deploy/base/scripts/configure-debian-apt-mirror.sh /usr/local/bin/configure-debian-apt-mirror
RUN chmod +x /usr/local/bin/configure-debian-apt-mirror && /usr/local/bin/configure-debian-apt-mirror
RUN apt-get update && apt-get install -y --no-install-recommends \
    bash ca-certificates tzdata \
    podman buildah fuse-overlayfs slirp4netns crun \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && rm -rf /var/lib/apt/lists/*

ARG USE_CN_REGISTRY_MIRROR=1
COPY deploy/prod/containers/registries.conf.d/000-docker-io-mirror.conf /tmp/000-docker-io-mirror.conf
RUN mkdir -p /etc/containers /var/lib/containers/storage /run/podman /run/containers/storage \
    && printf '[containers]\nnetns = "host"\n' > /etc/containers/containers.conf \
    && printf '%s\n' '[storage]' 'driver = "overlay"' 'runroot = "/run/containers/storage"' 'graphroot = "/var/lib/containers/storage"' > /etc/containers/storage.conf \
    && printf 'unqualified-search-registries = ["docker.io"]\n' > /etc/containers/registries.conf \
    && mkdir -p /etc/containers/registries.conf.d \
    && if [ "${USE_CN_REGISTRY_MIRROR}" = "1" ]; then \
         cp /tmp/000-docker-io-mirror.conf /etc/containers/registries.conf.d/; \
       fi

WORKDIR /app

COPY deploy/base/images/app-base/ /app/app-base/
COPY deploy/prod/entrypoint-app-base.sh /app/entrypoint-app-base.sh
RUN chmod +x /app/entrypoint-app-base.sh

ENTRYPOINT ["/app/entrypoint-app-base.sh"]
