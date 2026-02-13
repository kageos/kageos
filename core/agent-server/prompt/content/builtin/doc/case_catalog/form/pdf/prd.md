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

代码实现见同目录下各 .go 文件；read_doc 本案例时以本 PRD 为准，具体代码可用 read_go_file 按需查看。


---

## 代码实现

以下为本案目录下 Go 源码，供 read_doc 时一并参考。

### pdf_extract_text.go

```go
//<文件名>pdf_extract_text.go</文件名>

package pdf

import (
	"fmt"
	"os"
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
	InputFiles *types.Files `json:"input_files" widget:"name:上传PDF文件;type:files;accept:.pdf;max_size:100MB;max_count:10" validate:"required"`
}

// PdfExtractTextResp PDF 文本提取响应结构体
type PdfExtractTextResp struct {
	// 提取的文本内容（多个文件的结果合并）
	ExtractedText string `json:"extracted_text" widget:"name:提取的文本;type:text_area"`

	// 提取统计信息
	ExtractStats string `json:"extract_stats" widget:"name:提取统计;type:text_area"`
}

// PdfExtractText PDF 文本提取函数
func PdfExtractText(ctx *app.Context, resp response.Response) error {
	var req PdfExtractTextReq
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

	// 2. 批量提取文本
	// 从环境变量获取 pdftotext 路径
	pdftotextPath := os.Getenv("PDFTOTEXT_PATH")
	if pdftotextPath == "" {
		pdftotextPath = "/usr/bin/pdftotext" // 默认路径
	}

	var allTexts []string
	successCount := 0
	failCount := 0
	var errors []string

	for _, file := range inputFiles.GetFiles() {
		if file.LocalPath == "" {
			logger.Warnf(ctx, "[PdfExtractText] 文件 %s 没有本地路径，跳过", file.Name)
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 本地路径为空", file.Name))
			continue
		}

		// pdftotext input.pdf - （输出到标准输出）
		cmd := exec.Command(pdftotextPath, file.LocalPath, "-")
		output, err := cmd.CombinedOutput()

		// 检查命令执行是否失败
		if err != nil {
			// pdftotext 在某些情况下（如扫描版PDF）可能返回非零退出码，但仍有输出
			// 检查是否有实际输出内容
			extractedText := strings.TrimSpace(string(output))
			if extractedText == "" {
				// 真正的失败：命令失败且无输出
				logger.Errorf(ctx, "[PdfExtractText] 提取文本失败 %s: %v, output: %s", file.Name, err, string(output))
				failCount++
				errors = append(errors, fmt.Sprintf("文件 %s: 提取失败 - %v", file.Name, err))
				continue
			}
			// 命令返回错误但有输出，可能是警告信息，继续处理
			logger.Warnf(ctx, "[PdfExtractText] 文件 %s 提取时出现警告: %v, 但已提取到文本", file.Name, err)
		}

		extractedText := strings.TrimSpace(string(output))
		if extractedText == "" {
			// 提取成功但文本为空，说明是扫描版PDF或图片PDF
			allTexts = append(allTexts, fmt.Sprintf("=== %s ===\n（未提取到文本内容，可能是扫描版PDF或图片PDF）", file.Name))
			logger.Warnf(ctx, "[PdfExtractText] 文件 %s 未提取到文本内容，可能是扫描版PDF", file.Name)
		} else {
			allTexts = append(allTexts, fmt.Sprintf("=== %s ===\n%s", file.Name, extractedText))
			logger.Infof(ctx, "[PdfExtractText] 成功提取文件 %s 的文本，长度: %d 字符", file.Name, len(extractedText))
		}

		successCount++
	}

	// 3. 合并所有文本
	extractedText := strings.Join(allTexts, "\n\n")
	if extractedText == "" {
		extractedText = "（未提取到任何文本内容）"
	}

	// 4. 构建统计信息
	stats := fmt.Sprintf("提取完成！\n成功: %d 个\n失败: %d 个", successCount, failCount)
	if len(errors) > 0 {
		stats += "\n\n失败详情:\n" + strings.Join(errors, "\n")
	}

	// 5. 构建响应
	return resp.Form(&PdfExtractTextResp{
		ExtractedText: extractedText,
		ExtractStats:  stats,
	}).Build()
}

// PdfExtractTextTemplate PDF 文本提取配置
var PdfExtractTextTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "PDF文本提取",
		Desc:     "从PDF文件中提取文本内容。支持批量处理多个PDF文件。支持文本型PDF，如果是扫描版PDF或图片PDF可能无法提取文本。使用 Poppler-utils 的 pdftotext 工具。",
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
	"os"
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
	InputFiles *types.Files `json:"input_files" widget:"name:上传PDF文件;type:files;accept:.pdf;max_size:100MB;max_count:20" validate:"required"`
}

// PdfMergeResp PDF合并响应结构体
type PdfMergeResp struct {
	// 合并后的PDF文件
	OutputFile *types.Files `json:"output_file" widget:"name:合并后的PDF;type:files"`

	// 合并信息
	MergeInfo string `json:"merge_info" widget:"name:合并信息;type:text_area"`
}

// PdfMerge PDF合并函数
func PdfMerge(ctx *app.Context, resp response.Response) error {
	var req PdfMergeReq
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

	if len(inputFiles.GetFiles()) < 2 {
		return fmt.Errorf("至少需要 2 个PDF文件才能合并")
	}

	// 2. 使用 Ghostscript 合并PDF
	// 从环境变量获取 Ghostscript 路径
	gsPath := os.Getenv("GHOSTSCRIPT_PATH")
	if gsPath == "" {
		gsPath = "/usr/bin/gs" // 默认路径
	}

	// 使用 GetTraceOutputDir 生成唯一的输出目录（内部会自动创建）
	outputDir := fs.GetTraceOutputDir()
	outputPath := filepath.Join(outputDir, "merged.pdf")

	// 3. 构建 Ghostscript 命令
	// gs -dBATCH -dNOPAUSE -q -sDEVICE=pdfwrite -sOutputFile=output.pdf input1.pdf input2.pdf ...
	var args []string
	args = append(args,
		"-dBATCH",                  // 处理完所有文件后退出
		"-dNOPAUSE",                // 不暂停
		"-q",                       // 安静模式
		"-sDEVICE=pdfwrite",        // 输出设备为 PDF
		"-sOutputFile="+outputPath, // 输出文件
	)

	// 添加所有输入文件
	for _, file := range inputFiles.GetFiles() {
		if file.LocalPath == "" {
			logger.Warnf(ctx, "[PdfMerge] 文件 %s 没有本地路径，跳过", file.Name)
			continue
		}
		args = append(args, file.LocalPath)
	}

	if len(args) <= 5 {
		return fmt.Errorf("没有有效的PDF文件可以合并")
	}

	cmd := exec.Command(gsPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[PdfMerge] 合并PDF失败: %v, output: %s", err, string(output))
		return fmt.Errorf("合并PDF失败: %v", err)
	}

	logger.Infof(ctx, "[PdfMerge] 成功合并 %d 个PDF文件", len(inputFiles.GetFiles()))

	// 4. 上传合并后的文件
	outputFiles := fs.ResponseFiles([]string{outputPath})
	defer fs.RemoveFiles(outputFiles)

	// 5. 构建合并信息
	fileNames := make([]string, 0)
	for _, file := range inputFiles.GetFiles() {
		fileNames = append(fileNames, file.Name)
	}

	mergeInfo := fmt.Sprintf("合并完成！\n合并文件数: %d 个\n文件列表:\n%s\n输出文件: %s",
		len(inputFiles.GetFiles()),
		strings.Join(fileNames, "\n"),
		filepath.Base(outputPath))

	// 6. 构建响应
	return resp.Form(&PdfMergeResp{
		OutputFile: outputFiles,
		MergeInfo:  mergeInfo,
	}).Build()
}

// PdfMergeTemplate PDF合并配置
var PdfMergeTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "PDF合并",
		Desc:     "将多个PDF文件合并为一个PDF文件。支持批量合并，按上传顺序合并。使用 Ghostscript 进行合并，保持原PDF的质量和格式。应用场景：合并多个PDF文档、报告生成、批量处理等。",
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

### pdf_to_images.go

```go
//<文件名>pdf_to_images.go</文件名>

package pdf

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

// PdfToImagesReq PDF转图片请求结构体
type PdfToImagesReq struct {
	// 框架标签：widget:"type:files;accept:.pdf;max_size:100MB;max_count:10" - 文件上传组件，支持多文件上传
	InputFiles *types.Files `json:"input_files" widget:"name:上传PDF文件;type:files;accept:.pdf;max_size:100MB;max_count:10" validate:"required"`

	// 框架标签：select 须配 options_colors，与 options 一一对应，前端用颜色区分选项
	OutputFormat string `json:"output_format" widget:"name:输出格式;type:select;options:png,jpeg;options_colors:primary,success;default:png" validate:"required,oneof=png jpeg"`
}

// PdfToImagesResp PDF转图片响应结构体
type PdfToImagesResp struct {
	// 转换后的图片文件列表
	OutputFiles *types.Files `json:"output_files" widget:"name:转换后的图片;type:files"`

	// 转换统计信息
	ConvertStats string `json:"convert_stats" widget:"name:转换统计;type:text_area"`
}

// PdfToImages PDF转图片函数
func PdfToImages(ctx *app.Context, resp response.Response) error {
	var req PdfToImagesReq
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

	// 2. 使用 pdftoppm 将PDF页面转换为图片
	// 从环境变量获取 pdftoppm 路径
	pdftoppmPath := os.Getenv("PDFTOPPM_PATH")
	if pdftoppmPath == "" {
		pdftoppmPath = "/usr/bin/pdftoppm" // 默认路径
	}

	// 使用 GetTraceOutputDir 生成唯一的输出目录（内部会自动创建）
	outputDir := fs.GetTraceOutputDir()

	// pdftoppm 格式标志
	formatFlag := "-png"
	if req.OutputFormat == "jpeg" {
		formatFlag = "-jpeg"
	}

	// 3. 批量处理所有PDF文件
	outputFilePaths := make([]string, 0)
	successCount := 0
	failCount := 0
	totalPages := 0
	var errors []string

	for _, file := range inputFiles.GetFiles() {
		if file.LocalPath == "" {
			logger.Warnf(ctx, "[PdfToImages] 文件 %s 没有本地路径，跳过", file.Name)
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 本地路径为空", file.Name))
			continue
		}

		// 生成输出前缀（基于文件名）
		baseName := strings.TrimSuffix(filepath.Base(file.LocalPath), filepath.Ext(file.LocalPath))
		outputPrefix := filepath.Join(outputDir, baseName+"_page")

		// pdftoppm -png input.pdf output_prefix （将PDF页面转换为PNG）
		cmd := exec.Command(pdftoppmPath, formatFlag, file.LocalPath, outputPrefix)
		output, err := cmd.CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[PdfToImages] 转换失败 %s: %v, output: %s", file.Name, err, string(output))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: %v", file.Name, err))
			continue
		}

		// 查找生成的图片文件
		outputDirFiles, err := filepath.Glob(outputPrefix + "*")
		if err != nil {
			logger.Errorf(ctx, "[PdfToImages] 查找输出文件失败 %s: %v", file.Name, err)
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 查找输出文件失败: %v", file.Name, err))
			continue
		}

		if len(outputDirFiles) == 0 {
			logger.Warnf(ctx, "[PdfToImages] 文件 %s 未生成任何图片", file.Name)
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 未生成任何图片", file.Name))
			continue
		}

		outputFilePaths = append(outputFilePaths, outputDirFiles...)
		totalPages += len(outputDirFiles)
		successCount++
		logger.Infof(ctx, "[PdfToImages] 成功转换文件 %s，生成 %d 张图片", file.Name, len(outputDirFiles))
	}

	// 4. 上传转换后的文件
	var outputFiles *types.Files
	if len(outputFilePaths) > 0 {
		outputFiles = fs.ResponseFiles(outputFilePaths)
		defer fs.RemoveFiles(outputFiles)
	}

	// 5. 构建统计信息
	stats := fmt.Sprintf("转换完成！\n成功: %d 个文件\n失败: %d 个文件\n总图片数: %d 张\n输出格式: %s", successCount, failCount, totalPages, req.OutputFormat)
	if len(errors) > 0 {
		stats += "\n\n失败详情:\n" + strings.Join(errors, "\n")
	}

	// 6. 构建响应
	return resp.Form(&PdfToImagesResp{
		OutputFiles:  outputFiles,
		ConvertStats: stats,
	}).Build()
}

// PdfToImagesTemplate PDF转图片配置
var PdfToImagesTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "PDF转图片",
		Desc:     "将PDF文件的每一页转换为图片。支持批量处理多个PDF文件。支持PNG和JPEG格式输出。使用 Poppler-utils 的 pdftoppm 工具。应用场景：PDF预览、PDF页面截图、PDF内容提取等。",
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

