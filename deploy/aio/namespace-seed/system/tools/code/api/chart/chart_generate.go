//<文件名>chart_generate.go</文件名>

package chart

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	pythonRuntime "github.com/kageos/kageos/sdk/agent-app/runtime/python"
)

// ChartGenerateReq 图表生成请求结构体
type ChartGenerateReq struct {
	// 框架标签：widget:"type:select;options:柱状图,折线图,散点图,饼图,直方图,箱线图" - 图表类型
	ChartType string `json:"chart_type" widget:"name:图表类型;type:select;options:柱状图,折线图,散点图,饼图,直方图,箱线图;render_default:柱状图" validate:"required"`

	// 框架标签：widget:"type:text_area;placeholder:JSON格式的数据..." - 数据（JSON格式）
	DataJSON string `json:"data_json" widget:"name:数据(JSON格式);type:text_area;placeholder:请输入JSON格式的数据，例如: {\"x\": [1,2,3,4,5], \"y\": [10,20,30,40,50]} 或 {\"labels\": [\"A\",\"B\",\"C\"], \"values\": [30,40,30]}" validate:"required"`

	// 框架标签：widget:"type:input;placeholder:图表标题" - 图表标题
	Title string `json:"title" widget:"name:图表标题;type:input;placeholder:请输入图表标题"`

	// 框架标签：widget:"type:input;placeholder:X轴标签" - X轴标签
	XLabel string `json:"x_label" widget:"name:X轴标签;type:input;placeholder:请输入X轴标签"`

	// 框架标签：widget:"type:input;placeholder:Y轴标签" - Y轴标签
	YLabel string `json:"y_label" widget:"name:Y轴标签;type:input;placeholder:请输入Y轴标签"`

	// 框架标签：widget:"type:integer;render_default:800" - 图表宽度（像素）
	Width int `json:"width" widget:"name:图表宽度(像素);type:integer;render_default:800;placeholder:请输入图表宽度"`

	// 框架标签：widget:"type:integer;render_default:600" - 图表高度（像素）
	Height int `json:"height" widget:"name:图表高度(像素);type:integer;render_default:600;placeholder:请输入图表高度"`
}

// ChartGenerateResp 图表生成响应结构体
type ChartGenerateResp struct {
	// 图表图片文件
	ChartImage string `json:"chart_image" widget:"name:图表图片;type:files"`

	// 图表信息
	ChartInfo string `json:"chart_info" widget:"name:图表信息;type:text_area"`

	// 生成状态
	Status string `json:"status" widget:"name:生成状态;type:text"`
}

// ChartGenerate 图表生成函数
//
// 错误处理说明：
// - 系统错误（Python 执行失败、系统异常等）：直接 return err，框架会记录日志并返回系统错误
// - 业务错误（数据格式错误、参数验证失败等）：使用 resp.BizErrorf().Build()，返回给用户的业务错误提示
// 区分原则：系统级异常用 return err，业务逻辑验证失败用 BizErrorf
func ChartGenerate(ctx *app.Context, resp response.Response) error {
	start := time.Now()
	logViz(ctx, "chart_generate", "begin", start)

	var req ChartGenerateReq
	err := ctx.ShouldBindValidate(&req)
	if err != nil {
		return err
	}
	logViz(ctx, "chart_generate", "bind_validate_ok", start)

	// 设置默认值
	if req.Width <= 0 {
		req.Width = 800
	}
	if req.Height <= 0 {
		req.Height = 600
	}
	if req.Title == "" {
		req.Title = "数据图表"
	}

	// 验证数据格式（业务错误：用户输入的数据格式不正确）
	var dataMap map[string]interface{}
	if err := json.Unmarshal([]byte(req.DataJSON), &dataMap); err != nil {
		// 业务错误：用户输入的 JSON 格式错误，使用 BizErrorf 返回友好的错误提示
		return resp.BizErrorf("数据格式错误，请输入有效的 JSON 格式数据: %v", err).Build()
	}
	logViz(ctx, "chart_generate", "json_ok chart_type="+req.ChartType, start)

	// 构建 Python 代码
	pythonCode := buildChartGenerateCode()

	// 创建请求结构体
	type PythonRequest struct {
		Data       map[string]interface{} `json:"data"`
		ChartType  string                 `json:"chart_type"`
		Width      int                    `json:"width"`
		Height     int                    `json:"height"`
		Title      string                 `json:"title"`
		XLabel     string                 `json:"x_label"`
		YLabel     string                 `json:"y_label"`
		OutputPath string                 `json:"output_path"`
	}

	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()
	fileName := sanitizeFileName(req.Title)
	if fileName == "" {
		fileName = "数据图表"
	}
	outputPath := filepath.Join(outputDir, fileName+".png")

	pythonReq := PythonRequest{
		Data:       dataMap,
		ChartType:  req.ChartType,
		Width:      req.Width,
		Height:     req.Height,
		Title:      req.Title,
		XLabel:     req.XLabel,
		YLabel:     req.YLabel,
		OutputPath: outputPath,
	}

	// 创建 Python 执行器（须 defer Close；超时与 Python 默认 5m 区分，避免长时间挂死）
	executor := pythonRuntime.NewExecutor(pythonCode).
		WithRequest(pythonReq).
		WithOutputDir(outputDir).
		WithTimeout(30 * time.Second)
	defer func() {
		if cerr := executor.Close(); cerr != nil {
			logger.Warnf(ctx, "[ChartGenerate] executor.Close: %v", cerr)
		}
	}()

	logViz(ctx, "chart_generate", "python_execute_start timeout=30s", start)

	// 解析 JSON 输出的结构体
	var result struct {
		ChartInfo string `json:"chart_info"`
	}

	// 使用 ExecuteJSONWithResult 解析结构化结果与 output_files
	execResult, err := executor.ExecuteJSONWithResult(ctx, &result)
	if err != nil {
		logViz(ctx, "chart_generate", "python_execute_failed", start)
		logger.Errorf(ctx, "[ChartGenerate] Python 执行失败: %v", err)
		// 系统错误：Python 执行失败属于系统级错误（可能是 Python 环境问题、代码错误等）
		// 直接返回 error，框架会自动记录日志并返回系统错误响应
		return fmt.Errorf("执行图表生成失败: %w", err)
	}
	logViz(ctx, "chart_generate", "python_execute_ok", start)

	outputPaths, err := execResult.OutputFilePaths()
	if err != nil {
		logger.Errorf(ctx, "[ChartGenerate] 输出文件校验失败: %v", err)
		return fmt.Errorf("图表输出文件无效: %w", err)
	}
	logViz(ctx, "chart_generate", "output_files_ok count="+fmt.Sprintf("%d", len(outputPaths)), start)

	// 创建 Files 对象
	chartFiles := fs.ResponseFiles(outputPaths)

	logViz(ctx, "chart_generate", "response_build_ok", start)

	// 构建响应
	return resp.Form(&ChartGenerateResp{
		ChartImage: chartFiles,
		ChartInfo:  result.ChartInfo,
		Status:     "成功",
	}).Build()
}

// buildChartGenerateCode 构建图表生成的 Python 代码
func buildChartGenerateCode() string {
	// 根据图表类型生成不同的代码
	chartCode := ""
	// 注意：chart_type 会在运行时从 request 中获取
	chartCode = `# 根据图表类型绘制
if chart_type == "柱状图":
    if "x" in data and "y" in data:
        ax.bar(data["x"], data["y"])
    elif "labels" in data and "values" in data:
        ax.bar(data["labels"], data["values"])
    else:
        raise ValueError("柱状图需要 'x' 和 'y' 或 'labels' 和 'values' 数据")
elif chart_type == "折线图":
    if "x" in data and "y" in data:
        ax.plot(data["x"], data["y"], marker='o')
    elif "y" in data:
        ax.plot(data["y"], marker='o')
    else:
        raise ValueError("折线图需要 'x' 和 'y' 或 'y' 数据")
elif chart_type == "散点图":
    if "x" in data and "y" in data:
        ax.scatter(data["x"], data["y"])
    else:
        raise ValueError("散点图需要 'x' 和 'y' 数据")
elif chart_type == "饼图":
    if "labels" in data and "values" in data:
        ax.pie(data["values"], labels=data["labels"], autopct='%1.1f%%')
        ax.set_aspect('equal')
    else:
        raise ValueError("饼图需要 'labels' 和 'values' 数据")
elif chart_type == "直方图":
    if "values" in data:
        ax.hist(data["values"], bins=20)
    elif "y" in data:
        ax.hist(data["y"], bins=20)
    else:
        raise ValueError("直方图需要 'values' 或 'y' 数据")
elif chart_type == "箱线图":
    if "values" in data:
        ax.boxplot(data["values"])
    elif "y" in data:
        ax.boxplot(data["y"])
    else:
        raise ValueError("箱线图需要 'values' 或 'y' 数据")
else:
    # 默认柱状图
    if "x" in data and "y" in data:
        ax.bar(data["x"], data["y"])
    elif "labels" in data and "values" in data:
        ax.bar(data["labels"], data["values"])
    else:
        raise ValueError("需要提供有效的数据")`

	indentedChartCode := "    " + strings.ReplaceAll(chartCode, "\n", "\n    ")

	code := `import matplotlib
matplotlib.use('Agg')  # 使用非交互式后端
import matplotlib.pyplot as plt
import os

# 字体与中文：由镜像 matplotlibrc 统一处理（Noto Sans CJK / WenQuanYi Zen Hei），此处仅兜底负号显示
plt.rcParams['axes.unicode_minus'] = False

def kageos_entry(args, output_dir):
    data = args["data"]
    chart_type = args["chart_type"]
    width = args["width"]
    height = args["height"]
    title = args["title"]
    x_label = args["x_label"]
    y_label = args["y_label"]
    output_path = args["output_path"]

    fig, ax = plt.subplots(figsize=(width/100, height/100), dpi=100)

    ax.set_title(title, fontsize=14, fontweight='bold')
    if x_label:
        ax.set_xlabel(x_label, fontsize=12)
    if y_label:
        ax.set_ylabel(y_label, fontsize=12)

` + indentedChartCode + `

    plt.tight_layout()
    out_dir = os.path.dirname(output_path)
    if out_dir:
        os.makedirs(out_dir, exist_ok=True)
    plt.savefig(output_path, format='png', dpi=150, bbox_inches='tight')
    plt.close(fig)

    return {
        "data": {
            "chart_info": f"图表类型: {chart_type}, 尺寸: {width}x{height}px, 标题: {title}"
        },
        "output_files": [
            {"path": output_path, "name": os.path.basename(output_path)}
        ]
    }`

	return code
}

// ChartGenerateTemplate 图表生成配置
var ChartGenerateTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "数据可视化图表生成",
		Desc:     `使用 matplotlib 生成专业的数据可视化图表。支持柱状图、折线图、散点图、饼图、直方图、箱线图等多种图表类型，生成高质量 PNG 图片。图表可直接用于报告和演示。应用场景：数据分析报告、数据可视化、图表生成、报告制作等。`,
		Tags:     []string{"数据可视化", "图表生成", "matplotlib", "数据分析"},
		Request:  &ChartGenerateReq{},
		Response: &ChartGenerateResp{},
	},
}

// sanitizeFileName 清理文件名，移除非法字符
func sanitizeFileName(name string) string {
	if name == "" {
		return ""
	}

	// 替换非法字符为下划线
	illegalChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range illegalChars {
		result = strings.ReplaceAll(result, char, "_")
	}

	// 移除首尾空格
	result = strings.TrimSpace(result)

	// 如果清理后为空，返回默认值
	if result == "" {
		return "数据图表"
	}

	return result
}

func init() {
	// 注册Form函数 - 数据可视化图表生成
	packageContext.POST("chart_generate.form", ChartGenerate, ChartGenerateTemplate)
}
