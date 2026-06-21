package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// VideoEditReq 视频剪辑请求结构体
type VideoEditReq struct {
	// 上传视频文件
	InputFile string `json:"input_file" widget:"name:上传视频文件;type:files;accept:video/*;max_size:500MB;max_count:1" validate:"required"`

	// 剪辑操作类型
	Operation string `json:"operation" widget:"name:剪辑操作;type:select;options:裁剪时间,提取片段,添加水印,调整速度;options_colors:409EFF,67C23A,E6A23C,F56C6C;render_default:裁剪时间" validate:"required,oneof=裁剪时间 提取片段 添加水印 调整速度"`

	// 开始时间（秒）
	StartTime float64 `json:"start_time" widget:"name:开始时间(秒);type:float;min:0;step:0.1;unit:秒;render_default:0"`

	// 结束时间（秒）
	EndTime float64 `json:"end_time" widget:"name:结束时间(秒);type:float;min:0;step:0.1;unit:秒;render_default:10"`

	// 水印文字
	WatermarkText string `json:"watermark_text" widget:"name:水印文字;type:input;desc:添加水印模式显示并生效" validate:"required_if=Operation 添加水印"`

	// 水印位置
	WatermarkPosition string `json:"watermark_position" widget:"name:水印位置;type:select;options:左上角,右上角,左下角,右下角,居中;options_colors:409EFF,67C23A,909399,E6A23C,F56C6C;render_default:右下角;desc:添加水印模式显示并生效" validate:"required_if=Operation 添加水印"`

	// 水印字体大小
	WatermarkFontSize int `json:"watermark_font_size" widget:"name:水印字体大小;type:integer;min:12;max:200;step:1;render_default:36;unit:像素;desc:水印文字大小，添加水印模式显示并生效" validate:"required_if=Operation 添加水印"`

	// 播放速度
	PlaybackSpeed float64 `json:"playback_speed" widget:"name:播放速度;type:float;min:0.1;max:4.0;step:0.1;unit:倍速;render_default:1.0;desc:调整速度模式显示并生效" validate:"required_if=Operation 调整速度,min=0,max=4"`

	// 输出格式
	OutputFormat string `json:"output_format" widget:"name:输出格式;type:select;options:mp4,webm,avi,mkv;options_colors:409EFF,67C23A,909399,E6A23C;render_default:mp4" validate:"required,oneof=mp4 webm avi mkv"`
}

// VideoEditResp 视频剪辑响应结构体
type VideoEditResp struct {
	// 剪辑后的视频文件
	OutputFile string `json:"output_file" widget:"name:剪辑后的视频;type:files"`

	// 剪辑信息
	EditInfo string `json:"edit_info" widget:"name:剪辑信息;type:text_area"`

	// 操作详情
	OperationDetails string `json:"operation_details" widget:"name:操作详情;type:text_area"`
}

// VideoEdit 视频剪辑函数
func VideoEdit(ctx *app.Context, resp response.Response) error {
	var req VideoEditReq
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

	// 2. 使用 FFmpeg 进行视频剪辑（依赖 PATH，镜像内已安装）
	const ffmpegCmd = "ffmpeg"
	if _, err := exec.LookPath(ffmpegCmd); err != nil {
		logger.Errorf(ctx, "[系统错误]-[VideoEdit] FFmpeg 未在 PATH 中, req: %+v, err: %v", req, err)
		return fmt.Errorf("[系统错误]-[VideoEdit]： FFmpeg未安装或不在 PATH 中，请确保运行环境已安装 FFmpeg")
	}

	// 使用 GetTraceOutputDir 生成唯一的输出目录
	outputDir := fs.GetTraceOutputDir()

	// 生成输出文件名
	baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	timestamp := time.Now().Format("20060102_150405")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s_edited_%s.%s", baseName, timestamp, req.OutputFormat))

	// 构建FFmpeg命令
	var args []string
	var operationDetails string

	switch req.Operation {
	case "裁剪时间":
		// 裁剪指定时间段
		duration := req.EndTime - req.StartTime
		if duration <= 0 {
			return fmt.Errorf("结束时间必须大于开始时间")
		}

		args = []string{
			"-i", file,
			"-ss", fmt.Sprintf("%.2f", req.StartTime),
			"-t", fmt.Sprintf("%.2f", duration),
			"-c", "copy",
			"-y", outputPath,
		}
		operationDetails = fmt.Sprintf("裁剪时间: %.2f秒 - %.2f秒 (时长: %.2f秒)", req.StartTime, req.EndTime, duration)

	case "提取片段":
		// 提取指定片段
		duration := req.EndTime - req.StartTime
		if duration <= 0 {
			return fmt.Errorf("结束时间必须大于开始时间")
		}

		args = []string{
			"-i", file,
			"-ss", fmt.Sprintf("%.2f", req.StartTime),
			"-t", fmt.Sprintf("%.2f", duration),
			"-c:v", "libx264",
			"-c:a", "aac",
			"-y", outputPath,
		}
		operationDetails = fmt.Sprintf("提取片段: %.2f秒 - %.2f秒 (时长: %.2f秒)", req.StartTime, req.EndTime, duration)

	case "添加水印":
		// 添加文字水印
		if req.WatermarkText == "" {
			return fmt.Errorf("请填写水印文字")
		}

		// 构建水印位置参数
		var position string
		switch req.WatermarkPosition {
		case "左上角":
			position = "10:10"
		case "右上角":
			position = "main_w-text_w-10:10"
		case "左下角":
			position = "10:main_h-text_h-10"
		case "右下角":
			position = "main_w-text_w-10:main_h-text_h-10"
		case "居中":
			position = "(main_w-text_w)/2:(main_h-text_h)/2"
		default:
			position = "main_w-text_w-10:main_h-text_h-10"
		}

		args = []string{
			"-i", file,
			"-vf", fmt.Sprintf("drawtext=text='%s':fontcolor=white:fontsize=%d:box=1:boxcolor=black@0.5:boxborderw=5:x=%s:y=%s",
				escapeFFmpegDrawtextText(req.WatermarkText), req.WatermarkFontSize, strings.Split(position, ":")[0], strings.Split(position, ":")[1]),
			"-c:a", "copy",
			"-y", outputPath,
		}
		operationDetails = fmt.Sprintf("添加水印: '%s' (位置: %s, 字体大小: %d)", req.WatermarkText, req.WatermarkPosition, req.WatermarkFontSize)

	case "调整速度":
		// 调整播放速度
		if req.PlaybackSpeed <= 0 {
			return fmt.Errorf("播放速度必须大于0")
		}

		// 计算音频速度
		audioSpeed := 1.0 / req.PlaybackSpeed

		args = []string{
			"-i", file,
			"-filter_complex", fmt.Sprintf("[0:v]setpts=%f*PTS[v];[0:a]atempo=%f[a]", 1.0/req.PlaybackSpeed, audioSpeed),
			"-map", "[v]",
			"-map", "[a]",
			"-y", outputPath,
		}
		operationDetails = fmt.Sprintf("调整速度: %.1f倍速", req.PlaybackSpeed)

	default:
		return fmt.Errorf("不支持的剪辑操作")
	}

	// 执行FFmpeg命令
	logger.Infof(ctx, "[VideoEdit] 执行FFmpeg命令: %s %s", ffmpegCmd, strings.Join(args, " "))
	cmd := exec.Command(ffmpegCmd, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		errorMsg := string(output)
		if len(errorMsg) > 500 {
			errorMsg = errorMsg[:500] + "..."
		}
		logger.Errorf(ctx, "[系统错误]-[VideoEdit] 视频剪辑失败, req: %+v, err: %v, output: %s", req, err, errorMsg)
		return fmt.Errorf("[系统错误]-[VideoEdit]： 视频剪辑失败, err: %v", err)
	}

	logger.Infof(ctx, "[VideoEdit] 剪辑成功: %s -> %s", file, outputPath)

	// 3. 上传剪辑后的文件
	outputFiles := fs.ResponseFiles([]string{outputPath})

	// 4. 构建响应信息
	editInfo := fmt.Sprintf("视频剪辑完成！\n\n操作类型: %s\n输入文件: %s\n输出文件: %s\n输出格式: %s\n\n%s",
		req.Operation, filepath.Base(file), filepath.Base(outputPath), req.OutputFormat, operationDetails)

	// 获取文件大小信息
	fileInfo, err := os.Stat(outputPath)
	if err == nil {
		sizeMB := float64(fileInfo.Size()) / (1024 * 1024)
		editInfo += fmt.Sprintf("\n输出文件大小: %.2f MB", sizeMB)
	}

	// 5. 构建响应
	return resp.Form(&VideoEditResp{
		OutputFile:       outputFiles,
		EditInfo:         editInfo,
		OperationDetails: operationDetails,
	}).Build()
}

// VideoEditTemplate 视频剪辑配置
var VideoEditTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "视频剪辑",
		Desc:     `使用FFmpeg进行视频剪辑操作，支持裁剪时间、提取片段、添加水印、调整播放速度等功能。可精确控制开始和结束时间，添加自定义文字水印，调整视频播放速度。`,
		Tags:     []string{"视频处理", "视频剪辑", "FFmpeg", "工具"},
		Request:  &VideoEditReq{},
		Response: &VideoEditResp{},
	},
}

func init() {
	// 注册Form函数 - 视频剪辑
	packageContext.POST("edit.form", VideoEdit, VideoEditTemplate)
}
