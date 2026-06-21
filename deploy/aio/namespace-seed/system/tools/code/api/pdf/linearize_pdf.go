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

type LinearizePDFReq struct {
	InputFiles     string `json:"input_files" widget:"name:上传 PDF 文件;type:files;accept:.pdf,application/pdf;max_size:200MB;max_count:20" validate:"required"`
	OutputFileName string `json:"output_file_name" widget:"name:输出文件名;type:input;placeholder:可选，仅单文件时生效，例如 web.pdf"`
}

type LinearizePDFResp struct {
	OutputFiles   string `json:"output_files" widget:"name:线性化后的 PDF;type:files"`
	LinearizeInfo string `json:"linearize_info" widget:"name:线性化信息;type:text_area"`
}

func LinearizePDF(ctx *app.Context, resp response.Response) error {
	var req LinearizePDFReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoLinearizePDF(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoLinearizePDF(ctx *app.Context, req *LinearizePDFReq) (*LinearizePDFResp, error) {
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

	outputDir := fs.GetTraceOutputDir()
	seenNames := make(map[string]int)
	var outputPaths []string
	var infos []string
	successCount := 0
	failCount := 0

	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[QPDF/LinearizePDF] 文件 %s 无本地路径，跳过", filepath.Base(file))
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		outputName := outputPDFName(filepath.Base(file), file, "_linearized", seenNames)
		if len(files) == 1 && strings.TrimSpace(req.OutputFileName) != "" {
			outputName = explicitPDFName(req.OutputFileName, "linearized.pdf")
		}
		outputPath := filepath.Join(outputDir, outputName)
		cmd := exec.Command("qpdf", "--linearize", file, outputPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[QPDF/LinearizePDF] 线性化失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputName))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功线性化的 PDF 文件\n%s", strings.Join(infos, "\n"))
	}

	outputFiles := fs.ResponseFiles(outputPaths)

	info := fmt.Sprintf("PDF 线性化完成\n成功: %d\n失败: %d", successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &LinearizePDFResp{OutputFiles: outputFiles, LinearizeInfo: info}, nil
}

var LinearizePDFTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "线性化 PDF",
		Desc:     `使用 qpdf --linearize 生成 Web 友好的线性化 PDF，让浏览器或客户端可以更快开始预览大文件。该操作主要重排 PDF 结构，不负责压缩图片体积。`,
		Tags:     []string{"PDF", "线性化", "Web优化", "qpdf", "文档处理"},
		Request:  &LinearizePDFReq{},
		Response: &LinearizePDFResp{},
	},
}

func init() {
	packageContext.POST("linearize.form", LinearizePDF, LinearizePDFTemplate)
}
