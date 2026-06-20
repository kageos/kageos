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

// ChartSeries 图表系列定义
type ChartSeries struct {
	// 框架标签：widget:"type:input;placeholder:系列名称" - 系列名称
	Name string `json:"name" widget:"name:系列名称;type:input;placeholder:如:销售额" validate:"required"`

	// 框架标签：widget:"type:select;options:柱状图,折线图,虚线图,散点图,面积图" - 图表类型
	ChartType string `json:"chart_type" widget:"name:图表类型;type:select;options:柱状图,折线图,虚线图,散点图,面积图;render_default:柱状图" validate:"required"`

	// 框架标签：widget:"type:input;placeholder:如:120,130,145,160" - 数据值（逗号分隔）
	DataValues string `json:"data_values" widget:"name:数据值;type:input;placeholder:请输入逗号分隔的数值,如:120,130,145,160" validate:"required"`

	// 框架标签：widget:"type:select;options:blue,red,green,orange,purple,brown,pink,gray,cyan,magenta" - 颜色
	Color string `json:"color" widget:"name:颜色;type:select;options:blue,red,green,orange,purple,brown,pink,gray,cyan,magenta;render_default:blue"`

	// 框架标签：widget:"type:input;placeholder:如:万元" - 单位
	Unit string `json:"unit" widget:"name:单位;type:input;placeholder:如:万元、个、%"`

	// 框架标签：widget:"type:switch;render_default:true" - 是否显示数值标签
	ShowValueLabels bool `json:"show_value_labels" widget:"name:显示数值标签;type:switch;render_default:true"`

	// 框架标签：widget:"type:switch;render_default:false" - 是否使用右侧Y轴
	UseRightAxis bool `json:"use_right_axis" widget:"name:使用右侧Y轴;type:switch;render_default:false"`
}

// UniversalCombinedChartReq 通用组合图表生成请求结构体
type UniversalCombinedChartReq struct {
	// 框架标签：widget:"type:input;placeholder:如:1月,2月,3月,4月" - X轴标签（逗号分隔）
	XLabels string `json:"x_labels" widget:"name:X轴标签;type:input;placeholder:请输入逗号分隔的X轴标签,如:1月,2月,3月,4月" validate:"required"`

	// 框架标签：widget:"type:table" - 图表系列数组
	Series []ChartSeries `json:"series" widget:"name:图表系列;type:table" validate:"required,min=1,max=5"`

	// 框架标签：widget:"type:input;placeholder:图表标题" - 图表标题
	Title string `json:"title" widget:"name:图表标题;type:input;placeholder:请输入图表标题;render_default:通用组合图表"`

	// 框架标签：widget:"type:input;placeholder:X轴名称" - X轴名称
	XAxisLabel string `json:"x_axis_label" widget:"name:X轴名称;type:input;placeholder:请输入X轴名称;render_default:时间/类别"`

	// 框架标签：widget:"type:integer;render_default:1000" - 图表宽度（像素）
	Width int `json:"width" widget:"name:图表宽度(像素);type:integer;render_default:1000;placeholder:请输入图表宽度"`

	// 框架标签：widget:"type:integer;render_default:700" - 图表高度（像素）
	Height int `json:"height" widget:"name:图表高度(像素);type:integer;render_default:700;placeholder:请输入图表高度"`

	// 框架标签：widget:"type:switch;render_default:true" - 是否显示图例
	ShowLegend bool `json:"show_legend" widget:"name:显示图例;type:switch;render_default:true"`

	// 框架标签：widget:"type:switch;render_default:true" - 是否显示网格线
	ShowGrid bool `json:"show_grid" widget:"name:显示网格线;type:switch;render_default:true"`

	// 框架标签：widget:"type:switch;render_default:true" - 是否显示增长率
	ShowGrowthRate bool `json:"show_growth_rate" widget:"name:显示增长率;type:switch;render_default:true"`
}

// UniversalCombinedChartResp 通用组合图表生成响应结构体
type UniversalCombinedChartResp struct {
	// 图表图片文件
	ChartImage string `json:"chart_image" widget:"name:图表图片;type:files"`

	// 图表信息
	ChartInfo string `json:"chart_info" widget:"name:图表信息;type:text_area"`

	// 生成状态
	Status string `json:"status" widget:"name:生成状态;type:text"`
}

// UniversalCombinedChart 通用组合图表生成函数
func UniversalCombinedChart(ctx *app.Context, resp response.Response) error {
	start := time.Now()
	logViz(ctx, "universal_combined_chart", "begin", start)

	var req UniversalCombinedChartReq
	err := ctx.ShouldBindValidate(&req)
	if err != nil {
		return err
	}
	logViz(ctx, "universal_combined_chart", "bind_validate_ok", start)

	// 设置默认值
	if req.Width <= 0 {
		req.Width = 1000
	}
	if req.Height <= 0 {
		req.Height = 700
	}
	if req.Title == "" {
		req.Title = "通用组合图表"
	}
	if req.XAxisLabel == "" {
		req.XAxisLabel = "时间/类别"
	}

	// 验证X轴标签
	xLabels := strings.Split(req.XLabels, ",")
	if len(xLabels) == 0 {
		return resp.BizErrorf("X轴标签不能为空").Build()
	}

	// 验证系列数据
	for i, series := range req.Series {
		// 解析数据值
		values := strings.Split(series.DataValues, ",")
		if len(values) == 0 {
			return resp.BizErrorf("第%d个系列的数据值不能为空", i+1).Build()
		}
		if len(values) != len(xLabels) {
			return resp.BizErrorf("第%d个系列的数据值数量(%d)必须与X轴标签数量(%d)一致", i+1, len(values), len(xLabels)).Build()
		}

		// 验证数值格式
		for j, val := range values {
			val = strings.TrimSpace(val)
			if _, err := strconv.ParseFloat(val, 64); err != nil {
				return resp.BizErrorf("第%d个系列的第%d个数据值'%s'不是有效的数字", i+1, j+1, val).Build()
			}
		}
	}

	logViz(ctx, "universal_combined_chart", "params_ok series="+strconv.Itoa(len(req.Series))+" x_labels="+strconv.Itoa(len(xLabels)), start)

	// 构建 Python 代码
	pythonCode := buildUniversalCombinedChartCode()

	// 创建请求结构体
	type PythonRequest struct {
		XLabels        []string                 `json:"x_labels"`
		Series         []map[string]interface{} `json:"series"`
		Width          int                      `json:"width"`
		Height         int                      `json:"height"`
		Title          string                   `json:"title"`
		XAxisLabel     string                   `json:"x_axis_label"`
		ShowLegend     bool                     `json:"show_legend"`
		ShowGrid       bool                     `json:"show_grid"`
		ShowGrowthRate bool                     `json:"show_growth_rate"`
		OutputPath     string                   `json:"output_path"`
	}

	// 准备系列数据
	seriesData := make([]map[string]interface{}, len(req.Series))
	for i, series := range req.Series {
		// 解析数据值
		valuesStr := strings.Split(series.DataValues, ",")
		values := make([]float64, len(valuesStr))
		for j, val := range valuesStr {
			val = strings.TrimSpace(val)
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				values[j] = f
			}
		}

		seriesData[i] = map[string]interface{}{
			"name":              series.Name,
			"chart_type":        series.ChartType,
			"data_values":       values,
			"color":             series.Color,
			"unit":              series.Unit,
			"show_value_labels": series.ShowValueLabels,
			"use_right_axis":    series.UseRightAxis,
		}
	}

	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()
	fileName := sanitizeFileName(req.Title)
	if fileName == "" {
		fileName = "通用组合图表"
	}
	outputPath := filepath.Join(outputDir, fileName+".png")

	pythonReq := PythonRequest{
		XLabels:        xLabels,
		Series:         seriesData,
		Width:          req.Width,
		Height:         req.Height,
		Title:          req.Title,
		XAxisLabel:     req.XAxisLabel,
		ShowLegend:     req.ShowLegend,
		ShowGrid:       req.ShowGrid,
		ShowGrowthRate: req.ShowGrowthRate,
		OutputPath:     outputPath,
	}

	executor := pythonRuntime.NewExecutor(pythonCode).
		WithRequest(pythonReq).
		WithOutputDir(outputDir).
		WithTimeout(30 * time.Second)
	defer func() {
		if cerr := executor.Close(); cerr != nil {
			logger.Warnf(ctx, "[UniversalCombinedChart] executor.Close: %v", cerr)
		}
	}()

	logViz(ctx, "universal_combined_chart", "python_execute_start timeout=30s", start)

	var result struct {
		ChartInfo string `json:"chart_info"`
	}

	execResult, err := executor.ExecuteJSONWithResult(ctx, &result)
	if err != nil {
		logViz(ctx, "universal_combined_chart", "python_execute_failed", start)
		logger.Errorf(ctx, "[UniversalCombinedChart] Python 执行失败: %v", err)
		return fmt.Errorf("执行通用组合图表生成失败: %w", err)
	}
	logViz(ctx, "universal_combined_chart", "python_execute_ok", start)

	outputPaths, err := execResult.OutputFilePaths()
	if err != nil {
		logger.Errorf(ctx, "[UniversalCombinedChart] 输出文件校验失败: %v", err)
		return fmt.Errorf("组合图输出文件无效: %w", err)
	}
	logViz(ctx, "universal_combined_chart", "output_files_ok count="+strconv.Itoa(len(outputPaths)), start)

	chartFiles := fs.ResponseFiles(outputPaths)

	logViz(ctx, "universal_combined_chart", "response_build_ok", start)

	// 构建响应
	return resp.Form(&UniversalCombinedChartResp{
		ChartImage: chartFiles,
		ChartInfo:  result.ChartInfo,
		Status:     "成功",
	}).Build()
}

// buildUniversalCombinedChartCode 构建通用组合图表的 Python 代码
func buildUniversalCombinedChartCode() string {
	code := `import matplotlib
matplotlib.use('Agg')  # 使用非交互式后端
import matplotlib.pyplot as plt
import numpy as np
import os

# 字体与中文：由镜像 matplotlibrc 统一处理
plt.rcParams['axes.unicode_minus'] = False

def kageos_entry(args, output_dir):
    x_labels = args["x_labels"]
    series = args["series"]
    width = args["width"]
    height = args["height"]
    title = args["title"]
    x_axis_label = args["x_axis_label"]
    show_legend = args["show_legend"]
    show_grid = args["show_grid"]
    show_growth_rate = args["show_growth_rate"]
    output_path = args["output_path"]

    fig, ax1 = plt.subplots(figsize=(width/100, height/100), dpi=100)
    ax1.set_title(title, fontsize=16, fontweight='bold', pad=20)
    ax1.set_xlabel(x_axis_label, fontsize=12)

    x = np.arange(len(x_labels))
    ax1.set_xticks(x)
    ax1.set_xticklabels(x_labels, rotation=45, ha='right')

    if show_grid:
        ax1.grid(True, linestyle='--', alpha=0.3)

    color_map = {
        'blue': 'blue',
        'red': 'red',
        'green': 'green',
        'orange': 'orange',
        'purple': 'purple',
        'brown': 'brown',
        'pink': 'pink',
        'gray': 'gray',
        'cyan': 'cyan',
        'magenta': 'magenta',
    }

    right_axes = []
    series_info = []
    bar_series_count = len([s for s in series if s.get("chart_type") == "柱状图"])
    bar_series_index = 0

    for i, s in enumerate(series):
        name = s.get("name", f"系列{i+1}")
        chart_type = s.get("chart_type", "柱状图")
        data_values = s.get("data_values", [])
        color_name = s.get("color", "blue")
        unit = s.get("unit", "")
        show_value_labels = s.get("show_value_labels", True)
        use_right_axis = s.get("use_right_axis", False)
        color = color_map.get(color_name, 'blue')

        if use_right_axis:
            if len(right_axes) == 0:
                ax = ax1.twinx()
                right_axes.append(ax)
            else:
                ax = ax1.twinx()
                ax.spines["right"].set_position(("outward", 60 * len(right_axes)))
                right_axes.append(ax)
            current_ax = ax
        else:
            current_ax = ax1

        y_label = name
        if unit:
            y_label += f"（{unit}）"

        if use_right_axis:
            current_ax.set_ylabel(y_label, fontsize=12, color=color)
        elif i == 0:
            current_ax.set_ylabel(y_label, fontsize=12, color=color)

        if chart_type == "柱状图":
            safe_count = bar_series_count if bar_series_count > 0 else 1
            bar_width = 0.8 / safe_count
            bar_offset = -0.4 + (bar_series_index * bar_width)
            bar_series_index += 1

            bars = current_ax.bar(x + bar_offset, data_values, width=bar_width, color=color, alpha=0.7, label=name)
            if show_value_labels:
                max_value = max(data_values) if data_values else 0
                for bar in bars:
                    height = bar.get_height()
                    current_ax.text(
                        bar.get_x() + bar.get_width()/2.,
                        height + (max_value * 0.02 if max_value else 0.2),
                        f'{height:.1f}',
                        ha='center',
                        va='bottom',
                        fontsize=8,
                        color=color,
                    )

        elif chart_type == "折线图":
            current_ax.plot(x, data_values, color=color, marker='o', linestyle='-', linewidth=2, label=name)
            if show_value_labels:
                for j, txt in enumerate(data_values):
                    current_ax.annotate(f'{txt:.1f}', (x[j], data_values[j]), textcoords="offset points", xytext=(0,8), ha='center', fontsize=8, color=color)

        elif chart_type == "虚线图":
            current_ax.plot(x, data_values, color=color, marker='s', linestyle='--', linewidth=2, label=name)
            if show_value_labels:
                for j, txt in enumerate(data_values):
                    current_ax.annotate(f'{txt:.1f}', (x[j], data_values[j]), textcoords="offset points", xytext=(0,8), ha='center', fontsize=8, color=color)

        elif chart_type == "散点图":
            current_ax.scatter(x, data_values, color=color, s=50, label=name, alpha=0.7)
            if show_value_labels:
                for j, txt in enumerate(data_values):
                    current_ax.annotate(f'{txt:.1f}', (x[j], data_values[j]), textcoords="offset points", xytext=(0,8), ha='center', fontsize=8, color=color)

        elif chart_type == "面积图":
            current_ax.fill_between(x, 0, data_values, color=color, alpha=0.3, label=name)
            current_ax.plot(x, data_values, color=color, linewidth=2, alpha=0.7)
            if show_value_labels:
                for j, txt in enumerate(data_values):
                    current_ax.annotate(f'{txt:.1f}', (x[j], data_values[j]), textcoords="offset points", xytext=(0,8), ha='center', fontsize=8, color=color)

        series_info.append({
            "name": name,
            "data": data_values,
            "unit": unit,
        })

    all_lines = []
    all_labels = []
    for ax in [ax1] + right_axes:
        lines, labels = ax.get_legend_handles_labels()
        all_lines.extend(lines)
        all_labels.extend(labels)

    if show_legend and all_lines:
        seen = set()
        unique_lines = []
        unique_labels = []
        for line, label in zip(all_lines, all_labels):
            if label not in seen:
                seen.add(label)
                unique_lines.append(line)
                unique_labels.append(label)
        ax1.legend(unique_lines, unique_labels, loc='upper left', fontsize=10)

    plt.tight_layout()

    out_dir = os.path.dirname(output_path)
    if out_dir:
        os.makedirs(out_dir, exist_ok=True)
    plt.savefig(output_path, format='png', dpi=150, bbox_inches='tight')
    plt.close(fig)

    chart_info_lines = []
    chart_info_lines.append("图表信息：")
    chart_info_lines.append(f"- 图表类型：组合图表（支持多种图表类型）")
    chart_info_lines.append(f"- 尺寸：{width}x{height}px")
    chart_info_lines.append(f"- 标题：{title}")
    chart_info_lines.append(f"- 数据点数量：{len(x_labels)}个")
    chart_info_lines.append(f"- 系列数量：{len(series)}个")

    if show_growth_rate:
        chart_info_lines.append("- 增长率分析：")
        for info in series_info:
            data = info["data"]
            if len(data) >= 2:
                start_val = data[0]
                end_val = data[-1]
                if start_val != 0:
                    growth_rate = ((end_val - start_val) / start_val * 100)
                    unit_str = f" {info['unit']}" if info['unit'] else ""
                    chart_info_lines.append(f"  • {info['name']}：{start_val:.1f}{unit_str} → {end_val:.1f}{unit_str}，增长 {growth_rate:.1f}%")
                else:
                    chart_info_lines.append(f"  • {info['name']}：起始值为0，无法计算增长率")
            else:
                chart_info_lines.append(f"  • {info['name']}：数据点不足，无法计算增长率")

    chart_info = "\n".join(chart_info_lines)
    return {
        "data": {"chart_info": chart_info},
        "output_files": [{"path": output_path, "name": os.path.basename(output_path)}]
    }`

	return code
}

// UniversalCombinedChartTemplate 通用组合图表生成配置
var UniversalCombinedChartTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "通用组合图表生成",
		Desc:     `生成通用的组合图表，支持在同一张图中显示多种图表类型（柱状图、折线图、虚线图、散点图、面积图）。用户可以自定义X轴标签、多个数据系列、颜色、单位等。特别适合展示多维度业务数据趋势分析。应用场景：业务报告、数据对比分析、多指标趋势展示、通用数据可视化等。`,
		Tags:     []string{"数据可视化", "组合图表", "通用工具", "趋势分析", "业务报告"},
		Request:  &UniversalCombinedChartReq{},
		Response: &UniversalCombinedChartResp{},
	},
}

func init() {
	// 注册Form函数 - 通用组合图表生成
	packageContext.POST("universal_combined_chart.form", UniversalCombinedChart, UniversalCombinedChartTemplate)
}
