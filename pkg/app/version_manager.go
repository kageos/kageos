package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VersionManager 管理应用版本和软链接
type VersionManager struct {
	UserDir     string // 用户目录，如 /path/to/namespace/user1
	AppName     string // 应用名称，如 app1
	BinDir      string // bin目录路径
	ReleasesDir string // releases目录路径
}

// NewVersionManager 创建版本管理器
func NewVersionManager(userDir, appName string) *VersionManager {
	binDir := filepath.Join(userDir, appName, "workplace", "bin")
	releasesDir := filepath.Join(binDir, "releases")

	return &VersionManager{
		UserDir:     userDir,
		AppName:     appName,
		BinDir:      binDir,
		ReleasesDir: releasesDir,
	}
}

// UpdateAppSymlink 更新 app 软链接指向最新版本
func (vm *VersionManager) UpdateAppSymlink() error {
	// 获取最新版本
	latestVersion, err := vm.getLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to get latest version: %w", err)
	}

	if latestVersion == "" {
		return fmt.Errorf("no version found")
	}

	// 创建软链接
	appPath := filepath.Join(vm.BinDir, "app")

	// 删除旧的软链接（如果存在）
	if _, err := os.Lstat(appPath); err == nil {
		os.Remove(appPath)
	}

	// 创建新的软链接，指向 releases 目录下的文件
	releasesPath := filepath.Join("releases", latestVersion)
	err = os.Symlink(releasesPath, appPath)
	if err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// getLatestVersion 获取最新版本号（从 version.json 文件读取）
func (vm *VersionManager) getLatestVersion() (string, error) {
	// 从 version.json 文件读取最新版本
	versionFile := filepath.Join(vm.UserDir, vm.AppName, "workplace/metadata/version.json")

	// 检查文件是否存在
	if _, err := os.Stat(versionFile); os.IsNotExist(err) {
		return "", fmt.Errorf("version.json not found: %w", err)
	}

	// 读取文件内容
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "", fmt.Errorf("failed to read version.json: %w", err)
	}

	// 解析 JSON
	var versionData struct {
		LatestVersion string `json:"latest_version"`
	}
	if err := json.Unmarshal(data, &versionData); err != nil {
		return "", fmt.Errorf("failed to parse version.json: %w", err)
	}

	return versionData.LatestVersion, nil
}

// getAppPrefix 获取应用前缀
func (vm *VersionManager) getAppPrefix() string {
	// 从用户目录路径中提取用户名
	userName := filepath.Base(vm.UserDir)
	return fmt.Sprintf("%s_%s_", userName, vm.AppName)
}

// GetCurrentVersion 获取当前版本（从 version.json 文件读取）
func (vm *VersionManager) GetCurrentVersion() (string, error) {
	// 从 version.json 文件读取当前版本
	versionFile := filepath.Join(vm.UserDir, vm.AppName, "workplace/metadata/version.json")

	// 检查文件是否存在
	if _, err := os.Stat(versionFile); os.IsNotExist(err) {
		return "", fmt.Errorf("version.json not found: %w", err)
	}

	// 读取文件内容
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "", fmt.Errorf("failed to read version.json: %w", err)
	}

	// 解析 JSON
	var versionData struct {
		CurrentVersion string `json:"current_version"`
	}
	if err := json.Unmarshal(data, &versionData); err != nil {
		return "", fmt.Errorf("failed to parse version.json: %w", err)
	}

	return versionData.CurrentVersion, nil
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
