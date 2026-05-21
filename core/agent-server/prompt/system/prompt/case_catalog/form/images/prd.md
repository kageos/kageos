# 案例：图片工具（单 Form）

## 一、项目概要

- **类型**：单 Form，多个 POST，无 Table。
- **路由**：convert.form、resize.form、colors.form；路由组 `/form/images`。
- **适合参考**：files 上传、ImageMagick/exec、`GetTraceOutputDir`、`ResponseFiles`、多 POST 同目录。

---

## 二、结构化 PRD

本案例的产品经理输出样例统一维护在同目录 `prd.json`，使用 PRD v2：`project/tables/forms/charts/rules`。本 Markdown 只保留实现参考、SDK 写法和注意事项，不再承载旧 PRD 表格。

## 三、文件与路由

| 文件               | 说明     | 注册路由        |
|--------------------|----------|-----------------|
| images_convert.go  | 格式转换 | POST convert.form  |
| images_resize.go   | 尺寸调整 | POST resize.form   |
| images_colors.go   | 颜色提取 | POST colors.form   |

---

## 四、说明

- canonical Ubuntu 运行时镜像里已安装 **ImageMagick** 与 **GraphicsMagick**；图片处理默认优先使用 **ImageMagick**，即 `convert` / `identify` / `mogrify`
- `gm` 仍可兼容使用，但不再作为图片处理默认示例
代码随本案例一起提供；read_doc 本案例路径（如 `/system/prompt/case_catalog/form/images`）即获得 PRD 与代码，无需再调用 read_go_file。


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
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
)

// ImagesColorsReq 图片颜色提取请求结构体
type ImagesColorsReq struct {
	// 框架标签：widget:"type:files;accept:image/*;max_size:50MB;max_count:1" - 文件上传组件，只支持单文件上传
	InputFiles string `json:"input_files" widget:"name:上传图片;type:files;accept:image/*;max_size:50MB;max_count:1" validate:"required"`

	// 框架标签：widget:"type:number;min:0;max:1000;render_default:5;placeholder:0表示返回全部颜色" - 提取的颜色数量（0表示返回全部）
	ColorCount int `json:"color_count" widget:"name:提取颜色数量;type:number;min:0;max:1000;render_default:5;placeholder:0表示返回全部颜色" validate:"min=0,max=1000"`
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
	// 输出模式：显示为进度条，自动显示百分比和状态颜色（>80% 67C23A, 50-80% E6A23C, <50% F56C6C）
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

// ImagesColors 图片颜色提取入口（SDK 注册用）：解析请求 → 调 DoImagesColors → 写响应
func ImagesColors(ctx *app.Context, resp response.Response) error {
	var req ImagesColorsReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoImagesColors(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoImagesColors 图片颜色提取业务逻辑：(ctx, req) → (res, err)，便于单测与复用
func DoImagesColors(ctx *app.Context, req *ImagesColorsReq) (*ImagesColorsResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("请上传一张图片")
	}
	if len(files) > 1 {
		return nil, fmt.Errorf("只支持上传一张图片，当前上传了 %d 张", len(files))
	}
	file := files[0]
	if file == "" {
		return nil, fmt.Errorf("文件 %s 没有本地路径", filepath.Base(file))
	}

	colors, err := extractColors(ctx, file, req.ColorCount)
	if err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoImagesColors] 提取颜色失败 %s: %v, req: %+v", filepath.Base(file), err, req)
		return nil, fmt.Errorf("[系统错误]-[DoImagesColors]： 提取颜色失败, req: %+v, err: %v", req, err)
	}
	logger.Infof(ctx, "[ImagesColors] 提取成功: %s (提取了 %d 种颜色)", filepath.Base(file), len(colors))

	colorCountDesc := "全部"
	if req.ColorCount > 0 {
		colorCountDesc = fmt.Sprintf("%d 个", req.ColorCount)
	}
	stats := fmt.Sprintf("提取完成！\n文件名: %s\n提取颜色数量: %s\n实际提取: %d 种颜色", filepath.Base(file), colorCountDesc, len(colors))

	return &ImagesColorsResp{Colors: colors, Stats: stats}, nil
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
		Desc:     `从图片中提取主要颜色。使用 K-means 聚类算法分析图片中的主要颜色，返回颜色的十六进制值、RGB 值和占比。支持指定提取数量（1-100），或设置为 0 返回全部颜色。应用场景：图片主题色分析、配色方案生成、图片风格分析、UI 设计参考等。`,
		Tags:     []string{"图片处理", "颜色提取", "工具"},
		Request:  &ImagesColorsReq{},
		Response: &ImagesColorsResp{},
	},
}

func init() {
	// 注册Form函数 - 图片颜色提取
	packageContext.POST("colors.form", ImagesColors, ImagesColorsTemplate)
}

```

### images_convert.go

```go
//<文件名>images_convert.go</文件名>

package images

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
)

// ImagesConvertReq 图片格式转换请求结构体
type ImagesConvertReq struct {
	// 框架标签：widget:"type:files;accept:image/*;max_size:50MB;max_count:10" - 文件上传组件，支持多文件上传
	InputFiles string `json:"input_files" widget:"name:上传图片;type:files;accept:image/*;max_size:50MB;max_count:10" validate:"required"`

	// 框架标签：widget:"type:select;options:..." - 下拉选择须配 options_colors，与 options 一一对应，颜色只用 RRGGBB。
	// 注意：jpeg 和 jpg 是同一格式，统一使用 jpeg
	TargetFormat string `json:"target_format" widget:"name:目标格式;type:select;options:jpeg,png,gif,bmp,tiff;options_colors:E91E63,4CAF50,FF9800,2196F3,9E9E9E;render_default:png" validate:"required,oneof=jpeg png gif bmp tiff"`
}

// ImagesConvertResp 图片格式转换响应结构体
type ImagesConvertResp struct {
	// 转换后的文件列表
	OutputFiles string `json:"output_files" widget:"name:转换后的图片;type:files"`

	// 转换统计信息
	ConvertStats string `json:"convert_stats" widget:"name:转换统计;type:text_area"`
}

// ImagesConvert 图片格式转换入口（SDK 注册用）：解析请求 → 调 DoImagesConvert → 写响应
func ImagesConvert(ctx *app.Context, resp response.Response) error {
	var req ImagesConvertReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoImagesConvert(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoImagesConvert 图片格式转换业务逻辑：(ctx, req) → (res, err)，便于单测与复用
func DoImagesConvert(ctx *app.Context, req *ImagesConvertReq) (*ImagesConvertResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	outputFilePaths := make([]string, 0)
	successCount := 0
	failCount := 0
	var errors []string

	for _, file := range inputFiles {
		if file == "" {
			logger.Warnf(ctx, "[ImagesConvert] 文件 %s 没有本地路径，跳过", filepath.Base(file))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 本地路径为空", filepath.Base(file)))
			continue
		}

		outputPath, err := convertImageFormat(ctx, fs, file, req.TargetFormat)
		if err != nil {
			logger.Errorf(ctx, "[ImagesConvert] 转换图片失败 %s: %v, req: %+v", filepath.Base(file), err, req)
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: %v", filepath.Base(file), err))
			continue
		}

		outputFilePaths = append(outputFilePaths, outputPath)
		successCount++
	}

	var outputFiles string
	if len(outputFilePaths) > 0 {
		outputFiles = fs.ResponseFiles(outputFilePaths)
	}

	stats := fmt.Sprintf("转换完成！\n成功: %d 个\n失败: %d 个", successCount, failCount)
	if len(errors) > 0 {
		stats += "\n\n失败详情:\n" + strings.Join(errors, "\n")
	}

	return &ImagesConvertResp{
		OutputFiles:  outputFiles,
		ConvertStats: stats,
	}, nil
}

// convertImageFormat 转换图片格式（使用 ImageMagick）
func convertImageFormat(ctx *app.Context, fs *app.FS, inputPath string, targetFormat string) (string, error) {
	// 直接使用 convert，依赖 PATH（canonical Ubuntu 镜像中已安装 imagemagick）
	imCmd := "convert"

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

	// 使用 ImageMagick 转换图片格式
	// convert input.jpg output.png
	cmd := exec.Command(imCmd, inputPath, outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ImageMagick 转换失败: %v, output: %s", err, string(output))
	}

	logger.Infof(ctx, "[convertImageFormat] 转换成功: %s -> %s", inputPath, outputPath)
	return outputPath, nil
}

// ImagesConvertTemplate 图片格式转换配置
var ImagesConvertTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "图片格式转换",
		Desc:     `支持将图片转换为JPEG、PNG、GIF、BMP、TIFF等多种格式。可以批量处理多个图片文件。默认使用 ImageMagick（'convert'）进行格式转换；环境中也保留了 gm 兼容旧脚本。注意：JPEG和JPG是同一格式，统一使用jpeg。应用场景：图片格式统一、文件大小优化、兼容性转换等。`,
		Tags:     []string{"图片处理", "格式转换", "工具"},
		Request:  &ImagesConvertReq{},
		Response: &ImagesConvertResp{},
	},
}

func init() {
	// 💡 packageContext 是在当前目录下系统自动创建的变量，直接用即可，无需定义
	packageContext.POST("convert.form", ImagesConvert, ImagesConvertTemplate)
}
```

### images_resize.go

```go
//<文件名>images_resize.go</文件名>

package images

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
)

// ImagesResizeReq 图片裁剪/缩放请求结构体
type ImagesResizeReq struct {
	// 框架标签：widget:"type:files;accept:image/*;max_size:50MB;max_count:10" - 文件上传组件，支持多文件上传
	InputFiles string `json:"input_files" widget:"name:上传图片;type:files;accept:image/*;max_size:50MB;max_count:10" validate:"required"`

	// 框架标签：select 须配 options_colors，与 options 一一对应，前端用颜色区分选项
	TargetSize string `json:"target_size" widget:"name:目标尺寸;type:select;options:1920x1080,1280x720,800x600,640x480,自定义;options_colors:409EFF,67C23A,909399,E6A23C,9E9E9E;render_default:1920x1080" validate:"required"`

	// 框架标签：widget:"type:input;placeholder:例如:800x600" - 自定义尺寸（当选择"自定义"时使用）
	CustomSize string `json:"custom_size" widget:"name:自定义尺寸;type:input;placeholder:例如:800x600"`

	// 框架标签：select 须配 options_colors，与 options 一一对应，前端用颜色区分选项
	ResizeMode string `json:"resize_mode" widget:"name:缩放模式;type:select;options:保持宽高比,拉伸填充,裁剪填充;options_colors:67C23A,409EFF,E6A23C;render_default:保持宽高比" validate:"required,oneof=保持宽高比 拉伸填充 裁剪填充"`
}

// ImagesResizeResp 图片裁剪/缩放响应结构体
type ImagesResizeResp struct {
	// 处理后的文件列表
	OutputFiles string `json:"output_files" widget:"name:处理后的图片;type:files"`

	// 处理统计信息
	ResizeStats string `json:"resize_stats" widget:"name:处理统计;type:text_area"`
}

// ImagesResize 图片裁剪/缩放入口（SDK 注册用）：解析请求 → 调 DoImagesResize → 写响应
func ImagesResize(ctx *app.Context, resp response.Response) error {
	var req ImagesResizeReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoImagesResize(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoImagesResize 图片裁剪/缩放业务逻辑：(ctx, req) → (res, err)，便于单测与复用
func DoImagesResize(ctx *app.Context, req *ImagesResizeReq) (*ImagesResizeResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	targetSize := req.TargetSize
	if targetSize == "自定义" {
		targetSize = req.CustomSize
	}
	if targetSize == "" {
		return nil, fmt.Errorf("目标尺寸不能为空")
	}

	sizeParts := strings.Split(targetSize, "x")
	if len(sizeParts) != 2 {
		return nil, fmt.Errorf("目标尺寸格式错误，应为 宽x高，例如: 1920x1080")
	}
	width := strings.TrimSpace(sizeParts[0])
	height := strings.TrimSpace(sizeParts[1])

	if _, err := strconv.Atoi(width); err != nil {
		return nil, fmt.Errorf("宽度必须是数字: %s", width)
	}
	if _, err := strconv.Atoi(height); err != nil {
		return nil, fmt.Errorf("高度必须是数字: %s", height)
	}

	// 直接使用 convert，依赖 PATH（canonical Ubuntu 镜像中已安装 imagemagick）
	imCmd := "convert"

	outputDir := fs.GetTraceOutputDir()

	outputFilePaths := make([]string, 0)
	successCount := 0
	failCount := 0
	var errors []string

	for _, file := range inputFiles {
		if file == "" {
			logger.Warnf(ctx, "[ImagesResize] 文件 %s 没有本地路径，跳过", filepath.Base(file))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 本地路径为空", filepath.Base(file)))
			continue
		}

		baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		ext := filepath.Ext(file)
		outputPath := filepath.Join(outputDir, baseName+"_resized"+ext)

		var args []string
		args = append(args, "convert")

		switch req.ResizeMode {
		case "保持宽高比":
			args = append(args, "-resize", fmt.Sprintf("%sx%s", width, height))
		case "拉伸填充":
			args = append(args, "-resize", fmt.Sprintf("%sx%s!", width, height))
		case "裁剪填充":
			args = append(args, "-resize", fmt.Sprintf("%sx%s^", width, height))
			args = append(args, "-gravity", "center")
			args = append(args, "-crop", fmt.Sprintf("%sx%s+0+0", width, height))
		}

		args = append(args, file, outputPath)

		cmd := exec.Command(imCmd, args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[ImagesResize] 处理图片失败 %s: %v, req: %+v, output: %s", filepath.Base(file), err, req, string(output))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: %v", filepath.Base(file), err))
			continue
		}

		outputFilePaths = append(outputFilePaths, outputPath)
		successCount++
		logger.Infof(ctx, "[ImagesResize] 处理成功: %s -> %s (尺寸: %sx%s, 模式: %s)", filepath.Base(file), filepath.Base(outputPath), width, height, req.ResizeMode)
	}

	var outputFiles string
	if len(outputFilePaths) > 0 {
		outputFiles = fs.ResponseFiles(outputFilePaths)
	}

	stats := fmt.Sprintf("处理完成！\n成功: %d 个\n失败: %d 个\n目标尺寸: %sx%s\n缩放模式: %s",
		successCount, failCount, width, height, req.ResizeMode)
	if len(errors) > 0 {
		stats += "\n\n失败详情:\n" + strings.Join(errors, "\n")
	}

	return &ImagesResizeResp{
		OutputFiles: outputFiles,
		ResizeStats: stats,
	}, nil
}

// ImagesResizeTemplate 图片裁剪/缩放配置
var ImagesResizeTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "图片裁剪/缩放",
		Desc:     `调整图片尺寸，支持裁剪和缩放。可以设置目标尺寸和缩放模式（保持宽高比、拉伸填充、裁剪填充）。支持批量处理多个图片文件。默认使用 ImageMagick（'convert'）进行处理；环境中也保留了 gm 兼容旧脚本。应用场景：生成缩略图、调整图片尺寸、适配不同设备、图片裁剪等。`,
		Tags:     []string{"图片处理", "裁剪缩放", "工具"},
		Request:  &ImagesResizeReq{},
		Response: &ImagesResizeResp{},
	},
}

func init() {
	// 注册Form函数 - 图片裁剪/缩放
	packageContext.POST("resize.form", ImagesResize, ImagesResizeTemplate)
}
```

### images_run_command.go

```go
//<文件名>images_run_command.go</文件名>

package images

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
)

// ImagesRunCommandReq 自定义命令请求：上传图片 + 命令模板（占位符替换后执行），便于智能体灵活调用
type ImagesRunCommandReq struct {
	InputFiles string `json:"input_files" widget:"name:上传图片;type:files;accept:image/*,*/*;max_size:50MB;max_count:10" validate:"required"`

	// 命令模板，占位符：{{input}}=当前输入文件路径，{{output}}=当前输出文件路径。环境有 convert（ImageMagick）、gm 等
	CommandTemplate string `json:"command_template" widget:"name:命令模板;type:text_area;placeholder:convert {{input}} -resize 800x600 {{output}}" validate:"required"`

	// 输出文件扩展名，用于生成 {{output}} 路径
	OutputExtension string `json:"output_extension" widget:"name:输出扩展名;type:input;render_default:png" validate:"required"`
}

// ImagesRunCommandResp 自定义命令响应
type ImagesRunCommandResp struct {
	OutputFile string `json:"output_file" widget:"name:输出文件;type:files"`
	RunInfo    string       `json:"run_info" widget:"name:执行信息;type:text_area"`
}

// ImagesRunCommand 自定义命令入口（SDK 注册用）
func ImagesRunCommand(ctx *app.Context, resp response.Response) error {
	var req ImagesRunCommandReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoImagesRunCommand(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoImagesRunCommand 按文件逐个替换 {{input}}/{{output}} 并执行，不经过 shell，安全
func DoImagesRunCommand(ctx *app.Context, req *ImagesRunCommandReq) (*ImagesRunCommandResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	outputExt := strings.TrimSpace(req.OutputExtension)
	if outputExt == "" {
		outputExt = "png"
	}
	outputExt = strings.TrimPrefix(outputExt, ".")
	outputDir := fs.GetTraceOutputDir()
	hasOutputPlaceholder := strings.Contains(req.CommandTemplate, "{{output}}")

	var outputPaths []string
	var runInfos []string
	for i, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[ImagesRunCommand] 文件 %s 无本地路径，跳过", filepath.Base(file))
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
			logger.Errorf(ctx, "[ImagesRunCommand] 执行失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			runInfos = append(runInfos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, string(out)))
			continue
		}
		if hasOutputPlaceholder {
			outputPaths = append(outputPaths, outputPath)
		}
		runInfos = append(runInfos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputPath))
		if i == 0 && len(out) > 0 {
			runInfos = append(runInfos, "命令输出:\n"+strings.TrimSpace(string(out)))
		}
	}

	var outputFiles string
	if len(outputPaths) > 0 {
		outputFiles = fs.ResponseFiles(outputPaths)
	}

	return &ImagesRunCommandResp{
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

// ImagesRunCommandTemplate 自定义命令表单配置
var ImagesRunCommandTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name: "图片处理自定义命令",
		Desc: `上传图片后，用自定义命令模板处理（占位符 {{input}}、{{output}} 会替换为实际路径后执行）。不经过 shell，安全。环境有 convert（ImageMagick）、identify、mogrify，也保留 gm 兼容旧脚本；示例：convert {{input}} -resize 800x600 {{output}} 或 convert {{input}} -format png {{output}}。`,
		Tags:     []string{"图片处理", "ImageMagick", "自定义命令", "智能体"},
		Request:  &ImagesRunCommandReq{},
		Response: &ImagesRunCommandResp{},
	},
}

func init() {
	packageContext.POST("run_command.form", ImagesRunCommand, ImagesRunCommandTemplate)
}
```
