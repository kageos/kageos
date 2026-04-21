# 案例：PDF 工具（单 Form）

## 一、项目概要

- **类型**：单 Form，多个 POST，无 Table。
- **路由**：extract_text.form、merge.form、to_images.form；路由组 `/form/pdf`。
- **适合参考**：files 上传、响应 text_area 或 files、PDF 解析。

---

## 二、PRD 要点（表格格式）

### 提取文本（extract_text.form，POST）

**请求**：上传 PDF 文件（必填）+ 可选页码/范围等参数。

**响应**：提取的文本（text_area）+ 统计（页数、字符数等）。

### 合并 PDF（merge.form，POST）

**请求**：上传多个 PDF 文件（必填，至少 2 个）。

**响应**：合并后的 PDF 文件（files）或错误提示。

### 转图片（to_images.form，POST）

**请求**：上传 PDF 文件（必填）+ 可选分辨率、页码等。

**响应**：生成的图片文件列表（files）+ 统计。

---

## 三、文件与路由

| 文件                 | 说明         | 注册路由           |
|----------------------|--------------|--------------------|
| pdf_extract_text.go  | 提取文本     | POST extract_text.form |
| pdf_merge.go         | 合并 PDF     | POST merge.form    |
| pdf_to_images.go     | 转图片       | POST to_images.form |

---

## 四、说明

代码随本案例一起提供；read_doc 本案例路径（如 `/system/prompt/case_catalog/form/pdf`）即获得 PRD 与代码，无需再调用 read_go_file。


---

## 代码实现

以下为本案目录下 Go 源码，供 read_doc 时一并参考。

### pdf_extract_text.go

```go
//<文件名>pdf_extract_text.go</文件名>

package pdf

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// PdfExtractTextReq PDF 文本提取请求结构体
type PdfExtractTextReq struct {
	// 框架标签：widget:"type:files;accept:.pdf;max_size:100MB;max_count:10" - 文件上传组件，支持多文件上传
	InputFiles string `json:"input_files" widget:"name:上传PDF文件;type:files;accept:.pdf;max_size:100MB;max_count:10" validate:"required"`
}

// PdfExtractTextResp PDF 文本提取响应结构体
type PdfExtractTextResp struct {
	// 提取的文本内容（多个文件的结果合并）
	ExtractedText string `json:"extracted_text" widget:"name:提取的文本;type:text_area"`

	// 提取统计信息
	ExtractStats string `json:"extract_stats" widget:"name:提取统计;type:text_area"`
}

// PdfExtractText PDF 文本提取入口（SDK 注册用）：解析请求 → 调 DoPdfExtractText → 写响应
func PdfExtractText(ctx *app.Context, resp response.Response) error {
	var req PdfExtractTextReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoPdfExtractText(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoPdfExtractText PDF 文本提取业务逻辑：(ctx, req) → (res, err)，便于单测与复用
func DoPdfExtractText(ctx *app.Context, req *PdfExtractTextReq) (*PdfExtractTextResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	// 直接使用 pdftotext，依赖 PATH（镜像中已安装 poppler-utils）
	pdftotextPath := "pdftotext"

	var allTexts []string
	successCount := 0
	failCount := 0
	var errors []string

	for _, file := range inputFiles {
		if file == "" {
			logger.Warnf(ctx, "[PdfExtractText] 文件 %s 没有本地路径，跳过", filepath.Base(file))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 本地路径为空", filepath.Base(file)))
			continue
		}

		cmd := exec.Command(pdftotextPath, file, "-")
		output, err := cmd.CombinedOutput()

		if err != nil {
			extractedText := strings.TrimSpace(string(output))
			if extractedText == "" {
				logger.Errorf(ctx, "[PdfExtractText] 提取文本失败 %s: %v, output: %s", filepath.Base(file), err, string(output))
				failCount++
				errors = append(errors, fmt.Sprintf("文件 %s: 提取失败 - %v", filepath.Base(file), err))
				continue
			}
			logger.Warnf(ctx, "[PdfExtractText] 文件 %s 提取时出现警告: %v, 但已提取到文本", filepath.Base(file), err)
		}

		extractedText := strings.TrimSpace(string(output))
		if extractedText == "" {
			allTexts = append(allTexts, fmt.Sprintf("=== %s ===\n（未提取到文本内容，可能是扫描版PDF或图片PDF）", filepath.Base(file)))
			logger.Warnf(ctx, "[PdfExtractText] 文件 %s 未提取到文本内容，可能是扫描版PDF", filepath.Base(file))
		} else {
			allTexts = append(allTexts, fmt.Sprintf("=== %s ===\n%s", filepath.Base(file), extractedText))
			logger.Infof(ctx, "[PdfExtractText] 成功提取文件 %s 的文本，长度: %d 字符", filepath.Base(file), len(extractedText))
		}

		successCount++
	}

	extractedText := strings.Join(allTexts, "\n\n")
	if extractedText == "" {
		extractedText = "（未提取到任何文本内容）"
	}

	stats := fmt.Sprintf("提取完成！\n成功: %d 个\n失败: %d 个", successCount, failCount)
	if len(errors) > 0 {
		stats += "\n\n失败详情:\n" + strings.Join(errors, "\n")
	}

	return &PdfExtractTextResp{
		ExtractedText: extractedText,
		ExtractStats:  stats,
	}, nil
}

// PdfExtractTextTemplate PDF 文本提取配置
var PdfExtractTextTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "PDF文本提取",
		Desc:     `从PDF文件中提取文本内容。支持批量处理多个PDF文件。支持文本型PDF，如果是扫描版PDF或图片PDF可能无法提取文本。使用 Poppler-utils 的 pdftotext 工具。`,
		Tags:     []string{"PDF处理", "文本提取", "工具"},
		Request:  &PdfExtractTextReq{},
		Response: &PdfExtractTextResp{},
	},
}

func init() {
	// 直接使用 packageContext.POST 注册（新方式）
	packageContext.POST("extract_text.form", PdfExtractText, PdfExtractTextTemplate)
}
```

### pdf_merge.go

```go
//<文件名>pdf_merge.go</文件名>

package pdf

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// PdfMergeReq PDF合并请求结构体
type PdfMergeReq struct {
	// 框架标签：widget:"type:files;accept:.pdf;max_size:100MB;max_count:20" - 文件上传组件，支持多文件上传
	InputFiles string `json:"input_files" widget:"name:上传PDF文件;type:files;accept:.pdf;max_size:100MB;max_count:20" validate:"required"`
}

// PdfMergeResp PDF合并响应结构体
type PdfMergeResp struct {
	// 合并后的PDF文件
	OutputFile string `json:"output_file" widget:"name:合并后的PDF;type:files"`

	// 合并信息
	MergeInfo string `json:"merge_info" widget:"name:合并信息;type:text_area"`
}

// PdfMerge PDF合并入口（SDK 注册用）：解析请求 → 调 DoPdfMerge → 写响应
func PdfMerge(ctx *app.Context, resp response.Response) error {
	var req PdfMergeReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoPdfMerge(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoPdfMerge PDF合并业务逻辑：(ctx, req) → (res, err)，便于单测与复用。
// 仅需智能体介入的错误加 [系统错误] 前缀；此类错误打日志时须带足上下文（req %+v 等）便于排查。
func DoPdfMerge(ctx *app.Context, req *PdfMergeReq) (*PdfMergeResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	if len(inputFiles) < 2 {
		return nil, fmt.Errorf("至少需要 2 个PDF文件才能合并")
	}

	// 直接使用 gs，依赖 PATH（镜像中已安装 ghostscript）
	gsPath := "gs"

	outputDir := fs.GetTraceOutputDir()
	outputPath := filepath.Join(outputDir, "merged.pdf")

	var args []string
	args = append(args,
		"-dBATCH",
		"-dNOPAUSE",
		"-q",
		"-sDEVICE=pdfwrite",
		"-sOutputFile="+outputPath,
	)

	for _, file := range inputFiles {
		if file == "" {
			logger.Warnf(ctx, "[PdfMerge] 文件 %s 没有本地路径，跳过", filepath.Base(file))
			continue
		}
		args = append(args, file)
	}

	if len(args) <= 5 {
		return nil, fmt.Errorf("没有有效的PDF文件可以合并")
	}

	cmd := exec.Command(gsPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// [系统错误] 便于区分需智能体介入；打足上下文（req、err、output）方便后续排查
		logger.Errorf(ctx, "[系统错误]-[DoPdfMerge] 合并PDF失败, req: %+v, err: %v, output: %s", req, err, string(output))
		return nil, fmt.Errorf("[系统错误]-[DoPdfMerge]： 合并PDF失败, req: %+v, err: %v", req, err)
	}

	logger.Infof(ctx, "[PdfMerge] 成功合并 %d 个PDF文件", len(inputFiles))

	outputFiles := fs.ResponseFiles([]string{outputPath})

	fileNames := make([]string, 0)
	for _, file := range inputFiles {
		fileNames = append(fileNames, filepath.Base(file))
	}

	mergeInfo := fmt.Sprintf("合并完成！\n合并文件数: %d 个\n文件列表:\n%s\n输出文件: %s",
		len(inputFiles),
		strings.Join(fileNames, "\n"),
		filepath.Base(outputPath))

	return &PdfMergeResp{
		OutputFile: outputFiles,
		MergeInfo:  mergeInfo,
	}, nil
}

// PdfMergeTemplate PDF合并配置
var PdfMergeTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "PDF合并",
		Desc:     `将多个PDF文件合并为一个PDF文件。支持批量合并，按上传顺序合并。使用 Ghostscript 进行合并，保持原PDF的质量和格式。应用场景：合并多个PDF文档、报告生成、批量处理等。`,
		Tags:     []string{"PDF处理", "合并", "工具"},
		Request:  &PdfMergeReq{},
		Response: &PdfMergeResp{},
	},
}

func init() {
	// 注册Form函数 - PDF合并
	packageContext.POST("merge.form", PdfMerge, PdfMergeTemplate)
}
```

### pdf_run_command.go

```go
//<文件名>pdf_run_command.go</文件名>

package pdf

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// PdfRunCommandReq 自定义命令请求：上传 PDF + 命令模板（占位符替换后执行），便于智能体灵活调用
type PdfRunCommandReq struct {
	InputFiles string `json:"input_files" widget:"name:上传PDF文件;type:files;accept:.pdf,*/*;max_size:100MB;max_count:20" validate:"required"`

	// 命令模板，占位符：{{input}}=当前输入文件路径，{{output}}=当前输出文件路径。环境有 gs、pdftotext、pdftoppm、pdfinfo、pdfimages 等
	CommandTemplate string `json:"command_template" widget:"name:命令模板;type:text_area;placeholder:pdftotext {{input}} -" validate:"required"`

	// 输出文件扩展名，用于生成 {{output}} 路径（若命令中不用 {{output}} 可随意填）
	OutputExtension string `json:"output_extension" widget:"name:输出扩展名;type:input;default:txt" validate:"required"`
}

// PdfRunCommandResp 自定义命令响应
type PdfRunCommandResp struct {
	OutputFile string `json:"output_file" widget:"name:输出文件;type:files"`
	RunInfo    string       `json:"run_info" widget:"name:执行信息;type:text_area"`
}

// PdfRunCommand 自定义命令入口（SDK 注册用）
func PdfRunCommand(ctx *app.Context, resp response.Response) error {
	var req PdfRunCommandReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoPdfRunCommand(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoPdfRunCommand 按文件逐个替换 {{input}}/{{output}} 并执行，不经过 shell，安全
func DoPdfRunCommand(ctx *app.Context, req *PdfRunCommandReq) (*PdfRunCommandResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	outputExt := strings.TrimSpace(req.OutputExtension)
	if outputExt == "" {
		outputExt = "txt"
	}
	outputExt = strings.TrimPrefix(outputExt, ".")
	outputDir := fs.GetTraceOutputDir()
	hasOutputPlaceholder := strings.Contains(req.CommandTemplate, "{{output}}")

	var outputPaths []string
	var runInfos []string
	for i, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[PdfRunCommand] 文件 %s 无本地路径，跳过", filepath.Base(file))
			runInfos = append(runInfos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}
		baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		outputPath := filepath.Join(outputDir, baseName+"."+outputExt)

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
			logger.Errorf(ctx, "[PdfRunCommand] 执行失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			runInfos = append(runInfos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, string(out)))
			continue
		}
		if hasOutputPlaceholder {
			outputPaths = append(outputPaths, outputPath)
		}
		runInfos = append(runInfos, fmt.Sprintf("成功 %s", filepath.Base(file)))
		if i == 0 && len(out) > 0 {
			runInfos = append(runInfos, "命令输出:\n"+strings.TrimSpace(string(out)))
		}
	}

	var outputFiles string
	if len(outputPaths) > 0 {
		outputFiles = fs.ResponseFiles(outputPaths)
	}

	return &PdfRunCommandResp{
		OutputFile: outputFiles,
		RunInfo:    strings.Join(runInfos, "\n"),
	}, nil
}

func splitCommandLine(s string) []string {
	var out []string
	for _, part := range strings.Fields(s) {
		out = append(out, part)
	}
	return out
}

// PdfRunCommandTemplate 自定义命令表单配置
var PdfRunCommandTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name: "PDF处理自定义命令",
		Desc: `上传 PDF 后，用自定义命令模板处理（占位符 {{input}}、{{output}} 会替换为实际路径后执行）。不经过 shell，安全。环境有 gs、pdftotext、pdftoppm、pdfinfo、pdfimages 等；示例：pdftotext {{input}} - 或 pdftoppm -png {{input}} {{output}}。`,
		Tags:     []string{"PDF处理", "自定义命令", "智能体"},
		Request:  &PdfRunCommandReq{},
		Response: &PdfRunCommandResp{},
	},
}

func init() {
	packageContext.POST("run_command.form", PdfRunCommand, PdfRunCommandTemplate)
}
```

### pdf_to_images.go

```go
//<文件名>pdf_to_images.go</文件名>

package pdf

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// PdfToImagesReq PDF转图片请求结构体
type PdfToImagesReq struct {
	// 框架标签：widget:"type:files;accept:.pdf;max_size:100MB;max_count:10" - 文件上传组件，支持多文件上传
	InputFiles string `json:"input_files" widget:"name:上传PDF文件;type:files;accept:.pdf;max_size:100MB;max_count:10" validate:"required"`

	// 框架标签：select 须配 options_colors，与 options 一一对应，前端用颜色区分选项
	OutputFormat string `json:"output_format" widget:"name:输出格式;type:select;options:png,jpeg;options_colors:primary,success;default:png" validate:"required,oneof=png jpeg"`
}

// PdfToImagesResp PDF转图片响应结构体
type PdfToImagesResp struct {
	// 转换后的图片文件列表
	OutputFiles string `json:"output_files" widget:"name:转换后的图片;type:files"`

	// 转换统计信息
	ConvertStats string `json:"convert_stats" widget:"name:转换统计;type:text_area"`
}

// PdfToImages PDF转图片入口（SDK 注册用）：解析请求 → 调 DoPdfToImages → 写响应
func PdfToImages(ctx *app.Context, resp response.Response) error {
	var req PdfToImagesReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoPdfToImages(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoPdfToImages PDF转图片业务逻辑：(ctx, req) → (res, err)，便于单测与复用
func DoPdfToImages(ctx *app.Context, req *PdfToImagesReq) (*PdfToImagesResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	// 直接使用 pdftoppm，依赖 PATH（镜像中已安装 poppler-utils）
	pdftoppmPath := "pdftoppm"

	outputDir := fs.GetTraceOutputDir()

	formatFlag := "-png"
	if req.OutputFormat == "jpeg" {
		formatFlag = "-jpeg"
	}

	outputFilePaths := make([]string, 0)
	successCount := 0
	failCount := 0
	totalPages := 0
	var errors []string

	for _, file := range inputFiles {
		if file == "" {
			logger.Warnf(ctx, "[PdfToImages] 文件 %s 没有本地路径，跳过", filepath.Base(file))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 本地路径为空", filepath.Base(file)))
			continue
		}

		baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		outputPrefix := filepath.Join(outputDir, baseName+"_page")

		cmd := exec.Command(pdftoppmPath, formatFlag, file, outputPrefix)
		output, err := cmd.CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[PdfToImages] 转换失败 %s: %v, output: %s", filepath.Base(file), err, string(output))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: %v", filepath.Base(file), err))
			continue
		}

		outputDirFiles, err := filepath.Glob(outputPrefix + "*")
		if err != nil {
			logger.Errorf(ctx, "[PdfToImages] 查找输出文件失败 %s: %v", filepath.Base(file), err)
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 查找输出文件失败: %v", filepath.Base(file), err))
			continue
		}

		if len(outputDirFiles) == 0 {
			logger.Warnf(ctx, "[PdfToImages] 文件 %s 未生成任何图片", filepath.Base(file))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 未生成任何图片", filepath.Base(file)))
			continue
		}

		outputFilePaths = append(outputFilePaths, outputDirFiles...)
		totalPages += len(outputDirFiles)
		successCount++
		logger.Infof(ctx, "[PdfToImages] 成功转换文件 %s，生成 %d 张图片", filepath.Base(file), len(outputDirFiles))
	}

	var outputFiles string
	if len(outputFilePaths) > 0 {
		outputFiles = fs.ResponseFiles(outputFilePaths)
	}

	stats := fmt.Sprintf("转换完成！\n成功: %d 个文件\n失败: %d 个文件\n总图片数: %d 张\n输出格式: %s", successCount, failCount, totalPages, req.OutputFormat)
	if len(errors) > 0 {
		stats += "\n\n失败详情:\n" + strings.Join(errors, "\n")
	}

	return &PdfToImagesResp{
		OutputFiles:  outputFiles,
		ConvertStats: stats,
	}, nil
}

// PdfToImagesTemplate PDF转图片配置
var PdfToImagesTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "PDF转图片",
		Desc:     `将PDF文件的每一页转换为图片。支持批量处理多个PDF文件。支持PNG和JPEG格式输出。使用 Poppler-utils 的 pdftoppm 工具。应用场景：PDF预览、PDF页面截图、PDF内容提取等。`,
		Tags:     []string{"PDF处理", "图片转换", "工具"},
		Request:  &PdfToImagesReq{},
		Response: &PdfToImagesResp{},
	},
}

func init() {
	// 注册Form函数 - PDF转图片
	packageContext.POST("to_images.form", PdfToImages, PdfToImagesTemplate)
}
```

