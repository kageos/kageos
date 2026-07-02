package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/widget"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/functionschema"
)

func formatSearchMeta(data searchResultData) string {
	parts := []string{
		fmt.Sprintf("page=%d", data.Page),
		fmt.Sprintf("page_size=%d", data.PageSize),
	}
	if data.Keyword != "" {
		parts = append(parts, "keyword="+data.Keyword)
	}
	if data.FullCodePath != "" {
		parts = append(parts, "full_code_path="+data.FullCodePath)
	}
	if data.ResourceType != "" && data.ResourceType != "all" {
		parts = append(parts, "resource_type="+data.ResourceType)
	}
	if data.TemplateType != "" {
		parts = append(parts, "template_type="+data.TemplateType)
	}
	if data.Capability != "" {
		parts = append(parts, "capability="+data.Capability)
	}
	parts = append(parts, fmt.Sprintf("total=%d", data.Total))
	return "搜索参数：" + strings.Join(parts, " | ")
}

func formatSearchOutput(data searchResultData, requestOutput searchRequestOutput) string {
	var buf strings.Builder
	if data.Keyword == "" {
		buf.WriteString("搜索结果：未传 keyword，按完整路径或默认排序返回当前账号可见资源。\n")
	} else {
		buf.WriteString(fmt.Sprintf("搜索结果：%s\n", data.Keyword))
	}
	buf.WriteString(formatSearchMeta(data))
	buf.WriteString("\n")
	buf.WriteString(fmt.Sprintf("- 匹配到 %d 个内置工具\n", len(data.MatchedTools)))
	buf.WriteString(fmt.Sprintf("- 匹配到 %d 个可执行函数\n", len(data.Functions)))
	buf.WriteString(fmt.Sprintf("- 匹配到 %d 个目录/文档资源\n", len(data.Items)))
	buf.WriteString(fmt.Sprintf("- 匹配到 %d 条内置文档内容\n\n", len(data.DocMatches)))

	if len(data.MatchedTools) > 0 {
		buf.WriteString("【内置工具】\n")
		for _, tool := range data.MatchedTools {
			buf.WriteString("- ")
			buf.WriteString(tool.Name)
			buf.WriteString(": ")
			buf.WriteString(searchToolShortDescription(tool.Description))
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}

	if len(data.Functions) > 0 {
		buf.WriteString("【可执行函数】\n")
		buf.WriteString("调用方式：form → run_form_submit，table → 默认先 run_table_search，仅在能力摘要明确支持写入时再用 run_table_create/run_table_update/run_table_delete，chart → run_chart_query。\n")
		for i, fn := range data.Functions {
			buf.WriteString(formatSearchFunctionSummary(i, fn))
		}
		buf.WriteString("\n")
	}

	if len(data.Items) > 0 {
		buf.WriteString("【目录/文档资源】\n")
		for i, item := range data.Items {
			buf.WriteString(formatSearchResourceSummary(i, item))
		}
		buf.WriteString("\n")
	}

	if len(data.DocMatches) > 0 {
		buf.WriteString("【内置文档内容命中】\n")
		buf.WriteString("完整查看：read_doc(directory=<full_code_path>)。/system/prompt 案例是文档案例，不要用 read_go_file/read_go_file_lines 读取。\n")
		for i, item := range data.DocMatches {
			buf.WriteString(formatSearchDocMatchSummary(i, item))
		}
	}

	if requestOutput == searchRequestOutputBoth && len(data.Functions) > 0 {
		buf.WriteString("\n【函数 Schema JSON】\n")
		buf.WriteString(formatSearchFunctionSchemaJSON(data.Functions))
	}

	return strings.TrimSpace(buf.String())
}

func formatSearchFunctionSchemaJSON(functions []*dto.FunctionSearchResult) string {
	var buf strings.Builder
	for i, fn := range functions {
		buf.WriteString(fmt.Sprintf("%d. %s\n", i+1, fn.Name))
		buf.WriteString("   full_code_path: ")
		buf.WriteString(fn.FullCodePath)
		buf.WriteString("\n")
		if fn.RunCount > 0 {
			buf.WriteString(fmt.Sprintf("   已使用 %d 次\n", fn.RunCount))
		}
		if fn.Description != "" {
			buf.WriteString("   description: ")
			buf.WriteString(fn.Description)
			buf.WriteString("\n")
		}
		if fn.TemplateType != "" {
			buf.WriteString("   type: ")
			buf.WriteString(fn.TemplateType)
			buf.WriteString("\n")
		}
		if caps := formatSearchFunctionCapabilities(fn.TemplateType, fn.Callbacks); caps != "" {
			buf.WriteString("   capabilities: ")
			buf.WriteString(caps)
			buf.WriteString("\n")
		}
		if fn.Schema != nil {
			if schemaJSON, err := json.MarshalIndent(fn.Schema, "   ", "  "); err == nil {
				buf.WriteString("   schema: ")
				buf.Write(schemaJSON)
				buf.WriteString("\n")
			}
		}
	}
	return buf.String()
}

func formatSearchDocMatchSummary(index int, item searchDocMatch) string {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("%d. %s\n", index+1, firstNonEmptyString(item.Name, item.FullCodePath)))
	buf.WriteString("   full_code_path: ")
	buf.WriteString(item.FullCodePath)
	buf.WriteString("\n")
	if item.Line > 0 {
		buf.WriteString(fmt.Sprintf("   line: %d\n", item.Line))
	}
	if item.Snippet != "" {
		buf.WriteString("   snippet: ")
		buf.WriteString(item.Snippet)
		buf.WriteString("\n")
	}
	return buf.String()
}

func formatSearchFunctionSummary(index int, fn *dto.FunctionSearchResult) string {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("%d. %s\n", index+1, fn.Name))
	buf.WriteString("   full_code_path: ")
	buf.WriteString(fn.FullCodePath)
	buf.WriteString("\n")
	if fn.TemplateType != "" {
		buf.WriteString("   type: ")
		buf.WriteString(fn.TemplateType)
		buf.WriteString("\n")
	}
	if caps := formatSearchFunctionCapabilities(fn.TemplateType, fn.Callbacks); caps != "" {
		buf.WriteString("   capabilities: ")
		buf.WriteString(caps)
		buf.WriteString("\n")
	}
	if fn.RunCount > 0 {
		buf.WriteString(fmt.Sprintf("   已使用 %d 次\n", fn.RunCount))
	}
	if fn.Description != "" {
		buf.WriteString("   description: ")
		buf.WriteString(fn.Description)
		buf.WriteString("\n")
	}

	summaryLines := summarizeSearchSchema(fn.Schema)
	if len(summaryLines) > 0 {
		buf.WriteString("   字段摘要:\n")
		for _, line := range summaryLines {
			buf.WriteString("   ")
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}
	return buf.String()
}

func formatSearchResourceSummary(index int, item *dto.ResourceSearchResult) string {
	if item == nil {
		return ""
	}
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("%d. %s\n", index+1, item.Name))
	buf.WriteString("   type: ")
	buf.WriteString(item.Type)
	if item.TemplateType != "" {
		buf.WriteString(" / ")
		buf.WriteString(item.TemplateType)
	}
	buf.WriteString("\n")
	buf.WriteString("   full_code_path: ")
	buf.WriteString(item.FullCodePath)
	buf.WriteString("\n")
	if item.AppUser != "" || item.AppCode != "" {
		buf.WriteString(fmt.Sprintf("   app: %s/%s\n", item.AppUser, item.AppCode))
	}
	if item.MatchSource != "" {
		buf.WriteString("   match_source: ")
		buf.WriteString(item.MatchSource)
		buf.WriteString("\n")
	}
	if item.Description != "" {
		buf.WriteString("   description: ")
		buf.WriteString(item.Description)
		buf.WriteString("\n")
	}
	if item.Snippet != "" && item.Snippet != item.Description {
		buf.WriteString("   snippet: ")
		buf.WriteString(item.Snippet)
		buf.WriteString("\n")
	}
	if item.RunCount > 0 {
		buf.WriteString(fmt.Sprintf("   已使用 %d 次\n", item.RunCount))
	}
	return buf.String()
}

func formatSearchFunctionCapabilities(templateType string, callbacks []string) string {
	switch templateType {
	case functionschema.TypeTable:
		caps := []string{"read"}
		if hasSearchCallback(callbacks, "OnTableAddRow") {
			caps = append(caps, "create")
		}
		if hasSearchCallback(callbacks, "OnTableUpdateRow") {
			caps = append(caps, "update")
		}
		if hasSearchCallback(callbacks, "OnTableDeleteRows") {
			caps = append(caps, "delete")
		}
		if len(caps) == 1 {
			return "read-only"
		}
		return strings.Join(caps, ", ")
	case functionschema.TypeForm:
		return "submit"
	case functionschema.TypeChart:
		return "query"
	default:
		return ""
	}
}

func hasSearchCallback(callbacks []string, target string) bool {
	for _, callback := range callbacks {
		if callback == target {
			return true
		}
	}
	return false
}

func summarizeSearchSchema(schema *functionschema.FunctionSchema) []string {
	if schema == nil {
		return nil
	}
	switch schema.Type {
	case functionschema.TypeForm:
		if schema.Form == nil {
			return nil
		}
		return summarizeSearchFields("输入字段", schema.Form.Request)
	case functionschema.TypeTable:
		raw, _ := functionschema.Marshal(schema)
		lines := summarizeSearchFields("搜索字段", functionschema.TableSearchFields(raw))
		lines = append(lines, summarizeSearchFields("列表字段", functionschema.TableListFields(raw))...)
		lines = append(lines, summarizeSearchFields("新增字段", functionschema.TableCreateFields(raw))...)
		lines = append(lines, summarizeSearchFields("编辑字段", functionschema.TableUpdateFields(raw))...)
		return lines
	case functionschema.TypeChart:
		if schema.Chart == nil {
			return nil
		}
		return summarizeSearchFields("查询字段", schema.Chart.Request)
	default:
		return nil
	}
}

func summarizeSearchFields(label string, fields []*widget.Field) []string {
	if len(fields) == 0 {
		return nil
	}
	lines := make([]string, 0, len(fields)*2+1)
	lines = append(lines, label+"：")
	for _, field := range fields {
		for _, line := range field.LLMSummaryLines(widget.SummaryOptions{
			Mode:     widget.SummaryCompact,
			MaxDepth: 1,
		}) {
			lines = append(lines, "  "+line)
		}
	}
	return lines
}

func searchToolShortDescription(description string) string {
	description = strings.TrimSpace(description)
	if description == "" {
		return ""
	}
	for _, sep := range []string{"。参数：", "。full_code_path", "\n", "。"} {
		if idx := strings.Index(description, sep); idx > 0 {
			return strings.TrimSpace(description[:idx])
		}
	}
	return description
}
