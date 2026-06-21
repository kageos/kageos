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

type OptimizeGIFReq struct {
	InputFiles    string `json:"input_files" widget:"name:上传GIF图片;type:files;accept:.gif,image/gif;max_size:200MB;max_count:50" validate:"required"`
	OptimizeLevel string `json:"optimize_level" widget:"name:优化级别;type:select;options:O1,O2,O3;options_colors:67C23A,409EFF,E6A23C;render_default:O3" validate:"required,oneof=O1 O2 O3"`
}

type OptimizeGIFResp struct {
	OutputFiles  string `json:"output_files" widget:"name:优化后的GIF;type:files"`
	OptimizeInfo string `json:"optimize_info" widget:"name:优化信息;type:text_area"`
}

func OptimizeGIF(ctx *app.Context, resp response.Response) error {
	var req OptimizeGIFReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoOptimizeGIF(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoOptimizeGIF(ctx *app.Context, req *OptimizeGIFReq) (*OptimizeGIFResp, error) {
	if _, err := exec.LookPath("gifsicle"); err != nil {
		return nil, fmt.Errorf("未找到 gifsicle，请确认运行环境已安装 gifsicle")
	}

	level := strings.TrimSpace(req.OptimizeLevel)
	if level == "" {
		level = "O3"
	}

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
			logger.Warnf(ctx, "[ImageOptimize/OptimizeGIF] 文件 %s 无本地路径，跳过", filepath.Base(file))
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		outputName := outputFileName(filepath.Base(file), file, "_optimized", "gif", seenNames)
		outputPath := filepath.Join(outputDir, outputName)
		cmd := exec.Command("gifsicle", "-"+level, file, "-o", outputPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[ImageOptimize/OptimizeGIF] 优化失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputName))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功优化的 GIF 文件\n%s", strings.Join(infos, "\n"))
	}

	outputFiles := fs.ResponseFiles(outputPaths)

	info := fmt.Sprintf("GIF 优化完成\n优化级别: %s\n成功: %d\n失败: %d", level, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &OptimizeGIFResp{OutputFiles: outputFiles, OptimizeInfo: info}, nil
}

var OptimizeGIFTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "GIF 图片优化",
		Desc:     `使用 gifsicle 优化 GIF 动图体积，支持 O1/O2/O3 优化级别。适合动图压缩、表情包优化和网页 GIF 体积控制。`,
		Tags:     []string{"图片", "GIF", "动图", "压缩", "gifsicle", "图片优化"},
		Request:  &OptimizeGIFReq{},
		Response: &OptimizeGIFResp{},
	},
}

func init() {
	packageContext.POST("optimize_gif.form", OptimizeGIF, OptimizeGIFTemplate)
}
