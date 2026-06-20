package chart

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	pythonRuntime "github.com/kageos/kageos/sdk/agent-app/runtime/python"
)

// PieChartReq 饼图生成请求结构体
type PieChartReq struct {
	// 框架标签：widget:"type:input;placeholder:如:A产品,B产品,C产品" - 标签（逗号分隔）
	Labels string `json:"labels" widget:"name:标签;type:input;placeholder:请输入逗号分隔的标签,如:A产品,B产品,C产品" validate:"required"`

	// 框架标签：widget:"type:input;placeholder:如:30,40,30" - 数值（逗号分隔）
	Values string `json:"values" widget:"name:数值;type:input;placeholder:请输入逗号分隔的数值,如:30,40,30" validate:"required"`

	// 框架标签：widget:"type:input;placeholder:图表标题" - 图表标题
	Title string `json:"title" widget:"name:图表标题;type:input;placeholder:请输入图表标题;render_default:数据分布饼图"`

	// 框架标签：widget:"type:switch;render_default:true" - 是否显示百分比
	ShowPercentage bool `json:"show_percentage" widget:"name:显示百分比;type:switch;render_default:true"`

	// 框架标签：widget:"type:switch;render_default:true" - 是否显示图例
	ShowLegend bool `json:"show_legend" widget:"name:显示图例;type:switch;render_default:true"`

	// 框架标签：widget:"type:select;options:标准饼图,环形图,爆炸式饼图" - 饼图类型
	PieType string `json:"pie_type" widget:"name:饼图类型;type:select;options:标准饼图,环形图,爆炸式饼图;render_default:标准饼图"`

	// 框架标签：widget:"type:input;placeholder:如:市场份额,用户分布" - 数据说明
	DataDescription string `json:"data_description" widget:"name:数据说明;type:input;placeholder:如:市场份额,用户分布"`

	// 框架标签：widget:"type:integer;render_default:800" - 图表宽度（像素）
	Width int `json:"width" widget:"name:图表宽度(像素);type:integer;render_default:800;placeholder:请输入图表宽度"`

	// 框架标签：widget:"type:integer;render_default:600" - 图表高度（像素）
	Height int `json:"height" widget:"name:图表高度(像素);type:integer;render_default:600;placeholder:请输入图表高度"`

	// 框架标签：widget:"type:input;placeholder:如:#FF5733,#33FF57,#3357FF" - 自定义颜色（可选）
	Colors string `json:"colors" widget:"name:自定义颜色;type:input;placeholder:请输入逗号分隔的颜色代码,如:#FF5733,#33FF57,#3357FF（支持十六进制、RGB、颜色名）"`
}

// PieChartResp 饼图生成响应结构体
type PieChartResp struct {
	// 图表图片文件
	ChartImage string `json:"chart_image" widget:"name:图表图片;type:files"`

	// 图表信息
	ChartInfo string `json:"chart_info" widget:"name:图表信息;type:text_area"`

	// 生成状态
	Status string `json:"status" widget:"name:生成状态;type:text"`
}

// PieChart 饼图生成函数
func PieChart(ctx *app.Context, resp response.Response) error {
	start := time.Now()
	logViz(ctx, "pie_chart", "begin", start)

	var req PieChartReq
	err := ctx.ShouldBindValidate(&req)
	if err != nil {
		return err
	}
	logViz(ctx, "pie_chart", "bind_validate_ok", start)

	// 设置默认值
	if req.Width <= 0 {
		req.Width = 800
	}
	if req.Height <= 0 {
		req.Height = 600
	}
	if req.Title == "" {
		req.Title = "数据分布饼图"
	}
	if req.DataDescription == "" {
		req.DataDescription = "数据分布"
	}

	// 验证标签和数值
	labels := strings.Split(req.Labels, ",")
	valuesStr := strings.Split(req.Values, ",")

	if len(labels) == 0 {
		return resp.BizErrorf("标签不能为空").Build()
	}
	if len(valuesStr) == 0 {
		return resp.BizErrorf("数值不能为空").Build()
	}
	if len(labels) != len(valuesStr) {
		return resp.BizErrorf("标签数量(%d)必须与数值数量(%d)一致", len(labels), len(valuesStr)).Build()
	}

	// 验证数值格式并计算总和
	values := make([]float64, len(valuesStr))
	total := 0.0
	for i, valStr := range valuesStr {
		valStr = strings.TrimSpace(valStr)
		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return resp.BizErrorf("第%d个数值'%s'不是有效的数字", i+1, valStr).Build()
		}
		if val < 0 {
			return resp.BizErrorf("第%d个数值'%s'不能为负数", i+1, valStr).Build()
		}
		values[i] = val
		total += val
	}

	if total == 0 {
		return resp.BizErrorf("所有数值总和不能为0").Build()
	}

	logViz(ctx, "pie_chart", "params_ok labels="+strconv.Itoa(len(labels))+" pie_type="+req.PieType, start)

	// 构建 Python 代码
	pythonCode := buildPieChartCode()

	// 创建请求结构体
	type PythonRequest struct {
		Labels          []string  `json:"labels"`
		Values          []float64 `json:"values"`
		Title           string    `json:"title"`
		ShowPercentage  bool      `json:"show_percentage"`
		ShowLegend      bool      `json:"show_legend"`
		PieType         string    `json:"pie_type"`
		DataDescription string    `json:"data_description"`
		Width           int       `json:"width"`
		Height          int       `json:"height"`
		Colors          []string  `json:"colors,omitempty"` // 可选：自定义颜色
		OutputPath      string    `json:"output_path"`
	}

	// 处理自定义颜色
	var customColors []string
	if req.Colors != "" {
		customColors = strings.Split(req.Colors, ",")
		// 清理颜色字符串中的空格
		for i := range customColors {
			customColors[i] = strings.TrimSpace(customColors[i])
		}
	}

	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()
	fileName := sanitizeFileName(req.Title)
	if fileName == "" {
		fileName = "数据分布饼图"
	}
	outputPath := filepath.Join(outputDir, fileName+".png")

	pythonReq := PythonRequest{
		Labels:          labels,
		Values:          values,
		Title:           req.Title,
		ShowPercentage:  req.ShowPercentage,
		ShowLegend:      req.ShowLegend,
		PieType:         req.PieType,
		DataDescription: req.DataDescription,
		Width:           req.Width,
		Height:          req.Height,
		Colors:          customColors,
		OutputPath:      outputPath,
	}

	// 创建 Python 执行器（默认 Executor 超时为 5m，此处显式限制为 30s；须 defer Close 释放临时目录）
	executor := pythonRuntime.NewExecutor(pythonCode).
		WithRequest(pythonReq).
		WithOutputDir(outputDir).
		WithTimeout(30 * time.Second)
	defer func() {
		if cerr := executor.Close(); cerr != nil {
			logger.Warnf(ctx, "[PieChart] executor.Close: %v", cerr)
		}
	}()

	logViz(ctx, "pie_chart", "python_execute_start timeout=30s", start)

	var result struct {
		ChartInfo string `json:"chart_info"`
	}

	execResult, err := executor.ExecuteJSONWithResult(ctx, &result)
	if err != nil {
		logViz(ctx, "pie_chart", "python_execute_failed", start)
		logger.Errorf(ctx, "[PieChart] Python 执行失败: %v", err)
		return fmt.Errorf("执行饼图生成失败: %w", err)
	}
	logViz(ctx, "pie_chart", "python_execute_ok", start)

	outputPaths, err := execResult.OutputFilePaths()
	if err != nil {
		logger.Errorf(ctx, "[PieChart] 输出文件校验失败: %v", err)
		return fmt.Errorf("饼图输出文件无效: %w", err)
	}
	logViz(ctx, "pie_chart", "output_files_ok count="+strconv.Itoa(len(outputPaths)), start)

	chartFiles := fs.ResponseFiles(outputPaths)

	logViz(ctx, "pie_chart", "response_build_ok", start)

	// 构建响应
	return resp.Form(&PieChartResp{
		ChartImage: chartFiles,
		ChartInfo:  result.ChartInfo,
		Status:     "成功",
	}).Build()
}

// buildPieChartCode 构建饼图的 Python 代码
func buildPieChartCode() string {
	code := `import matplotlib
matplotlib.use('Agg')  # 使用非交互式后端
import matplotlib.pyplot as plt
import numpy as np
import os

# 字体与中文：由镜像 matplotlibrc 统一处理
plt.rcParams['axes.unicode_minus'] = False

def kageos_entry(args, output_dir):
    labels = args["labels"]
    values = args["values"]
    title = args["title"]
    show_percentage = args["show_percentage"]
    show_legend = args["show_legend"]
    pie_type = args["pie_type"]
    data_description = args["data_description"]
    width = args["width"]
    height = args["height"]
    colors = args.get("colors") or []
    output_path = args["output_path"]

    fig, ax = plt.subplots(figsize=(width/100, height/100), dpi=100)
    fig.set_facecolor('white')
    ax.set_facecolor('white')
    ax.set_title(title, fontsize=16, fontweight='bold', pad=20)

    if colors and len(colors) == len(labels):
        custom_colors = colors
    else:
        rich_colors = [
            '#FF6B6B', '#FFA726', '#FFD166', '#06D6A0', '#118AB2',
            '#073B4C', '#7209B7', '#F72585', '#4CC9F0', '#4361EE',
            '#3A0CA3', '#560BAD', '#480CA8', '#3F37C9', '#4895EF',
            '#4EA8DE', '#56CFE1', '#64DFDF', '#72EFDD', '#80FFDB'
        ]
        if len(labels) <= len(rich_colors):
            custom_colors = rich_colors[:len(labels)]
        else:
            extended_colors = []
            for i in range(len(labels)):
                if i < len(rich_colors):
                    extended_colors.append(rich_colors[i])
                else:
                    color_map_idx = i % 4
                    if color_map_idx == 0:
                        color = plt.cm.tab20c((i // 4) % 20)
                    elif color_map_idx == 1:
                        color = plt.cm.Set3((i // 4) % 12)
                    elif color_map_idx == 2:
                        color = plt.cm.Pastel1((i // 4) % 9)
                    else:
                        color = plt.cm.Set2((i // 4) % 8)

                    if hasattr(color, '__len__') and len(color) >= 3:
                        r, g, b = int(color[0]*255), int(color[1]*255), int(color[2]*255)
                        extended_colors.append('#{:02x}{:02x}{:02x}'.format(r, g, b))
                    else:
                        extended_colors.append(str(color))
            custom_colors = extended_colors

    total = sum(values)
    percentages = [v / total * 100 for v in values]

    if pie_type == "标准饼图":
        if show_percentage:
            wedges, texts, autotexts = ax.pie(
                values,
                labels=labels if show_legend else None,
                autopct='%1.1f%%',
                startangle=90,
                colors=custom_colors,
                wedgeprops=dict(edgecolor='w'),
                textprops=dict(fontsize=10),
            )
        else:
            wedges, texts = ax.pie(
                values,
                labels=labels if show_legend else None,
                startangle=90,
                colors=custom_colors,
                wedgeprops=dict(edgecolor='w'),
                textprops=dict(fontsize=10),
            )
            autotexts = []

    elif pie_type == "环形图":
        if show_percentage:
            wedges, texts, autotexts = ax.pie(
                values,
                labels=labels if show_legend else None,
                autopct='%1.1f%%',
                startangle=90,
                colors=custom_colors,
                wedgeprops=dict(width=0.4, edgecolor='w'),
                textprops=dict(fontsize=10),
                pctdistance=0.85,
            )
        else:
            wedges, texts = ax.pie(
                values,
                labels=labels if show_legend else None,
                startangle=90,
                colors=custom_colors,
                wedgeprops=dict(width=0.4, edgecolor='w'),
                textprops=dict(fontsize=10),
                pctdistance=0.85,
            )
            autotexts = []

    elif pie_type == "爆炸式饼图":
        explode = [0.1 if i == values.index(max(values)) else 0 for i in range(len(values))]
        if show_percentage:
            wedges, texts, autotexts = ax.pie(
                values,
                explode=explode,
                labels=labels if show_legend else None,
                autopct='%1.1f%%',
                startangle=90,
                colors=custom_colors,
                shadow=True,
                wedgeprops=dict(edgecolor='w'),
                textprops=dict(fontsize=10),
            )
        else:
            wedges, texts = ax.pie(
                values,
                explode=explode,
                labels=labels if show_legend else None,
                startangle=90,
                colors=custom_colors,
                shadow=True,
                wedgeprops=dict(edgecolor='w'),
                textprops=dict(fontsize=10),
            )
            autotexts = []
    else:
        if show_percentage:
            wedges, texts, autotexts = ax.pie(
                values,
                labels=labels if show_legend else None,
                autopct='%1.1f%%',
                startangle=90,
                colors=custom_colors,
                wedgeprops=dict(edgecolor='w'),
                textprops=dict(fontsize=10),
            )
        else:
            wedges, texts = ax.pie(
                values,
                labels=labels if show_legend else None,
                startangle=90,
                colors=custom_colors,
                wedgeprops=dict(edgecolor='w'),
                textprops=dict(fontsize=10),
            )
            autotexts = []

    ax.set_aspect('equal')

    if show_percentage:
        for autotext, color in zip(autotexts, custom_colors):
            try:
                color_str = str(color).lower()
                light_colors = ['light', 'yellow', 'white', 'pink', 'cyan', 'lime', 'beige', 'ivory']
                is_light = any(light in color_str for light in light_colors)
                text_color = 'black' if is_light else 'white'
            except Exception:
                text_color = 'white'
            autotext.set_color(text_color)
            autotext.set_fontsize(9)
            autotext.set_fontweight('bold')

    if not show_legend and labels:
        ax.legend(wedges, labels, title="图例", loc="center left", bbox_to_anchor=(1, 0, 0.5, 1), fontsize=10)

    if len(labels) > 20:
        start_angle = 90
        for i, (label, value) in enumerate(zip(labels, values)):
            angle = (value / total) * 360
            mid_angle = start_angle - angle / 2
            mid_angle_rad = np.radians(mid_angle)
            start_radius = 0.7
            end_radius = 1.1
            start_x = start_radius * np.cos(mid_angle_rad)
            start_y = start_radius * np.sin(mid_angle_rad)
            end_x = end_radius * np.cos(mid_angle_rad)
            end_y = end_radius * np.sin(mid_angle_rad)

            ax.plot([start_x, end_x], [start_y, end_y], color='gray', linewidth=0.5, alpha=0.7)

            if -90 <= mid_angle <= 90:
                ha = 'left'
                label_x = end_x + 0.05
            else:
                ha = 'right'
                label_x = end_x - 0.05

            if mid_angle_rad > 0:
                va = 'bottom'
                label_y = end_y + 0.02
            else:
                va = 'top'
                label_y = end_y - 0.02

            label_text = f"{i+1}. {label}"
            ax.text(
                label_x,
                label_y,
                label_text,
                ha=ha,
                va=va,
                fontsize=8,
                bbox=dict(boxstyle="round,pad=0.3", facecolor="white", edgecolor="gray", alpha=0.8),
            )
            start_angle -= angle

    plt.tight_layout()

    out_dir = os.path.dirname(output_path)
    if out_dir:
        os.makedirs(out_dir, exist_ok=True)
    plt.savefig(output_path, format='png', dpi=150, bbox_inches='tight', facecolor=fig.get_facecolor())
    plt.close(fig)

    max_value = max(values)
    max_index = values.index(max_value)
    max_label = labels[max_index]
    max_percentage = percentages[max_index]

    min_value = min(values)
    min_index = values.index(min_value)
    min_label = labels[min_index]
    min_percentage = percentages[min_index]

    chart_info_lines = []
    chart_info_lines.append("图表信息：")
    chart_info_lines.append(f"- 图表类型：{pie_type}")
    chart_info_lines.append(f"- 尺寸：{width}x{height}px")
    chart_info_lines.append(f"- 标题：{title}")
    chart_info_lines.append(f"- 数据说明：{data_description}")
    chart_info_lines.append(f"- 数据点数量：{len(labels)}个")
    chart_info_lines.append(f"- 数据总和：{total:.2f}")
    chart_info_lines.append("")
    chart_info_lines.append("数据分布详情：")
    for i, (label, value, percentage) in enumerate(zip(labels, values, percentages)):
        chart_info_lines.append(f"  {i+1}. {label}: {value:.2f} ({percentage:.1f}%)")
    chart_info_lines.append("")
    chart_info_lines.append("关键发现：")
    chart_info_lines.append(f"  • 最大值：{max_label} ({max_value:.2f}, 占{max_percentage:.1f}%)")
    chart_info_lines.append(f"  • 最小值：{min_label} ({min_value:.2f}, 占{min_percentage:.1f}%)")
    if len(values) >= 2:
        sorted_values = sorted(values, reverse=True)
        top2_percentage = (sorted_values[0] + sorted_values[1]) / total * 100
        chart_info_lines.append(f"  • 前2名集中度：{top2_percentage:.1f}%")

    chart_info = "\n".join(chart_info_lines)
    return {
        "data": {"chart_info": chart_info},
        "output_files": [{"path": output_path, "name": os.path.basename(output_path)}]
    }`

	return code
}

// PieChartTemplate 饼图生成配置
var PieChartTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "饼图生成",
		Desc:     `生成专业的饼图、环形图和爆炸式饼图，适合展示数据比例和分布情况。支持显示百分比、图例、多种饼图样式。应用场景：市场份额分析、用户分布、预算分配、构成比例等。`,
		Tags:     []string{"数据可视化", "饼图", "比例分析", "分布图", "环形图"},
		Request:  &PieChartReq{},
		Response: &PieChartResp{},
	},
}

func init() {
	// 注册Form函数 - 饼图生成
	packageContext.POST("pie_chart.form", PieChart, PieChartTemplate)
}
