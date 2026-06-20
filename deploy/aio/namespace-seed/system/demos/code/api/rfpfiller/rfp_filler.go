package rfpfiller

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

type RFPFillRun struct {
	ID               int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt        types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:生成时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	ProjectName      string         `json:"project_name" gorm:"column:project_name;comment:项目名;index" widget:"name:项目名;type:input" validate:"required"`
	QuestionCount    int            `json:"question_count" gorm:"column:question_count;comment:问题数" widget:"name:问题数;type:integer" hide:"create,update"`
	HighConfidence   int            `json:"high_confidence" gorm:"column:high_confidence;comment:高置信度答案数" widget:"name:高置信度;type:integer" hide:"create,update"`
	NeedsReview      int            `json:"needs_review" gorm:"column:needs_review;comment:需复核答案数" widget:"name:需复核;type:integer" hide:"create,update"`
	AverageScore     int            `json:"average_score" gorm:"column:average_score;comment:平均置信度" widget:"name:平均置信度;type:progress;min:0;max:100;unit:%" hide:"create,update"`
	ReportFile       string         `json:"report_file" gorm:"column:report_file;type:text;comment:报告文件" widget:"name:报告文件;type:files" hide:"create,update"`
	ExecutiveSummary string         `json:"executive_summary" gorm:"column:executive_summary;type:text;comment:摘要" widget:"name:摘要;type:text_area" hide:"create,update"`
}

func (RFPFillRun) TableName() string {
	return "rfp_fill_run"
}

type RFPFillReq struct {
	ProjectName        string `json:"project_name" widget:"name:项目名;type:input;placeholder:例如 Acme Security Questionnaire" validate:"required"`
	CompanyProfile     string `json:"company_profile" widget:"name:公司/产品资料;type:text_area;placeholder:粘贴公司介绍、产品说明、安全制度、隐私政策、历史答案等"`
	KnowledgeFiles     string `json:"knowledge_files" widget:"name:资料文件;type:files;accept:.txt,.md,.csv,.json,text/*,*/*;max_size:20MB;max_count:20"`
	QuestionnaireText  string `json:"questionnaire_text" widget:"name:客户问卷/RFP 问题;type:text_area;placeholder:每行一个问题，或粘贴整段问卷内容" validate:"required_without=QuestionnaireFiles"`
	QuestionnaireFiles string `json:"questionnaire_files" widget:"name:问卷文件;type:files;accept:.txt,.md,.csv,.json,text/*,*/*;max_size:20MB;max_count:10"`
	Tone               string `json:"tone" widget:"name:回答语气;type:select;options:简洁商务,详细说明,保守待确认;options_colors:409EFF,67C23A,E6A23C;render_default:简洁商务"`
}

type AnswerDraft struct {
	No         int    `json:"no" widget:"name:序号;type:integer"`
	Question   string `json:"question" widget:"name:问题;type:text_area"`
	Answer     string `json:"answer" widget:"name:回答草稿;type:text_area"`
	Confidence int    `json:"confidence" widget:"name:置信度;type:progress;min:0;max:100;unit:%"`
	Status     string `json:"status" widget:"name:状态;type:select;options:可直接使用,建议复核,缺少资料;options_colors:67C23A,E6A23C,F56C6C"`
	Source     string `json:"source" widget:"name:引用来源;type:text_area"`
	Category   string `json:"category" widget:"name:类别;type:input"`
}

type RFPFillResp struct {
	ExecutiveSummary string        `json:"executive_summary" widget:"name:执行摘要;type:text_area"`
	Answers          []AnswerDraft `json:"answers" widget:"name:答案草稿;type:table"`
	ReviewChecklist  string        `json:"review_checklist" widget:"name:复核清单;type:text_area"`
	ReportFile       string        `json:"report_file" widget:"name:Markdown 报告;type:files"`
}

func RFPFill(ctx *app.Context, resp response.Response) error {
	var req RFPFillReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, run, err := DoRFPFill(ctx, &req)
	if err != nil {
		return err
	}
	if db := ctx.GetGormDB(); db != nil {
		if err := db.Create(run).Error; err != nil {
			logger.Warnf(ctx, "[RFPFill] 保存运行历史失败: %v", err)
		}
	}
	return resp.Form(res).Build()
}

func DoRFPFill(ctx *app.Context, req *RFPFillReq) (*RFPFillResp, *RFPFillRun, error) {
	knowledgeText, downloadedKnowledge, err := loadTextInputs(ctx, req.CompanyProfile, req.KnowledgeFiles)
	if err != nil {
		return nil, nil, err
	}
	defer ctx.GetFS().RemoveFiles(downloadedKnowledge)

	questionnaireText, downloadedQuestionnaires, err := loadTextInputs(ctx, req.QuestionnaireText, req.QuestionnaireFiles)
	if err != nil {
		return nil, nil, err
	}
	defer ctx.GetFS().RemoveFiles(downloadedQuestionnaires)

	questions := extractQuestions(questionnaireText)
	if len(questions) == 0 {
		return nil, nil, fmt.Errorf("没有识别到问卷问题，请每行放一个问题或粘贴包含问号/编号的问卷文本")
	}
	chunks := splitKnowledge(knowledgeText)
	if len(chunks) == 0 {
		chunks = []knowledgeChunk{{Source: "空资料", Text: ""}}
	}

	answers := make([]AnswerDraft, 0, len(questions))
	totalConfidence := 0
	highConfidence := 0
	needsReview := 0
	for i, question := range questions {
		answer := draftAnswer(i+1, question, chunks, req.Tone)
		answers = append(answers, answer)
		totalConfidence += answer.Confidence
		if answer.Confidence >= 75 {
			highConfidence++
		}
		if answer.Status != "可直接使用" {
			needsReview++
		}
	}

	average := 0
	if len(answers) > 0 {
		average = totalConfidence / len(answers)
	}
	summary := buildExecutiveSummary(req.ProjectName, len(answers), highConfidence, needsReview, average)
	checklist := buildReviewChecklist(answers)
	reportPath, reportRef, err := writeRFPReport(ctx, req.ProjectName, summary, answers, checklist)
	if err != nil {
		return nil, nil, err
	}

	res := &RFPFillResp{
		ExecutiveSummary: summary,
		Answers:          answers,
		ReviewChecklist:  checklist,
		ReportFile:       reportRef,
	}
	run := &RFPFillRun{
		ProjectName:      strings.TrimSpace(req.ProjectName),
		QuestionCount:    len(answers),
		HighConfidence:   highConfidence,
		NeedsReview:      needsReview,
		AverageScore:     average,
		ReportFile:       reportPath,
		ExecutiveSummary: summary,
	}
	return res, run, nil
}

type RFPFillRunListReq struct {
	ProjectName       string `json:"project_name" form:"project_name" widget:"name:项目名;type:input"`
	query.PageSortReq `widget:"-"`
}

func RFPFillRunList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}
	var req RFPFillRunListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	queryDB := db.Model(&RFPFillRun{})
	if strings.TrimSpace(req.ProjectName) != "" {
		queryDB = queryDB.Where("project_name LIKE ?", "%"+strings.TrimSpace(req.ProjectName)+"%")
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("id DESC")
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	var items []RFPFillRun
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&items).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{Items: items, TotalCount: total, PageInfo: &req.PageSortReq}).Build()
}

func loadTextInputs(ctx *app.Context, inlineText, fileRefs string) (string, []string, error) {
	parts := make([]string, 0)
	if strings.TrimSpace(inlineText) != "" {
		parts = append(parts, strings.TrimSpace(inlineText))
	}
	var downloaded []string
	if strings.TrimSpace(fileRefs) != "" {
		fs := ctx.GetFS()
		downloaded = fs.DownloadFiles(fileRefs)
		for _, path := range downloaded {
			if path == "" {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return "", downloaded, fmt.Errorf("读取文件失败 %s: %w", filepath.Base(path), err)
			}
			parts = append(parts, fmt.Sprintf("[File: %s]\n%s", filepath.Base(path), printableText(data, 2*1024*1024)))
		}
	}
	return strings.Join(parts, "\n\n"), downloaded, nil
}

func printableText(data []byte, max int) string {
	if len(data) > max {
		data = data[:max]
	}
	var b strings.Builder
	for _, r := range string(data) {
		if r == '\n' || r == '\t' || r == '\r' || unicode.IsPrint(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

var questionLineRe = regexp.MustCompile(`^\s*(?:[-*]|\d+[\.)]|[A-Z][\.)])\s+`)

func extractQuestions(input string) []string {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	lines := strings.Split(input, "\n")
	var questions []string
	var current strings.Builder
	flush := func() {
		value := strings.TrimSpace(current.String())
		value = questionLineRe.ReplaceAllString(value, "")
		if len([]rune(value)) >= 8 {
			questions = append(questions, value)
		}
		current.Reset()
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			flush()
			continue
		}
		if questionLineRe.MatchString(line) && current.Len() > 0 {
			flush()
		}
		if current.Len() > 0 {
			current.WriteByte(' ')
		}
		current.WriteString(line)
		if strings.HasSuffix(line, "?") || strings.HasSuffix(line, "？") {
			flush()
		}
	}
	flush()
	return dedupeStrings(questions)
}

type knowledgeChunk struct {
	Source string
	Text   string
}

func splitKnowledge(input string) []knowledgeChunk {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	blocks := regexp.MustCompile(`\n{2,}`).Split(input, -1)
	chunks := make([]knowledgeChunk, 0, len(blocks))
	source := "资料"
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		if strings.HasPrefix(block, "[File: ") {
			if end := strings.Index(block, "]"); end > 0 {
				source = strings.TrimPrefix(block[:end], "[File: ")
				block = strings.TrimSpace(block[end+1:])
			}
		}
		for len([]rune(block)) > 1200 {
			part := string([]rune(block)[:1200])
			chunks = append(chunks, knowledgeChunk{Source: source, Text: part})
			block = string([]rune(block)[1200:])
		}
		if strings.TrimSpace(block) != "" {
			chunks = append(chunks, knowledgeChunk{Source: source, Text: block})
		}
	}
	return chunks
}

func draftAnswer(no int, question string, chunks []knowledgeChunk, tone string) AnswerDraft {
	bestChunk, score := bestKnowledgeMatch(question, chunks)
	confidence := scoreToConfidence(score)
	category := categorizeQuestion(question)
	status := "缺少资料"
	answer := "未在资料中找到足够依据。建议补充公司安全制度、产品文档、隐私政策或历史问卷答案后再填写。"
	source := "未匹配到明确来源"
	if confidence >= 70 {
		status = "可直接使用"
		answer = composeAnswer(question, bestChunk.Text, tone, false)
		source = bestChunk.Source + ": " + truncateRunes(bestChunk.Text, 220)
	} else if confidence >= 40 {
		status = "建议复核"
		answer = composeAnswer(question, bestChunk.Text, tone, true)
		source = bestChunk.Source + ": " + truncateRunes(bestChunk.Text, 220)
	}
	return AnswerDraft{No: no, Question: question, Answer: answer, Confidence: confidence, Status: status, Source: source, Category: category}
}

func bestKnowledgeMatch(question string, chunks []knowledgeChunk) (knowledgeChunk, int) {
	qTokens := keywordSet(question)
	best := knowledgeChunk{}
	bestScore := 0
	for _, chunk := range chunks {
		cTokens := keywordSet(chunk.Text)
		score := 0
		for token := range qTokens {
			if _, ok := cTokens[token]; ok {
				score += len([]rune(token))
			}
		}
		if score > bestScore {
			bestScore = score
			best = chunk
		}
	}
	return best, bestScore
}

func keywordSet(input string) map[string]struct{} {
	input = strings.ToLower(input)
	words := regexp.MustCompile(`[a-z0-9\u4e00-\u9fa5]{2,}`).FindAllString(input, -1)
	stop := map[string]struct{}{
		"the": {}, "and": {}, "for": {}, "with": {}, "your": {}, "you": {}, "are": {}, "does": {}, "have": {}, "that": {}, "this": {},
		"是否": {}, "什么": {}, "如何": {}, "进行": {}, "支持": {}, "提供": {}, "客户": {}, "系统": {},
	}
	out := map[string]struct{}{}
	for _, word := range words {
		if _, skip := stop[word]; skip {
			continue
		}
		out[word] = struct{}{}
	}
	return out
}

func scoreToConfidence(score int) int {
	switch {
	case score >= 28:
		return 88
	case score >= 18:
		return 76
	case score >= 10:
		return 58
	case score >= 5:
		return 42
	default:
		return 20
	}
}

func composeAnswer(question, source, tone string, cautious bool) string {
	source = strings.TrimSpace(source)
	source = truncateRunes(source, 420)
	prefix := "Based on the provided materials, "
	if strings.Contains(question, "是否") || strings.Contains(question, "吗") {
		prefix = "根据已提供资料，"
	}
	if cautious || tone == "保守待确认" {
		return prefix + "the preliminary answer is: " + source + "\n\nPlease review this answer against the latest internal policy before sending it to the customer."
	}
	if tone == "详细说明" {
		return prefix + source + "\n\nThis answer should be validated by the security or product owner before final submission."
	}
	return prefix + source
}

func categorizeQuestion(question string) string {
	q := strings.ToLower(question)
	cases := []struct {
		Name     string
		Keywords []string
	}{
		{"Data Security", []string{"security", "encryption", "加密", "安全", "访问控制", "access"}},
		{"Privacy", []string{"privacy", "gdpr", "个人信息", "隐私", "data protection"}},
		{"Compliance", []string{"soc", "iso", "compliance", "合规", "审计", "audit"}},
		{"Infrastructure", []string{"hosting", "cloud", "backup", "灾备", "备份", "infrastructure"}},
		{"Product", []string{"feature", "api", "integration", "产品", "功能", "接口"}},
		{"Commercial", []string{"pricing", "sla", "contract", "价格", "合同", "付款", "服务等级"}},
	}
	for _, item := range cases {
		for _, keyword := range item.Keywords {
			if strings.Contains(q, strings.ToLower(keyword)) {
				return item.Name
			}
		}
	}
	return "General"
}

func buildExecutiveSummary(project string, total, high, review, average int) string {
	return fmt.Sprintf("项目: %s\n问题数: %d\n可直接使用: %d\n需要复核/补充: %d\n平均置信度: %d%%\n\n建议先处理低置信度问题，再由安全、法务或产品负责人做最终确认。", strings.TrimSpace(project), total, high, review, average)
}

func buildReviewChecklist(answers []AnswerDraft) string {
	var lines []string
	for _, answer := range answers {
		if answer.Status != "可直接使用" {
			lines = append(lines, fmt.Sprintf("- #%d [%s] %s", answer.No, answer.Status, truncateRunes(answer.Question, 100)))
		}
	}
	if len(lines) == 0 {
		return "所有问题都有较高置信度答案。仍建议提交前做一次人工通读。"
	}
	return strings.Join(lines, "\n")
}

func writeRFPReport(ctx *app.Context, project, summary string, answers []AnswerDraft, checklist string) (string, string, error) {
	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()
	fileName := sanitizeFileName(project) + "-rfp-answer-draft.md"
	outputPath := filepath.Join(outputDir, fileName)
	var b strings.Builder
	b.WriteString("# Security Questionnaire / RFP Answer Draft\n\n")
	b.WriteString(summary)
	b.WriteString("\n\n## Answers\n\n")
	for _, answer := range answers {
		b.WriteString(fmt.Sprintf("### %d. %s\n\n", answer.No, answer.Question))
		b.WriteString(fmt.Sprintf("- Category: %s\n- Status: %s\n- Confidence: %d%%\n- Source: %s\n\n", answer.Category, answer.Status, answer.Confidence, answer.Source))
		b.WriteString(answer.Answer)
		b.WriteString("\n\n")
	}
	b.WriteString("## Review Checklist\n\n")
	b.WriteString(checklist)
	b.WriteString("\n\nGenerated at ")
	b.WriteString(time.Now().Format(time.RFC3339))
	b.WriteByte('\n')
	if err := os.WriteFile(outputPath, []byte(b.String()), 0644); err != nil {
		return "", "", fmt.Errorf("写入报告失败: %w", err)
	}
	return outputPath, fs.ResponseFiles([]string{outputPath}), nil
}

func sanitizeFileName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = regexp.MustCompile(`[^a-z0-9\u4e00-\u9fa5-]+`).ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if name == "" {
		name = "rfp"
	}
	return name
}

func truncateRunes(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "..."
}

func dedupeStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, strings.TrimSpace(item))
	}
	sort.Strings(out)
	return out
}

var RFPFillTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "安全问卷/RFP 自动填写",
		Desc:         "根据公司资料、产品文档和历史答案，为客户安全问卷或 RFP 生成回答草稿、引用来源、置信度和复核清单。适合 B2B SaaS、外包公司和售前团队。",
		Tags:         []string{"RFP", "安全问卷", "售前", "合规", "知识库", "商业模板"},
		Request:      &RFPFillReq{},
		Response:     &RFPFillResp{},
		CreateTables: []interface{}{&RFPFillRun{}},
	},
}

var RFPFillRunListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "RFP 填写历史",
		Desc:         "查看历史生成记录、问题数量、置信度和报告文件。",
		Tags:         []string{"RFP", "安全问卷", "历史记录"},
		Request:      &RFPFillRunListReq{},
		CreateTables: []interface{}{&RFPFillRun{}},
	},
	AutoCrudTable: &RFPFillRun{},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		ids := req.GetIds()
		if len(ids) == 0 {
			return nil, fmt.Errorf("请选择要删除的历史记录")
		}
		if err := ctx.GetGormDB().Delete(&RFPFillRun{}, ids).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func init() {
	packageContext.POST("rfp_fill.form", RFPFill, RFPFillTemplate)
	packageContext.GET("rfp_fill_history.table", RFPFillRunList, RFPFillRunListTemplate)
}
