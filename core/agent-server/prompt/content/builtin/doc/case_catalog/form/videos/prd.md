# 案例：视频工具（单 Form）

## 一、项目概要

- **类型**：单 Form，POST，无 Table。
- **路由**：上传视频 + 目标格式，FFmpeg 转换后返回文件；路由组 `/form/videos`。
- **适合参考**：files 上传、GetFS、exec、响应 files。

---

## 二、PRD 要点（表格格式）

### 视频转换（video_convert，POST）

**请求**

| 字段       | 类型     | 必填 | 说明 |
|------------|----------|------|------|
| 上传视频   | 文件上传 | ✓   | 视频文件 |
| 目标格式   | 下拉选择 | ✓   | 如 mp4/avi/mov 等 |

**响应**

| 字段       | 类型     | 说明 |
|------------|----------|------|
| 转换后的视频 | 文件     | 转换后文件 |
| 转换统计   | 多行文本 | 成功/失败说明 |

---

## 三、文件与路由

| 文件               | 说明     | 注册 |
|--------------------|----------|------|
| video_convert.go   | 视频转换 | POST |

---

## 四、说明

代码实现见同目录下 video_convert.go；read_doc 本案例时以本 PRD 为准，具体代码可用 read_go_file 按需查看。


---

## 代码实现

以下为本案目录下 Go 源码，供 read_doc 时一并参考。

### video_convert.go

```go
//<文件名>video_convert.go</文件名>

package videos

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// VideoConvertReq 视频格式转换请求结构体
type VideoConvertReq struct {
	// 框架标签：widget:"type:files;accept:video/*;max_size:500MB;max_count:10" - 文件上传组件，支持多文件上传
	InputFiles *types.Files `json:"input_files" widget:"name:上传视频文件;type:files;accept:video/*;max_size:500MB;max_count:10" validate:"required"`

	// 框架标签：select 须配 options_colors，与 options 一一对应，前端用颜色区分选项
	OutputFormat string `json:"output_format" widget:"name:目标格式;type:select;options:mp4,webm,avi,mkv;options_colors:primary,success,info,warning;default:mp4" validate:"required,oneof=mp4 webm avi mkv"`
}

// VideoConvertResp 视频格式转换响应结构体
type VideoConvertResp struct {
	// 转换后的视频文件
	OutputFile *types.Files `json:"output_file" widget:"name:转换后的视频;type:files"`

	// 转换信息
	ConvertInfo string `json:"convert_info" widget:"name:转换信息;type:text_area"`
}

// VideoConvert 视频格式转换函数
func VideoConvert(ctx *app.Context, resp response.Response) error {
	var req VideoConvertReq
	err := ctx.ShouldBindValidate(&req)
	if err != nil {
		return err
	}

	fs := ctx.GetFS()

	// 1. 下载文件到本地
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	if len(inputFiles.GetFiles()) == 0 {
		return fmt.Errorf("没有找到输入文件")
	}

	// 2. 使用 FFmpeg 转换视频格式
	// 从环境变量获取 FFmpeg 路径
	ffmpegPath := os.Getenv("FFMPEG_PATH")
	if ffmpegPath == "" {
		ffmpegPath = "/usr/bin/ffmpeg" // 默认路径
	}

	// 使用 GetTraceOutputDir 生成唯一的输出目录（内部会自动创建）
	outputDir := fs.GetTraceOutputDir()

	// 3. 批量处理所有视频文件
	outputFilePaths := make([]string, 0)
	successCount := 0
	failCount := 0
	var errors []string

	for _, file := range inputFiles.GetFiles() {
		if file.LocalPath == "" {
			logger.Warnf(ctx, "[VideoConvert] 文件 %s 没有本地路径，跳过", file.Name)
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 本地路径为空", file.Name))
			continue
		}

		// 检查格式是否相同
		inputExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.LocalPath), "."))
		baseName := strings.TrimSuffix(filepath.Base(file.LocalPath), filepath.Ext(file.LocalPath))
		outputPath := filepath.Join(outputDir, baseName+"."+req.OutputFormat)

		if inputExt == req.OutputFormat {
			logger.Infof(ctx, "[VideoConvert] 文件 %s 格式相同，跳过转换: %s", file.Name, inputExt)
			outputPath = file.LocalPath
		} else {
			// ffmpeg -i input.mp4 -c:v libvpx -c:a libopus output.webm
			// 根据输出格式选择编码器
			var args []string
			args = append(args, "-i", file.LocalPath)

			switch req.OutputFormat {
			case "webm":
				args = append(args, "-c:v", "libvpx", "-c:a", "libopus")
			case "mp4":
				// 使用内置编码器（LGPL-only 构建不支持 libx264）
				args = append(args, "-c:v", "libvpx", "-c:a", "libopus", "-f", "mp4")
			default:
				// 其他格式使用默认编码
				args = append(args, "-c", "copy")
			}

			args = append(args, "-y", outputPath) // -y 覆盖输出文件

			cmd := exec.Command(ffmpegPath, args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				logger.Errorf(ctx, "[VideoConvert] 转换失败 %s: %v, output: %s", file.Name, err, string(output))
				failCount++
				errors = append(errors, fmt.Sprintf("文件 %s: %v", file.Name, err))
				continue
			}

			logger.Infof(ctx, "[VideoConvert] 转换成功: %s -> %s", file.LocalPath, outputPath)
		}

		outputFilePaths = append(outputFilePaths, outputPath)
		successCount++
	}

	// 4. 上传转换后的文件
	var outputFiles *types.Files
	if len(outputFilePaths) > 0 {
		outputFiles = fs.ResponseFiles(outputFilePaths)
		defer fs.RemoveFiles(outputFiles)
	}

	// 5. 构建转换信息
	convertInfo := fmt.Sprintf("转换完成！\n成功: %d 个\n失败: %d 个\n输出格式: %s", successCount, failCount, req.OutputFormat)
	if len(errors) > 0 {
		convertInfo += "\n\n失败详情:\n" + strings.Join(errors, "\n")
	}

	// 5. 构建响应
	return resp.Form(&VideoConvertResp{
		OutputFile:  outputFiles,
		ConvertInfo: convertInfo,
	}).Build()
}

// VideoConvertTemplate 视频格式转换配置
var VideoConvertTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "视频格式转换",
		Desc:     "支持将视频转换为MP4、WebM、AVI、MKV等多种格式。支持批量处理多个视频文件。使用 FFmpeg 进行格式转换，支持VP8/VP9和Opus编码（LGPL-only构建）。应用场景：视频格式统一、兼容性转换、文件大小优化等。",
		Tags:     []string{"视频处理", "格式转换", "工具"},
		Request:  &VideoConvertReq{},
		Response: &VideoConvertResp{},
	},
}

func init() {
	// 注册Form函数 - 视频格式转换
	packageContext.POST("convert", VideoConvert, VideoConvertTemplate)
}
```

