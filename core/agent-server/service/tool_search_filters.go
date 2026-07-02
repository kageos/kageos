package service

import (
	"context"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/functionschema"
)

func shouldSearchBuiltinTools(resourceType string, keyword string) bool {
	return resourceType == "tool" || (resourceType == "all" && strings.TrimSpace(keyword) != "")
}

func shouldSearchFunctions(resourceType string, templateType string, capability string, fullCodePath string) bool {
	if templateType != "" || capability != "" {
		return true
	}
	return resourceType == "all" || resourceType == "function" || looksLikeFunctionPath(fullCodePath)
}

func shouldSearchResources(resourceType string) bool {
	return resourceType == "all" || resourceType == "directory" || resourceType == "docs"
}

func looksLikeFunctionPath(fullCodePath string) bool {
	for _, suffix := range []string{".form", ".table", ".chart"} {
		if strings.HasSuffix(fullCodePath, suffix) {
			return true
		}
	}
	return false
}

func searchBuiltinTools(ctx context.Context, registry *ToolRegistry, keyword string, page int, pageSize int) []dto.ToolDef {
	if registry == nil {
		return nil
	}
	allTools, _ := registry.ListTools(ctx, nil)
	keywords := splitSearchKeywords(keyword)
	matched := make([]dto.ToolDef, 0, len(allTools))
	if len(keywords) == 0 {
		matched = append(matched, allTools...)
		return paginateSearchToolDefs(matched, page, pageSize)
	}
	lowerKeywords := make([]string, len(keywords))
	for i, k := range keywords {
		lowerKeywords[i] = strings.ToLower(k)
	}
	for _, tool := range allTools {
		text := strings.ToLower(tool.Name + " " + tool.Description)
		for _, k := range lowerKeywords {
			if strings.Contains(text, k) {
				matched = append(matched, tool)
				break
			}
		}
	}
	return matched
}

func paginateSearchToolDefs(tools []dto.ToolDef, page int, pageSize int) []dto.ToolDef {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || len(tools) == 0 {
		return tools
	}
	start := (page - 1) * pageSize
	if start >= len(tools) {
		return nil
	}
	end := start + pageSize
	if end > len(tools) {
		end = len(tools)
	}
	return tools[start:end]
}

func filterSearchResourceItemsByType(items []*dto.ResourceSearchResult, resourceType string, include bool) []*dto.ResourceSearchResult {
	if resourceType == "" {
		return items
	}
	out := make([]*dto.ResourceSearchResult, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		matches := item.Type == resourceType
		if matches == include {
			out = append(out, item)
		}
	}
	return out
}

func filterSearchFunctionsByFullCodePath(functions []*dto.FunctionSearchResult, fullCodePath string) []*dto.FunctionSearchResult {
	fullCodePath = normalizeWorkspacePath(fullCodePath)
	if fullCodePath == "" {
		return functions
	}
	out := make([]*dto.FunctionSearchResult, 0, len(functions))
	for _, fn := range functions {
		if fn == nil {
			continue
		}
		if workspacePathHasPrefix(fn.FullCodePath, fullCodePath) {
			out = append(out, fn)
		}
	}
	return out
}

func paginateSearchFunctions(functions []*dto.FunctionSearchResult, page int, pageSize int) []*dto.FunctionSearchResult {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || len(functions) == 0 {
		return functions
	}
	start := (page - 1) * pageSize
	if start >= len(functions) {
		return nil
	}
	end := start + pageSize
	if end > len(functions) {
		end = len(functions)
	}
	return functions[start:end]
}

func normalizeSearchResourceType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "all":
		return "all"
	case "tool", "tools", "builtin", "built_in", "built-in":
		return "tool"
	case "function", "functions":
		return "function"
	case "directory", "directories", "dir", "folder", "package":
		return "directory"
	case "docs", "doc", "document":
		return "docs"
	default:
		return "all"
	}
}

func apiSearchResourceType(resourceType string) string {
	switch resourceType {
	case "directory":
		return "package"
	case "docs":
		return "docs"
	default:
		return "all"
	}
}

func normalizeSearchCapability(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "read", "read-only", "create", "update", "delete", "submit", "query":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

func filterSearchFunctionsByCapability(functions []*dto.FunctionSearchResult, capability string) []*dto.FunctionSearchResult {
	if capability == "" {
		return functions
	}
	out := make([]*dto.FunctionSearchResult, 0, len(functions))
	for _, fn := range functions {
		if searchFunctionHasCapability(fn, capability) {
			out = append(out, fn)
		}
	}
	return out
}

func searchFunctionHasCapability(fn *dto.FunctionSearchResult, capability string) bool {
	if fn == nil {
		return false
	}
	switch capability {
	case "submit":
		return fn.TemplateType == functionschema.TypeForm
	case "query":
		return fn.TemplateType == functionschema.TypeChart
	case "read":
		return fn.TemplateType == functionschema.TypeTable
	case "read-only":
		return fn.TemplateType == functionschema.TypeTable &&
			!hasSearchCallback(fn.Callbacks, "OnTableAddRow") &&
			!hasSearchCallback(fn.Callbacks, "OnTableUpdateRow") &&
			!hasSearchCallback(fn.Callbacks, "OnTableDeleteRows")
	case "create":
		return fn.TemplateType == functionschema.TypeTable && hasSearchCallback(fn.Callbacks, "OnTableAddRow")
	case "update":
		return fn.TemplateType == functionschema.TypeTable && hasSearchCallback(fn.Callbacks, "OnTableUpdateRow")
	case "delete":
		return fn.TemplateType == functionschema.TypeTable && hasSearchCallback(fn.Callbacks, "OnTableDeleteRows")
	default:
		return true
	}
}
