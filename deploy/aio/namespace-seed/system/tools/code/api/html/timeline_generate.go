package html

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type TimelineGenerateReq struct {
	Title    string `json:"title" widget:"name:时间线标题;type:input;placeholder:例如：产品发布里程碑" validate:"required"`
	Events   string `json:"events" widget:"name:事件列表;type:text_area;placeholder:每行一个事件，格式：日期|标题|描述（描述可选）\n例如：\n2024-01|项目启动|团队组建完成，开始需求调研\n2024-03|Alpha 发布|核心功能完成\n2024-06|正式上线|面向全部用户开放" validate:"required"`
	FileName string `json:"file_name" widget:"name:文件名;type:input;placeholder:输入文件名（无需后缀），例如 product-roadmap" validate:"required"`
	Theme    string `json:"theme" widget:"name:主题;type:select;options:蓝色,绿色,紫色,深色;render_default:蓝色" validate:"required"`
}

type TimelineGenerateResp struct {
	OutputFile string `json:"output_file" widget:"name:HTML 文件;type:files"`
	Info       string `json:"info" widget:"name:信息;type:text"`
}

type timelineEvent struct {
	Date  string
	Title string
	Desc  string
}

func TimelineGenerate(ctx *app.Context, resp response.Response) error {
	var req TimelineGenerateReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	baseName := sanitizeFileName(req.FileName)
	events := parseTimelineEvents(req.Events)
	if len(events) == 0 {
		return fmt.Errorf("至少需要一个事件")
	}

	htmlContent := buildTimelineHTML(req.Title, events, req.Theme)

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

	return resp.Form(&TimelineGenerateResp{
		OutputFile: outputFiles,
		Info:       fmt.Sprintf("共 %d 个事件节点", len(events)),
	}).Build()
}

func parseTimelineEvents(s string) []timelineEvent {
	var events []timelineEvent
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		ev := timelineEvent{Date: strings.TrimSpace(parts[0])}
		if len(parts) >= 2 {
			ev.Title = strings.TrimSpace(parts[1])
		}
		if len(parts) >= 3 {
			ev.Desc = strings.TrimSpace(parts[2])
		}
		events = append(events, ev)
	}
	return events
}

func buildTimelineHTML(title string, events []timelineEvent, theme string) string {
	accent, accentLight, bg, cardBg, textColor, subColor := timelineThemeColors(theme)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Noto Sans SC",sans-serif;background:%s;color:%s;padding:2rem 1rem}
.container{max-width:800px;margin:0 auto}
.tl-title{text-align:center;font-size:1.6rem;font-weight:700;margin-bottom:2.5rem;color:%s}
.timeline{position:relative;padding:1rem 0}
.timeline::before{content:'';position:absolute;left:50%%;transform:translateX(-50%%);width:3px;height:100%%;background:%s;border-radius:2px}
.tl-item{position:relative;margin-bottom:2rem;display:flex;align-items:flex-start}
.tl-item:nth-child(odd){flex-direction:row-reverse}
.tl-content{width:44%%;background:%s;border-radius:10px;padding:1.2rem 1.5rem;box-shadow:0 2px 10px rgba(0,0,0,.06);border:1px solid %s;position:relative}
.tl-item:nth-child(odd) .tl-content{margin-right:6%%}
.tl-item:nth-child(even) .tl-content{margin-left:6%%}
.tl-dot{position:absolute;left:50%%;transform:translateX(-50%%);width:16px;height:16px;border-radius:50%%;background:%s;border:3px solid %s;z-index:2}
.tl-date{font-size:.82rem;color:%s;font-weight:600;margin-bottom:.3rem;letter-spacing:.5px}
.tl-event-title{font-size:1.05rem;font-weight:600;color:%s;margin-bottom:.3rem}
.tl-desc{font-size:.88rem;color:%s;line-height:1.55}
@media(max-width:640px){
.timeline::before{left:20px}
.tl-item,.tl-item:nth-child(odd){flex-direction:row}
.tl-content{width:calc(100%% - 50px) !important;margin-left:50px !important;margin-right:0 !important}
.tl-dot{left:20px}
}
@keyframes fadeUp{from{opacity:0;transform:translateY(20px)}to{opacity:1;transform:translateY(0)}}
.tl-item{animation:fadeUp .5s ease both}
`,
		template.HTMLEscapeString(title),
		bg, textColor, textColor,
		accentLight,
		cardBg, accentLight,
		accent, accentLight,
		accent, textColor, subColor,
	))

	for i := range events {
		sb.WriteString(fmt.Sprintf(`.tl-item:nth-child(%d){animation-delay:%.1fs}`, i+1, float64(i)*0.1))
	}

	sb.WriteString(fmt.Sprintf(`</style>
</head>
<body>
<div class="container">
<div class="tl-title">%s</div>
<div class="timeline">`, template.HTMLEscapeString(title)))

	for _, ev := range events {
		sb.WriteString(`<div class="tl-item"><div class="tl-dot"></div><div class="tl-content">`)
		sb.WriteString(fmt.Sprintf(`<div class="tl-date">%s</div>`, template.HTMLEscapeString(ev.Date)))
		if ev.Title != "" {
			sb.WriteString(fmt.Sprintf(`<div class="tl-event-title">%s</div>`, template.HTMLEscapeString(ev.Title)))
		}
		if ev.Desc != "" {
			sb.WriteString(fmt.Sprintf(`<div class="tl-desc">%s</div>`, template.HTMLEscapeString(ev.Desc)))
		}
		sb.WriteString(`</div></div>`)
	}

	sb.WriteString(`</div></div></body></html>`)
	return sb.String()
}

func timelineThemeColors(theme string) (accent, accentLight, bg, cardBg, textColor, subColor string) {
	switch theme {
	case "绿色":
		return "#059669", "rgba(5,150,105,.2)", "#f0fdf4", "#fff", "#1a3a2a", "#6b7280"
	case "紫色":
		return "#7c3aed", "rgba(124,58,237,.2)", "#faf5ff", "#fff", "#2e1065", "#6b7280"
	case "深色":
		return "#60a5fa", "rgba(96,165,250,.25)", "#0f172a", "#1e293b", "#e2e8f0", "#94a3b8"
	default:
		return "#2563eb", "rgba(37,99,235,.2)", "#eff6ff", "#fff", "#1e293b", "#6b7280"
	}
}

var TimelineGenerateTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "时间线生成",
		Desc:     `根据事件列表生成可直接访问的交互式时间线网页。左右交替排列，带入场动画。支持多种主题配色。常用于项目里程碑、产品发布记录、历史事件梳理、个人履历展示等场景。`,
		Tags:     []string{"时间线", "Timeline", "里程碑", "路线图", "事件", "历史", "HTML"},
		Request:  &TimelineGenerateReq{},
		Response: &TimelineGenerateResp{},
	},
}

func init() {
	packageContext.POST("timeline_generate.form", TimelineGenerate, TimelineGenerateTemplate)
}
