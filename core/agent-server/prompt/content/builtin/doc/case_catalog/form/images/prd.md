# 案例：图片工具（单 Form）

## 一、项目概要

- **类型**：单 Form，多个 POST，无 Table。
- **路由**：convert、resize、colors；路由组 `/form/images`。
- **适合参考**：files 上传、图片处理、多 POST 同目录。

---

## 二、PRD 要点（表格格式）

### 格式转换（images_convert，POST）

**请求**

| 字段     | 类型     | 必填 | 说明 |
|----------|----------|------|------|
| 上传图片 | 文件上传 | ✓   | 图片，最大 50MB，最多 10 个 |
| 目标格式 | 下拉选择 | ✓   | jpeg/png/gif/bmp/tiff，默认 png |

**响应**

| 字段           | 类型     | 说明 |
|----------------|----------|------|
| 转换后的图片   | 文件     | 转换后文件列表 |
| 转换统计       | 多行文本 | 成功/失败数量等 |

**说明**：images_resize、images_colors 等为「请求 files + 参数 → 响应 files 或 text_area」，结构类似。

---

## 三、文件与路由

| 文件               | 说明     | 注册 |
|--------------------|----------|------|
| images_convert.go  | 格式转换 | POST |
| images_resize.go   | 尺寸调整 | POST |
| images_colors.go   | 颜色提取 | POST |

---

## 四、说明

代码实现见同目录下各 .go 文件；read_doc 本案例时以本 PRD 为准，具体代码可用 read_go_file 按需查看。


---

## 代码实现

以下为本案目录下 Go 源码，供 read_doc 时一并参考。

### images_colors.go

```go
//<文件名>images_colors.go</文件名>

package images

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/EdlinOrg/prominentcolor"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// ImagesColorsReq 图片颜色提取请求结构体
type ImagesColorsReq struct {
	// 框架标签：widget:"type:files;accept:image/*;max_size:50MB;max_count:1" - 文件上传组件，只支持单文件上传
	InputFiles *types.Files `json:"input_files" widget:"name:上传图片;type:files;accept:image/*;max_size:50MB;max_count:1" validate:"required"`

	// 框架标签：widget:"type:number;min:0;max:1000;default:5;placeholder:0表示返回全部颜色" - 提取的颜色数量（0表示返回全部）
	ColorCount int `json:"color_count" widget:"name:提取颜色数量;type:number;min:0;max:1000;default:5;placeholder:0表示返回全部颜色" validate:"min=0,max=1000"`
}

// ColorInfo 颜色信息结构体
type ColorInfo struct {
	// 框架标签：widget:"type:color;format:hex" - 颜色选择器组件
	// 输入模式：显示为颜色选择器（支持 hex、rgb、rgba 格式）
	// 输出模式：显示颜色块和颜色值
	// 参数说明：format（颜色格式：hex/rgb/rgba，默认hex）
	Hex string `json:"hex" widget:"name:颜色值;type:color;format:hex"`
	
	// RGB 值
	RGB string `json:"rgb" widget:"name:RGB值;type:text"`
	
	// 框架标签：widget:"type:slider;min:0;max:100;unit:%" - 滑块/进度条组件
	// 输入模式：显示为滑块，用于编辑/新增表单
	// 输出模式：显示为进度条，自动显示百分比和状态颜色（>80% success, 50-80% warning, <50% danger）
	// 搜索模式：自动支持范围搜索（gte/lte）
	// 参数说明：min（最小值，必需）、max（最大值，必需）、unit（单位，可选）
	// 其他功能（提示、百分比、状态颜色等）自动处理，无需配置
	Percentage float64 `json:"percentage" widget:"name:占比;type:slider;min:0;max:100;unit:%"`
}

// ImagesColorsResp 图片颜色提取响应结构体
type ImagesColorsResp struct {
	// 提取的颜色列表
	Colors []ColorInfo `json:"colors" widget:"name:颜色列表;type:table"`

	// 处理统计信息
	Stats string `json:"stats" widget:"name:处理统计;type:text_area"`
}

// ImagesColors 图片颜色提取函数
func ImagesColors(ctx *app.Context, resp response.Response) error {
	var req ImagesColorsReq
	err := ctx.ShouldBindValidate(&req)
	if err != nil {
		return err
	}

	fs := ctx.GetFS()

	// 1. 下载文件到本地
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	// 2. 验证只上传了一张图片
	files := inputFiles.GetFiles()
	if len(files) == 0 {
		return fmt.Errorf("请上传一张图片")
	}
	if len(files) > 1 {
		return fmt.Errorf("只支持上传一张图片，当前上传了 %d 张", len(files))
	}

	file := files[0]
	if file.LocalPath == "" {
		return fmt.Errorf("文件 %s 没有本地路径", file.Name)
	}

	// 3. 提取颜色
	colors, err := extractColors(ctx, file.LocalPath, req.ColorCount)
	if err != nil {
		logger.Errorf(ctx, "[ImagesColors] 提取颜色失败 %s: %v", file.Name, err)
		return fmt.Errorf("提取颜色失败: %v", err)
	}

	logger.Infof(ctx, "[ImagesColors] 提取成功: %s (提取了 %d 种颜色)", file.Name, len(colors))

	// 4. 构建统计信息
	colorCountDesc := "全部"
	if req.ColorCount > 0 {
		colorCountDesc = fmt.Sprintf("%d 个", req.ColorCount)
	}
	stats := fmt.Sprintf("提取完成！\n文件名: %s\n提取颜色数量: %s\n实际提取: %d 种颜色", file.Name, colorCountDesc, len(colors))

	// 5. 构建响应
	return resp.Form(&ImagesColorsResp{
		Colors: colors,
		Stats:   stats,
	}).Build()
}

// extractColors 从图片中提取主要颜色
func extractColors(ctx *app.Context, imagePath string, k int) ([]ColorInfo, error) {
	// 打开图片文件
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 解码图片
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("解码图片失败: %w", err)
	}

	// 如果 k <= 0，表示返回全部颜色，使用一个足够大的数值来提取尽可能多的颜色
	// 如果 k > 0，则提取指定数量的颜色
	extractK := k
	if k <= 0 {
		// 使用足够大的数值（10000）来提取尽可能多的颜色
		// K-means 算法会根据实际颜色分布返回聚类结果，即使设置很大的k值，
		// 如果图片只有几种颜色，也只会返回实际的颜色种类数
		extractK = 10000
	}

	// 使用 K-means 算法提取主要颜色
	// KmeansWithAll 参数说明：
	// - k: 要提取的颜色数量
	// - img: 图片对象
	// - arguments: 处理参数（使用默认值 0）
	// - imageReSize: 图片缩放大小（80 像素，与默认值一致）
	// - bgmasks: 背景掩码（使用默认掩码，过滤白色/黑色/绿色背景）
	colors, err := prominentcolor.KmeansWithAll(
		extractK,
		img,
		prominentcolor.ArgumentDefault,
		80,
		prominentcolor.GetDefaultMasks(),
	)
	if err != nil {
		return nil, fmt.Errorf("颜色提取失败: %w", err)
	}

	// 转换为 ColorInfo 结构
	result := make([]ColorInfo, 0, len(colors))
	totalCount := 0
	for _, c := range colors {
		totalCount += c.Cnt
	}

	for _, c := range colors {
		hex := c.AsString()
		rgb := fmt.Sprintf("rgb(%d, %d, %d)", c.Color.R, c.Color.G, c.Color.B)
		percentage := float64(c.Cnt) / float64(totalCount) * 100.0

		result = append(result, ColorInfo{
			Hex:        hex,
			RGB:        rgb,
			Percentage: percentage,
		})
	}

	return result, nil
}

// ImagesColorsTemplate 图片颜色提取配置
var ImagesColorsTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "图片颜色提取",
		Desc:     "从图片中提取主要颜色。使用 K-means 聚类算法分析图片中的主要颜色，返回颜色的十六进制值、RGB 值和占比。支持指定提取数量（1-100），或设置为 0 返回全部颜色。应用场景：图片主题色分析、配色方案生成、图片风格分析、UI 设计参考等。",
		Tags:     []string{"图片处理", "颜色提取", "工具"},
		Request:  &ImagesColorsReq{},
		Response: &ImagesColorsResp{},
	},
}

func init() {
	// 注册Form函数 - 图片颜色提取
	packageContext.POST("colors", ImagesColors, ImagesColorsTemplate)
}

```

### images_convert.go

```go
//<文件名>images_convert.go</文件名>

package images

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

// ImagesConvertReq 图片格式转换请求结构体
type ImagesConvertReq struct {
	// 框架标签：widget:"type:files;accept:image/*;max_size:50MB;max_count:10" - 文件上传组件，支持多文件上传
	InputFiles *types.Files `json:"input_files" widget:"name:上传图片;type:files;accept:image/*;max_size:50MB;max_count:10" validate:"required"`

	// 框架标签：widget:"type:select;options:jpeg,png,gif,bmp,tiff;default:png" - 下拉选择组件
	// 注意：jpeg 和 jpg 是同一格式，统一使用 jpeg
	TargetFormat string `json:"target_format" widget:"name:目标格式;type:select;options:jpeg,png,gif,bmp,tiff;default:png" validate:"required,oneof=jpeg png gif bmp tiff"`
}

// ImagesConvertResp 图片格式转换响应结构体
type ImagesConvertResp struct {
	// 转换后的文件列表
	OutputFiles *types.Files `json:"output_files" widget:"name:转换后的图片;type:files"`

	// 转换统计信息
	ConvertStats string `json:"convert_stats" widget:"name:转换统计;type:text_area"`
}

// ImagesConvert 图片格式转换函数
func ImagesConvert(ctx *app.Context, resp response.Response) error {
	var req ImagesConvertReq
	err := ctx.ShouldBindValidate(&req)
	if err != nil {
		return err
	}

	fs := ctx.GetFS()

	// 1. 下载文件到本地
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	// 2. 转换图片格式
	outputFilePaths := make([]string, 0)
	successCount := 0
	failCount := 0
	var errors []string

	for _, file := range inputFiles.GetFiles() {
		if file.LocalPath == "" {
			logger.Warnf(ctx, "[ImagesConvert] 文件 %s 没有本地路径，跳过", file.Name)
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 本地路径为空", file.Name))
			continue
		}

		// 转换图片
		outputPath, err := convertImageFormat(ctx, fs, file.LocalPath, req.TargetFormat)
		if err != nil {
			logger.Errorf(ctx, "[ImagesConvert] 转换图片失败 %s: %v", file.Name, err)
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: %v", file.Name, err))
			continue
		}

		outputFilePaths = append(outputFilePaths, outputPath)
		successCount++
	}

	// 3. 上传转换后的文件
	var outputFiles *types.Files
	if len(outputFilePaths) > 0 {
		outputFiles = fs.ResponseFiles(outputFilePaths)
		defer fs.RemoveFiles(outputFiles)
	}

	// 4. 构建统计信息
	stats := fmt.Sprintf("转换完成！\n成功: %d 个\n失败: %d 个", successCount, failCount)
	if len(errors) > 0 {
		stats += "\n\n失败详情:\n" + strings.Join(errors, "\n")
	}

	// 5. 构建响应
	return resp.Form(&ImagesConvertResp{
		OutputFiles:  outputFiles,
		ConvertStats: stats,
	}).Build()
}

// convertImageFormat 转换图片格式（使用 GraphicsMagick）
func convertImageFormat(ctx *app.Context, fs *app.FS, inputPath string, targetFormat string) (string, error) {
	// 从环境变量获取 GraphicsMagick 路径
	gmPath := os.Getenv("GRAPHICSMAGICK_PATH")
	if gmPath == "" {
		gmPath = "/usr/bin/gm" // 默认路径
	}

	// 标准化格式名称（统一转为小写，JPG 统一为 jpeg）
	normalizedTarget := strings.ToLower(targetFormat)
	if normalizedTarget == "jpg" {
		normalizedTarget = "jpeg"
	}

	// 获取输入文件格式
	inputExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(inputPath), "."))
	if inputExt == "jpg" {
		inputExt = "jpeg"
	}

	// 如果格式相同，直接返回原文件路径
	if inputExt == normalizedTarget {
		logger.Infof(ctx, "[convertImageFormat] 格式相同，跳过转换: %s", inputExt)
		return inputPath, nil
	}

	// 使用 GetTraceOutputDir 生成唯一的输出目录（内部会自动创建）
	outputDir := fs.GetTraceOutputDir()
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outputPath := filepath.Join(outputDir, baseName+"."+normalizedTarget)

	// 使用 GraphicsMagick 转换图片格式
	// gm convert input.jpg output.png
	cmd := exec.Command(gmPath, "convert", inputPath, outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("GraphicsMagick 转换失败: %v, output: %s", err, string(output))
	}

	logger.Infof(ctx, "[convertImageFormat] 转换成功: %s -> %s", inputPath, outputPath)
	return outputPath, nil
}

// ImagesConvertTemplate 图片格式转换配置
var ImagesConvertTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "图片格式转换",
		Desc:     "支持将图片转换为JPEG、PNG、GIF、BMP、TIFF等多种格式。可以批量处理多个图片文件。使用 GraphicsMagick 进行格式转换，支持更多格式和更好的质量。注意：JPEG和JPG是同一格式，统一使用jpeg。应用场景：图片格式统一、文件大小优化、兼容性转换等。",
		Tags:     []string{"图片处理", "格式转换", "工具"},
		Request:  &ImagesConvertReq{},
		Response: &ImagesConvertResp{},
	},
}

func init() {
	// 💡 packageContext 是在当前目录下系统自动创建的变量，直接用即可，无需定义
	packageContext.POST("convert", ImagesConvert, ImagesConvertTemplate)
}
```

### images_resize.go

```go
//<文件名>images_resize.go</文件名>

package images

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// ImagesResizeReq 图片裁剪/缩放请求结构体
type ImagesResizeReq struct {
	// 框架标签：widget:"type:files;accept:image/*;max_size:50MB;max_count:10" - 文件上传组件，支持多文件上传
	InputFiles *types.Files `json:"input_files" widget:"name:上传图片;type:files;accept:image/*;max_size:50MB;max_count:10" validate:"required"`

	// 框架标签：widget:"type:select;options:1920x1080,1280x720,800x600,640x480,自定义;default:1920x1080" - 目标尺寸
	TargetSize string `json:"target_size" widget:"name:目标尺寸;type:select;options:1920x1080,1280x720,800x600,640x480,自定义;default:1920x1080" validate:"required"`

	// 框架标签：widget:"type:input;placeholder:例如:800x600" - 自定义尺寸（当选择"自定义"时使用）
	CustomSize string `json:"custom_size" widget:"name:自定义尺寸;type:input;placeholder:例如:800x600"`

	// 框架标签：widget:"type:select;options:保持宽高比,拉伸填充,裁剪填充;default:保持宽高比" - 缩放模式
	ResizeMode string `json:"resize_mode" widget:"name:缩放模式;type:select;options:保持宽高比,拉伸填充,裁剪填充;default:保持宽高比" validate:"required,oneof=保持宽高比 拉伸填充 裁剪填充"`
}

// ImagesResizeResp 图片裁剪/缩放响应结构体
type ImagesResizeResp struct {
	// 处理后的文件列表
	OutputFiles *types.Files `json:"output_files" widget:"name:处理后的图片;type:files"`

	// 处理统计信息
	ResizeStats string `json:"resize_stats" widget:"name:处理统计;type:text_area"`
}

// ImagesResize 图片裁剪/缩放函数
func ImagesResize(ctx *app.Context, resp response.Response) error {
	var req ImagesResizeReq
	err := ctx.ShouldBindValidate(&req)
	if err != nil {
		return err
	}

	fs := ctx.GetFS()

	// 1. 下载文件到本地
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	// 2. 确定目标尺寸
	targetSize := req.TargetSize
	if targetSize == "自定义" {
		targetSize = req.CustomSize
	}
	if targetSize == "" {
		return fmt.Errorf("目标尺寸不能为空")
	}

	// 解析尺寸（格式：宽x高）
	sizeParts := strings.Split(targetSize, "x")
	if len(sizeParts) != 2 {
		return fmt.Errorf("目标尺寸格式错误，应为 宽x高，例如: 1920x1080")
	}
	width := strings.TrimSpace(sizeParts[0])
	height := strings.TrimSpace(sizeParts[1])

	// 验证尺寸
	if _, err := strconv.Atoi(width); err != nil {
		return fmt.Errorf("宽度必须是数字: %s", width)
	}
	if _, err := strconv.Atoi(height); err != nil {
		return fmt.Errorf("高度必须是数字: %s", height)
	}

	// 3. 从环境变量获取 GraphicsMagick 路径
	gmPath := os.Getenv("GRAPHICSMAGICK_PATH")
	if gmPath == "" {
		gmPath = "/usr/bin/gm" // 默认路径
	}

	// 使用 GetTraceOutputDir 生成唯一的输出目录（内部会自动创建）
	outputDir := fs.GetTraceOutputDir()

	// 4. 批量处理所有图片文件
	outputFilePaths := make([]string, 0)
	successCount := 0
	failCount := 0
	var errors []string

	for _, file := range inputFiles.GetFiles() {
		if file.LocalPath == "" {
			logger.Warnf(ctx, "[ImagesResize] 文件 %s 没有本地路径，跳过", file.Name)
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 本地路径为空", file.Name))
			continue
		}

		// 生成输出文件路径
		baseName := strings.TrimSuffix(filepath.Base(file.LocalPath), filepath.Ext(file.LocalPath))
		ext := filepath.Ext(file.LocalPath)
		outputPath := filepath.Join(outputDir, baseName+"_resized"+ext)

		// 构建 GraphicsMagick 命令
		var args []string
		args = append(args, "convert")

		// 根据缩放模式选择参数
		switch req.ResizeMode {
		case "保持宽高比":
			// scale=width:height:force_original_aspect_ratio=decrease 保持宽高比，不拉伸
			args = append(args, "-resize", fmt.Sprintf("%sx%s", width, height))
		case "拉伸填充":
			// scale=width:height! 强制拉伸到目标尺寸
			args = append(args, "-resize", fmt.Sprintf("%sx%s!", width, height))
		case "裁剪填充":
			// 先缩放保持宽高比，然后裁剪到目标尺寸
			// 使用 -resize 和 -gravity center -crop 组合
			args = append(args, "-resize", fmt.Sprintf("%sx%s^", width, height)) // ^ 表示至少达到这个尺寸
			args = append(args, "-gravity", "center")
			args = append(args, "-crop", fmt.Sprintf("%sx%s+0+0", width, height))
		}

		args = append(args, file.LocalPath, outputPath)

		cmd := exec.Command(gmPath, args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[ImagesResize] 处理图片失败 %s: %v, output: %s", file.Name, err, string(output))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: %v", file.Name, err))
			continue
		}

		outputFilePaths = append(outputFilePaths, outputPath)
		successCount++
		logger.Infof(ctx, "[ImagesResize] 处理成功: %s -> %s (尺寸: %sx%s, 模式: %s)", file.Name, filepath.Base(outputPath), width, height, req.ResizeMode)
	}

	// 5. 上传处理后的文件
	var outputFiles *types.Files
	if len(outputFilePaths) > 0 {
		outputFiles = fs.ResponseFiles(outputFilePaths)
		defer fs.RemoveFiles(outputFiles)
	}

	// 6. 构建统计信息
	stats := fmt.Sprintf("处理完成！\n成功: %d 个\n失败: %d 个\n目标尺寸: %sx%s\n缩放模式: %s",
		successCount, failCount, width, height, req.ResizeMode)
	if len(errors) > 0 {
		stats += "\n\n失败详情:\n" + strings.Join(errors, "\n")
	}

	// 7. 构建响应
	return resp.Form(&ImagesResizeResp{
		OutputFiles: outputFiles,
		ResizeStats: stats,
	}).Build()
}

// ImagesResizeTemplate 图片裁剪/缩放配置
var ImagesResizeTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "图片裁剪/缩放",
		Desc:     "调整图片尺寸，支持裁剪和缩放。可以设置目标尺寸和缩放模式（保持宽高比、拉伸填充、裁剪填充）。支持批量处理多个图片文件。使用 GraphicsMagick 进行处理。应用场景：生成缩略图、调整图片尺寸、适配不同设备、图片裁剪等。",
		Tags:     []string{"图片处理", "裁剪缩放", "工具"},
		Request:  &ImagesResizeReq{},
		Response: &ImagesResizeResp{},
	},
}

func init() {
	// 注册Form函数 - 图片裁剪/缩放
	packageContext.POST("resize", ImagesResize, ImagesResizeTemplate)
}

```

