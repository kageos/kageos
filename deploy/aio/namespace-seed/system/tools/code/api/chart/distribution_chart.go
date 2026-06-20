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

// DistributionChartReq 分布图表生成请求结构体
type DistributionChartReq struct {
	// 框架标签：widget:"type:select;options:箱线图,直方图,密度图" - 图表类型
	ChartType string `json:"chart_type" widget:"name:图表类型;type:select;options:箱线图,直方图,密度图;render_default:箱线图" validate:"required"`

	// 框架标签：widget:"type:text_area;placeholder:请输入数据，每行一组数据，格式：组名:数值1,数值2,数值3" - 数据
	Data string `json:"data" widget:"name:数据;type:text_area;placeholder:请输入数据，每行一组数据，格式：组名:数值1,数值2,数值3\n例如：\nA组:12,15,18,20,22,25,28\nB组:10,12,14,16,18,20,22\nC组:8,10,12,14,16,18,20" validate:"required"`

	// 框架标签：widget:"type:input;placeholder:图表标题" - 图表标题
	Title string `json:"title" widget:"name:图表标题;type:input;placeholder:请输入图表标题;render_default:数据分布分析"`

	// 框架标签：widget:"type:input;placeholder:X轴标签" - X轴标签
	XLabel string `json:"x_label" widget:"name:X轴标签;type:input;placeholder:请输入X轴标签;render_default:数据组"`

	// 框架标签：widget:"type:input;placeholder:Y轴标签" - Y轴标签
	YLabel string `json:"y_label" widget:"name:Y轴标签;type:input;placeholder:请输入Y轴标签;render_default:数值"`

	// 框架标签：widget:"type:switch;render_default:true" - 是否显示均值线
	ShowMeanLine bool `json:"show_mean_line" widget:"name:显示均值线;type:switch;render_default:true"`

	// 框架标签：widget:"type:switch;render_default:true" - 是否显示中位数线
	ShowMedianLine bool `json:"show_median_line" widget:"name:显示中位数线;type:switch;render_default:true"`

	// 框架标签：widget:"type:switch;render_default:true" - 是否显示异常值
	ShowOutliers bool `json:"show_outliers" widget:"name:显示异常值;type:switch;render_default:true"`

	// 框架标签：widget:"type:integer;render_default:20" - 直方图分组数（仅直方图有效）
	Bins int `json:"bins" widget:"name:分组数(直方图);type:integer;render_default:20;placeholder:请输入分组数"`

	// 框架标签：widget:"type:switch;render_default:false" - 是否显示核密度估计（仅直方图有效）
	ShowKDE bool `json:"show_kde" widget:"name:显示核密度估计;type:switch;render_default:false"`

	// 框架标签：widget:"type:integer;render_default:1000" - 图表宽度（像素）
	Width int `json:"width" widget:"name:图表宽度(像素);type:integer;render_default:1000;placeholder:请输入图表宽度"`

	// 框架标签：widget:"type:integer;render_default:700" - 图表高度（像素）
	Height int `json:"height" widget:"name:图表高度(像素);type:integer;render_default:700;placeholder:请输入图表高度"`
}

// DistributionChartResp 分布图表生成响应结构体
type DistributionChartResp struct {
	// 图表图片文件
	ChartImage string `json:"chart_image" widget:"name:图表图片;type:files"`

	// 图表信息
	ChartInfo string `json:"chart_info" widget:"name:图表信息;type:text_area"`

	// 生成状态
	Status string `json:"status" widget:"name:生成状态;type:text"`
}

// DistributionChart 分布图表生成函数
func DistributionChart(ctx *app.Context, resp response.Response) error {
	start := time.Now()
	logViz(ctx, "distribution_chart", "begin", start)

	var req DistributionChartReq
	err := ctx.ShouldBindValidate(&req)
	if err != nil {
		return err
	}
	logViz(ctx, "distribution_chart", "bind_validate_ok", start)

	// 设置默认值
	if req.Width <= 0 {
		req.Width = 1000
	}
	if req.Height <= 0 {
		req.Height = 700
	}
	if req.Title == "" {
		req.Title = "数据分布分析"
	}
	if req.XLabel == "" {
		req.XLabel = "数据组"
	}
	if req.YLabel == "" {
		req.YLabel = "数值"
	}
	if req.Bins <= 0 {
		req.Bins = 20
	}

	// 解析数据
	lines := strings.Split(strings.TrimSpace(req.Data), "\n")
	if len(lines) == 0 {
		return resp.BizErrorf("数据不能为空").Build()
	}

	// 解析数据为组
	groups := make(map[string][]float64)
	groupNames := make([]string, 0)

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析格式：组名:数值1,数值2,数值3
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return resp.BizErrorf("第%d行格式错误，应为'组名:数值1,数值2,数值3'格式", i+1).Build()
		}

		groupName := strings.TrimSpace(parts[0])
		if groupName == "" {
			return resp.BizErrorf("第%d行组名不能为空", i+1).Build()
		}

		// 解析数值
		valuesStr := strings.Split(parts[1], ",")
		values := make([]float64, 0)

		for j, valStr := range valuesStr {
			valStr = strings.TrimSpace(valStr)
			if valStr == "" {
				continue
			}

			val, err := strconv.ParseFloat(valStr, 64)
			if err != nil {
				return resp.BizErrorf("第%d行第%d个数值'%s'不是有效的数字", i+1, j+1, valStr).Build()
			}
			values = append(values, val)
		}

		if len(values) == 0 {
			return resp.BizErrorf("第%d行没有有效的数值", i+1).Build()
		}

		groups[groupName] = values
		groupNames = append(groupNames, groupName)
	}

	if len(groups) == 0 {
		return resp.BizErrorf("没有有效的数据组").Build()
	}

	logViz(ctx, "distribution_chart", "params_ok chart_type="+req.ChartType+" groups="+strconv.Itoa(len(groups)), start)

	// 构建 Python 代码
	pythonCode := buildDistributionChartCode()

	// 创建请求结构体
	type PythonRequest struct {
		ChartType      string               `json:"chart_type"`
		Groups         map[string][]float64 `json:"groups"`
		GroupNames     []string             `json:"group_names"`
		Title          string               `json:"title"`
		XLabel         string               `json:"x_label"`
		YLabel         string               `json:"y_label"`
		ShowMeanLine   bool                 `json:"show_mean_line"`
		ShowMedianLine bool                 `json:"show_median_line"`
		ShowOutliers   bool                 `json:"show_outliers"`
		Bins           int                  `json:"bins"`
		ShowKDE        bool                 `json:"show_kde"`
		Width          int                  `json:"width"`
		Height         int                  `json:"height"`
		OutputPath     string               `json:"output_path"`
	}

	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()
	fileName := sanitizeFileName(req.Title)
	if fileName == "" {
		fileName = "数据分布分析"
	}
	outputPath := filepath.Join(outputDir, fileName+".png")

	pythonReq := PythonRequest{
		ChartType:      req.ChartType,
		Groups:         groups,
		GroupNames:     groupNames,
		Title:          req.Title,
		XLabel:         req.XLabel,
		YLabel:         req.YLabel,
		ShowMeanLine:   req.ShowMeanLine,
		ShowMedianLine: req.ShowMedianLine,
		ShowOutliers:   req.ShowOutliers,
		Bins:           req.Bins,
		ShowKDE:        req.ShowKDE,
		Width:          req.Width,
		Height:         req.Height,
		OutputPath:     outputPath,
	}

	executor := pythonRuntime.NewExecutor(pythonCode).
		WithRequest(pythonReq).
		WithOutputDir(outputDir).
		WithTimeout(30 * time.Second)
	defer func() {
		if cerr := executor.Close(); cerr != nil {
			logger.Warnf(ctx, "[DistributionChart] executor.Close: %v", cerr)
		}
	}()

	logViz(ctx, "distribution_chart", "python_execute_start timeout=30s", start)

	var result struct {
		ChartInfo string `json:"chart_info"`
	}

	execResult, err := executor.ExecuteJSONWithResult(ctx, &result)
	if err != nil {
		logViz(ctx, "distribution_chart", "python_execute_failed", start)
		logger.Errorf(ctx, "[DistributionChart] Python 执行失败: %v", err)
		return fmt.Errorf("执行分布图表生成失败: %w", err)
	}
	logViz(ctx, "distribution_chart", "python_execute_ok", start)

	outputPaths, err := execResult.OutputFilePaths()
	if err != nil {
		logger.Errorf(ctx, "[DistributionChart] 输出文件校验失败: %v", err)
		return fmt.Errorf("分布图输出文件无效: %w", err)
	}
	logViz(ctx, "distribution_chart", "output_files_ok count="+strconv.Itoa(len(outputPaths)), start)

	chartFiles := fs.ResponseFiles(outputPaths)

	logViz(ctx, "distribution_chart", "response_build_ok", start)

	// 构建响应
	return resp.Form(&DistributionChartResp{
		ChartImage: chartFiles,
		ChartInfo:  result.ChartInfo,
		Status:     "成功",
	}).Build()
}

// buildDistributionChartCode 构建分布图表的 Python 代码
func buildDistributionChartCode() string {
	code := `import matplotlib
matplotlib.use('Agg')  # 使用非交互式后端
import matplotlib.pyplot as plt
import numpy as np
import os
from scipy import stats

# 字体与中文：由镜像 matplotlibrc 统一处理
plt.rcParams['axes.unicode_minus'] = False

def kageos_entry(args, output_dir):
    chart_type = args["chart_type"]
    groups = args["groups"]
    group_names = args["group_names"]
    title = args["title"]
    x_label = args["x_label"]
    y_label = args["y_label"]
    show_mean_line = args["show_mean_line"]
    show_median_line = args["show_median_line"]
    show_outliers = args["show_outliers"]
    bins = args["bins"]
    show_kde = args["show_kde"]
    width = args["width"]
    height = args["height"]
    output_path = args["output_path"]

    fig, ax = plt.subplots(figsize=(width/100, height/100), dpi=100)
    fig.set_facecolor('white')
    ax.set_facecolor('white')

    ax.set_title(title, fontsize=16, fontweight='bold', pad=20)
    ax.set_xlabel(x_label, fontsize=12)
    ax.set_ylabel(y_label, fontsize=12)

    data_values = []
    group_labels = []
    all_data = []

    for group_name in group_names:
        values = groups[group_name]
        data_values.append(values)
        group_labels.append(group_name)
        all_data.extend(values)

    if chart_type == "箱线图":
        box = ax.boxplot(
            data_values,
            tick_labels=group_labels,
            showmeans=show_mean_line,
            showfliers=show_outliers,
            patch_artist=True,
            medianprops=dict(color='red', linewidth=2),
            meanprops=dict(marker='D', markeredgecolor='black', markerfacecolor='green'),
            flierprops=dict(marker='o', color='red', alpha=0.5)
        )

        colors = plt.cm.Set3(np.arange(len(data_values)) % 12)
        for patch, color in zip(box['boxes'], colors):
            patch.set_facecolor(color)
            patch.set_alpha(0.7)

        ax.grid(True, linestyle='--', alpha=0.3, axis='y')

        if show_mean_line or show_median_line:
            legend_elements = []
            if show_mean_line:
                legend_elements.append(plt.Line2D([0], [0], marker='D', color='w', label='均值',
                                                markerfacecolor='green', markersize=10))
            if show_median_line:
                legend_elements.append(plt.Line2D([0], [0], color='red', lw=2, label='中位数'))
            if show_outliers:
                legend_elements.append(plt.Line2D([0], [0], marker='o', color='w', label='异常值',
                                                markerfacecolor='red', markersize=8, alpha=0.5))

            if legend_elements:
                ax.legend(handles=legend_elements, loc='upper right', fontsize=10)

    elif chart_type == "直方图":
        if len(data_values) == 1:
            ax.hist(data_values[0], bins=bins, alpha=0.7, edgecolor='black',
                    label=group_labels[0] if len(group_labels) > 0 else "数据")

            if show_kde:
                kde = stats.gaussian_kde(data_values[0])
                x_range = np.linspace(min(data_values[0]), max(data_values[0]), 1000)
                ax.plot(
                    x_range,
                    kde(x_range) * len(data_values[0]) * (max(data_values[0]) - min(data_values[0])) / bins,
                    color='red',
                    linewidth=2,
                    label='核密度估计',
                )
        else:
            colors = plt.cm.Set3(np.arange(len(data_values)) % 12)
            for i, (values, label) in enumerate(zip(data_values, group_labels)):
                ax.hist(values, bins=bins, alpha=0.5, edgecolor='black', label=label, color=colors[i])

        ax.grid(True, linestyle='--', alpha=0.3)
        if len(group_labels) > 0:
            ax.legend(loc='upper right', fontsize=10)

    elif chart_type == "密度图":
        colors = plt.cm.Set3(np.arange(len(data_values)) % 12)

        for i, (values, label) in enumerate(zip(data_values, group_labels)):
            kde = stats.gaussian_kde(values)
            x_range = np.linspace(min(values), max(values), 1000)
            y_range = kde(x_range)

            ax.plot(x_range, y_range, color=colors[i], linewidth=2, label=label)
            ax.fill_between(x_range, y_range, alpha=0.3, color=colors[i])

        ax.grid(True, linestyle='--', alpha=0.3)
        ax.legend(loc='upper right', fontsize=10)
        ax.set_ylabel('密度', fontsize=12)

    else:
        box = ax.boxplot(data_values, labels=group_labels, patch_artist=True)
        colors = plt.cm.Set3(np.arange(len(data_values)) % 12)
        for patch, color in zip(box['boxes'], colors):
            patch.set_facecolor(color)
            patch.set_alpha(0.7)
        ax.grid(True, linestyle='--', alpha=0.3, axis='y')

    plt.tight_layout()

    out_dir = os.path.dirname(output_path)
    if out_dir:
        os.makedirs(out_dir, exist_ok=True)
    plt.savefig(output_path, format='png', dpi=150, bbox_inches='tight', facecolor=fig.get_facecolor())
    plt.close(fig)

    chart_info_lines = []
    chart_info_lines.append("图表信息：")
    chart_info_lines.append(f"- 图表类型：{chart_type}")
    chart_info_lines.append(f"- 尺寸：{width}x{height}px")
    chart_info_lines.append(f"- 标题：{title}")
    chart_info_lines.append(f"- 数据组数量：{len(group_names)}组")
    chart_info_lines.append(f"- 总数据点数：{len(all_data)}个")

    chart_info_lines.append("")
    chart_info_lines.append("各组统计信息：")

    for group_name in group_names:
        values = groups[group_name]
        if len(values) > 0:
            mean_val = np.mean(values)
            median_val = np.median(values)
            std_val = np.std(values)
            min_val = np.min(values)
            max_val = np.max(values)
            q1 = np.percentile(values, 25)
            q3 = np.percentile(values, 75)
            iqr = q3 - q1

            chart_info_lines.append(f"  【{group_name}】")
            chart_info_lines.append(f"    • 数据点数：{len(values)}")
            chart_info_lines.append(f"    • 均值：{mean_val:.4f}")
            chart_info_lines.append(f"    • 中位数：{median_val:.4f}")
            chart_info_lines.append(f"    • 标准差：{std_val:.4f}")
            chart_info_lines.append(f"    • 最小值：{min_val:.4f}")
            chart_info_lines.append(f"    • 最大值：{max_val:.4f}")
            chart_info_lines.append(f"    • 四分位距(IQR)：{iqr:.4f} (Q1={q1:.4f}, Q3={q3:.4f})")

            lower_bound = q1 - 1.5 * iqr
            upper_bound = q3 + 1.5 * iqr
            outliers = [v for v in values if v < lower_bound or v > upper_bound]
            if outliers:
                chart_info_lines.append(f"    • 异常值数量：{len(outliers)}个")
                chart_info_lines.append(f"    • 异常值范围：{min(outliers):.4f} ~ {max(outliers):.4f}")
            else:
                chart_info_lines.append(f"    • 异常值数量：0个")

            chart_info_lines.append("")

    if len(all_data) > 0:
        chart_info_lines.append("总体统计信息：")
        chart_info_lines.append(f"  • 总体均值：{np.mean(all_data):.4f}")
        chart_info_lines.append(f"  • 总体中位数：{np.median(all_data):.4f}")
        chart_info_lines.append(f"  • 总体标准差：{np.std(all_data):.4f}")
        chart_info_lines.append(f"  • 数据范围：{np.min(all_data):.4f} ~ {np.max(all_data):.4f}")

        if len(all_data) >= 20:
            from scipy.stats import shapiro
            try:
                stat, p_value = shapiro(all_data)
                chart_info_lines.append(f"  • Shapiro-Wilk正态性检验：")
                chart_info_lines.append(f"    - 统计量：{stat:.4f}")
                chart_info_lines.append(f"    - P值：{p_value:.4f}")
                if p_value > 0.05:
                    chart_info_lines.append(f"    - 结论：数据符合正态分布 (p > 0.05)")
                else:
                    chart_info_lines.append(f"    - 结论：数据不符合正态分布 (p ≤ 0.05)")
            except Exception:
                pass

    chart_info = "\n".join(chart_info_lines)
    return {
        "data": {"chart_info": chart_info},
        "output_files": [{"path": output_path, "name": os.path.basename(output_path)}]
    }`

	return code
}

// DistributionChartTemplate 分布图表生成配置
var DistributionChartTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "分布图表生成",
		Desc:     `生成专业的分布图表，包括箱线图、直方图和密度图。适合展示数据分布、统计特征和异常值检测。支持多组数据对比、统计指标计算。应用场景：数据质量分析、异常检测、统计分布、多组对比等。`,
		Tags:     []string{"数据可视化", "箱线图", "直方图", "统计分布", "异常检测"},
		Request:  &DistributionChartReq{},
		Response: &DistributionChartResp{},
	},
}

func init() {
	// 注册Form函数 - 分布图表生成
	packageContext.POST("distribution_chart.form", DistributionChart, DistributionChartTemplate)
}
