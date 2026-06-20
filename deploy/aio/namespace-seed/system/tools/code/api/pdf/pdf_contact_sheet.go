package pdf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type PDFContactSheetReq struct {
	InputFiles     string `json:"input_files" widget:"name:上传 PDF 文件;type:files;accept:.pdf,application/pdf;max_size:500MB;max_count:20" validate:"required"`
	FirstPage      int    `json:"first_page" widget:"name:开始页;type:integer;min:0;render_default:0;placeholder:0 表示从第一页开始" validate:"min=0"`
	LastPage       int    `json:"last_page" widget:"name:结束页;type:integer;min:0;render_default:0;placeholder:0 表示到最后一页" validate:"min=0"`
	DPI            int    `json:"dpi" widget:"name:渲染 DPI;type:integer;min:36;max:300;render_default:100" validate:"min=0,max=300"`
	ThumbnailWidth int    `json:"thumbnail_width" widget:"name:单页缩略图宽度;type:integer;min:80;max:1200;render_default:220" validate:"min=0,max=1200"`
	Columns        int    `json:"columns" widget:"name:每行列数;type:integer;min:1;max:12;render_default:4" validate:"min=0,max=12"`
	OutputFileName string `json:"output_file_name" widget:"name:输出文件名;type:input;placeholder:可选，仅单文件时生效，例如 pdf-overview.jpg"`
}

type PDFContactSheetResp struct {
	OutputFiles string `json:"output_files" widget:"name:PDF 页面总览图;type:files"`
	SheetInfo   string `json:"sheet_info" widget:"name:生成信息;type:text_area"`
}

func PDFContactSheet(ctx *app.Context, resp response.Response) error {
	var req PDFContactSheetReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoPDFContactSheet(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoPDFContactSheet(ctx *app.Context, req *PDFContactSheetReq) (*PDFContactSheetResp, error) {
	if err := ensureCommand("pdftoppm", "poppler-utils"); err != nil {
		return nil, err
	}
	if err := ensureCommand("montage", "ImageMagick"); err != nil {
		return nil, err
	}
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	dpi := req.DPI
	if dpi <= 0 {
		dpi = 100
	}
	if dpi > 300 {
		dpi = 300
	}
	width := req.ThumbnailWidth
	if width <= 0 {
		width = 220
	}
	columns := req.Columns
	if columns <= 0 {
		columns = 4
	}

	outputDir := fs.GetTraceOutputDir()
	tempDir := filepath.Join(outputDir, "pdf_contact_sheet_tmp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	var outputPaths []string
	var infos []string
	seen := make(map[string]int)
	for _, input := range inputFiles {
		if input == "" {
			continue
		}
		stem := outputStem(filepath.Base(input), input, "_overview", seen)
		prefix := filepath.Join(tempDir, stem)
		args := []string{"-png", "-r", strconv.Itoa(dpi), "-scale-to", strconv.Itoa(width)}
		var err error
		args, err = appendPageArgs(args, req.FirstPage, req.LastPage)
		if err != nil {
			return nil, err
		}
		args = append(args, input, prefix)
		out, err := exec.Command("pdftoppm", args...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[PDFContactSheet] pdftoppm 失败 %s: %v, output: %s", filepath.Base(input), err, string(out))
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(input), err, strings.TrimSpace(string(out))))
			continue
		}
		pages, err := collectOutputFiles(tempDir, filepath.Base(prefix)+"-", map[string]bool{"png": true})
		if err != nil {
			infos = append(infos, fmt.Sprintf("失败 %s: 查找页面图片失败: %v", filepath.Base(input), err))
			continue
		}
		if len(pages) == 0 {
			infos = append(infos, fmt.Sprintf("失败 %s: 未生成页面图片", filepath.Base(input)))
			continue
		}

		outputName := stem + ".jpg"
		if len(inputFiles) == 1 && strings.TrimSpace(req.OutputFileName) != "" {
			outputName = sanitizeFileName(strings.TrimSuffix(req.OutputFileName, filepath.Ext(req.OutputFileName)), "pdf-overview") + ".jpg"
		}
		outputPath := filepath.Join(outputDir, outputName)
		montageArgs := append([]string{}, pages...)
		montageArgs = append(montageArgs, "-tile", strconv.Itoa(columns)+"x", "-geometry", "+8+8", "-background", "#f8fafc", outputPath)
		out, err = exec.Command("montage", montageArgs...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[PDFContactSheet] montage 失败 %s: %v, output: %s", filepath.Base(input), err, string(out))
			infos = append(infos, fmt.Sprintf("失败 %s: montage 失败: %v\n%s", filepath.Base(input), err, strings.TrimSpace(string(out))))
			continue
		}
		outputPaths = append(outputPaths, outputPath)
		infos = append(infos, fmt.Sprintf("成功 %s -> %s（%d 页）", filepath.Base(input), outputName, len(pages)))
	}
	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功生成的 PDF 页面总览图\n%s", strings.Join(infos, "\n"))
	}
	return &PDFContactSheetResp{
		OutputFiles: fs.ResponseFiles(outputPaths),
		SheetInfo:   fmt.Sprintf("PDF 页面总览生成完成\nDPI: %d\n单页宽度: %d\n每行列数: %d\n输出文件数: %d\n\n详情:\n%s", dpi, width, columns, len(outputPaths), strings.Join(infos, "\n")),
	}, nil
}

var PDFContactSheetTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "PDF 页面总览图",
		Desc:     `使用 Poppler 将 PDF 页面渲染为缩略图，并用 ImageMagick 拼成一张页面总览图。适合快速预览长 PDF、扫描件、合同、报告或论文结构。`,
		Tags:     []string{"PDF", "页面预览", "缩略图", "Poppler", "pdftoppm", "montage"},
		Request:  &PDFContactSheetReq{},
		Response: &PDFContactSheetResp{},
	},
}

func init() {
	packageContext.POST("contact_sheet.form", PDFContactSheet, PDFContactSheetTemplate)
}
