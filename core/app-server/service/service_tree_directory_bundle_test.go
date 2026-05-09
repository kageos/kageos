package service

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

func TestBuildDirectoryBundleNodeUsesMinimalTreeAndSkipsInitFile(t *testing.T) {
	root := &model.ServiceTree{Code: "tools", Name: "工具", Description: "root", FullCodePath: "/system/tools"}
	child := &model.ServiceTree{Code: "excel", Name: "Excel", Description: "excel tools", FullCodePath: "/system/tools/excel"}

	node := buildDirectoryBundleNode(root, []*model.ServiceTree{root, child}, map[string][]*model.FileSnapshot{
		"/system/tools": {
			{FileName: "init_", RelativePath: "init_.go", Content: "generated", FileType: "go"},
			{FileName: "readme", RelativePath: "readme.md", Content: "# tools", FileType: "md"},
		},
		"/system/tools/excel": {
			{FileName: "reader", RelativePath: "reader.go", Content: "package excel", FileType: "go"},
		},
	})

	if node.Code != "tools" || node.Name != "工具" || node.Description != "root" {
		t.Fatalf("unexpected root node: %#v", node)
	}
	gotRootFiles := bundleFilePaths(node.Files)
	if !reflect.DeepEqual(gotRootFiles, []string{"readme.md"}) {
		t.Fatalf("unexpected root files: %#v", gotRootFiles)
	}
	if len(node.Children) != 1 || node.Children[0].Code != "excel" {
		t.Fatalf("unexpected children: %#v", node.Children)
	}
	gotChildFiles := bundleFilePaths(node.Children[0].Files)
	if !reflect.DeepEqual(gotChildFiles, []string{"reader.go"}) {
		t.Fatalf("unexpected child files: %#v", gotChildFiles)
	}
}

func TestBuildDirectoryBundleInstallItemsPastesRootUnderTarget(t *testing.T) {
	bundle := &dto.DirectoryBundle{
		SchemaVersion: dto.DirectoryBundleSchemaVersion,
		Root: &dto.DirectoryBundleNode{
			Code:        "tools",
			Name:        "工具",
			Description: "root",
			Files: []*dto.DirectoryBundleFile{
				{Path: "readme.md", Content: "# tools"},
			},
			Children: []*dto.DirectoryBundleNode{
				{
					Code: "excel",
					Name: "Excel",
					Files: []*dto.DirectoryBundleFile{
						{Path: "reader.go", Content: "package excel"},
					},
				},
			},
		},
	}
	if err := validateDirectoryBundle(bundle); err != nil {
		t.Fatalf("expected valid bundle: %v", err)
	}

	directoryItems := make([]*dto.DirectoryScaffoldItem, 0)
	fileItems := make([]*dto.FileWriteItem, 0)
	if err := buildDirectoryBundleInstallItems(bundle.Root, "/target/app/foo", &directoryItems, &fileItems); err != nil {
		t.Fatal(err)
	}

	gotDirs := make([]string, 0, len(directoryItems))
	for _, item := range directoryItems {
		gotDirs = append(gotDirs, item.FullCodePath)
	}
	wantDirs := []string{"/target/app/foo/tools", "/target/app/foo/tools/excel"}
	if !reflect.DeepEqual(gotDirs, wantDirs) {
		t.Fatalf("unexpected directories:\nwant=%#v\ngot=%#v", wantDirs, gotDirs)
	}

	gotFiles := make([]string, 0, len(fileItems))
	for _, item := range fileItems {
		gotFiles = append(gotFiles, item.FullCodePath+"::"+item.FileName+"."+item.FileType)
	}
	wantFiles := []string{
		"/target/app/foo/tools::readme.md",
		"/target/app/foo/tools/excel::reader.go",
	}
	if !reflect.DeepEqual(gotFiles, wantFiles) {
		t.Fatalf("unexpected files:\nwant=%#v\ngot=%#v", wantFiles, gotFiles)
	}
}

func TestValidateDirectoryBundleRejectsGeneratedInitFile(t *testing.T) {
	err := validateDirectoryBundle(&dto.DirectoryBundle{
		SchemaVersion: dto.DirectoryBundleSchemaVersion,
		Root: &dto.DirectoryBundleNode{
			Code: "tools",
			Name: "工具",
			Files: []*dto.DirectoryBundleFile{
				{Path: "init_.go", Content: "generated"},
			},
		},
	})
	if err == nil {
		t.Fatal("expected init_.go validation error")
	}
	if !strings.Contains(err.Error(), "init_.go") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func bundleFilePaths(files []*dto.DirectoryBundleFile) []string {
	out := make([]string, 0, len(files))
	for _, file := range files {
		out = append(out, file.Path)
	}
	return out
}
