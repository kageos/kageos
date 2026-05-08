package service

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func buildValidNestedAppDirectoryTree() *dto.DirectoryTreeNode {
	return &dto.DirectoryTreeNode{
		Type: "package",
		Name: "招聘",
		Code: "recruitment",
		Path: "/luobei/demo/recruitment",
		Files: []*dto.FileSnapshotInfo{
			{
				FileName:     "README",
				RelativePath: "README.md",
				Content:      "# recruitment",
				FileType:     "md",
			},
		},
		Subdirectories: []*dto.DirectoryTreeNode{
			{
				Type: "package",
				Name: "职位",
				Code: "jobs",
				Path: "/luobei/demo/recruitment/jobs",
				Files: []*dto.FileSnapshotInfo{
					{
						FileName:     "job_handler",
						RelativePath: "job_handler.go",
						Content:      "package jobs",
						FileType:     "go",
					},
				},
				Subdirectories: []*dto.DirectoryTreeNode{
					{
						Type: "package",
						Name: "归档",
						Code: "archive",
						Path: "/luobei/demo/recruitment/jobs/archive",
						Files: []*dto.FileSnapshotInfo{
							{
								FileName:     "archive_jobs",
								RelativePath: "archive_jobs.json",
								Content:      "{\"ok\":true}",
								FileType:     "json",
							},
						},
					},
				},
			},
		},
	}
}

func TestValidateHubDirectoryTreeForPublish_AllowsDeepNestedTree(t *testing.T) {
	tree := buildValidNestedAppDirectoryTree()

	if err := validateHubDirectoryTreeForPublishImpl(tree); err != nil {
		t.Fatalf("expected nested tree to pass validation, got error: %v", err)
	}
}

func TestValidateHubDirectoryTreeForPublish_RejectsDuplicateSiblingCodes(t *testing.T) {
	tree := buildValidNestedAppDirectoryTree()
	tree.Subdirectories = append(tree.Subdirectories, &dto.DirectoryTreeNode{
		Type: "package",
		Name: "重复职位",
		Code: "jobs",
		Path: "/luobei/demo/recruitment/jobs-duplicate",
	})

	err := validateHubDirectoryTreeForPublishImpl(tree)
	if err == nil {
		t.Fatal("expected duplicate sibling code validation error")
	}
	if !strings.Contains(err.Error(), "重复子目录 code: jobs") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateHubDirectoryTreeRejectsInvalidGoPackageCode(t *testing.T) {
	tree := buildValidNestedAppDirectoryTree()
	tree.Subdirectories[0].Code = "job-center"

	err := validateHubDirectoryTreeForInstallImpl(tree)
	if err == nil {
		t.Fatal("expected invalid package code validation error")
	}
	if !strings.Contains(err.Error(), "合法 Go package 名称") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateHubDirectoryTreeForInstall_RejectsEmptyNestedCode(t *testing.T) {
	tree := buildValidNestedAppDirectoryTree()
	tree.Subdirectories[0].Subdirectories[0].Code = ""
	tree.Subdirectories[0].Subdirectories[0].Path = ""

	err := validateHubDirectoryTreeForInstallImpl(tree)
	if err == nil {
		t.Fatal("expected empty nested code validation error")
	}
	if !strings.Contains(err.Error(), "目录 code 不能为空") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateHubDirectoryTreeRejectsInvalidFunctionSchema(t *testing.T) {
	tree := buildValidNestedAppDirectoryTree()
	tree.Functions = []*dto.HubFunctionInfo{
		{
			Name:         "坏 schema",
			Code:         "bad_schema",
			TemplateType: "table",
			Schema:       []byte(`{"version":1,"type":"form","form":{"request":[],"response":[]}}`),
		},
	}

	err := validateHubDirectoryTreeForInstallImpl(tree)
	if err == nil {
		t.Fatal("expected invalid function schema validation error")
	}
	if !strings.Contains(err.Error(), "template_type 与 schema.type 不一致") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildItemsFromTree_ExpandsNestedDirectoriesRecursively(t *testing.T) {
	service := &serviceTreeHubService{}
	tree := buildValidNestedAppDirectoryTree()

	directoryItems := make([]*dto.DirectoryScaffoldItem, 0)
	fileItems := make([]*dto.FileWriteItem, 0)
	service.buildItemsFromTree(tree, "/target/workspace", &directoryItems, &fileItems)

	gotDirectoryPaths := make([]string, 0, len(directoryItems))
	for _, item := range directoryItems {
		gotDirectoryPaths = append(gotDirectoryPaths, item.FullCodePath)
	}
	wantDirectoryPaths := []string{
		"/target/workspace/recruitment",
		"/target/workspace/recruitment/jobs",
		"/target/workspace/recruitment/jobs/archive",
	}
	if !reflect.DeepEqual(gotDirectoryPaths, wantDirectoryPaths) {
		t.Fatalf("unexpected directory paths:\nwant: %#v\ngot:  %#v", wantDirectoryPaths, gotDirectoryPaths)
	}

	gotFileLocations := make([]string, 0, len(fileItems))
	for _, item := range fileItems {
		gotFileLocations = append(gotFileLocations, item.FullCodePath+"::"+item.RelativePath)
	}
	wantFileLocations := []string{
		"/target/workspace/recruitment::README.md",
		"/target/workspace/recruitment/jobs::job_handler.go",
		"/target/workspace/recruitment/jobs/archive::archive_jobs.json",
	}
	if !reflect.DeepEqual(gotFileLocations, wantFileLocations) {
		t.Fatalf("unexpected file locations:\nwant: %#v\ngot:  %#v", wantFileLocations, gotFileLocations)
	}
}
