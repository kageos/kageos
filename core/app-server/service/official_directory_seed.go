package service

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

const officialDirectorySeedRelDir = "core/app-server/official-seed"

func initOfficialDirectorySeeds(ctx context.Context, serviceTreeService *ServiceTreeService) error {
	if serviceTreeService == nil {
		return nil
	}

	seedDir := officialDirectorySeedDir()
	info, err := os.Stat(seedDir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Infof(ctx, "[SystemWorkspace] 官方目录种子目录不存在，跳过: %s", seedDir)
			return nil
		}
		return fmt.Errorf("读取官方目录种子目录失败: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("官方目录种子路径不是目录: %s", seedDir)
	}

	files, err := listOfficialDirectorySeedFiles(seedDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		logger.Infof(ctx, "[SystemWorkspace] 官方目录种子目录为空，跳过: %s", seedDir)
		return nil
	}

	for _, filePath := range files {
		targetPath, err := officialDirectorySeedTargetPath(seedDir, filePath)
		if err != nil {
			return err
		}
		resp, err := serviceTreeService.InstallCapabilityBundleFromFile(ctx, &dto.InstallCapabilityOptions{
			TargetDirectoryPath: targetPath,
			Overwrite:           true,
			ForceDiff:           true,
		}, filePath)
		if err != nil {
			return fmt.Errorf("导入官方目录种子失败: file=%s target=%s: %w", filePath, targetPath, err)
		}
		logger.Infof(ctx, "[SystemWorkspace] 官方目录种子已处理: file=%s target=%s result=%s", filePath, targetPath, resp.Message)
	}
	return nil
}

func officialDirectorySeedDir() string {
	if root := config.GetAgentOSRoot(); root != "" {
		return filepath.Join(root, officialDirectorySeedRelDir)
	}
	return officialDirectorySeedRelDir
}

func listOfficialDirectorySeedFiles(seedDir string) ([]string, error) {
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
		return nil, fmt.Errorf("扫描官方目录种子失败: %w", err)
	}
	sort.Strings(files)
	return files, nil
}

func officialDirectorySeedTargetPath(seedDir, filePath string) (string, error) {
	rel, err := filepath.Rel(seedDir, filePath)
	if err != nil {
		return "", err
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("官方能力种子文件必须放在 system/{app}/... 下: %s", filePath)
	}
	if strings.TrimSpace(parts[0]) != "system" {
		return "", fmt.Errorf("官方能力种子文件必须放在 system 工作空间下: %s", filePath)
	}
	systemApp := strings.TrimSpace(parts[1])
	if !knownSystemAppCode(systemApp) {
		return "", fmt.Errorf("官方目录种子文件使用了未知系统目录 %q: %s", systemApp, filePath)
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
