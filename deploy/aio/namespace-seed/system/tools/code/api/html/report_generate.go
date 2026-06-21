package html

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"sort"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type ReportGenerateReq struct {
	Title           string `json:"title" widget:"name:报告标题;type:input;placeholder:例如：Q4 运营复盘报告" validate:"required"`
	Subtitle        string `json:"subtitle" widget:"name:副标题;type:input;placeholder:可选，例如：增长、转化与风险分析"`
	Summary         string `json:"summary" widget:"name:摘要;type:text_area;placeholder:可选，报告顶部摘要，支持 {{asset:文件名}} 图片占位符"`
	ContentMarkdown string `json:"content_markdown" widget:"name:正文 Markdown;type:text_area;placeholder:支持标题、列表、表格、引用、代码块等 Markdown/GFM 语法" validate:"required"`
	KPICards        string `json:"kpi_cards" widget:"name:KPI 指标卡片;type:text_area;placeholder:可选，每行一个指标，格式：名称|数值|描述\n例如：\n总收入|¥128.5万|同比增长 23%\n活跃用户|5,320|本月新增 480"`
	TableJSON       string `json:"table_json" widget:"name:附加表格 JSON;type:text_area;placeholder:可选，JSON 数组或对象，例如 [{\"渠道\":\"搜索\",\"转化率\":\"12%\"}]"`
	Charts          string `json:"charts" widget:"name:图表数据;type:text_area;placeholder:可选，多个图表用 === 分隔。格式：\n类型|标题\n标签1,标签2,...\n数据集名|数值1,数值2,...\n\n类型支持 bar、line、pie，例如：\nbar|月度销售额\n1月,2月,3月\n产品A|120,200,150\n产品B|80,160,200"`
	FileName        string `json:"file_name" widget:"name:文件名;type:input;placeholder:输入文件名（无需后缀），例如 q4-report" validate:"required"`
	Theme           string `json:"theme" widget:"name:主题;type:select;options:专业蓝,暖色报告,深色;options_colors:409EFF,E6A23C,909399;render_default:专业蓝" validate:"required,oneof=专业蓝 暖色报告 深色"`
	Assets          string `json:"assets" widget:"name:图片资源;type:files;accept:image/*,.svg,.webp,.png,.jpg,.jpeg,.gif;max_size:20MB;max_count:30"`
}

type ReportGenerateResp struct {
	OutputFile string `json:"output_file" widget:"name:HTML 报告;type:files"`
	Info       string `json:"info" widget:"name:生成信息;type:text_area"`
}

func ReportGenerate(ctx *app.Context, resp response.Response) error {
	var req ReportGenerateReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	fs := ctx.GetFS()
	downloadedAssets, assetURIs, err := downloadAssetDataURIs(ctx, req.Assets)
	if err != nil {
		return err
	}
	defer fs.RemoveFiles(downloadedAssets)

	summary, summaryAssetCount := embedAssetPlaceholders(req.Summary, assetURIs)
	summaryHTML := ""
	if strings.TrimSpace(summary) != "" {
		summaryHTML, err = markdownToHTMLFragment(summary)
		if err != nil {
			return err
		}
	}
	markdownText, contentAssetCount := embedAssetPlaceholders(req.ContentMarkdown, assetURIs)
	contentHTML, err := markdownToHTMLFragment(markdownText)
	if err != nil {
		return err
	}

	cards := parseKPICards(req.KPICards)
	charts := parseDashCharts(req.Charts)
	tableColumns, tableRows, err := parseReportTableJSON(req.TableJSON)
	if err != nil {
		return err
	}

	htmlContent := buildReportHTML(req.Title, req.Subtitle, summaryHTML, contentHTML, cards, tableColumns, tableRows, charts, req.Theme)
	outputFiles, _, err := writeHTMLFile(ctx, req.FileName, htmlContent)
	if err != nil {
		return err
	}

	info := fmt.Sprintf("HTML 报告已生成\n主题: %s\nKPI: %d 个\n表格: %d 行 × %d 列\n图表: %d 个", req.Theme, len(cards), len(tableRows), len(tableColumns), len(charts))
	info += assetUsageInfo(assetFileCount(req.Assets), summaryAssetCount+contentAssetCount)

	return resp.Form(&ReportGenerateResp{
		OutputFile: outputFiles,
		Info:       info,
	}).Build()
}

func parseReportTableJSON(input string) ([]string, []map[string]interface{}, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil, nil
	}

	var raw interface{}
	if err := json.Unmarshal([]byte(input), &raw); err != nil {
		return nil, nil, fmt.Errorf("附加表格 JSON 解析失败: %v", err)
	}

	var rows []map[string]interface{}
	switch value := raw.(type) {
	case []interface{}:
		for _, item := range value {
			if row, ok := item.(map[string]interface{}); ok {
				rows = append(rows, row)
			}
		}
	case map[string]interface{}:
		rows = append(rows, value)
	default:
		return nil, nil, fmt.Errorf("附加表格 JSON 必须是对象或对象数组")
	}
	if len(rows) == 0 {
		return nil, nil, nil
	}

	colSet := make(map[string]bool)
	for _, row := range rows {
		for key := range row {
			colSet[key] = true
		}
	}
	columns := make([]string, 0, len(colSet))
	for key := range colSet {
		columns = append(columns, key)
	}
	sort.Strings(columns)
	return columns, rows, nil
}

func buildReportHTML(title, subtitle, summary, contentHTML string, cards []kpiCard, tableColumns []string, tableRows []map[string]interface{}, charts []dashChart, theme string) string {
	body := strings.Builder{}
	body.WriteString(`<main class="report-shell">`)
	body.WriteString(`<section class="report-hero">`)
	body.WriteString(`<div class="hero-kicker">AgentOS HTML Report</div>`)
	body.WriteString(fmt.Sprintf(`<h1>%s</h1>`, template.HTMLEscapeString(title)))
	if strings.TrimSpace(subtitle) != "" {
		body.WriteString(fmt.Sprintf(`<p class="subtitle">%s</p>`, template.HTMLEscapeString(subtitle)))
	}
	if strings.TrimSpace(summary) != "" {
		body.WriteString(fmt.Sprintf(`<div class="summary">%s</div>`, summary))
	}
	body.WriteString(`</section>`)

	if len(cards) > 0 {
		body.WriteString(`<section class="kpi-grid">`)
		for _, card := range cards {
			body.WriteString(`<article class="kpi-card">`)
			body.WriteString(fmt.Sprintf(`<div class="kpi-name">%s</div>`, template.HTMLEscapeString(card.Name)))
			body.WriteString(fmt.Sprintf(`<div class="kpi-value">%s</div>`, template.HTMLEscapeString(card.Value)))
			if card.Desc != "" {
				body.WriteString(fmt.Sprintf(`<div class="kpi-desc">%s</div>`, template.HTMLEscapeString(card.Desc)))
			}
			body.WriteString(`</article>`)
		}
		body.WriteString(`</section>`)
	}

	body.WriteString(`<section class="report-section markdown-body">`)
	body.WriteString(contentHTML)
	body.WriteString(`</section>`)

	if len(tableRows) > 0 {
		body.WriteString(`<section class="report-section">`)
		body.WriteString(`<h2>附加数据表</h2>`)
		body.WriteString(buildReportTableHTML(tableColumns, tableRows))
		body.WriteString(`</section>`)
	}

	if len(charts) > 0 {
		body.WriteString(`<section class="report-section">`)
		body.WriteString(`<h2>数据图表</h2>`)
		body.WriteString(`<div class="chart-grid">`)
		for _, chart := range charts {
			body.WriteString(buildReportChartHTML(chart))
		}
		body.WriteString(`</div>`)
		body.WriteString(`</section>`)
	}

	body.WriteString(`</main>`)
	return htmlPageShell(title, body.String(), reportCSS(theme))
}

func renderPlainTextBlocks(text string) string {
	var parts []string
	for _, block := range strings.Split(text, "\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		parts = append(parts, "<p>"+template.HTMLEscapeString(block)+"</p>")
	}
	return strings.Join(parts, "")
}

func buildReportTableHTML(columns []string, rows []map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString(`<div class="report-table-wrap"><table class="report-table"><thead><tr>`)
	for _, col := range columns {
		sb.WriteString(fmt.Sprintf(`<th>%s</th>`, template.HTMLEscapeString(col)))
	}
	sb.WriteString(`</tr></thead><tbody>`)
	for _, row := range rows {
		sb.WriteString(`<tr>`)
		for _, col := range columns {
			sb.WriteString(fmt.Sprintf(`<td>%s</td>`, cellToString(row[col])))
		}
		sb.WriteString(`</tr>`)
	}
	sb.WriteString(`</tbody></table></div>`)
	return sb.String()
}

func buildReportChartHTML(chart dashChart) string {
	chartType := strings.ToLower(strings.TrimSpace(chart.Type))
	if chartType == "" {
		chartType = "bar"
	}
	switch chartType {
	case "pie", "doughnut", "polararea":
		return buildReportPieChartHTML(chart)
	case "line", "radar":
		return buildReportLineChartHTML(chart)
	default:
		return buildReportBarChartHTML(chart)
	}
}

func buildReportBarChartHTML(chart dashChart) string {
	maxVal := maxChartValue(chart.DataSets)
	colors := reportChartColors()
	var sb strings.Builder
	sb.WriteString(`<article class="chart-card">`)
	sb.WriteString(fmt.Sprintf(`<h3>%s</h3>`, template.HTMLEscapeString(chart.Title)))
	sb.WriteString(`<div class="bar-chart">`)
	for i, label := range chart.Labels {
		sb.WriteString(`<div class="bar-row">`)
		sb.WriteString(fmt.Sprintf(`<div class="bar-label">%s</div><div class="bar-series">`, template.HTMLEscapeString(label)))
		for j, ds := range chart.DataSets {
			if i >= len(ds.Data) {
				continue
			}
			value := ds.Data[i]
			width := math.Abs(value) / maxVal * 100
			color := colors[j%len(colors)]
			sb.WriteString(fmt.Sprintf(`<div class="bar-line"><span class="bar-name">%s</span><span class="bar-track"><span class="bar-fill" style="width:%.1f%%;background:%s"></span></span><span class="bar-num">%g</span></div>`,
				template.HTMLEscapeString(ds.Label), width, color, value))
		}
		sb.WriteString(`</div></div>`)
	}
	sb.WriteString(`</div></article>`)
	return sb.String()
}

func buildReportPieChartHTML(chart dashChart) string {
	if len(chart.DataSets) == 0 {
		return ""
	}
	colors := reportChartColors()
	data := chart.DataSets[0].Data
	total := 0.0
	for _, value := range data {
		if value > 0 {
			total += value
		}
	}
	if total <= 0 {
		return buildReportBarChartHTML(chart)
	}

	start := 0.0
	var segments []string
	for i, value := range data {
		if value <= 0 {
			continue
		}
		end := start + value/total*360
		segments = append(segments, fmt.Sprintf("%s %.2fdeg %.2fdeg", colors[i%len(colors)], start, end))
		start = end
	}

	var sb strings.Builder
	sb.WriteString(`<article class="chart-card pie-card">`)
	sb.WriteString(fmt.Sprintf(`<h3>%s</h3>`, template.HTMLEscapeString(chart.Title)))
	sb.WriteString(fmt.Sprintf(`<div class="pie-layout"><div class="pie" style="background:conic-gradient(%s)"></div><div class="pie-legend">`, strings.Join(segments, ",")))
	for i, value := range data {
		label := fmt.Sprintf("项%d", i+1)
		if i < len(chart.Labels) {
			label = chart.Labels[i]
		}
		percent := 0.0
		if total > 0 {
			percent = value / total * 100
		}
		sb.WriteString(fmt.Sprintf(`<div class="legend-row"><span class="legend-dot" style="background:%s"></span><span>%s</span><b>%g</b><em>%.1f%%</em></div>`,
			colors[i%len(colors)], template.HTMLEscapeString(label), value, percent))
	}
	sb.WriteString(`</div></div></article>`)
	return sb.String()
}

func buildReportLineChartHTML(chart dashChart) string {
	colors := reportChartColors()
	maxVal := maxChartValue(chart.DataSets)
	width := 720.0
	height := 240.0
	pad := 28.0
	var polylines []string
	var legends []string
	for i, ds := range chart.DataSets {
		if len(ds.Data) == 0 {
			continue
		}
		points := make([]string, 0, len(ds.Data))
		for j, value := range ds.Data {
			x := width / 2
			if len(ds.Data) > 1 {
				x = pad + float64(j)*(width-2*pad)/float64(len(ds.Data)-1)
			}
			y := height - pad - (value/maxVal)*(height-2*pad)
			points = append(points, fmt.Sprintf("%.1f,%.1f", x, y))
		}
		color := colors[i%len(colors)]
		polylines = append(polylines, fmt.Sprintf(`<polyline points="%s" fill="none" stroke="%s" stroke-width="4" stroke-linecap="round" stroke-linejoin="round"/>`, strings.Join(points, " "), color))
		for _, point := range points {
			xy := strings.Split(point, ",")
			if len(xy) == 2 {
				polylines = append(polylines, fmt.Sprintf(`<circle cx="%s" cy="%s" r="4" fill="%s"/>`, xy[0], xy[1], color))
			}
		}
		legends = append(legends, fmt.Sprintf(`<span><i style="background:%s"></i>%s</span>`, color, template.HTMLEscapeString(ds.Label)))
	}
	return fmt.Sprintf(`<article class="chart-card"><h3>%s</h3><div class="line-legend">%s</div><svg class="line-chart" viewBox="0 0 720 240" role="img">%s</svg></article>`,
		template.HTMLEscapeString(chart.Title), strings.Join(legends, ""), strings.Join(polylines, ""))
}

func maxChartValue(dataSets []chartDataSet) float64 {
	maxVal := 0.0
	for _, ds := range dataSets {
		for _, value := range ds.Data {
			if math.Abs(value) > maxVal {
				maxVal = math.Abs(value)
			}
		}
	}
	if maxVal <= 0 {
		return 1
	}
	return maxVal
}

func reportChartColors() []string {
	return []string{"#2563eb", "#f97316", "#16a34a", "#dc2626", "#7c3aed", "#0891b2", "#ca8a04", "#db2777"}
}

func reportCSS(theme string) string {
	vars := map[string]string{
		"bg":     "#eef4ff",
		"card":   "#ffffff",
		"text":   "#132033",
		"muted":  "#64748b",
		"accent": "#2563eb",
		"border": "#dbe7f6",
		"soft":   "#eff6ff",
	}
	switch theme {
	case "暖色报告":
		vars["bg"] = "#fff4e8"
		vars["text"] = "#2a1a12"
		vars["muted"] = "#84624a"
		vars["accent"] = "#e05d1f"
		vars["border"] = "#f1d8c1"
		vars["soft"] = "#fff1df"
	case "深色":
		vars["bg"] = "#0d1420"
		vars["card"] = "#141f30"
		vars["text"] = "#e6eef9"
		vars["muted"] = "#9aa8bd"
		vars["accent"] = "#60a5fa"
		vars["border"] = "#26364e"
		vars["soft"] = "#111b2a"
	}
	css := `
* { box-sizing: border-box; }
body { margin: 0; background: radial-gradient(circle at 12% 0, {{accent}}33, transparent 32rem), {{bg}}; color: {{text}}; font-family: "Avenir Next", "Noto Sans SC", "PingFang SC", sans-serif; }
.report-shell { width: min(1180px, calc(100% - 32px)); margin: 0 auto; padding: 42px 0 70px; }
.report-hero { border: 1px solid {{border}}; background: linear-gradient(135deg, {{card}}, {{soft}}); border-radius: 34px; padding: 46px; box-shadow: 0 28px 80px rgba(15,23,42,.14); position: relative; overflow: hidden; }
.report-hero::after { content: ""; position: absolute; right: -90px; top: -90px; width: 260px; height: 260px; border-radius: 50%; background: {{accent}}22; }
.hero-kicker { color: {{accent}}; font-size: 12px; font-weight: 800; letter-spacing: .18em; text-transform: uppercase; margin-bottom: 12px; }
h1 { font-size: clamp(2.4rem, 5vw, 5rem); line-height: .98; margin: 0; letter-spacing: -.06em; max-width: 920px; }
.subtitle { max-width: 760px; color: {{muted}}; font-size: 1.2rem; margin: 18px 0 0; }
.summary { max-width: 780px; margin-top: 28px; padding: 18px 22px; border-left: 5px solid {{accent}}; background: {{card}}cc; border-radius: 0 18px 18px 0; color: {{muted}}; }
.summary p { margin: .3em 0; }
.kpi-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); gap: 16px; margin: 22px 0; }
.kpi-card, .report-section, .chart-card { background: {{card}}; border: 1px solid {{border}}; border-radius: 24px; box-shadow: 0 18px 45px rgba(15,23,42,.08); }
.kpi-card { padding: 22px; }
.kpi-name { color: {{muted}}; font-size: .88rem; font-weight: 700; }
.kpi-value { font-size: 2rem; font-weight: 850; margin-top: 8px; letter-spacing: -.04em; }
.kpi-desc { color: {{accent}}; margin-top: 8px; font-size: .92rem; }
.report-section { padding: 34px; margin-top: 22px; }
.markdown-body { font-family: Georgia, "Noto Serif SC", serif; line-height: 1.74; }
.markdown-body h1, .markdown-body h2, .markdown-body h3 { font-family: "Avenir Next", "Noto Sans SC", sans-serif; line-height: 1.2; letter-spacing: -.03em; }
.markdown-body h2 { margin-top: 1.7em; border-top: 1px solid {{border}}; padding-top: 1em; }
.markdown-body a { color: {{accent}}; }
.markdown-body blockquote { margin: 1.2em 0; padding: .2em 1em; border-left: 4px solid {{accent}}; background: {{soft}}; border-radius: 0 14px 14px 0; color: {{muted}}; }
.markdown-body code { background: {{soft}}; border: 1px solid {{border}}; border-radius: 7px; padding: .12em .38em; }
.markdown-body pre { overflow-x: auto; background: {{soft}}; border: 1px solid {{border}}; border-radius: 16px; padding: 16px; }
.markdown-body pre code { border: 0; padding: 0; }
.markdown-body table, .report-table { width: 100%; border-collapse: collapse; font-family: "Avenir Next", "Noto Sans SC", sans-serif; }
.markdown-body th, .markdown-body td, .report-table th, .report-table td { border: 1px solid {{border}}; padding: 10px 12px; text-align: left; vertical-align: top; }
.markdown-body th, .report-table th { background: {{soft}}; }
.markdown-body img { max-width: 100%; border-radius: 18px; }
.report-table-wrap { overflow-x: auto; }
.chart-grid { display: grid; gap: 18px; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); }
.chart-card { padding: 22px; }
.chart-card h3 { margin: 0 0 18px; font-size: 1rem; }
.bar-row { display: grid; grid-template-columns: 92px 1fr; gap: 12px; margin: 13px 0; align-items: start; }
.bar-label { color: {{muted}}; font-weight: 700; font-size: .86rem; }
.bar-line { display: grid; grid-template-columns: 88px 1fr 54px; gap: 8px; align-items: center; margin: 6px 0; }
.bar-name, .bar-num { font-size: .78rem; color: {{muted}}; }
.bar-track { height: 10px; border-radius: 99px; background: {{soft}}; overflow: hidden; }
.bar-fill { display: block; height: 100%; border-radius: inherit; }
.pie-layout { display: grid; grid-template-columns: 190px 1fr; gap: 22px; align-items: center; }
.pie { width: 180px; height: 180px; border-radius: 50%; box-shadow: inset 0 0 0 18px {{card}}, 0 16px 36px rgba(15,23,42,.14); }
.legend-row { display: grid; grid-template-columns: 12px 1fr auto auto; gap: 10px; align-items: center; margin: 9px 0; color: {{muted}}; }
.legend-dot { width: 10px; height: 10px; border-radius: 50%; }
.legend-row b { color: {{text}}; }
.legend-row em { font-style: normal; font-size: .82rem; }
.line-chart { width: 100%; height: auto; background: {{soft}}; border-radius: 18px; padding: 12px; }
.line-legend { display: flex; flex-wrap: wrap; gap: 12px; margin-bottom: 10px; color: {{muted}}; font-size: .84rem; }
.line-legend i { display: inline-block; width: 10px; height: 10px; border-radius: 50%; margin-right: 6px; }
@media (max-width: 720px) { .report-shell { width: min(100% - 20px, 1180px); padding: 20px 0 36px; } .report-hero, .report-section { padding: 22px; } .pie-layout { grid-template-columns: 1fr; } .bar-row { grid-template-columns: 1fr; } }
@media print { body { background: #fff; } .report-shell { width: 100%; padding: 0; } .report-hero, .kpi-card, .report-section, .chart-card { box-shadow: none; } }
`
	replacer := strings.NewReplacer(
		"{{bg}}", vars["bg"],
		"{{card}}", vars["card"],
		"{{text}}", vars["text"],
		"{{muted}}", vars["muted"],
		"{{accent}}", vars["accent"],
		"{{border}}", vars["border"],
		"{{soft}}", vars["soft"],
	)
	return replacer.Replace(css)
}

var ReportGenerateTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "通用 HTML 报告生成",
		Desc:     `根据标题、摘要、Markdown 正文、KPI 指标、JSON 表格和图表数据生成完整 HTML 报告页。图表使用纯 HTML/CSS/SVG 渲染，不依赖外部 CDN；正文和摘要支持 {{asset:文件名}} 图片资源占位符。适合周报、复盘、调研报告、年度总结和数据分析交付。`,
		Tags:     []string{"HTML", "报告", "Markdown", "KPI", "表格", "图表", "数据分析", "网页"},
		Request:  &ReportGenerateReq{},
		Response: &ReportGenerateResp{},
	},
}

func init() {
	packageContext.POST("report_generate.form", ReportGenerate, ReportGenerateTemplate)
}
