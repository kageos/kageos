package html

import (
	"fmt"
	"html/template"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type MarkdownToHTMLReq struct {
	Markdown string `json:"markdown" widget:"name:Markdown 内容;type:text_area;placeholder:支持标题、表格、列表、引用、代码块、任务列表等 GFM 语法" validate:"required"`
	Title    string `json:"title" widget:"name:页面标题;type:input;placeholder:例如：项目复盘报告"`
	FileName string `json:"file_name" widget:"name:文件名;type:input;placeholder:输入文件名（无需后缀），例如 project-review" validate:"required"`
	Theme    string `json:"theme" widget:"name:主题;type:select;options:阅读白,商务蓝,深色;options_colors:909399,409EFF,909399;render_default:阅读白" validate:"required,oneof=阅读白 商务蓝 深色"`
	Assets   string `json:"assets" widget:"name:图片资源;type:files;accept:image/*,.svg,.webp,.png,.jpg,.jpeg,.gif;max_size:20MB;max_count:30"`
}

type MarkdownToHTMLResp struct {
	OutputFile string `json:"output_file" widget:"name:HTML 文件;type:files"`
	Info       string `json:"info" widget:"name:生成信息;type:text_area"`
}

func MarkdownToHTML(ctx *app.Context, resp response.Response) error {
	var req MarkdownToHTMLReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	fs := ctx.GetFS()
	downloadedAssets, assetURIs, err := downloadAssetDataURIs(ctx, req.Assets)
	if err != nil {
		return err
	}
	defer fs.RemoveFiles(downloadedAssets)

	markdownText, usedAssets := embedAssetPlaceholders(req.Markdown, assetURIs)
	bodyHTML, err := markdownToHTMLFragment(markdownText)
	if err != nil {
		return err
	}

	title := req.Title
	if title == "" {
		title = sanitizeFileName(req.FileName)
	}
	htmlContent := buildMarkdownDocumentHTML(title, bodyHTML, req.Theme)
	outputFiles, _, err := writeHTMLFile(ctx, req.FileName, htmlContent)
	if err != nil {
		return err
	}

	info := fmt.Sprintf("Markdown 已渲染为 HTML\n标题: %s\n主题: %s\n内容长度: %d 字符", title, req.Theme, len(markdownText))
	info += assetUsageInfo(assetFileCount(req.Assets), usedAssets)

	return resp.Form(&MarkdownToHTMLResp{
		OutputFile: outputFiles,
		Info:       info,
	}).Build()
}

func buildMarkdownDocumentHTML(title string, bodyHTML string, theme string) string {
	isDark := theme == "深色"
	accent := "#2563eb"
	if theme == "商务蓝" {
		accent = "#0f4c81"
	}
	if isDark {
		accent = "#60a5fa"
	}
	style := markdownDocumentCSS(isDark, accent)
	body := fmt.Sprintf(`<main class="doc-shell">
<article class="markdown-doc">
<header class="doc-hero">
<p class="eyebrow">Markdown Document</p>
<h1>%s</h1>
</header>
<section class="doc-content">%s</section>
</article>
</main>`, template.HTMLEscapeString(title), bodyHTML)
	return htmlPageShell(title, body, style)
}

func markdownDocumentCSS(isDark bool, accent string) string {
	bg := "#f6f2ea"
	card := "#fffdf8"
	text := "#26221d"
	muted := "#746b60"
	border := "#e7ded1"
	codeBg := "#f3eadf"
	if isDark {
		bg = "#10151f"
		card = "#151d2b"
		text = "#e7edf7"
		muted = "#9aa7bd"
		border = "#263246"
		codeBg = "#0d1320"
	}
	return fmt.Sprintf(`
* { box-sizing: border-box; }
body { margin: 0; background: radial-gradient(circle at top left, %s22, transparent 34rem), %s; color: %s; font-family: Georgia, "Noto Serif SC", "Songti SC", serif; line-height: 1.72; }
.doc-shell { width: min(1080px, calc(100%% - 32px)); margin: 0 auto; padding: 56px 0 72px; }
.markdown-doc { background: %s; border: 1px solid %s; border-radius: 28px; box-shadow: 0 24px 70px rgba(20,24,30,.14); overflow: hidden; }
.doc-hero { padding: 48px 56px 34px; border-bottom: 1px solid %s; background: linear-gradient(135deg, %s18, transparent); }
.eyebrow { margin: 0 0 10px; color: %s; text-transform: uppercase; letter-spacing: .16em; font: 700 12px "Avenir Next", "Noto Sans SC", sans-serif; }
h1 { margin: 0; font-size: clamp(2rem, 4vw, 4rem); line-height: 1.08; letter-spacing: -.04em; }
.doc-content { padding: 36px 56px 56px; }
.doc-content h1, .doc-content h2, .doc-content h3 { font-family: "Avenir Next", "Noto Sans SC", sans-serif; line-height: 1.22; margin: 1.6em 0 .65em; letter-spacing: -.025em; }
.doc-content h2 { border-top: 1px solid %s; padding-top: 1.2em; }
.doc-content p { margin: .75em 0; }
.doc-content a { color: %s; text-decoration-thickness: 2px; text-underline-offset: 3px; }
.doc-content blockquote { margin: 1.4em 0; padding: 1px 1.2em; border-left: 4px solid %s; background: %s11; color: %s; border-radius: 0 14px 14px 0; }
.doc-content code { background: %s; border: 1px solid %s; padding: .12em .38em; border-radius: 7px; font-family: "SFMono-Regular", Consolas, monospace; font-size: .92em; }
.doc-content pre { background: %s; border: 1px solid %s; border-radius: 16px; padding: 18px; overflow-x: auto; }
.doc-content pre code { background: transparent; border: 0; padding: 0; }
.doc-content table { width: 100%%; border-collapse: collapse; margin: 1.3em 0; font-family: "Avenir Next", "Noto Sans SC", sans-serif; font-size: .95rem; }
.doc-content th, .doc-content td { border: 1px solid %s; padding: 10px 12px; text-align: left; vertical-align: top; }
.doc-content th { background: %s18; }
.doc-content img { max-width: 100%%; border-radius: 16px; display: block; margin: 1.2em auto; }
@media (max-width: 720px) { .doc-shell { width: min(100%% - 20px, 1080px); padding: 20px 0 36px; } .doc-hero, .doc-content { padding-left: 22px; padding-right: 22px; } }
@media print { body { background: #fff; } .doc-shell { width: 100%%; padding: 0; } .markdown-doc { box-shadow: none; border: 0; border-radius: 0; } }
`, accent, bg, text, card, border, border, accent, muted, border, accent, accent, accent, muted, codeBg, border, codeBg, border, border, accent)
}

var MarkdownToHTMLTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "Markdown 转 HTML 文档",
		Desc:     `将 Markdown/GFM 内容渲染为可直接访问的漂亮 HTML 文档，支持表格、任务列表、代码块、引用、链接和图片资源占位符 {{asset:文件名}}。适合把模型输出的长报告、说明书、会议纪要、项目文档快速变成可分享页面。`,
		Tags:     []string{"Markdown", "HTML", "文档", "报告", "GFM", "网页", "资源内嵌"},
		Request:  &MarkdownToHTMLReq{},
		Response: &MarkdownToHTMLResp{},
	},
}

func init() {
	packageContext.POST("markdown_to_html.form", MarkdownToHTML, MarkdownToHTMLTemplate)
}
