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

type ExtractPDFImagesReq struct {
	InputFiles string `json:"input_files" widget:"name:上传 PDF 文件;type:files;accept:.pdf,application/pdf;max_size:200MB;max_count:20" validate:"required"`
	OutputMode string `json:"output_mode" widget:"name:输出模式;type:select;options:保留原始格式,转为PNG;options_colors:409EFF,67C23A;render_default:保留原始格式" validate:"required,oneof=保留原始格式 转为PNG"`
	FirstPage  int    `json:"first_page" widget:"name:开始页;type:integer;min:0;render_default:0;placeholder:0 表示从第一页开始" validate:"min=0"`
	LastPage   int    `json:"last_page" widget:"name:结束页;type:integer;min:0;render_default:0;placeholder:0 表示到最后一页" validate:"min=0"`
}

type ExtractPDFImagesResp struct {
	OutputFiles string `json:"output_files" widget:"name:提取出的图片;type:files"`
	ExtractInfo string `json:"extract_info" widget:"name:提取信息;type:text_area"`
}

func ExtractPDFImages(ctx *app.Context, resp response.Response) error {
	var req ExtractPDFImagesReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoExtractPDFImages(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoExtractPDFImages(ctx *app.Context, req *ExtractPDFImagesReq) (*ExtractPDFImagesResp, error) {
	if err := ensureCommand("pdfimages", "poppler-utils"); err != nil {
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
	seenStems := make(map[string]int)
	var outputPaths []string
	var infos []string
	successCount := 0
	failCount := 0

	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[Poppler/ExtractPDFImages] 文件 %s 无本地路径，跳过", filepath.Base(file))
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		stem := outputStem(filepath.Base(file), file, "_image", seenStems)
		rootPath := filepath.Join(outputDir, stem)
		args := []string{}
		if req.OutputMode == "转为PNG" {
			args = append(args, "-png")
		} else {
			args = append(args, "-all")
		}
		args = append(args, "-p")
		var err error
		args, err = appendPageArgs(args, req.FirstPage, req.LastPage)
		if err != nil {
			return nil, err
		}
		args = append(args, file, rootPath)
		out, err := exec.Command("pdfimages", args...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[Poppler/ExtractPDFImages] 提取失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}

		paths, err := collectOutputFiles(outputDir, filepath.Base(rootPath)+"-", nil)
		if err != nil {
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: 查找输出图片失败: %v", filepath.Base(file), err))
			continue
		}
		if len(paths) == 0 {
			failCount++
			infos = append(infos, fmt.Sprintf("未提取 %s: PDF 中没有可提取的内嵌图片", filepath.Base(file)))
			continue
		}

		outputPaths = append(outputPaths, paths...)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %d 张内嵌图片", filepath.Base(file), len(paths)))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功提取的 PDF 内嵌图片\n%s", strings.Join(infos, "\n"))
	}

	outputFiles := fs.ResponseFiles(outputPaths)

	info := fmt.Sprintf("PDF 内嵌图片提取完成\n输出模式: %s\n成功输入文件: %d\n失败/无图片输入文件: %d\n输出图片数: %d", req.OutputMode, successCount, failCount, len(outputPaths))
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &ExtractPDFImagesResp{OutputFiles: outputFiles, ExtractInfo: info}, nil
}

var ExtractPDFImagesTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "提取 PDF 内嵌图片",
		Desc:     `使用 pdfimages 提取 PDF 中真实嵌入的图片资源，不是把整页渲染成截图。适合提取论文、报告、扫描件中的原始图片素材；如果要把每一页变成图片，请使用 PDF 页面转图片。`,
		Tags:     []string{"PDF", "提取图片", "Poppler", "pdfimages", "素材提取"},
		Request:  &ExtractPDFImagesReq{},
		Response: &ExtractPDFImagesResp{},
	},
}

func init() {
	packageContext.POST("extract_images.form", ExtractPDFImages, ExtractPDFImagesTemplate)
}
