package audio

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

type WaveformReq struct {
	InputFiles     string `json:"input_files" widget:"name:上传音频或视频文件;type:files;accept:audio/*,video/*,*/*;max_size:1000MB;max_count:20" validate:"required"`
	Width          int    `json:"width" widget:"name:图片宽度;type:integer;min:320;max:4096;render_default:1200" validate:"min=0,max=4096"`
	Height         int    `json:"height" widget:"name:图片高度;type:integer;min:120;max:2048;render_default:320" validate:"min=0,max=2048"`
	WaveColor      string `json:"wave_color" widget:"name:波形颜色;type:input;render_default:#2563eb;placeholder:例如 #2563eb 或 steelblue"`
	OutputFileName string `json:"output_file_name" widget:"name:输出文件名;type:input;placeholder:可选，仅单文件时生效，例如 waveform.png"`
}

type WaveformResp struct {
	OutputFiles  string `json:"output_files" widget:"name:波形图;type:files"`
	WaveformInfo string `json:"waveform_info" widget:"name:生成信息;type:text_area"`
}

func Waveform(ctx *app.Context, resp response.Response) error {
	var req WaveformReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoWaveform(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoWaveform(ctx *app.Context, req *WaveformReq) (*WaveformResp, error) {
	if err := ensureFFmpeg(); err != nil {
		return nil, err
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	width := req.Width
	if width <= 0 {
		width = 1200
	}
	if width > 4096 {
		width = 4096
	}
	height := req.Height
	if height <= 0 {
		height = 320
	}
	if height > 2048 {
		height = 2048
	}
	waveColor := normalizeWaveColor(req.WaveColor)

	outputDir := fs.GetTraceOutputDir()
	seenNames := make(map[string]int)
	var outputPaths []string
	var infos []string
	successCount := 0
	failCount := 0

	for _, file := range inputFiles {
		if file == "" {
			failCount++
			infos = append(infos, "跳过空文件路径")
			continue
		}

		outputName := outputFileName(filepath.Base(file), file, "_waveform", "png", seenNames)
		if len(inputFiles) == 1 && strings.TrimSpace(req.OutputFileName) != "" {
			outputName = sanitizeFileName(strings.TrimSuffix(req.OutputFileName, filepath.Ext(req.OutputFileName)), "waveform") + ".png"
		}
		outputPath := filepath.Join(outputDir, outputName)
		filter := fmt.Sprintf("showwavespic=s=%dx%d:colors=%s", width, height, waveColor)
		args := []string{
			"-y",
			"-i", file,
			"-filter_complex", filter,
			"-frames:v", "1",
			outputPath,
		}
		out, err := exec.Command("ffmpeg", args...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[Audio/Waveform] 生成波形图失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputName))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功生成的波形图\n%s", strings.Join(infos, "\n"))
	}

	info := fmt.Sprintf("音频波形图生成完成\n尺寸: %sx%s\n波形颜色: %s\n成功: %d\n失败: %d", strconv.Itoa(width), strconv.Itoa(height), waveColor, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &WaveformResp{
		OutputFiles:  fs.ResponseFiles(outputPaths),
		WaveformInfo: info,
	}, nil
}

func normalizeWaveColor(color string) string {
	color = strings.TrimSpace(color)
	if color == "" {
		return "#2563eb"
	}
	return color
}

var WaveformTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "生成音频波形图",
		Desc:     `使用 FFmpeg 的 showwavespic 滤镜，把音频或视频音轨渲染为 PNG 波形图。适合给录音、播客、课程视频、会议音频生成可视化预览。`,
		Tags:     []string{"音频", "波形图", "可视化", "FFmpeg", "showwavespic", "PNG"},
		Request:  &WaveformReq{},
		Response: &WaveformResp{},
	},
}

func init() {
	packageContext.POST("waveform.form", Waveform, WaveformTemplate)
}
