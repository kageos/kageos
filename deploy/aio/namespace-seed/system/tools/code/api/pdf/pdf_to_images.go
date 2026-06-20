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

type PDFToImagesReq struct {
	InputFiles   string `json:"input_files" widget:"name:上传 PDF 文件;type:files;accept:.pdf,application/pdf;max_size:200MB;max_count:20" validate:"required"`
	OutputFormat string `json:"output_format" widget:"name:输出格式;type:select;options:png,jpeg,tiff;options_colors:67C23A,409EFF,E6A23C;render_default:png" validate:"required,oneof=png jpeg tiff"`
	DPI          int    `json:"dpi" widget:"name:分辨率 DPI;type:integer;min:36;max:600;render_default:150" validate:"min=0,max=600"`
	FirstPage    int    `json:"first_page" widget:"name:开始页;type:integer;min:0;render_default:0;placeholder:0 表示从第一页开始" validate:"min=0"`
	LastPage     int    `json:"last_page" widget:"name:结束页;type:integer;min:0;render_default:0;placeholder:0 表示到最后一页" validate:"min=0"`
	OutputPrefix string `json:"output_prefix" widget:"name:输出文件名前缀;type:input;placeholder:可选，仅单文件时生效，例如 page"`
}

type PDFToImagesResp struct {
	OutputFiles string `json:"output_files" widget:"name:页面图片;type:files"`
	ConvertInfo string `json:"convert_info" widget:"name:转换信息;type:text_area"`
}

func PDFToImages(ctx *app.Context, resp response.Response) error {
	var req PDFToImagesReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoPDFToImages(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoPDFToImages(ctx *app.Context, req *PDFToImagesReq) (*PDFToImagesResp, error) {
	if err := ensureCommand("pdftoppm", "poppler-utils"); err != nil {
		return nil, err
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	formatFlag, ext := normalizeImageFormat(req.OutputFormat)
	dpi := normalizeDPI(req.DPI)
	outputDir := fs.GetTraceOutputDir()
	seenStems := make(map[string]int)
	var outputPaths []string
	var infos []string
	successCount := 0
	failCount := 0

	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[Poppler/PDFToImages] 文件 %s 无本地路径，跳过", filepath.Base(file))
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		stem := outputStem(filepath.Base(file), file, "_page", seenStems)
		if len(files) == 1 && strings.TrimSpace(req.OutputPrefix) != "" {
			stem = sanitizeFileName(req.OutputPrefix, "page")
		}
		prefixPath := filepath.Join(outputDir, stem)
		args := []string{formatFlag, "-r", strconv.Itoa(dpi)}
		var err error
		args, err = appendPageArgs(args, req.FirstPage, req.LastPage)
		if err != nil {
			return nil, err
		}
		args = append(args, file, prefixPath)
		out, err := exec.Command("pdftoppm", args...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[Poppler/PDFToImages] 转换失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		paths, err := collectOutputFiles(outputDir, filepath.Base(prefixPath)+"-", map[string]bool{ext: true})
		if err != nil {
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: 查找输出图片失败: %v", filepath.Base(file), err))
			continue
		}
		if len(paths) == 0 {
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: pdftoppm 执行成功但未找到输出图片", filepath.Base(file)))
			continue
		}

		outputPaths = append(outputPaths, paths...)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %d 张图片", filepath.Base(file), len(paths)))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功转换的页面图片\n%s", strings.Join(infos, "\n"))
	}

	outputFiles := fs.ResponseFiles(outputPaths)

	info := fmt.Sprintf("PDF 页面转图片完成\n输出格式: %s\nDPI: %d\n成功输入文件: %d\n失败输入文件: %d\n输出图片数: %d", ext, dpi, successCount, failCount, len(outputPaths))
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &PDFToImagesResp{OutputFiles: outputFiles, ConvertInfo: info}, nil
}

var PDFToImagesTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "PDF 页面转图片",
		Desc:     `使用 pdftoppm 将 PDF 页面渲染为 PNG/JPEG/TIFF 图片，适合生成预览图、缩略图、页面截图或为后续 OCR/图片处理准备输入。`,
		Tags:     []string{"PDF", "转图片", "页面渲染", "Poppler", "pdftoppm"},
		Request:  &PDFToImagesReq{},
		Response: &PDFToImagesResp{},
	},
}

func init() {
	packageContext.POST("render_pages.form", PDFToImages, PDFToImagesTemplate)
}
