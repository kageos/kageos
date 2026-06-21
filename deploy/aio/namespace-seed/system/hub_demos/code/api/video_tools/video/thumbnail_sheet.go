package video

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type ThumbnailSheetReq struct {
	InputFiles      string `json:"input_files" widget:"name:上传视频文件;type:files;accept:video/*,*/*;max_size:2000MB;max_count:20" validate:"required"`
	AutoLayout      bool   `json:"auto_layout" widget:"name:自动识别布局;type:switch;render_default:true"`
	OutputFileName  string `json:"output_file_name" widget:"name:输出文件名;type:input;placeholder:可选，仅单文件时生效，例如 preview.jpg"`
	IntervalSeconds int    `json:"interval_seconds" widget:"name:手动：抽帧间隔秒数;type:integer;min:1;max:3600;render_default:10;desc:关闭自动识别布局后显示并生效" validate:"required_if=AutoLayout false,min=0,max=3600"`
	MaxFrames       int    `json:"max_frames" widget:"name:手动：最多帧数;type:integer;min:1;max:100;render_default:12;desc:关闭自动识别布局后显示并生效" validate:"required_if=AutoLayout false,min=0,max=100"`
	Columns         int    `json:"columns" widget:"name:手动：每行列数;type:integer;min:1;max:10;render_default:4;desc:关闭自动识别布局后显示并生效" validate:"required_if=AutoLayout false,min=0,max=10"`
	ThumbnailWidth  int    `json:"thumbnail_width" widget:"name:手动：单张缩略图宽度;type:integer;min:80;max:1280;render_default:320;desc:关闭自动识别布局后显示并生效" validate:"required_if=AutoLayout false,min=0,max=1280"`
}

type ThumbnailSheetResp struct {
	OutputFiles string `json:"output_files" widget:"name:缩略图总览;type:files"`
	SheetInfo   string `json:"sheet_info" widget:"name:生成信息;type:text_area"`
}

func ThumbnailSheet(ctx *app.Context, resp response.Response) error {
	var req ThumbnailSheetReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoThumbnailSheet(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoThumbnailSheet(ctx *app.Context, req *ThumbnailSheetReq) (*ThumbnailSheetResp, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("未找到 ffmpeg，请确认运行环境已安装 FFmpeg")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return nil, fmt.Errorf("未找到 ffprobe，请确认运行环境已安装 FFmpeg")
	}
	if _, err := exec.LookPath("montage"); err != nil {
		return nil, fmt.Errorf("未找到 montage，请确认运行环境已安装 ImageMagick")
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	autoLayout := req.AutoLayout || (req.IntervalSeconds <= 0 && req.MaxFrames <= 0 && req.Columns <= 0 && req.ThumbnailWidth <= 0)

	outputDir := fs.GetTraceOutputDir()
	tempDir := filepath.Join(outputDir, "thumbnail_sheet_tmp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	var outputPaths []string
	var infos []string
	for i, input := range inputFiles {
		if input == "" {
			continue
		}
		base := mediaSafeBase(filepath.Base(input), "video")
		frameBase := fmt.Sprintf("%s_%03d", base, i+1)
		outputName := base + "_sheet.jpg"
		if len(inputFiles) == 1 && strings.TrimSpace(req.OutputFileName) != "" {
			outputName = mediaSafeBase(req.OutputFileName, "preview") + ".jpg"
		}
		outputPath := filepath.Join(outputDir, outputName)
		plan, err := buildThumbnailSheetPlan(input, req, autoLayout)
		if err != nil {
			infos = append(infos, fmt.Sprintf("失败 %s: %v", filepath.Base(input), err))
			continue
		}
		framePaths, err := extractSheetFrames(ctx, input, tempDir, frameBase, plan)
		if err != nil {
			infos = append(infos, fmt.Sprintf("失败 %s: %v", filepath.Base(input), err))
			continue
		}
		if plan.Auto {
			plan.Columns, plan.Rows = bestSheetGrid(len(framePaths))
		} else if plan.Rows <= 0 {
			plan.Rows = int(math.Ceil(float64(len(framePaths)) / float64(plan.Columns)))
		}
		args := append([]string{}, framePaths...)
		args = append(args, "-tile", fmt.Sprintf("%dx%d", plan.Columns, plan.Rows), "-geometry", "+6+6", "-background", "#111827", "-bordercolor", "#111827", outputPath)
		out, err := exec.Command("montage", args...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[ThumbnailSheet] montage 失败 %s: %v, output: %s", filepath.Base(input), err, string(out))
			infos = append(infos, fmt.Sprintf("失败 %s: montage 失败: %v\n%s", filepath.Base(input), err, strings.TrimSpace(string(out))))
			continue
		}
		outputPaths = append(outputPaths, outputPath)
		infos = append(infos, fmt.Sprintf("成功 %s -> %s（%s，%d 帧，布局 %dx%d，单图宽度 %d）", filepath.Base(input), outputName, plan.ModeLabel(), len(framePaths), plan.Columns, plan.Rows, plan.Width))
	}
	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功生成的视频缩略图总览\n%s", strings.Join(infos, "\n"))
	}
	outputFiles := fs.ResponseFiles(outputPaths)
	return &ThumbnailSheetResp{
		OutputFiles: outputFiles,
		SheetInfo:   fmt.Sprintf("视频缩略图总览生成完成\n模式: %s\n输出文件数: %d\n\n详情:\n%s", layoutModeLabel(autoLayout), len(outputPaths), strings.Join(infos, "\n")),
	}, nil
}

type thumbnailSheetPlan struct {
	Auto       bool
	Duration   float64
	FrameCount int
	Interval   int
	Columns    int
	Rows       int
	Width      int
}

func (p thumbnailSheetPlan) ModeLabel() string {
	if p.Auto {
		if p.Duration > 0 {
			return fmt.Sprintf("自动均匀抽帧，时长 %.1f 秒", p.Duration)
		}
		return "自动布局"
	}
	return fmt.Sprintf("手动间隔 %d 秒", p.Interval)
}

func buildThumbnailSheetPlan(input string, req *ThumbnailSheetReq, autoLayout bool) (thumbnailSheetPlan, error) {
	if autoLayout {
		info, err := probeThumbnailVideo(input)
		if err != nil {
			return thumbnailSheetPlan{}, err
		}
		frameCount := autoSheetFrameCount(info.Duration)
		columns, rows := sheetGridForPlannedCount(frameCount)
		return thumbnailSheetPlan{
			Auto:       true,
			Duration:   info.Duration,
			FrameCount: frameCount,
			Interval:   fallbackInterval(info.Duration, frameCount),
			Columns:    columns,
			Rows:       rows,
			Width:      autoThumbnailWidth(info.Width, info.Height, columns),
		}, nil
	}

	interval := req.IntervalSeconds
	if interval <= 0 {
		interval = 10
	}
	maxFrames := req.MaxFrames
	if maxFrames <= 0 {
		maxFrames = 12
	}
	if maxFrames > 100 {
		maxFrames = 100
	}
	columns := req.Columns
	if columns <= 0 {
		columns = 4
	}
	width := req.ThumbnailWidth
	if width <= 0 {
		width = 320
	}
	return thumbnailSheetPlan{
		Auto:       false,
		FrameCount: maxFrames,
		Interval:   interval,
		Columns:    columns,
		Rows:       int(math.Ceil(float64(maxFrames) / float64(columns))),
		Width:      width,
	}, nil
}

func extractSheetFrames(ctx *app.Context, input, tempDir, base string, plan thumbnailSheetPlan) ([]string, error) {
	if plan.Auto && plan.Duration > 0 {
		return extractSheetFramesAtTimes(ctx, input, tempDir, base, thumbnailTimestamps(plan.Duration, plan.FrameCount), plan.Width)
	}
	pattern := filepath.Join(tempDir, base+"_sheet_%03d.jpg")
	filter := fmt.Sprintf("fps=1/%d,scale=%d:-1", plan.Interval, plan.Width)
	args := []string{"-hide_banner", "-loglevel", "error", "-i", input, "-vf", filter, "-frames:v", strconv.Itoa(plan.FrameCount), "-q:v", "3", pattern}
	out, err := exec.Command("ffmpeg", args...).CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[ThumbnailSheet] ffmpeg 抽帧失败 %s: %v, output: %s", filepath.Base(input), err, string(out))
		return nil, fmt.Errorf("ffmpeg 抽帧失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	paths, err := filepath.Glob(filepath.Join(tempDir, base+"_sheet_*.jpg"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("未生成任何缩略图帧")
	}
	return paths, nil
}

func extractSheetFramesAtTimes(ctx *app.Context, input, tempDir, base string, timestamps []float64, width int) ([]string, error) {
	var paths []string
	for i, timestamp := range timestamps {
		outputPath := filepath.Join(tempDir, fmt.Sprintf("%s_sheet_%03d.jpg", base, i+1))
		args := []string{
			"-hide_banner", "-loglevel", "error",
			"-ss", fmt.Sprintf("%.3f", timestamp),
			"-i", input,
			"-frames:v", "1",
			"-vf", fmt.Sprintf("scale=%d:-1", width),
			"-q:v", "3",
			"-y", outputPath,
		}
		out, err := exec.Command("ffmpeg", args...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[ThumbnailSheet] ffmpeg 指定时间抽帧失败 %s @ %.3f: %v, output: %s", filepath.Base(input), timestamp, err, string(out))
			return nil, fmt.Errorf("时间点 %.3f 抽帧失败: %v\n%s", timestamp, err, strings.TrimSpace(string(out)))
		}
		paths = append(paths, outputPath)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("未生成任何缩略图帧")
	}
	return paths, nil
}

type thumbnailVideoInfo struct {
	Duration float64
	Width    int
	Height   int
}

func probeThumbnailVideo(input string) (thumbnailVideoInfo, error) {
	out, err := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,duration:format=duration",
		"-of", "json",
		input,
	).CombinedOutput()
	if err != nil {
		return thumbnailVideoInfo{}, fmt.Errorf("ffprobe 读取视频信息失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	var payload struct {
		Streams []struct {
			Width    int    `json:"width"`
			Height   int    `json:"height"`
			Duration string `json:"duration"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return thumbnailVideoInfo{}, fmt.Errorf("解析 ffprobe 输出失败: %w", err)
	}
	info := thumbnailVideoInfo{}
	if len(payload.Streams) > 0 {
		info.Width = payload.Streams[0].Width
		info.Height = payload.Streams[0].Height
		info.Duration, _ = strconv.ParseFloat(payload.Streams[0].Duration, 64)
	}
	if info.Duration <= 0 {
		info.Duration, _ = strconv.ParseFloat(payload.Format.Duration, 64)
	}
	return info, nil
}

func autoSheetFrameCount(duration float64) int {
	switch {
	case duration <= 0:
		return 12
	case duration <= 20:
		return 6
	case duration <= 90:
		return 9
	case duration <= 300:
		return 12
	case duration <= 900:
		return 16
	case duration <= 2400:
		return 20
	default:
		return 24
	}
}

func sheetGridForPlannedCount(frameCount int) (int, int) {
	switch frameCount {
	case 6:
		return 3, 2
	case 8:
		return 4, 2
	case 9:
		return 3, 3
	case 12:
		return 4, 3
	case 16:
		return 4, 4
	case 20:
		return 5, 4
	case 24:
		return 6, 4
	default:
		return bestSheetGrid(frameCount)
	}
}

func bestSheetGrid(frameCount int) (int, int) {
	if frameCount <= 1 {
		return 1, 1
	}
	bestColumns := frameCount
	bestRows := 1
	bestScore := math.MaxFloat64
	for rows := 1; rows <= frameCount; rows++ {
		if frameCount%rows != 0 {
			continue
		}
		columns := frameCount / rows
		if columns < rows {
			continue
		}
		score := math.Abs(float64(columns)/float64(rows) - 1.45)
		if score < bestScore {
			bestScore = score
			bestColumns = columns
			bestRows = rows
		}
	}
	return bestColumns, bestRows
}

func autoThumbnailWidth(sourceWidth, sourceHeight, columns int) int {
	targetSheetWidth := 1440
	if sourceHeight > sourceWidth && sourceWidth > 0 {
		targetSheetWidth = 1180
	}
	width := (targetSheetWidth - (columns+1)*12) / columns
	if width < 180 {
		width = 180
	}
	if width > 420 {
		width = 420
	}
	if sourceWidth > 0 && width > sourceWidth {
		width = sourceWidth
	}
	if width < 120 {
		width = 120
	}
	return width
}

func thumbnailTimestamps(duration float64, frameCount int) []float64 {
	if frameCount <= 0 {
		frameCount = 12
	}
	if duration <= 0 {
		return nil
	}
	timestamps := make([]float64, 0, frameCount)
	step := duration / float64(frameCount+1)
	for i := 1; i <= frameCount; i++ {
		t := step * float64(i)
		if t < 0.05 {
			t = 0.05
		}
		if t > duration-0.05 {
			t = duration - 0.05
		}
		if t < 0 {
			t = 0
		}
		timestamps = append(timestamps, t)
	}
	return timestamps
}

func fallbackInterval(duration float64, frameCount int) int {
	if duration <= 0 || frameCount <= 0 {
		return 10
	}
	interval := int(math.Round(duration / float64(frameCount)))
	if interval < 1 {
		interval = 1
	}
	if interval > 3600 {
		interval = 3600
	}
	return interval
}

func layoutModeLabel(autoLayout bool) string {
	if autoLayout {
		return "自动识别布局"
	}
	return "手动参数"
}

var ThumbnailSheetTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "视频缩略图总览",
		Desc:     `默认自动读取视频时长和分辨率，均匀抽取合适数量的画面，并选择紧凑矩阵布局生成缩略图总览，避免最后一行大面积空白。关闭自动识别后可手动设置抽帧间隔、帧数、列数和缩略图宽度。`,
		Tags:     []string{"视频", "缩略图", "预览", "自动布局", "抽帧", "FFmpeg"},
		Request:  &ThumbnailSheetReq{},
		Response: &ThumbnailSheetResp{},
	},
}

func init() {
	packageContext.POST("thumbnail_sheet.form", ThumbnailSheet, ThumbnailSheetTemplate)
}
