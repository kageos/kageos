# 运行时容器镜像构建（Canonical App Base）

本目录包含用于构建用户应用运行时基础镜像的 canonical Dockerfile。默认标签是 `agentos-app-runtime-base:latest`，也可以在构建时自行改 tag。

## 镜像用途

这个镜像是用于运行用户生成的代码（SDK app）的运行时容器。每个用户应用都会在这个容器中运行。

## 构建镜像

```bash
# 在项目根目录执行
podman build -t agentos-app-runtime-base:latest deploy/base/images/app-base
```

或者使用 Docker：

```bash
docker build -t agentos-app-runtime-base:latest deploy/base/images/app-base
```

如需显式覆盖 pip 源 / 超时 / 重试（例如网络较差的生产环境）：

```bash
podman build \
  --build-arg PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple \
  --build-arg PIP_DEFAULT_TIMEOUT=300 \
  --build-arg PIP_RETRIES=10 \
  -t agentos-app-runtime-base:latest \
  deploy/base/images/app-base
```

如果本机 Podman / Docker 虚拟机时钟漂移，Ubuntu APT 可能报：

```text
Release file is not valid yet
```

canonical 构建脚本默认会传 `APT_CHECK_DATE=0` 关闭这类日期校验。  
如需严格校验，可手工改成：

```bash
APP_BASE_APT_CHECK_DATE=1 bash deploy/base/scripts/build-app-base-image.sh
```

如果后面需要统一改名，也可以直接：

```bash
APP_BASE_IMAGE="agentos-app-runtime-base:latest" bash deploy/base/scripts/build-app-base-image.sh
```

如果本地已存在同 tag 镜像，想强制重建：

```bash
bash deploy/base/scripts/build-app-base-image.sh --force
```

如果还想禁用 layer 缓存：

```bash
bash deploy/base/scripts/build-app-base-image.sh --force --no-cache
```

## 镜像内容

- **基础镜像**：Ubuntu 22.04
- **Init 系统**：tini（处理 PID 1 问题）
- **FFmpeg**：Ubuntu 仓库版本（可直接 `exec.Command` 调用）
- **Ghostscript**：PDF/PostScript 处理
- **Poppler-utils**：PDF 工具（pdftotext, pdfinfo, pdfimages 等）
- **GraphicsMagick**：图像处理（ImageMagick 的轻量替代）
- **ImageMagick**：通用图片处理（canonical Ubuntu 22.04 镜像内为 IM6，直接用 `convert` / `identify` / `mogrify` 等）
- **ExifTool**：图片/视频/PDF 元数据读取、写入、清洗
- **OCRmyPDF**：为扫描 PDF 添加可搜索文字层
- **libvips-tools**：低内存、高性能图片处理与缩略图生成
- **WebP tools**：`cwebp`/`dwebp`/`webpmux` 等 WebP 工具
- **pngquant / gifsicle / unpaper / LibRaw**：PNG 压缩、GIF 优化、扫描件预处理、RAW 图片工具
- **Lua**：轻量级脚本语言（数据转换、验证、模板处理等）
- **Python 3**：Python 解释器和 pip（用于执行 Python 代码）
- **启动脚本**：`/start.sh`（用于启动用户应用）

## FFmpeg 配置

镜像中直接安装 Ubuntu 仓库提供的 FFmpeg，可在用户应用里通过 `exec.Command` 调用。

- **来源**：Ubuntu 22.04 仓库
- **常见能力**：转码、抽帧、音频处理、字幕/滤镜处理、`drawtext` 中文
- **中文支持**：镜像内已安装 `fontconfig`、`Noto CJK`、`WenQuanYi Zen Hei` 字体，可直接配合 `drawtext` 使用；matplotlib 默认也会优先使用中文字体，避免标题、坐标轴、图例出现方框。

**注意**：具体编译选项和编码器能力以镜像内 `ffmpeg -version`、`ffmpeg -encoders`、`ffmpeg -codecs` 的实际输出为准；分发时仍需遵守 FFmpeg 及其启用编解码器的许可证要求。

## 使用方式

在用户生成的代码中，直接通过 `exec.Command` 调用 PATH 里的可执行程序即可。新增工具不再额外维护 `*_PATH` 环境变量，文档统一维护可执行程序调用方式。

## 可执行程序表

| 工具 | 可执行程序 | 典型调用 |
| --- | --- | --- |
| FFmpeg | `ffmpeg` | `ffmpeg -i input.mp4 output.mp3` |
| Ghostscript | `gs` | `gs -sDEVICE=pdfwrite -o out.pdf in.ps` |
| Poppler | `pdftotext` `pdfinfo` `pdfimages` `pdftoppm` | `pdftotext in.pdf -` |
| GraphicsMagick | `gm` | `gm convert in.jpg -resize 800x600 out.png` |
| ImageMagick | `convert` `identify` `mogrify` `composite` | `convert in.jpg -resize 800x600 out.png` |
| ExifTool | `exiftool` | `exiftool -all= -overwrite_original image.jpg` |
| OCRmyPDF | `ocrmypdf` | `ocrmypdf --skip-text -l chi_sim+eng in.pdf out.pdf` |
| libvips | `vips` `vipsthumbnail` | `vipsthumbnail in.jpg --size 512x512 --path out.jpg` |
| WebP tools | `cwebp` `dwebp` `webpmux` `img2webp` | `cwebp -q 80 in.png -o out.webp` |
| pngquant | `pngquant` | `pngquant --quality=65-80 --output out.png --force in.png` |
| gifsicle | `gifsicle` | `gifsicle -O3 in.gif -o out.gif` |
| unpaper | `unpaper` | `unpaper in.pnm out.pnm` |
| LibRaw | `dcraw_emu` `raw-identify` | `raw-identify photo.cr2` |
| Lua | `lua` | `lua script.lua` |
| Python | `python3` `pip3` | `python3 script.py` |

说明：

- canonical `app-base` 是 Ubuntu 22.04，所以 ImageMagick 这里默认按 IM6 记忆：直接调用 `convert` / `identify`，不要假设有 `magick`
- `Dockerfile.alpine` 安装的通常是 IM7；那边既可用 `magick`，也兼容 `convert`，但 canonical 文档以 Ubuntu 为准

下面是常见调用示例：

### FFmpeg（音视频处理）

```go
func ProcessVideo(input, output string) error {
    cmd := exec.Command("ffmpeg", "-i", input, output)
    return cmd.Run()
}
```

### Ghostscript（PDF/PostScript 处理）

```go
func ConvertPS2PDF(input, output string) error {
    cmd := exec.Command("gs", "-sDEVICE=pdfwrite", "-o", output, input)
    return cmd.Run()
}
```

### Poppler-utils（PDF 工具）

```go
func ExtractPDFText(pdfFile string) (string, error) {
    cmd := exec.Command("pdftotext", pdfFile, "-")
    output, err := cmd.Output()
    return string(output), err
}

func GetPDFInfo(pdfFile string) (string, error) {
    cmd := exec.Command("pdfinfo", pdfFile)
    output, err := cmd.Output()
    return string(output), err
}
```

### GraphicsMagick（图像处理）

```go
func ResizeImage(input, output string, width, height int) error {
    cmd := exec.Command("gm", "convert", input, "-resize", 
        fmt.Sprintf("%dx%d", width, height), output)
    return cmd.Run()
}
```

### ImageMagick（图像处理）

```go
func ResizeWithIM(input, output string, width, height int) error {
    cmd := exec.Command("convert", input, "-resize",
        fmt.Sprintf("%dx%d", width, height), output)
    return cmd.Run()
}
```

### OCRmyPDF（扫描 PDF OCR）

```go
func OCRScannedPDF(input, output string) error {
    cmd := exec.Command("ocrmypdf", "--skip-text", "-l", "chi_sim+eng", input, output)
    return cmd.Run()
}
```

### ExifTool（元数据提取/清洗）

```go
func StripMetadata(file string) error {
    cmd := exec.Command("exiftool", "-all=", "-overwrite_original", file)
    return cmd.Run()
}
```

### libvips / WebP / pngquant / gifsicle（增强图片工具）

```go
func MakeThumbnail(input, output string) error {
    cmd := exec.Command("vipsthumbnail", input, "--size", "512x512", "--path", output)
    return cmd.Run()
}

func ConvertToWebP(input, output string) error {
    cmd := exec.Command("cwebp", "-q", "80", input, "-o", output)
    return cmd.Run()
}
```

### Lua（脚本处理）

```go
// 数据验证
func ValidateData(data string) error {
    cmd := exec.Command("lua", "scripts/validate.lua", data)
    return cmd.Run()
}

// 数据转换
func TransformData(input string) (string, error) {
    cmd := exec.Command("lua", "scripts/transform.lua", input)
    output, err := cmd.Output()
    return string(output), err
}

// 模板渲染
func RenderTemplate(template, data string) (string, error) {
    cmd := exec.Command("lua", "scripts/render.lua", template, data)
    output, err := cmd.Output()
    return string(output), err
}
```

**Lua 脚本示例**：

```lua
-- validate.lua: 数据验证
local json = require("json")
local data = json.decode(arg[1])

if data.amount and data.amount < 0 then
    error("金额不能为负")
end

print("验证通过")
```

```lua
-- transform.lua: 数据转换
local json = require("json")
local data = json.decode(arg[1])

-- 转换逻辑
data.created_at = os.date("%Y-%m-%d %H:%M:%S")
data.status = data.status or "pending"

print(json.encode(data))
```

### Python（数据科学、AI/ML、文档处理）

**推荐使用 SDK 封装**：

```go
import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/runtime/python"

// 执行 Python 代码并自动解析 JSON 输出
code := `
import pandas as pd

def agentos_entry(args, output_dir):
    df = pd.DataFrame(args["data"])
    summary = {
        "total": len(df),
        "columns": df.columns.tolist()
    }
    return {"data": summary}
`

var result struct {
    Total   int      `json:"total"`
    Columns []string `json:"columns"`
}

executor := python.NewExecutor(code).
    WithRequest(map[string]interface{}{
        "data": []map[string]interface{}{
        {"name": "Alice", "age": 30},
        {"name": "Bob", "age": 25},
        },
    }).
    WithPackages("pandas").
    WithTimeout(2 * time.Minute)
defer executor.Close()

err := executor.ExecuteJSON(ctx, &result)
```

**直接调用 Python**（不推荐，但也可以）：

```go
// 执行 Python 脚本
func ExecutePythonScript(scriptPath string, args map[string]interface{}) ([]byte, error) {
    argsJSON, _ := json.Marshal(args)
    cmd := exec.Command("python3", scriptPath, string(argsJSON))
    return cmd.CombinedOutput()
}
```

**Python 应用场景**：

- **数据分析**：使用 Pandas、NumPy 处理数据
- **AI/ML**：使用 Transformers、PyTorch 进行模型推理
- **文档处理**：使用 PyPDF2、openpyxl 处理 PDF/Excel
- **图像处理**：使用 Pillow、OpenCV 处理图像
- **科学计算**：使用 SciPy、SymPy 进行科学计算

**Python 包管理**：

Python 包可以通过 `pip install` 动态安装，SDK 的 `WithPackages()` 方法会自动安装依赖包。

**常用 Python 包**（基础镜像预装；SDK 仍支持按需安装额外包）：

- `pandas` - 数据分析
- `numpy` - 数值计算
- `matplotlib` - 数据可视化
- `requests` - HTTP 请求
- `Pillow` - 图像处理
- `PyPDF2` - PDF 处理
- `openpyxl` - Excel 处理
- `xlsxwriter` / `xlrd` - Excel 写入增强 / 老版 Excel 读取
- `xlwt` - 老版 Excel `.xls` 写入
- `python-pptx` - PPT 生成
- `plotly` / `pyecharts` - 交互图表 / 中文图表
- `beautifulsoup4` / `lxml` / `aiohttp` - 网页解析 / 异步 HTTP
- `jieba` / `snownlp` - 中文分词 / 中文情感分析
- `wordcloud` - 词云图
- `tabulate` - 终端/Markdown 表格输出
- `arrow` / `python-dateutil` - 日期处理 / 智能日期解析
- `pymysql` - MySQL 连接
- `PyYAML` / `toml` - 配置文件解析
- `qrcode` / `python-barcode` - 二维码 / 条形码生成

## 许可证合规性

所有工具都通过命令行调用，不涉及库链接，完全满足合规要求：
- **FFmpeg**：LGPL-only 构建，避免 GPL 约束
- **Ghostscript**：AGPL 3.0（通过 cmd 调用不影响你的代码）
- **Poppler-utils**：GPL 2.0+（通过 cmd 调用不影响你的代码）
- **GraphicsMagick**：MIT 许可证（非常宽松）
- **ImageMagick**：请以项目官方许可证说明为准
- **Lua**：MIT 许可证（非常宽松）
- **Python 3**：PSF 许可证（非常宽松）
- **新增工具**：ExifTool、OCRmyPDF、libvips、libwebp、pngquant、gifsicle、unpaper、LibRaw 请以各项目官方许可证说明为准

用户代码可以保持闭源，详细的许可证信息请参考项目根目录的 `THIRD_PARTY_LICENSES.md`

**注意**：Python 包可能有各自的许可证，使用时请注意遵守。

## 验证镜像

构建完成后，可以验证所有工具是否正常安装：

```bash
podman run --rm agentos-app-runtime-base:latest sh -c "
    ffmpeg -version | head -n 1 && \
    gs --version && \
    pdftotext -v 2>&1 | head -n 1 && \
    gm version | head -n 1 && \
    convert --version | head -n 1 && \
    exiftool -ver && \
    ocrmypdf --version 2>&1 | head -n 1 && \
    vips --version && \
    cwebp -version && \
    pngquant --version && \
    gifsicle --version 2>&1 | head -n 1 && \
    unpaper --version 2>&1 | head -n 1 && \
    lua -v && \
    python3 --version && \
    pip3 --version
"
```

## 注意事项

1. **构建时间**：FFmpeg 构建可能需要 10-30 分钟，取决于硬件性能
2. **镜像大小**：构建时已做瘦身（清 /tmp、缓存、man/doc、site-packages 的 tests/docs、FFmpeg strip 等），实际体积取决于依赖；若仍较大可继续裁剪缓存、文档和调试符号
3. **功能限制**：FFmpeg 不支持 H.264/H.265 编码（GPL 许可），但支持解码
4. **Python 包**：Python 包会在运行时动态安装，首次使用可能需要下载时间

---

**最后更新**：2026-04-18
