package image

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	pythonRuntime "github.com/kageos/kageos-sdk/agent-app/runtime/python"
)

type QRCodeReq struct {
	Content         string `json:"content" widget:"name:二维码内容;type:text_area;placeholder:可以是 URL、文本、JSON、联系方式等" validate:"required"`
	OutputFileName  string `json:"output_file_name" widget:"name:输出文件名;type:input;render_default:qrcode;placeholder:不用写扩展名"`
	OutputFormat    string `json:"output_format" widget:"name:输出格式;type:select;options:png,svg;options_colors:409EFF,67C23A;render_default:png" validate:"required,oneof=png svg"`
	ErrorCorrection string `json:"error_correction" widget:"name:容错等级;type:select;options:低,中,较高,高;options_colors:909399,409EFF,E6A23C,67C23A;render_default:中" validate:"required,oneof=低 中 较高 高"`
	BoxSize         int    `json:"box_size" widget:"name:模块大小;type:integer;min:2;max:40;render_default:10;unit:px" validate:"min=0,max=40"`
	Border          int    `json:"border" widget:"name:边框模块数;type:integer;min:0;max:20;render_default:4" validate:"min=0,max=20"`
	FillColor       string `json:"fill_color" widget:"name:前景色;type:select;options:黑色,蓝色,绿色,红色;options_colors:909399,409EFF,67C23A,F56C6C;render_default:黑色" validate:"required,oneof=黑色 蓝色 绿色 红色"`
	BackColor       string `json:"back_color" widget:"name:背景色;type:select;options:白色,透明,浅灰;options_colors:909399,909399,909399;render_default:白色" validate:"required,oneof=白色 透明 浅灰"`
}

type QRCodeResp struct {
	OutputFile string `json:"output_file" widget:"name:二维码图片;type:files"`
	QRInfo     string `json:"qr_info" widget:"name:生成信息;type:text_area"`
}

func QRCode(ctx *app.Context, resp response.Response) error {
	var req QRCodeReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoQRCode(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoQRCode(ctx *app.Context, req *QRCodeReq) (*QRCodeResp, error) {
	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()
	format := strings.ToLower(strings.TrimSpace(req.OutputFormat))
	if format == "" {
		format = "png"
	}
	baseName := sanitizeFileName(strings.TrimSuffix(strings.TrimSpace(req.OutputFileName), filepath.Ext(req.OutputFileName)), "qrcode")
	outputPath := filepath.Join(outputDir, baseName+"."+format)

	boxSize := req.BoxSize
	if boxSize <= 0 {
		boxSize = 10
	}
	border := req.Border
	if border < 0 {
		border = 4
	}
	executor := pythonRuntime.NewExecutor(qrcodePythonCode()).
		WithRequest(map[string]interface{}{
			"content":          req.Content,
			"output_path":      outputPath,
			"output_format":    format,
			"error_correction": req.ErrorCorrection,
			"box_size":         boxSize,
			"border":           border,
			"fill_color":       req.FillColor,
			"back_color":       req.BackColor,
		}).
		WithOutputDir(outputDir).
		WithTimeout(30 * time.Second)
	defer func() { _ = executor.Close() }()

	var result struct {
		FileName string `json:"file_name"`
		Size     string `json:"size"`
	}
	if err := executor.ExecuteJSON(ctx, &result); err != nil {
		return nil, fmt.Errorf("生成二维码失败: %w", err)
	}

	return &QRCodeResp{
		OutputFile: fs.ResponseFiles([]string{outputPath}),
		QRInfo:     fmt.Sprintf("二维码生成完成\n输出文件: %s\n格式: %s\n容错等级: %s\n模块大小: %d\n边框: %d\n图片尺寸: %s", filepath.Base(outputPath), format, req.ErrorCorrection, boxSize, border, result.Size),
	}, nil
}

func qrcodePythonCode() string {
	return `import os
import qrcode
import qrcode.image.svg

ERROR_MAP = {
    "低": qrcode.constants.ERROR_CORRECT_L,
    "中": qrcode.constants.ERROR_CORRECT_M,
    "较高": qrcode.constants.ERROR_CORRECT_Q,
    "高": qrcode.constants.ERROR_CORRECT_H,
}
COLOR_MAP = {
    "黑色": "#111827",
    "蓝色": "#2563eb",
    "绿色": "#16a34a",
    "红色": "#dc2626",
    "白色": "#ffffff",
    "浅灰": "#f3f4f6",
    "透明": "transparent",
}

def kageos_entry(args, output_dir):
    content = args["content"]
    output_path = args["output_path"]
    output_format = (args.get("output_format") or "png").lower()
    box_size = int(args.get("box_size") or 10)
    border = int(args.get("border") or 4)
    err = ERROR_MAP.get(args.get("error_correction"), qrcode.constants.ERROR_CORRECT_M)
    fill = COLOR_MAP.get(args.get("fill_color"), "#111827")
    back = COLOR_MAP.get(args.get("back_color"), "#ffffff")

    os.makedirs(os.path.dirname(output_path), exist_ok=True)
    image_factory = qrcode.image.svg.SvgPathImage if output_format == "svg" else None
    qr = qrcode.QRCode(error_correction=err, box_size=box_size, border=border, image_factory=image_factory)
    qr.add_data(content)
    qr.make(fit=True)
    img = qr.make_image(fill_color=fill, back_color=back)
    if output_format != "svg":
        img = img.convert("RGBA" if back == "transparent" else "RGB")
    img.save(output_path)
    size = ""
    if hasattr(img, "size"):
        size = f"{img.size[0]}x{img.size[1]}"
    return {"data": {"file_name": os.path.basename(output_path), "size": size}}`
}

var QRCodeTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "生成二维码",
		Desc:     `根据文本、URL、JSON 或联系方式生成 PNG/SVG 二维码，支持容错等级、颜色、边框和模块大小设置。适合资料链接、活动码、文件交付入口和工作台分享。`,
		Tags:     []string{"二维码", "QR Code", "图片", "链接", "qrcode"},
		Request:  &QRCodeReq{},
		Response: &QRCodeResp{},
	},
}

func init() {
	packageContext.POST("qrcode.form", QRCode, QRCodeTemplate)
}
