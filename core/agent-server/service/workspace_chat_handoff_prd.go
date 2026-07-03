package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

func renderWorkspacePRDExecutionMarkdown(artifact map[string]interface{}, executeDirectory, targetAppDirectory string) string {
	if len(artifact) == 0 {
		return ""
	}
	project := workspaceMapField(artifact, "project")
	projectName := firstNonEmptyString(workspaceStringField(project, "name"), workspaceStringField(artifact, "project_name"))
	projectCode := firstNonEmptyString(workspaceStringField(project, "code"), workspaceStringField(artifact, "project_code"))
	projectSummary := firstNonEmptyString(workspaceStringField(project, "summary"), workspaceStringField(artifact, "summary"))

	var b strings.Builder
	title := firstNonEmptyString(projectName, projectCode, "未命名应用")
	b.WriteString("# 已确认 PRD：")
	b.WriteString(workspaceMarkdownHeading(title))
	b.WriteString("\n\n")
	workspaceWriteMarkdownTable(&b, []string{"项", "内容"}, [][]string{
		{"项目名称", projectName},
		{"项目 code", projectCode},
		{"业务目标", projectSummary},
		{"执行目录", executeDirectory},
		{"目标应用目录", targetAppDirectory},
	})

	tables := workspaceMapsFromSlice(workspaceSliceField(artifact, "tables"))
	forms := workspaceMapsFromSlice(workspaceSliceField(artifact, "forms"))
	charts := workspaceMapsFromSlice(workspaceSliceField(artifact, "charts"))
	maintainTables, readonlyTables := workspaceSplitPRDTables(tables)

	resourceRows := [][]string{}
	for _, table := range maintainTables {
		resourceRows = append(resourceRows, []string{"可维护 Table", workspacePRDResourceName(table), workspaceStringField(table, "code"), workspaceJoinOrDash(workspacePRDFieldNames(table, "fields")), workspaceJoinOrDash(workspaceStringItems(table, "handlers"))})
	}
	for _, form := range forms {
		resourceRows = append(resourceRows, []string{"Form", workspacePRDResourceName(form), workspaceStringField(form, "code"), workspaceJoinOrDash(workspacePRDFieldNames(form, "request_fields")), firstNonEmptyString(workspaceStringField(form, "target_table"), "-")})
	}
	for _, table := range readonlyTables {
		resourceRows = append(resourceRows, []string{"只读 Table", workspacePRDResourceName(table), workspaceStringField(table, "code"), workspaceJoinOrDash(workspacePRDFieldNames(table, "fields")), "只读查询"})
	}
	for _, chart := range charts {
		resourceRows = append(resourceRows, []string{"Chart", workspacePRDResourceName(chart), workspaceStringField(chart, "code"), workspaceJoinOrDash(workspaceNamedItems(chart, "metrics")), firstNonEmptyString(workspaceStringField(chart, "source_table"), "-")})
	}
	if len(resourceRows) > 0 {
		b.WriteString("\n\n## 资源总览\n\n")
		b.WriteString("生成顺序：可维护 Table -> Form -> 只读 Table -> Chart。\n\n")
		workspaceWriteMarkdownTable(&b, []string{"类型", "名称", "code", "核心字段/指标", "操作/目标"}, resourceRows)
	}

	for _, table := range maintainTables {
		workspaceWritePRDTableSection(&b, table)
	}
	for _, form := range forms {
		workspaceWritePRDFormSection(&b, form)
	}
	for _, table := range readonlyTables {
		workspaceWritePRDTableSection(&b, table)
	}
	for _, chart := range charts {
		workspaceWritePRDChartSection(&b, chart)
	}
	workspaceWritePRDRulesSection(&b, workspaceRules(artifact))
	return strings.TrimSpace(b.String())
}

func workspaceMapsFromSlice(items []interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		m := workspaceAsMap(item)
		if len(m) > 0 {
			out = append(out, m)
		}
	}
	return out
}

func workspaceSplitPRDTables(tables []map[string]interface{}) ([]map[string]interface{}, []map[string]interface{}) {
	maintainTables := []map[string]interface{}{}
	readonlyTables := []map[string]interface{}{}
	for _, table := range tables {
		if len(workspaceStringItems(table, "handlers")) > 0 {
			maintainTables = append(maintainTables, table)
		} else {
			readonlyTables = append(readonlyTables, table)
		}
	}
	return maintainTables, readonlyTables
}

func workspaceWritePRDTableSection(b *strings.Builder, table map[string]interface{}) {
	name := workspacePRDResourceName(table)
	b.WriteString("\n\n## Table：")
	b.WriteString(workspaceMarkdownHeading(name))
	b.WriteString("\n\n")
	handlers := workspaceStringItems(table, "handlers")
	operation := "只读查询"
	if len(handlers) > 0 {
		operation = strings.Join(handlers, "、")
	}
	workspaceWriteMarkdownTable(b, []string{"项", "内容"}, [][]string{
		{"标题", firstNonEmptyString(workspaceStringField(table, "title"), name)},
		{"说明", workspaceStringField(table, "desc")},
		{"操作能力", operation},
		{"搜索字段", workspaceJoinOrDash(workspacePRDFieldNames(table, "search_fields"))},
	})
	workspaceWritePRDFieldTable(b, "字段", workspaceSliceField(table, "fields"))
	workspaceWritePRDFieldTable(b, "搜索字段", workspaceSliceField(table, "search_fields"))
	workspaceWritePRDExamplesTable(b, "示例数据", workspaceSliceField(table, "examples"))
}

func workspaceWritePRDFormSection(b *strings.Builder, form map[string]interface{}) {
	name := workspacePRDResourceName(form)
	b.WriteString("\n\n## Form：")
	b.WriteString(workspaceMarkdownHeading(name))
	b.WriteString("\n\n")
	workspaceWriteMarkdownTable(b, []string{"项", "内容"}, [][]string{
		{"说明", workspaceStringField(form, "desc")},
		{"目标表", firstNonEmptyString(workspaceStringField(form, "target_table"), "纯处理型 Form")},
		{"请求字段", workspaceJoinOrDash(workspacePRDFieldNames(form, "request_fields"))},
		{"响应字段", workspaceJoinOrDash(workspacePRDFieldNames(form, "response_fields"))},
	})
	workspaceWritePRDFieldTable(b, "请求字段", workspaceSliceField(form, "request_fields"))
	workspaceWritePRDFieldTable(b, "响应字段", workspaceSliceField(form, "response_fields"))
	workspaceWritePRDExamplesTable(b, "提交示例", []interface{}{form["example"]})
}

func workspaceWritePRDChartSection(b *strings.Builder, chart map[string]interface{}) {
	name := workspacePRDResourceName(chart)
	b.WriteString("\n\n## Chart：")
	b.WriteString(workspaceMarkdownHeading(name))
	b.WriteString("\n\n")
	workspaceWriteMarkdownTable(b, []string{"项", "内容"}, [][]string{
		{"说明", workspaceStringField(chart, "desc")},
		{"来源表", workspaceStringField(chart, "source_table")},
		{"图表类型", workspaceStringField(chart, "chart_type")},
		{"维度", workspaceStringField(chart, "dimension")},
		{"指标", workspaceJoinOrDash(workspaceNamedItems(chart, "metrics"))},
	})
	workspaceWritePRDFieldTable(b, "筛选字段", workspaceSliceField(chart, "filters"))
	workspaceWritePRDExamplesTable(b, "图表示例", workspaceSliceField(chart, "examples"))
}

func workspaceWritePRDRulesSection(b *strings.Builder, rules []string) {
	b.WriteString("\n\n## 业务规则与复杂逻辑\n\n")
	rows := [][]string{}
	for i, rule := range rules {
		rows = append(rows, []string{fmt.Sprintf("R%d", i+1), rule, "必须实现并测试"})
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"R0", "PRD 未写明复杂逻辑", "开发前若发现状态计算、重复提交、权限、只读、跨表写入、统计口径或异常边界不明确，先补充确认。"})
	}
	workspaceWriteMarkdownTable(b, []string{"编号", "规则", "开发要求"}, rows)
}

func workspaceWritePRDFieldTable(b *strings.Builder, title string, items []interface{}) {
	if len(items) == 0 {
		return
	}
	rows := [][]string{}
	for _, item := range items {
		field := workspaceAsMap(item)
		if len(field) == 0 {
			if name := workspaceStringValue(item); name != "" {
				rows = append(rows, []string{name, "-", "-", "-", "-"})
			}
			continue
		}
		rows = append(rows, []string{
			workspaceStringField(field, "name"),
			workspaceStringField(field, "widget"),
			workspaceRequiredText(field["required"]),
			workspaceStringField(field, "desc"),
			workspaceStringField(field, "hide"),
		})
	}
	if len(rows) == 0 {
		return
	}
	b.WriteString("\n\n### ")
	b.WriteString(workspaceMarkdownHeading(title))
	b.WriteString("\n\n")
	workspaceWriteMarkdownTable(b, []string{"字段", "组件", "必填", "说明", "展示限制"}, rows)
}

func workspaceWritePRDExamplesTable(b *strings.Builder, title string, items []interface{}) {
	rows := [][]string{}
	for _, item := range items {
		if item == nil {
			continue
		}
		if text := workspaceCompactJSON(item, 500); text != "" && text != "null" && text != "{}" {
			rows = append(rows, []string{fmt.Sprintf("E%d", len(rows)+1), text})
		}
	}
	if len(rows) == 0 {
		return
	}
	b.WriteString("\n\n### ")
	b.WriteString(workspaceMarkdownHeading(title))
	b.WriteString("\n\n")
	workspaceWriteMarkdownTable(b, []string{"序号", "内容"}, rows)
}

func workspacePRDResourceName(m map[string]interface{}) string {
	return firstNonEmptyString(workspaceStringField(m, "name"), workspaceStringField(m, "title"), workspaceStringField(m, "code"), "未命名资源")
}

func workspacePRDFieldNames(m map[string]interface{}, key string) []string {
	items := workspaceSliceField(m, key)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s := workspaceStringValue(item); s != "" {
			out = appendUniqueWorkspaceString(out, s, 18)
		}
	}
	return out
}

func workspaceRequiredText(value interface{}) string {
	switch v := value.(type) {
	case bool:
		if v {
			return "是"
		}
		return "否"
	case string:
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return "否"
}

func workspaceJoinOrDash(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	return strings.Join(items, "、")
}

func workspaceCompactJSON(value interface{}, maxRunes int) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return compactText(workspaceStringValue(value), maxRunes)
	}
	return compactText(string(raw), maxRunes)
}

func workspaceWriteMarkdownTable(b *strings.Builder, headers []string, rows [][]string) {
	if len(headers) == 0 || len(rows) == 0 {
		return
	}
	b.WriteString("| ")
	for i, header := range headers {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(workspaceMarkdownCell(header))
	}
	b.WriteString(" |\n| ")
	for i := range headers {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString("---")
	}
	b.WriteString(" |\n")
	for _, row := range rows {
		b.WriteString("| ")
		for i := range headers {
			if i > 0 {
				b.WriteString(" | ")
			}
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			b.WriteString(workspaceMarkdownCell(cell))
		}
		b.WriteString(" |\n")
	}
}

func workspaceMarkdownHeading(s string) string {
	s = compactText(s, 120)
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func workspaceMarkdownCell(s string) string {
	s = compactText(s, 360)
	if s == "" {
		return "-"
	}
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", "<br>")
	return s
}
