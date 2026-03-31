package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ai-agent-os/hub/backend/dto"
)

var nonFileNameCharsRegexp = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func (s *HubDirectoryService) ExportDirectoryBundle(ctx context.Context, hubDirectoryID int64, fullCodePath, version, host string) (*dto.HubDirectoryInstallBundle, error) {
	detail, err := s.GetDirectoryDetail(ctx, hubDirectoryID, fullCodePath, version, true, host)
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.DirectoryTree == nil {
		return nil, fmt.Errorf("当前版本没有可导出的目录树数据")
	}
	if err := validateDirectoryTreeForPersistence(detail.DirectoryTree, detail.FullCodePath); err != nil {
		return nil, fmt.Errorf("目录树校验失败: %w", err)
	}

	return buildHubDirectoryInstallBundle(detail, time.Now().UTC()), nil
}

func buildHubDirectoryInstallBundle(detail *dto.HubDirectoryDetailDTO, exportedAt time.Time) *dto.HubDirectoryInstallBundle {
	if detail == nil {
		return nil
	}

	return &dto.HubDirectoryInstallBundle{
		SchemaVersion:    dto.HubDirectoryBundleSchemaVersion,
		BundleType:       dto.HubDirectoryBundleType,
		ExportedAt:       exportedAt.Format(time.RFC3339),
		HubDirectoryName: detail.Name,
		HubFullCodePath:  detail.FullCodePath,
		HubVersionNum:    detail.VersionNum,
		DirectoryTree:    detail.DirectoryTree,
	}
}

func BuildHubDirectoryBundleDownloadFileName(fullCodePath, version string) string {
	rawName := strings.TrimSpace(fullCodePath)
	if rawName == "" {
		rawName = "hub-directory"
	}

	safeName := strings.TrimLeft(strings.ReplaceAll(rawName, "/", "_"), "_")
	safeName = nonFileNameCharsRegexp.ReplaceAllString(safeName, "_")
	if safeName == "" {
		safeName = "hub-directory"
	}

	versionLabel := strings.TrimSpace(version)
	if versionLabel == "" {
		versionLabel = "latest"
	}
	versionLabel = nonFileNameCharsRegexp.ReplaceAllString(versionLabel, "_")
	if versionLabel == "" {
		versionLabel = "latest"
	}

	return fmt.Sprintf("hub-bundle-%s-%s.json", safeName, versionLabel)
}
