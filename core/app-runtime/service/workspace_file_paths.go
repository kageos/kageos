package service

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func normalizeWorkspaceRelativePath(relativePath string) (string, error) {
	if relativePath != strings.TrimSpace(relativePath) {
		return "", fmt.Errorf("路径不能包含首尾空格: %s", relativePath)
	}
	trimmed := strings.Trim(relativePath, "/")
	if trimmed == "" {
		return "", nil
	}

	parts := strings.Split(trimmed, "/")
	for _, part := range parts {
		if err := validateGoPackagePathSegment(part); err != nil {
			return "", fmt.Errorf("非法路径 %s: %w", relativePath, err)
		}
	}

	return strings.Join(parts, "/"), nil
}

func resolveWorkspaceDirectoryPath(appPaths runtimeAppPaths, fullCodePath string) (string, error) {
	relativePath, err := normalizeWorkspaceRelativePath(appPaths.TrimAppPrefix(fullCodePath))
	if err != nil {
		return "", err
	}

	directoryPath := appPaths.APIDir()
	if relativePath != "" {
		directoryPath = filepath.Join(directoryPath, relativePath)
	}
	if err := ensurePathWithinBase(appPaths.APIDir(), directoryPath); err != nil {
		return "", err
	}

	return directoryPath, nil
}

func resolveWorkspaceFilePath(appPaths runtimeAppPaths, directoryPath, fileName string) (string, error) {
	dirPath, err := resolveWorkspaceDirectoryPath(appPaths, directoryPath)
	if err != nil {
		return "", err
	}

	baseName := strings.TrimSpace(fileName)
	fileExt := filepath.Ext(baseName)
	if fileExt != "" {
		baseName = strings.TrimSuffix(baseName, fileExt)
	}
	if err := validateBatchWriteFileName(baseName); err != nil {
		return "", err
	}

	ext := ".go"
	if fileExt != "" {
		ext = fileExt
	}
	filePath := filepath.Join(dirPath, baseName+ext)
	if err := ensurePathWithinBase(appPaths.APIDir(), filePath); err != nil {
		return "", err
	}

	return filePath, nil
}

func resolveSourceFileWriteTarget(appPaths runtimeAppPaths, spec *dto.SourceFileWrite) (string, string, error) {
	if spec == nil {
		return "", "", fmt.Errorf("source file spec 不能为空")
	}

	packagePath, err := normalizeWorkspaceRelativePath(spec.DirectoryPath)
	if err != nil {
		return "", "", err
	}

	fileName := strings.TrimSpace(spec.FileName)
	if ext := filepath.Ext(fileName); ext != "" {
		if ext != ".go" {
			return "", "", fmt.Errorf("source file 仅支持 .go 扩展名: %s", spec.FileName)
		}
		fileName = strings.TrimSuffix(fileName, ext)
	}
	if err := validateBatchWriteFileName(fileName); err != nil {
		return "", "", err
	}

	packageDir := appPaths.APIDir()
	if packagePath != "" {
		packageDir = filepath.Join(packageDir, packagePath)
	}
	if err := ensurePathWithinBase(appPaths.APIDir(), packageDir); err != nil {
		return "", "", err
	}

	filePath := filepath.Join(packageDir, fileName+".go")
	if err := ensurePathWithinBase(appPaths.APIDir(), filePath); err != nil {
		return "", "", err
	}

	return packageDir, filePath, nil
}
