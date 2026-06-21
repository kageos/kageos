package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// VideoCompressReq 视频压缩请求结构体
type VideoCompressReq struct {
	// 上传视频文件
	InputFile string `json:"input_file" widget:"name:上传视频文件;type:files;accept:video/*;max_size:2000MB;max_count:1" validate:"required"`

	// 压缩模式
	CompressionMode string `json:"compression_mode" widget:"name:压缩模式;type:select;options:智能压缩,指定文件大小,指定视频码率,指定分辨率;options_colors:409EFF,67C23A,909399,E6A23C;render_default:智能压缩" validate:"required,oneof=智能压缩 指定文件大小 指定视频码率 指定分辨率"`

	// 目标文件大小（MB）
	TargetSize float64 `json:"target_size" widget:"name:目标文件大小;type:float;min:1;max:1000;step:1;unit:MB;render_default:10;desc:指定文件大小模式显示并生效" validate:"required_if=CompressionMode 指定文件大小,min=0,max=1000"`

	// 目标视频码率（kbps）
	TargetBitrate int `json:"target_bitrate" widget:"name:目标视频码率;type:integer;min:100;max:10000;step:100;unit:kbps;render_default:1000;desc:指定视频码率模式显示并生效" validate:"required_if=CompressionMode 指定视频码率,min=0,max=10000"`

	// 目标分辨率
	TargetResolution string `json:"target_resolution" widget:"name:目标分辨率;type:select;options:1080p(1920x1080),720p(1280x720),480p(854x480),360p(640x360),240p(426x240);options_colors:67C23A,409EFF,E6A23C,909399,F56C6C;render_default:720p(1280x720);desc:指定分辨率模式显示并生效" validate:"required_if=CompressionMode 指定分辨率,omitempty,oneof=1080p(1920x1080) 720p(1280x720) 480p(854x480) 360p(640x360) 240p(426x240)"`

	// 压缩质量
	CompressionQuality string `json:"compression_quality" widget:"name:压缩质量;type:select;options:高质量(文件较大),平衡质量,高压缩(文件较小);options_colors:67C23A,409EFF,E6A23C;render_default:平衡质量"`

	// 保持原始帧率
	KeepOriginalFramerate bool `json:"keep_original_framerate" widget:"name:保持原始帧率;type:switch;render_default:true"`

	// 目标帧率
	TargetFramerate int `json:"target_framerate" widget:"name:目标帧率;type:integer;min:1;max:120;step:1;unit:fps;render_default:30;desc:关闭保持原始帧率后显示并生效" validate:"required_if=KeepOriginalFramerate false,min=0,max=120"`

	// 保持原始音频质量
	KeepOriginalAudio bool `json:"keep_original_audio" widget:"name:保持原始音频质量;type:switch;render_default:true"`

	// 目标音频码率
	AudioBitrate int `json:"audio_bitrate" widget:"name:目标音频码率;type:integer;min:32;max:320;step:16;unit:kbps;render_default:128;desc:关闭保持原始音频质量后显示并生效" validate:"required_if=KeepOriginalAudio false,min=0,max=320"`

	// 输出格式
	OutputFormat string `json:"output_format" widget:"name:输出格式;type:select;options:mp4,webm;options_colors:409EFF,67C23A;render_default:mp4" validate:"required,oneof=mp4 webm"`
}

// VideoCompressResp 视频压缩响应结构体
type VideoCompressResp struct {
	// 压缩后的视频文件
	OutputFile string `json:"output_file" widget:"name:压缩后的视频;type:files"`

	// 压缩信息
	CompressInfo string `json:"compress_info" widget:"name:压缩信息;type:text_area"`

	// 压缩统计
	CompressionStats string `json:"compression_stats" widget:"name:压缩统计;type:text_area"`

	// 原始文件信息
	OriginalFileInfo string `json:"original_file_info" widget:"name:原始文件信息;type:text_area"`
}

// VideoCompress 视频压缩函数
func VideoCompress(ctx *app.Context, resp response.Response) error {
	var req VideoCompressReq
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

	// 2. 使用 FFmpeg 进行视频压缩（依赖 PATH，镜像内已安装）
	const ffmpegCmd = "ffmpeg"
	if _, err := exec.LookPath(ffmpegCmd); err != nil {
		logger.Errorf(ctx, "[系统错误]-[VideoCompress] FFmpeg 未在 PATH 中, req: %+v, err: %v", req, err)
		return fmt.Errorf("[系统错误]-[VideoCompress]： FFmpeg未安装或不在 PATH 中，请确保运行环境已安装 FFmpeg")
	}

	// 获取原始文件信息
	var originalSize float64
	if stat, err := os.Stat(file); err == nil {
		originalSize = float64(stat.Size()) / (1024 * 1024) // MB
	}
	originalInfo := fmt.Sprintf("原始文件: %s\n大小: %.2f MB", filepath.Base(file), originalSize)

	// 使用 GetTraceOutputDir 生成唯一的输出目录
	outputDir := fs.GetTraceOutputDir()

	// 生成输出文件名
	baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	timestamp := time.Now().Format("20060102_150405")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s_compressed_%s.%s", baseName, timestamp, req.OutputFormat))

	// 构建FFmpeg命令
	var args []string
	var compressionDetails string

	// 基本参数
	args = []string{"-i", file}

	// 根据压缩模式添加参数
	switch req.CompressionMode {
	case "智能压缩":
		// 智能压缩：根据原始文件大小自动选择参数
		if originalSize > 100 {
			// 大文件：使用较强的压缩
			args = append(args, "-crf", "28")
			compressionDetails = "智能压缩模式：大文件优化 (CRF: 28)"
		} else if originalSize > 20 {
			// 中等文件：平衡压缩
			args = append(args, "-crf", "23")
			compressionDetails = "智能压缩模式：平衡优化 (CRF: 23)"
		} else {
			// 小文件：轻度压缩
			args = append(args, "-crf", "18")
			compressionDetails = "智能压缩模式：轻度优化 (CRF: 18)"
		}

	case "指定文件大小":
		// 计算目标码率（kbps）
		duration := 60.0 // 读取失败时按 1 分钟降级估算。
		if info, err := probeThumbnailVideo(file); err == nil && info.Duration > 0 {
			duration = info.Duration
		} else if err != nil {
			logger.Warnf(ctx, "[VideoCompress] 读取视频时长失败，使用 60 秒估算目标码率: %v", err)
		}
		targetBitrateKbps := int((req.TargetSize * 8 * 1024) / duration)

		// 限制范围
		if targetBitrateKbps < 100 {
			targetBitrateKbps = 100
		}
		if targetBitrateKbps > 10000 {
			targetBitrateKbps = 10000
		}

		args = append(args, "-b:v", fmt.Sprintf("%dk", targetBitrateKbps))
		compressionDetails = fmt.Sprintf("指定文件大小：目标 %.1f MB，视频时长 %.1f 秒，计算码率 %d kbps", req.TargetSize, duration, targetBitrateKbps)

	case "指定视频码率":
		args = append(args, "-b:v", fmt.Sprintf("%dk", req.TargetBitrate))
		compressionDetails = fmt.Sprintf("指定视频码率：%d kbps", req.TargetBitrate)

	case "指定分辨率":
		// 添加分辨率缩放
		var scale string
		switch req.TargetResolution {
		case "1080p(1920x1080)":
			scale = "scale=1920:1080"
		case "720p(1280x720)":
			scale = "scale=1280:720"
		case "480p(854x480)":
			scale = "scale=854:480"
		case "360p(640x360)":
			scale = "scale=640:360"
		case "240p(426x240)":
			scale = "scale=426:240"
		default:
			scale = "scale=1280:720"
		}
		args = append(args, "-vf", scale)
		compressionDetails = fmt.Sprintf("指定分辨率：%s", req.TargetResolution)
	}

	// 添加压缩质量参数
	switch req.CompressionQuality {
	case "高质量(文件较大)":
		if !strings.Contains(strings.Join(args, " "), "-crf") {
			args = append(args, "-crf", "18")
		}
		compressionDetails += "，质量：高质量"
	case "平衡质量":
		if !strings.Contains(strings.Join(args, " "), "-crf") {
			args = append(args, "-crf", "23")
		}
		compressionDetails += "，质量：平衡"
	case "高压缩(文件较小)":
		if !strings.Contains(strings.Join(args, " "), "-crf") {
			args = append(args, "-crf", "28")
		}
		compressionDetails += "，质量：高压缩"
	}

	// 帧率处理
	if !req.KeepOriginalFramerate {
		targetFramerate := req.TargetFramerate
		if targetFramerate <= 0 {
			targetFramerate = 30
		}
		if targetFramerate > 120 {
			targetFramerate = 120
		}
		args = append(args, "-r", strconv.Itoa(targetFramerate))
		compressionDetails += fmt.Sprintf("，帧率：%dfps", targetFramerate)
	} else {
		compressionDetails += "，帧率：保持原始"
	}

	// 音频处理
	if req.KeepOriginalAudio {
		args = append(args, "-c:a", "copy")
		compressionDetails += "，音频：保持原始"
	} else {
		audioBitrate := req.AudioBitrate
		if audioBitrate <= 0 {
			audioBitrate = 128
		}
		if audioBitrate > 320 {
			audioBitrate = 320
		}
		audioCodec := "aac"
		if req.OutputFormat == "webm" {
			audioCodec = "libopus"
		}
		args = append(args, "-c:a", audioCodec, "-b:a", fmt.Sprintf("%dk", audioBitrate))
		compressionDetails += fmt.Sprintf("，音频：压缩(%dk)", audioBitrate)
	}

	// 视频编码器
	if req.OutputFormat == "mp4" {
		args = append(args, "-c:v", "libx264")
	} else if req.OutputFormat == "webm" {
		args = append(args, "-c:v", "libvpx-vp9")
	}

	// 添加输出文件
	args = append(args, "-y", outputPath)

	// 执行FFmpeg命令
	logger.Infof(ctx, "[VideoCompress] 执行FFmpeg命令: %s %s", ffmpegCmd, strings.Join(args, " "))
	cmd := exec.Command(ffmpegCmd, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		errorMsg := string(output)
		if len(errorMsg) > 500 {
			errorMsg = errorMsg[:500] + "..."
		}
		logger.Errorf(ctx, "[系统错误]-[VideoCompress] 视频压缩失败, req: %+v, err: %v, output: %s", req, err, errorMsg)
		return fmt.Errorf("[系统错误]-[VideoCompress]： 视频压缩失败, err: %v", err)
	}

	logger.Infof(ctx, "[VideoCompress] 压缩成功: %s -> %s", file, outputPath)

	// 3. 上传压缩后的文件
	outputFiles := fs.ResponseFiles([]string{outputPath})

	// 4. 获取压缩统计信息
	compressedSize := 0.0
	fileInfo, err := os.Stat(outputPath)
	if err == nil {
		compressedSize = float64(fileInfo.Size()) / (1024 * 1024) // MB
	}

	compressionRatio := 0.0
	if originalSize > 0 {
		compressionRatio = (1 - compressedSize/originalSize) * 100
	}

	compressionStats := fmt.Sprintf("原始大小: %.2f MB\n压缩后大小: %.2f MB\n压缩比例: %.1f%%\n节省空间: %.2f MB",
		originalSize, compressedSize, compressionRatio, originalSize-compressedSize)

	// 5. 构建响应信息
	compressInfo := fmt.Sprintf("视频压缩完成！\n\n%s\n输出格式: %s\n压缩详情: %s",
		originalInfo, req.OutputFormat, compressionDetails)

	// 6. 构建响应
	return resp.Form(&VideoCompressResp{
		OutputFile:       outputFiles,
		CompressInfo:     compressInfo,
		CompressionStats: compressionStats,
		OriginalFileInfo: originalInfo,
	}).Build()
}

// VideoCompressTemplate 视频压缩配置
var VideoCompressTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "视频压缩",
		Desc:     `使用FFmpeg进行视频压缩，支持智能压缩、指定文件大小、指定视频码率、指定分辨率等多种压缩模式。可调整压缩质量、帧率、音频质量等参数，有效减小视频文件大小。`,
		Tags:     []string{"视频处理", "视频压缩", "FFmpeg", "工具"},
		Request:  &VideoCompressReq{},
		Response: &VideoCompressResp{},
	},
}

func init() {
	// 注册Form函数 - 视频压缩
	packageContext.POST("compress.form", VideoCompress, VideoCompressTemplate)
}
