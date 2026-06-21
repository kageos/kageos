package image

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
)

// ImageConvertReq 图片格式转换请求结构体
type ImageConvertReq struct {
	InputFiles   string `json:"input_files" widget:"name:上传图片;type:files;accept:image/*;max_size:50MB;max_count:10" validate:"required"`
	TargetFormat string `json:"target_format" widget:"name:目标格式;type:select;options:jpeg,png,gif,bmp,tiff;options_colors:E91E63,4CAF50,FF9800,2196F3,9E9E9E;render_default:png" validate:"required,oneof=jpeg png gif bmp tiff"`
}

// ImageConvertResp 图片格式转换响应结构体
type ImageConvertResp struct {
	OutputFiles  string `json:"output_files" widget:"name:转换后的图片;type:files"`
	ConvertStats string `json:"convert_stats" widget:"name:转换统计;type:text_area"`
}

// ImageConvert 图片格式转换入口
func ImageConvert(ctx *app.Context, resp response.Response) error {
	var req ImageConvertReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoImageConvert(ctx, &req)
	if err != nil {
		return err
	}

	return resp.Form(res).Build()
}

// DoImageConvert 图片格式转换业务逻辑
func DoImageConvert(ctx *app.Context, req *ImageConvertReq) (*ImageConvertResp, error) {
	const imCmd = "convert"

	if _, err := exec.LookPath(imCmd); err != nil {
		logger.Errorf(ctx, "[系统错误]-[ImageConvert] ImageMagick 未在 PATH 中, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[ImageConvert]： ImageMagick未安装或不在 PATH 中，请确保运行环境已安装 ImageMagick")
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	outputFilePaths := make([]string, 0)
	successCount := 0
	failCount := 0
	var errors []string

	for _, file := range inputFiles {
		if file == "" {
			logger.Warnf(ctx, "[ImageConvert] 文件 %s 没有本地路径，跳过", filepath.Base(file))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 本地路径为空", filepath.Base(file)))
			continue
		}

		outputPath, err := convertImageFormat(ctx, fs, file, req.TargetFormat)
		if err != nil {
			logger.Errorf(ctx, "[ImageConvert] 转换失败 %s: %v, req: %+v", filepath.Base(file), err, req)
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

	stats := fmt.Sprintf("转换完成！\n成功: %d 个\n失败: %d 个\n目标格式: %s", successCount, failCount, req.TargetFormat)
	if len(errors) > 0 {
		stats += "\n\n失败详情:\n" + strings.Join(errors, "\n")
	}

	return &ImageConvertResp{
		OutputFiles:  outputFiles,
		ConvertStats: stats,
	}, nil
}

func convertImageFormat(ctx *app.Context, fs *app.FS, inputPath, targetFormat string) (string, error) {
	const imCmd = "convert"

	outputDir := fs.GetTraceOutputDir()
	normalizedTarget := normalizeImageFormat(targetFormat)
	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outputPath := filepath.Join(outputDir, baseName+"."+normalizedTarget)

	inputExt := normalizeImageFormat(strings.TrimPrefix(filepath.Ext(inputPath), "."))
	if inputExt == normalizedTarget {
		if err := copyFile(inputPath, outputPath); err != nil {
			return "", fmt.Errorf("复制同格式文件失败: %w", err)
		}
		logger.Infof(ctx, "[convertImageFormat] 格式相同，复制到输出目录: %s -> %s", inputPath, outputPath)
		return outputPath, nil
	}

	cmd := exec.Command(imCmd, inputPath, outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ImageMagick 转换失败: %v, output: %s", err, string(output))
	}

	logger.Infof(ctx, "[convertImageFormat] 转换成功: %s -> %s", inputPath, outputPath)
	return outputPath, nil
}

func normalizeImageFormat(format string) string {
	normalized := strings.ToLower(strings.TrimSpace(format))
	if normalized == "jpg" {
		return "jpeg"
	}
	if normalized == "tif" {
		return "tiff"
	}
	return normalized
}

func copyFile(srcPath, dstPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		dstFile.Close()
		return err
	}

	return dstFile.Close()
}

var ImageConvertTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "图片格式转换",
		Desc:     `支持将图片转换为 JPEG、PNG、GIF、BMP、TIFF 等多种格式。默认直接调用 ImageMagick 的 convert 命令；即使目标格式与输入格式相同，也会先复制到输出目录后再作为附件返回。`,
		Tags:     []string{"图片", "图片处理", "格式转换", "ImageMagick", "convert"},
		Request:  &ImageConvertReq{},
		Response: &ImageConvertResp{},
	},
}

func init() {
	packageContext.POST("convert.form", ImageConvert, ImageConvertTemplate)
}
