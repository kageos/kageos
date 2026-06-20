package document

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type OfficeToPDFReq struct {
	InputFiles string `json:"input_files" widget:"name:上传办公文档;type:files;accept:.doc,.docx,.odt,.rtf,.ppt,.pptx,.pptm,.odp,.xls,.xlsx,.ods,.csv,.txt,.html,.htm;max_size:500MB;max_count:50" validate:"required"`
}

type OfficeToPDFResp struct {
	OutputFiles string `json:"output_files" widget:"name:输出 PDF;type:files"`
	ConvertInfo string `json:"convert_info" widget:"name:转换信息;type:text_area"`
}

func OfficeToPDF(ctx *app.Context, resp response.Response) error {
	var req OfficeToPDFReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoOfficeToPDF(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoOfficeToPDF(ctx *app.Context, req *OfficeToPDFReq) (*OfficeToPDFResp, error) {
	if _, err := findLibreOfficeBin(); err != nil {
		return nil, err
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	outputDir := fs.GetTraceOutputDir()
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
		outputPath, outputName, err := convertOfficeToPDF(ctx, file, filepath.Base(file), outputDir, "_pdf", seenNames)
		if err != nil {
			logger.Warnf(ctx, "[Document/OfficeToPDF] 转换失败 %s: %v", filepath.Base(file), err)
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v", filepath.Base(file), err))
			continue
		}
		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(file), outputName))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功转换的 PDF 文件\n%s", strings.Join(infos, "\n"))
	}

	info := fmt.Sprintf("办公文档转 PDF 完成\n成功: %d\n失败: %d", successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &OfficeToPDFResp{OutputFiles: fs.ResponseFiles(outputPaths), ConvertInfo: info}, nil
}

var OfficeToPDFTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "办公文档转 PDF",
		Desc:     `使用 LibreOffice 将 Word、PPT、Excel、ODF、HTML、文本等办公文档批量转换为 PDF。适合资料预览、报告交付和统一归档。`,
		Tags:     []string{"文档", "PDF", "Word", "PPT", "Excel", "LibreOffice", "格式转换"},
		Request:  &OfficeToPDFReq{},
		Response: &OfficeToPDFResp{},
	},
}

func init() {
	packageContext.POST("office_to_pdf.form", OfficeToPDF, OfficeToPDFTemplate)
}
