# 第三方软件许可证

本文档列出了本项目中使用的第三方开源软件及其许可证信息。

---

## FFmpeg

### 基本信息

- **软件名称**：FFmpeg
- **版本**：6.1.1（或构建时指定的版本）
- **许可证**：LGPL 2.1+（GNU Lesser General Public License v2.1 or later）
- **官方网站**：https://ffmpeg.org/
- **源代码**：https://ffmpeg.org/download.html

### 构建配置

本镜像中的 FFmpeg 采用以下配置构建：

```bash
./configure \
    --enable-gpl=no \
    --enable-nonfree=no \
    --enable-static \
    --disable-shared \
    --enable-libvpx \
    --enable-libopus \
    --enable-libvorbis \
    --enable-libtheora \
    --enable-libass \
    --enable-libfreetype \
    --enable-libfontconfig
```

**关键配置说明**：
- `--enable-gpl=no`：禁用 GPL 组件，仅使用 LGPL 许可的组件
- `--enable-nonfree=no`：禁用非自由组件
- `--enable-static`：静态链接构建，生成独立的可执行文件
- `--disable-shared`：禁用动态库，不生成共享库文件

### 使用方式

本项目中，FFmpeg 通过命令行调用方式使用，不涉及库链接：

```go
import (
    "os/exec"
    "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/ffmpeg"
)

func ProcessVideo(input, output string) error {
    ffmpegPath := ffmpeg.GetPath()
    cmd := exec.Command(ffmpegPath, "-i", input, output)
    return cmd.Run()
}
```

### 许可证合规性

1. **不构成衍生作品**：通过 `exec.Command` 调用 FFmpeg 可执行文件，属于进程调用，不是库链接，因此不构成衍生作品。

2. **用户代码可闭源**：用户生成的代码可以保持闭源，因为只是调用外部程序，不涉及库链接。

3. **分发要求**：根据 LGPL 2.1+ 的要求，本镜像在分发时已提供 FFmpeg 的源代码获取方式（见上方"源代码"链接）。

### 许可证全文

LGPL 2.1 许可证全文可在以下地址获取：
- https://www.gnu.org/licenses/lgpl-2.1.html

### 版权声明

FFmpeg 的版权归其贡献者所有。详细信息请参考：
- https://ffmpeg.org/legal.html

---

## 其他第三方软件

（如有其他第三方软件，请在此处添加）

---

**最后更新**：2025-01-XX

