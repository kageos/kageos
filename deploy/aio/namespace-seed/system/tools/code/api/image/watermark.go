package image

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

type WatermarkReq struct {
	InputFiles    string `json:"input_files" widget:"name:上传图片;type:files;accept:image/*;max_size:200MB;max_count:50" validate:"required"`
	Mode          string `json:"mode" widget:"name:水印类型;type:select;options:文字水印,图片水印;options_colors:409EFF,67C23A;render_default:文字水印" validate:"required,oneof=文字水印 图片水印"`
	WatermarkText string `json:"watermark_text" widget:"name:水印文字;type:input;placeholder:例如 公司内部资料;desc:文字水印模式显示并生效" validate:"required_if=Mode 文字水印"`
	LogoFile      string `json:"logo_file" widget:"name:Logo水印图片;type:files;accept:image/*;max_size:50MB;max_count:1;desc:图片水印模式显示并生效" validate:"required_if=Mode 图片水印"`
	Position      string `json:"position" widget:"name:水印位置;type:select;options:右下角,右上角,左下角,左上角,居中;options_colors:409EFF,67C23A,909399,E6A23C,F56C6C;render_default:右下角" validate:"required,oneof=右下角 右上角 左下角 左上角 居中"`
	OffsetX       int    `json:"offset_x" widget:"name:水平边距;type:integer;min:0;max:1000;step:1;unit:px;render_default:24" validate:"min=0,max=1000"`
	OffsetY       int    `json:"offset_y" widget:"name:垂直边距;type:integer;min:0;max:1000;step:1;unit:px;render_default:24" validate:"min=0,max=1000"`
	Opacity       int    `json:"opacity" widget:"name:透明度;type:slider;min:1;max:100;step:1;unit:%;render_default:65" validate:"min=0,max=100"`
	FontSize      int    `json:"font_size" widget:"name:文字字号;type:integer;min:8;max:200;step:1;unit:px;render_default:36;desc:文字水印模式显示并生效" validate:"required_if=Mode 文字水印,min=0,max=200"`
	TextColor     string `json:"text_color" widget:"name:文字颜色;type:select;options:白色,黑色,红色,黄色;options_colors:909399,409EFF,F56C6C,E6A23C;render_default:白色;desc:文字水印模式显示并生效" validate:"required_if=Mode 文字水印,omitempty,oneof=白色 黑色 红色 黄色"`
	OutputFormat  string `json:"output_format" widget:"name:输出格式;type:select;options:原格式,jpeg,png,webp;options_colors:909399,409EFF,67C23A,909399;render_default:原格式" validate:"required,oneof=原格式 jpeg png webp"`
}

type WatermarkResp struct {
	OutputFiles string `json:"output_files" widget:"name:加水印后的图片;type:files"`
	RunInfo     string `json:"run_info" widget:"name:处理信息;type:text_area"`
}

func Watermark(ctx *app.Context, resp response.Response) error {
	var req WatermarkReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoWatermark(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoWatermark(ctx *app.Context, req *WatermarkReq) (*WatermarkResp, error) {
	if req.Mode == "图片水印" {
		if _, err := exec.LookPath("convert"); err != nil {
			return nil, fmt.Errorf("未找到 convert，请确认运行环境已安装 ImageMagick")
		}
	} else {
		if _, err := exec.LookPath("ffmpeg"); err != nil {
			return nil, fmt.Errorf("未找到 ffmpeg，请确认运行环境已安装 FFmpeg")
		}
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入图片")
	}
	var logoPath string
	var logoFiles []string
	if req.Mode == "图片水印" {
		logoFiles = fs.DownloadFiles(req.LogoFile)
		defer fs.RemoveFiles(logoFiles)
		if len(logoFiles) == 0 || logoFiles[0] == "" {
			return nil, fmt.Errorf("图片水印模式需要上传 Logo 水印图片")
		}
		logoPath = logoFiles[0]
	}

	outputDir := fs.GetTraceOutputDir()
	tempDir, err := os.MkdirTemp(outputDir, "image_watermark_*")
	if err != nil {
		return nil, fmt.Errorf("创建水印临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	fontFile := ""
	if req.Mode == "文字水印" {
		fontFile, err = watermarkChineseFontFile()
		if err != nil {
			return nil, err
		}
	}
	opacity := clampWatermarkOpacity(req.Opacity)
	fontSize := req.FontSize
	if fontSize <= 0 {
		fontSize = 36
	}
	offsetX := maxInt(req.OffsetX, 0)
	offsetY := maxInt(req.OffsetY, 0)
	gravity := watermarkGravity(req.Position)
	geometry := fmt.Sprintf("+%d+%d", offsetX, offsetY)
	seenNames := make(map[string]int)
	var outputPaths []string
	var infos []string
	successCount := 0
	failCount := 0

	for _, file := range inputFiles {
		if file == "" {
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}
		ext := watermarkOutputExt(req.OutputFormat, file)
		outputName := outputFileName(filepath.Base(file), file, "_watermarked", ext, seenNames)
		outputPath := filepath.Join(outputDir, outputName)

		var err error
		if req.Mode == "图片水印" {
			err = applyLogoWatermark(ctx, file, logoPath, outputPath, tempDir, gravity, geometry, opacity)
		} else {
			err = applyTextWatermark(ctx, file, outputPath, req.Position, offsetX, offsetY, req.WatermarkText, req.TextColor, opacity, fontSize, fontFile, tempDir)
		}
		if err != nil {
			logger.Warnf(ctx, "[Image/Watermark] 处理失败 %s: %v", filepath.Base(file), err)
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v", filepath.Base(file), err))
			continue
		}
		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputName))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功添加水印的图片\n%s", strings.Join(infos, "\n"))
	}

	info := fmt.Sprintf("图片水印处理完成\n水印类型: %s\n位置: %s\n透明度: %d%%\n成功: %d\n失败: %d", req.Mode, req.Position, opacity, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &WatermarkResp{OutputFiles: fs.ResponseFiles(outputPaths), RunInfo: info}, nil
}

func applyTextWatermark(ctx *app.Context, inputPath, outputPath, position string, offsetX, offsetY int, text, color string, opacity, fontSize int, fontFile, tempDir string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("文字水印内容不能为空")
	}
	textFile, err := os.CreateTemp(tempDir, "watermark_text_*.txt")
	if err != nil {
		return fmt.Errorf("创建水印文本临时文件失败: %w", err)
	}
	if _, err := textFile.WriteString(text); err != nil {
		_ = textFile.Close()
		return fmt.Errorf("写入水印文本失败: %w", err)
	}
	if err := textFile.Close(); err != nil {
		return fmt.Errorf("关闭水印文本临时文件失败: %w", err)
	}

	x, y := watermarkDrawtextPosition(position, offsetX, offsetY)
	filter := fmt.Sprintf(
		"drawtext=fontfile='%s':textfile='%s':fontcolor=%s:fontsize=%s:x=%s:y=%s:expansion=none",
		escapeFFmpegFilterQuoted(fontFile),
		escapeFFmpegFilterQuoted(textFile.Name()),
		watermarkDrawtextColor(color, opacity),
		strconv.Itoa(fontSize),
		x,
		y,
	)
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-i",
		inputPath,
		"-vf", filter,
		"-frames:v", "1",
		outputPath,
	}
	out, err := exec.Command("ffmpeg", args...).CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[Image/applyTextWatermark] ffmpeg drawtext 失败 %s: %v, output: %s", filepath.Base(inputPath), err, string(out))
		return fmt.Errorf("文字水印失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func applyLogoWatermark(ctx *app.Context, inputPath, logoPath, outputPath, tempDir, gravity, geometry string, opacity int) error {
	opacityValue := fmt.Sprintf("%.3f", float64(opacity)/100.0)
	adjustedLogo := filepath.Join(tempDir, "logo_"+sanitizeFileName(filepath.Base(logoPath), "watermark.png"))
	out, err := exec.Command("convert", logoPath, "-alpha", "set", "-channel", "A", "-evaluate", "multiply", opacityValue, "+channel", adjustedLogo).CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[Image/applyLogoWatermark] logo 透明度处理失败: %v, output: %s", err, string(out))
		return fmt.Errorf("处理 Logo 透明度失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	out, err = exec.Command("convert", inputPath, adjustedLogo, "-gravity", gravity, "-geometry", geometry, "-composite", outputPath).CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[Image/applyLogoWatermark] convert 失败 %s: %v, output: %s", filepath.Base(inputPath), err, string(out))
		return fmt.Errorf("图片水印失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func watermarkGravity(position string) string {
	switch position {
	case "右上角":
		return "NorthEast"
	case "左下角":
		return "SouthWest"
	case "左上角":
		return "NorthWest"
	case "居中":
		return "Center"
	default:
		return "SouthEast"
	}
}

func watermarkChineseFontFile() (string, error) {
	candidates := []string{
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.otf",
		"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
	}
	for _, path := range candidates {
		if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
			return path, nil
		}
	}
	if fcMatch, err := exec.LookPath("fc-match"); err == nil {
		out, err := exec.Command(fcMatch, "-f", "%{file}", "Noto Sans CJK SC").Output()
		if err == nil {
			path := strings.TrimSpace(string(out))
			if stat, statErr := os.Stat(path); path != "" && statErr == nil && !stat.IsDir() && likelyChineseFontPath(path) {
				return path, nil
			}
		}
	}
	return "", fmt.Errorf("未找到可用于中文水印的字体，请确认运行环境已安装 Noto CJK 字体，优先路径为 /usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc")
}

func likelyChineseFontPath(path string) bool {
	lower := strings.ToLower(path)
	keywords := []string{
		"noto", "cjk", "wqy", "wenquanyi", "sourcehan", "source-han",
		"pingfang", "hiragino", "songti", "heiti", "kaiti", "simhei",
		"simsun", "yahei", "sarasa", "droidsansfallback",
	}
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func watermarkDrawtextPosition(position string, offsetX, offsetY int) (string, string) {
	x := strconv.Itoa(maxInt(offsetX, 0))
	y := strconv.Itoa(maxInt(offsetY, 0))
	switch position {
	case "右上角":
		return fmt.Sprintf("w-text_w-%d", maxInt(offsetX, 0)), y
	case "左下角":
		return x, fmt.Sprintf("h-text_h-%d", maxInt(offsetY, 0))
	case "左上角":
		return x, y
	case "居中":
		return drawtextCenterExpr("w", "text_w", offsetX), drawtextCenterExpr("h", "text_h", offsetY)
	default:
		return fmt.Sprintf("w-text_w-%d", maxInt(offsetX, 0)), fmt.Sprintf("h-text_h-%d", maxInt(offsetY, 0))
	}
}

func drawtextCenterExpr(full, text string, offset int) string {
	expr := fmt.Sprintf("(%s-%s)/2", full, text)
	if offset > 0 {
		return fmt.Sprintf("%s+%d", expr, offset)
	}
	return expr
}

func watermarkDrawtextColor(color string, opacity int) string {
	alpha := float64(clampWatermarkOpacity(opacity)) / 100.0
	switch color {
	case "黑色":
		return fmt.Sprintf("black@%.3f", alpha)
	case "红色":
		return fmt.Sprintf("0xDC2626@%.3f", alpha)
	case "黄色":
		return fmt.Sprintf("0xFACC15@%.3f", alpha)
	default:
		return fmt.Sprintf("white@%.3f", alpha)
	}
}

func escapeFFmpegFilterQuoted(value string) string {
	return strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(value)
}

func watermarkOutputExt(format, inputPath string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "jpeg", "png", "webp":
		return format
	default:
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(inputPath)), ".")
		switch ext {
		case "jpg":
			return "jpeg"
		case "jpeg", "png", "webp", "gif", "tif", "tiff", "bmp":
			return ext
		default:
			return "png"
		}
	}
}

func clampWatermarkOpacity(opacity int) int {
	if opacity <= 0 {
		return 65
	}
	if opacity > 100 {
		return 100
	}
	return opacity
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var WatermarkTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "图片添加水印",
		Desc:     `使用 FFmpeg drawtext 为图片批量添加中文文字水印，显式指定 Noto CJK 字体避免中文方框；Logo 图片水印继续使用 ImageMagick 叠加。支持位置、边距、透明度、字号和输出格式设置。适合报告配图、版权标识、内部资料标记和交付图片处理。`,
		Tags:     []string{"图片", "水印", "Logo", "文字水印", "FFmpeg", "drawtext", "中文", "ImageMagick", "批量处理"},
		Request:  &WatermarkReq{},
		Response: &WatermarkResp{},
	},
}

func init() {
	packageContext.POST("watermark.form", Watermark, WatermarkTemplate)
}
