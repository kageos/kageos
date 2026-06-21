package pdf

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type ExtractPagesReq struct {
	InputFiles     string `json:"input_files" widget:"name:上传 PDF 文件;type:files;accept:.pdf,application/pdf;max_size:200MB;max_count:20" validate:"required"`
	PageRange      string `json:"page_range" widget:"name:页码范围;type:input;placeholder:例如 1-3,5,z 或 1-10:even" validate:"required"`
	OutputFileName string `json:"output_file_name" widget:"name:输出文件名;type:input;placeholder:可选，仅单文件时生效，例如 extract.pdf"`
}

type ExtractPagesResp struct {
	OutputFiles string `json:"output_files" widget:"name:抽取后的 PDF;type:files"`
	ExtractInfo string `json:"extract_info" widget:"name:抽取信息;type:text_area"`
}

func ExtractPages(ctx *app.Context, resp response.Response) error {
	var req ExtractPagesReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoExtractPages(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoExtractPages(ctx *app.Context, req *ExtractPagesReq) (*ExtractPagesResp, error) {
	if err := ensureQPDF(); err != nil {
		return nil, err
	}
	pageRange, err := normalizePageRange(req.PageRange)
	if err != nil {
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
	var infos []string
	successCount := 0
	failCount := 0

	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[QPDF/ExtractPages] 文件 %s 无本地路径，跳过", filepath.Base(file))
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		outputName := outputPDFName(filepath.Base(file), file, "_pages", seenNames)
		if len(files) == 1 && strings.TrimSpace(req.OutputFileName) != "" {
			outputName = explicitPDFName(req.OutputFileName, "extract.pdf")
		}
		outputPath := filepath.Join(outputDir, outputName)
		cmd := exec.Command("qpdf", file, "--pages", ".", pageRange, "--", outputPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[QPDF/ExtractPages] 抽取失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputName))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功抽取的 PDF 文件\n%s", strings.Join(infos, "\n"))
	}

	outputFiles := fs.ResponseFiles(outputPaths)

	info := fmt.Sprintf("PDF 页面抽取完成\n页码范围: %s\n成功: %d\n失败: %d", pageRange, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &ExtractPagesResp{OutputFiles: outputFiles, ExtractInfo: info}, nil
}

var ExtractPagesTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "抽取 PDF 页面",
		Desc:     `使用 qpdf 从 PDF 中抽取指定页面范围，适合从大 PDF 中导出部分章节、合同页或附件页。页码范围支持 1-3,5,z、r1、1-10:even 等 qpdf 语法。`,
		Tags:     []string{"PDF", "抽页", "页面", "qpdf", "文档处理"},
		Request:  &ExtractPagesReq{},
		Response: &ExtractPagesResp{},
	},
}

func init() {
	packageContext.POST("extract_pages.form", ExtractPages, ExtractPagesTemplate)
}
