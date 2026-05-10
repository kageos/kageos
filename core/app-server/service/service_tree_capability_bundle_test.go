package service

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

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
			{Path: "message", Name: "消息能力"},
			{Path: "permission", Name: "权限能力"},
		},
		Files: []*dto.CapabilityBundleFile{
			{PackagePath: "message", Path: "send.go", Content: "package message"},
			{PackagePath: "permission", Path: "apply.go", Content: "package permission"},
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
		"/system/tools/openapi/message::消息能力",
		"/system/tools/openapi/permission::权限能力",
	}
	if !reflect.DeepEqual(gotDirs, wantDirs) {
		t.Fatalf("unexpected directories:\nwant=%#v\ngot=%#v", wantDirs, gotDirs)
	}

	gotFiles := make([]string, 0, len(plan.fileItems))
	for _, item := range plan.fileItems {
		gotFiles = append(gotFiles, item.FullCodePath+"::"+item.FileName+"."+item.FileType)
	}
	wantFiles := []string{
		"/system/tools/openapi/message::send.go",
		"/system/tools/openapi/permission::apply.go",
	}
	if !reflect.DeepEqual(gotFiles, wantFiles) {
		t.Fatalf("unexpected files:\nwant=%#v\ngot=%#v", wantFiles, gotFiles)
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
