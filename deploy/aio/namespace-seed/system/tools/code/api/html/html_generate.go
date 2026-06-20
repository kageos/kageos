package html

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type HTMLGenerateReq struct {
	HTMLCode string `json:"html_code" widget:"name:HTML 代码;type:text_area;placeholder:输入完整的 HTML 代码（包含 html/head/body），或只输入 body 内容" validate:"required"`
	FileName string `json:"file_name" widget:"name:文件名;type:input;placeholder:输入文件名（无需后缀），例如 my-page" validate:"required"`
	WrapPage string `json:"wrap_page" widget:"name:自动包装;type:select;options:自动检测,强制包装完整页面,不包装直接输出;render_default:自动检测" validate:"required"`
	Title    string `json:"page_title" widget:"name:页面标题;type:input;placeholder:可选，仅包装模式有效"`
	Assets   string `json:"assets" widget:"name:页面资源;type:files;accept:image/*,.svg,.webp,.png,.jpg,.jpeg,.gif;max_size:20MB;max_count:30"`
}

type HTMLGenerateResp struct {
	OutputFile string `json:"output_file" widget:"name:HTML 文件;type:files"`
	Info       string `json:"info" widget:"name:生成信息;type:text_area"`
}

func HTMLGenerate(ctx *app.Context, resp response.Response) error {
	var req HTMLGenerateReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	baseName := sanitizeFileName(req.FileName)

	code := req.HTMLCode
	needsWrap := false

	switch req.WrapPage {
	case "强制包装完整页面":
		needsWrap = true
	case "不包装直接输出":
		needsWrap = false
	default:
		needsWrap = !looksLikeFullPage(code)
	}

	if needsWrap {
		title := req.Title
		if title == "" {
			title = baseName
		}
		code = wrapHTMLPage(title, code)
	}

	fs := ctx.GetFS()
	downloadedAssets, assetURIs, err := downloadAssetDataURIs(ctx, req.Assets)
	if err != nil {
		return err
	}
	defer fs.RemoveFiles(downloadedAssets)
	code, usedAssets := embedAssetPlaceholders(code, assetURIs)

	outputFiles, _, err := writeHTMLFile(ctx, baseName, code)
	if err != nil {
		return err
	}

	info := fmt.Sprintf("文件: %s.html\n大小: %d 字符", baseName, len(code))
	if needsWrap {
		info += "\n已自动包装为完整 HTML 页面"
	}
	info += assetUsageInfo(assetFileCount(req.Assets), usedAssets)

	return resp.Form(&HTMLGenerateResp{
		OutputFile: outputFiles,
		Info:       info,
	}).Build()
}

func looksLikeFullPage(code string) bool {
	lower := strings.ToLower(strings.TrimSpace(code))
	return strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html")
}

func wrapHTMLPage(title, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
* { box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Noto Sans SC", sans-serif; margin: 0; padding: 2rem; line-height: 1.6; color: #333; background: #fff; }
</style>
</head>
<body>
%s
</body>
</html>`, template.HTMLEscapeString(title), body)
}

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".html")
	name = strings.TrimSuffix(name, ".HTML")
	name = strings.TrimSuffix(name, ".htm")
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	name = replacer.Replace(name)
	if strings.TrimSpace(name) == "" {
		name = "page"
	}
	runes := []rune(name)
	if len(runes) > 200 {
		name = string(runes[:200])
	}
	return name
}

var HTMLGenerateTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "HTML 页面生成",
		Desc:     `将 HTML 代码生成为可直接访问的网页文件。输入完整 HTML 或片段代码，自动检测并包装为完整页面。支持上传图片资源，并在 HTML 中用 {{asset:文件名}} 占位符内嵌为 data URI。常用于快速生成活动页、展示页、工具页等场景。`,
		Tags:     []string{"HTML", "网页", "页面生成", "HTML生成", "前端", "网页制作", "资源内嵌"},
		Request:  &HTMLGenerateReq{},
		Response: &HTMLGenerateResp{},
	},
}

func init() {
	packageContext.POST("html_generate.form", HTMLGenerate, HTMLGenerateTemplate)
}
