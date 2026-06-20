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

type PDFToTextReq struct {
	InputFiles     string `json:"input_files" widget:"name:上传 PDF 文件;type:files;accept:.pdf,application/pdf;max_size:200MB;max_count:20" validate:"required"`
	PreserveLayout bool   `json:"preserve_layout" widget:"name:尽量保留版式;type:switch;render_default:true"`
	FirstPage      int    `json:"first_page" widget:"name:开始页;type:integer;min:0;render_default:0;placeholder:0 表示从第一页开始" validate:"min=0"`
	LastPage       int    `json:"last_page" widget:"name:结束页;type:integer;min:0;render_default:0;placeholder:0 表示到最后一页" validate:"min=0"`
}

type PDFToTextResp struct {
	OutputFiles string `json:"output_files" widget:"name:文本文件;type:files"`
	OutputText  string `json:"output_text" widget:"name:文本预览;type:text_area"`
	ExtractInfo string `json:"extract_info" widget:"name:提取信息;type:text_area"`
}

func PDFToText(ctx *app.Context, resp response.Response) error {
	var req PDFToTextReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoPDFToText(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoPDFToText(ctx *app.Context, req *PDFToTextReq) (*PDFToTextResp, error) {
	if err := ensureCommand("pdftotext", "poppler-utils"); err != nil {
		return nil, err
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
	var previews []string
	var infos []string
	successCount := 0
	failCount := 0

	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[Poppler/PDFToText] 文件 %s 无本地路径，跳过", filepath.Base(file))
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		outputName := outputFileName(filepath.Base(file), file, "_text", "txt", seenNames)
		outputPath := filepath.Join(outputDir, outputName)
		args := make([]string, 0, 8)
		if req.PreserveLayout {
			args = append(args, "-layout")
		}
		var err error
		args, err = appendPageArgs(args, req.FirstPage, req.LastPage)
		if err != nil {
			return nil, err
		}
		args = append(args, file, outputPath)
		out, err := exec.Command("pdftotext", args...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[Poppler/PDFToText] 提取失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		outputPaths = append(outputPaths, outputPath)
		successCount++
		preview := readTextPreview(outputPath, 120000)
		if len(files) > 1 {
			preview = fmt.Sprintf("## %s\n%s", filepath.Base(file), preview)
		}
		previews = append(previews, preview)
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputName))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功提取文本的 PDF 文件\n%s", strings.Join(infos, "\n"))
	}

	outputFiles := fs.ResponseFiles(outputPaths)

	pageInfo := "全部页面"
	if req.FirstPage > 0 || req.LastPage > 0 {
		pageInfo = fmt.Sprintf("%s-%s", pageNumLabel(req.FirstPage, "第一页"), pageNumLabel(req.LastPage, "最后一页"))
	}
	info := fmt.Sprintf("PDF 文本提取完成\n页面范围: %s\n保留版式: %t\n成功: %d\n失败: %d", pageInfo, req.PreserveLayout, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &PDFToTextResp{
		OutputFiles: outputFiles,
		OutputText:  strings.Join(previews, "\n\n"),
		ExtractInfo: info,
	}, nil
}

func pageNumLabel(page int, fallback string) string {
	if page <= 0 {
		return fallback
	}
	return strconv.Itoa(page)
}

var PDFToTextTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "PDF 直接提取文本",
		Desc:     `使用 pdftotext 直接提取 PDF 内嵌文本层，适合文本型 PDF、电子合同、报告和论文。扫描件或图片型 PDF 请优先使用 OCRmyPDF 或 Tesseract OCR。`,
		Tags:     []string{"PDF", "文本提取", "Poppler", "pdftotext", "文档处理"},
		Request:  &PDFToTextReq{},
		Response: &PDFToTextResp{},
	},
}

func init() {
	packageContext.POST("extract_text.form", PDFToText, PDFToTextTemplate)
}
