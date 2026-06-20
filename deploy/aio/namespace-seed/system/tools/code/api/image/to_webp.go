package image

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type ToWebPReq struct {
	InputFiles string `json:"input_files" widget:"name:上传图片;type:files;accept:image/*;max_size:200MB;max_count:50" validate:"required"`
	Quality    int    `json:"quality" widget:"name:质量;type:slider;min:1;max:100;unit:%;render_default:80" validate:"min=0,max=100"`
}

type ToWebPResp struct {
	OutputFiles string `json:"output_files" widget:"name:WebP图片;type:files"`
	ConvertInfo string `json:"convert_info" widget:"name:转换信息;type:text_area"`
}

func ToWebP(ctx *app.Context, resp response.Response) error {
	var req ToWebPReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoToWebP(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoToWebP(ctx *app.Context, req *ToWebPReq) (*ToWebPResp, error) {
	if _, err := exec.LookPath("cwebp"); err != nil {
		return nil, fmt.Errorf("未找到 cwebp，请确认运行环境已安装 WebP tools")
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	outputDir := fs.GetTraceOutputDir()
	quality := clampQuality(req.Quality, 80)
	seenNames := make(map[string]int)
	var outputPaths []string
	var infos []string
	successCount := 0
	failCount := 0

	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[ImageOptimize/ToWebP] 文件 %s 无本地路径，跳过", filepath.Base(file))
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		outputName := outputFileName(filepath.Base(file), file, "_webp", "webp", seenNames)
		outputPath := filepath.Join(outputDir, outputName)
		cmd := exec.Command("cwebp", "-q", strconv.Itoa(quality), file, "-o", outputPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[ImageOptimize/ToWebP] 转换失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputName))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功转换的 WebP 文件\n%s", strings.Join(infos, "\n"))
	}

	outputFiles := fs.ResponseFiles(outputPaths)

	info := fmt.Sprintf("WebP 转换完成\n质量: %d\n成功: %d\n失败: %d", quality, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &ToWebPResp{OutputFiles: outputFiles, ConvertInfo: info}, nil
}

var ToWebPTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "图片转 WebP",
		Desc:     `使用 cwebp 将图片转换为 WebP 格式，可配置质量参数。适合 Web 交付、页面图片压缩和批量图片格式优化。`,
		Tags:     []string{"图片", "WebP", "压缩", "cwebp", "格式转换", "图片优化"},
		Request:  &ToWebPReq{},
		Response: &ToWebPResp{},
	},
}

func init() {
	packageContext.POST("to_webp.form", ToWebP, ToWebPTemplate)
}
