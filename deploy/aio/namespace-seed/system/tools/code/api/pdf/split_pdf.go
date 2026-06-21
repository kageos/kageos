package pdf

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type SplitPDFReq struct {
	InputFiles   string `json:"input_files" widget:"name:上传 PDF 文件;type:files;accept:.pdf,application/pdf;max_size:200MB;max_count:20" validate:"required"`
	PagesPerFile int    `json:"pages_per_file" widget:"name:每个文件页数;type:integer;min:1;max:100;render_default:1" validate:"min=0,max=100"`
	OutputPrefix string `json:"output_prefix" widget:"name:输出文件名前缀;type:input;placeholder:可选，仅单文件时生效，例如 chapter"`
}

type SplitPDFResp struct {
	OutputFiles string `json:"output_files" widget:"name:拆分后的 PDF;type:files"`
	SplitInfo   string `json:"split_info" widget:"name:拆分信息;type:text_area"`
}

func SplitPDF(ctx *app.Context, resp response.Response) error {
	var req SplitPDFReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoSplitPDF(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoSplitPDF(ctx *app.Context, req *SplitPDFReq) (*SplitPDFResp, error) {
	if err := ensureQPDF(); err != nil {
		return nil, err
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	pagesPerFile := req.PagesPerFile
	if pagesPerFile <= 0 {
		pagesPerFile = 1
	}
	if pagesPerFile > 100 {
		return nil, fmt.Errorf("每个文件页数不能超过 100")
	}

	outputDir := fs.GetTraceOutputDir()
	seenStems := make(map[string]int)
	var outputPaths []string
	var infos []string
	successCount := 0
	failCount := 0

	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[QPDF/SplitPDF] 文件 %s 无本地路径，跳过", filepath.Base(file))
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		stem := outputPDFStem(filepath.Base(file), file, "_split", seenStems)
		if len(files) == 1 && strings.TrimSpace(req.OutputPrefix) != "" {
			stem = sanitizeFileName(req.OutputPrefix, "split")
		}
		outputPattern := filepath.Join(outputDir, stem+"_%d.pdf")
		cmd := exec.Command("qpdf", "--split-pages="+strconv.Itoa(pagesPerFile), file, outputPattern)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[QPDF/SplitPDF] 拆分失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		paths, err := splitPDFOutputPaths(outputPattern)
		if err != nil {
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: 查找输出文件失败: %v", filepath.Base(file), err))
			continue
		}
		if len(paths) == 0 {
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: qpdf 执行成功但未找到拆分结果", filepath.Base(file)))
			continue
		}

		outputPaths = append(outputPaths, paths...)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %d 个文件", filepath.Base(file), len(paths)))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功拆分的 PDF 文件\n%s", strings.Join(infos, "\n"))
	}

	outputFiles := fs.ResponseFiles(outputPaths)

	info := fmt.Sprintf("PDF 拆分完成\n每个文件页数: %d\n成功输入文件: %d\n失败输入文件: %d\n输出文件数: %d", pagesPerFile, successCount, failCount, len(outputPaths))
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &SplitPDFResp{OutputFiles: outputFiles, SplitInfo: info}, nil
}

var SplitPDFTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "拆分 PDF",
		Desc:     `使用 qpdf 按页数拆分 PDF。默认每页输出一个 PDF，也可以设置每 N 页输出一个文件，适合把扫描件、合同、报告按页面或章节拆开。`,
		Tags:     []string{"PDF", "拆分", "分页", "qpdf", "文档处理"},
		Request:  &SplitPDFReq{},
		Response: &SplitPDFResp{},
	},
}

func init() {
	packageContext.POST("split.form", SplitPDF, SplitPDFTemplate)
}
