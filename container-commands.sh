#!/usr/bin/env bash

# 说明：
# - 这是容器/部署常用命令清单，不建议整文件执行。
# - 推荐在编辑器里按需选择某一行执行，或复制单行到终端执行。
# - dev 默认 app-base 镜像 tag：agentos-app-runtime-base:latest。
# - prod 命令需要在部署机器上执行，并确保 deploy/prod/.env 已初始化。

echo "container-commands.sh 是命令清单，请在编辑器中选择单行执行，不要整文件运行。"
exit 0

# Dev infra
# 启动 dev 依赖服务：MySQL / NATS / MinIO，并幂等初始化数据库。
bash deploy/dev/scripts/infra.sh up -d

# 停止 dev 依赖服务。
bash deploy/dev/scripts/infra.sh down

# 查看 dev 依赖服务状态。
bash deploy/dev/scripts/infra.sh ps

# 持续查看 dev 依赖服务日志。
bash deploy/dev/scripts/infra.sh logs -f

# Dev app-base image
# 构建 dev 用户应用运行时基础镜像；如果同名 tag 已存在，脚本会跳过。
bash deploy/dev/scripts/build-app-base.sh

# 强制重建 dev app-base 镜像，但允许复用 Docker/Podman layer 缓存。
bash deploy/dev/scripts/build-app-base.sh --force

# 强制且不使用缓存重建 dev app-base 镜像；改了系统包、字体、Python 依赖时推荐用这一行。
bash deploy/dev/scripts/build-app-base.sh --force --no-cache

# 指定 app-base 镜像 tag 后强制无缓存重建；当 dev 配置改了 base_image 时使用。
APP_BASE_IMAGE="agentos-app-runtime-base:latest" bash deploy/dev/scripts/build-app-base.sh --force --no-cache

# Dev backend
# 启动 dev 后端；如果缺 app-base 镜像，会先尝试自动构建。
bash deploy/dev/scripts/run-backend.sh

# 启动 dev 后端并跳过 app-base 镜像检查；确认镜像已存在时可用。
AOS_SKIP_APP_BASE_BUILD=1 bash deploy/dev/scripts/run-backend.sh

# Image checks
# 检查本地 Podman 是否存在 dev app-base 镜像。
podman image exists agentos-app-runtime-base:latest

# 列出本地 app-base 镜像。
podman images | grep agentos-app-runtime-base

# 在 app-base 镜像内验证常用 Python 包是否可 import。
podman run --rm --entrypoint /bin/sh agentos-app-runtime-base:latest -lc 'python3 -c "import pandas, numpy, matplotlib, openpyxl, xlsxwriter, pptx, plotly, pyecharts, bs4, yaml, qrcode, barcode, xlrd, xlwt, aiohttp, toml, snownlp, tabulate, arrow, dateutil, wordcloud, pymysql; print(\"python packages OK\")"'

# 在 app-base 镜像内验证中文字体和 matplotlib 配置。
podman run --rm --entrypoint /bin/sh agentos-app-runtime-base:latest -lc 'fc-match "Noto Sans CJK SC"; python3 -c "import matplotlib; print(matplotlib.matplotlib_fname())"'

# Prod local deploy
# 生产本地构建初始化：准备 .env、中间件镜像、主镜像、APP_BASE_IMAGE。
cd deploy/prod && bash build.sh init

# 启动生产服务；不会重建镜像。
cd deploy/prod && bash build.sh up

# 更新生产 main/scheduler/backup；本地重建主镜像，不重启中间件。
cd deploy/prod && bash build.sh update

# 在与 main 相同运行环境里无缓存重建生产 APP_BASE_IMAGE。
cd deploy/prod && bash build.sh build-app-base --no-cache

# 仅重启生产 main 服务。
cd deploy/prod && bash build.sh restart-main

# 仅重启生产 scheduler 服务。
cd deploy/prod && bash build.sh restart-scheduler

# 执行生产健康检查。
cd deploy/prod && bash build.sh verify

# 查看生产 main 日志。
cd deploy/prod && bash build.sh logs main

# 查看生产服务状态。
cd deploy/prod && bash build.sh status

# Prod image mode
# 生产固定镜像模式初始化：拉取 MAIN_IMAGE，然后准备 APP_BASE_IMAGE。
cd deploy/prod && bash build.sh init --image

# 生产固定镜像模式更新：拉取 MAIN_IMAGE 并重建 main/scheduler/backup 服务。
cd deploy/prod && bash build.sh update --image
