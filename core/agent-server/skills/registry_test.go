package skills

import (
	"strings"
	"testing"
)

func TestLoadEmbeddedSkills(t *testing.T) {
	registry := LoadEmbedded()
	if err := registry.LoadError(); err != nil {
		t.Fatalf("LoadEmbedded returned errors: %v", err)
	}
	if got := len(registry.All()); got != 26 {
		t.Fatalf("embedded skill count = %d, want 26", got)
	}
	if _, ok := registry.Get("sop.create-project"); !ok {
		t.Fatal("expected sop.create-project to be loadable by id")
	}
	if _, ok := registry.Get("create-project"); !ok {
		t.Fatal("expected create-project to be loadable by name")
	}
	if _, ok := registry.Get("sop/create-project"); !ok {
		t.Fatal("expected sop/create-project to be loadable by path prefix")
	}
	if _, ok := registry.Get("system.openapi"); !ok {
		t.Fatal("expected system.openapi to be loadable by id")
	}
	if _, ok := registry.Get("system.openapi.message"); !ok {
		t.Fatal("expected system.openapi.message to be loadable by id")
	}
	if _, ok := registry.Get("system.tools"); !ok {
		t.Fatal("expected system.tools to be loadable by id")
	}
	if _, ok := registry.Get("system.tools.pdf"); !ok {
		t.Fatal("expected system.tools.pdf to be loadable by id")
	}
	if _, ok := registry.Get("sdk.widget-selection"); !ok {
		t.Fatal("expected sdk.widget-selection to be loadable by id")
	}
	for _, id := range []string{
		"sdk.form-submit-basic",
		"sdk.table-crud-basic",
		"sdk.combo-table-form",
		"sdk.combo-table-form-chart",
	} {
		if _, ok := registry.Get(id); !ok {
			t.Fatalf("expected %s to be loadable by id", id)
		}
	}
}

func TestSearchOpenAPISkill(t *testing.T) {
	registry := LoadEmbedded()
	results := registry.Search(SearchOptions{
		Keyword: "平台接口 hub 发送消息",
		Mode:    "execute",
		Limit:   3,
	})
	if len(results) == 0 {
		t.Fatal("expected openapi search results")
	}
	if results[0].Meta.ID != "system.openapi" {
		t.Fatalf("top result = %s, want system.openapi", results[0].Meta.ID)
	}
}

func TestLegacySkillIDsAreNotLoadableOrSearchable(t *testing.T) {
	registry := LoadEmbedded()
	legacyIDs := map[string]struct{}{
		"openapi.platform":        {},
		"openapi.hub":             {},
		"openapi.message":         {},
		"openapi.scheduled-task":  {},
		"openapi.permission":      {},
		"openapi.audit":           {},
		"tools.official":          {},
		"tools.official.pdf":      {},
		"tools.official.image":    {},
		"tools.official.document": {},
		"tools.official.table":    {},
		"tools.official.runtime":  {},
	}
	for legacyID := range legacyIDs {
		if skill, ok := registry.Get(legacyID); ok {
			t.Fatalf("legacy skill id %q should not be loadable, got %s", legacyID, skill.Meta.ID)
		}
	}
	for _, result := range registry.Search(SearchOptions{Keyword: "openapi tools official message hub pdf", Mode: "execute", Limit: 50}) {
		if _, ok := legacyIDs[result.Meta.ID]; ok {
			t.Fatalf("legacy skill id %q should not appear in search results: %#v", result.Meta.ID, result)
		}
	}
}

func TestSearchSpecificOpenAPISkills(t *testing.T) {
	registry := LoadEmbedded()
	cases := []struct {
		name    string
		keyword string
		want    string
	}{
		{name: "message", keyword: "发送消息 通知用户 message-server", want: "system.openapi.message"},
		{name: "hub", keyword: "Hub 发布 推送 复制", want: "system.openapi.hub"},
		{name: "scheduled task", keyword: "创建定时任务 cron schedule", want: "system.openapi.scheduled-task"},
		{name: "permission", keyword: "权限查询 申请权限 审批", want: "system.openapi.permission"},
		{name: "audit", keyword: "操作日志 审计 变更记录", want: "system.openapi.audit"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results := registry.Search(SearchOptions{
				Keyword: tc.keyword,
				Mode:    "execute",
				Limit:   3,
			})
			if len(results) == 0 {
				t.Fatalf("expected search results for %q", tc.keyword)
			}
			if results[0].Meta.ID != tc.want {
				t.Fatalf("top result = %s, want %s; results=%#v", results[0].Meta.ID, tc.want, results)
			}
		})
	}
}

func TestSearchSystemToolsSkills(t *testing.T) {
	registry := LoadEmbedded()
	cases := []struct {
		name    string
		keyword string
		want    string
	}{
		{name: "image", keyword: "图片压缩 图片转换 缩略图", want: "system.tools.image"},
		{name: "pdf", keyword: "PDF 合并 拆分 OCR", want: "system.tools.pdf"},
		{name: "table", keyword: "Excel CSV 表格清洗", want: "system.tools.table"},
		{name: "runtime", keyword: "Python 一次性脚本 run_official_python", want: "system.tools.runtime"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results := registry.Search(SearchOptions{
				Keyword: tc.keyword,
				Mode:    "execute",
				Limit:   3,
			})
			if len(results) == 0 {
				t.Fatalf("expected system tools search results for %q", tc.keyword)
			}
			if results[0].Meta.ID != tc.want {
				t.Fatalf("top result = %s, want %s; results=%#v", results[0].Meta.ID, tc.want, results)
			}
		})
	}
}

func TestSearchSDKSkill(t *testing.T) {
	registry := LoadEmbedded()
	results := registry.Search(SearchOptions{
		Keyword: "[]int 数组 item_type widget 组件选择",
		Mode:    "dev",
		Limit:   3,
	})
	if len(results) == 0 {
		t.Fatal("expected sdk search results")
	}
	if results[0].Meta.ID != "sdk.widget-selection" {
		t.Fatalf("top result = %s, want sdk.widget-selection", results[0].Meta.ID)
	}
}

func TestSearchSkillsByKeywordAndMode(t *testing.T) {
	registry := LoadEmbedded()
	results := registry.Search(SearchOptions{
		Keyword: "创建项目",
		Mode:    "execute",
		Limit:   3,
	})
	if len(results) == 0 {
		t.Fatal("expected search results")
	}
	if results[0].Meta.ID != "sop.create-project" {
		t.Fatalf("top result = %s, want sop.create-project", results[0].Meta.ID)
	}

	results = registry.Search(SearchOptions{
		Keyword: "解释",
		Mode:    "modify",
		Limit:   10,
	})
	if len(results) == 0 {
		t.Fatal("expected modify/explain search results")
	}
	for _, result := range results {
		if result.Meta.ID == "sop.create-project" {
			t.Fatalf("create-project should not match modify mode for keyword 解释: %#v", results)
		}
	}
}

func TestParseSkillValidatesFrontmatter(t *testing.T) {
	_, err := ParseSkill("# missing frontmatter", "bad/SKILL.md")
	if err == nil || !strings.Contains(err.Error(), "missing YAML frontmatter") {
		t.Fatalf("ParseSkill error = %v, want missing frontmatter", err)
	}

	skill, err := ParseSkill(`---
id: demo.skill
name: demo
description: demo skill
triggers:
  - demo
  - demo
---

# Body
`, "demo/SKILL.md")
	if err != nil {
		t.Fatalf("ParseSkill returned error: %v", err)
	}
	if skill.Meta.ID != "demo.skill" || skill.Meta.Path != "demo/SKILL.md" {
		t.Fatalf("unexpected meta: %#v", skill.Meta)
	}
	if len(skill.Meta.Triggers) != 1 {
		t.Fatalf("triggers = %#v, want deduplicated single trigger", skill.Meta.Triggers)
	}
}
