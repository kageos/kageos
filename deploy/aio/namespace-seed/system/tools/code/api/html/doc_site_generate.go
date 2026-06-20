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

type DocSiteGenerateReq struct {
	SiteTitle string `json:"site_title" widget:"name:站点标题;type:input;placeholder:例如：项目文档" validate:"required"`
	Pages     string `json:"pages" widget:"name:页面内容;type:text_area;placeholder:用 === 分隔多个页面，每页格式：\n页面标题\n---\n正文内容（支持简单 Markdown）\n\n例如：\n快速开始\n---\n## 安装\n执行 npm install\n## 运行\nnpm run dev\n===\nAPI 文档\n---\n### 接口列表\nGET /api/users" validate:"required"`
	FileName  string `json:"file_name" widget:"name:文件名;type:input;placeholder:输入文件名（无需后缀），例如 project-docs" validate:"required"`
	Theme     string `json:"theme" widget:"name:主题;type:select;options:浅色,深色;render_default:浅色" validate:"required"`
}

type DocSiteGenerateResp struct {
	OutputFile string `json:"output_file" widget:"name:HTML 文件;type:files"`
	Info       string `json:"info" widget:"name:信息;type:text"`
}

type docPage struct {
	ID    string
	Title string
	Body  string
}

func DocSiteGenerate(ctx *app.Context, resp response.Response) error {
	var req DocSiteGenerateReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	baseName := sanitizeFileName(req.FileName)
	pages := parseDocSitePages(req.Pages)
	if len(pages) == 0 {
		return fmt.Errorf("至少需要一个页面")
	}

	for i := range pages {
		pages[i].ID = fmt.Sprintf("page-%d", i+1)
		pages[i].Body = simpleMarkdownToHTML(pages[i].Body)
	}

	isDark := req.Theme == "深色"
	htmlContent := buildDocSiteHTML(req.SiteTitle, pages, isDark)

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

	return resp.Form(&DocSiteGenerateResp{
		OutputFile: outputFiles,
		Info:       fmt.Sprintf("%d 个页面", len(pages)),
	}).Build()
}

func parseDocSitePages(s string) []docPage {
	var pages []docPage
	blocks := strings.Split(s, "===")
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		parts := strings.SplitN(block, "\n---\n", 2)
		title := strings.TrimSpace(parts[0])
		body := ""
		if len(parts) >= 2 {
			body = strings.TrimSpace(parts[1])
		}
		if title != "" {
			pages = append(pages, docPage{Title: title, Body: body})
		}
	}
	return pages
}

func simpleMarkdownToHTML(md string) string {
	lines := strings.Split(md, "\n")
	var out []string
	inCode := false
	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inCode {
				out = append(out, "</code></pre>")
				inCode = false
			} else {
				lang := strings.TrimSpace(strings.TrimPrefix(line, "```"))
				if lang != "" {
					out = append(out, "<pre><code class=\"language-"+template.HTMLEscapeString(lang)+"\">")
				} else {
					out = append(out, "<pre><code>")
				}
				inCode = true
			}
			continue
		}
		if inCode {
			out = append(out, template.HTMLEscapeString(line))
			continue
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "### ") {
			out = append(out, "<h3>"+template.HTMLEscapeString(line[4:])+"</h3>")
		} else if strings.HasPrefix(line, "## ") {
			out = append(out, "<h2>"+template.HTMLEscapeString(line[3:])+"</h2>")
		} else if strings.HasPrefix(line, "# ") {
			out = append(out, "<h1>"+template.HTMLEscapeString(line[2:])+"</h1>")
		} else if line == "" {
			out = append(out, "")
		} else {
			out = append(out, "<p>"+template.HTMLEscapeString(line)+"</p>")
		}
	}
	if inCode {
		out = append(out, "</code></pre>")
	}
	return strings.Join(out, "\n")
}

func buildDocSiteHTML(siteTitle string, pages []docPage, isDark bool) string {
	bg, sidebarBg, contentBg, textColor, linkColor, borderColor := docSiteTheme(isDark)

	var navSb, contentSb strings.Builder
	for i, p := range pages {
		navSb.WriteString(fmt.Sprintf(`<a href="#%s" class="nav-item" data-page="%s">%s</a>`, p.ID, p.ID, template.HTMLEscapeString(p.Title)))
		contentSb.WriteString(fmt.Sprintf(`<div id="%s" class="doc-page"><div class="doc-content">%s</div></div>`, p.ID, p.Body))
		if i < len(pages)-1 {
			navSb.WriteString("\n")
		}
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Noto Sans SC",sans-serif;background:%s;color:%s;display:flex;min-height:100vh}
.sidebar{width:260px;background:%s;border-right:1px solid %s;padding:1.5rem;flex-shrink:0}
.sidebar h2{font-size:1rem;font-weight:600;margin-bottom:1rem;color:%s}
.nav-item{display:block;padding:8px 12px;border-radius:6px;color:%s;text-decoration:none;margin-bottom:4px;transition:background .15s}
.nav-item:hover,.nav-item.active{background:rgba(128,128,128,.15)}
.main{flex:1;padding:2rem 3rem;overflow-y:auto;background:%s}
.doc-page{display:none}
.doc-page.active{display:block}
.doc-content h1{font-size:1.5rem;margin-bottom:1rem;padding-bottom:.5rem;border-bottom:1px solid %s}
.doc-content h2{font-size:1.2rem;margin:1.5rem 0 .5rem}
.doc-content h3{font-size:1.05rem;margin:1rem 0 .4rem}
.doc-content p{margin:.6rem 0;line-height:1.7}
.doc-content pre{background:rgba(0,0,0,.06);padding:1rem;border-radius:8px;overflow-x:auto;margin:1rem 0;font-size:.9rem}
.doc-content code{background:rgba(0,0,0,.06);padding:2px 6px;border-radius:4px;font-size:.88em}
.doc-content pre code{background:none;padding:0}
@media(max-width:768px){.sidebar{width:100%%;border-right:none;border-bottom:1px solid %s}.main{padding:1.5rem}}
</style>
</head>
<body>
<aside class="sidebar"><h2>%s</h2><nav>%s</nav></aside>
<main class="main">%s</main>
<script>
document.querySelectorAll('.nav-item').forEach(el=>{
  el.addEventListener('click',e=>{e.preventDefault();
    document.querySelectorAll('.nav-item').forEach(x=>x.classList.remove('active'));
    document.querySelectorAll('.doc-page').forEach(x=>x.classList.remove('active'));
    el.classList.add('active');
    document.getElementById(el.dataset.page).classList.add('active');
  });
});
document.querySelector('.nav-item').click();
</script>
</body>
</html>`,
		template.HTMLEscapeString(siteTitle),
		bg, textColor, sidebarBg, borderColor, textColor, linkColor, contentBg, borderColor, borderColor,
		template.HTMLEscapeString(siteTitle), navSb.String(), contentSb.String(),
	)
}

func docSiteTheme(isDark bool) (bg, sidebarBg, contentBg, textColor, linkColor, borderColor string) {
	if isDark {
		return "#0f172a", "#1e293b", "#0f172a", "#e2e8f0", "#93c5fd", "#334155"
	}
	return "#f8fafc", "#fff", "#fff", "#1e293b", "#2563eb", "#e2e8f0"
}

var DocSiteGenerateTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "多页文档站点生成",
		Desc:     `根据多页内容生成带侧边栏导航的文档站点 HTML。用 === 分隔页面，每页用 --- 分隔标题和正文，支持简单 Markdown。单文件输出，可直接访问。常用于项目文档、知识库、产品说明等场景。`,
		Tags:     []string{"文档", "文档站", "多页", "知识库", "项目文档", "HTML"},
		Request:  &DocSiteGenerateReq{},
		Response: &DocSiteGenerateResp{},
	},
}

func init() {
	packageContext.POST("doc_site_generate.form", DocSiteGenerate, DocSiteGenerateTemplate)
}
