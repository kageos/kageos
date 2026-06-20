package html

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type ChartGenerateReq struct {
	ChartType string `json:"chart_type" widget:"name:图表类型;type:select;options:柱状图 bar,折线图 line,饼图 pie,环形图 doughnut,雷达图 radar,极地图 polarArea;render_default:柱状图 bar" validate:"required"`
	Title     string `json:"title" widget:"name:图表标题;type:input;placeholder:例如：2024年季度销售额"`
	Labels    string `json:"labels" widget:"name:标签;type:text_area;placeholder:每行一个标签，例如：\nQ1\nQ2\nQ3\nQ4" validate:"required"`
	DataSets  string `json:"data_sets" widget:"name:数据集;type:text_area;placeholder:每行一个数据集，格式：名称|数值1,数值2,...\n例如：\n产品A|120,200,150,300\n产品B|80,160,200,250" validate:"required"`
	FileName  string `json:"file_name" widget:"name:文件名;type:input;placeholder:输入文件名（无需后缀），例如 monthly-sales" validate:"required"`
	Theme     string `json:"theme" widget:"name:配色;type:select;options:默认彩色,蓝色系,绿色系,暖色系;render_default:默认彩色" validate:"required"`
}

type ChartGenerateResp struct {
	OutputFile string `json:"output_file" widget:"name:HTML 文件;type:files"`
	Info       string `json:"info" widget:"name:信息;type:text"`
}

type chartDataSet struct {
	Label string    `json:"label"`
	Data  []float64 `json:"data"`
}

func ChartGenerate(ctx *app.Context, resp response.Response) error {
	var req ChartGenerateReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	chartType := parseChartType(req.ChartType)

	labels := parseLines(req.Labels)
	if len(labels) == 0 {
		return fmt.Errorf("至少需要一个标签")
	}

	dataSets, err := parseDataSets(req.DataSets)
	if err != nil {
		return err
	}
	if len(dataSets) == 0 {
		return fmt.Errorf("至少需要一个数据集")
	}

	title := req.Title
	if title == "" {
		title = "图表"
	}

	baseName := sanitizeFileName(req.FileName)

	colors := getChartColors(req.Theme)
	htmlContent := buildChartHTML(title, chartType, labels, dataSets, colors)

	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	outputPath := filepath.Join(outputDir, baseName+".html")
	if err := os.WriteFile(outputPath, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("保存文件失败: %v", err)
	}

	outputFiles := fs.ResponseFiles([]string{outputPath})

	return resp.Form(&ChartGenerateResp{
		OutputFile: outputFiles,
		Info:       fmt.Sprintf("%s | %d 个标签 × %d 个数据集", req.ChartType, len(labels), len(dataSets)),
	}).Build()
}

func parseChartType(t string) string {
	parts := strings.Fields(t)
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	switch t {
	case "柱状图":
		return "bar"
	case "折线图":
		return "line"
	case "饼图":
		return "pie"
	case "环形图":
		return "doughnut"
	case "雷达图":
		return "radar"
	case "极地图":
		return "polarArea"
	default:
		return "bar"
	}
}

func parseLines(s string) []string {
	var result []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func parseDataSets(s string) ([]chartDataSet, error) {
	var result []chartDataSet
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 2)
		var label, dataStr string
		if len(parts) == 2 {
			label = strings.TrimSpace(parts[0])
			dataStr = strings.TrimSpace(parts[1])
		} else {
			label = fmt.Sprintf("数据%d", len(result)+1)
			dataStr = line
		}

		var data []float64
		for _, v := range strings.Split(dataStr, ",") {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			var num float64
			if _, err := fmt.Sscanf(v, "%f", &num); err != nil {
				return nil, fmt.Errorf("无法解析数值 '%s': %v", v, err)
			}
			data = append(data, num)
		}

		if len(data) > 0 {
			result = append(result, chartDataSet{Label: label, Data: data})
		}
	}
	return result, nil
}

func getChartColors(theme string) []string {
	switch theme {
	case "蓝色系":
		return []string{
			"rgba(59,130,246,0.7)", "rgba(37,99,235,0.7)", "rgba(29,78,216,0.7)",
			"rgba(96,165,250,0.7)", "rgba(147,197,253,0.7)", "rgba(30,64,175,0.7)",
		}
	case "绿色系":
		return []string{
			"rgba(34,197,94,0.7)", "rgba(22,163,74,0.7)", "rgba(21,128,61,0.7)",
			"rgba(74,222,128,0.7)", "rgba(134,239,172,0.7)", "rgba(5,150,105,0.7)",
		}
	case "暖色系":
		return []string{
			"rgba(249,115,22,0.7)", "rgba(234,88,12,0.7)", "rgba(239,68,68,0.7)",
			"rgba(245,158,11,0.7)", "rgba(251,191,36,0.7)", "rgba(220,38,38,0.7)",
		}
	default:
		return []string{
			"rgba(59,130,246,0.7)", "rgba(239,68,68,0.7)", "rgba(34,197,94,0.7)",
			"rgba(245,158,11,0.7)", "rgba(168,85,247,0.7)", "rgba(236,72,153,0.7)",
			"rgba(20,184,166,0.7)", "rgba(249,115,22,0.7)", "rgba(99,102,241,0.7)",
		}
	}
}

func buildChartHTML(title, chartType string, labels []string, dataSets []chartDataSet, colors []string) string {
	labelsJSON, _ := json.Marshal(labels)

	var dsJSON []string
	isPie := chartType == "pie" || chartType == "doughnut" || chartType == "polarArea"

	for i, ds := range dataSets {
		dataBytes, _ := json.Marshal(ds.Data)
		if isPie {
			var bgColors []string
			for j := range ds.Data {
				bgColors = append(bgColors, colors[j%len(colors)])
			}
			bgJSON, _ := json.Marshal(bgColors)

			var borderColors []string
			for _, c := range bgColors {
				borderColors = append(borderColors, strings.Replace(c, "0.7)", "1)", 1))
			}
			bdJSON, _ := json.Marshal(borderColors)

			dsJSON = append(dsJSON, fmt.Sprintf(`{label:%q,data:%s,backgroundColor:%s,borderColor:%s,borderWidth:2}`,
				ds.Label, string(dataBytes), string(bgJSON), string(bdJSON)))
		} else {
			bg := colors[i%len(colors)]
			border := strings.Replace(bg, "0.7)", "1)", 1)
			dsJSON = append(dsJSON, fmt.Sprintf(`{label:%q,data:%s,backgroundColor:%q,borderColor:%q,borderWidth:2,tension:0.3}`,
				ds.Label, string(dataBytes), bg, border))
		}
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4/dist/chart.umd.min.js"></script>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans SC", sans-serif; background: #f8fafc; display: flex; justify-content: center; align-items: center; min-height: 100vh; padding: 2rem; }
.chart-container { background: #fff; border-radius: 12px; box-shadow: 0 2px 16px rgba(0,0,0,.08); padding: 2rem; width: 100%%; max-width: 900px; }
.chart-title { text-align: center; font-size: 1.3rem; color: #334155; margin-bottom: 1.5rem; font-weight: 600; }
canvas { width: 100%% !important; }
</style>
</head>
<body>
<div class="chart-container">
<div class="chart-title">%s</div>
<canvas id="chart"></canvas>
</div>
<script>
new Chart(document.getElementById('chart'), {
  type: '%s',
  data: { labels: %s, datasets: [%s] },
  options: {
    responsive: true,
    plugins: { legend: { position: 'top' } },
    scales: %s
  }
});
</script>
</body>
</html>`,
		template.HTMLEscapeString(title),
		template.HTMLEscapeString(title),
		chartType,
		string(labelsJSON),
		strings.Join(dsJSON, ","),
		chartScalesOption(chartType),
	)
}

func chartScalesOption(chartType string) string {
	switch chartType {
	case "pie", "doughnut", "polarArea", "radar":
		return "{}"
	default:
		return `{y:{beginAtZero:true,grid:{color:'#f1f5f9'}},x:{grid:{display:false}}}`
	}
}

var ChartGenerateTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "数据图表生成",
		Desc:     `根据数据生成可直接访问的交互式图表网页。支持柱状图、折线图、饼图、环形图、雷达图等类型，多种配色方案。基于 Chart.js，生成的网页可直接在浏览器中打开查看和交互。常用于数据汇报、趋势展示、报告图表等场景。`,
		Tags:     []string{"图表", "Chart", "数据可视化", "柱状图", "折线图", "饼图", "HTML", "网页"},
		Request:  &ChartGenerateReq{},
		Response: &ChartGenerateResp{},
	},
}

func init() {
	packageContext.POST("chart_generate.form", ChartGenerate, ChartGenerateTemplate)
}
