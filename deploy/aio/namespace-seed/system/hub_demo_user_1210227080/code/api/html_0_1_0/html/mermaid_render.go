package html

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type MermaidRenderReq struct {
	MermaidCode string `json:"mermaid_code" widget:"name:Mermaid 代码;type:text_area;placeholder:输入 Mermaid 语法，例如：\ngraph TD\n  A[开始] --> B{判断}\n  B -->|是| C[执行]\n  B -->|否| D[结束]" validate:"required"`
	FileName    string `json:"file_name" widget:"name:文件名;type:input;placeholder:输入文件名（无需后缀），例如 system-architecture" validate:"required"`
	Title       string `json:"title" widget:"name:页面标题;type:input;placeholder:可选，显示在图表上方"`
	Theme       string `json:"theme" widget:"name:主题;type:select;options:default 默认,dark 深色,forest 森林,neutral 素雅;render_default:default 默认" validate:"required"`
}

type MermaidRenderResp struct {
	OutputFile string `json:"output_file" widget:"name:HTML 文件;type:files"`
	Info       string `json:"info" widget:"name:信息;type:text"`
}

func MermaidRender(ctx *app.Context, resp response.Response) error {
	var req MermaidRenderReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	baseName := sanitizeFileName(req.FileName)
	theme := parseMermaidTheme(req.Theme)
	title := req.Title

	htmlContent := buildMermaidHTML(title, req.MermaidCode, theme)

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

	return resp.Form(&MermaidRenderResp{
		OutputFile: outputFiles,
		Info:       fmt.Sprintf("Mermaid 图表已生成 | 主题: %s | 代码 %d 字符", theme, len(req.MermaidCode)),
	}).Build()
}

func parseMermaidTheme(t string) string {
	switch t {
	case "dark 深色":
		return "dark"
	case "forest 森林":
		return "forest"
	case "neutral 素雅":
		return "neutral"
	default:
		return "default"
	}
}

func buildMermaidHTML(title, mermaidCode, theme string) string {
	pageTitle := title
	if pageTitle == "" {
		pageTitle = "Mermaid Diagram"
	}

	var titleHTML string
	if title != "" {
		titleHTML = fmt.Sprintf(`<h1 class="title">%s</h1>`, template.HTMLEscapeString(title))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans SC", sans-serif; background: %s; display: flex; flex-direction: column; align-items: center; min-height: 100vh; padding: 2rem; }
.title { font-size: 1.4rem; color: %s; margin-bottom: 1.5rem; font-weight: 600; }
.mermaid-wrap { background: %s; border-radius: 12px; box-shadow: 0 2px 16px rgba(0,0,0,.08); padding: 2rem; max-width: 100%%; overflow-x: auto; }
.mermaid { display: flex; justify-content: center; }
</style>
</head>
<body>
%s
<div class="mermaid-wrap">
<pre class="mermaid">
%s
</pre>
</div>
<script type="module">
import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
mermaid.initialize({ startOnLoad: true, theme: '%s', securityLevel: 'loose' });
</script>
</body>
</html>`,
		template.HTMLEscapeString(pageTitle),
		mermaidBgColor(theme),
		mermaidTitleColor(theme),
		mermaidCardBg(theme),
		titleHTML,
		template.HTMLEscapeString(mermaidCode),
		theme,
	)
}

func mermaidBgColor(theme string) string {
	if theme == "dark" {
		return "#1a1a2e"
	}
	return "#f8fafc"
}

func mermaidTitleColor(theme string) string {
	if theme == "dark" {
		return "#e0e0e0"
	}
	return "#334155"
}

func mermaidCardBg(theme string) string {
	if theme == "dark" {
		return "#16213e"
	}
	return "#fff"
}

var MermaidRenderTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "Mermaid 图表渲染",
		Desc:     `将 Mermaid 语法渲染为可直接访问的交互式图表网页。支持流程图、时序图、甘特图、思维导图、类图、状态图、ER 图等所有 Mermaid 图表类型。多种主题可选。常用于架构设计、流程梳理、技术文档等场景。`,
		Tags:     []string{"Mermaid", "流程图", "时序图", "甘特图", "思维导图", "类图", "图表", "可视化", "HTML"},
		Request:  &MermaidRenderReq{},
		Response: &MermaidRenderResp{},
	},
}

func init() {
	packageContext.POST("mermaid_render.form", MermaidRender, MermaidRenderTemplate)
}
