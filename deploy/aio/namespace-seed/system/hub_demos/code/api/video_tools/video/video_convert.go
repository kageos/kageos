package video

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// VideoConvertReq 视频格式转换请求结构体
type VideoConvertReq struct {
	// 上传视频文件
	InputFiles string `json:"input_files" widget:"name:上传视频文件;type:files;accept:video/*;max_size:500MB;max_count:10" validate:"required"`

	// 目标格式
	OutputFormat string `json:"output_format" widget:"name:目标格式;type:select;options:mp4,webm,avi,mkv,flv,mov,wmv;options_colors:409EFF,67C23A,909399,E6A23C,F56C6C,909399,909399;render_default:mp4" validate:"required,oneof=mp4 webm avi mkv flv mov wmv"`

	// 视频质量（可选）
	Quality string `json:"quality" widget:"name:视频质量;type:select;options:原画质,高质量,中等质量,低质量;options_colors:67C23A,409EFF,E6A23C,909399;render_default:原画质"`

	// 分辨率（可选）
	Resolution string `json:"resolution" widget:"name:分辨率;type:select;options:保持原分辨率,1080p(1920x1080),720p(1280x720),480p(854x480),360p(640x360);options_colors:909399,67C23A,409EFF,E6A23C,909399;render_default:保持原分辨率"`

	// 帧率（可选）
	FrameRate string `json:"frame_rate" widget:"name:帧率;type:select;options:保持原帧率,60fps,30fps,25fps,24fps;options_colors:909399,67C23A,409EFF,E6A23C,909399;render_default:保持原帧率"`

	// 音频质量（可选）
	AudioQuality string `json:"audio_quality" widget:"name:音频质量;type:select;options:保持原音质,高质量,中等质量,低质量;options_colors:909399,67C23A,409EFF,E6A23C;render_default:保持原音质"`
}

// VideoConvertResp 视频格式转换响应结构体
type VideoConvertResp struct {
	// 转换后的视频文件
	OutputFile string `json:"output_file" widget:"name:转换后的视频;type:files"`

	// 转换信息
	ConvertInfo string `json:"convert_info" widget:"name:转换信息;type:text_area"`

	// 原始文件信息
	OriginalInfo string `json:"original_info" widget:"name:原始文件信息;type:text_area"`

	// 转换统计
	Statistics string `json:"statistics" widget:"name:转换统计;type:text_area"`
}

// VideoConvert 视频格式转换函数
func VideoConvert(ctx *app.Context, resp response.Response) error {
	var req VideoConvertReq
	err := ctx.ShouldBindValidate(&req)
	if err != nil {
		return err
	}

	fs := ctx.GetFS()

	// 1. 下载文件到本地
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	if len(inputFiles) == 0 {
		return fmt.Errorf("没有找到输入文件")
	}

	// 2. 使用 FFmpeg 转换视频格式（依赖 PATH，镜像内已安装）
	const ffmpegCmd = "ffmpeg"
	if _, err := exec.LookPath(ffmpegCmd); err != nil {
		logger.Errorf(ctx, "[系统错误]-[VideoConvert] FFmpeg 未在 PATH 中, req: %+v, err: %v", req, err)
		return fmt.Errorf("[系统错误]-[VideoConvert]： FFmpeg未安装或不在 PATH 中，请确保运行环境已安装 FFmpeg")
	}

	// 使用 GetTraceOutputDir 生成唯一的输出目录
	outputDir := fs.GetTraceOutputDir()

	// 3. 批量处理所有视频文件
	outputFilePaths := make([]string, 0)
	successCount := 0
	failCount := 0
	var errors []string
	var originalInfos []string
	var statistics []string

	for _, file := range inputFiles {
		if file == "" {
			logger.Warnf(ctx, "[VideoConvert] 文件 %s 没有本地路径，跳过", filepath.Base(file))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 本地路径为空", filepath.Base(file)))
			continue
		}

		// 获取文件信息
		inputExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(file), "."))
		baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		outputPath := filepath.Join(outputDir, baseName+"_converted."+req.OutputFormat)

		// 获取原始视频信息
		originalInfo := getVideoInfo(ctx, ffmpegCmd, file, filepath.Base(file))
		originalInfos = append(originalInfos, originalInfo)

		if inputExt == req.OutputFormat {
			logger.Infof(ctx, "[VideoConvert] 文件 %s 格式相同，跳过转换: %s", filepath.Base(file), inputExt)
			outputPath = file
		} else {
			// 构建FFmpeg命令参数
			args := []string{"-i", file}

			// 添加视频质量参数
			args = append(args, getQualityArgs(req.Quality)...)

			// 添加分辨率参数
			args = append(args, getResolutionArgs(req.Resolution)...)

			// 添加帧率参数
			args = append(args, getFrameRateArgs(req.FrameRate)...)

			// 添加音频质量参数
			args = append(args, getAudioQualityArgs(req.AudioQuality)...)

			// 根据输出格式选择编码器
			switch req.OutputFormat {
			case "webm":
				args = append(args, "-c:v", "libvpx", "-c:a", "libopus")
			case "mp4":
				args = append(args, "-c:v", "libx264", "-c:a", "aac")
			case "avi":
				args = append(args, "-c:v", "mpeg4", "-c:a", "mp3")
			case "mkv":
				args = append(args, "-c:v", "libx264", "-c:a", "aac")
			case "flv":
				args = append(args, "-c:v", "flv", "-c:a", "mp3")
			case "mov":
				args = append(args, "-c:v", "libx264", "-c:a", "aac")
			case "wmv":
				args = append(args, "-c:v", "wmv2", "-c:a", "wmav2")
			default:
				args = append(args, "-c", "copy")
			}

			args = append(args, "-y", outputPath) // -y 覆盖输出文件

			// 执行FFmpeg命令
			cmd := exec.Command(ffmpegCmd, args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				logger.Errorf(ctx, "[VideoConvert] 转换失败 %s: %v, output: %s", filepath.Base(file), err, string(output))
				failCount++
				errors = append(errors, fmt.Sprintf("文件 %s: %v", filepath.Base(file), err))
				continue
			}

			logger.Infof(ctx, "[VideoConvert] 转换成功: %s -> %s", file, outputPath)

			// 获取转换后文件信息
			convertedInfo := getVideoInfo(ctx, ffmpegCmd, outputPath, baseName+"_converted."+req.OutputFormat)
			statistics = append(statistics, fmt.Sprintf("文件: %s\n%s", filepath.Base(file), convertedInfo))
		}

		outputFilePaths = append(outputFilePaths, outputPath)
		successCount++
	}

	// 4. 上传转换后的文件
	var outputFiles string
	if len(outputFilePaths) > 0 {
		outputFiles = fs.ResponseFiles(outputFilePaths)
	}

	// 5. 构建响应信息
	convertInfo := fmt.Sprintf("视频转换完成！\n\n成功转换: %d 个文件\n失败: %d 个文件\n目标格式: %s", successCount, failCount, req.OutputFormat)

	if len(errors) > 0 {
		convertInfo += "\n\n失败详情:\n" + strings.Join(errors, "\n")
	}

	originalInfoStr := strings.Join(originalInfos, "\n\n")
	statisticsStr := strings.Join(statistics, "\n\n")

	// 6. 构建响应
	return resp.Form(&VideoConvertResp{
		OutputFile:   outputFiles,
		ConvertInfo:  convertInfo,
		OriginalInfo: originalInfoStr,
		Statistics:   statisticsStr,
	}).Build()
}

// getVideoInfo 获取视频文件信息
func getVideoInfo(ctx *app.Context, ffmpegPath, filePath, fileName string) string {
	cmd := exec.Command(ffmpegPath, "-i", filePath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// FFmpeg在获取信息时通常会返回错误，但输出中包含信息
		outputStr := string(output)
		if strings.Contains(outputStr, "Duration:") {
			// 提取关键信息
			lines := strings.Split(outputStr, "\n")
			var infoLines []string
			for _, line := range lines {
				if strings.Contains(line, "Duration:") || strings.Contains(line, "Stream") || strings.Contains(line, "Video:") || strings.Contains(line, "Audio:") {
					infoLines = append(infoLines, strings.TrimSpace(line))
				}
			}
			return fmt.Sprintf("文件: %s\n%s", fileName, strings.Join(infoLines, "\n"))
		}
		return fmt.Sprintf("文件: %s\n无法获取详细信息", fileName)
	}

	return fmt.Sprintf("文件: %s\n基本信息已获取", fileName)
}

// getQualityArgs 获取视频质量参数
func getQualityArgs(quality string) []string {
	switch quality {
	case "高质量":
		return []string{"-crf", "18"}
	case "中等质量":
		return []string{"-crf", "23"}
	case "低质量":
		return []string{"-crf", "28"}
	default:
		return []string{} // 原画质
	}
}

// getResolutionArgs 获取分辨率参数
func getResolutionArgs(resolution string) []string {
	switch resolution {
	case "1080p(1920x1080)":
		return []string{"-vf", "scale=1920:1080"}
	case "720p(1280x720)":
		return []string{"-vf", "scale=1280:720"}
	case "480p(854x480)":
		return []string{"-vf", "scale=854:480"}
	case "360p(640x360)":
		return []string{"-vf", "scale=640:360"}
	default:
		return []string{} // 保持原分辨率
	}
}

// getFrameRateArgs 获取帧率参数
func getFrameRateArgs(frameRate string) []string {
	switch frameRate {
	case "60fps":
		return []string{"-r", "60"}
	case "30fps":
		return []string{"-r", "30"}
	case "25fps":
		return []string{"-r", "25"}
	case "24fps":
		return []string{"-r", "24"}
	default:
		return []string{} // 保持原帧率
	}
}

// getAudioQualityArgs 获取音频质量参数
func getAudioQualityArgs(audioQuality string) []string {
	switch audioQuality {
	case "高质量":
		return []string{"-b:a", "192k"}
	case "中等质量":
		return []string{"-b:a", "128k"}
	case "低质量":
		return []string{"-b:a", "64k"}
	default:
		return []string{} // 保持原音质
	}
}

// VideoConvertTemplate 视频格式转换配置
var VideoConvertTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "视频格式转换",
		Desc:     `使用FFmpeg进行视频格式转换，支持MP4、WebM、AVI、MKV、FLV、MOV、WMV等多种格式。可调整视频质量、分辨率、帧率和音频质量。支持批量处理多个视频文件。`,
		Tags:     []string{"视频处理", "格式转换", "FFmpeg", "工具"},
		Request:  &VideoConvertReq{},
		Response: &VideoConvertResp{},
	},
}

func init() {
	// 注册Form函数 - 视频格式转换
	packageContext.POST("convert.form", VideoConvert, VideoConvertTemplate)
}
