package pdf

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type MakeSearchablePDFReq struct {
	InputFiles    string `json:"input_files" widget:"name:上传PDF文件;type:files;accept:.pdf,application/pdf;max_size:500MB;max_count:20" validate:"required"`
	Language      string `json:"language" widget:"name:OCR语言;type:select;options:中文+英文,中文,英文;options_colors:409EFF,67C23A,909399;render_default:中文+英文" validate:"required,oneof=中文+英文 中文 英文"`
	OCRMode       string `json:"ocr_mode" widget:"name:OCR模式;type:select;options:跳过已有文本,强制重新OCR;options_colors:409EFF,E6A23C;render_default:跳过已有文本" validate:"required,oneof=跳过已有文本 强制重新OCR"`
	Deskew        bool   `json:"deskew" widget:"name:自动纠偏;type:switch;render_default:true"`
	RotatePages   bool   `json:"rotate_pages" widget:"name:自动旋转页面;type:switch;render_default:true"`
	OptimizeLevel int    `json:"optimize_level" widget:"name:优化级别;type:integer;min:0;max:3;render_default:1" validate:"min=0,max=3"`
}

type MakeSearchablePDFResp struct {
	OutputFiles string `json:"output_files" widget:"name:可搜索PDF;type:files"`
	OCRInfo     string `json:"ocr_info" widget:"name:OCR信息;type:text_area"`
}

func MakeSearchablePDF(ctx *app.Context, resp response.Response) error {
	var req MakeSearchablePDFReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoMakeSearchablePDF(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoMakeSearchablePDF(ctx *app.Context, req *MakeSearchablePDFReq) (*MakeSearchablePDFResp, error) {
	if _, err := exec.LookPath("ocrmypdf"); err != nil {
		return nil, fmt.Errorf("未找到 ocrmypdf，请确认运行环境已安装 OCRmyPDF")
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	outputDir := fs.GetTraceOutputDir()
	seenNames := make(map[string]int)
	var outputPaths []string
	var infos []string
	successCount := 0
	failCount := 0

	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[OCRmyPDF/MakeSearchablePDF] 文件 %s 无本地路径，跳过", filepath.Base(file))
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		outputName := outputFileName(filepath.Base(file), file, "_ocr", "pdf", seenNames)
		outputPath := filepath.Join(outputDir, outputName)
		args := buildOCRArgs(req, file, outputPath)

		cmd := exec.Command("ocrmypdf", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[OCRmyPDF/MakeSearchablePDF] OCR失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputName))
		if text := strings.TrimSpace(string(out)); text != "" {
			infos = append(infos, text)
		}
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功生成的 OCR PDF\n%s", strings.Join(infos, "\n"))
	}

	outputFiles := fs.ResponseFiles(outputPaths)

	info := fmt.Sprintf(
		"OCR PDF 生成完成\n语言: %s\n模式: %s\n自动纠偏: %t\n自动旋转: %t\n优化级别: %d\n成功: %d\n失败: %d",
		req.Language,
		req.OCRMode,
		req.Deskew,
		req.RotatePages,
		req.OptimizeLevel,
		successCount,
		failCount,
	)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &MakeSearchablePDFResp{
		OutputFiles: outputFiles,
		OCRInfo:     info,
	}, nil
}

func buildOCRArgs(req *MakeSearchablePDFReq, inputPath, outputPath string) []string {
	args := []string{"-l", languageCode(req.Language)}
	switch req.OCRMode {
	case "强制重新OCR":
		args = append(args, "--force-ocr")
	default:
		args = append(args, "--skip-text")
	}
	if req.Deskew {
		args = append(args, "--deskew")
	}
	if req.RotatePages {
		args = append(args, "--rotate-pages")
	}
	args = append(args, "--optimize", strconv.Itoa(req.OptimizeLevel), inputPath, outputPath)
	return args
}

var MakeSearchablePDFTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "扫描PDF转可搜索PDF",
		Desc:     `使用 OCRmyPDF 为扫描版 PDF 添加文字层，输出可搜索、可复制文本的 PDF。支持中文、英文、中英混合，支持自动纠偏和页面自动旋转。默认跳过已有文本层，避免重复 OCR。`,
		Tags:     []string{"PDF", "OCR", "扫描件", "可搜索PDF", "OCRmyPDF", "中文OCR"},
		Request:  &MakeSearchablePDFReq{},
		Response: &MakeSearchablePDFResp{},
	},
}

func init() {
	packageContext.POST("ocr.form", MakeSearchablePDF, MakeSearchablePDFTemplate)
}
