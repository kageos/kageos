package service

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func validateHubDirectoryInstallBundleForImportImpl(bundle *dto.HubDirectoryInstallBundle) error {
	if bundle == nil {
		return fmt.Errorf("安装包不能为空")
	}
	if bundle.SchemaVersion != dto.HubDirectoryBundleSchemaVersion {
		return fmt.Errorf("不支持的安装包 schema_version: %d", bundle.SchemaVersion)
	}
	if bundle.BundleType != dto.HubDirectoryBundleType {
		return fmt.Errorf("不支持的安装包类型: %s", bundle.BundleType)
	}
	if bundle.DirectoryTree == nil {
		return fmt.Errorf("安装包缺少 directory_tree")
	}
	if err := validateHubDirectoryTreeForInstallImpl(bundle.DirectoryTree); err != nil {
		return fmt.Errorf("目录树校验失败: %w", err)
	}
	return nil
}
