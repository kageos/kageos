package pdf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type CompressPDFReq struct {
	InputFiles     string `json:"input_files" widget:"name:上传 PDF 文件;type:files;accept:.pdf,application/pdf;max_size:500MB;max_count:20" validate:"required"`
	Quality        string `json:"quality" widget:"name:压缩档位;type:select;options:屏幕预览,电子书,打印,印前,默认;options_colors:E6A23C,409EFF,67C23A,909399,909399;render_default:电子书" validate:"required,oneof=屏幕预览 电子书 打印 印前 默认"`
	OutputFileName string `json:"output_file_name" widget:"name:输出文件名;type:input;placeholder:可选，仅单文件时生效，例如 compressed.pdf"`
}

type CompressPDFResp struct {
	OutputFiles  string `json:"output_files" widget:"name:压缩后的 PDF;type:files"`
	CompressInfo string `json:"compress_info" widget:"name:压缩信息;type:text_area"`
}

func CompressPDF(ctx *app.Context, resp response.Response) error {
	var req CompressPDFReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoCompressPDF(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoCompressPDF(ctx *app.Context, req *CompressPDFReq) (*CompressPDFResp, error) {
	if _, err := exec.LookPath("gs"); err != nil {
		return nil, fmt.Errorf("未找到 gs，请确认运行环境已安装 Ghostscript")
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	outputDir := fs.GetTraceOutputDir()
	seen := make(map[string]int)
	var outputPaths []string
	var infos []string
	successCount := 0
	failCount := 0
	pdfSetting := gsPDFSetting(req.Quality)

	for _, input := range inputFiles {
		if input == "" {
			continue
		}
		outputName := gsOutputPDFName(filepath.Base(input), "_compressed", seen)
		if len(inputFiles) == 1 && strings.TrimSpace(req.OutputFileName) != "" {
			outputName = gsEnsurePDFName(req.OutputFileName)
		}
		outputPath := filepath.Join(outputDir, outputName)
		args := []string{
			"-sDEVICE=pdfwrite",
			"-dCompatibilityLevel=1.4",
			"-dPDFSETTINGS=" + pdfSetting,
			"-dNOPAUSE",
			"-dQUIET",
			"-dBATCH",
			"-dDetectDuplicateImages=true",
			"-dCompressFonts=true",
			"-dSubsetFonts=true",
			"-sOutputFile=" + outputPath,
			input,
		}
		out, err := exec.Command("gs", args...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[CompressPDF] gs 压缩失败 %s: %v, output: %s", filepath.Base(input), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(input), err, strings.TrimSpace(string(out))))
			continue
		}
		origSize := fileSize(input)
		newSize := fileSize(outputPath)
		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s，%s -> %s（%.1f%%）",
			filepath.Base(input), outputName, gsHumanBytes(origSize), gsHumanBytes(newSize), compressionPercent(origSize, newSize)))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功压缩的 PDF 文件\n%s", strings.Join(infos, "\n"))
	}

	outputFiles := fs.ResponseFiles(outputPaths)
	info := fmt.Sprintf("PDF 压缩完成\n压缩档位: %s (%s)\n成功: %d\n失败: %d", req.Quality, pdfSetting, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &CompressPDFResp{OutputFiles: outputFiles, CompressInfo: info}, nil
}

func gsPDFSetting(quality string) string {
	switch strings.TrimSpace(quality) {
	case "屏幕预览":
		return "/screen"
	case "打印":
		return "/printer"
	case "印前":
		return "/prepress"
	case "默认":
		return "/default"
	default:
		return "/ebook"
	}
}

func gsOutputPDFName(name, suffix string, seen map[string]int) string {
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	base = gsSanitizeName(base, "document")
	candidate := base + suffix + ".pdf"
	if _, ok := seen[candidate]; !ok {
		seen[candidate] = 1
		return candidate
	}
	for {
		seen[candidate]++
		next := fmt.Sprintf("%s%s_%d.pdf", base, suffix, seen[candidate])
		if _, ok := seen[next]; !ok {
			seen[next] = 1
			return next
		}
	}
}

func gsEnsurePDFName(name string) string {
	name = gsSanitizeName(strings.TrimSpace(name), "compressed")
	if strings.EqualFold(filepath.Ext(name), ".pdf") {
		return name
	}
	return strings.TrimSuffix(name, filepath.Ext(name)) + ".pdf"
}

func gsSanitizeName(name, fallback string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = fallback
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	name = replacer.Replace(name)
	if strings.TrimSpace(name) == "" {
		return fallback
	}
	return name
}

func fileSize(path string) int64 {
	stat, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return stat.Size()
}

func compressionPercent(original, compressed int64) float64 {
	if original <= 0 {
		return 0
	}
	return float64(compressed) / float64(original) * 100
}

func gsHumanBytes(size int64) string {
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

var CompressPDFTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "压缩 PDF",
		Desc:     `使用 Ghostscript 压缩 PDF，支持屏幕预览、电子书、打印、印前和默认档位。适合把扫描件、图片型 PDF、报告 PDF 体积变小；如果只是拆页/抽页/线性化，请使用 qpdf 工具。`,
		Tags:     []string{"PDF", "压缩", "Ghostscript", "gs", "文件变小", "扫描件"},
		Request:  &CompressPDFReq{},
		Response: &CompressPDFResp{},
	},
}

func init() {
	packageContext.POST("compress.form", CompressPDF, CompressPDFTemplate)
}
