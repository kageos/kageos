package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// VideoInfoReq 视频信息提取请求结构体
type VideoInfoReq struct {
	// 上传视频文件
	InputFile string `json:"input_file" widget:"name:上传视频文件;type:files;accept:video/*;max_size:2000MB;max_count:1" validate:"required"`

	// 信息类型
	InfoType string `json:"info_type" widget:"name:信息类型;type:select;options:基本信息,详细技术信息,流信息,格式信息;options_colors:409EFF,67C23A,909399,E6A23C;render_default:基本信息" validate:"required,oneof=基本信息 详细技术信息 流信息 格式信息"`
}

// VideoInfoResp 视频信息提取响应结构体
type VideoInfoResp struct {
	// 视频信息
	VideoInfo string `json:"video_info" widget:"name:视频信息;type:text_area"`

	// 格式化信息
	FormattedInfo string `json:"formatted_info" widget:"name:格式化信息;type:text_area"`

	// 技术参数
	TechnicalParams string `json:"technical_params" widget:"name:技术参数;type:text_area"`

	// 文件信息
	FileInfo string `json:"file_info" widget:"name:文件信息;type:text_area"`
}

// VideoInfo 视频信息提取函数
func VideoInfo(ctx *app.Context, resp response.Response) error {
	var req VideoInfoReq
	err := ctx.ShouldBindValidate(&req)
	if err != nil {
		return err
	}

	fs := ctx.GetFS()

	// 1. 下载文件到本地
	inputFiles := fs.DownloadFiles(req.InputFile)
	defer fs.RemoveFiles(inputFiles)

	if len(inputFiles) == 0 {
		return fmt.Errorf("没有找到输入文件")
	}

	file := inputFiles[0]
	if file == "" {
		return fmt.Errorf("文件本地路径为空")
	}

	// 2. 使用 FFprobe 提取视频信息（依赖 PATH，镜像内已安装）
	const ffprobeCmd = "ffprobe"
	if _, err := exec.LookPath(ffprobeCmd); err != nil {
		logger.Errorf(ctx, "[系统错误]-[VideoInfo] FFprobe 未在 PATH 中, req: %+v, err: %v", req, err)
		return fmt.Errorf("[系统错误]-[VideoInfo]： FFprobe未安装或不在 PATH 中，请确保运行环境已安装 FFmpeg 套件")
	}

	// 构建命令参数
	var args []string

	switch req.InfoType {
	case "基本信息":
		args = []string{"-i", file}
	case "详细技术信息":
		args = []string{"-i", file, "-hide_banner"}
	case "流信息":
		args = []string{"-i", file, "-show_streams", "-print_format", "json"}
	case "格式信息":
		args = []string{"-i", file, "-show_format", "-print_format", "json"}
	default:
		args = []string{"-i", file}
	}

	// 执行命令
	logger.Infof(ctx, "[VideoInfo] 执行命令: %s %s", ffprobeCmd, strings.Join(args, " "))
	cmd := exec.Command(ffprobeCmd, args...)
	output, err := cmd.CombinedOutput()

	outputStr := string(output)
	var videoInfo, formattedInfo, technicalParams, fileInfo string

	if err != nil {
		// FFprobe/FFmpeg在获取信息时通常会返回错误，但输出中包含信息
		logger.Warnf(ctx, "[VideoInfo] 命令执行返回错误，但可能包含有效信息: %v", err)

		if strings.Contains(outputStr, "Duration:") || strings.Contains(outputStr, "Stream") {
			// 从输出中提取信息
			videoInfo = outputStr
			formattedInfo = formatVideoInfo(outputStr, req.InfoType)
			technicalParams = extractTechnicalParams(outputStr)
		} else {
			logger.Errorf(ctx, "[系统错误]-[VideoInfo] 无法获取视频信息, req: %+v, err: %v, output: %s", req, err, outputStr)
			return fmt.Errorf("[系统错误]-[VideoInfo]： 无法获取视频信息, err: %v", err)
		}
	} else {
		videoInfo = outputStr
		formattedInfo = formatVideoInfo(outputStr, req.InfoType)
		technicalParams = extractTechnicalParams(outputStr)
	}

	// 构建文件信息
	fileInfo = buildFileInfo(file, file)

	// 3. 构建响应
	return resp.Form(&VideoInfoResp{
		VideoInfo:       videoInfo,
		FormattedInfo:   formattedInfo,
		TechnicalParams: technicalParams,
		FileInfo:        fileInfo,
	}).Build()
}

// formatVideoInfo 格式化视频信息
func formatVideoInfo(rawInfo string, infoType string) string {
	lines := strings.Split(rawInfo, "\n")
	var formattedLines []string

	switch infoType {
	case "基本信息":
		formattedLines = append(formattedLines, "=== 视频基本信息 ===")
		for _, line := range lines {
			if strings.Contains(line, "Duration:") || strings.Contains(line, "bitrate:") {
				formattedLines = append(formattedLines, "• "+strings.TrimSpace(line))
			}
		}

	case "详细技术信息":
		formattedLines = append(formattedLines, "=== 详细技术信息 ===")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.Contains(trimmed, "ffmpeg version") &&
				!strings.Contains(trimmed, "built with") && !strings.Contains(trimmed, "configuration:") {
				formattedLines = append(formattedLines, "• "+trimmed)
			}
		}

	case "流信息":
		formattedLines = append(formattedLines, "=== 流信息 ===")
		// 尝试解析JSON格式的流信息
		if strings.Contains(rawInfo, "streams") {
			formattedLines = append(formattedLines, "• 流信息以JSON格式返回")
			formattedLines = append(formattedLines, "• 包含视频流、音频流等详细信息")
		} else {
			// 非JSON格式，直接显示
			for _, line := range lines {
				if strings.Contains(line, "Stream") {
					formattedLines = append(formattedLines, "• "+strings.TrimSpace(line))
				}
			}
		}

	case "格式信息":
		formattedLines = append(formattedLines, "=== 格式信息 ===")
		if strings.Contains(rawInfo, "format") {
			formattedLines = append(formattedLines, "• 格式信息以JSON格式返回")
			formattedLines = append(formattedLines, "• 包含容器格式、时长、大小等信息")
		} else {
			for _, line := range lines {
				if strings.Contains(line, "Input") || strings.Contains(line, "Duration:") {
					formattedLines = append(formattedLines, "• "+strings.TrimSpace(line))
				}
			}
		}
	}

	return strings.Join(formattedLines, "\n")
}

// extractTechnicalParams 提取技术参数
func extractTechnicalParams(rawInfo string) string {
	var params []string
	params = append(params, "=== 技术参数 ===")

	// 提取时长
	durationRe := regexp.MustCompile(`Duration: (\d{2}):(\d{2}):(\d{2}\.\d+)`)
	if matches := durationRe.FindStringSubmatch(rawInfo); len(matches) >= 4 {
		hours, _ := strconv.Atoi(matches[1])
		minutes, _ := strconv.Atoi(matches[2])
		seconds, _ := strconv.ParseFloat(matches[3], 64)
		totalSeconds := float64(hours*3600+minutes*60) + seconds
		params = append(params, fmt.Sprintf("• 总时长: %.2f 秒", totalSeconds))
		params = append(params, fmt.Sprintf("• 时长: %s:%s:%s", matches[1], matches[2], matches[3]))
	}

	// 提取码率
	bitrateRe := regexp.MustCompile(`bitrate: (\d+) kb/s`)
	if matches := bitrateRe.FindStringSubmatch(rawInfo); len(matches) >= 2 {
		params = append(params, fmt.Sprintf("• 码率: %s kb/s", matches[1]))
	}

	// 提取视频流信息
	videoStreamRe := regexp.MustCompile(`Video: ([^,]+), ([^,]+), (\d+)x(\d+)`)
	if matches := videoStreamRe.FindStringSubmatch(rawInfo); len(matches) >= 5 {
		params = append(params, fmt.Sprintf("• 视频编码: %s", matches[1]))
		params = append(params, fmt.Sprintf("• 视频格式: %s", matches[2]))
		params = append(params, fmt.Sprintf("• 分辨率: %sx%s", matches[3], matches[4]))
	}

	// 提取音频流信息
	audioStreamRe := regexp.MustCompile(`Audio: ([^,]+), (\d+) Hz`)
	if matches := audioStreamRe.FindStringSubmatch(rawInfo); len(matches) >= 3 {
		params = append(params, fmt.Sprintf("• 音频编码: %s", matches[1]))
		params = append(params, fmt.Sprintf("• 采样率: %s Hz", matches[2]))
	}

	// 提取帧率
	fpsRe := regexp.MustCompile(`(\d+(\.\d+)?) fps`)
	if matches := fpsRe.FindStringSubmatch(rawInfo); len(matches) >= 2 {
		params = append(params, fmt.Sprintf("• 帧率: %s fps", matches[1]))
	}

	if len(params) == 1 {
		params = append(params, "• 未检测到详细技术参数")
	}

	return strings.Join(params, "\n")
}

// buildFileInfo 构建文件信息
func buildFileInfo(file string, filePath string) string {
	var info []string
	info = append(info, "=== 文件信息 ===")

	// 文件名
	info = append(info, fmt.Sprintf("• 文件名: %s", filepath.Base(file)))

	// 文件大小
	var fileSize int64
	if stat, err := os.Stat(file); err == nil {
		fileSize = stat.Size()
	}
	sizeMB := float64(fileSize) / (1024 * 1024)
	info = append(info, fmt.Sprintf("• 文件大小: %.2f MB", sizeMB))

	// 文件扩展名
	ext := filepath.Ext(filepath.Base(file))
	if ext != "" {
		info = append(info, fmt.Sprintf("• 文件格式: %s", strings.TrimPrefix(ext, ".")))
	}

	// 文件路径
	info = append(info, fmt.Sprintf("• 本地路径: %s", filePath))

	return strings.Join(info, "\n")
}

// VideoInfoTemplate 视频信息提取配置
var VideoInfoTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "视频信息提取",
		Desc:     `使用FFprobe/FFmpeg提取视频文件的详细信息，包括基本信息、技术参数、流信息、格式信息等。支持多种信息类型选择，提供格式化的技术参数展示。`,
		Tags:     []string{"视频处理", "信息提取", "FFmpeg", "工具"},
		Request:  &VideoInfoReq{},
		Response: &VideoInfoResp{},
	},
}

func init() {
	// 注册Form函数 - 视频信息提取
	packageContext.POST("info.form", VideoInfo, VideoInfoTemplate)
}
