package html

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type KanbanGenerateReq struct {
	Title    string `json:"title" widget:"name:看板标题;type:input;placeholder:例如：Sprint 12 任务看板" validate:"required"`
	Columns  string `json:"columns" widget:"name:看板数据;type:text_area;placeholder:列名用 === 分隔，每列下方每行一个卡片\n格式：卡片标题|描述|标签（描述和标签可选）\n\n例如：\n待办\n重构用户模块|优化代码结构|高优先\n修复登录BUG|偶发 token 过期\n===\n进行中\n新增搜索功能|已完成 70%|开发中\n===\n已完成\n部署 CI/CD|流水线配置完毕|运维" validate:"required"`
	FileName string `json:"file_name" widget:"name:文件名;type:input;placeholder:输入文件名（无需后缀），例如 sprint-12-board" validate:"required"`
	Theme    string `json:"theme" widget:"name:主题;type:select;options:浅色,深色;render_default:浅色" validate:"required"`
}

type KanbanGenerateResp struct {
	OutputFile string `json:"output_file" widget:"name:HTML 文件;type:files"`
	Info       string `json:"info" widget:"name:信息;type:text"`
}

type kanbanColumn struct {
	Name  string
	Cards []kanbanCard
}

type kanbanCard struct {
	Title string
	Desc  string
	Tag   string
}

func KanbanGenerate(ctx *app.Context, resp response.Response) error {
	var req KanbanGenerateReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	baseName := sanitizeFileName(req.FileName)
	columns := parseKanbanColumns(req.Columns)
	if len(columns) == 0 {
		return fmt.Errorf("至少需要一列")
	}

	isDark := req.Theme == "深色"
	htmlContent := buildKanbanHTML(req.Title, columns, isDark)

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

	totalCards := 0
	for _, col := range columns {
		totalCards += len(col.Cards)
	}

	return resp.Form(&KanbanGenerateResp{
		OutputFile: outputFiles,
		Info:       fmt.Sprintf("%d 列 × %d 张卡片", len(columns), totalCards),
	}).Build()
}

func parseKanbanColumns(s string) []kanbanColumn {
	var columns []kanbanColumn
	blocks := strings.Split(s, "===")

	for _, block := range blocks {
		lines := parseLines(strings.TrimSpace(block))
		if len(lines) == 0 {
			continue
		}

		col := kanbanColumn{Name: lines[0]}
		for _, line := range lines[1:] {
			parts := strings.SplitN(line, "|", 3)
			card := kanbanCard{Title: strings.TrimSpace(parts[0])}
			if len(parts) >= 2 {
				card.Desc = strings.TrimSpace(parts[1])
			}
			if len(parts) >= 3 {
				card.Tag = strings.TrimSpace(parts[2])
			}
			col.Cards = append(col.Cards, card)
		}
		columns = append(columns, col)
	}
	return columns
}

var kanbanTagColors = []struct{ bg, text string }{
	{"#dbeafe", "#1d4ed8"},
	{"#dcfce7", "#15803d"},
	{"#fef3c7", "#92400e"},
	{"#fce7f3", "#be185d"},
	{"#e0e7ff", "#4338ca"},
	{"#ccfbf1", "#0f766e"},
	{"#fee2e2", "#b91c1c"},
	{"#f3e8ff", "#7e22ce"},
}

var kanbanTagColorsDark = []struct{ bg, text string }{
	{"rgba(59,130,246,.2)", "#93c5fd"},
	{"rgba(34,197,94,.2)", "#86efac"},
	{"rgba(245,158,11,.2)", "#fcd34d"},
	{"rgba(236,72,153,.2)", "#f9a8d4"},
	{"rgba(99,102,241,.2)", "#a5b4fc"},
	{"rgba(20,184,166,.2)", "#5eead4"},
	{"rgba(239,68,68,.2)", "#fca5a5"},
	{"rgba(168,85,247,.2)", "#d8b4fe"},
}

func buildKanbanHTML(title string, columns []kanbanColumn, isDark bool) string {
	bg := "#f0f4f8"
	colBg := "#e2e8f0"
	cardBg := "#fff"
	titleColor := "#0f172a"
	colTitleColor := "#334155"
	cardTitleColor := "#1e293b"
	descColor := "#64748b"
	countColor := "#94a3b8"
	tagColors := kanbanTagColors

	if isDark {
		bg = "#0f172a"
		colBg = "#1e293b"
		cardBg = "#334155"
		titleColor = "#f1f5f9"
		colTitleColor = "#e2e8f0"
		cardTitleColor = "#f1f5f9"
		descColor = "#94a3b8"
		countColor = "#64748b"
		tagColors = kanbanTagColorsDark
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Noto Sans SC",sans-serif;background:%s;min-height:100vh;padding:2rem}
.board-title{font-size:1.5rem;font-weight:700;color:%s;margin-bottom:1.5rem;text-align:center}
.board{display:flex;gap:1.2rem;overflow-x:auto;padding-bottom:1rem;align-items:flex-start}
.column{background:%s;border-radius:10px;min-width:280px;max-width:320px;flex-shrink:0;padding:1rem}
.col-header{display:flex;align-items:center;justify-content:space-between;margin-bottom:.8rem;padding:0 .2rem}
.col-name{font-weight:600;font-size:.95rem;color:%s}
.col-count{font-size:.78rem;color:%s;background:rgba(148,163,184,.15);padding:2px 8px;border-radius:10px}
.card{background:%s;border-radius:8px;padding:.9rem 1rem;margin-bottom:.6rem;box-shadow:0 1px 3px rgba(0,0,0,.06);transition:transform .15s,box-shadow .15s;cursor:default}
.card:hover{transform:translateY(-2px);box-shadow:0 4px 12px rgba(0,0,0,.1)}
.card-title{font-weight:600;font-size:.9rem;color:%s;margin-bottom:.25rem}
.card-desc{font-size:.8rem;color:%s;line-height:1.45}
.card-tag{display:inline-block;font-size:.72rem;padding:2px 8px;border-radius:4px;margin-top:.5rem;font-weight:500}
</style>
</head>
<body>
<div class="board-title">%s</div>
<div class="board">`,
		template.HTMLEscapeString(title),
		bg, titleColor, colBg, colTitleColor, countColor, cardBg, cardTitleColor, descColor,
		template.HTMLEscapeString(title),
	))

	tagColorIdx := 0
	tagColorMap := make(map[string]int)

	for _, col := range columns {
		sb.WriteString(fmt.Sprintf(`<div class="column"><div class="col-header"><span class="col-name">%s</span><span class="col-count">%d</span></div>`,
			template.HTMLEscapeString(col.Name), len(col.Cards)))

		for _, card := range col.Cards {
			sb.WriteString(fmt.Sprintf(`<div class="card"><div class="card-title">%s</div>`, template.HTMLEscapeString(card.Title)))
			if card.Desc != "" {
				sb.WriteString(fmt.Sprintf(`<div class="card-desc">%s</div>`, template.HTMLEscapeString(card.Desc)))
			}
			if card.Tag != "" {
				idx, ok := tagColorMap[card.Tag]
				if !ok {
					idx = tagColorIdx % len(tagColors)
					tagColorMap[card.Tag] = idx
					tagColorIdx++
				}
				tc := tagColors[idx]
				sb.WriteString(fmt.Sprintf(`<span class="card-tag" style="background:%s;color:%s">%s</span>`,
					tc.bg, tc.text, template.HTMLEscapeString(card.Tag)))
			}
			sb.WriteString(`</div>`)
		}

		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</div></body></html>`)
	return sb.String()
}

var KanbanGenerateTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "看板生成",
		Desc:     `生成可直接访问的 Trello 风格看板网页。支持多列分组、卡片标签、浅色/深色主题。同一标签自动统一颜色，卡片带悬停效果。常用于任务管理可视化、项目进度展示、Sprint 计划、团队协作看板等场景。`,
		Tags:     []string{"看板", "Kanban", "任务管理", "卡片", "项目管理", "Trello", "敏捷", "HTML"},
		Request:  &KanbanGenerateReq{},
		Response: &KanbanGenerateResp{},
	},
}

func init() {
	packageContext.POST("kanban_generate.form", KanbanGenerate, KanbanGenerateTemplate)
}
