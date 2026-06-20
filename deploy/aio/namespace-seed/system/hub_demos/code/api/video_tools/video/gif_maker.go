package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// GifMakerReq GIF制作请求结构体
type GifMakerReq struct {
	// 上传视频文件
	InputFile string `json:"input_file" widget:"name:上传视频文件;type:files;accept:video/*;max_size:500MB;max_count:1" validate:"required"`

	// 开始时间（秒）
	StartTime float64 `json:"start_time" widget:"name:开始时间(秒);type:float;min:0;step:0.1;unit:秒;render_default:0"`

	// 持续时间（秒）
	Duration float64 `json:"duration" widget:"name:持续时间(秒);type:float;min:0.1;max:30;step:0.1;unit:秒;render_default:5"`

	// GIF宽度
	GifWidth int `json:"gif_width" widget:"name:GIF宽度;type:integer;min:100;max:1920;step:10;unit:像素;render_default:480"`

	// GIF帧率
	GifFps int `json:"gif_fps" widget:"name:GIF帧率;type:integer;min:1;max:30;step:1;unit:fps;render_default:10"`

	// 优化级别
	OptimizationLevel string `json:"optimization_level" widget:"name:优化级别;type:select;options:高质量(文件较大),平衡质量,高压缩(文件较小);options_colors:67C23A,409EFF,E6A23C;render_default:平衡质量"`

	// 是否循环
	LoopEnabled bool `json:"loop_enabled" widget:"name:循环播放;type:switch;render_default:true"`

	// 循环次数（0表示无限循环）
	LoopCount int `json:"loop_count" widget:"name:循环次数;type:integer;min:0;max:100;step:1;unit:次;render_default:0;desc:开启循环播放后显示，0 表示无限循环" validate:"required_if=LoopEnabled true,min=0,max=100"`

	// 添加文字水印
	WatermarkText string `json:"watermark_text" widget:"name:文字水印;type:input"`

	// 水印位置
	WatermarkPosition string `json:"watermark_position" widget:"name:水印位置;type:select;options:左上角,右上角,左下角,右下角,居中;options_colors:409EFF,67C23A,909399,E6A23C,F56C6C;render_default:右下角;desc:填写文字水印后显示并生效" validate:"required_with=WatermarkText"`
}

// GifMakerResp GIF制作响应结构体
type GifMakerResp struct {
	// 生成的GIF文件
	OutputGif string `json:"output_gif" widget:"name:生成的GIF;type:files"`

	// 制作信息
	MakeInfo string `json:"make_info" widget:"name:制作信息;type:text_area"`

	// GIF参数
	GifParams string `json:"gif_params" widget:"name:GIF参数;type:text_area"`

	// 文件大小对比
	SizeComparison string `json:"size_comparison" widget:"name:文件大小对比;type:text_area"`
}

// GifMaker GIF制作函数
func GifMaker(ctx *app.Context, resp response.Response) error {
	var req GifMakerReq
	err := ctx.ShouldBindValidate(&req)
	if err != nil {
		return err
	}

	// 验证参数
	if req.Duration <= 0 {
		return fmt.Errorf("持续时间必须大于0")
	}
	if req.GifWidth <= 0 {
		return fmt.Errorf("GIF宽度必须大于0")
	}
	if req.GifFps <= 0 {
		return fmt.Errorf("GIF帧率必须大于0")
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

	// 2. 使用 FFmpeg 制作 GIF（依赖 PATH，镜像内已安装）
	const ffmpegCmd = "ffmpeg"
	if _, err := exec.LookPath(ffmpegCmd); err != nil {
		logger.Errorf(ctx, "[系统错误]-[GifMaker] FFmpeg 未在 PATH 中, req: %+v, err: %v", req, err)
		return fmt.Errorf("[系统错误]-[GifMaker]： FFmpeg未安装或不在 PATH 中，请确保运行环境已安装 FFmpeg")
	}

	// 使用 GetTraceOutputDir 生成唯一的输出目录
	outputDir := fs.GetTraceOutputDir()

	// 生成输出文件名
	baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	timestamp := time.Now().Format("20060102_150405")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s_%s.gif", baseName, timestamp))

	// 构建FFmpeg命令
	var args []string
	var gifParams []string

	// 基本参数
	args = []string{
		"-i", file,
		"-ss", fmt.Sprintf("%.2f", req.StartTime),
		"-t", fmt.Sprintf("%.2f", req.Duration),
	}

	// 添加水印
	if req.WatermarkText != "" {
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

		args = append(args, "-vf", fmt.Sprintf("scale=%d:-1,drawtext=text='%s':fontcolor=white:fontsize=20:box=1:boxcolor=black@0.5:boxborderw=5:x=%s:y=%s",
			req.GifWidth, escapeFFmpegDrawtextText(req.WatermarkText), strings.Split(position, ":")[0], strings.Split(position, ":")[1]))
		gifParams = append(gifParams, fmt.Sprintf("水印: '%s' (位置: %s)", req.WatermarkText, req.WatermarkPosition))
	} else {
		args = append(args, "-vf", fmt.Sprintf("scale=%d:-1", req.GifWidth))
	}

	// 帧率
	args = append(args, "-r", strconv.Itoa(req.GifFps))
	gifParams = append(gifParams, fmt.Sprintf("帧率: %d fps", req.GifFps))

	// 优化级别
	switch req.OptimizationLevel {
	case "高质量(文件较大)":
		args = append(args, "-gifflags", "+transdiff")
		gifParams = append(gifParams, "优化: 高质量")
	case "平衡质量":
		args = append(args, "-gifflags", "+transdiff", "-compression_level", "6")
		gifParams = append(gifParams, "优化: 平衡质量")
	case "高压缩(文件较小)":
		args = append(args, "-gifflags", "+transdiff", "-compression_level", "9")
		gifParams = append(gifParams, "优化: 高压缩")
	}

	// 循环设置
	if req.LoopEnabled {
		if req.LoopCount == 0 {
			args = append(args, "-loop", "0")
			gifParams = append(gifParams, "循环: 无限循环")
		} else {
			args = append(args, "-loop", strconv.Itoa(req.LoopCount))
			gifParams = append(gifParams, fmt.Sprintf("循环: %d 次", req.LoopCount))
		}
	} else {
		args = append(args, "-loop", "-1")
		gifParams = append(gifParams, "循环: 不循环")
	}

	// 添加输出文件
	args = append(args, "-y", outputPath)

	// 执行FFmpeg命令
	logger.Infof(ctx, "[GifMaker] 执行FFmpeg命令: %s %s", ffmpegCmd, strings.Join(args, " "))
	cmd := exec.Command(ffmpegCmd, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		errorMsg := string(output)
		if len(errorMsg) > 500 {
			errorMsg = errorMsg[:500] + "..."
		}
		logger.Errorf(ctx, "[系统错误]-[GifMaker] GIF制作失败, req: %+v, err: %v, output: %s", req, err, errorMsg)
		return fmt.Errorf("[系统错误]-[GifMaker]： GIF制作失败, err: %v", err)
	}

	logger.Infof(ctx, "[GifMaker] GIF制作成功: %s -> %s", file, outputPath)

	// 3. 上传生成的GIF文件
	outputFiles := fs.ResponseFiles([]string{outputPath})

	// 4. 获取文件大小信息
	var originalSize float64
	if stat, err := os.Stat(file); err == nil {
		originalSize = float64(stat.Size()) / (1024 * 1024) // MB
	}
	compressedSize := 0.0
	fileInfo, err := os.Stat(outputPath)
	if err == nil {
		compressedSize = float64(fileInfo.Size()) / 1024 // KB
	}

	sizeComparison := fmt.Sprintf("原始视频: %.2f MB\n生成GIF: %.2f KB", originalSize, compressedSize)

	// 5. 构建制作信息
	makeInfo := fmt.Sprintf("GIF制作完成！\n\n输入文件: %s\n开始时间: %.2f 秒\n持续时间: %.2f 秒\nGIF宽度: %d 像素\n\n%s",
		filepath.Base(file), req.StartTime, req.Duration, req.GifWidth, strings.Join(gifParams, "\n"))

	// 6. 构建响应
	return resp.Form(&GifMakerResp{
		OutputGif:      outputFiles,
		MakeInfo:       makeInfo,
		GifParams:      strings.Join(gifParams, "\n"),
		SizeComparison: sizeComparison,
	}).Build()
}

// GifMakerTemplate GIF制作配置
var GifMakerTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "GIF制作",
		Desc:     `使用FFmpeg将视频片段转换为GIF动画。支持设置开始时间、持续时间、GIF宽度、帧率、优化级别、循环播放等参数。可添加文字水印，支持多种水印位置选择。`,
		Tags:     []string{"视频处理", "GIF制作", "FFmpeg", "工具"},
		Request:  &GifMakerReq{},
		Response: &GifMakerResp{},
	},
}

func init() {
	// 注册Form函数 - GIF制作
	packageContext.POST("gif.form", GifMaker, GifMakerTemplate)
}
