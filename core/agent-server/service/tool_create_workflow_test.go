package service

import (
	"strings"
	"testing"
)

func TestNormalizeWorkflowDefinitionRawRejectsWrapperObject(t *testing.T) {
	_, err := normalizeWorkflowDefinitionRaw(`{
		"workflow_name": "测试工作流",
			"definition": {
				"schema_version": "workflow.v1",
				"mode": "graph",
				"nodes": [],
				"edges": []
			}
	}`)
	if err == nil {
		t.Fatal("expected wrapper object to be rejected")
	}
	if !strings.Contains(err.Error(), "不要传 {workflow_name, definition}") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeWorkflowDefinitionRawAcceptsDefinitionObject(t *testing.T) {
	raw, err := normalizeWorkflowDefinitionRaw(`{
		"schema_version": "workflow.v1",
		"mode": "graph",
		"nodes": [
			{"id":"start","type":"workflow.start","schema":{"version":1,"type":"form","form":{"request":[]}}},
			{"id":"output","type":"workflow.output","schema":{"version":1,"type":"form","form":{"response":[]}}}
		],
		"edges": [{"from":"start","to":"output"}]
	}`)
	if err != nil {
		t.Fatalf("normalizeWorkflowDefinitionRaw returned error: %v", err)
	}
	if !strings.Contains(string(raw), `"schema_version":"workflow.v1"`) {
		t.Fatalf("raw definition should be compacted workflow definition, got %s", string(raw))
	}
}

func TestParseWorkflowFullCodePathNormalizesSuffixAndParent(t *testing.T) {
	got, err := parseWorkflowFullCodePath("/liubeiluo/ee/a/test1")
	if err != nil {
		t.Fatalf("parseWorkflowFullCodePath returned error: %v", err)
	}
	if got.FullCodePath != "/liubeiluo/ee/a/test1.workflow" {
		t.Fatalf("full path=%s", got.FullCodePath)
	}
	if got.ParentPath != "/liubeiluo/ee/a" {
		t.Fatalf("parent path=%s", got.ParentPath)
	}
	if got.User != "liubeiluo" || got.App != "ee" || got.Code != "test1.workflow" {
		t.Fatalf("unexpected path parts: %#v", got)
	}
}
