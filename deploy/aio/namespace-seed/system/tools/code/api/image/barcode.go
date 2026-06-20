package image

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	pythonRuntime "github.com/kageos/kageos/sdk/agent-app/runtime/python"
)

type BarcodeReq struct {
	Content        string  `json:"content" widget:"name:条码内容;type:input;placeholder:Code128 支持普通文本，EAN/UPC/ISBN 需要数字" validate:"required"`
	BarcodeType    string  `json:"barcode_type" widget:"name:条码类型;type:select;options:Code128,EAN13,EAN8,UPC-A,ISBN13;options_colors:409EFF,67C23A,909399,E6A23C,909399;render_default:Code128" validate:"required,oneof=Code128 EAN13 EAN8 UPC-A ISBN13"`
	OutputFileName string  `json:"output_file_name" widget:"name:输出文件名;type:input;render_default:barcode;placeholder:不用写扩展名"`
	OutputFormat   string  `json:"output_format" widget:"name:输出格式;type:select;options:png,svg;options_colors:409EFF,67C23A;render_default:png" validate:"required,oneof=png svg"`
	ShowText       bool    `json:"show_text" widget:"name:显示条码文本;type:switch;render_default:true"`
	ModuleWidth    float64 `json:"module_width" widget:"name:条宽;type:float;min:0.1;max:2;step:0.1;unit:mm;render_default:0.3" validate:"min=0,max=2"`
	ModuleHeight   float64 `json:"module_height" widget:"name:条高;type:float;min:5;max:80;step:1;unit:mm;render_default:15" validate:"min=0,max=80"`
}

type BarcodeResp struct {
	OutputFile  string `json:"output_file" widget:"name:条形码图片;type:files"`
	BarcodeInfo string `json:"barcode_info" widget:"name:生成信息;type:text_area"`
}

func Barcode(ctx *app.Context, resp response.Response) error {
	var req BarcodeReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoBarcode(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoBarcode(ctx *app.Context, req *BarcodeReq) (*BarcodeResp, error) {
	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()
	format := strings.ToLower(strings.TrimSpace(req.OutputFormat))
	if format == "" {
		format = "png"
	}
	baseName := sanitizeFileName(strings.TrimSuffix(strings.TrimSpace(req.OutputFileName), filepath.Ext(req.OutputFileName)), "barcode")
	outputPath := filepath.Join(outputDir, baseName+"."+format)
	moduleWidth := req.ModuleWidth
	if moduleWidth <= 0 {
		moduleWidth = 0.3
	}
	moduleHeight := req.ModuleHeight
	if moduleHeight <= 0 {
		moduleHeight = 15
	}

	executor := pythonRuntime.NewExecutor(barcodePythonCode()).
		WithRequest(map[string]interface{}{
			"content":       req.Content,
			"barcode_type":  req.BarcodeType,
			"output_path":   outputPath,
			"output_format": format,
			"show_text":     req.ShowText,
			"module_width":  moduleWidth,
			"module_height": moduleHeight,
		}).
		WithOutputDir(outputDir).
		WithTimeout(30 * time.Second)
	defer func() { _ = executor.Close() }()

	var result struct {
		FileName string `json:"file_name"`
		FullCode string `json:"full_code"`
	}
	if err := executor.ExecuteJSON(ctx, &result); err != nil {
		return nil, fmt.Errorf("生成条形码失败: %w", err)
	}
	return &BarcodeResp{
		OutputFile:  fs.ResponseFiles([]string{outputPath}),
		BarcodeInfo: fmt.Sprintf("条形码生成完成\n输出文件: %s\n类型: %s\n格式: %s\n编码内容: %s\n完整条码: %s", filepath.Base(outputPath), req.BarcodeType, format, req.Content, result.FullCode),
	}, nil
}

func barcodePythonCode() string {
	return `import os
import shutil
import barcode
from barcode.writer import ImageWriter

TYPE_MAP = {
    "Code128": "code128",
    "EAN13": "ean13",
    "EAN8": "ean8",
    "UPC-A": "upca",
    "ISBN13": "isbn13",
}

def kageos_entry(args, output_dir):
    content = str(args["content"]).strip()
    kind_label = args.get("barcode_type") or "Code128"
    kind = TYPE_MAP.get(kind_label, "code128")
    output_path = args["output_path"]
    output_format = (args.get("output_format") or "png").lower()
    base_without_ext = os.path.splitext(output_path)[0]
    os.makedirs(os.path.dirname(output_path), exist_ok=True)

    cls = barcode.get_barcode_class(kind)
    writer = ImageWriter() if output_format == "png" else None
    options = {
        "write_text": bool(args.get("show_text", True)),
        "module_width": float(args.get("module_width") or 0.3),
        "module_height": float(args.get("module_height") or 15),
        "quiet_zone": 6.5,
    }
    obj = cls(content, writer=writer)
    actual_path = obj.save(base_without_ext, options=options)
    if actual_path != output_path:
        shutil.move(actual_path, output_path)
    return {
        "data": {
            "file_name": os.path.basename(output_path),
            "full_code": getattr(obj, "get_fullcode", lambda: content)(),
        }
    }`
}

var BarcodeTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "生成条形码",
		Desc:     `根据文本或数字生成 PNG/SVG 条形码。Code128 适合普通文本和资产编号，EAN/UPC/ISBN 适合标准商品码、书号等数字编码。`,
		Tags:     []string{"条形码", "Barcode", "图片", "资产编号", "商品码", "python-barcode"},
		Request:  &BarcodeReq{},
		Response: &BarcodeResp{},
	},
}

func init() {
	packageContext.POST("barcode.form", Barcode, BarcodeTemplate)
}
