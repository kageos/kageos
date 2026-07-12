package service

import (
	"fmt"
	"path"
	"strings"

	"github.com/kageos/kageos/pkg/naming"
)

func validateCapabilityPackagePath(packagePath string, field string, allowEmpty bool) (string, error) {
	if packagePath != strings.TrimSpace(packagePath) {
		return "", fmt.Errorf("%s 不能包含首尾空格: %s", field, packagePath)
	}
	if strings.HasPrefix(packagePath, "/") || strings.HasSuffix(packagePath, "/") {
		return "", fmt.Errorf("%s 必须是相对 package 路径: %s", field, packagePath)
	}
	packagePath = strings.Trim(packagePath, "/")
	if packagePath == "" {
		if allowEmpty {
			return "", nil
		}
		return "", fmt.Errorf("%s 不能为空", field)
	}
	if strings.Contains(packagePath, "\\") || path.IsAbs(packagePath) {
		return "", fmt.Errorf("%s 必须是相对 package 路径: %s", field, packagePath)
	}
	if cleaned := path.Clean(packagePath); cleaned != packagePath {
		return "", fmt.Errorf("%s 必须使用规范相对路径: %s", field, packagePath)
	}
	parts := strings.Split(packagePath, "/")
	if err := rejectWorkspaceBoundCapabilityPath(parts, field, packagePath); err != nil {
		return "", err
	}
	for _, part := range parts {
		if err := naming.ValidateGoPackageName(part, "目录英文标识"); err != nil {
			return "", fmt.Errorf("%s 包含不支持的目录英文标识 %q: %w", field, part, err)
		}
	}
	return packagePath, nil
}

func validateCapabilityFilePath(filePath string, field string) (string, error) {
	if filePath != strings.TrimSpace(filePath) {
		return "", fmt.Errorf("%s 不能包含首尾空格: %s", field, filePath)
	}
	if filePath == "" {
		return "", fmt.Errorf("%s 不能为空", field)
	}
	if strings.ContainsAny(filePath, `/\`) || path.IsAbs(filePath) {
		return "", fmt.Errorf("%s 必须是目录内直接文件名: %s", field, filePath)
	}
	if cleaned := path.Clean(filePath); cleaned != filePath || cleaned == "." {
		return "", fmt.Errorf("%s 必须使用规范相对文件名: %s", field, filePath)
	}

	parts := strings.Split(filePath, "/")
	if err := rejectWorkspaceBoundCapabilityPath(parts, field, filePath); err != nil {
		return "", err
	}
	base := filePath
	if base == "." || base == ".." || strings.HasPrefix(base, ".") {
		return "", fmt.Errorf("%s 文件名非法: %s", field, filePath)
	}
	if base == "init_.go" {
		return "", fmt.Errorf("%s 不允许包含 init_.go，该文件由目标应用目录脚手架生成", field)
	}
	if isInternalWorkspaceManifestFile(base, "") {
		return "", fmt.Errorf("%s 不允许包含 %s，该文件仅用于本地目录种子声明", field, base)
	}
	if path.Ext(base) == "" {
		return "", fmt.Errorf("%s 必须包含文件扩展名: %s", field, filePath)
	}
	return filePath, nil
}

func validateCapabilityTargetDirectoryPath(targetPath string) (string, error) {
	if targetPath != strings.TrimSpace(targetPath) {
		return "", fmt.Errorf("target_directory_path 不能包含首尾空格: %s", targetPath)
	}
	if targetPath == "" {
		return "", fmt.Errorf("target_directory_path 不能为空")
	}
	if strings.Contains(targetPath, "\\") || !strings.HasPrefix(targetPath, "/") {
		return "", fmt.Errorf("target_directory_path 必须是目标节点完整路径: %s", targetPath)
	}
	if cleaned := path.Clean(targetPath); cleaned != targetPath || cleaned == "." || cleaned == "/" {
		return "", fmt.Errorf("target_directory_path 必须使用规范完整路径: %s", targetPath)
	}
	parts := strings.Split(strings.Trim(targetPath, "/"), "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("target_directory_path 必须至少包含 workspace/app: %s", targetPath)
	}
	for i, part := range parts {
		if part == "" || part == "." || part == ".." || strings.HasPrefix(part, ".") {
			return "", fmt.Errorf("target_directory_path 包含非法路径片段: %s", targetPath)
		}
		if i >= 2 {
			if err := naming.ValidateGoPackageName(part, "目标目录英文标识"); err != nil {
				return "", fmt.Errorf("target_directory_path 包含不支持的目录英文标识 %q: %w", part, err)
			}
		}
	}
	if err := rejectWorkspaceBoundCapabilityPath(parts, "target_directory_path", targetPath); err != nil {
		return "", err
	}
	return "/" + strings.Join(parts, "/"), nil
}

func rejectWorkspaceBoundCapabilityPath(parts []string, field string, original string) error {
	if len(parts) == 0 {
		return nil
	}
	if parts[0] == "namespace" {
		return fmt.Errorf("%s 不能包含 namespace 工作空间路径: %s", field, original)
	}
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "code" && parts[i+1] == "api" {
			return fmt.Errorf("%s 不能包含 code/api 工作空间路径: %s", field, original)
		}
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("%s 包含非法路径片段: %s", field, original)
		}
	}
	return nil
}

func addCapabilityPackageAncestors(paths map[string]struct{}, packagePath string) {
	parts := strings.Split(strings.Trim(packagePath, "/"), "/")
	for i := range parts {
		ancestor := strings.Join(parts[:i+1], "/")
		if ancestor != "" {
			paths[ancestor] = struct{}{}
		}
	}
}

func joinCapabilityFullCodePath(targetRootPath, relativePackagePath string) string {
	targetRootPath = "/" + strings.Trim(targetRootPath, "/")
	relativePackagePath = strings.Trim(relativePackagePath, "/")
	if relativePackagePath == "" {
		return targetRootPath
	}
	return strings.TrimRight(targetRootPath, "/") + "/" + relativePackagePath
}
