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

type CompressPNGReq struct {
	InputFiles string `json:"input_files" widget:"name:上传PNG图片;type:files;accept:.png,image/png;max_size:200MB;max_count:50" validate:"required"`
	MinQuality int    `json:"min_quality" widget:"name:最低质量;type:integer;min:1;max:100;render_default:65" validate:"min=0,max=100"`
	MaxQuality int    `json:"max_quality" widget:"name:最高质量;type:integer;min:1;max:100;render_default:85" validate:"min=0,max=100"`
}

type CompressPNGResp struct {
	OutputFiles  string `json:"output_files" widget:"name:压缩后的PNG;type:files"`
	CompressInfo string `json:"compress_info" widget:"name:压缩信息;type:text_area"`
}

func CompressPNG(ctx *app.Context, resp response.Response) error {
	var req CompressPNGReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoCompressPNG(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoCompressPNG(ctx *app.Context, req *CompressPNGReq) (*CompressPNGResp, error) {
	if _, err := exec.LookPath("pngquant"); err != nil {
		return nil, fmt.Errorf("未找到 pngquant，请确认运行环境已安装 pngquant")
	}

	minQ := clampQuality(req.MinQuality, 65)
	maxQ := clampQuality(req.MaxQuality, 85)
	if minQ > maxQ {
		minQ, maxQ = maxQ, minQ
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
			logger.Warnf(ctx, "[ImageOptimize/CompressPNG] 文件 %s 无本地路径，跳过", filepath.Base(file))
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		outputName := outputFileName(filepath.Base(file), file, "_compressed", "png", seenNames)
		outputPath := filepath.Join(outputDir, outputName)
		qualityRange := strconv.Itoa(minQ) + "-" + strconv.Itoa(maxQ)
		cmd := exec.Command("pngquant", "--quality", qualityRange, "--output", outputPath, "--force", file)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[ImageOptimize/CompressPNG] 压缩失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputName))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功压缩的 PNG 文件\n%s", strings.Join(infos, "\n"))
	}

	outputFiles := fs.ResponseFiles(outputPaths)

	info := fmt.Sprintf("PNG 压缩完成\n质量范围: %d-%d\n成功: %d\n失败: %d", minQ, maxQ, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &CompressPNGResp{OutputFiles: outputFiles, CompressInfo: info}, nil
}

var CompressPNGTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "PNG 图片压缩",
		Desc:     `使用 pngquant 压缩 PNG 图片，可设置质量范围。适合网页 PNG、截图、透明图标等文件体积优化。`,
		Tags:     []string{"图片", "PNG", "压缩", "pngquant", "图片优化"},
		Request:  &CompressPNGReq{},
		Response: &CompressPNGResp{},
	},
}

func init() {
	packageContext.POST("compress_png.form", CompressPNG, CompressPNGTemplate)
}
