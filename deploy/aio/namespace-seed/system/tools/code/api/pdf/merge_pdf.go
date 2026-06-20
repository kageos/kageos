package pdf

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type MergePDFReq struct {
	InputFiles     string `json:"input_files" widget:"name:上传 PDF 文件;type:files;accept:.pdf,application/pdf;max_size:200MB;max_count:50" validate:"required"`
	OutputFileName string `json:"output_file_name" widget:"name:输出文件名;type:input;placeholder:可选，例如 merged.pdf"`
}

type MergePDFResp struct {
	OutputFiles string `json:"output_files" widget:"name:合并后的 PDF;type:files"`
	MergeInfo   string `json:"merge_info" widget:"name:合并信息;type:text_area"`
}

func MergePDF(ctx *app.Context, resp response.Response) error {
	var req MergePDFReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoMergePDF(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoMergePDF(ctx *app.Context, req *MergePDFReq) (*MergePDFResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) < 2 {
		return nil, fmt.Errorf("至少需要 2 个 PDF 文件")
	}

	if _, err := exec.LookPath("pdfunite"); err != nil {
		return nil, fmt.Errorf("未找到 pdfunite，请确认运行环境已安装 poppler-utils")
	}

	outputBase := strings.TrimSpace(req.OutputFileName)
	if outputBase == "" {
		outputBase = "merged.pdf"
	}
	outputBase = strings.TrimSuffix(outputBase, filepath.Ext(outputBase)) + ".pdf"
	outputBase = sanitizePDFName(outputBase)
	outputPath := filepath.Join(fs.GetTraceOutputDir(), outputBase)

	args := make([]string, 0, len(files)+1)
	var sourceNames []string
	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[MergePDF] 文件 %s 无本地路径，跳过", filepath.Base(file))
			continue
		}
		args = append(args, file)
		sourceNames = append(sourceNames, filepath.Base(file))
	}
	if len(args) < 2 {
		return nil, fmt.Errorf("可合并的 PDF 文件不足 2 个")
	}
	args = append(args, outputPath)

	cmd := exec.Command("pdfunite", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[MergePDF] pdfunite 执行失败: %v, output: %s", err, string(out))
		return nil, fmt.Errorf("PDF 合并失败: %v", err)
	}

	outputFiles := fs.ResponseFiles([]string{outputPath})

	info := fmt.Sprintf("PDF 合并完成\n输出文件: %s\n输入文件数: %d\n输入顺序: %s", outputBase, len(sourceNames), strings.Join(sourceNames, " -> "))
	if strings.TrimSpace(string(out)) != "" {
		info += "\n命令输出:\n" + strings.TrimSpace(string(out))
	}

	return &MergePDFResp{
		OutputFiles: outputFiles,
		MergeInfo:   info,
	}, nil
}

func sanitizePDFName(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	name = replacer.Replace(strings.TrimSpace(name))
	if name == "" {
		return "merged.pdf"
	}
	if filepath.Ext(name) == "" {
		name += ".pdf"
	}
	return name
}

var MergePDFTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "合并 PDF",
		Desc:     `按上传顺序把多个 PDF 文件合并成一个 PDF，适合把扫描件、分章节报告或批量导出结果合成单文件。`,
		Tags:     []string{"PDF", "合并", "文档", "Poppler"},
		Request:  &MergePDFReq{},
		Response: &MergePDFResp{},
	},
}

func init() {
	packageContext.POST("merge.form", MergePDF, MergePDFTemplate)
}
