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

type RotateReorderPDFReq struct {
	InputFiles  string `json:"input_files" widget:"name:上传 PDF 文件;type:files;accept:.pdf,application/pdf;max_size:500MB;max_count:20" validate:"required"`
	Operation   string `json:"operation" widget:"name:处理方式;type:select;options:旋转页面,重排/删除页面;options_colors:409EFF,E6A23C;render_default:旋转页面" validate:"required,oneof=旋转页面 重排/删除页面"`
	RotateAngle string `json:"rotate_angle" widget:"name:旋转角度;type:select;options:90,180,270,-90;options_colors:409EFF,67C23A,E6A23C,909399;render_default:90;desc:旋转页面模式显示并生效" validate:"required_if=Operation 旋转页面,omitempty,oneof=90 180 270 -90"`
	RotatePages string `json:"rotate_pages" widget:"name:旋转页码范围;type:input;render_default:1-z;placeholder:例如 1-z、1-3,5,z;desc:旋转页面模式显示并生效" validate:"required_if=Operation 旋转页面"`
	PageRange   string `json:"page_range" widget:"name:保留/重排页码范围;type:input;placeholder:例如 1-3,5,z 或 z-1 倒序;desc:重排/删除页面模式显示并生效" validate:"required_if=Operation 重排/删除页面"`
}

type RotateReorderPDFResp struct {
	OutputFiles string `json:"output_files" widget:"name:处理后的 PDF;type:files"`
	RunInfo     string `json:"run_info" widget:"name:处理信息;type:text_area"`
}

func RotateReorderPDF(ctx *app.Context, resp response.Response) error {
	var req RotateReorderPDFReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoRotateReorderPDF(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoRotateReorderPDF(ctx *app.Context, req *RotateReorderPDFReq) (*RotateReorderPDFResp, error) {
	if err := ensureQPDF(); err != nil {
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

		suffix := "_rotated"
		if req.Operation == "重排/删除页面" {
			suffix = "_reordered"
		}
		outputName := outputPDFName(filepath.Base(file), file, suffix, seenNames)
		outputPath := filepath.Join(outputDir, outputName)
		args, detail, err := buildRotateReorderArgs(req, file, outputPath)
		if err != nil {
			return nil, err
		}
		out, err := exec.Command("qpdf", args...).CombinedOutput()
		if err != nil {
			logger.Errorf(ctx, "[PDF/RotateReorder] qpdf 失败 %s: %v, output: %s", filepath.Base(file), err, string(out))
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: %v\n%s", filepath.Base(file), err, strings.TrimSpace(string(out))))
			continue
		}
		outputPaths = append(outputPaths, outputPath)
		successCount++
		infos = append(infos, fmt.Sprintf("成功 %s -> %s（%s）", filepath.Base(file), outputName, detail))
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功处理的 PDF 文件\n%s", strings.Join(infos, "\n"))
	}

	info := fmt.Sprintf("PDF 页面处理完成\n处理方式: %s\n成功: %d\n失败: %d", req.Operation, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &RotateReorderPDFResp{OutputFiles: fs.ResponseFiles(outputPaths), RunInfo: info}, nil
}

func buildRotateReorderArgs(req *RotateReorderPDFReq, inputPath, outputPath string) ([]string, string, error) {
	switch req.Operation {
	case "旋转页面":
		angle := normalizeRotateAngle(req.RotateAngle)
		if angle == "" {
			return nil, "", fmt.Errorf("旋转角度仅支持 90、180、270、-90")
		}
		pageRange, err := normalizePageRangeOrAll(req.RotatePages)
		if err != nil {
			return nil, "", err
		}
		rotateArg := fmt.Sprintf("--rotate=%s:%s", angle, pageRange)
		return []string{rotateArg, inputPath, outputPath}, fmt.Sprintf("旋转 %s 度，页码 %s", angle, pageRange), nil
	case "重排/删除页面":
		pageRange, err := normalizePageRange(req.PageRange)
		if err != nil {
			return nil, "", err
		}
		return []string{inputPath, "--pages", ".", pageRange, "--", outputPath}, "页码范围 " + pageRange, nil
	default:
		return nil, "", fmt.Errorf("不支持的处理方式: %s", req.Operation)
	}
}

func normalizeRotateAngle(angle string) string {
	switch strings.TrimSpace(angle) {
	case "90":
		return "+90"
	case "180":
		return "+180"
	case "270":
		return "+270"
	case "-90":
		return "-90"
	default:
		return ""
	}
}

func normalizePageRangeOrAll(pageRange string) (string, error) {
	pageRange = strings.TrimSpace(pageRange)
	if pageRange == "" {
		return "1-z", nil
	}
	return normalizePageRange(pageRange)
}

var RotateReorderPDFTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "PDF 旋转/重排页面",
		Desc:     `使用 qpdf 旋转 PDF 页面，或按页码范围重排、倒序、删除页面。页码支持 1-3,5,z、z-1 等 qpdf 语法。`,
		Tags:     []string{"PDF", "旋转", "重排", "删除页面", "qpdf", "页面处理"},
		Request:  &RotateReorderPDFReq{},
		Response: &RotateReorderPDFResp{},
	},
}

func init() {
	packageContext.POST("rotate_reorder.form", RotateReorderPDF, RotateReorderPDFTemplate)
}
