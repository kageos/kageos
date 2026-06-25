# 使用 Ubuntu 仓库 FFmpeg（免去 arm64 源码编译，自带 libx264/freetype/fontconfig，支持 drawtext 中文）
# 合规：通过 exec 调用 ffmpeg 可商用，分发需遵守 GPL
FROM ubuntu:22.04
WORKDIR /app

# 禁止 apt/tzdata 交互式提问（构建时 tzdata 会问时区）
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai
ENV PIP_DISABLE_PIP_VERSION_CHECK=1
ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

# 国内环境下 files.pythonhosted.org 经常超时，这里给出更稳的默认 pip 源与重试/超时设置。
# 海外或自建源场景可在构建时覆盖：
#   --build-arg PIP_INDEX_URL=...
#   --build-arg PIP_DEFAULT_TIMEOUT=...
#   --build-arg PIP_RETRIES=...
ARG PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple
ARG PIP_DEFAULT_TIMEOUT=300
ARG PIP_RETRIES=10
ARG APT_CHECK_DATE=0
ARG UBUNTU_APT_MIRROR=http://mirrors.tuna.tsinghua.edu.cn/ubuntu
ARG UBUNTU_PORTS_APT_MIRROR=http://mirrors.tuna.tsinghua.edu.cn/ubuntu-ports

# 配置完整 APT 源（main、restricted、universe、multiverse），按架构选镜像
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "arm64" ] || [ "$ARCH" = "armhf" ] || [ "$ARCH" = "ppc64el" ] || [ "$ARCH" = "s390x" ]; then \
        echo "deb ${UBUNTU_PORTS_APT_MIRROR} jammy main restricted universe multiverse" > /etc/apt/sources.list && \
        echo "deb ${UBUNTU_PORTS_APT_MIRROR} jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb ${UBUNTU_PORTS_APT_MIRROR} jammy-security main restricted universe multiverse" >> /etc/apt/sources.list; \
    else \
        echo "deb ${UBUNTU_APT_MIRROR} jammy main restricted universe multiverse" > /etc/apt/sources.list && \
        echo "deb ${UBUNTU_APT_MIRROR} jammy-updates main restricted universe multiverse" >> /etc/apt/sources.list && \
        echo "deb ${UBUNTU_APT_MIRROR} jammy-security main restricted universe multiverse" >> /etc/apt/sources.list; \
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
    python3-fonttools \
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
    fonts-noto \
    fonts-noto-extra \
    fonts-noto-mono \
    fonts-noto-ui-core \
    fonts-noto-ui-extra \
    fonts-noto-cjk \
    fonts-noto-color-emoji \
    fonts-wqy-zenhei \
    fonts-dejavu-core \
    fonts-liberation \
    && fc-cache -fv \
    && rm -rf /var/lib/apt/lists/* /usr/share/man /usr/share/doc /usr/share/info \
    && echo "✓ Runtime + CLI/image/PDF OCR tools + FFmpeg + Tesseract + fonts"

# 系统级默认字体替换：Linux 没有“一个覆盖所有语言的单字体”，系统原生做法是
# fontconfig fallback 链。这里把 generic family 与常见西文字体名都强制 prepend 到
# Noto/CJK/Emoji 链，让 fontconfig-aware 工具默认拿到多语言字体。
# 这会影响 LibreOffice/ImageMagick/FFmpeg font= 等；不影响绕过 fontconfig 的库
# （例如 ReportLab 的 PDF 内置 Helvetica，下面另做 Python 层兜底）。
RUN mkdir -p /usr/local/share/fonts/kageos \
    && ln -sf /usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc /usr/local/share/fonts/kageos/NotoSansCJK-Regular.ttc \
    && ln -sf /usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc /usr/local/share/fonts/kageos/NotoSansCJK-Bold.ttc \
    && ln -sf /usr/share/fonts/truetype/wqy/wqy-zenhei.ttc /usr/local/share/fonts/kageos/wqy-zenhei.ttc \
    && python3 -c "from fontTools.ttLib import TTCollection; TTCollection('/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc').fonts[2].save('/usr/local/share/fonts/kageos/NotoSansCJKSC-Regular.otf'); TTCollection('/usr/share/fonts/opentype/noto/NotoSansCJK-Bold.ttc').fonts[2].save('/usr/local/share/fonts/kageos/NotoSansCJKSC-Bold.otf')" \
    && replace_font_file() { original="$1"; target="$2"; if [ -e "${original}" ] || [ -L "${original}" ]; then if [ ! -e "${original}.kageos-original" ] && [ ! -L "${original}.kageos-original" ]; then mv "${original}" "${original}.kageos-original"; else rm -f "${original}"; fi; ln -sf "${target}" "${original}"; fi; } \
    && replace_font_file /usr/share/fonts/truetype/dejavu/DejaVuSans.ttf /usr/local/share/fonts/kageos/NotoSansCJKSC-Regular.otf \
    && replace_font_file /usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf /usr/local/share/fonts/kageos/NotoSansCJKSC-Bold.otf \
    && replace_font_file /usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf /usr/local/share/fonts/kageos/NotoSansCJKSC-Regular.otf \
    && replace_font_file /usr/share/fonts/truetype/dejavu/DejaVuSansMono-Bold.ttf /usr/local/share/fonts/kageos/NotoSansCJKSC-Bold.otf \
    && replace_font_file /usr/share/fonts/truetype/dejavu/DejaVuSerif.ttf /usr/local/share/fonts/kageos/NotoSansCJKSC-Regular.otf \
    && replace_font_file /usr/share/fonts/truetype/dejavu/DejaVuSerif-Bold.ttf /usr/local/share/fonts/kageos/NotoSansCJKSC-Bold.otf \
    && replace_font_file /usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf /usr/local/share/fonts/kageos/NotoSansCJKSC-Regular.otf \
    && replace_font_file /usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf /usr/local/share/fonts/kageos/NotoSansCJKSC-Bold.otf \
    && replace_font_file /usr/share/fonts/truetype/liberation/LiberationSans-Italic.ttf /usr/local/share/fonts/kageos/NotoSansCJKSC-Regular.otf \
    && replace_font_file /usr/share/fonts/truetype/liberation/LiberationSans-BoldItalic.ttf /usr/local/share/fonts/kageos/NotoSansCJKSC-Bold.otf \
    && replace_font_file /usr/share/fonts/truetype/liberation/LiberationMono-Regular.ttf /usr/local/share/fonts/kageos/NotoSansCJKSC-Regular.otf \
    && replace_font_file /usr/share/fonts/truetype/liberation/LiberationMono-Bold.ttf /usr/local/share/fonts/kageos/NotoSansCJKSC-Bold.otf \
    && replace_font_file /usr/share/fonts/truetype/liberation/LiberationSerif-Regular.ttf /usr/local/share/fonts/kageos/NotoSansCJKSC-Regular.otf \
    && replace_font_file /usr/share/fonts/truetype/liberation/LiberationSerif-Bold.ttf /usr/local/share/fonts/kageos/NotoSansCJKSC-Bold.otf \
    && printf '%s\n' \
        '<?xml version="1.0"?>' \
        '<!DOCTYPE fontconfig SYSTEM "urn:fontconfig:fonts.dtd">' \
        '<fontconfig>' \
        '  <match target="pattern">' \
        '    <test qual="any" name="family"><string>sans-serif</string></test>' \
        '    <edit name="family" mode="prepend" binding="strong"><string>Noto Sans CJK SC</string><string>Noto Sans</string><string>Noto Color Emoji</string><string>WenQuanYi Zen Hei</string><string>DejaVu Sans</string></edit>' \
        '  </match>' \
        '  <match target="pattern">' \
        '    <test qual="any" name="family"><string>serif</string></test>' \
        '    <edit name="family" mode="prepend" binding="strong"><string>Noto Serif CJK SC</string><string>Noto Serif</string><string>Noto Color Emoji</string><string>DejaVu Serif</string></edit>' \
        '  </match>' \
        '  <match target="pattern">' \
        '    <test qual="any" name="family"><string>monospace</string></test>' \
        '    <edit name="family" mode="prepend" binding="strong"><string>Noto Sans Mono CJK SC</string><string>Noto Sans Mono</string><string>Noto Color Emoji</string><string>WenQuanYi Zen Hei</string><string>DejaVu Sans Mono</string></edit>' \
        '  </match>' \
        '  <alias binding="strong"><family>Arial</family><prefer><family>Noto Sans CJK SC</family><family>Noto Sans</family><family>Noto Color Emoji</family><family>WenQuanYi Zen Hei</family></prefer></alias>' \
        '  <alias binding="strong"><family>Helvetica</family><prefer><family>Noto Sans CJK SC</family><family>Noto Sans</family><family>Noto Color Emoji</family><family>WenQuanYi Zen Hei</family></prefer></alias>' \
        '  <alias binding="strong"><family>Liberation Sans</family><prefer><family>Noto Sans CJK SC</family><family>Noto Sans</family><family>Noto Color Emoji</family><family>WenQuanYi Zen Hei</family></prefer></alias>' \
        '  <alias binding="strong"><family>DejaVu Sans</family><prefer><family>Noto Sans CJK SC</family><family>Noto Sans</family><family>Noto Color Emoji</family><family>WenQuanYi Zen Hei</family></prefer></alias>' \
        '  <alias binding="strong"><family>Times New Roman</family><prefer><family>Noto Serif CJK SC</family><family>Noto Serif</family><family>Noto Color Emoji</family></prefer></alias>' \
        '  <alias binding="strong"><family>Liberation Serif</family><prefer><family>Noto Serif CJK SC</family><family>Noto Serif</family><family>Noto Color Emoji</family></prefer></alias>' \
        '  <alias binding="strong"><family>DejaVu Serif</family><prefer><family>Noto Serif CJK SC</family><family>Noto Serif</family><family>Noto Color Emoji</family></prefer></alias>' \
        '  <alias binding="strong"><family>Courier New</family><prefer><family>Noto Sans Mono CJK SC</family><family>Noto Sans Mono</family><family>Noto Color Emoji</family><family>WenQuanYi Zen Hei</family></prefer></alias>' \
        '  <alias binding="strong"><family>Liberation Mono</family><prefer><family>Noto Sans Mono CJK SC</family><family>Noto Sans Mono</family><family>Noto Color Emoji</family><family>WenQuanYi Zen Hei</family></prefer></alias>' \
        '  <alias binding="strong"><family>DejaVu Sans Mono</family><prefer><family>Noto Sans Mono CJK SC</family><family>Noto Sans Mono</family><family>Noto Color Emoji</family><family>WenQuanYi Zen Hei</family></prefer></alias>' \
        '</fontconfig>' \
        > /etc/fonts/local.conf \
    && fc-cache -fv \
    && fc-match 'sans-serif' \
    && fc-match 'sans-serif:lang=zh-cn' \
    && fc-match 'Arial:lang=zh-cn' \
    && fc-match 'Helvetica:lang=zh-cn' \
    && fc-match 'monospace:lang=zh-cn' \
    && test "$(readlink -f /usr/share/fonts/truetype/dejavu/DejaVuSans.ttf)" = "/usr/local/share/fonts/kageos/NotoSansCJKSC-Regular.otf" \
    && echo "✓ OS font defaults replaced by Noto CJK SC fallback chain"
ENV KAGEOS_CJK_FONT=/usr/local/share/fonts/kageos/NotoSansCJKSC-Regular.otf
ENV KAGEOS_CJK_FONT_INDEX=0
ENV KAGEOS_PIL_CJK_DEFAULT=1
ENV KAGEOS_REPORTLAB_CJK_FONT=/usr/local/share/fonts/kageos/wqy-zenhei.ttc
ENV KAGEOS_REPORTLAB_CJK_AUTOREGISTER=1

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

# Python 运行时兜底：ReportLab 不走 fontconfig，使用 PDF 内置 Helvetica/Times/Courier 时
# 无法显示中文；通过 sitecustomize 启动钩子把常见 ReportLab 字体名注册到可用的中文 TrueType 字体。
COPY kageos_font_defaults.py /tmp/kageos_font_defaults.py
RUN site_dir="$(python3 -c 'import site; print(site.getsitepackages()[0])')" \
    && stdlib_dir="$(python3 -c 'import sysconfig; print(sysconfig.get_path("stdlib"))')" \
    && cp /tmp/kageos_font_defaults.py "${site_dir}/kageos_font_defaults.py" \
    && if [ -f "${stdlib_dir}/sitecustomize.py" ]; then \
        printf '\n# Kageos font defaults\ntry:\n    import kageos_font_defaults\nexcept Exception:\n    pass\n' >> "${stdlib_dir}/sitecustomize.py"; \
    else \
        printf '# Kageos font defaults\ntry:\n    import kageos_font_defaults\nexcept Exception:\n    pass\n' > "${stdlib_dir}/sitecustomize.py"; \
    fi \
    && rm /tmp/kageos_font_defaults.py \
    && python3 -c "from reportlab.pdfbase import pdfmetrics; assert 'Kageos-CJK' in pdfmetrics.getRegisteredFontNames(); from PIL import ImageFont; assert ImageFont.load_default(size=16).getname()[0] == 'Noto Sans CJK SC'; print('✓ Python font defaults configured')"

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
    echo "Font defaults:" && \
    fc-match 'sans-serif' && \
    fc-match 'sans-serif:lang=zh-cn' && \
    fc-match 'Arial:lang=zh-cn' && \
    fc-match 'Helvetica:lang=zh-cn' && \
    fc-match 'monospace:lang=zh-cn' && \
    test -f "${KAGEOS_CJK_FONT}" && \
    test -f "${KAGEOS_REPORTLAB_CJK_FONT}" && \
    test "$(readlink -f /usr/share/fonts/truetype/dejavu/DejaVuSans.ttf)" = "/usr/local/share/fonts/kageos/NotoSansCJKSC-Regular.otf" && \
    python3 -c "from PIL import ImageFont; assert ImageFont.truetype('/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf', 48).getname()[0] == 'Noto Sans CJK SC'" && \
    python3 -c "from reportlab.pdfbase import pdfmetrics; from reportlab.pdfbase.ttfonts import TTFont; from reportlab.pdfgen import canvas; pdfmetrics.registerFont(TTFont('CompatDejaVuSans','/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf')); c=canvas.Canvas('/tmp/font-file-alias-check.pdf'); c.setFont('CompatDejaVuSans', 12); c.drawString(10, 10, '恰研智能'); c.save()" && \
    pdftotext /tmp/font-file-alias-check.pdf - | grep -q "恰研智能" && \
    echo "  font file aliases OK" && \
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
    python3 -c "from PIL import Image, ImageDraw, ImageFont; img=Image.new('RGB',(640,160),'black'); draw=ImageDraw.Draw(img); font=ImageFont.load_default(size=48); assert font.getname()[0] == 'Noto Sans CJK SC'; draw.text((20,40),'恰研智能',fill='white',font=font); img.save('/tmp/pillow-font-check.png')" && \
    echo "  pillow default CJK font OK" && \
    python3 -c "from reportlab.pdfgen import canvas; c=canvas.Canvas('/tmp/reportlab-font-check.pdf'); c.setFont('Helvetica-Bold', 12); c.drawString(10, 10, '千幻智能'); c.save()" && \
    pdftotext /tmp/reportlab-font-check.pdf - | grep -q "千幻智能" && \
    echo "  reportlab CJK fallback OK" && \
    echo "FFmpeg drawtext + 中文字体:" && \
    ffmpeg -hide_banner -loglevel error -f lavfi -i color=c=black:s=640x160:d=1 -frames:v 1 -vf "drawtext=text='恰研智能':fontsize=48:fontcolor=white:x=20:y=40" -y /tmp/ffmpeg-drawtext-check.png && \
    ( (ffmpeg -filters 2>/dev/null | grep -q "drawtext") && echo "  drawtext 滤镜 OK" || (echo "  drawtext 缺失"; exit 1) ) && \
    (test -f "${KAGEOS_CJK_FONT}" && echo "  Noto CJK 字体 OK（drawtext 默认走系统 fontconfig）") && \
    echo "✓ All dependencies verified"
