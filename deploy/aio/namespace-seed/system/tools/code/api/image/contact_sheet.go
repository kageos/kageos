package image

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

type ContactSheetReq struct {
	InputFiles      string `json:"input_files" widget:"name:上传图片;type:files;accept:image/*;max_size:500MB;max_count:100" validate:"required"`
	Columns         int    `json:"columns" widget:"name:每行列数;type:integer;min:1;max:20;render_default:5" validate:"min=0,max=20"`
	ThumbnailWidth  int    `json:"thumbnail_width" widget:"name:单图宽度;type:integer;min:80;max:2048;render_default:240" validate:"min=0,max=2048"`
	IncludeLabels   bool   `json:"include_labels" widget:"name:显示文件名;type:switch;render_default:true"`
	BackgroundColor string `json:"background_color" widget:"name:背景色;type:input;placeholder:例如 #111827 或 white"`
	OutputFileName  string `json:"output_file_name" widget:"name:输出文件名;type:input;placeholder:可选，例如 contact-sheet.jpg"`
}

type ContactSheetResp struct {
	OutputFiles string `json:"output_files" widget:"name:联系表图片;type:files"`
	SheetInfo   string `json:"sheet_info" widget:"name:生成信息;type:text_area"`
}

func ContactSheet(ctx *app.Context, resp response.Response) error {
	var req ContactSheetReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoContactSheet(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoContactSheet(ctx *app.Context, req *ContactSheetReq) (*ContactSheetResp, error) {
	if _, err := exec.LookPath("montage"); err != nil {
		return nil, fmt.Errorf("未找到 montage，请确认运行环境已安装 ImageMagick")
	}
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入图片")
	}

	columns := req.Columns
	if columns <= 0 {
		columns = 5
	}
	width := req.ThumbnailWidth
	if width <= 0 {
		width = 240
	}
	bg := strings.TrimSpace(req.BackgroundColor)
	if bg == "" {
		bg = "#f8fafc"
	}
	outputName := strings.TrimSpace(req.OutputFileName)
	if outputName == "" {
		outputName = "contact_sheet.jpg"
	} else {
		outputName = imEnsureExt(outputName, "jpg")
	}
	outputPath := filepath.Join(fs.GetTraceOutputDir(), outputName)

	args := []string{}
	if req.IncludeLabels {
		args = append(args, "-label", "%f")
	}
	args = append(args, inputFiles...)
	args = append(args,
		"-thumbnail", strconv.Itoa(width)+"x",
		"-tile", strconv.Itoa(columns)+"x",
		"-geometry", "+10+10",
		"-background", bg,
		"-bordercolor", bg,
		outputPath,
	)
	out, err := exec.Command("montage", args...).CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[ContactSheet] montage 失败: %v, output: %s", err, string(out))
		return nil, fmt.Errorf("生成联系表失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}

	outputFiles := fs.ResponseFiles([]string{outputPath})
	info := fmt.Sprintf("图片联系表生成完成\n输入图片数: %d\n每行列数: %d\n单图宽度: %d\n显示文件名: %t\n输出文件: %s",
		len(inputFiles), columns, width, req.IncludeLabels, outputName)
	return &ContactSheetResp{OutputFiles: outputFiles, SheetInfo: info}, nil
}

var ContactSheetTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "多图联系表",
		Desc:     `使用 ImageMagick montage 将多张图片拼成一张联系表/索引图，可显示文件名。适合批量图片处理后的快速验收、素材总览、缩略图报告。`,
		Tags:     []string{"图片", "拼图", "联系表", "缩略图", "ImageMagick", "montage", "批量验收"},
		Request:  &ContactSheetReq{},
		Response: &ContactSheetResp{},
	},
}

func init() {
	packageContext.POST("contact_sheet.form", ContactSheet, ContactSheetTemplate)
}
