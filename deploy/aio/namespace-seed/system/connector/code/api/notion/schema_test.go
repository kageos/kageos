package notion

import (
	"strings"
	"testing"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/widget"
)

func TestNotionTemplatesDecodeSchema(t *testing.T) {
	formTemplates := map[string]*app.FormTemplate{
		"NotionConnectionStatusTemplate": NotionConnectionStatusTemplate,
		"NotionMeTemplate":               NotionMeTemplate,
		"NotionPageContentTemplate":      NotionPageContentTemplate,
		"NotionPageDetailTemplate":       NotionPageDetailTemplate,
	}
	for name, template := range formTemplates {
		if _, _, err := widget.DecodeForm(notionTemplateCallbacks(template.OnSelectFuzzyMap), template.Request, template.Response); err != nil {
			t.Fatalf("%s schema decode failed: %v", name, err)
		}
	}

	tableTemplates := map[string]*app.TableTemplate{
		"NotionBlockChildrenTemplate": NotionBlockChildrenTemplate,
		"NotionSearchTemplate":        NotionSearchTemplate,
	}
	for name, template := range tableTemplates {
		if _, _, err := widget.DecodeTable(notionTemplateCallbacks(template.OnSelectFuzzyMap), template.Request, template.AutoCrudTable); err != nil {
			t.Fatalf("%s schema decode failed: %v", name, err)
		}
	}
}

func TestNotionBlocksTextIncludesNestedChildren(t *testing.T) {
	blocks := []map[string]interface{}{
		{
			"id":           "parent",
			"type":         "toggle",
			"has_children": true,
			"toggle": map[string]interface{}{
				"rich_text": []interface{}{
					map[string]interface{}{"plain_text": "模型配置"},
				},
			},
			"children": []map[string]interface{}{
				{
					"id":   "child",
					"type": "bulleted_list_item",
					"bulleted_list_item": map[string]interface{}{
						"rich_text": []interface{}{
							map[string]interface{}{"plain_text": "可用模型: gpt-5"},
						},
					},
				},
			},
		},
		{
			"id":   "row",
			"type": "table_row",
			"table_row": map[string]interface{}{
				"cells": []interface{}{
					[]interface{}{map[string]interface{}{"plain_text": "字段"}},
					[]interface{}{map[string]interface{}{"plain_text": "可用模型"}},
				},
			},
		},
	}

	got := notionBlocksText(blocks)
	if !strings.Contains(got, "可用模型: gpt-5") {
		t.Fatalf("nested block text missing from content preview: %q", got)
	}
	if !strings.Contains(got, "| 字段 | 可用模型 |") {
		t.Fatalf("table row text missing from content preview: %q", got)
	}
}

func notionTemplateCallbacks(fuzzy map[string]app.OnSelectFuzzy) map[string][]string {
	if len(fuzzy) == 0 {
		return nil
	}
	callbacks := make(map[string][]string, len(fuzzy))
	for field := range fuzzy {
		callbacks[field] = []string{app.CallbackTypeOnSelectFuzzy}
	}
	return callbacks
}
