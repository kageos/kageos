package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
)

func TestDownloadCapabilityBundleUsesInstallKeyHeader(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Install-Key"); got != "install-key-123" {
			t.Fatalf("X-Install-Key = %q, want install-key-123", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"schema_version":"capability.bundle.v1",
			"name":"remote bundle",
			"packages":[],
			"files":[]
		}`))
	}))
	defer server.Close()

	bundle, err := downloadCapabilityBundle(context.Background(), server.URL+"/bundle", "install-key-123")
	if err != nil {
		t.Fatalf("downloadCapabilityBundle() error = %v", err)
	}
	if bundle.Name != "remote bundle" || bundle.SchemaVersion != dto.CapabilityBundleSchemaVersion {
		t.Fatalf("unexpected bundle: %#v", bundle)
	}
}

func TestDownloadCapabilityBundleExtractsInstallKeyFromURL(t *testing.T) {
	t.Parallel()

	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if got := r.Header.Get("X-Install-Key"); got != "url-key-123" {
			t.Fatalf("X-Install-Key = %q, want url-key-123", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"schema_version":"capability.bundle.v1",
			"name":"url key bundle",
			"packages":[],
			"files":[]
		}`))
	}))
	defer server.Close()

	bundle, err := downloadCapabilityBundle(context.Background(), server.URL+"/api/v1/products/demo/bundle/url-key-123", "")
	if err != nil {
		t.Fatalf("downloadCapabilityBundle() error = %v", err)
	}
	if gotPath != "/api/v1/products/demo/bundle" {
		t.Fatalf("request path = %q, want /api/v1/products/demo/bundle", gotPath)
	}
	if bundle.Name != "url key bundle" {
		t.Fatalf("bundle name = %q, want url key bundle", bundle.Name)
	}
}

func TestDownloadCapabilityBundleRejectsUnsupportedScheme(t *testing.T) {
	t.Parallel()

	_, err := downloadCapabilityBundle(context.Background(), "file:///tmp/bundle.json", "")
	if err == nil || !strings.Contains(err.Error(), "http/https") {
		t.Fatalf("expected scheme rejection, got %v", err)
	}
}

func TestValidateCapabilityBundleRejectsWorkspaceBoundPaths(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		bundle *dto.CapabilityBundle
		want   string
	}{
		{
			name: "absolute file path",
			bundle: &dto.CapabilityBundle{
				SchemaVersion: dto.CapabilityBundleSchemaVersion,
				Files: []*dto.CapabilityBundleFile{
					{Path: "/system/openapi/message/send.go", Content: "package message"},
				},
			},
			want: "目录内直接文件名",
		},
		{
			name: "namespace package path",
			bundle: &dto.CapabilityBundle{
				SchemaVersion: dto.CapabilityBundleSchemaVersion,
				Packages: []*dto.CapabilityBundlePackage{
					{Path: "namespace/system/openapi/message"},
				},
				Files: []*dto.CapabilityBundleFile{
					{PackagePath: "namespace/system/openapi/message", Path: "send.go", Content: "package message"},
				},
			},
			want: "namespace",
		},
		{
			name: "code api package path",
			bundle: &dto.CapabilityBundle{
				SchemaVersion: dto.CapabilityBundleSchemaVersion,
				Packages: []*dto.CapabilityBundlePackage{
					{Path: "system/openapi/code/api/message"},
				},
				Files: []*dto.CapabilityBundleFile{
					{PackagePath: "system/openapi/code/api/message", Path: "send.go", Content: "package message"},
				},
			},
			want: "code/api",
		},
		{
			name: "parent traversal",
			bundle: &dto.CapabilityBundle{
				SchemaVersion: dto.CapabilityBundleSchemaVersion,
				Packages: []*dto.CapabilityBundlePackage{
					{Path: "../message"},
				},
				Files: []*dto.CapabilityBundleFile{
					{PackagePath: "../message", Path: "send.go", Content: "package message"},
				},
			},
			want: "非法路径片段",
		},
		{
			name: "generated init",
			bundle: &dto.CapabilityBundle{
				SchemaVersion: dto.CapabilityBundleSchemaVersion,
				Files: []*dto.CapabilityBundleFile{
					{Path: "init_.go", Content: "package message"},
				},
			},
			want: "init_.go",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateCapabilityBundle(tc.bundle)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unexpected error:\nwant contains %q\ngot %v", tc.want, err)
			}
		})
	}
}

func TestBuildCapabilityBundleInstallPlanMountsRelativePackages(t *testing.T) {
	t.Parallel()

	bundle := &dto.CapabilityBundle{
		SchemaVersion: dto.CapabilityBundleSchemaVersion,
		Name:          "开放能力",
		Packages: []*dto.CapabilityBundlePackage{
			{Path: "invoice", Name: "发票能力"},
			{Path: "report", Name: "报表能力"},
		},
		Files: []*dto.CapabilityBundleFile{
			{PackagePath: "invoice", Path: "create.go", Content: "package invoice"},
			{PackagePath: "report", Path: "query.go", Content: "package report"},
		},
	}
	if err := validateCapabilityBundle(bundle); err != nil {
		t.Fatalf("expected valid bundle: %v", err)
	}

	plan, err := buildCapabilityBundleInstallPlan("/system/tools/openapi", bundle)
	if err != nil {
		t.Fatal(err)
	}

	gotDirs := make([]string, 0, len(plan.directoryItems))
	for _, item := range plan.directoryItems {
		gotDirs = append(gotDirs, item.FullCodePath+"::"+item.Name)
	}
	wantDirs := []string{
		"/system/tools/openapi/invoice::发票能力",
		"/system/tools/openapi/report::报表能力",
	}
	if !reflect.DeepEqual(gotDirs, wantDirs) {
		t.Fatalf("unexpected directories:\nwant=%#v\ngot=%#v", wantDirs, gotDirs)
	}

	gotFiles := make([]string, 0, len(plan.fileItems))
	for _, item := range plan.fileItems {
		gotFiles = append(gotFiles, item.FullCodePath+"::"+item.FileName+"."+item.FileType)
	}
	wantFiles := []string{
		"/system/tools/openapi/invoice::create.go",
		"/system/tools/openapi/report::query.go",
	}
	if !reflect.DeepEqual(gotFiles, wantFiles) {
		t.Fatalf("unexpected files:\nwant=%#v\ngot=%#v", wantFiles, gotFiles)
	}
}

func TestBuildCapabilityBundleInstallPlanIncludesDocs(t *testing.T) {
	t.Parallel()

	bundle := &dto.CapabilityBundle{
		SchemaVersion: dto.CapabilityBundleSchemaVersion,
		Name:          "CRM",
		TreeNodes: []*dto.CapabilityBundleTreeNode{
			{RelativePath: "crm", Type: model.ServiceTreeTypePackage, Code: "crm", Name: "CRM"},
			{RelativePath: "crm/readme.docs", ParentPath: "crm", Type: model.ServiceTreeTypeDocs, Code: "readme.docs", Name: "使用说明", Tags: []string{"docs", "crm"}},
		},
		Docs: []*dto.CapabilityBundleDoc{
			{RelativePath: "crm/readme.docs", Name: "使用说明", Content: "# 使用说明\n", Format: "markdown"},
		},
		Packages: []*dto.CapabilityBundlePackage{
			{Path: "crm", Name: "CRM"},
		},
		Files: []*dto.CapabilityBundleFile{},
	}

	plan, err := buildCapabilityBundleInstallPlan("/alice/app/openapi", bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.directoryItems) != 1 || plan.directoryItems[0].FullCodePath != "/alice/app/openapi/crm" {
		t.Fatalf("unexpected directories: %#v", plan.directoryItems)
	}
	if len(plan.docItems) != 1 {
		t.Fatalf("unexpected doc count: %d", len(plan.docItems))
	}
	got := plan.docItems[0]
	if got.FullCodePath != "/alice/app/openapi/crm/readme.docs" || got.ParentFullCodePath != "/alice/app/openapi/crm" {
		t.Fatalf("unexpected doc path: %#v", got)
	}
	if got.Name != "使用说明" || got.Content != "# 使用说明\n" || got.Tags != "docs,crm" {
		t.Fatalf("unexpected doc metadata: %#v", got)
	}
}

func TestBuildCapabilityBundleInstallPlanSupportsRootPackageFiles(t *testing.T) {
	t.Parallel()

	bundle := &dto.CapabilityBundle{
		SchemaVersion: dto.CapabilityBundleSchemaVersion,
		Name:          "消息能力",
		Files: []*dto.CapabilityBundleFile{
			{Path: "send.go", Content: "package openapi"},
		},
	}

	plan, err := buildCapabilityBundleInstallPlan("/customer/crm/openapi", bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.directoryItems) != 0 {
		t.Fatalf("unexpected directories: %#v", plan.directoryItems)
	}
	if len(plan.fileItems) != 1 || plan.fileItems[0].FullCodePath != "/customer/crm/openapi" {
		t.Fatalf("unexpected files: %#v", plan.fileItems)
	}
}

func TestValidateCapabilityBundleTreeNodes(t *testing.T) {
	t.Parallel()

	bundle := &dto.CapabilityBundle{
		SchemaVersion: dto.CapabilityBundleSchemaVersion,
		Name:          "CRM",
		TreeNodes: []*dto.CapabilityBundleTreeNode{
			{RelativePath: "crm", Type: model.ServiceTreeTypePackage, Code: "crm", Name: "CRM"},
			{RelativePath: "crm/customers", ParentPath: "crm", Type: model.ServiceTreeTypePackage, Code: "customers", Name: "Customers"},
			{RelativePath: "crm/customers/list.table", ParentPath: "crm/customers", Type: model.ServiceTreeTypeFunction, Code: "list.table", Name: "Customer List", TemplateType: "table"},
		},
		Packages: []*dto.CapabilityBundlePackage{
			{Path: "crm", Name: "CRM"},
			{Path: "crm/customers", Name: "Customers"},
		},
		Files: []*dto.CapabilityBundleFile{
			{PackagePath: "crm/customers", Path: "list.go", Content: "package customers"},
		},
	}

	if err := validateCapabilityBundle(bundle); err != nil {
		t.Fatalf("expected valid bundle with nested tree nodes: %v", err)
	}
}

func TestValidateCapabilityBundleTreeNodesRejectsMissingParent(t *testing.T) {
	t.Parallel()

	bundle := &dto.CapabilityBundle{
		SchemaVersion: dto.CapabilityBundleSchemaVersion,
		TreeNodes: []*dto.CapabilityBundleTreeNode{
			{RelativePath: "crm/customers/list.table", ParentPath: "crm/customers", Type: model.ServiceTreeTypeFunction, Code: "list.table"},
		},
		Files: []*dto.CapabilityBundleFile{
			{Path: "list.go", Content: "package crm"},
		},
	}

	err := validateCapabilityBundle(bundle)
	if err == nil {
		t.Fatal("expected missing parent validation error")
	}
	if !strings.Contains(err.Error(), "parent_path 未在 tree_nodes 中声明") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCapabilityBundleDocsRejectsNonDocsNode(t *testing.T) {
	t.Parallel()

	bundle := &dto.CapabilityBundle{
		SchemaVersion: dto.CapabilityBundleSchemaVersion,
		TreeNodes: []*dto.CapabilityBundleTreeNode{
			{RelativePath: "crm", Type: model.ServiceTreeTypePackage, Code: "crm"},
		},
		Docs: []*dto.CapabilityBundleDoc{
			{RelativePath: "crm", Content: "# CRM\n"},
		},
	}

	err := validateCapabilityBundle(bundle)
	if err == nil {
		t.Fatal("expected docs validation error")
	}
	if !strings.Contains(err.Error(), "不是 docs") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCapabilityRelativeTreeNodePathSupportsNestedFunction(t *testing.T) {
	t.Parallel()

	root := &model.ServiceTree{Code: "crm", Type: model.ServiceTreeTypePackage, FullCodePath: "/alice/app/crm"}
	node := &model.ServiceTree{Code: "list.table", Type: model.ServiceTreeTypeFunction, FullCodePath: "/alice/app/crm/customers/list.table"}

	got, err := capabilityRelativeTreeNodePath(root, node, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "crm/customers/list.table" {
		t.Fatalf("relative path = %q", got)
	}
}

func TestCapabilityRelativePackagePathCanExcludeRootCode(t *testing.T) {
	t.Parallel()

	root := &model.ServiceTree{
		Code:         "openapi",
		FullCodePath: "/system/openapi",
	}
	nested := &model.ServiceTree{
		Code:         "message",
		FullCodePath: "/system/openapi/platform/message",
	}

	got, err := capabilityRelativePackagePath(root, nested, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "platform/message" {
		t.Fatalf("unexpected relative path: %s", got)
	}

	got, err = capabilityRelativePackagePath(root, nested, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "openapi/platform/message" {
		t.Fatalf("unexpected include-root relative path: %s", got)
	}
}
