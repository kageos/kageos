package html

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type CalendarGenerateReq struct {
	Year     int    `json:"year" widget:"name:年份;type:integer;min:2000;max:2100;render_default:2024" validate:"required"`
	Month    int    `json:"month" widget:"name:月份;type:integer;min:1;max:12;render_default:3" validate:"required"`
	Title    string `json:"title" widget:"name:日历标题;type:input;placeholder:例如：2024年3月日程"`
	Events   string `json:"events" widget:"name:事件列表;type:text_area;placeholder:每行一个，格式：日期|事件内容\n例如：\n5|项目评审会\n12|产品上线\n18|团队聚餐\n25|月度总结"`
	FileName string `json:"file_name" widget:"name:文件名;type:input;placeholder:输入文件名（无需后缀），例如 2024-03-schedule" validate:"required"`
	Theme    string `json:"theme" widget:"name:主题;type:select;options:浅色,深色,绿色;render_default:浅色" validate:"required"`
}

type CalendarGenerateResp struct {
	OutputFile string `json:"output_file" widget:"name:HTML 文件;type:files"`
	Info       string `json:"info" widget:"name:信息;type:text"`
}

type calEvent struct {
	Day  int
	Text string
}

func CalendarGenerate(ctx *app.Context, resp response.Response) error {
	var req CalendarGenerateReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	if req.Month < 1 || req.Month > 12 {
		return fmt.Errorf("月份必须在 1-12 之间")
	}

	baseName := sanitizeFileName(req.FileName)
	events := parseCalendarEvents(req.Events)

	firstDay := time.Date(req.Year, time.Month(req.Month), 1, 0, 0, 0, 0, time.Local)
	lastDay := firstDay.AddDate(0, 1, -1)
	daysInMonth := lastDay.Day()
	weekdayOfFirst := int(firstDay.Weekday())
	if weekdayOfFirst == 0 {
		weekdayOfFirst = 7
	}

	eventMap := make(map[int][]string)
	for _, e := range events {
		if e.Day >= 1 && e.Day <= daysInMonth {
			eventMap[e.Day] = append(eventMap[e.Day], e.Text)
		}
	}

	title := req.Title
	if title == "" {
		title = fmt.Sprintf("%d年%d月", req.Year, req.Month)
	}

	monthNames := []string{"", "一月", "二月", "三月", "四月", "五月", "六月", "七月", "八月", "九月", "十月", "十一月", "十二月"}
	monthName := monthNames[req.Month]

	htmlContent := buildCalendarHTML(title, req.Year, req.Month, monthName, daysInMonth, weekdayOfFirst, eventMap, req.Theme)

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

	return resp.Form(&CalendarGenerateResp{
		OutputFile: outputFiles,
		Info:       fmt.Sprintf("%d年%d月 | %d 天 | %d 个事件", req.Year, req.Month, daysInMonth, len(events)),
	}).Build()
}

func parseCalendarEvents(s string) []calEvent {
	var list []calEvent
	for _, line := range parseLines(s) {
		parts := strings.SplitN(line, "|", 2)
		if len(parts) < 2 {
			continue
		}
		day, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || day < 1 || day > 31 {
			continue
		}
		text := strings.TrimSpace(parts[1])
		if text != "" {
			list = append(list, calEvent{Day: day, Text: text})
		}
	}
	return list
}

func buildCalendarHTML(title string, year, month int, monthName string, daysInMonth, weekdayOfFirst int, eventMap map[int][]string, theme string) string {
	bg, headerBg, cellBg, todayBg, eventBg, textColor, subColor, borderColor := calendarThemeColors(theme)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Noto Sans SC",sans-serif;background:%s;color:%s;padding:2rem}
.cal{max-width:900px;margin:0 auto;background:%s;border-radius:12px;box-shadow:0 4px 24px rgba(0,0,0,.08);overflow:hidden}
.cal-header{background:%s;color:#fff;padding:1.5rem;text-align:center;font-size:1.4rem;font-weight:600}
.cal-weekdays{display:grid;grid-template-columns:repeat(7,1fr);background:%s;font-size:.85rem;font-weight:600;color:%s}
.cal-weekdays div{padding:10px;text-align:center;border-bottom:1px solid %s}
.cal-grid{display:grid;grid-template-columns:repeat(7,1fr);min-height:400px}
.cal-day{min-height:100px;padding:8px;border:1px solid %s;background:%s;font-size:.9rem}
.cal-day.empty{background:%s;opacity:.5}
.cal-day-num{font-weight:600;color:%s;margin-bottom:4px;font-size:.95rem}
.cal-day.today{background:%s}
.cal-day.today .cal-day-num{color:%s}
.cal-events{font-size:.78rem;color:%s}
.cal-event{background:%s;padding:2px 6px;border-radius:4px;margin-bottom:2px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
@media print{.cal{box-shadow:none}body{padding:0}}
</style>
</head>
<body>
<div class="cal">
<div class="cal-header">%s</div>
<div class="cal-weekdays"><div>一</div><div>二</div><div>三</div><div>四</div><div>五</div><div>六</div><div>日</div></div>
<div class="cal-grid">`,
		template.HTMLEscapeString(title),
		bg, textColor, cellBg, headerBg, cellBg, subColor, borderColor,
		borderColor, cellBg, bg, textColor, todayBg, "#fff", subColor, eventBg,
		template.HTMLEscapeString(title),
	))

	today := time.Now()
	isToday := func(day int) bool {
		return today.Year() == year && int(today.Month()) == month && today.Day() == day
	}

	emptyCells := weekdayOfFirst - 1
	for i := 0; i < emptyCells; i++ {
		sb.WriteString(`<div class="cal-day empty"></div>`)
	}

	for d := 1; d <= daysInMonth; d++ {
		dayClass := "cal-day"
		if isToday(d) {
			dayClass += " today"
		}
		sb.WriteString(fmt.Sprintf(`<div class="%s"><div class="cal-day-num">%d</div><div class="cal-events">`, dayClass, d))
		for _, ev := range eventMap[d] {
			sb.WriteString(fmt.Sprintf(`<div class="cal-event">%s</div>`, template.HTMLEscapeString(ev)))
		}
		sb.WriteString(`</div></div>`)
	}

	sb.WriteString(`</div></div></body></html>`)
	return sb.String()
}

func calendarThemeColors(theme string) (bg, headerBg, cellBg, todayBg, eventBg, textColor, subColor, borderColor string) {
	switch theme {
	case "深色":
		return "#0f172a", "#1e293b", "#1e293b", "#334155", "rgba(96,165,250,.25)", "#e2e8f0", "#94a3b8", "#334155"
	case "绿色":
		return "#f0fdf4", "#059669", "#fff", "#dcfce7", "#bbf7d0", "#1a3a2a", "#6b7280", "#e2e8f0"
	default:
		return "#f8fafc", "#475569", "#fff", "#e0f2fe", "#e0f2fe", "#1e293b", "#64748b", "#e2e8f0"
	}
}

var CalendarGenerateTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "日历生成",
		Desc:     `根据年份、月份和事件列表生成可直接访问的月历 HTML 页面。事件标注在对应日期上，支持浅色/深色/绿色主题，可打印。常用于排期、会议安排、课程表、日程管理等场景。`,
		Tags:     []string{"日历", "Calendar", "日程", "月历", "排期", "课程表", "HTML"},
		Request:  &CalendarGenerateReq{},
		Response: &CalendarGenerateResp{},
	},
}

func init() {
	packageContext.POST("calendar_generate.form", CalendarGenerate, CalendarGenerateTemplate)
}
