package service

import (
	"testing"
	"time"

	"github.com/ai-agent-os/hub/backend/dto"
)

func TestBuildHubDirectoryInstallBundle_UsesCanonicalSchema(t *testing.T) {
	exportedAt := time.Date(2026, 3, 31, 8, 0, 0, 0, time.UTC)
	detail := &dto.HubDirectoryDetailDTO{
		HubDirectoryDTO: dto.HubDirectoryDTO{
			Name:         "投递简历",
			FullCodePath: "/luobei/minimax/salon_hr/hr_resume_list",
			VersionNum:   12,
		},
		DirectoryTree: &dto.DirectoryTreeNode{
			Type: "package",
			Name: "投递简历",
			Code: "hr_resume_list",
			Path: "/luobei/minimax/salon_hr/hr_resume_list",
		},
	}

	bundle := buildHubDirectoryInstallBundle(detail, exportedAt)
	if bundle == nil {
		t.Fatal("expected bundle")
	}
	if bundle.SchemaVersion != dto.HubDirectoryBundleSchemaVersion {
		t.Fatalf("unexpected schema version: %d", bundle.SchemaVersion)
	}
	if bundle.BundleType != dto.HubDirectoryBundleType {
		t.Fatalf("unexpected bundle type: %s", bundle.BundleType)
	}
	if bundle.ExportedAt != exportedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected exported_at: %s", bundle.ExportedAt)
	}
	if bundle.HubDirectoryName != detail.Name || bundle.HubFullCodePath != detail.FullCodePath || bundle.HubVersionNum != detail.VersionNum {
		t.Fatalf("unexpected bundle metadata: %#v", bundle)
	}
	if bundle.DirectoryTree != detail.DirectoryTree {
		t.Fatal("expected bundle to reuse detail directory tree")
	}
}

func TestBuildHubDirectoryBundleDownloadFileName_SanitizesPathAndVersion(t *testing.T) {
	filename := BuildHubDirectoryBundleDownloadFileName("/luobei/minimax/salon hr/hr_resume_list", "v12 beta")
	expected := "hub-bundle-luobei_minimax_salon_hr_hr_resume_list-v12_beta.json"
	if filename != expected {
		t.Fatalf("unexpected filename:\nwant: %s\ngot:  %s", expected, filename)
	}
}
