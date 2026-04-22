package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ai-agent-os/hub/backend/dto"
)

func buildValidNestedHubDirectoryTree() *dto.DirectoryTreeNode {
	return &dto.DirectoryTreeNode{
		Type: "package",
		Name: "供应商",
		Code: "vendors",
		Path: "/luobei/demo/vendors",
		Files: []*dto.FileSnapshotInfo{
			{
				FileName:     "vendors_readme",
				RelativePath: "README.md",
				Content:      "# vendors",
				FileType:     "md",
			},
		},
		Functions: []*dto.HubFunctionInfo{
			{
				Name:         "供应商列表",
				Code:         "vendor_list",
				FullCodePath: "/luobei/demo/vendors/vendor_list.table",
				TemplateType: "table",
			},
		},
		Subdirectories: []*dto.DirectoryTreeNode{
			{
				Type: "package",
				Name: "区域",
				Code: "regions",
				Path: "/luobei/demo/vendors/regions",
				Subdirectories: []*dto.DirectoryTreeNode{
					{
						Type: "package",
						Name: "华北",
						Code: "north",
						Path: "/luobei/demo/vendors/regions/north",
						Files: []*dto.FileSnapshotInfo{
							{
								FileName:     "north_vendor",
								RelativePath: "north_vendor.json",
								Content:      "{\"vendor\":\"beijing\"}",
								FileType:     "json",
							},
						},
						Functions: []*dto.HubFunctionInfo{
							{
								Name:         "华北供应商详情",
								Code:         "north_vendor_detail",
								FullCodePath: "/luobei/demo/vendors/regions/north/north_vendor_detail.form",
								TemplateType: "form",
							},
						},
					},
				},
			},
		},
	}
}

func TestValidateDirectoryTreeForPersistence_RejectsPathMismatch(t *testing.T) {
	tree := buildValidNestedHubDirectoryTree()
	tree.Subdirectories[0].Subdirectories[0].Path = "/luobei/demo/vendors/regions/wrong"

	err := validateDirectoryTreeForPersistence(tree, "/luobei/demo/vendors")
	if err == nil {
		t.Fatal("expected path mismatch validation error")
	}
	if !strings.Contains(err.Error(), "目录 path 与父目录不一致") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateDirectoryTreeForPersistence_RejectsInvalidFunctionSchema(t *testing.T) {
	tree := buildValidNestedHubDirectoryTree()
	tree.Functions[0].Schema = []byte(`{"version":1,"type":"form","form":{"request":[],"response":[]}}`)

	err := validateDirectoryTreeForPersistence(tree, "/luobei/demo/vendors")
	if err == nil {
		t.Fatal("expected invalid function schema validation error")
	}
	if !strings.Contains(err.Error(), "template_type 与 schema.type 不一致") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSplitAndMergeSnapshotParts_PreservesNestedStructureAndFileContent(t *testing.T) {
	tree := buildValidNestedHubDirectoryTree()

	treeJSON, filesJSON, defsJSON, err := splitDirectoryTreeIntoSnapshotParts(tree)
	if err != nil {
		t.Fatalf("splitDirectoryTreeIntoSnapshotParts returned error: %v", err)
	}

	var defs []*dto.HubFunctionInfo
	if err := json.Unmarshal([]byte(defsJSON), &defs); err != nil {
		t.Fatalf("failed to unmarshal defs json: %v", err)
	}
	if len(defs) != 2 {
		t.Fatalf("expected 2 flattened function defs, got %d", len(defs))
	}

	mergedTree, err := mergeSnapshotPartsIntoTree(treeJSON, filesJSON)
	if err != nil {
		t.Fatalf("mergeSnapshotPartsIntoTree returned error: %v", err)
	}
	if mergedTree == nil {
		t.Fatal("expected merged tree")
	}

	northNode := mergedTree.Subdirectories[0].Subdirectories[0]
	if northNode.Code != "north" {
		t.Fatalf("expected nested node code north, got %s", northNode.Code)
	}
	if len(northNode.Files) != 1 || northNode.Files[0].Content != "{\"vendor\":\"beijing\"}" {
		t.Fatalf("expected nested file content to be restored, got %#v", northNode.Files)
	}
	if len(northNode.Functions) != 1 || northNode.Functions[0].Code != "north_vendor_detail" {
		t.Fatalf("expected nested functions to be preserved, got %#v", northNode.Functions)
	}
}
