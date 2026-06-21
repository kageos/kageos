package pdf

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

type OCRToTextReq struct {
	InputFiles     string `json:"input_files" widget:"name:上传 PDF 文件;type:files;accept:.pdf,application/pdf;max_size:500MB;max_count:20" validate:"required"`
	Language       string `json:"language" widget:"name:OCR语言;type:select;options:中文+英文,中文,英文;options_colors:409EFF,67C23A,909399;render_default:中文+英文" validate:"required,oneof=中文+英文 中文 英文"`
	DPI            int    `json:"dpi" widget:"name:OCR渲染 DPI;type:integer;min:100;max:400;render_default:200" validate:"min=0,max=400"`
	FirstPage      int    `json:"first_page" widget:"name:开始页;type:integer;min:0;render_default:0;placeholder:0 表示从第一页开始" validate:"min=0"`
	LastPage       int    `json:"last_page" widget:"name:结束页;type:integer;min:0;render_default:0;placeholder:0 表示到最后一页" validate:"min=0"`
	MinTextChars   int    `json:"min_text_chars" widget:"name:直接提取最少字符数;type:integer;min:0;max:2000;render_default:50;placeholder:达到该字符数则不再 OCR" validate:"min=0,max=2000"`
	PreserveLayout bool   `json:"preserve_layout" widget:"name:直接提取时保留版式;type:switch;render_default:true"`
}

type OCRToTextResp struct {
	OutputFiles string `json:"output_files" widget:"name:文本文件;type:files"`
	OutputText  string `json:"output_text" widget:"name:文本预览;type:text_area"`
	OCRInfo     string `json:"ocr_info" widget:"name:处理信息;type:text_area"`
}

func OCRToText(ctx *app.Context, resp response.Response) error {
	var req OCRToTextReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoOCRToText(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoOCRToText(ctx *app.Context, req *OCRToTextReq) (*OCRToTextResp, error) {
	if err := ensureCommand("pdftotext", "poppler-utils"); err != nil {
		return nil, err
	}
	if err := ensureCommand("pdftoppm", "poppler-utils"); err != nil {
		return nil, err
	}
	if err := ensureCommand("tesseract", "Tesseract"); err != nil {
		return nil, err
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	outputDir := fs.GetTraceOutputDir()
	tempDir, err := os.MkdirTemp(outputDir, "pdf_ocr_text_*")
	if err != nil {
		return nil, fmt.Errorf("创建 OCR 临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	minTextChars := req.MinTextChars
	if minTextChars <= 0 {
		minTextChars = 50
	}
	dpi := req.DPI
	if dpi <= 0 {
		dpi = 200
	}
	if dpi < 100 {
		dpi = 100
	}
	if dpi > 400 {
		dpi = 400
	}

	seenNames := make(map[string]int)
	var outputPaths []string
	var previews []string
	var infos []string
	successCount := 0
	failCount := 0

	for _, file := range inputFiles {
		if file == "" {
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		text, mode, err := extractPDFTextOrOCR(ctx, file, tempDir, req, dpi, minTextChars)
		if err != nil {
			logger.Errorf(ctx, "[PDF/OCRToText] 处理失败 %s: %v", filepath.Base(file), err)
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v", filepath.Base(file), err))
			continue
		}
		outputName := outputFileName(filepath.Base(file), file, "_ocr_text", "txt", seenNames)
		outputPath := filepath.Join(outputDir, outputName)
		if err := os.WriteFile(outputPath, []byte(strings.TrimSpace(text)+"\n"), 0644); err != nil {
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: 写入文本文件失败: %v", filepath.Base(file), err))
			continue
		}
		outputPaths = append(outputPaths, outputPath)
		successCount++
		preview := readTextPreview(outputPath, 120000)
		if len(inputFiles) > 1 {
			preview = fmt.Sprintf("## %s\n%s", filepath.Base(file), preview)
		}
		previews = append(previews, preview)
		infos = append(infos, fmt.Sprintf("成功 %s -> %s（%s）", filepath.Base(file), outputName, mode))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功识别文本的 PDF 文件\n%s", strings.Join(infos, "\n"))
	}

	pageInfo := "全部页面"
	if req.FirstPage > 0 || req.LastPage > 0 {
		pageInfo = fmt.Sprintf("%s-%s", pageNumLabel(req.FirstPage, "第一页"), pageNumLabel(req.LastPage, "最后一页"))
	}
	info := fmt.Sprintf("PDF OCR 文本提取完成\n页面范围: %s\nOCR语言: %s\nDPI: %d\n成功: %d\n失败: %d", pageInfo, req.Language, dpi, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &OCRToTextResp{
		OutputFiles: fs.ResponseFiles(outputPaths),
		OutputText:  strings.Join(previews, "\n\n"),
		OCRInfo:     info,
	}, nil
}

func extractPDFTextOrOCR(ctx *app.Context, file, tempDir string, req *OCRToTextReq, dpi int, minTextChars int) (string, string, error) {
	directText, directErr := directPDFText(ctx, file, req.PreserveLayout, req.FirstPage, req.LastPage)
	if directErr == nil && len([]rune(strings.TrimSpace(directText))) >= minTextChars {
		return directText, "直接提取文本层", nil
	}

	prefix := sanitizeFileName(strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)), "pdf") + "_ocr"
	imagePaths, err := renderPDFPagesForOCR(ctx, file, tempDir, prefix, dpi, req.FirstPage, req.LastPage)
	if err != nil {
		if directErr != nil {
			return "", "", fmt.Errorf("直接提取失败且 OCR 渲染失败: %v; %w", directErr, err)
		}
		return "", "", err
	}
	for _, path := range imagePaths {
		defer os.Remove(path)
	}

	var pageTexts []string
	for i, imagePath := range imagePaths {
		text, err := tesseractImageToText(ctx, imagePath, languageCode(req.Language))
		if err != nil {
			logger.Warnf(ctx, "[PDF/OCRToText] 第 %d 页 OCR 失败: %v", i+1, err)
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		pageTexts = append(pageTexts, fmt.Sprintf("--- 第 %d 页 ---\n%s", i+1, text))
	}
	if len(pageTexts) == 0 {
		return "", "", fmt.Errorf("OCR 未识别到可用文本")
	}
	return strings.Join(pageTexts, "\n\n"), "渲染页面后 OCR", nil
}

func directPDFText(ctx *app.Context, file string, preserveLayout bool, firstPage, lastPage int) (string, error) {
	args := make([]string, 0, 8)
	if preserveLayout {
		args = append(args, "-layout")
	}
	var err error
	args, err = appendPageArgs(args, firstPage, lastPage)
	if err != nil {
		return "", err
	}
	args = append(args, file, "-")
	out, err := exec.Command("pdftotext", args...).CombinedOutput()
	if err != nil {
		logger.Warnf(ctx, "[PDF/directPDFText] pdftotext 失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
		return "", fmt.Errorf("pdftotext 执行失败: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func renderPDFPagesForOCR(ctx *app.Context, file, tempDir, prefix string, dpi, firstPage, lastPage int) ([]string, error) {
	prefixPath := filepath.Join(tempDir, prefix)
	args := []string{"-png", "-r", strconv.Itoa(dpi)}
	var err error
	args, err = appendPageArgs(args, firstPage, lastPage)
	if err != nil {
		return nil, err
	}
	args = append(args, file, prefixPath)
	out, err := exec.Command("pdftoppm", args...).CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[PDF/renderPDFPagesForOCR] pdftoppm 失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
		return nil, fmt.Errorf("pdftoppm 渲染失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	paths, err := collectOutputFiles(tempDir, filepath.Base(prefixPath)+"-", map[string]bool{"png": true})
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("PDF 渲染后未生成页面图片")
	}
	return paths, nil
}

func tesseractImageToText(ctx *app.Context, imagePath, lang string) (string, error) {
	out, err := exec.Command("tesseract", imagePath, "stdout", "-l", lang).CombinedOutput()
	if err != nil {
		logger.Warnf(ctx, "[PDF/tesseractImageToText] OCR 失败 %s: %v, output: %s", filepath.Base(imagePath), err, string(out))
		return "", fmt.Errorf("tesseract 执行失败: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

var OCRToTextTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "PDF OCR 提取文本",
		Desc:     `上传 PDF 输出文本文件：先尝试直接提取内嵌文本层，文本不足时自动渲染页面并用 Tesseract OCR 识别。适合扫描件、图片型 PDF、合同和资料抽取。`,
		Tags:     []string{"PDF", "OCR", "文本提取", "Tesseract", "Poppler", "扫描件"},
		Request:  &OCRToTextReq{},
		Response: &OCRToTextResp{},
	},
}

func init() {
	packageContext.POST("ocr_to_text.form", OCRToText, OCRToTextTemplate)
}
