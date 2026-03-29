# 运行时容器镜像构建（Canonical App Base）

本目录包含用于构建 `ai-agent-os:latest` 运行时容器镜像的 canonical Dockerfile。

## 镜像用途

这个镜像是用于运行用户生成的代码（SDK app）的运行时容器。每个用户应用都会在这个容器中运行。

## 构建镜像

```bash
# 在项目根目录执行
podman build -t ai-agent-os:latest deploy/base/images/app-base
```

或者使用 Docker：

```bash
docker build -t ai-agent-os:latest deploy/base/images/app-base
```

如需显式覆盖 pip 源 / 超时 / 重试（例如网络较差的生产环境）：

```bash
podman build \
  --build-arg PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple \
  --build-arg PIP_DEFAULT_TIMEOUT=300 \
  --build-arg PIP_RETRIES=10 \
  -t ai-agent-os:latest \
  deploy/base/images/app-base
```

## 镜像内容

- **基础镜像**：Ubuntu 22.04
- **Init 系统**：tini（处理 PID 1 问题）
- **FFmpeg**：静态链接构建，LGPL-only 配置
- **Ghostscript**：PDF/PostScript 处理
- **Poppler-utils**：PDF 工具（pdftotext, pdfinfo, pdfimages 等）
- **GraphicsMagick**：图像处理（ImageMagick 的轻量替代）
- **Lua**：轻量级脚本语言（数据转换、验证、模板处理等）
- **Python 3**：Python 解释器和 pip（用于执行 Python 代码）
- **启动脚本**：`/start.sh`（用于启动用户应用）

## FFmpeg 配置

镜像中包含静态链接的 FFmpeg，配置如下：

- **许可证**：LGPL-only（`--enable-gpl=no --enable-nonfree=no`）
- **构建方式**：静态链接（`--enable-static --disable-shared`）
- **支持的编码器**：
  - libvpx (VP8/VP9)
  - libopus (Opus)
  - libvorbis (Vorbis)
  - libtheora (Theora)
  - libass (字幕)
  - libfreetype (字体)
  - libfontconfig (字体配置)

**注意**：不支持 H.264/H.265 编码（GPL 许可），但支持解码。

## 使用方式

在用户生成的代码中，可以通过 `exec.Command` 调用这些工具：

### FFmpeg（音视频处理）

```go
import (
    "os/exec"
    "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/ffmpeg"
)

func ProcessVideo(input, output string) error {
    ffmpegPath := ffmpeg.GetPath()  // 返回 "/usr/bin/ffmpeg"
    cmd := exec.Command(ffmpegPath, "-i", input, output)
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
import json
import pandas as pd

df = pd.DataFrame(data)
summary = {
    "total": len(df),
    "columns": df.columns.tolist()
}
print(json.dumps(summary))
`

var result struct {
    Total   int      `json:"total"`
    Columns []string `json:"columns"`
}

executor := python.NewExecutor(code).
    WithArg("data", []map[string]interface{}{
        {"name": "Alice", "age": 30},
        {"name": "Bob", "age": 25},
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

**常用 Python 包**（需要时自动安装）：

- `pandas` - 数据分析
- `numpy` - 数值计算
- `matplotlib` - 数据可视化
- `requests` - HTTP 请求
- `Pillow` - 图像处理
- `PyPDF2` - PDF 处理
- `openpyxl` - Excel 处理

## 许可证合规性

所有工具都通过命令行调用，不涉及库链接，完全满足合规要求：
- **FFmpeg**：LGPL-only 构建，避免 GPL 约束
- **Ghostscript**：AGPL 3.0（通过 cmd 调用不影响你的代码）
- **Poppler-utils**：GPL 2.0+（通过 cmd 调用不影响你的代码）
- **GraphicsMagick**：MIT 许可证（非常宽松）
- **Lua**：MIT 许可证（非常宽松）
- **Python 3**：PSF 许可证（非常宽松）

用户代码可以保持闭源，详细的许可证信息请参考项目根目录的 `THIRD_PARTY_LICENSES.md`

**注意**：Python 包可能有各自的许可证，使用时请注意遵守。

## 验证镜像

构建完成后，可以验证所有工具是否正常安装：

```bash
podman run --rm ai-agent-os:latest sh -c "
    ffmpeg -version | head -n 1 && \
    gs --version && \
    pdftotext -v 2>&1 | head -n 1 && \
    gm version | head -n 1 && \
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

**最后更新**：2025-01-XX
