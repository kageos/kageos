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

type DiffRenderReq struct {
	TextA    string `json:"text_a" widget:"name:原始文本;type:text_area;placeholder:输入原始文本（修改前）" validate:"required"`
	TextB    string `json:"text_b" widget:"name:对比文本;type:text_area;placeholder:输入对比文本（修改后）" validate:"required"`
	TitleA   string `json:"title_a" widget:"name:左侧标题;type:input;placeholder:例如：修改前"`
	TitleB   string `json:"title_b" widget:"name:右侧标题;type:input;placeholder:例如：修改后"`
	FileName string `json:"file_name" widget:"name:文件名;type:input;placeholder:输入文件名（无需后缀），例如 config-diff" validate:"required"`
	Theme    string `json:"theme" widget:"name:主题;type:select;options:浅色,深色;render_default:浅色" validate:"required"`
}

type DiffRenderResp struct {
	OutputFile string `json:"output_file" widget:"name:HTML 文件;type:files"`
	Info       string `json:"info" widget:"name:信息;type:text"`
}

type diffLine struct {
	Type     string // "equal", "delete", "insert"
	LineA    int
	LineB    int
	ContentA string
	ContentB string
}

func DiffRender(ctx *app.Context, resp response.Response) error {
	var req DiffRenderReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	baseName := sanitizeFileName(req.FileName)
	titleA := req.TitleA
	if titleA == "" {
		titleA = "原始文本"
	}
	titleB := req.TitleB
	if titleB == "" {
		titleB = "对比文本"
	}
	isDark := req.Theme == "深色"

	linesA := strings.Split(req.TextA, "\n")
	linesB := strings.Split(req.TextB, "\n")
	diffs := computeDiff(linesA, linesB)

	added, removed := 0, 0
	for _, d := range diffs {
		switch d.Type {
		case "insert":
			added++
		case "delete":
			removed++
		}
	}

	htmlContent := buildDiffHTML(titleA, titleB, diffs, isDark)

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

	return resp.Form(&DiffRenderResp{
		OutputFile: outputFiles,
		Info:       fmt.Sprintf("+%d 新增  -%d 删除  %d 总行", added, removed, len(diffs)),
	}).Build()
}

func computeDiff(a, b []string) []diffLine {
	m, n := len(a), len(b)
	// LCS via Myers-like DP
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	var result []diffLine
	i, j := m, n
	var stack []diffLine

	for i > 0 || j > 0 {
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
			stack = append(stack, diffLine{Type: "equal", LineA: i, LineB: j, ContentA: a[i-1], ContentB: b[j-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			stack = append(stack, diffLine{Type: "insert", LineB: j, ContentB: b[j-1]})
			j--
		} else {
			stack = append(stack, diffLine{Type: "delete", LineA: i, ContentA: a[i-1]})
			i--
		}
	}

	for k := len(stack) - 1; k >= 0; k-- {
		result = append(result, stack[k])
	}
	return result
}

func buildDiffHTML(titleA, titleB string, diffs []diffLine, isDark bool) string {
	bg := "#f8fafc"
	cardBg := "#fff"
	headerBg := "#f1f5f9"
	textColor := "#334155"
	lineNumColor := "#94a3b8"
	addBg := "#dcfce7"
	addBorder := "#86efac"
	delBg := "#fee2e2"
	delBorder := "#fca5a5"
	borderColor := "#e2e8f0"

	if isDark {
		bg = "#0f172a"
		cardBg = "#1e293b"
		headerBg = "#1e293b"
		textColor = "#e2e8f0"
		lineNumColor = "#475569"
		addBg = "rgba(34,197,94,.15)"
		addBorder = "rgba(34,197,94,.4)"
		delBg = "rgba(239,68,68,.15)"
		delBorder = "rgba(239,68,68,.4)"
		borderColor = "#334155"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Diff: %s vs %s</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:ui-monospace,SFMono-Regular,Menlo,"Noto Sans SC",monospace;background:%s;color:%s;padding:2rem;font-size:13px}
.container{max-width:1400px;margin:0 auto}
.diff-table{width:100%%;border-collapse:collapse;background:%s;border-radius:8px;overflow:hidden;box-shadow:0 1px 8px rgba(0,0,0,.08);border:1px solid %s}
.diff-header th{background:%s;padding:10px 16px;font-weight:600;font-size:14px;text-align:left;border-bottom:2px solid %s}
.diff-row td{padding:0 12px;white-space:pre-wrap;word-break:break-all;vertical-align:top;border-bottom:1px solid %s;line-height:1.7}
.ln{width:50px;text-align:right;color:%s;user-select:none;padding-right:12px;border-right:1px solid %s}
.code{min-width:200px}
.diff-add{background:%s;border-left:3px solid %s}
.diff-del{background:%s;border-left:3px solid %s}
.diff-add .code::before{content:"+";color:#16a34a;font-weight:700;margin-right:6px}
.diff-del .code::before{content:"-";color:#dc2626;font-weight:700;margin-right:6px}
</style>
</head>
<body>
<div class="container">
<table class="diff-table">
<tr class="diff-header"><th class="ln">#</th><th>%s</th><th class="ln">#</th><th>%s</th></tr>`,
		template.HTMLEscapeString(titleA), template.HTMLEscapeString(titleB),
		bg, textColor, cardBg, borderColor,
		headerBg, borderColor, borderColor,
		lineNumColor, borderColor,
		addBg, addBorder, delBg, delBorder,
		template.HTMLEscapeString(titleA), template.HTMLEscapeString(titleB),
	))

	for _, d := range diffs {
		switch d.Type {
		case "equal":
			sb.WriteString(fmt.Sprintf(`<tr class="diff-row"><td class="ln">%d</td><td class="code">%s</td><td class="ln">%d</td><td class="code">%s</td></tr>`,
				d.LineA, template.HTMLEscapeString(d.ContentA), d.LineB, template.HTMLEscapeString(d.ContentB)))
		case "delete":
			sb.WriteString(fmt.Sprintf(`<tr class="diff-row diff-del"><td class="ln">%d</td><td class="code">%s</td><td class="ln"></td><td class="code"></td></tr>`,
				d.LineA, template.HTMLEscapeString(d.ContentA)))
		case "insert":
			sb.WriteString(fmt.Sprintf(`<tr class="diff-row diff-add"><td class="ln"></td><td class="code"></td><td class="ln">%d</td><td class="code">%s</td></tr>`,
				d.LineB, template.HTMLEscapeString(d.ContentB)))
		}
	}

	sb.WriteString(`</table></div></body></html>`)
	return sb.String()
}

var DiffRenderTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "文本对比",
		Desc:     `将两段文本进行逐行对比，生成可直接访问的 Diff 对比网页。类似 GitHub 的并排 Diff 视图，绿色标注新增、红色标注删除。浅色/深色主题可选。常用于代码审查、文档版本对比、配置变更追踪等场景。`,
		Tags:     []string{"Diff", "对比", "文本对比", "代码对比", "版本对比", "差异", "HTML"},
		Request:  &DiffRenderReq{},
		Response: &DiffRenderResp{},
	},
}

func init() {
	packageContext.POST("diff_render.form", DiffRender, DiffRenderTemplate)
}
