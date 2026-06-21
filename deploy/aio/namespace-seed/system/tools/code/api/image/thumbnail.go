package image

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type ThumbnailReq struct {
	InputFiles   string `json:"input_files" widget:"name:上传图片;type:files;accept:image/*;max_size:500MB;max_count:100" validate:"required"`
	Size         string `json:"size" widget:"name:缩略图尺寸;type:input;render_default:512x512;placeholder:例如 512x512 或 512"`
	OutputFormat string `json:"output_format" widget:"name:输出格式;type:select;options:jpeg,png,webp;options_colors:409EFF,67C23A,909399;render_default:jpeg" validate:"required,oneof=jpeg png webp"`
	CropMode     string `json:"crop_mode" widget:"name:裁剪模式;type:select;options:保持比例,智能裁剪填充;options_colors:409EFF,E6A23C;render_default:保持比例" validate:"required,oneof=保持比例 智能裁剪填充"`
}

type ThumbnailResp struct {
	OutputFiles   string `json:"output_files" widget:"name:缩略图;type:files"`
	ThumbnailInfo string `json:"thumbnail_info" widget:"name:生成信息;type:text_area"`
}

func Thumbnail(ctx *app.Context, resp response.Response) error {
	var req ThumbnailReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoThumbnail(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoThumbnail(ctx *app.Context, req *ThumbnailReq) (*ThumbnailResp, error) {
	if _, err := exec.LookPath("vipsthumbnail"); err != nil {
		return nil, fmt.Errorf("未找到 vipsthumbnail，请确认运行环境已安装 libvips-tools")
	}

	size := strings.TrimSpace(req.Size)
	if size == "" {
		size = "512x512"
	}
	format := normalizeOutputFormat(req.OutputFormat)

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	outputDir := fs.GetTraceOutputDir()
	seenNames := make(map[string]int)
	var outputPaths []string
	var infos []string
	successCount := 0
	failCount := 0

	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[ImageOptimize/Thumbnail] 文件 %s 无本地路径，跳过", filepath.Base(file))
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		outputName := outputFileName(filepath.Base(file), file, "_thumb", format, seenNames)
		outputPath := filepath.Join(outputDir, outputName)
		args := []string{file, "--size", size, "--output", outputPath}
		if req.CropMode == "智能裁剪填充" {
			args = append(args, "--smartcrop", "attention")
		}
		cmd := exec.Command("vipsthumbnail", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[ImageOptimize/Thumbnail] 生成失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputName))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功生成的缩略图\n%s", strings.Join(infos, "\n"))
	}

	outputFiles := fs.ResponseFiles(outputPaths)

	info := fmt.Sprintf("缩略图生成完成\n尺寸: %s\n输出格式: %s\n裁剪模式: %s\n成功: %d\n失败: %d", size, format, req.CropMode, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &ThumbnailResp{OutputFiles: outputFiles, ThumbnailInfo: info}, nil
}

var ThumbnailTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "生成图片缩略图",
		Desc:     `使用 vipsthumbnail 生成缩略图，适合大图、批量图片和低内存缩略图场景。支持保持比例或智能裁剪填充，支持 jpeg/png/webp 输出。`,
		Tags:     []string{"图片", "缩略图", "libvips", "vipsthumbnail", "图片优化", "批量处理"},
		Request:  &ThumbnailReq{},
		Response: &ThumbnailResp{},
	},
}

func init() {
	packageContext.POST("thumbnail.form", Thumbnail, ThumbnailTemplate)
}
