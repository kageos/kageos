#!/usr/bin/env bash

# 说明：
# - 这是容器/部署常用命令清单，不建议整文件执行。
# - 推荐在编辑器里按需选择某一行执行，或复制单行到终端执行。
# - dev 默认 app-base 镜像 tag：agentos-app-runtime-base:latest。
# - prod 命令需要在部署机器上执行，并确保 deploy/prod/aos.yaml 已初始化。

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
bash deploy/base/scripts/build-app-base-image.sh

# 强制重建 dev app-base 镜像，但允许复用 Docker/Podman layer 缓存。
bash deploy/base/scripts/build-app-base-image.sh --force

# 强制且不使用缓存重建 dev app-base 镜像；改了系统包、字体、Python 依赖时推荐用这一行。
bash deploy/base/scripts/build-app-base-image.sh --force --no-cache

# 指定 app-base 镜像 tag 后强制无缓存重建；当 dev 配置改了 base_image 时使用。
APP_BASE_IMAGE="agentos-app-runtime-base:latest" bash deploy/base/scripts/build-app-base-image.sh --force --no-cache

# Dev backend
# 后端本地开发使用 GoLand 启动 core/cmd/main/main.go，并设置 APP_ENV=dev。
# 命令行临时启动时可用这一行。
APP_ENV=dev go run ./core/cmd/main

# Image checks
# 检查本地 Podman 是否存在 dev app-base 镜像。
podman image exists agentos-app-runtime-base:latest

# 列出本地 app-base 镜像。
podman images | grep agentos-app-runtime-base

# 在 app-base 镜像内验证常用 Python 包是否可 import。
podman run --rm --entrypoint /bin/sh agentos-app-runtime-base:latest -lc 'python3 -c "import pandas, numpy, matplotlib, openpyxl, xlsxwriter, pptx, plotly, pyecharts, bs4, yaml, qrcode, barcode, xlrd, xlwt, aiohttp, toml, snownlp, tabulate, arrow, dateutil, wordcloud, pymysql, pytesseract, yt_dlp; print(\"python packages OK\")" && yt-dlp --version'

# 在 app-base 镜像内验证常用 CLI 辅助工具。
podman run --rm --entrypoint /bin/sh agentos-app-runtime-base:latest -lc 'wget --version | head -n 1 && mediainfo --Version && 7z i | head -n 2 && rsync --version | head -n 1 && zstd --version'

# 在 app-base 镜像内验证 Tesseract 命令和中英文语言包。
podman run --rm --entrypoint /bin/sh agentos-app-runtime-base:latest -lc 'tesseract --version | head -n 1 && tesseract --list-langs | tee /tmp/tesseract-langs.txt && grep -x eng /tmp/tesseract-langs.txt && grep -x chi_sim /tmp/tesseract-langs.txt'

# 在 app-base 镜像内验证中文字体和 matplotlib 配置。
podman run --rm --entrypoint /bin/sh agentos-app-runtime-base:latest -lc 'fc-match "Noto Sans CJK SC"; python3 -c "import matplotlib; print(matplotlib.matplotlib_fname())"'

# Prod local deploy
# 生产配置初始化：生成 deploy/prod/aos.yaml。
go run ./cmd/aosctl init --base-url http://your-ip-or-domain

# 执行生产预检。
go run ./cmd/aosctl doctor --config deploy/prod/aos.yaml

# 本地构建主镜像并启动/更新生产服务。
go run ./cmd/aosctl up --config deploy/prod/aos.yaml

# 使用已发布主镜像启动/更新生产服务。
go run ./cmd/aosctl up --config deploy/prod/aos.yaml --image

# 执行生产健康检查。
go run ./cmd/aosctl verify --config deploy/prod/aos.yaml

# 查看生产 main 日志。
go run ./cmd/aosctl logs --config deploy/prod/aos.yaml main

# 查看生产服务状态。
go run ./cmd/aosctl status --config deploy/prod/aos.yaml

# 停止生产服务。
go run ./cmd/aosctl down --config deploy/prod/aos.yaml
