package service

import (
	"strings"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func buildCanonicalHubInstallBundle() *dto.HubDirectoryInstallBundle {
	return &dto.HubDirectoryInstallBundle{
		SchemaVersion:    dto.HubDirectoryBundleSchemaVersion,
		BundleType:       dto.HubDirectoryBundleType,
		ExportedAt:       "2026-03-31T08:00:00Z",
		HubDirectoryName: "投递简历",
		HubFullCodePath:  "/luobei/minimax/salon_hr/hr_resume_list",
		HubVersionNum:    12,
		DirectoryTree: &dto.DirectoryTreeNode{
			Type: "package",
			Name: "投递简历",
			Code: "hr_resume_list",
			Path: "/luobei/minimax/salon_hr/hr_resume_list",
			Subdirectories: []*dto.DirectoryTreeNode{
				{
					Type: "package",
					Name: "子目录",
					Code: "child",
					Path: "/luobei/minimax/salon_hr/hr_resume_list/child",
				},
			},
		},
	}
}

func TestValidateHubDirectoryInstallBundleForImport_AllowsCanonicalBundle(t *testing.T) {
	bundle := buildCanonicalHubInstallBundle()

	if err := validateHubDirectoryInstallBundleForImportImpl(bundle); err != nil {
		t.Fatalf("expected canonical bundle to pass validation, got %v", err)
	}
}

func TestValidateHubDirectoryInstallBundleForImport_RejectsInvalidBundleType(t *testing.T) {
	bundle := buildCanonicalHubInstallBundle()
	bundle.BundleType = "legacy_bundle"

	err := validateHubDirectoryInstallBundleForImportImpl(bundle)
	if err == nil {
		t.Fatal("expected bundle type validation error")
	}
	if !strings.Contains(err.Error(), "不支持的安装包类型") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateHubDirectoryInstallBundleForImport_RejectsMissingDirectoryTree(t *testing.T) {
	bundle := buildCanonicalHubInstallBundle()
	bundle.DirectoryTree = nil

	err := validateHubDirectoryInstallBundleForImportImpl(bundle)
	if err == nil {
		t.Fatal("expected missing directory_tree validation error")
	}
	if !strings.Contains(err.Error(), "安装包缺少 directory_tree") {
		t.Fatalf("unexpected error: %v", err)
	}
}
