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

type ResumeGenerateReq struct {
	Name       string `json:"name" widget:"name:姓名;type:input;placeholder:例如：张三" validate:"required"`
	Phone      string `json:"phone" widget:"name:手机;type:input;placeholder:例如：138****1234"`
	Email      string `json:"email" widget:"name:邮箱;type:input;placeholder:例如：zhangsan@example.com"`
	Objective  string `json:"objective" widget:"name:求职意向;type:text_area;placeholder:例如：寻求前端开发岗位，3年 React 经验"`
	Education  string `json:"education" widget:"name:教育经历;type:text_area;placeholder:每行一条，格式：学校|专业|学历|时间\n例如：\nXX大学|计算机科学|本科|2018-2022" validate:"required"`
	Experience string `json:"experience" widget:"name:工作经历;type:text_area;placeholder:每行一条，格式：公司|职位|时间|描述\n例如：\nXX科技|前端开发|2022-至今|负责核心业务开发"`
	Projects   string `json:"projects" widget:"name:项目经历;type:text_area;placeholder:每行一条，格式：项目名|描述|技术栈\n例如：\n电商平台|负责商品模块|Vue3,TypeScript"`
	Skills     string `json:"skills" widget:"name:技能;type:text_area;placeholder:每行一个或逗号分隔，例如：\nJavaScript\nVue / React\nTypeScript"`
	FileName   string `json:"file_name" widget:"name:文件名;type:input;placeholder:输入文件名（无需后缀），例如 zhangsan-resume" validate:"required"`
	Theme      string `json:"theme" widget:"name:主题;type:select;options:简洁白,蓝色商务,深色;render_default:简洁白" validate:"required"`
}

type ResumeGenerateResp struct {
	OutputFile string `json:"output_file" widget:"name:HTML 文件;type:files"`
	Info       string `json:"info" widget:"name:信息;type:text"`
}

type resumeEdu struct {
	School   string
	Major    string
	Degree   string
	Duration string
}

type resumeExp struct {
	Company  string
	Position string
	Duration string
	Desc     string
}

type resumeProj struct {
	Name string
	Desc string
	Tech string
}

func ResumeGenerate(ctx *app.Context, resp response.Response) error {
	var req ResumeGenerateReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	baseName := sanitizeFileName(req.FileName)
	edu := parseResumeEducation(req.Education)
	exp := parseResumeExperience(req.Experience)
	proj := parseResumeProjects(req.Projects)
	skills := parseResumeSkills(req.Skills)

	htmlContent := buildResumeHTML(req, edu, exp, proj, skills)

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

	return resp.Form(&ResumeGenerateResp{
		OutputFile: outputFiles,
		Info:       fmt.Sprintf("简历已生成 | %d 条教育 + %d 条工作 + %d 条项目 + %d 项技能", len(edu), len(exp), len(proj), len(skills)),
	}).Build()
}

func parseResumeEducation(s string) []resumeEdu {
	var list []resumeEdu
	for _, line := range parseLines(s) {
		parts := strings.SplitN(line, "|", 4)
		e := resumeEdu{}
		if len(parts) >= 1 {
			e.School = strings.TrimSpace(parts[0])
		}
		if len(parts) >= 2 {
			e.Major = strings.TrimSpace(parts[1])
		}
		if len(parts) >= 3 {
			e.Degree = strings.TrimSpace(parts[2])
		}
		if len(parts) >= 4 {
			e.Duration = strings.TrimSpace(parts[3])
		}
		if e.School != "" {
			list = append(list, e)
		}
	}
	return list
}

func parseResumeExperience(s string) []resumeExp {
	var list []resumeExp
	for _, line := range parseLines(s) {
		parts := strings.SplitN(line, "|", 4)
		e := resumeExp{}
		if len(parts) >= 1 {
			e.Company = strings.TrimSpace(parts[0])
		}
		if len(parts) >= 2 {
			e.Position = strings.TrimSpace(parts[1])
		}
		if len(parts) >= 3 {
			e.Duration = strings.TrimSpace(parts[2])
		}
		if len(parts) >= 4 {
			e.Desc = strings.TrimSpace(parts[3])
		}
		if e.Company != "" {
			list = append(list, e)
		}
	}
	return list
}

func parseResumeProjects(s string) []resumeProj {
	var list []resumeProj
	for _, line := range parseLines(s) {
		parts := strings.SplitN(line, "|", 3)
		p := resumeProj{}
		if len(parts) >= 1 {
			p.Name = strings.TrimSpace(parts[0])
		}
		if len(parts) >= 2 {
			p.Desc = strings.TrimSpace(parts[1])
		}
		if len(parts) >= 3 {
			p.Tech = strings.TrimSpace(parts[2])
		}
		if p.Name != "" {
			list = append(list, p)
		}
	}
	return list
}

func parseResumeSkills(s string) []string {
	var list []string
	for _, line := range parseLines(s) {
		for _, item := range strings.Split(line, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				list = append(list, item)
			}
		}
	}
	return list
}

func buildResumeHTML(req ResumeGenerateReq, edu []resumeEdu, exp []resumeExp, proj []resumeProj, skills []string) string {
	bg, cardBg, accent, textColor, subColor, borderColor := resumeThemeColors(req.Theme)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s - 简历</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","Noto Sans SC",sans-serif;background:%s;color:%s;padding:2rem;line-height:1.6}
.resume{max-width:800px;margin:0 auto;background:%s;border-radius:12px;box-shadow:0 4px 24px rgba(0,0,0,.08);overflow:hidden}
.header{background:%s;color:#fff;padding:2rem 2.5rem}
.header h1{font-size:1.8rem;font-weight:700;margin-bottom:.5rem}
.header .contact{font-size:.9rem;opacity:.95}
.header .contact span{margin-right:1.5rem}
.section{padding:1.5rem 2.5rem;border-bottom:1px solid %s}
.section:last-child{border-bottom:none}
.section-title{font-size:1rem;font-weight:600;color:%s;margin-bottom:1rem;padding-bottom:.4rem;border-bottom:2px solid %s;display:inline-block}
.item{margin-bottom:1.2rem}
.item:last-child{margin-bottom:0}
.item-title{font-weight:600;color:%s;font-size:1rem}
.item-meta{font-size:.85rem;color:%s;margin:.3rem 0}
.item-desc{font-size:.9rem;color:%s;line-height:1.55}
.skills-wrap{display:flex;flex-wrap:wrap;gap:.5rem}
.skill-tag{background:%s;color:%s;padding:4px 12px;border-radius:6px;font-size:.85rem}
@media print{.resume{box-shadow:none}body{padding:0}}
</style>
</head>
<body>
<div class="resume">
<div class="header">
<h1>%s</h1>
<div class="contact">`,
		template.HTMLEscapeString(req.Name),
		bg, textColor, cardBg, accent, borderColor, accent, accent, textColor, subColor, subColor,
		accent, "#fff",
		template.HTMLEscapeString(req.Name),
	))

	if req.Phone != "" {
		sb.WriteString(fmt.Sprintf(`<span>📱 %s</span>`, template.HTMLEscapeString(req.Phone)))
	}
	if req.Email != "" {
		sb.WriteString(fmt.Sprintf(`<span>✉️ %s</span>`, template.HTMLEscapeString(req.Email)))
	}
	sb.WriteString(`</div>`)
	if req.Objective != "" {
		sb.WriteString(fmt.Sprintf(`<p style="margin-top:1rem;font-size:.95rem;opacity:.95">%s</p>`, template.HTMLEscapeString(req.Objective)))
	}
	sb.WriteString(`</div>`)

	if len(edu) > 0 {
		sb.WriteString(`<div class="section"><div class="section-title">教育经历</div>`)
		for _, e := range edu {
			sb.WriteString(fmt.Sprintf(`<div class="item"><div class="item-title">%s · %s</div><div class="item-meta">%s | %s</div></div>`,
				template.HTMLEscapeString(e.School), template.HTMLEscapeString(e.Major),
				template.HTMLEscapeString(e.Degree), template.HTMLEscapeString(e.Duration)))
		}
		sb.WriteString(`</div>`)
	}

	if len(exp) > 0 {
		sb.WriteString(`<div class="section"><div class="section-title">工作经历</div>`)
		for _, e := range exp {
			sb.WriteString(fmt.Sprintf(`<div class="item"><div class="item-title">%s · %s</div><div class="item-meta">%s</div>`,
				template.HTMLEscapeString(e.Company), template.HTMLEscapeString(e.Position), template.HTMLEscapeString(e.Duration)))
			if e.Desc != "" {
				sb.WriteString(fmt.Sprintf(`<div class="item-desc">%s</div>`, template.HTMLEscapeString(e.Desc)))
			}
			sb.WriteString(`</div>`)
		}
		sb.WriteString(`</div>`)
	}

	if len(proj) > 0 {
		sb.WriteString(`<div class="section"><div class="section-title">项目经历</div>`)
		for _, p := range proj {
			sb.WriteString(fmt.Sprintf(`<div class="item"><div class="item-title">%s</div>`, template.HTMLEscapeString(p.Name)))
			if p.Desc != "" {
				sb.WriteString(fmt.Sprintf(`<div class="item-desc">%s</div>`, template.HTMLEscapeString(p.Desc)))
			}
			if p.Tech != "" {
				sb.WriteString(fmt.Sprintf(`<div class="item-meta">%s</div>`, template.HTMLEscapeString(p.Tech)))
			}
			sb.WriteString(`</div>`)
		}
		sb.WriteString(`</div>`)
	}

	if len(skills) > 0 {
		sb.WriteString(`<div class="section"><div class="section-title">技能</div><div class="skills-wrap">`)
		for _, s := range skills {
			sb.WriteString(fmt.Sprintf(`<span class="skill-tag">%s</span>`, template.HTMLEscapeString(s)))
		}
		sb.WriteString(`</div></div>`)
	}

	sb.WriteString(`</div></body></html>`)
	return sb.String()
}

func resumeThemeColors(theme string) (bg, cardBg, accent, textColor, subColor, borderColor string) {
	switch theme {
	case "蓝色商务":
		return "#f0f9ff", "#fff", "#0369a1", "#0f172a", "#64748b", "#e2e8f0"
	case "深色":
		return "#0f172a", "#1e293b", "#334155", "#e2e8f0", "#94a3b8", "#334155"
	default:
		return "#f8fafc", "#fff", "#475569", "#1e293b", "#64748b", "#e2e8f0"
	}
}

var ResumeGenerateTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "简历生成",
		Desc:     `根据结构化信息生成可直接访问、可打印的简历 HTML 页面。支持教育经历、工作经历、项目经历、技能等模块，多种主题风格。常用于求职、个人介绍、作品集等场景。`,
		Tags:     []string{"简历", "Resume", "求职", "个人介绍", "HTML", "打印"},
		Request:  &ResumeGenerateReq{},
		Response: &ResumeGenerateResp{},
	},
}

func init() {
	packageContext.POST("resume_generate.form", ResumeGenerate, ResumeGenerateTemplate)
}
