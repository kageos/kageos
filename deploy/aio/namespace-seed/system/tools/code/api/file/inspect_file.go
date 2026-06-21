package file

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type InspectFileReq struct {
	InputFiles    string `json:"input_files" widget:"name:上传文件;type:files;accept:*/*;max_size:2000MB;max_count:50" validate:"required"`
	Detailed      bool   `json:"detailed" widget:"name:详细模式;type:switch;render_default:false"`
	ComputeSHA256 bool   `json:"compute_sha256" widget:"name:计算 SHA256;type:switch;render_default:false"`
	OutputReport  bool   `json:"output_report" widget:"name:输出报告文件;type:switch;render_default:true"`
}

type InspectFileResp struct {
	InspectionText string `json:"inspection_text" widget:"name:体检结果;type:text_area"`
	OutputFile     string `json:"output_file" widget:"name:体检报告;type:files"`
	Summary        string `json:"summary" widget:"name:处理说明;type:text_area"`
}

func InspectFile(ctx *app.Context, resp response.Response) error {
	var req InspectFileReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoInspectFile(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoInspectFile(ctx *app.Context, req *InspectFileReq) (*InspectFileResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	var blocks []string
	successCount := 0
	for _, path := range inputFiles {
		if path == "" {
			continue
		}
		block, err := inspectOneFile(ctx, path, req.Detailed, req.ComputeSHA256)
		if err != nil {
			logger.Warnf(ctx, "[InspectFile] 体检失败 %s: %v", filepath.Base(path), err)
			blocks = append(blocks, fmt.Sprintf("## %s\n体检失败: %v", filepath.Base(path), err))
			continue
		}
		successCount++
		blocks = append(blocks, block)
	}

	if len(blocks) == 0 {
		return nil, fmt.Errorf("没有成功体检的文件")
	}

	report := strings.Join(blocks, "\n\n---\n\n")
	var outputFile string
	if req.OutputReport || len(report) > 120000 {
		outputDir := fs.GetTraceOutputDir()
		outputPath := filepath.Join(outputDir, "file_inspection_report.txt")
		if err := os.WriteFile(outputPath, []byte(report), 0644); err != nil {
			return nil, fmt.Errorf("写入体检报告失败: %w", err)
		}
		outputFile = fs.ResponseFiles([]string{outputPath})
	}

	return &InspectFileResp{
		InspectionText: report,
		OutputFile:     outputFile,
		Summary:        fmt.Sprintf("文件体检完成\n输入文件数: %d\n成功体检: %d\n详细模式: %t\n计算 SHA256: %t", len(inputFiles), successCount, req.Detailed, req.ComputeSHA256),
	}, nil
}

func inspectOneFile(ctx *app.Context, path string, detailed bool, computeSHA256 bool) (string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	mimeType := strings.TrimSpace(runOptionalCommand(ctx, "file", "-b", "--mime-type", path))
	fileDesc := strings.TrimSpace(runOptionalCommand(ctx, "file", "-b", path))

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s\n", filepath.Base(path)))
	sb.WriteString(fmt.Sprintf("- 路径: `%s`\n", path))
	sb.WriteString(fmt.Sprintf("- 大小: %s (%d bytes)\n", humanBytes(stat.Size()), stat.Size()))
	sb.WriteString(fmt.Sprintf("- 修改时间: %s\n", stat.ModTime().Format(time.RFC3339)))
	if fileDesc != "" {
		sb.WriteString(fmt.Sprintf("- file: %s\n", fileDesc))
	}
	if mimeType != "" {
		sb.WriteString(fmt.Sprintf("- MIME: %s\n", mimeType))
	}
	if computeSHA256 {
		hash, err := sha256File(path)
		if err != nil {
			sb.WriteString(fmt.Sprintf("- SHA256: 计算失败: %v\n", err))
		} else {
			sb.WriteString(fmt.Sprintf("- SHA256: `%s`\n", hash))
		}
	}

	if isPDFFile(path, mimeType, fileDesc) {
		appendCommandBlock(&sb, "pdfinfo", runOptionalCommand(ctx, "pdfinfo", path))
		appendCommandBlock(&sb, "qpdf --check", runOptionalCommand(ctx, "qpdf", "--check", path))
	}
	if isImageFile(mimeType) {
		appendCommandBlock(&sb, "identify", runOptionalCommand(ctx, "identify", "-format", "format=%m\nwidth=%w\nheight=%h\ncolorspace=%r\nquality=%Q\n", path))
	}
	if isMediaFile(mimeType) {
		appendCommandBlock(&sb, "ffprobe", runOptionalCommand(ctx, "ffprobe", "-v", "error", "-show_entries", "format=format_name,duration,size,bit_rate:stream=index,codec_type,codec_name,width,height,sample_rate,channels,duration,bit_rate", "-of", "json", path))
	}
	if detailed || isImageFile(mimeType) || isPDFFile(path, mimeType, fileDesc) || isMediaFile(mimeType) {
		args := []string{"-FileType", "-MIMEType", "-FileSize", "-ImageSize", "-Duration", "-VideoFrameRate", "-AudioSampleRate", "-PDFVersion", "-PageCount", path}
		if detailed {
			args = []string{"-a", "-G1", "-s", path}
		}
		appendCommandBlock(&sb, "exiftool", truncateText(runOptionalCommand(ctx, "exiftool", args...), 30000))
	}

	return sb.String(), nil
}

func runOptionalCommand(ctx *app.Context, name string, args ...string) string {
	if _, err := exec.LookPath(name); err != nil {
		return ""
	}
	cmdCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cmdCtx, name, args...).CombinedOutput()
	text := strings.TrimSpace(string(out))
	if cmdCtx.Err() == context.DeadlineExceeded {
		return "命令超时"
	}
	if err != nil && text == "" {
		logger.Debugf(ctx, "[InspectFile] %s 执行失败: %v", name, err)
		return ""
	}
	return text
}

func appendCommandBlock(sb *strings.Builder, title string, output string) {
	output = strings.TrimSpace(output)
	if output == "" {
		return
	}
	sb.WriteString(fmt.Sprintf("\n### %s\n```text\n%s\n```\n", title, output))
}

func isImageFile(mimeType string) bool {
	return strings.HasPrefix(strings.ToLower(mimeType), "image/")
}

func isMediaFile(mimeType string) bool {
	mimeType = strings.ToLower(mimeType)
	return strings.HasPrefix(mimeType, "video/") || strings.HasPrefix(mimeType, "audio/")
}

func isPDFFile(path, mimeType, fileDesc string) bool {
	lowerPath := strings.ToLower(path)
	return strings.Contains(strings.ToLower(mimeType), "pdf") ||
		strings.Contains(strings.ToLower(fileDesc), "pdf") ||
		strings.HasSuffix(lowerPath, ".pdf")
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func humanBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

func truncateText(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes]) + "\n...（输出过长，已截断）"
}

var InspectFileTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "文件体检",
		Desc:     `上传任意文件，自动识别文件类型、大小、MIME、图片尺寸、音视频编码/时长、PDF 页数/结构和常见元数据，可选计算 SHA256。适合工作台在处理文件前先判断应该走 PDF、图片、音视频、OCR 还是文档转换链路。`,
		Tags:     []string{"文件", "体检", "MIME", "元数据", "PDF", "图片", "音视频", "file", "exiftool", "ffprobe"},
		Request:  &InspectFileReq{},
		Response: &InspectFileResp{},
	},
}

func init() {
	packageContext.POST("inspect.form", InspectFile, InspectFileTemplate)
}
