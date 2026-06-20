# 案例：视频工具（单 Form）

## 一、项目概要

- **类型**：单 Form，POST，无 Table。
- **路由**：convert.form（视频转换）；路由组 `/form/videos`。
- **适合参考**：files 上传、`DownloadFiles`、`GetTraceOutputDir`、`exec.Command`、`ResponseFiles`、同格式文件复制到输出目录。

---

## 二、结构化 PRD

本案例的产品经理输出样例统一维护在同目录 `prd.json`，使用 PRD v2：`project/tables/forms/charts/rules`。本 Markdown 只保留实现参考、SDK 写法和注意事项，不再承载旧 PRD 表格。

## 三、文件与路由

| 文件               | 说明     | 注册路由       |
|--------------------|----------|----------------|
| video_convert.go   | 视频转换 | POST convert.form |

---

## 四、说明

- 运行镜像内为 **GPL FFmpeg**（含 libx264），视频转换（convert.form）及自定义命令（run_command.form）均可使用 libx264 做 H.264 编码（如 mov→mp4）。
- 代码随本案例一起提供；read_doc 本案例路径（如 `/system/prompt/case_catalog/form/videos`）即获得 PRD 与代码，无需再调用 read_go_file。

## 五、标准模式：上传视频 → FFmpeg 转换 → 输出附件

涉及视频/音频文件处理时，优先按下面的固定模式写：

1. 请求字段使用 `string`
2. `inputFiles := fs.DownloadFiles(req.InputFiles)`，并 `defer fs.RemoveFiles(inputFiles)`
3. 所有输出都写到 `outputDir := fs.GetTraceOutputDir()`
4. 子进程直接调用 `exec.Command("ffmpeg", ...)`
5. 用 `fs.ResponseFiles(outputPaths)` 返回给用户下载

特别注意：

- **不要直接返回输入临时文件**。如果输入格式和输出格式相同，先复制到 `outputDir` 再返回
- 视频转换默认优先显式指定编码器，例如 MP4 用 `libx264 + aac`
- 自定义命令模式下，只替换 `{{input}}` / `{{output}}`，不经过 shell

最小骨架：

```go
fs := ctx.GetFS()
inputFiles := fs.DownloadFiles(req.InputFiles)
defer fs.RemoveFiles(inputFiles)

outputDir := fs.GetTraceOutputDir()
outputPath := filepath.Join(outputDir, "output.mp4")

cmd := exec.Command("ffmpeg", "-i", inputPath, "-c:v", "libx264", "-c:a", "aac", "-y", outputPath)
output, err := cmd.CombinedOutput()
if err != nil {
    return nil, fmt.Errorf("ffmpeg 执行失败: %v, output: %s", err, string(output))
}

outputFiles := fs.ResponseFiles([]string{outputPath})
```


---

## 代码实现

以下为本案目录下 Go 源码，供 read_doc 时一并参考。

### video_convert.go

```go
//<文件名>video_convert.go</文件名>

package videos

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
)

// VideoConvertReq 视频格式转换请求结构体
type VideoConvertReq struct {
	// 框架标签：widget:"type:files;accept:video/*;max_size:500MB;max_count:10" - 文件上传组件，支持多文件上传
	InputFiles string `json:"input_files" widget:"name:上传视频文件;type:files;accept:video/*;max_size:500MB;max_count:10" validate:"required"`

	// 框架标签：select 须配 options_colors，与 options 一一对应，前端用颜色区分选项
	OutputFormat string `json:"output_format" widget:"name:目标格式;type:select;options:mp4,webm,avi,mkv;options_colors:409EFF,67C23A,909399,E6A23C;render_default:mp4" validate:"required,oneof=mp4 webm avi mkv"`
}

// VideoConvertResp 视频格式转换响应结构体
type VideoConvertResp struct {
	// 转换后的视频文件
	OutputFile string `json:"output_file" widget:"name:转换后的视频;type:files"`

	// 转换信息
	ConvertInfo string `json:"convert_info" widget:"name:转换信息;type:text_area"`
}

// VideoConvert 视频格式转换入口（SDK 注册用）：解析请求 → 调 DoVideoConvert → 写响应
func VideoConvert(ctx *app.Context, resp response.Response) error {
	var req VideoConvertReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoVideoConvert(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoVideoConvert 视频格式转换业务逻辑：(ctx, req) → (res, err)，便于单测与复用
func DoVideoConvert(ctx *app.Context, req *VideoConvertReq) (*VideoConvertResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	// 直接使用 ffmpeg，依赖 PATH（canonical Ubuntu 运行时镜像中已安装）
	ffmpegPath := "ffmpeg"

	outputDir := fs.GetTraceOutputDir()

	outputFilePaths := make([]string, 0)
	successCount := 0
	failCount := 0
	var errors []string

	for _, file := range inputFiles {
		if file == "" {
			logger.Warnf(ctx, "[VideoConvert] 文件 %s 没有本地路径，跳过", filepath.Base(file))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 本地路径为空", filepath.Base(file)))
			continue
		}

		inputExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(file), "."))
		baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		outputPath := filepath.Join(outputDir, baseName+"."+req.OutputFormat)

		if inputExt == req.OutputFormat {
			logger.Infof(ctx, "[VideoConvert] 文件 %s 格式相同，复制到输出目录: %s", filepath.Base(file), inputExt)
			if err := copyFile(file, outputPath); err != nil {
				logger.Errorf(ctx, "[VideoConvert] 复制失败 %s: %v", filepath.Base(file), err)
				failCount++
				errors = append(errors, fmt.Sprintf("文件 %s: 复制失败 %v", filepath.Base(file), err))
				continue
			}
		} else {
			var args []string
			args = append(args, "-i", file)

			switch req.OutputFormat {
			case "webm":
				args = append(args, "-c:v", "libvpx", "-c:a", "libopus")
			case "mp4":
				// MP4 使用 H.264+AAC，兼容性最好；mov/录屏等转 mp4 不会报错（libvpx+opus 在 mp4 中易导致 exit 234）
				args = append(args, "-c:v", "libx264", "-preset", "medium", "-crf", "23", "-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart", "-pix_fmt", "yuv420p")
			default:
				args = append(args, "-c", "copy")
			}

			args = append(args, "-y", outputPath)

			cmd := exec.Command(ffmpegPath, args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				logger.Errorf(ctx, "[VideoConvert] 转换失败 %s: %v, req: %+v, output: %s", filepath.Base(file), err, req, string(output))
				failCount++
				errors = append(errors, fmt.Sprintf("文件 %s: %v", filepath.Base(file), err))
				continue
			}

			logger.Infof(ctx, "[VideoConvert] 转换成功: %s -> %s", file, outputPath)
		}

		outputFilePaths = append(outputFilePaths, outputPath)
		successCount++
	}

	var outputFiles string
	if len(outputFilePaths) > 0 {
		outputFiles = fs.ResponseFiles(outputFilePaths)
	}

	convertInfo := fmt.Sprintf("转换完成！\n成功: %d 个\n失败: %d 个\n输出格式: %s", successCount, failCount, req.OutputFormat)
	if len(errors) > 0 {
		convertInfo += "\n\n失败详情:\n" + strings.Join(errors, "\n")
	}

	return &VideoConvertResp{
		OutputFile:  outputFiles,
		ConvertInfo: convertInfo,
	}, nil
}

// copyFile 复制文件，用于同格式时复制到输出目录，避免返回路径被 RemoveFiles(inputFiles) 删除
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// VideoConvertTemplate 视频格式转换配置
var VideoConvertTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "视频格式转换",
		Desc:     `支持将视频转换为MP4、WebM、AVI、MKV等多种格式，支持批量处理。运行环境为 GPL FFmpeg（含 libx264）。MP4 使用 H.264+AAC（mov/录屏等转 mp4 兼容性好）；WebM 使用 VP8/Opus。应用场景：视频格式统一、兼容性转换、文件大小优化等。`,
		Tags:     []string{"视频处理", "格式转换", "工具"},
		Request:  &VideoConvertReq{},
		Response: &VideoConvertResp{},
	},
}

func init() {
	// 注册Form函数 - 视频格式转换
	packageContext.POST("convert.form", VideoConvert, VideoConvertTemplate)
}
```

### video_run_command.go

```go
//<文件名>video_run_command.go</文件名>

package videos

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
)

// VideoRunCommandReq 自定义命令请求：上传文件 + 命令模板（占位符替换后执行），便于智能体灵活调用
type VideoRunCommandReq struct {
	InputFiles string `json:"input_files" widget:"name:上传视频/媒体文件;type:files;accept:video/*,*/*;max_size:500MB;max_count:10" validate:"required"`

	// 命令模板，占位符：{{input}}=当前输入文件路径，{{output}}=当前输出文件路径。运行环境为 GPL FFmpeg（含 libx264），可用 -c copy 或 -c:v libx264 -c:a aac 等
	CommandTemplate string `json:"command_template" widget:"name:命令模板;type:text_area;placeholder:ffmpeg -i {{input}} -c:v libx264 -c:a aac -y {{output}}" validate:"required"`

	// 输出文件扩展名，用于生成 {{output}} 路径
	OutputExtension string `json:"output_extension" widget:"name:输出扩展名;type:input;render_default:mp4" validate:"required"`
}

// VideoRunCommandResp 自定义命令响应
type VideoRunCommandResp struct {
	OutputFile string `json:"output_file" widget:"name:输出文件;type:files"`
	RunInfo    string       `json:"run_info" widget:"name:执行信息;type:text_area"`
}

// VideoRunCommand 自定义命令入口（SDK 注册用）
func VideoRunCommand(ctx *app.Context, resp response.Response) error {
	var req VideoRunCommandReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoVideoRunCommand(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoVideoRunCommand 自定义命令业务逻辑：按文件逐个替换 {{input}}/{{output}} 并执行，不经过 shell，安全
func DoVideoRunCommand(ctx *app.Context, req *VideoRunCommandReq) (*VideoRunCommandResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	outputExt := strings.TrimSpace(req.OutputExtension)
	if outputExt == "" {
		outputExt = "mp4"
	}
	outputExt = strings.TrimPrefix(outputExt, ".")
	outputDir := fs.GetTraceOutputDir()

	var outputPaths []string
	var runInfos []string
	for i, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[VideoRunCommand] 文件 %s 无本地路径，跳过", filepath.Base(file))
			runInfos = append(runInfos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}
		baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		outputPath := filepath.Join(outputDir, baseName+"."+outputExt)

		// 先按空格拆成参数，再替换占位符，这样路径中含空格时仍为单个参数
		args := splitCommandLine(req.CommandTemplate)
		for j := range args {
			if args[j] == "{{input}}" {
				args[j] = file
			} else if args[j] == "{{output}}" {
				args[j] = outputPath
			}
		}
		if len(args) == 0 {
			runInfos = append(runInfos, fmt.Sprintf("文件 %s: 命令为空", filepath.Base(file)))
			continue
		}
		cmd := exec.Command(args[0], args[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[VideoRunCommand] 执行失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			runInfos = append(runInfos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, string(out)))
			continue
		}
		outputPaths = append(outputPaths, outputPath)
		runInfos = append(runInfos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputPath))
		if i == 0 && len(out) > 0 {
			runInfos = append(runInfos, "命令输出:\n"+strings.TrimSpace(string(out)))
		}
	}

	var outputFiles string
	if len(outputPaths) > 0 {
		outputFiles = fs.ResponseFiles(outputPaths)
	}

	return &VideoRunCommandResp{
		OutputFile: outputFiles,
		RunInfo:    strings.Join(runInfos, "\n"),
	}, nil
}

// splitCommandLine 按空格拆分命令行为参数列表（占位符已替换为路径，路径中若有空格会拆坏，建议路径不含空格）
func splitCommandLine(s string) []string {
	var out []string
	for _, part := range strings.Fields(s) {
		out = append(out, part)
	}
	return out
}

// VideoRunCommandTemplate 自定义命令表单配置
var VideoRunCommandTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name: "视频格式转换自定义命令",
		Desc: `上传视频/媒体文件后，用自定义命令模板处理（占位符 {{input}}、{{output}} 会替换为实际路径后执行）。不经过 shell，安全。运行环境为 GPL FFmpeg（含 libx264），可用 -c copy 或 -c:v libx264 -c:a aac 等；示例：ffmpeg -i {{input}} -c:v libx264 -c:a aac -y {{output}}。`,
		Tags:     []string{"视频处理", "ffmpeg", "自定义命令", "智能体"},
		Request:  &VideoRunCommandReq{},
		Response: &VideoRunCommandResp{},
	},
}

func init() {
	packageContext.POST("run_command.form", VideoRunCommand, VideoRunCommandTemplate)
}
```
