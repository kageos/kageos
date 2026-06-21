package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type ConcatVideoReq struct {
	InputFiles     string `json:"input_files" widget:"name:上传视频文件;type:files;accept:video/*;max_size:500MB;max_count:20" validate:"required"`
	OutputFileName string `json:"output_file_name" widget:"name:输出文件名;type:input;placeholder:可选，例如 merged_video.mp4"`
	Resolution     string `json:"resolution" widget:"name:输出分辨率;type:select;options:自动跟第一个视频一致,1080p(1920x1080),720p(1280x720),480p(854x480);options_colors:409EFF,67C23A,909399,E6A23C;render_default:自动跟第一个视频一致"`
}

type ConcatVideoResp struct {
	OutputFiles string `json:"output_files" widget:"name:拼接后的视频;type:files"`
	ConcatInfo  string `json:"concat_info" widget:"name:拼接信息;type:text_area"`
}

func ConcatVideo(ctx *app.Context, resp response.Response) error {
	var req ConcatVideoReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoConcatVideo(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoConcatVideo(ctx *app.Context, req *ConcatVideoReq) (*ConcatVideoResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) < 2 {
		return nil, fmt.Errorf("至少需要 2 个视频文件")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("未找到 ffmpeg，请确认运行环境已安装 FFmpeg")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return nil, fmt.Errorf("未找到 ffprobe，请确认运行环境已安装 FFmpeg")
	}

	targetWidth, targetHeight, err := resolveConcatResolution(req.Resolution, files)
	if err != nil {
		return nil, err
	}

	outputDir := fs.GetTraceOutputDir()
	tempDir := filepath.Join(outputDir, "concat_segments")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	var normalizedPaths []string
	var sourceNames []string
	// concat demuxer 只适合同编码参数的片段，先把每段统一到相同视频和音频规格。
	for index, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[ConcatVideo] 文件 %s 无本地路径，跳过", filepath.Base(file))
			continue
		}
		sourceNames = append(sourceNames, filepath.Base(file))
		tempPath := filepath.Join(tempDir, fmt.Sprintf("segment_%03d.mp4", index+1))
		hasAudio, probeErr := probeHasAudio(file)
		if probeErr != nil {
			return nil, fmt.Errorf("探测视频音轨失败(%s): %w", filepath.Base(file), probeErr)
		}
		if err := normalizeVideoForConcat(ctx, file, tempPath, targetWidth, targetHeight, hasAudio); err != nil {
			return nil, fmt.Errorf("预处理视频失败(%s): %w", filepath.Base(file), err)
		}
		normalizedPaths = append(normalizedPaths, tempPath)
	}
	if len(normalizedPaths) < 2 {
		return nil, fmt.Errorf("可拼接的视频文件不足 2 个")
	}

	listPath := filepath.Join(tempDir, "concat_list.txt")
	if err := writeConcatListFile(listPath, normalizedPaths); err != nil {
		return nil, fmt.Errorf("写入拼接列表失败: %w", err)
	}

	outputName := strings.TrimSpace(req.OutputFileName)
	if outputName == "" {
		outputName = "merged_video.mp4"
	}
	outputName = sanitizeVideoFileName(outputName)
	outputPath := filepath.Join(outputDir, outputName)

	args := []string{
		"-f", "concat",
		"-safe", "0",
		"-i", listPath,
		"-c", "copy",
		"-movflags", "+faststart",
		"-y", outputPath,
	}
	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[ConcatVideo] ffmpeg 拼接失败: %v, output: %s", err, string(out))
		return nil, fmt.Errorf("视频拼接失败: %v", err)
	}

	outputFiles := fs.ResponseFiles([]string{outputPath})

	info := fmt.Sprintf(
		"视频拼接完成\n输出文件: %s\n输入文件数: %d\n统一分辨率: %dx%d\n统一帧率: 30fps\n输入顺序: %s",
		outputName,
		len(normalizedPaths),
		targetWidth,
		targetHeight,
		strings.Join(sourceNames, " -> "),
	)
	if strings.TrimSpace(string(out)) != "" {
		info += "\n命令输出:\n" + strings.TrimSpace(string(out))
	}

	return &ConcatVideoResp{
		OutputFiles: outputFiles,
		ConcatInfo:  info,
	}, nil
}

func resolveConcatResolution(option string, files []string) (int, int, error) {
	switch strings.TrimSpace(option) {
	case "", "自动跟第一个视频一致":
		for _, file := range files {
			if file != "" {
				return probeVideoSize(file)
			}
		}
		return 0, 0, fmt.Errorf("无法探测第一个视频分辨率")
	case "1080p(1920x1080)":
		return 1920, 1080, nil
	case "720p(1280x720)":
		return 1280, 720, nil
	case "480p(854x480)":
		return 854, 480, nil
	default:
		return 0, 0, fmt.Errorf("不支持的分辨率选项: %s", option)
	}
}

func probeVideoSize(filePath string) (int, int, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=s=x:p=0",
		filePath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe 执行失败: %v", err)
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("无法解析视频尺寸: %s", strings.TrimSpace(string(out)))
	}
	width, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}
	height, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}
	if width <= 0 || height <= 0 {
		return 0, 0, fmt.Errorf("视频尺寸无效: %dx%d", width, height)
	}
	return width, height, nil
}

func probeHasAudio(filePath string) (bool, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "a",
		"-show_entries", "stream=index",
		"-of", "csv=p=0",
		filePath,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func normalizeVideoForConcat(ctx *app.Context, inputPath, outputPath string, width, height int, hasAudio bool) error {
	scaleFilter := fmt.Sprintf(
		"scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black,setsar=1,fps=30",
		width, height, width, height,
	)

	args := []string{"-i", inputPath}
	if !hasAudio {
		// 没有音轨的片段补一条静音音轨，避免 concat 后容器结构不一致。
		args = append(args, "-f", "lavfi", "-i", "anullsrc=channel_layout=stereo:sample_rate=48000")
	}
	args = append(args,
		"-vf", scaleFilter,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "23",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-b:a", "192k",
		"-ar", "48000",
		"-ac", "2",
	)
	if !hasAudio {
		args = append(args, "-shortest")
	}
	args = append(args, "-movflags", "+faststart", "-y", outputPath)

	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[ConcatVideo] 预处理失败 input=%s output=%s err=%v output=%s", inputPath, outputPath, err, string(out))
		return err
	}
	return nil
}

func writeConcatListFile(listPath string, filePaths []string) error {
	var lines []string
	for _, filePath := range filePaths {
		escaped := strings.ReplaceAll(filePath, "'", "'\\''")
		lines = append(lines, "file '"+escaped+"'")
	}
	return os.WriteFile(listPath, []byte(strings.Join(lines, "\n")), 0644)
}

func sanitizeVideoFileName(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	name = replacer.Replace(strings.TrimSpace(name))
	if name == "" {
		return "merged_video.mp4"
	}
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return name + ".mp4"
}

var ConcatVideoTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "拼接多个视频",
		Desc:     `把多个视频按上传顺序拼接成一个 MP4 文件。系统会先统一分辨率、帧率和音频格式，再进行无缝拼接，适合课程分段、素材合并或监控片段汇总。`,
		Tags:     []string{"视频", "拼接", "FFmpeg", "合并视频"},
		Request:  &ConcatVideoReq{},
		Response: &ConcatVideoResp{},
	},
}

func init() {
	packageContext.POST("concat.form", ConcatVideo, ConcatVideoTemplate)
}
