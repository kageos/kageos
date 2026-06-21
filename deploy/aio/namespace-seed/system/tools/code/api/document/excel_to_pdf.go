package document

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	pythonRuntime "github.com/kageos/kageos-sdk/agent-app/runtime/python"
)

type ExcelToPDFReq struct {
	InputFiles      string `json:"input_files" widget:"name:上传 Excel 文件;type:files;accept:.xlsx,.xls;max_size:200MB;max_count:20" validate:"required"`
	PageSize        string `json:"page_size" widget:"name:页面尺寸;type:select;options:A4,A3,Letter,Legal,A5,B5,自定义;options_colors:409EFF,67C23A,909399,E6A23C,F56C6C,909399,909399;render_default:A4" validate:"required,oneof=A4 A3 Letter Legal A5 B5 自定义"`
	CustomWidth     int    `json:"custom_width" widget:"name:自定义宽度;type:integer;min:50;max:1000;step:1;unit:mm;render_default:210;desc:页面尺寸选择自定义后显示并生效" validate:"required_if=PageSize 自定义,min=0,max=1000"`
	CustomHeight    int    `json:"custom_height" widget:"name:自定义高度;type:integer;min:50;max:1000;step:1;unit:mm;render_default:297;desc:页面尺寸选择自定义后显示并生效" validate:"required_if=PageSize 自定义,min=0,max=1000"`
	Orientation     string `json:"orientation" widget:"name:页面方向;type:select;options:纵向,横向;options_colors:409EFF,67C23A;render_default:纵向" validate:"required,oneof=纵向 横向"`
	Margins         string `json:"margins" widget:"name:边距(mm);type:input;render_default:20,20,20,20;placeholder:上,右,下,左，逗号分隔"`
	ScalePercentage int    `json:"scale_percentage" widget:"name:缩放比例;type:slider;render_default:100;min:50;max:200;step:1;unit:%" validate:"min=0,max=200"`
}

type ExcelToPDFResp struct {
	OutputFiles string `json:"output_files" widget:"name:输出 PDF;type:files"`
	ConvertInfo string `json:"convert_info" widget:"name:转换信息;type:text_area"`
}

func ExcelToPDF(ctx *app.Context, resp response.Response) error {
	var req ExcelToPDFReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoExcelToPDF(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoExcelToPDF(ctx *app.Context, req *ExcelToPDFReq) (*ExcelToPDFResp, error) {
	if _, err := findLibreOfficeBin(); err != nil {
		return nil, err
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	margins := parseExcelMargins(req.Margins)
	width, height := excelPageSize(req.PageSize, req.CustomWidth, req.CustomHeight, req.Orientation)
	scale := req.ScalePercentage
	if scale <= 0 {
		scale = 100
	}
	if scale < 50 {
		scale = 50
	}
	if scale > 200 {
		scale = 200
	}

	outputDir := fs.GetTraceOutputDir()
	tempDir, err := os.MkdirTemp(outputDir, "excel_pdf_*")
	if err != nil {
		return nil, fmt.Errorf("创建 Excel PDF 临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

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
		ext := strings.ToLower(filepath.Ext(file))
		if ext != ".xlsx" && ext != ".xls" {
			failCount++
			infos = append(infos, fmt.Sprintf("跳过 %s: 不是 Excel 文件", filepath.Base(file)))
			continue
		}

		adjustedPath := filepath.Join(tempDir, "print_"+filepath.Base(file))
		warning, err := prepareExcelPageSetup(ctx, file, adjustedPath, req, width, height, margins, scale, tempDir)
		if err != nil {
			logger.Warnf(ctx, "[Document/ExcelToPDF] 页面设置失败 %s: %v", filepath.Base(file), err)
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: 页面设置失败: %v", filepath.Base(file), err))
			continue
		}
		outputPath, outputName, err := convertOfficeToPDF(ctx, adjustedPath, filepath.Base(file), outputDir, "_pdf", seenNames)
		if err != nil {
			logger.Warnf(ctx, "[Document/ExcelToPDF] 转 PDF 失败 %s: %v", filepath.Base(file), err)
			failCount++
			infos = append(infos, fmt.Sprintf("失败 %s: PDF 转换失败: %v", filepath.Base(file), err))
			continue
		}
		outputPaths = append(outputPaths, outputPath)
		successCount++
		line := fmt.Sprintf("成功 %s -> %s（%dx%dmm，%s，缩放 %d%%）", filepath.Base(file), outputName, width, height, req.Orientation, scale)
		if warning != "" {
			line += "\n" + warning
		}
		infos = append(infos, line)
	}

	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功转换的 Excel PDF\n%s", strings.Join(infos, "\n"))
	}

	info := fmt.Sprintf("Excel 转 PDF 完成\n页面尺寸: %s\n方向: %s\n缩放: %d%%\n成功: %d\n失败: %d", req.PageSize, req.Orientation, scale, successCount, failCount)
	if len(infos) > 0 {
		info += "\n\n详情:\n" + strings.Join(infos, "\n")
	}
	return &ExcelToPDFResp{OutputFiles: fs.ResponseFiles(outputPaths), ConvertInfo: info}, nil
}

func prepareExcelPageSetup(ctx *app.Context, inputPath, outputPath string, req *ExcelToPDFReq, width, height int, margins []float64, scale int, outputDir string) (string, error) {
	executor := pythonRuntime.NewExecutor(excelPageSetupPythonCode()).
		WithRequest(map[string]interface{}{
			"input_file":       inputPath,
			"output_file":      outputPath,
			"page_size":        req.PageSize,
			"page_width":       width,
			"page_height":      height,
			"orientation":      req.Orientation,
			"margin_top":       margins[0],
			"margin_right":     margins[1],
			"margin_bottom":    margins[2],
			"margin_left":      margins[3],
			"scale_percentage": scale,
		}).
		WithPackages("openpyxl").
		WithOutputDir(outputDir).
		WithTimeout(60 * time.Second)

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Warning string `json:"warning"`
	}
	_, err := executor.ExecuteJSONWithResult(ctx, &result)
	closeErr := executor.Close()
	if closeErr != nil {
		logger.Warnf(ctx, "[Document/ExcelToPDF] Python executor close: %v", closeErr)
	}
	if err != nil {
		return "", err
	}
	if !result.Success {
		if result.Message == "" {
			result.Message = "页面设置脚本未成功"
		}
		return "", fmt.Errorf("%s", result.Message)
	}
	if _, err := os.Stat(outputPath); err != nil {
		return "", fmt.Errorf("页面设置脚本未生成调整后的 Excel 文件")
	}
	return strings.TrimSpace(result.Warning), nil
}

func parseExcelMargins(input string) []float64 {
	margins := []float64{20, 20, 20, 20}
	parts := strings.Split(input, ",")
	if len(parts) != 4 {
		return margins
	}
	for i, part := range parts {
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err == nil && value >= 0 {
			margins[i] = value
		}
	}
	return margins
}

func excelPageSize(pageSize string, customWidth, customHeight int, orientation string) (int, int) {
	width, height := 210, 297
	switch pageSize {
	case "A3":
		width, height = 297, 420
	case "Letter":
		width, height = 216, 279
	case "Legal":
		width, height = 216, 356
	case "A5":
		width, height = 148, 210
	case "B5":
		width, height = 176, 250
	case "自定义":
		width, height = customWidth, customHeight
		if width <= 0 {
			width = 210
		}
		if height <= 0 {
			height = 297
		}
	}
	if orientation == "横向" && width < height {
		width, height = height, width
	}
	if orientation == "纵向" && width > height {
		width, height = height, width
	}
	return width, height
}

func excelPageSetupPythonCode() string {
	return `import os
from openpyxl import load_workbook
from openpyxl.worksheet.page import PageMargins

PAPER_SIZES = {
    "Letter": 1,
    "Legal": 5,
    "A3": 8,
    "A4": 9,
    "A5": 11,
    "B5": 13,
}

def kageos_entry(args, output_dir):
    input_file = args["input_file"]
    output_file = args["output_file"]
    page_size = args["page_size"]
    page_width = args["page_width"]
    page_height = args["page_height"]
    orientation = args["orientation"]
    margin_top = args["margin_top"]
    margin_right = args["margin_right"]
    margin_bottom = args["margin_bottom"]
    margin_left = args["margin_left"]
    scale_percentage = args["scale_percentage"]

    warnings = []
    workbook = load_workbook(input_file)
    page_orientation = "landscape" if orientation == "横向" else "portrait"

    for worksheet in workbook.worksheets:
        worksheet.page_setup.orientation = page_orientation
        worksheet.page_margins = PageMargins(
            top=margin_top / 25.4,
            right=margin_right / 25.4,
            bottom=margin_bottom / 25.4,
            left=margin_left / 25.4,
        )
        if scale_percentage > 0:
            worksheet.page_setup.scale = scale_percentage
            worksheet.page_setup.fitToWidth = 0
            worksheet.page_setup.fitToHeight = 0
        if page_size in PAPER_SIZES:
            worksheet.page_setup.paperSize = PAPER_SIZES[page_size]
        elif page_size == "自定义":
            warnings.append(
                f"工作表 {worksheet.title}: openpyxl 不能直接设置 {page_width}x{page_height}mm 自定义纸张，已应用方向/边距/缩放并保留默认纸张"
            )

    out_dir = os.path.dirname(output_file)
    if out_dir:
        os.makedirs(out_dir, exist_ok=True)
    workbook.save(output_file)
    return {
        "data": {
            "success": True,
            "message": f"已生成页面设置文件: {os.path.basename(output_file)}",
            "warning": "\n".join(warnings),
        }
    }`
}

var ExcelToPDFTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "Excel 转 PDF（打印页面设置）",
		Desc:     `将 Excel 文件转换为 PDF，并在转换前设置页面尺寸、方向、边距和缩放比例。适合报表、财务表、清单和需要打印交付的表格。`,
		Tags:     []string{"Excel", "PDF", "打印", "页面设置", "LibreOffice", "openpyxl"},
		Request:  &ExcelToPDFReq{},
		Response: &ExcelToPDFResp{},
	},
}

func init() {
	packageContext.POST("excel_to_pdf.form", ExcelToPDF, ExcelToPDFTemplate)
}
