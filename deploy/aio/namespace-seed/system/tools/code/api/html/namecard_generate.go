package html

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type NamecardGenerateReq struct {
	Name     string `json:"name" widget:"name:姓名;type:input;placeholder:例如：张三" validate:"required"`
	Title    string `json:"title" widget:"name:职位;type:input;placeholder:例如：高级前端工程师"`
	Company  string `json:"company" widget:"name:公司;type:input;placeholder:例如：XX科技有限公司"`
	Phone    string `json:"phone" widget:"name:手机;type:input;placeholder:例如：138 0000 0000"`
	Email    string `json:"email" widget:"name:邮箱;type:input;placeholder:例如：zhangsan@company.com"`
	Address  string `json:"address" widget:"name:地址;type:input;placeholder:例如：北京市朝阳区"`
	FileName string `json:"file_name" widget:"name:文件名;type:input;placeholder:输入文件名（无需后缀），例如 zhangsan-card" validate:"required"`
	Theme    string `json:"theme" widget:"name:风格;type:select;options:简洁白,蓝色商务,深色,绿色;render_default:简洁白" validate:"required"`
}

type NamecardGenerateResp struct {
	OutputFile string `json:"output_file" widget:"name:HTML 文件;type:files"`
	Info       string `json:"info" widget:"name:信息;type:text"`
}

func NamecardGenerate(ctx *app.Context, resp response.Response) error {
	var req NamecardGenerateReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	baseName := sanitizeFileName(req.FileName)
	htmlContent := buildNamecardHTML(req)

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

	return resp.Form(&NamecardGenerateResp{
		OutputFile: outputFiles,
		Info:       "名片已生成 | 标准尺寸 90×54mm，可打印",
	}).Build()
}

func buildNamecardHTML(req NamecardGenerateReq) string {
	bg, cardBg, accent, textColor, subColor := namecardTheme(req.Theme)

	var topLines, bottomLines string
	if req.Title != "" {
		topLines += fmt.Sprintf(`<div class="nc-line nc-title">%s</div>`, template.HTMLEscapeString(req.Title))
	}
	if req.Company != "" {
		topLines += fmt.Sprintf(`<div class="nc-line nc-company">%s</div>`, template.HTMLEscapeString(req.Company))
	}
	if req.Phone != "" {
		bottomLines += fmt.Sprintf(`<div class="nc-line">📱 %s</div>`, template.HTMLEscapeString(req.Phone))
	}
	if req.Email != "" {
		bottomLines += fmt.Sprintf(`<div class="nc-line">✉️ %s</div>`, template.HTMLEscapeString(req.Email))
	}
	if req.Address != "" {
		bottomLines += fmt.Sprintf(`<div class="nc-line">📍 %s</div>`, template.HTMLEscapeString(req.Address))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s - 名片</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Noto Sans SC",sans-serif;background:%s;min-height:100vh;display:flex;align-items:center;justify-content:center;padding:2rem}
.card{width:340px;height:212px;background:%s;border-radius:8px;box-shadow:0 8px 32px rgba(0,0,0,.12);padding:24px 28px;display:flex;flex-direction:column;justify-content:space-between;border-left:4px solid %s}
.nc-name{font-size:1.4rem;font-weight:700;color:%s;margin-bottom:4px}
.nc-title{font-size:.9rem;color:%s;margin-bottom:2px}
.nc-company{font-size:.85rem;color:%s;font-weight:500;margin-bottom:8px}
.nc-line{font-size:.8rem;color:%s;line-height:1.6}
@media print{body{padding:0;background:#fff}.card{box-shadow:none}}
</style>
</head>
<body>
<div class="card">
<div>
<div class="nc-name">%s</div>
%s
</div>
<div class="nc-lines">%s</div>
</div>
</body>
</html>`,
		template.HTMLEscapeString(req.Name),
		bg, cardBg, accent, textColor, subColor, subColor, subColor,
		template.HTMLEscapeString(req.Name),
		topLines,
		bottomLines,
	)
}

func namecardTheme(theme string) (bg, cardBg, accent, textColor, subColor string) {
	switch theme {
	case "蓝色商务":
		return "#eff6ff", "#fff", "#2563eb", "#1e293b", "#64748b"
	case "深色":
		return "#0f172a", "#1e293b", "#60a5fa", "#f1f5f9", "#94a3b8"
	case "绿色":
		return "#f0fdf4", "#fff", "#059669", "#1a3a2a", "#6b7280"
	default:
		return "#f8fafc", "#fff", "#475569", "#1e293b", "#64748b"
	}
}

var NamecardGenerateTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "名片生成",
		Desc:     `根据姓名、职位、公司、联系方式等信息生成可直接访问、可打印的名片 HTML 页面。标准名片比例，多种风格可选。常用于会议、展会、社交等场景。`,
		Tags:     []string{"名片", "Namecard", "商务", "打印", "HTML"},
		Request:  &NamecardGenerateReq{},
		Response: &NamecardGenerateResp{},
	},
}

func init() {
	packageContext.POST("namecard_generate.form", NamecardGenerate, NamecardGenerateTemplate)
}
