package image

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type CompareImagesReq struct {
	BaseImage      string `json:"base_image" widget:"name:基准图片;type:files;accept:image/*;max_size:200MB;max_count:1" validate:"required"`
	CompareImage   string `json:"compare_image" widget:"name:对比图片;type:files;accept:image/*;max_size:200MB;max_count:1" validate:"required"`
	Metric         string `json:"metric" widget:"name:差异指标;type:select;options:AE,RMSE,MAE;options_colors:409EFF,67C23A,E6A23C;render_default:AE" validate:"required,oneof=AE RMSE MAE"`
	OutputFileName string `json:"output_file_name" widget:"name:输出文件名;type:input;placeholder:可选，例如 diff.png"`
}

type CompareImagesResp struct {
	OutputFiles string `json:"output_files" widget:"name:差异图;type:files"`
	CompareInfo string `json:"compare_info" widget:"name:对比信息;type:text_area"`
}

func CompareImages(ctx *app.Context, resp response.Response) error {
	var req CompareImagesReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoCompareImages(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoCompareImages(ctx *app.Context, req *CompareImagesReq) (*CompareImagesResp, error) {
	if _, err := exec.LookPath("compare"); err != nil {
		return nil, fmt.Errorf("未找到 compare，请确认运行环境已安装 ImageMagick")
	}

	fs := ctx.GetFS()
	baseFiles := fs.DownloadFiles(req.BaseImage)
	defer fs.RemoveFiles(baseFiles)
	compareFiles := fs.DownloadFiles(req.CompareImage)
	defer fs.RemoveFiles(compareFiles)
	if len(baseFiles) == 0 || len(compareFiles) == 0 {
		return nil, fmt.Errorf("需要上传两张图片")
	}
	basePath := baseFiles[0]
	comparePath := compareFiles[0]
	if basePath == "" || comparePath == "" {
		return nil, fmt.Errorf("图片本地路径为空")
	}

	outputName := strings.TrimSpace(req.OutputFileName)
	if outputName == "" {
		outputName = imSafeBase(filepath.Base(basePath), "image") + "_diff.png"
	} else {
		outputName = imEnsureExt(outputName, "png")
	}
	outputPath := filepath.Join(fs.GetTraceOutputDir(), outputName)
	metric := normalizeCompareMetric(req.Metric)
	cmd := exec.Command("compare", "-metric", metric, basePath, comparePath, outputPath)
	out, err := cmd.CombinedOutput()
	metricOutput := strings.TrimSpace(string(out))
	if err != nil {
		if _, statErr := os.Stat(outputPath); statErr != nil {
			logger.Errorf(ctx, "[CompareImages] compare 失败: %v, output: %s", err, string(out))
			return nil, fmt.Errorf("图片对比失败: %v\n%s", err, metricOutput)
		}
	}

	outputFiles := fs.ResponseFiles([]string{outputPath})
	info := fmt.Sprintf("图片对比完成\n基准图片: %s\n对比图片: %s\n指标: %s\n指标输出: %s\n输出文件: %s",
		filepath.Base(basePath), filepath.Base(comparePath), metric, metricOutput, outputName)
	return &CompareImagesResp{OutputFiles: outputFiles, CompareInfo: info}, nil
}

func normalizeCompareMetric(metric string) string {
	switch strings.TrimSpace(strings.ToUpper(metric)) {
	case "RMSE":
		return "RMSE"
	case "MAE":
		return "MAE"
	default:
		return "AE"
	}
}

func imSafeBase(name, fallback string) string {
	name = strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	name = strings.TrimSpace(name)
	if name == "" {
		name = fallback
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_", " ", "_")
	name = replacer.Replace(name)
	if strings.TrimSpace(name) == "" {
		return fallback
	}
	return name
}

func imEnsureExt(name, ext string) string {
	name = imSafeBase(name, "output")
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" {
		ext = "png"
	}
	return name + "." + ext
}

var CompareImagesTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "图片差异对比",
		Desc:     `使用 ImageMagick compare 对比两张图片并生成差异图，支持 AE、RMSE、MAE 指标。适合 UI 截图回归、设计稿对比、图片处理前后效果检查。`,
		Tags:     []string{"图片", "对比", "差异图", "ImageMagick", "compare", "UI验收"},
		Request:  &CompareImagesReq{},
		Response: &CompareImagesResp{},
	},
}

func init() {
	packageContext.POST("compare.form", CompareImages, CompareImagesTemplate)
}
