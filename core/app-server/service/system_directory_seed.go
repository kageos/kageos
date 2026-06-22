package service

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
)

const systemDirectorySeedRelDir = "core/app-server/system-seed"

type systemDirectorySeedFile struct {
	filePath   string
	targetPath string
	appCode    string
}

func initSystemDirectorySeeds(ctx context.Context, serviceTreeService *ServiceTreeService, createdApps map[string]bool) error {
	if serviceTreeService == nil {
		return nil
	}

	seedDir := systemDirectorySeedDir()
	info, err := os.Stat(seedDir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Infof(ctx, "[SystemWorkspace] 系统目录种子目录不存在，跳过: %s", seedDir)
			return nil
		}
		return fmt.Errorf("读取系统目录种子目录失败: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("系统目录种子路径不是目录: %s", seedDir)
	}

	files, err := listSystemDirectorySeedFiles(seedDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		logger.Infof(ctx, "[SystemWorkspace] 系统目录种子目录为空，跳过: %s", seedDir)
		return nil
	}

	seedFiles, err := resolveSystemDirectorySeedFiles(seedDir, files)
	if err != nil {
		return err
	}
	initialVersions, err := initialSystemDirectorySeedAppVersions(serviceTreeService, seedFiles)
	if err != nil {
		return err
	}

	for _, seedFile := range seedFiles {
		initialVersion := strings.TrimSpace(initialVersions[seedFile.appCode])
		shouldInstall, err := systemDirectorySeedShouldInstall(serviceTreeService, seedFile, initialVersion, createdApps[seedFile.appCode])
		if err != nil {
			return err
		}
		if !shouldInstall {
			logger.Infof(ctx, "[SystemWorkspace] 系统目录种子已在首次部署完成，跳过写文件和编译: app=%s/%s version=%s file=%s target=%s",
				SystemUsername, seedFile.appCode, initialVersion, seedFile.filePath, seedFile.targetPath)
			continue
		}
		resp, err := serviceTreeService.InstallCapabilityBundleFromFile(ctx, &dto.InstallCapabilityOptions{
			TargetDirectoryPath: seedFile.targetPath,
			Overwrite:           true,
			ForceDiff:           true,
		}, seedFile.filePath)
		if err != nil {
			return fmt.Errorf("导入系统目录种子失败: file=%s target=%s: %w", seedFile.filePath, seedFile.targetPath, err)
		}
		logger.Infof(ctx, "[SystemWorkspace] 系统目录种子已处理: file=%s target=%s result=%s", seedFile.filePath, seedFile.targetPath, resp.Message)
	}
	return nil
}

func systemDirectorySeedDir() string {
	if root := config.GetKageosRoot(); root != "" {
		return filepath.Join(root, systemDirectorySeedRelDir)
	}
	return systemDirectorySeedRelDir
}

func listSystemDirectorySeedFiles(seedDir string) ([]string, error) {
	files := make([]string, 0)
	if err := filepath.WalkDir(seedDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("扫描系统目录种子失败: %w", err)
	}
	sort.Strings(files)
	return files, nil
}

func resolveSystemDirectorySeedFiles(seedDir string, files []string) ([]systemDirectorySeedFile, error) {
	seedFiles := make([]systemDirectorySeedFile, 0, len(files))
	for _, filePath := range files {
		targetPath, err := systemDirectorySeedTargetPath(seedDir, filePath)
		if err != nil {
			return nil, err
		}
		appCode, err := systemDirectorySeedAppCodeFromTargetPath(targetPath)
		if err != nil {
			return nil, err
		}
		seedFiles = append(seedFiles, systemDirectorySeedFile{
			filePath:   filePath,
			targetPath: targetPath,
			appCode:    appCode,
		})
	}
	return seedFiles, nil
}

func initialSystemDirectorySeedAppVersions(serviceTreeService *ServiceTreeService, seedFiles []systemDirectorySeedFile) (map[string]string, error) {
	versions := make(map[string]string)
	if len(seedFiles) == 0 {
		return versions, nil
	}
	if serviceTreeService == nil || serviceTreeService.capabilityBundle == nil || serviceTreeService.capabilityBundle.appRepo == nil {
		return nil, fmt.Errorf("系统目录种子无法读取应用版本，serviceTreeService 未完整初始化")
	}
	for _, seedFile := range seedFiles {
		if _, exists := versions[seedFile.appCode]; exists {
			continue
		}
		appModel, err := serviceTreeService.capabilityBundle.appRepo.GetAppByUserName(SystemUsername, seedFile.appCode)
		if err != nil {
			return nil, fmt.Errorf("查询系统应用 %s/%s 版本失败: %w", SystemUsername, seedFile.appCode, err)
		}
		versions[seedFile.appCode] = strings.TrimSpace(appModel.Version)
	}
	return versions, nil
}

func systemDirectorySeedAppCodeFromTargetPath(targetPath string) (string, error) {
	parts := strings.Split(strings.Trim(targetPath, "/"), "/")
	if len(parts) < 2 || parts[0] != SystemUsername {
		return "", fmt.Errorf("系统目录种子目标路径必须位于 /%s/{app}: %s", SystemUsername, targetPath)
	}
	appCode := strings.TrimSpace(parts[1])
	if !knownSystemAppCode(appCode) {
		return "", fmt.Errorf("系统目录种子目标路径使用了未知系统目录 %q: %s", appCode, targetPath)
	}
	return appCode, nil
}

func systemDirectorySeedShouldInstall(serviceTreeService *ServiceTreeService, seedFile systemDirectorySeedFile, initialAppVersion string, appCreated bool) (bool, error) {
	if appCreated {
		return true, nil
	}
	version := strings.TrimSpace(initialAppVersion)
	if version == "" {
		return true, nil
	}
	if version != "v1" {
		return false, nil
	}
	complete, err := systemDirectorySeedTargetMatchesBundle(serviceTreeService, seedFile)
	if err != nil {
		return false, err
	}
	return !complete, nil
}

func systemDirectorySeedTargetMatchesBundle(serviceTreeService *ServiceTreeService, seedFile systemDirectorySeedFile) (bool, error) {
	if serviceTreeService == nil || serviceTreeService.capabilityBundle == nil || serviceTreeService.capabilityBundle.serviceTreeRepo == nil {
		return false, fmt.Errorf("系统目录种子无法检查目标目录，serviceTreeService 未完整初始化")
	}
	bundle, err := readCapabilityBundleFile(seedFile.filePath)
	if err != nil {
		return false, err
	}
	expectedPaths := systemDirectorySeedBundleTargetPaths(seedFile.targetPath, bundle)
	if len(expectedPaths) == 0 {
		return true, nil
	}
	existing, err := serviceTreeService.capabilityBundle.serviceTreeRepo.GetServiceTreeByFullPaths(expectedPaths)
	if err != nil {
		return false, fmt.Errorf("检查系统目录种子目标节点失败: %w", err)
	}
	for _, expectedPath := range expectedPaths {
		if _, ok := existing[expectedPath]; !ok {
			return false, nil
		}
	}
	return true, nil
}

func systemDirectorySeedBundleTargetPaths(targetPath string, bundle *dto.CapabilityBundle) []string {
	if bundle == nil {
		return nil
	}
	targetPrefix := strings.Trim(targetPath, "/")
	if targetPrefix == "" {
		return nil
	}
	seen := make(map[string]struct{}, len(bundle.TreeNodes))
	paths := make([]string, 0, len(bundle.TreeNodes))
	for _, node := range bundle.TreeNodes {
		if node == nil {
			continue
		}
		relativePath := strings.Trim(node.RelativePath, "/")
		if relativePath == "" {
			continue
		}
		fullPath := "/" + path.Join(targetPrefix, relativePath)
		if _, exists := seen[fullPath]; exists {
			continue
		}
		seen[fullPath] = struct{}{}
		paths = append(paths, fullPath)
	}
	sort.Strings(paths)
	return paths
}

func systemDirectorySeedTargetPath(seedDir, filePath string) (string, error) {
	rel, err := filepath.Rel(seedDir, filePath)
	if err != nil {
		return "", err
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("系统能力种子文件必须放在 system/{app}/... 下: %s", filePath)
	}
	if strings.TrimSpace(parts[0]) != "system" {
		return "", fmt.Errorf("系统能力种子文件必须放在 system 工作空间下: %s", filePath)
	}
	systemApp := strings.TrimSpace(parts[1])
	if !knownSystemAppCode(systemApp) {
		return "", fmt.Errorf("系统目录种子文件使用了未知系统目录 %q: %s", systemApp, filePath)
	}
	dirParts := parts[:len(parts)-1]
	return "/" + strings.Join(dirParts, "/"), nil
}

func knownSystemAppCode(code string) bool {
	for _, def := range systemAppDefinitions() {
		if def.Code == code {
			return true
		}
	}
	return false
}
