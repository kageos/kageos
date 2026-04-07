package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VersionManager 管理应用版本信息
type VersionManager struct {
	UserDir     string // 用户目录，如 /path/to/namespace/user1
	AppName     string // 应用名称，如 app1
	ReleasesDir string // releases目录路径
}

// NewVersionManager 创建版本管理器
func NewVersionManager(userDir, appName string) *VersionManager {
	releasesDir := filepath.Join(userDir, appName, "workplace", "bin", "releases")

	return &VersionManager{
		UserDir:     userDir,
		AppName:     appName,
		ReleasesDir: releasesDir,
	}
}

// getAppPrefix 获取应用前缀
func (vm *VersionManager) getAppPrefix() string {
	// 从用户目录路径中提取用户名
	userName := filepath.Base(vm.UserDir)
	return fmt.Sprintf("%s_%s_", userName, vm.AppName)
}

// GetCurrentVersion 获取当前版本（从 current_version.txt 读取）
func (vm *VersionManager) GetCurrentVersion() (string, error) {
	versionFile := filepath.Join(vm.UserDir, vm.AppName, "workplace", "metadata", "current_version.txt")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("current_version.txt not found: %w", err)
		}
		return "", fmt.Errorf("failed to read current_version.txt: %w", err)
	}

	currentVersion := strings.TrimSpace(string(data))
	if currentVersion == "" {
		return "", fmt.Errorf("current_version.txt is empty")
	}

	return currentVersion, nil
}

// ListVersions 列出所有可用版本
func (vm *VersionManager) ListVersions() ([]string, error) {
	entries, err := os.ReadDir(vm.ReleasesDir)
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), vm.getAppPrefix()) {
			versions = append(versions, entry.Name())
		}
	}

	return versions, nil
}
