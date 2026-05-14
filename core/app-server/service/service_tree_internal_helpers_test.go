package service

import (
	"reflect"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

func TestNormalizeSearchFunctionsPagination_DefaultsInvalidValues(t *testing.T) {
	page, pageSize := normalizeSearchFunctionsPagination(0, -1)
	if page != 1 || pageSize != 10 {
		t.Fatalf("unexpected normalized pagination: page=%d pageSize=%d", page, pageSize)
	}
}

func TestCalculateSearchFunctionsFetchSize_ExpandsFirstKeywordPage(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		keyword  string
		want     int
	}{
		{name: "default page size", page: 1, pageSize: 10, keyword: "alpha", want: 100},
		{name: "cap at 200", page: 1, pageSize: 30, keyword: "alpha", want: 200},
		{name: "no keyword", page: 1, pageSize: 10, keyword: "", want: 10},
		{name: "not first page", page: 2, pageSize: 10, keyword: "alpha", want: 10},
	}

	for _, tt := range tests {
		got := calculateSearchFunctionsFetchSize(tt.page, tt.pageSize, tt.keyword)
		if got != tt.want {
			t.Fatalf("%s: want %d, got %d", tt.name, tt.want, got)
		}
	}
}

func TestSplitSearchKeywordsForRelevance_TrimsAndDropsEmptyParts(t *testing.T) {
	got := splitSearchKeywordsForRelevance(" alpha | | beta|gamma ")
	want := []string{"alpha", "beta", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected keywords:\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestRankAndLimitSearchFunctions_FirstPageKeywordReranksByRelevance(t *testing.T) {
	trees := []*model.ServiceTree{
		{Name: "Alpha", Code: "misc", Tags: "tools", RunCount: 1},
		{Name: "Other", Code: "alpha", Tags: "misc", RunCount: 20},
		{Name: "Low", Code: "misc", Tags: "alpha", RunCount: 99},
	}

	got := rankAndLimitSearchFunctions(trees, "alpha", 1, 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 ranked items, got %d", len(got))
	}
	if got[0].Name != "Alpha" {
		t.Fatalf("expected exact name match first, got %s", got[0].Name)
	}
	if got[1].Code != "alpha" {
		t.Fatalf("expected exact code match second, got %s", got[1].Code)
	}
}

func TestGetParentPathForBatch(t *testing.T) {
	tests := []struct {
		fullCodePath string
		want         string
	}{
		{fullCodePath: "/user/app", want: ""},
		{fullCodePath: "/user/app/pkg", want: "/user/app"},
		{fullCodePath: "/user/app/pkg/sub", want: "/user/app/pkg"},
	}

	for _, tt := range tests {
		if got := getParentPathForBatch(tt.fullCodePath); got != tt.want {
			t.Fatalf("path %s: want %s, got %s", tt.fullCodePath, tt.want, got)
		}
	}
}

func TestFilterTreeByPermission_PromotesPermittedDescendant(t *testing.T) {
	queryView := &serviceTreeQueryView{}
	root := &dto.GetServiceTreeResp{
		ID:       1,
		Type:     model.ServiceTreeTypePackage,
		Children: []*dto.GetServiceTreeResp{},
	}
	hiddenParent := &dto.GetServiceTreeResp{
		ID:          2,
		Name:        "hidden",
		Type:        model.ServiceTreeTypePackage,
		Permissions: map[string]bool{"read": false, "write": false},
		Children: []*dto.GetServiceTreeResp{
			{
				ID:          3,
				Name:        "visible-func",
				Type:        model.ServiceTreeTypeFunction,
				Permissions: map[string]bool{"read": true},
			},
		},
	}
	root.Children = append(root.Children, hiddenParent)

	filtered := queryView.filterTreeByPermission(root)
	if len(filtered.Children) != 1 {
		t.Fatalf("expected promoted visible child, got %d children", len(filtered.Children))
	}
	if filtered.Children[0].Name != "visible-func" {
		t.Fatalf("expected visible descendant to be promoted, got %s", filtered.Children[0].Name)
	}
}

func TestServiceTreeQueryViewCalculateExpandedKeys_ExpandsRootAndPendingAncestors(t *testing.T) {
	trees := []*dto.GetServiceTreeResp{
		{
			ID:           1,
			Type:         model.ServiceTreeTypePackage,
			FullCodePath: "/user/app",
			Children: []*dto.GetServiceTreeResp{
				{
					ID:           2,
					Type:         model.ServiceTreeTypePackage,
					FullCodePath: "/user/app/pkg",
					Children: []*dto.GetServiceTreeResp{
						{
							ID:           3,
							Type:         model.ServiceTreeTypeFunction,
							FullCodePath: "/user/app/pkg/run",
							PendingCount: 2,
						},
					},
				},
			},
		},
	}

	expandedKeys := (&serviceTreeQueryView{}).calculateExpandedKeys(trees)
	got := make(map[int64]bool, len(expandedKeys))
	for _, id := range expandedKeys {
		got[id] = true
	}

	for _, wantID := range []int64{1, 2, 3} {
		if !got[wantID] {
			t.Fatalf("expected expanded key %d to exist, got %#v", wantID, expandedKeys)
		}
	}
}

func TestBuildBatchWriteFilesResp(t *testing.T) {
	runtimeResp := &dto.BatchWriteFilesRuntimeResp{
		FileCount:     2,
		WrittenPaths:  []string{"/alice/demo/a", "/alice/demo/b"},
		OldVersion:    "v3",
		NewVersion:    "v4",
		GitCommitHash: "abc123",
	}

	resp := buildBatchWriteFilesResp(runtimeResp, []string{"metadata warning"})
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.FileCount != 2 || len(resp.WrittenPaths) != 2 {
		t.Fatalf("unexpected file mapping: %+v", resp)
	}
	if resp.OldVersion != "v3" || resp.NewVersion != "v4" {
		t.Fatalf("unexpected version mapping: %+v", resp)
	}
	if len(resp.Warnings) != 1 || resp.Warnings[0] != "metadata warning" {
		t.Fatalf("unexpected warnings: %#v", resp.Warnings)
	}
}

func TestPlanCopyDirectoryTargets_MapsRootAndDescendants(t *testing.T) {
	sourceTrees := map[string]*model.ServiceTree{
		"/alice/source/tools":       {FullCodePath: "/alice/source/tools", Name: "Tools", Code: "tools"},
		"/alice/source/tools/image": {FullCodePath: "/alice/source/tools/image", Name: "Image", Code: "image"},
	}

	targets, err := planCopyDirectoryTargets("/alice/source/tools", "/bob/target", sourceTrees)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}
	if targets[0].targetPath != "/bob/target/tools" {
		t.Fatalf("expected root target first, got %s", targets[0].targetPath)
	}
	if targets[1].targetPath != "/bob/target/tools/image" {
		t.Fatalf("unexpected child target path: %s", targets[1].targetPath)
	}
}

func TestBuildCopyFileItems_SkipsInitAndPreservesRelativeTargets(t *testing.T) {
	files := map[string][]*model.FileSnapshot{
		"/alice/source/tools": {
			{FileName: "main", RelativePath: "main.go", FileType: "go", Content: "package tools"},
			{FileName: "init_", RelativePath: "init_.go", FileType: "go", Content: "package tools"},
		},
		"/alice/source/tools/image": {
			{RelativePath: "resize.go", FileType: "go", Content: "package image"},
		},
	}

	items := buildCopyFileItems("/alice/source/tools", "/bob/target/tools", files)
	if len(items) != 2 {
		t.Fatalf("expected 2 copied file items, got %d", len(items))
	}

	byName := map[string]*dto.FileWriteItem{}
	for _, item := range items {
		byName[item.FullCodePath+"/"+item.FileName] = item
	}
	if byName["/bob/target/tools/main"] == nil {
		t.Fatalf("expected root main file item, got %#v", byName)
	}
	if byName["/bob/target/tools/image/resize"] == nil {
		t.Fatalf("expected child resize file item, got %#v", byName)
	}
}
