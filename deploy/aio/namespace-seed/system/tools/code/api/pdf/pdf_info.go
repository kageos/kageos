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

type PDFInfoReq struct {
	InputFiles      string `json:"input_files" widget:"name:上传 PDF 文件;type:files;accept:.pdf,application/pdf;max_size:200MB;max_count:20" validate:"required"`
	IncludeBoxes    bool   `json:"include_boxes" widget:"name:包含页面边界框;type:switch;render_default:false"`
	IncludeMetadata bool   `json:"include_metadata" widget:"name:包含 XML 元数据;type:switch;render_default:false"`
}

type PDFInfoResp struct {
	InfoText string `json:"info_text" widget:"name:PDF 信息;type:text_area"`
	Summary  string `json:"summary" widget:"name:处理说明;type:text_area"`
}

func PDFInfo(ctx *app.Context, resp response.Response) error {
	var req PDFInfoReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoPDFInfo(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoPDFInfo(ctx *app.Context, req *PDFInfoReq) (*PDFInfoResp, error) {
	if err := ensureCommand("pdfinfo", "poppler-utils"); err != nil {
		return nil, err
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	var blocks []string
	var summary []string
	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[Poppler/PDFInfo] 文件 %s 无本地路径，跳过", filepath.Base(file))
			summary = append(summary, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}
		args := []string{"-isodates"}
		if req.IncludeBoxes {
			args = append(args, "-box")
		}
		if req.IncludeMetadata {
			args = append(args, "-meta")
		}
		args = append(args, file)
		out, err := exec.Command("pdfinfo", args...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[Poppler/PDFInfo] 读取失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			summary = append(summary, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}
		blocks = append(blocks, fmt.Sprintf("## %s\n%s", filepath.Base(file), strings.TrimSpace(string(out))))
		summary = append(summary, fmt.Sprintf("成功 %s", filepath.Base(file)))
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("没有成功读取的 PDF 信息\n%s", strings.Join(summary, "\n"))
	}
	return &PDFInfoResp{
		InfoText: strings.Join(blocks, "\n\n"),
		Summary:  strings.Join(summary, "\n"),
	}, nil
}

var PDFInfoTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "读取 PDF 信息",
		Desc:     `使用 pdfinfo 读取 PDF 页数、页面尺寸、作者、标题、创建时间、修改时间、是否加密等基础信息。适合在处理 PDF 前先判断文件结构和页数。`,
		Tags:     []string{"PDF", "信息", "页数", "元数据", "Poppler", "pdfinfo"},
		Request:  &PDFInfoReq{},
		Response: &PDFInfoResp{},
	},
}

func init() {
	packageContext.POST("inspect.form", PDFInfo, PDFInfoTemplate)
}
