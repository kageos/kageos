package image

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type InspectReq struct {
	InputFiles   string `json:"input_files" widget:"name:上传图片;type:files;accept:image/*,*/*;max_size:500MB;max_count:50" validate:"required"`
	OutputReport bool   `json:"output_report" widget:"name:输出体检报告文件;type:switch;render_default:true"`
}

type InspectResp struct {
	ReportText string `json:"report_text" widget:"name:图片体检报告;type:text_area"`
	OutputFile string `json:"output_file" widget:"name:体检报告;type:files"`
	Summary    string `json:"summary" widget:"name:处理说明;type:text_area"`
}

func Inspect(ctx *app.Context, resp response.Response) error {
	var req InspectReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoInspect(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoInspect(ctx *app.Context, req *InspectReq) (*InspectResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	var blocks []string
	var summaries []string
	for _, path := range inputFiles {
		if path == "" {
			continue
		}
		block, err := inspectImageFile(path)
		if err != nil {
			summaries = append(summaries, fmt.Sprintf("失败 %s: %v", filepath.Base(path), err))
			continue
		}
		blocks = append(blocks, block)
		summaries = append(summaries, fmt.Sprintf("成功 %s", filepath.Base(path)))
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("没有成功读取的图片信息\n%s", strings.Join(summaries, "\n"))
	}

	report := strings.TrimSpace(strings.Join(blocks, "\n\n"))
	var outputFile string
	if req.OutputReport {
		outputPath := filepath.Join(fs.GetTraceOutputDir(), "image_inspection_report.txt")
		if err := os.WriteFile(outputPath, []byte(report), 0644); err != nil {
			return nil, fmt.Errorf("写入体检报告失败: %w", err)
		}
		outputFile = fs.ResponseFiles([]string{outputPath})
	}
	return &InspectResp{
		ReportText: report,
		OutputFile: outputFile,
		Summary:    "图片体检完成\n" + strings.Join(summaries, "\n"),
	}, nil
}

func inspectImageFile(path string) (string, error) {
	var parts []string
	parts = append(parts, "## "+filepath.Base(path))
	if stat, err := os.Stat(path); err == nil {
		parts = append(parts, fmt.Sprintf("大小: %d bytes", stat.Size()))
	}
	if out, err := exec.Command("identify", "-verbose", path).CombinedOutput(); err == nil {
		parts = append(parts, "### identify\n"+trimImageInspection(string(out), 30000))
	} else {
		parts = append(parts, fmt.Sprintf("identify 失败: %v\n%s", err, strings.TrimSpace(string(out))))
	}
	if _, err := exec.LookPath("exiftool"); err == nil {
		if out, err := exec.Command("exiftool", path).CombinedOutput(); err == nil {
			parts = append(parts, "### exiftool\n"+trimImageInspection(string(out), 20000))
		}
	}
	return strings.Join(parts, "\n"), nil
}

func trimImageInspection(text string, max int) string {
	text = strings.TrimSpace(text)
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max]) + "\n...（内容过长，已截断）"
}

var InspectTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "图片体检",
		Desc:     `读取图片格式、尺寸、色彩空间、通道、压缩信息和 EXIF 元数据。底层优先使用 ImageMagick identify，并在可用时补充 exiftool 信息。`,
		Tags:     []string{"图片", "体检", "元数据", "EXIF", "ImageMagick", "identify"},
		Request:  &InspectReq{},
		Response: &InspectResp{},
	},
}

func init() {
	packageContext.POST("inspect.form", Inspect, InspectTemplate)
}
