package audio

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type CutAudioReq struct {
	InputFiles      string  `json:"input_files" widget:"name:上传音频或视频文件;type:files;accept:audio/*,video/*,*/*;max_size:1000MB;max_count:20" validate:"required"`
	StartSeconds    float64 `json:"start_seconds" widget:"name:开始时间(秒);type:float;min:0;step:0.1;render_default:0" validate:"min=0"`
	DurationSeconds float64 `json:"duration_seconds" widget:"name:持续时间(秒);type:float;min:0;step:0.1;render_default:0;placeholder:0 表示裁到结尾" validate:"min=0"`
	OutputFormat    string  `json:"output_format" widget:"name:输出格式;type:select;options:mp3,m4a,wav,ogg,flac;options_colors:409EFF,67C23A,909399,E6A23C,909399;render_default:mp3" validate:"required,oneof=mp3 m4a wav ogg flac"`
	AudioQuality    string  `json:"audio_quality" widget:"name:音频质量;type:select;options:高质量,标准,低体积;options_colors:67C23A,409EFF,E6A23C;render_default:标准"`
}

type CutAudioResp struct {
	OutputFiles string `json:"output_files" widget:"name:裁剪后的音频;type:files"`
	CutInfo     string `json:"cut_info" widget:"name:裁剪信息;type:text_area"`
}

func CutAudio(ctx *app.Context, resp response.Response) error {
	var req CutAudioReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoCutAudio(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoCutAudio(ctx *app.Context, req *CutAudioReq) (*CutAudioResp, error) {
	if err := ensureFFmpeg(); err != nil {
		return nil, err
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	format := normalizeAudioFormat(req.OutputFormat)
	outputDir := fs.GetTraceOutputDir()
	seenNames := make(map[string]int)
	var outputPaths []string
	var infos []string
	successCount := 0
	failCount := 0

	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[Audio/CutAudio] 文件 %s 无本地路径，跳过", filepath.Base(file))
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		outputName := outputFileName(filepath.Base(file), file, "_clip", format, seenNames)
		outputPath := filepath.Join(outputDir, outputName)
		args := []string{"-y"}
		if req.StartSeconds > 0 {
			args = append(args, "-ss", formatSeconds(req.StartSeconds))
		}
		args = append(args, "-i", file)
		if req.DurationSeconds > 0 {
			args = append(args, "-t", formatSeconds(req.DurationSeconds))
		}
		args = append(args, "-vn")
		args = append(args, audioCodecArgs(format, req.AudioQuality)...)
		args = append(args, outputPath)
		out, err := exec.Command("ffmpeg", args...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[Audio/CutAudio] 裁剪失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputName))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功裁剪的音频文件\n%s", strings.Join(infos, "\n"))
	}

	outputFiles := fs.ResponseFiles(outputPaths)

	durationLabel := "到文件结尾"
	if req.DurationSeconds > 0 {
		durationLabel = formatSeconds(req.DurationSeconds) + " 秒"
	}
	info := fmt.Sprintf("音频裁剪完成\n开始时间: %s 秒\n持续时间: %s\n输出格式: %s\n成功: %d\n失败: %d", formatSeconds(req.StartSeconds), durationLabel, format, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &CutAudioResp{OutputFiles: outputFiles, CutInfo: info}, nil
}

var CutAudioTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "裁剪音频片段",
		Desc:     `使用 FFmpeg 按开始时间和持续时间裁剪音频，也可从视频中截取一段音轨。适合截取会议片段、播客片段、语音识别分段和素材剪辑。`,
		Tags:     []string{"音频", "裁剪", "截取片段", "FFmpeg", "剪辑"},
		Request:  &CutAudioReq{},
		Response: &CutAudioResp{},
	},
}

func init() {
	packageContext.POST("cut.form", CutAudio, CutAudioTemplate)
}
