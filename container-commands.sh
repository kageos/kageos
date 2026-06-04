#!/usr/bin/env bash

# 说明：
# - 这是容器/部署常用命令清单，不建议整文件执行。
# - 推荐在编辑器里按需选择某一行执行，或复制单行到终端执行。
# - dev 默认 app-base 镜像 tag：kagebase:latest。
# - prod 命令需要在部署机器上执行，并确保 .kageos/prod/kage.yaml 已初始化。

echo "container-commands.sh 是命令清单，请在编辑器中选择单行执行，不要整文件运行。"
exit 0

# Dev lifecycle
# 初始化 dev 模式：写入 .kageos/kageos.env，启动 MySQL / NATS / MinIO，并幂等初始化数据库。
go run ./cmd/kagectl init --dev

# 启动 dev 后端主进程；Ctrl-C 停止后端。
go run ./cmd/kagectl up

# 查看 dev 基础设施状态。
go run ./cmd/kagectl status

# 停止 dev 依赖服务。
go run ./cmd/kagectl down

# 持续查看 dev 依赖服务日志。
go run ./cmd/kagectl logs infra

# Dev app-base image
# 构建 dev 用户应用运行时基础镜像；如果同名 tag 已存在，脚本会跳过。
bash deploy/base/scripts/build-app-base-image.sh

# 强制重建 dev app-base 镜像，但允许复用 Docker/Podman layer 缓存。
bash deploy/base/scripts/build-app-base-image.sh --force

# 强制且不使用缓存重建 dev app-base 镜像；改了系统包、字体、Python 依赖时推荐用这一行。
bash deploy/base/scripts/build-app-base-image.sh --force --no-cache

# 指定 app-base 镜像 tag 后强制无缓存重建；当 dev 配置改了 base_image 时使用。
KAGEOS_APP_BASE_IMAGE="kagebase:latest" bash deploy/base/scripts/build-app-base-image.sh --force --no-cache

# Dev backend debug
# 如需绕过 kagectl 调试主进程，先执行 init --dev，再从 GoLand 启动 core/cmd/main/main.go。

# Image checks
# 检查本地 Podman 是否存在 dev app-base 镜像。
podman image exists kagebase:latest

# 列出本地 app-base 镜像。
podman images | grep kagebase

# 在 app-base 镜像内验证常用 Python 包是否可 import。
podman run --rm --entrypoint /bin/sh kagebase:latest -lc 'python3 -c "import pandas, numpy, matplotlib, openpyxl, xlsxwriter, pptx, plotly, pyecharts, bs4, yaml, qrcode, barcode, xlrd, xlwt, aiohttp, toml, snownlp, tabulate, arrow, dateutil, wordcloud, pymysql, pytesseract, yt_dlp; print(\"python packages OK\")" && yt-dlp --version'

# 在 app-base 镜像内验证常用 CLI 辅助工具。
podman run --rm --entrypoint /bin/sh kagebase:latest -lc 'wget --version | head -n 1 && mediainfo --Version && 7z i | head -n 2 && rsync --version | head -n 1 && zstd --version'

# 在 app-base 镜像内验证 Tesseract 命令和中英文语言包。
podman run --rm --entrypoint /bin/sh kagebase:latest -lc 'tesseract --version | head -n 1 && tesseract --list-langs | tee /tmp/tesseract-langs.txt && grep -x eng /tmp/tesseract-langs.txt && grep -x chi_sim /tmp/tesseract-langs.txt'

# 在 app-base 镜像内验证中文字体、fontconfig、matplotlib 与 ReportLab 兜底。
podman run --rm --entrypoint /bin/sh kagebase:latest -lc 'fc-match "sans-serif"; fc-match "sans-serif:lang=zh-cn"; fc-match "Arial:lang=zh-cn"; fc-match "Helvetica:lang=zh-cn"; fc-match "monospace:lang=zh-cn"; test -f "$KAGEOS_CJK_FONT"; test -f "$KAGEOS_REPORTLAB_CJK_FONT"; python3 -c "import matplotlib; print(matplotlib.matplotlib_fname())"; tmp=/tmp/reportlab-font-check.pdf; python3 -c "from reportlab.pdfgen import canvas; c=canvas.Canvas(\"/tmp/reportlab-font-check.pdf\"); c.setFont(\"Helvetica-Bold\", 12); c.drawString(10, 10, \"千幻智能\"); c.save()"; pdftotext "$tmp" - | grep -q "千幻智能"'

# Prod local deploy
# 生产配置初始化：生成 .kageos/prod/kage.yaml。
go run ./cmd/kagectl init --base-url http://your-ip-or-domain

# 执行生产预检。
go run ./cmd/kagectl doctor

# 本地构建主镜像并启动/更新生产服务。
go run ./cmd/kagectl up

# 使用已发布主镜像启动/更新生产服务。
go run ./cmd/kagectl up --image

# 执行生产健康检查。
go run ./cmd/kagectl verify

# 查看生产 main 日志。
go run ./cmd/kagectl logs main

# 查看生产服务状态。
go run ./cmd/kagectl status

# 停止生产服务。
go run ./cmd/kagectl down
