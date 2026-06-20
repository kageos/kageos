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

type DashboardGenerateReq struct {
	Title    string `json:"title" widget:"name:看板标题;type:input;placeholder:例如：2024年Q4运营数据看板" validate:"required"`
	FileName string `json:"file_name" widget:"name:文件名;type:input;placeholder:输入文件名（无需后缀），例如 q4-dashboard" validate:"required"`
	KPICards string `json:"kpi_cards" widget:"name:KPI 指标卡片;type:text_area;placeholder:每行一个指标，格式：名称|数值|描述\n例如：\n总收入|¥128.5万|同比增长 23%\n用户数|5,320|本月新增 480\n转化率|12.8%|环比提升 2.1pp" validate:"required"`
	Charts   string `json:"charts" widget:"name:图表数据;type:text_area;placeholder:多个图表用 === 分隔，每个图表格式：\n类型|标题\n标签1,标签2,...\n数据集名|数值1,数值2,...\n\n例如：\nbar|月度销售额\n1月,2月,3月,4月\n产品A|120,200,150,300\n产品B|80,160,200,250\n===\npie|客户来源分布\n搜索引擎,社交媒体,直接访问,邮件\n来源|40,25,20,15"`
	Theme    string `json:"theme" widget:"name:主题;type:select;options:浅色,深色;render_default:浅色" validate:"required"`
}

type DashboardGenerateResp struct {
	OutputFile string `json:"output_file" widget:"name:HTML 文件;type:files"`
	Info       string `json:"info" widget:"name:信息;type:text"`
}

type kpiCard struct {
	Name  string
	Value string
	Desc  string
}

type dashChart struct {
	Type     string
	Title    string
	Labels   []string
	DataSets []chartDataSet
}

func DashboardGenerate(ctx *app.Context, resp response.Response) error {
	var req DashboardGenerateReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	baseName := sanitizeFileName(req.FileName)
	cards := parseKPICards(req.KPICards)
	charts := parseDashCharts(req.Charts)
	isDark := req.Theme == "深色"

	htmlContent := buildDashboardHTML(req.Title, cards, charts, isDark)

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

	return resp.Form(&DashboardGenerateResp{
		OutputFile: outputFiles,
		Info:       fmt.Sprintf("%d 个指标卡片 + %d 个图表", len(cards), len(charts)),
	}).Build()
}

func parseKPICards(s string) []kpiCard {
	var cards []kpiCard
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		card := kpiCard{Name: strings.TrimSpace(parts[0])}
		if len(parts) >= 2 {
			card.Value = strings.TrimSpace(parts[1])
		}
		if len(parts) >= 3 {
			card.Desc = strings.TrimSpace(parts[2])
		}
		cards = append(cards, card)
	}
	return cards
}

func parseDashCharts(s string) []dashChart {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var charts []dashChart
	blocks := strings.Split(s, "===")
	for _, block := range blocks {
		lines := parseLines(strings.TrimSpace(block))
		if len(lines) < 2 {
			continue
		}

		headerParts := strings.SplitN(lines[0], "|", 2)
		chartType := "bar"
		chartTitle := lines[0]
		if len(headerParts) == 2 {
			chartType = strings.TrimSpace(headerParts[0])
			chartTitle = strings.TrimSpace(headerParts[1])
		}

		labels := strings.Split(lines[1], ",")
		for i := range labels {
			labels[i] = strings.TrimSpace(labels[i])
		}

		var dataSets []chartDataSet
		for _, dl := range lines[2:] {
			parts := strings.SplitN(dl, "|", 2)
			label := fmt.Sprintf("数据%d", len(dataSets)+1)
			dataStr := dl
			if len(parts) == 2 {
				label = strings.TrimSpace(parts[0])
				dataStr = strings.TrimSpace(parts[1])
			}
			var data []float64
			for _, v := range strings.Split(dataStr, ",") {
				var num float64
				if _, err := fmt.Sscanf(strings.TrimSpace(v), "%f", &num); err == nil {
					data = append(data, num)
				}
			}
			if len(data) > 0 {
				dataSets = append(dataSets, chartDataSet{Label: label, Data: data})
			}
		}

		if len(dataSets) > 0 {
			charts = append(charts, dashChart{Type: chartType, Title: chartTitle, Labels: labels, DataSets: dataSets})
		}
	}
	return charts
}

func buildDashboardHTML(title string, cards []kpiCard, charts []dashChart, isDark bool) string {
	var sb strings.Builder

	bg := "#f0f4f8"
	cardBg := "#fff"
	textColor := "#334155"
	subColor := "#64748b"
	valueColor := "#0f172a"
	borderColor := "#e2e8f0"
	if isDark {
		bg = "#0f172a"
		cardBg = "#1e293b"
		textColor = "#e2e8f0"
		subColor = "#94a3b8"
		valueColor = "#f1f5f9"
		borderColor = "#334155"
	}

	sb.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4/dist/chart.umd.min.js"></script>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Noto Sans SC",sans-serif;background:%s;color:%s;padding:2rem}
.container{max-width:1200px;margin:0 auto}
.dash-title{font-size:1.6rem;font-weight:700;margin-bottom:2rem;color:%s}
.kpi-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:1rem;margin-bottom:2rem}
.kpi-card{background:%s;border-radius:10px;padding:1.2rem 1.5rem;border:1px solid %s}
.kpi-name{font-size:.85rem;color:%s;margin-bottom:.4rem}
.kpi-value{font-size:1.8rem;font-weight:700;color:%s;line-height:1.2}
.kpi-desc{font-size:.78rem;color:%s;margin-top:.4rem}
.charts-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(460px,1fr));gap:1.5rem}
.chart-card{background:%s;border-radius:10px;padding:1.5rem;border:1px solid %s}
.chart-title{font-size:1rem;font-weight:600;margin-bottom:1rem;color:%s}
canvas{width:100%% !important}
</style>
</head>
<body>
<div class="container">
<div class="dash-title">%s</div>`,
		template.HTMLEscapeString(title),
		bg, textColor, valueColor,
		cardBg, borderColor,
		subColor, valueColor, subColor,
		cardBg, borderColor, textColor,
		template.HTMLEscapeString(title),
	))

	if len(cards) > 0 {
		sb.WriteString(`<div class="kpi-grid">`)
		for _, c := range cards {
			sb.WriteString(fmt.Sprintf(`<div class="kpi-card"><div class="kpi-name">%s</div><div class="kpi-value">%s</div>`,
				template.HTMLEscapeString(c.Name), template.HTMLEscapeString(c.Value)))
			if c.Desc != "" {
				sb.WriteString(fmt.Sprintf(`<div class="kpi-desc">%s</div>`, template.HTMLEscapeString(c.Desc)))
			}
			sb.WriteString(`</div>`)
		}
		sb.WriteString(`</div>`)
	}

	if len(charts) > 0 {
		sb.WriteString(`<div class="charts-grid">`)
		defaultColors := getChartColors("默认彩色")
		for i, ch := range charts {
			canvasID := fmt.Sprintf("chart%d", i)
			sb.WriteString(fmt.Sprintf(`<div class="chart-card"><div class="chart-title">%s</div><canvas id="%s"></canvas></div>`,
				template.HTMLEscapeString(ch.Title), canvasID))

			labelsJSON, _ := json.Marshal(ch.Labels)
			isPie := ch.Type == "pie" || ch.Type == "doughnut" || ch.Type == "polarArea"

			var dsEntries []string
			for di, ds := range ch.DataSets {
				dataBytes, _ := json.Marshal(ds.Data)
				if isPie {
					var bgColors []string
					for j := range ds.Data {
						bgColors = append(bgColors, defaultColors[j%len(defaultColors)])
					}
					bgJSON, _ := json.Marshal(bgColors)
					dsEntries = append(dsEntries, fmt.Sprintf(`{label:%q,data:%s,backgroundColor:%s,borderWidth:2}`,
						ds.Label, string(dataBytes), string(bgJSON)))
				} else {
					c := defaultColors[di%len(defaultColors)]
					border := strings.Replace(c, "0.7)", "1)", 1)
					dsEntries = append(dsEntries, fmt.Sprintf(`{label:%q,data:%s,backgroundColor:%q,borderColor:%q,borderWidth:2,tension:0.3}`,
						ds.Label, string(dataBytes), c, border))
				}
			}

			scales := `{y:{beginAtZero:true,grid:{color:'rgba(148,163,184,0.15)'}},x:{grid:{display:false}}}`
			if isPie || ch.Type == "radar" {
				scales = "{}"
			}

			sb.WriteString(fmt.Sprintf(`<script>new Chart(document.getElementById('%s'),{type:'%s',data:{labels:%s,datasets:[%s]},options:{responsive:true,plugins:{legend:{position:'top'}},scales:%s}});</script>`,
				canvasID, ch.Type, string(labelsJSON), strings.Join(dsEntries, ","), scales))
		}
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</div></body></html>`)
	return sb.String()
}

var DashboardGenerateTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "数据看板生成",
		Desc:     `生成可直接访问的数据看板网页，包含 KPI 指标卡片和多个图表。支持柱状图、折线图、饼图等多种图表类型组合展示。浅色/深色主题可选。常用于运营数据汇报、项目数据总览、业务指标监控等场景。`,
		Tags:     []string{"看板", "Dashboard", "数据看板", "KPI", "图表", "数据可视化", "报表", "HTML"},
		Request:  &DashboardGenerateReq{},
		Response: &DashboardGenerateResp{},
	},
}

func init() {
	packageContext.POST("dashboard_generate.form", DashboardGenerate, DashboardGenerateTemplate)
}
