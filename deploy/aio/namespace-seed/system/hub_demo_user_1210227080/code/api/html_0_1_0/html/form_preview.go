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

type FormPreviewReq struct {
	Title     string `json:"title" widget:"name:表单标题;type:input;placeholder:例如：用户调研问卷" validate:"required"`
	Questions string `json:"questions" widget:"name:题目列表;type:text_area;placeholder:每行一题，格式：类型|题目|选项（逗号分隔）\n\n类型：单选、多选、填空、多行填空\n\n例如：\n单选|你的性别？|男,女,其他\n多选|你的技能？|JavaScript,Vue,React,Python\n填空|你的姓名|\n多行填空|请描述你的项目经验|" validate:"required"`
	FileName  string `json:"file_name" widget:"name:文件名;type:input;placeholder:输入文件名（无需后缀），例如 survey-form" validate:"required"`
	Theme     string `json:"theme" widget:"name:主题;type:select;options:浅色,深色;render_default:浅色" validate:"required"`
}

type FormPreviewResp struct {
	OutputFile string `json:"output_file" widget:"name:HTML 文件;type:files"`
	Info       string `json:"info" widget:"name:信息;type:text"`
}

type formQuestion struct {
	Type    string
	Title   string
	Options []string
}

func FormPreview(ctx *app.Context, resp response.Response) error {
	var req FormPreviewReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	baseName := sanitizeFileName(req.FileName)
	questions := parseFormQuestions(req.Questions)
	if len(questions) == 0 {
		return fmt.Errorf("至少需要一个题目")
	}

	isDark := req.Theme == "深色"
	htmlContent := buildFormPreviewHTML(req.Title, questions, isDark)

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

	return resp.Form(&FormPreviewResp{
		OutputFile: outputFiles,
		Info:       fmt.Sprintf("%d 道题目", len(questions)),
	}).Build()
}

func parseFormQuestions(s string) []formQuestion {
	var list []formQuestion
	for _, line := range parseLines(s) {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 2 {
			continue
		}
		q := formQuestion{
			Type:  strings.TrimSpace(parts[0]),
			Title: strings.TrimSpace(parts[1]),
		}
		if len(parts) >= 3 && strings.TrimSpace(parts[2]) != "" {
			for _, o := range strings.Split(parts[2], ",") {
				o = strings.TrimSpace(o)
				if o != "" {
					q.Options = append(q.Options, o)
				}
			}
		}
		if q.Title != "" {
			list = append(list, q)
		}
	}
	return list
}

func buildFormPreviewHTML(title string, questions []formQuestion, isDark bool) string {
	bg, cardBg, accent, textColor, subColor, borderColor := formPreviewTheme(isDark)

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
.form-wrap{max-width:600px;margin:0 auto;background:%s;border-radius:12px;box-shadow:0 4px 24px rgba(0,0,0,.08);padding:2rem;border:1px solid %s}
.form-title{font-size:1.4rem;font-weight:600;margin-bottom:1.5rem;color:%s}
.q-item{margin-bottom:1.5rem}
.q-title{font-size:1rem;font-weight:500;margin-bottom:.5rem;color:%s}
.q-options{display:flex;flex-direction:column;gap:6px}
.q-option{display:flex;align-items:center;gap:8px;cursor:default;padding:8px 12px;border-radius:8px;border:1px solid %s;background:%s}
.q-option input{accent-color:%s}
.q-input,.q-textarea{width:100%%;padding:10px 14px;border:1px solid %s;border-radius:8px;font-size:1rem;background:%s;color:%s}
.q-textarea{min-height:100px;resize:vertical}
.q-input:focus,.q-textarea:focus{outline:none;border-color:%s}
.form-note{font-size:.85rem;color:%s;margin-top:1.5rem;text-align:center}
</style>
</head>
<body>
<div class="form-wrap">
<div class="form-title">%s</div>
<form>
`,
		template.HTMLEscapeString(title),
		bg, textColor, cardBg, borderColor, textColor, textColor, borderColor, bg, accent, borderColor, cardBg, textColor, accent, subColor,
		template.HTMLEscapeString(title),
	))

	for i, q := range questions {
		sb.WriteString(fmt.Sprintf(`<div class="q-item"><div class="q-title">%d. %s</div>`, i+1, template.HTMLEscapeString(q.Title)))
		switch q.Type {
		case "单选":
			sb.WriteString(`<div class="q-options">`)
			for _, opt := range q.Options {
				sb.WriteString(fmt.Sprintf(`<label class="q-option"><input type="radio" name="q%d" value="%s">%s</label>`,
					i, template.HTMLEscapeString(opt), template.HTMLEscapeString(opt)))
			}
			sb.WriteString(`</div>`)
		case "多选":
			sb.WriteString(`<div class="q-options">`)
			for _, opt := range q.Options {
				sb.WriteString(fmt.Sprintf(`<label class="q-option"><input type="checkbox" name="q%d" value="%s">%s</label>`,
					i, template.HTMLEscapeString(opt), template.HTMLEscapeString(opt)))
			}
			sb.WriteString(`</div>`)
		case "多行填空":
			sb.WriteString(fmt.Sprintf(`<textarea class="q-textarea" name="q%d" placeholder="%s"></textarea>`, i, template.HTMLEscapeString(strings.Join(q.Options, " "))))
		default:
			sb.WriteString(fmt.Sprintf(`<input type="text" class="q-input" name="q%d" placeholder="%s">`, i, template.HTMLEscapeString(strings.Join(q.Options, " "))))
		}
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`<div class="form-note">（此为展示预览，不提交数据）</div></form></div></body></html>`)
	return sb.String()
}

func formPreviewTheme(isDark bool) (bg, cardBg, accent, textColor, subColor, borderColor string) {
	if isDark {
		return "#0f172a", "#1e293b", "#60a5fa", "#e2e8f0", "#94a3b8", "#334155"
	}
	return "#f8fafc", "#fff", "#2563eb", "#1e293b", "#64748b", "#e2e8f0"
}

var FormPreviewTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "表单预览生成",
		Desc:     `根据题目列表生成可直接访问的表单展示 HTML 页面。支持单选、多选、填空、多行填空。纯展示用途，不提交数据。常用于问卷预览、需求收集展示、调研表单设计等场景。`,
		Tags:     []string{"表单", "问卷", "调查", "预览", "HTML"},
		Request:  &FormPreviewReq{},
		Response: &FormPreviewResp{},
	},
}

func init() {
	packageContext.POST("form_preview.form", FormPreview, FormPreviewTemplate)
}
