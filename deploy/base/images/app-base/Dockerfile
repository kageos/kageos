# 使用 Ubuntu 仓库 FFmpeg（免去 arm64 源码编译，自带 libx264/freetype/fontconfig，支持 drawtext 中文）
# 合规：通过 exec 调用 ffmpeg 可商用，分发需遵守 GPL
FROM ubuntu:22.04
WORKDIR /app

# 禁止 apt/tzdata 交互式提问（构建时 tzdata 会问时区）
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai
ENV PIP_DISABLE_PIP_VERSION_CHECK=1

# 国内环境下 files.pythonhosted.org 经常超时，这里给出更稳的默认 pip 源与重试/超时设置。
# 海外或自建源场景可在构建时覆盖：
#   --build-arg PIP_INDEX_URL=...
#   --build-arg PIP_DEFAULT_TIMEOUT=...
#   --build-arg PIP_RETRIES=...
ARG PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple
ARG PIP_DEFAULT_TIMEOUT=300
ARG PIP_RETRIES=10
ARG APT_CHECK_DATE=0

# 配置完整 APT 源（main、restricted、universe、multiverse），按架构选镜像
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "arm64" ] || [ "$ARCH" = "armhf" ] || [ "$ARCH" = "ppc64el" ] || [ "$ARCH" = "s390x" ]; then \
        echo "deb http://ports.ubuntu.com/ubuntu-ports jammy main restricted universe multiverse" > /etc/apt/sources.list && \
        echo "deb http://ports.ubuntu.com/ubuntu-ports jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://ports.ubuntu.com/ubuntu-ports jammy-security main restricted universe multiverse" >> /etc/apt/sources.list; \
    else \
        echo "deb http://archive.ubuntu.com/ubuntu jammy main restricted universe multiverse" > /etc/apt/sources.list && \
        echo "deb http://archive.ubuntu.com/ubuntu jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb http://security.ubuntu.com/ubuntu jammy-security main restricted universe multiverse" >> /etc/apt/sources.list; \
    fi

# 某些 Podman / Docker Desktop 虚拟机时钟会短暂漂移，导致 apt 报
# "Release file is not valid yet"。默认关闭日期校验，避免 runtime base 首次构建被宿主机时钟问题卡死。
# 如需严格校验，可在构建时传 --build-arg APT_CHECK_DATE=1。
RUN if [ "${APT_CHECK_DATE}" = "0" ]; then \
        printf 'Acquire::Check-Date "false";\nAcquire::Check-Valid-Until "false";\n' > /etc/apt/apt.conf.d/99disable-check-date; \
    fi

# 安装 tini：用 Ubuntu universe 包，避免构建时访问 GitHub（国内/内网常导致 wget exit 8）
RUN apt-get update && apt-get install -y --no-install-recommends tini \
    && rm -rf /var/lib/apt/lists/* \
    && test -x /usr/bin/tini && echo "✓ tini from apt"

# 安装运行时依赖 + 字体（分层便于缓存：只改文档工具时只重建后面几层）
# e2fsprogs 提供 chattr/lsattr，用于 start.sh 对 workplace 做 +a 保护（禁止容器内删除）
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    file \
    jq \
    less \
    mediainfo \
    p7zip-full \
    qpdf \
    ripgrep \
    rsync \
    sqlite3 \
    unzip \
    wget \
    zip \
    zstd \
    e2fsprogs \
    tzdata \
    ghostscript \
    poppler-utils \
    graphicsmagick \
    imagemagick \
    gifsicle \
    libimage-exiftool-perl \
    libraw-bin \
    libvips-tools \
    lua5.4 \
    ocrmypdf \
    pngquant \
    python3 \
    python3-pip \
    python3-pandas \
    python3-numpy \
    python3-scipy \
    python3-matplotlib \
    python3-pil \
    python3-openpyxl \
    python3-requests \
    libass9 \
    libvpx7 \
    libopus0 \
    libvorbis0a \
    libtheora0 \
    libfreetype6 \
    fontconfig \
    libharfbuzz0b \
    libfribidi0 \
    libx264-163 \
    ffmpeg \
    tesseract-ocr \
    tesseract-ocr-chi-sim \
    unpaper \
    webp \
    fonts-noto-cjk \
    fonts-wqy-zenhei \
    fonts-dejavu-core \
    fonts-liberation \
    && fc-cache -fv \
    && rm -rf /var/lib/apt/lists/* /usr/share/man /usr/share/doc /usr/share/info \
    && echo "✓ Runtime + CLI/image/PDF OCR tools + FFmpeg + Tesseract + fonts"

# LibreOffice（单独一层：新增/变更 LO 时只重建本层及之后）
RUN apt-get update && apt-get install -y --no-install-recommends \
    libreoffice-core \
    libreoffice-writer \
    libreoffice-calc \
    libreoffice-impress \
    && rm -rf /var/lib/apt/lists/* \
    && echo "✓ LibreOffice"

# Pandoc（单独一层）
RUN apt-get update && apt-get install -y --no-install-recommends \
    pandoc \
    && rm -rf /var/lib/apt/lists/* \
    && echo "✓ Pandoc"

# Graphviz（单独一层：dot/neato/fdp 等，用代码画流程图/架构图/关系图）
RUN apt-get update && apt-get install -y --no-install-recommends \
    graphviz \
    && rm -rf /var/lib/apt/lists/* \
    && echo "✓ Graphviz"

# matplotlib 中文字体：优先使用简体中文 Noto CJK，注册名因系统/字体缓存差异可能表现为 SC/JP/TC。
RUN mkdir -p /etc/matplotlib /tmp/matplotlib \
    && chmod 1777 /tmp/matplotlib \
    && printf 'font.family: sans-serif\nfont.sans-serif: Noto Sans CJK SC, Noto Sans CJK JP, Noto Sans CJK TC, WenQuanYi Zen Hei, DejaVu Sans\naxes.unicode_minus: False\n' > /etc/matplotlib/matplotlibrc \
    && echo "✓ matplotlib matplotlibrc (CJK) configured"
ENV MATPLOTLIBRC=/etc/matplotlib/matplotlibrc \
    MPLCONFIGDIR=/tmp/matplotlib

# 创建 lua 命令的符号链接
RUN ln -sf /usr/bin/lua5.4 /usr/bin/lua

# 创建 python 命令的符号链接
RUN ln -sf /usr/bin/python3 /usr/bin/python

# 安装编译依赖 + pip 包（不删编译工具，避免连锁删 scipy/matplotlib）
# 若增删此处或上方 python3-* 包：请同步更新 core/agent-server 中 runPythonPreinstallDoc（工作台 run_python 工具说明）
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc g++ python3-dev libfreetype6-dev libpng-dev libjpeg-dev zlib1g-dev \
    && pip3 install --no-cache-dir --prefer-binary \
        --index-url "${PIP_INDEX_URL}" \
        --timeout "${PIP_DEFAULT_TIMEOUT}" \
        --retries "${PIP_RETRIES}" \
        seaborn python-docx PyPDF2 pdfplumber reportlab cryptography jieba snownlp \
        xlsxwriter python-pptx plotly pyecharts \
        beautifulsoup4 lxml aiohttp \
        PyYAML qrcode python-barcode xlrd xlwt toml \
        tabulate arrow python-dateutil wordcloud pymysql pytesseract yt-dlp \
    && rm -rf /tmp/* /root/.cache /var/lib/apt/lists/* \
    && echo "✓ Python packages installed"

# FFmpeg 已通过 apt 安装于 /usr/bin，与 fontconfig/中文字体同环境，支持 drawtext 中文

# 生成已安装 Python 包列表（用于快速检查）
RUN python3 -c "from importlib.metadata import distributions; packages = sorted(d.metadata['Name'] for d in distributions()); print('\n'.join(packages))" \
    > /etc/python-installed-packages.txt 2>/dev/null || echo "" > /etc/python-installed-packages.txt && \
    echo 'export PYTHON_INSTALLED_PACKAGES="$(cat /etc/python-installed-packages.txt | tr "\n" "," | sed "s/,$//")"' \
    > /etc/profile.d/python-packages.sh && \
    chmod +x /etc/profile.d/python-packages.sh && \
    echo "✓ Python installed packages list generated"

# 设置环境变量（PYTHON_INSTALLED_PACKAGES 由 start.sh 动态加载，不在此处设置以免覆盖）
ENV PYTHON_PATH=/usr/bin/python3
ENV PIP_PATH=/usr/bin/pip3

# 复制启动脚本（要求 build context 为当前目录）
COPY start.sh /start.sh
RUN chmod +x /start.sh

# 使用 tini 作为 init 系统
ENTRYPOINT ["/usr/bin/tini", "--"]

# 默认命令
CMD ["/start.sh"]

# 验证安装（全量检查，构建失败时一目了然）
RUN echo "=== 验证安装 ===" && \
    echo "Python:" && python3 --version && \
    echo "pip:" && pip3 --version && \
    echo "FFmpeg:" && ffmpeg -version 2>/dev/null | head -n 1 && \
    echo "GraphicsMagick:" && gm version | head -n 1 && \
    echo "ImageMagick:" && convert --version | head -n 1 && \
    echo "ExifTool:" && exiftool -ver && \
    echo "OCRmyPDF:" && ocrmypdf --version 2>&1 | head -n 1 && \
    echo "VIPS:" && vips --version && \
    echo "WebP:" && cwebp -version && \
    echo "pngquant:" && pngquant --version && \
    echo "gifsicle:" && gifsicle --version 2>&1 | head -n 1 && \
    echo "unpaper:" && unpaper --version 2>&1 | head -n 1 && \
    echo "LibRaw:" && (dcraw_emu -v 2>&1 | head -n 1 || raw-identify -v 2>&1 | head -n 1) && \
    echo "Lua:" && lua -v && \
    echo "LibreOffice:" && libreoffice --version 2>/dev/null | head -n 1 && \
    echo "Pandoc:" && pandoc --version 2>/dev/null | head -n 1 && \
    echo "Tesseract:" && tesseract --version 2>/dev/null | head -n 1 && \
    tesseract --list-langs 2>/dev/null | tee /tmp/tesseract-langs.txt && \
    grep -x eng /tmp/tesseract-langs.txt && \
    grep -x chi_sim /tmp/tesseract-langs.txt && \
    echo "Graphviz:" && dot -V 2>&1 | head -n 1 && \
    echo "CLI (子进程常用):" && \
    jq --version && \
    curl --version | head -n 1 && \
    command -v wget && wget --version | head -n 1 && \
    yt-dlp --version && \
    command -v mediainfo && mediainfo --Version && \
    command -v 7z && 7z i | head -n 2 && \
    command -v rsync && rsync --version | head -n 1 && \
    zstd --version && \
    sqlite3 --version && \
    zip -v | head -n 1 && \
    rg --version | head -n 1 && \
    qpdf --version | head -n 1 && \
    echo "Python 核心包:" && \
    python3 -c "import pandas; print('  pandas', pandas.__version__)" && \
    python3 -c "import numpy; print('  numpy', numpy.__version__)" && \
    python3 -c "import scipy; print('  scipy', scipy.__version__)" && \
    python3 -c "import matplotlib; print('  matplotlib', matplotlib.__version__)" && \
    python3 -c "import jieba; print('  jieba OK')" && \
    python3 -c "import snownlp; print('  snownlp OK')" && \
    python3 -c "import requests; print('  requests OK')" && \
    python3 -c "import openpyxl; print('  openpyxl OK')" && \
    python3 -c "import seaborn; print('  seaborn OK')" && \
    python3 -c "import xlsxwriter; print('  xlsxwriter OK')" && \
    python3 -c "import pptx; print('  python-pptx OK')" && \
    python3 -c "import plotly; print('  plotly OK')" && \
    python3 -c "import pyecharts; print('  pyecharts OK')" && \
    python3 -c "import bs4; print('  beautifulsoup4 OK')" && \
    python3 -c "import lxml; print('  lxml OK')" && \
    python3 -c "import aiohttp; print('  aiohttp OK')" && \
    python3 -c "import yaml; print('  PyYAML OK')" && \
    python3 -c "import qrcode; print('  qrcode OK')" && \
    python3 -c "import barcode; print('  python-barcode OK')" && \
    python3 -c "import xlrd; print('  xlrd OK')" && \
    python3 -c "import xlwt; print('  xlwt OK')" && \
    python3 -c "import toml; print('  toml OK')" && \
    python3 -c "import tabulate; print('  tabulate OK')" && \
    python3 -c "import arrow; print('  arrow OK')" && \
    python3 -c "import dateutil; print('  python-dateutil OK')" && \
    python3 -c "import wordcloud; print('  wordcloud OK')" && \
    python3 -c "import pymysql; print('  pymysql OK')" && \
    python3 -c "import pytesseract; print('  pytesseract OK')" && \
    python3 -c "import yt_dlp; print('  yt_dlp OK')" && \
    echo "FFmpeg drawtext + 中文字体:" && \
    (ffmpeg -filters 2>/dev/null | grep -q "drawtext") && echo "  drawtext 滤镜 OK" || (echo "  drawtext 缺失"; exit 1) && \
    (test -f /usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc || test -f /usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc) && echo "  Noto CJK 字体 OK（drawtext 中文可用 fontfile= 指向该路径）" || echo "  Noto CJK 路径可能不同，请用 fc-list :lang=zh 查看" && \
    echo "✓ All dependencies verified"
