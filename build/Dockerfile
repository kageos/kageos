# 使用 Ubuntu 仓库 FFmpeg（免去 arm64 源码编译，自带 libx264/freetype/fontconfig，支持 drawtext 中文）
# 合规：通过 exec 调用 ffmpeg 可商用，分发需遵守 GPL
FROM ubuntu:22.04
WORKDIR /app

# 禁止 apt/tzdata 交互式提问（构建时 tzdata 会问时区）
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai

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

# 安装 tini（多架构：amd64/arm64），保留 ARG 便于构建时覆盖版本
ARG TINI_VERSION=0.19.0
RUN apt-get update && apt-get install -y --no-install-recommends wget ca-certificates \
    && TINI_ARCH=$(dpkg --print-architecture) \
    && wget -q -O /usr/bin/tini "https://github.com/krallin/tini/releases/download/v${TINI_VERSION}/tini-${TINI_ARCH}" \
    && chmod +x /usr/bin/tini \
    && apt-get remove -y wget && apt-get autoremove -y && rm -rf /var/lib/apt/lists/*

# 安装运行时依赖 + 字体（分层便于缓存：只改文档工具时只重建后面几层）
# e2fsprogs 提供 chattr/lsattr，用于 start.sh 对 workplace 做 +a 保护（禁止容器内删除）
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    file \
    e2fsprogs \
    tzdata \
    ghostscript \
    poppler-utils \
    graphicsmagick \
    lua5.4 \
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
    fonts-noto-cjk \
    fonts-dejavu-core \
    fonts-liberation \
    && fc-cache -fv \
    && rm -rf /var/lib/apt/lists/* /usr/share/man /usr/share/doc /usr/share/info \
    && echo "✓ Runtime + FFmpeg + Tesseract + fonts"

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

# 一劳永逸：matplotlib 中文字体由 matplotlibrc 统一配置（仅用 Noto Sans CJK + DejaVu）
RUN mkdir -p /etc/matplotlib \
    && printf 'font.family: sans-serif\nfont.sans-serif: Noto Sans CJK SC, Noto Sans CJK TC, Noto Sans CJK JP, DejaVu Sans\naxes.unicode_minus: False\n' > /etc/matplotlib/matplotlibrc \
    && echo "✓ matplotlib matplotlibrc (CJK) configured"
ENV MPLCONFIGDIR=/etc/matplotlib

# 创建 lua 命令的符号链接
RUN ln -sf /usr/bin/lua5.4 /usr/bin/lua

# 创建 python 命令的符号链接
RUN ln -sf /usr/bin/python3 /usr/bin/python

# 安装编译依赖 + pip 包（不删编译工具，避免连锁删 scipy/matplotlib）
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc g++ python3-dev libfreetype6-dev libpng-dev libjpeg-dev zlib1g-dev \
    && pip3 install --no-cache-dir \
        seaborn python-docx PyPDF2 pdfplumber jieba \
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

# 复制启动脚本（与 Dockerfile 同级）
COPY build/start.sh /start.sh
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
    echo "Lua:" && lua -v && \
    echo "LibreOffice:" && libreoffice --version 2>/dev/null | head -n 1 && \
    echo "Pandoc:" && pandoc --version 2>/dev/null | head -n 1 && \
    echo "Tesseract:" && tesseract --version 2>/dev/null | head -n 1 && \
    echo "Graphviz:" && dot -V 2>&1 | head -n 1 && \
    echo "Python 核心包:" && \
    python3 -c "import pandas; print('  pandas', pandas.__version__)" && \
    python3 -c "import numpy; print('  numpy', numpy.__version__)" && \
    python3 -c "import scipy; print('  scipy', scipy.__version__)" && \
    python3 -c "import matplotlib; print('  matplotlib', matplotlib.__version__)" && \
    python3 -c "import jieba; print('  jieba OK')" && \
    python3 -c "import requests; print('  requests OK')" && \
    python3 -c "import openpyxl; print('  openpyxl OK')" && \
    python3 -c "import seaborn; print('  seaborn OK')" && \
    echo "FFmpeg drawtext + 中文字体:" && \
    (ffmpeg -filters 2>/dev/null | grep -q "drawtext") && echo "  drawtext 滤镜 OK" || (echo "  drawtext 缺失"; exit 1) && \
    (test -f /usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc || test -f /usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc) && echo "  Noto CJK 字体 OK（drawtext 中文可用 fontfile= 指向该路径）" || echo "  Noto CJK 路径可能不同，请用 fc-list :lang=zh 查看" && \
    echo "✓ All dependencies verified"
