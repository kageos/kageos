package image

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// ImageResizeReq 图片裁剪/缩放请求结构体
type ImageResizeReq struct {
	InputFiles string `json:"input_files" widget:"name:上传图片;type:files;accept:image/*;max_size:50MB;max_count:10" validate:"required"`
	TargetSize string `json:"target_size" widget:"name:目标尺寸;type:select;options:1920x1080,1280x720,800x600,640x480,自定义;options_colors:409EFF,67C23A,909399,E6A23C,9E9E9E;render_default:1920x1080" validate:"required"`
	CustomSize string `json:"custom_size" widget:"name:自定义尺寸;type:input;placeholder:例如:800x600;desc:目标尺寸选择自定义后显示并生效" validate:"required_if=TargetSize 自定义"`
	ResizeMode string `json:"resize_mode" widget:"name:缩放模式;type:select;options:保持宽高比,拉伸填充,裁剪填充;options_colors:67C23A,409EFF,E6A23C;render_default:保持宽高比" validate:"required,oneof=保持宽高比 拉伸填充 裁剪填充"`
}

// ImageResizeResp 图片裁剪/缩放响应结构体
type ImageResizeResp struct {
	OutputFiles string `json:"output_files" widget:"name:处理后的图片;type:files"`
	ResizeStats string `json:"resize_stats" widget:"name:处理统计;type:text_area"`
}

// ImageResize 图片裁剪/缩放入口
func ImageResize(ctx *app.Context, resp response.Response) error {
	var req ImageResizeReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoImageResize(ctx, &req)
	if err != nil {
		return err
	}

	return resp.Form(res).Build()
}

// DoImageResize 图片裁剪/缩放业务逻辑
func DoImageResize(ctx *app.Context, req *ImageResizeReq) (*ImageResizeResp, error) {
	const imCmd = "convert"

	if _, err := exec.LookPath(imCmd); err != nil {
		logger.Errorf(ctx, "[系统错误]-[ImageResize] ImageMagick 未在 PATH 中, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[ImageResize]： ImageMagick未安装或不在 PATH 中，请确保运行环境已安装 ImageMagick")
	}

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

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	outputDir := fs.GetTraceOutputDir()
	outputFilePaths := make([]string, 0)
	successCount := 0
	failCount := 0
	var errors []string

	for _, file := range inputFiles {
		if file == "" {
			logger.Warnf(ctx, "[ImageResize] 文件 %s 没有本地路径，跳过", filepath.Base(file))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 本地路径为空", filepath.Base(file)))
			continue
		}

		baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		outputPath := filepath.Join(outputDir, baseName+"_resized"+filepath.Ext(file))

		args := []string{file}
		switch req.ResizeMode {
		case "保持宽高比":
			args = append(args, "-resize", fmt.Sprintf("%sx%s", width, height))
		case "拉伸填充":
			args = append(args, "-resize", fmt.Sprintf("%sx%s!", width, height))
		case "裁剪填充":
			args = append(args,
				"-resize", fmt.Sprintf("%sx%s^", width, height),
				"-gravity", "center",
				"-crop", fmt.Sprintf("%sx%s+0+0", width, height),
				"+repage",
			)
		}
		args = append(args, outputPath)

		cmd := exec.Command(imCmd, args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[ImageResize] 处理失败 %s: %v, req: %+v, output: %s", filepath.Base(file), err, req, string(output))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: %v", filepath.Base(file), err))
			continue
		}

		outputFilePaths = append(outputFilePaths, outputPath)
		successCount++
		logger.Infof(ctx, "[ImageResize] 处理成功: %s -> %s (尺寸: %sx%s, 模式: %s)", filepath.Base(file), filepath.Base(outputPath), width, height, req.ResizeMode)
	}

	var outputFiles string
	if len(outputFilePaths) > 0 {
		outputFiles = fs.ResponseFiles(outputFilePaths)
	}

	stats := fmt.Sprintf("处理完成！\n成功: %d 个\n失败: %d 个\n目标尺寸: %sx%s\n缩放模式: %s", successCount, failCount, width, height, req.ResizeMode)
	if len(errors) > 0 {
		stats += "\n\n失败详情:\n" + strings.Join(errors, "\n")
	}

	return &ImageResizeResp{
		OutputFiles: outputFiles,
		ResizeStats: stats,
	}, nil
}

var ImageResizeTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "图片裁剪/缩放",
		Desc:     `调整图片尺寸，支持保持宽高比、拉伸填充、裁剪填充三种模式。默认直接调用 ImageMagick 的 convert 命令。`,
		Tags:     []string{"图片", "图片处理", "裁剪", "缩放", "ImageMagick", "convert"},
		Request:  &ImageResizeReq{},
		Response: &ImageResizeResp{},
	},
}

func init() {
	packageContext.POST("resize.form", ImageResize, ImageResizeTemplate)
}
