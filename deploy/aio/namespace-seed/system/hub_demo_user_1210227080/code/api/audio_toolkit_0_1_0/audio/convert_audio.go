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

type ConvertAudioReq struct {
	InputFiles   string `json:"input_files" widget:"name:上传音频文件;type:files;accept:audio/*,*/*;max_size:1000MB;max_count:50" validate:"required"`
	OutputFormat string `json:"output_format" widget:"name:输出格式;type:select;options:mp3,m4a,wav,ogg,flac;options_colors:409EFF,67C23A,909399,E6A23C,909399;render_default:mp3" validate:"required,oneof=mp3 m4a wav ogg flac"`
	AudioQuality string `json:"audio_quality" widget:"name:音频质量;type:select;options:高质量,标准,低体积;options_colors:67C23A,409EFF,E6A23C;render_default:标准"`
}

type ConvertAudioResp struct {
	OutputFiles string `json:"output_files" widget:"name:转换后的音频;type:files"`
	ConvertInfo string `json:"convert_info" widget:"name:转换信息;type:text_area"`
}

func ConvertAudio(ctx *app.Context, resp response.Response) error {
	var req ConvertAudioReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoConvertAudio(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoConvertAudio(ctx *app.Context, req *ConvertAudioReq) (*ConvertAudioResp, error) {
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
			logger.Warnf(ctx, "[Audio/ConvertAudio] 文件 %s 无本地路径，跳过", filepath.Base(file))
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		outputName := outputFileName(filepath.Base(file), file, "_converted", format, seenNames)
		outputPath := filepath.Join(outputDir, outputName)
		args := []string{"-y", "-i", file, "-vn"}
		args = append(args, audioCodecArgs(format, req.AudioQuality)...)
		args = append(args, outputPath)
		out, err := exec.Command("ffmpeg", args...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[Audio/ConvertAudio] 转换失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputName))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功转换的音频文件\n%s", strings.Join(infos, "\n"))
	}

	outputFiles := fs.ResponseFiles(outputPaths)

	info := fmt.Sprintf("音频转换完成\n输出格式: %s\n质量: %s\n成功: %d\n失败: %d", format, req.AudioQuality, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &ConvertAudioResp{OutputFiles: outputFiles, ConvertInfo: info}, nil
}

var ConvertAudioTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "音频格式转换",
		Desc:     `使用 FFmpeg 将音频转换为 mp3、m4a、wav、ogg 或 flac，支持批量处理和常用质量档位。适合录音格式转换、Web 发布、语音识别前处理。`,
		Tags:     []string{"音频", "格式转换", "FFmpeg", "mp3", "m4a", "wav"},
		Request:  &ConvertAudioReq{},
		Response: &ConvertAudioResp{},
	},
}

func init() {
	packageContext.POST("convert.form", ConvertAudio, ConvertAudioTemplate)
}
