package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/servicetree"
)

func TestBuildRecursiveTreeTreatsDocsAsLeaf(t *testing.T) {
	workspaceCtx := &dto.GetWorkspaceContextResp{
		Directory: dto.WorkspaceContextDirectory{
			Name:         "邮箱物流",
			FullCodePath: "/system/democase/qq_mail/docs",
			Type:         servicetree.TypePackage,
		},
		Children: []dto.WorkspaceContextNode{
			{
				Name:         "海运物流方案",
				Code:         "headhaul_logistics.docs",
				Type:         servicetree.TypeDocs,
				FullCodePath: "/system/democase/qq_mail/docs/headhaul_logistics.docs",
			},
		},
	}

	loadCalls := 0
	loader := func(context.Context, string, string) (*dto.GetWorkspaceContextResp, error) {
		loadCalls++
		return nil, nil
	}

	result, hasErr := buildRecursiveTreeWithLoader(context.Background(), workspaceCtx, workspaceCtx.Directory.FullCodePath, 0, -1, true, true, "runtime", "tree", loader)
	if hasErr {
		t.Fatal("buildRecursiveTreeWithLoader() returned an unexpected error")
	}
	if loadCalls != 0 {
		t.Fatalf("docs leaf triggered %d workspace context loads, want 0", loadCalls)
	}
	if !strings.Contains(result, "headhaul_logistics.docs") || !strings.Contains(result, "[docs]") {
		t.Fatalf("tree output does not contain docs leaf: %s", result)
	}
}

func TestBuildRecursiveTreeStopsWhenDirectoryResolvesToVisitedPath(t *testing.T) {
	rootPath := "/system/democase/qq_mail"
	workspaceCtx := &dto.GetWorkspaceContextResp{
		Directory: dto.WorkspaceContextDirectory{
			Name:         "QQ 邮箱",
			FullCodePath: rootPath,
			Type:         servicetree.TypePackage,
		},
		Children: []dto.WorkspaceContextNode{
			{
				Name:         "文档目录",
				Code:         "docs",
				Type:         servicetree.TypePackage,
				FullCodePath: rootPath + "/docs",
			},
		},
	}

	loadCalls := 0
	loader := func(context.Context, string, string) (*dto.GetWorkspaceContextResp, error) {
		loadCalls++
		return &dto.GetWorkspaceContextResp{
			Directory: dto.WorkspaceContextDirectory{
				Name:         "QQ 邮箱",
				FullCodePath: rootPath,
				Type:         servicetree.TypePackage,
			},
		}, nil
	}

	result, hasErr := buildRecursiveTreeWithLoader(context.Background(), workspaceCtx, rootPath, 0, -1, true, true, "runtime", "tree", loader)
	if hasErr {
		t.Fatal("buildRecursiveTreeWithLoader() returned an unexpected error")
	}
	if loadCalls != 1 {
		t.Fatalf("cycle triggered %d workspace context loads, want 1", loadCalls)
	}
	if !strings.Contains(result, "目录被解析到已访问路径，已停止展开") {
		t.Fatalf("tree output does not report the stopped cycle: %s", result)
	}
}

func TestBuildReadDirResultDataSeparatesDocumentsFromDirectories(t *testing.T) {
	workspaceCtx := &dto.GetWorkspaceContextResp{
		Directory: dto.WorkspaceContextDirectory{FullCodePath: "/system/democase/qq_mail"},
		Children: []dto.WorkspaceContextNode{
			{Code: "docs", Type: servicetree.TypePackage, FullCodePath: "/system/democase/qq_mail/docs"},
			{Code: "readme.docs", Type: servicetree.TypeDocs, FullCodePath: "/system/democase/qq_mail/readme.docs"},
		},
	}

	data := buildReadDirResultData(workspaceCtx.Directory.FullCodePath, workspaceCtx.Directory.FullCodePath, false, "tree", true, -1, true, true, false, workspaceCtx)
	if len(data.Directories) != 1 || data.Summary.DirectoryCount != 1 {
		t.Fatalf("directories = %d, summary = %d, want 1", len(data.Directories), data.Summary.DirectoryCount)
	}
	if len(data.Documents) != 1 || data.Summary.DocumentCount != 1 {
		t.Fatalf("documents = %d, summary = %d, want 1", len(data.Documents), data.Summary.DocumentCount)
	}
}
