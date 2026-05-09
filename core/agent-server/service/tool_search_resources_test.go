package service

import (
	"strings"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func TestNormalizeSearchResourcesType(t *testing.T) {
	cases := map[string]string{
		"":         "all",
		"all":      "all",
		"doc":      "docs",
		"document": "docs",
		"function": "function",
		"weird":    "all",
	}
	for input, want := range cases {
		if got := normalizeSearchResourcesType(input); got != want {
			t.Fatalf("normalizeSearchResourcesType(%q)=%q want %q", input, got, want)
		}
	}
}

func TestFormatSearchResourcesOutputIncludesResourceMetadata(t *testing.T) {
	out := formatSearchResourcesOutput(searchResourcesResultData{
		Keyword:      "工单",
		ResourceType: "function",
		Scope:        searchScopeCurrentApp,
		User:         "beiluo",
		App:          "demo",
		Page:         1,
		PageSize:     20,
		Total:        1,
		Items: []*dto.ResourceSearchResult{
			{
				Name:         "工单列表",
				Type:         "function",
				TemplateType: "table",
				FullCodePath: "/beiluo/demo/tickets/ticket_list.table",
				AppUser:      "beiluo",
				AppCode:      "demo",
				MatchSource:  "node",
				Description:  "查询工单",
				RunCount:     3,
			},
		},
	})

	for _, want := range []string{
		"服务树搜索结果：关键词",
		"范围=current_app",
		"resource_type=function",
		"full_code_path: /beiluo/demo/tickets/ticket_list.table",
		"type: function / table",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output should contain %q, got:\n%s", want, out)
		}
	}
}
