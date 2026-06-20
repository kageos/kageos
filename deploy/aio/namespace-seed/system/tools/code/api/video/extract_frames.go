package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type ExtractFramesReq struct {
	InputFiles       string `json:"input_files" widget:"name:上传视频文件;type:files;accept:video/*,*/*;max_size:2000MB;max_count:20" validate:"required"`
	Mode             string `json:"mode" widget:"name:抽帧模式;type:select;options:按间隔抽帧,指定时间点;options_colors:409EFF,67C23A;render_default:按间隔抽帧" validate:"required,oneof=按间隔抽帧 指定时间点"`
	IntervalSeconds  int    `json:"interval_seconds" widget:"name:间隔秒数;type:integer;min:1;max:3600;render_default:10;desc:按间隔抽帧模式显示并生效" validate:"required_if=Mode 按间隔抽帧,min=0,max=3600"`
	Timestamps       string `json:"timestamps" widget:"name:指定时间点;type:text_area;placeholder:支持每行一个或逗号分隔，例如：1.5\n00:00:10\n00:01:20;desc:指定时间点模式显示并生效" validate:"required_if=Mode 指定时间点"`
	MaxFrames        int    `json:"max_frames" widget:"name:最多输出帧数;type:integer;min:1;max:200;render_default:30" validate:"min=0,max=200"`
	OutputImageWidth int    `json:"output_image_width" widget:"name:输出图片宽度;type:integer;min:0;max:4096;render_default:0;placeholder:0 表示保持原宽度" validate:"min=0,max=4096"`
}

type ExtractFramesResp struct {
	OutputFiles string `json:"output_files" widget:"name:抽取的帧图片;type:files"`
	ExtractInfo string `json:"extract_info" widget:"name:抽帧信息;type:text_area"`
}

func ExtractFrames(ctx *app.Context, resp response.Response) error {
	var req ExtractFramesReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoExtractFrames(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoExtractFrames(ctx *app.Context, req *ExtractFramesReq) (*ExtractFramesResp, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("未找到 ffmpeg，请确认运行环境已安装 FFmpeg")
	}
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	maxFrames := req.MaxFrames
	if maxFrames <= 0 {
		maxFrames = 30
	}
	if maxFrames > 200 {
		maxFrames = 200
	}

	outputDir := fs.GetTraceOutputDir()
	tempDir := filepath.Join(outputDir, "frames_tmp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	var outputPaths []string
	var infos []string
	for _, input := range inputFiles {
		if input == "" {
			continue
		}
		base := mediaSafeBase(filepath.Base(input), "video")
		var paths []string
		var err error
		if req.Mode == "指定时间点" {
			paths, err = extractFramesAtTimestamps(ctx, input, outputDir, base, parseFrameTimestamps(req.Timestamps), maxFrames, req.OutputImageWidth)
		} else {
			interval := req.IntervalSeconds
			if interval <= 0 {
				interval = 10
			}
			paths, err = extractFramesByInterval(ctx, input, tempDir, outputDir, base, interval, maxFrames, req.OutputImageWidth)
		}
		if err != nil {
			infos = append(infos, fmt.Sprintf("失败 %s: %v", filepath.Base(input), err))
			continue
		}
		outputPaths = append(outputPaths, paths...)
		infos = append(infos, fmt.Sprintf("成功 %s -> %d 张图片", filepath.Base(input), len(paths)))
	}
	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功抽取的视频帧\n%s", strings.Join(infos, "\n"))
	}
	outputFiles := fs.ResponseFiles(outputPaths)
	return &ExtractFramesResp{
		OutputFiles: outputFiles,
		ExtractInfo: fmt.Sprintf("视频抽帧完成\n模式: %s\n输出图片数: %d\n\n详情:\n%s", req.Mode, len(outputPaths), strings.Join(infos, "\n")),
	}, nil
}

func extractFramesByInterval(ctx *app.Context, input, tempDir, outputDir, base string, interval, maxFrames, width int) ([]string, error) {
	pattern := filepath.Join(tempDir, base+"_%03d.jpg")
	filter := fmt.Sprintf("fps=1/%d", interval)
	if width > 0 {
		filter += fmt.Sprintf(",scale=%d:-1", width)
	}
	args := []string{"-hide_banner", "-loglevel", "error", "-i", input, "-vf", filter, "-frames:v", strconv.Itoa(maxFrames), "-q:v", "2", pattern}
	out, err := exec.Command("ffmpeg", args...).CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[ExtractFrames] ffmpeg 抽帧失败 %s: %v, output: %s", filepath.Base(input), err, string(out))
		return nil, fmt.Errorf("ffmpeg 抽帧失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	tempPaths, err := filepath.Glob(filepath.Join(tempDir, base+"_*.jpg"))
	if err != nil {
		return nil, err
	}
	var outputPaths []string
	for i, tempPath := range tempPaths {
		outputPath := filepath.Join(outputDir, fmt.Sprintf("%s_frame_%03d.jpg", base, i+1))
		if err := os.Rename(tempPath, outputPath); err != nil {
			return nil, err
		}
		outputPaths = append(outputPaths, outputPath)
	}
	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("未生成任何帧图片")
	}
	return outputPaths, nil
}

func extractFramesAtTimestamps(ctx *app.Context, input, outputDir, base string, timestamps []string, maxFrames, width int) ([]string, error) {
	if len(timestamps) == 0 {
		return nil, fmt.Errorf("指定时间点不能为空")
	}
	if len(timestamps) > maxFrames {
		timestamps = timestamps[:maxFrames]
	}
	var outputPaths []string
	for i, ts := range timestamps {
		outputPath := filepath.Join(outputDir, fmt.Sprintf("%s_at_%03d.jpg", base, i+1))
		args := []string{"-hide_banner", "-loglevel", "error", "-ss", ts, "-i", input, "-frames:v", "1"}
		if width > 0 {
			args = append(args, "-vf", fmt.Sprintf("scale=%d:-1", width))
		}
		args = append(args, "-q:v", "2", "-y", outputPath)
		out, err := exec.Command("ffmpeg", args...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[ExtractFrames] 指定时间点抽帧失败 %s @ %s: %v, output: %s", filepath.Base(input), ts, err, string(out))
			return nil, fmt.Errorf("时间点 %s 抽帧失败: %v\n%s", ts, err, strings.TrimSpace(string(out)))
		}
		outputPaths = append(outputPaths, outputPath)
	}
	return outputPaths, nil
}

func parseFrameTimestamps(input string) []string {
	input = strings.NewReplacer(",", "\n", "，", "\n", ";", "\n", "；", "\n").Replace(input)
	var result []string
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func mediaSafeBase(name, fallback string) string {
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

var ExtractFramesTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "视频抽帧",
		Desc:     `使用 FFmpeg 从视频中按固定间隔或指定时间点抽取 JPG 图片帧。适合视频预览、关键画面提取、后续 OCR/图片处理或报告配图。`,
		Tags:     []string{"视频", "抽帧", "截图", "FFmpeg", "图片", "预览"},
		Request:  &ExtractFramesReq{},
		Response: &ExtractFramesResp{},
	},
}

func init() {
	packageContext.POST("extract_frames.form", ExtractFrames, ExtractFramesTemplate)
}
