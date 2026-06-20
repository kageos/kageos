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

type AudioInfoReq struct {
	InputFiles string `json:"input_files" widget:"name:上传音频或视频文件;type:files;accept:audio/*,video/*,*/*;max_size:1000MB;max_count:20" validate:"required"`
	OutputMode string `json:"output_mode" widget:"name:输出模式;type:select;options:摘要,JSON;options_colors:409EFF,67C23A;render_default:摘要" validate:"required,oneof=摘要 JSON"`
}

type AudioInfoResp struct {
	InfoText string `json:"info_text" widget:"name:音频信息;type:text_area"`
	Summary  string `json:"summary" widget:"name:处理说明;type:text_area"`
}

func AudioInfo(ctx *app.Context, resp response.Response) error {
	var req AudioInfoReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoAudioInfo(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoAudioInfo(ctx *app.Context, req *AudioInfoReq) (*AudioInfoResp, error) {
	if err := ensureFFprobe(); err != nil {
		return nil, err
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	var blocks []string
	var summary []string
	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[Audio/AudioInfo] 文件 %s 无本地路径，跳过", filepath.Base(file))
			summary = append(summary, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		args := []string{
			"-v", "error",
			"-select_streams", "a",
			"-show_entries", "stream=index,codec_name,codec_type,sample_rate,channels,channel_layout,bit_rate,duration:format=format_name,duration,size,bit_rate",
		}
		if req.OutputMode == "JSON" {
			args = append(args, "-of", "json")
		} else {
			args = append(args, "-of", "default=noprint_wrappers=1")
		}
		args = append(args, file)
		out, err := exec.Command("ffprobe", args...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[Audio/AudioInfo] 读取失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			summary = append(summary, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		text := strings.TrimSpace(string(out))
		if text == "" {
			text = "未发现音频流"
		}
		blocks = append(blocks, fmt.Sprintf("## %s\n%s", filepath.Base(file), text))
		summary = append(summary, fmt.Sprintf("成功 %s", filepath.Base(file)))
	}

	if len(blocks) == 0 {
		return nil, fmt.Errorf("没有成功读取的音频信息\n%s", strings.Join(summary, "\n"))
	}
	return &AudioInfoResp{
		InfoText: strings.Join(blocks, "\n\n"),
		Summary:  strings.Join(summary, "\n"),
	}, nil
}

var AudioInfoTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "读取音频信息",
		Desc:     `使用 FFprobe 读取音频流、编码格式、采样率、声道数、码率、时长和容器信息。适合在转码、裁剪、语音识别前检查输入文件。`,
		Tags:     []string{"音频", "信息", "FFprobe", "时长", "码率", "采样率"},
		Request:  &AudioInfoReq{},
		Response: &AudioInfoResp{},
	},
}

func init() {
	packageContext.POST("inspect.form", AudioInfo, AudioInfoTemplate)
}
