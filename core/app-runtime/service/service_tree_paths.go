package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/naming"
)

func validateBatchWritePathSegment(segment string) error {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return fmt.Errorf("路径片段不能为空")
	}
	if segment == "." || segment == ".." {
		return fmt.Errorf("路径片段不能为 . 或 ..: %s", segment)
	}
	if strings.ContainsAny(segment, `/\`) {
		return fmt.Errorf("路径片段不能包含路径分隔符: %s", segment)
	}
	if strings.HasPrefix(segment, ".") {
		return fmt.Errorf("路径片段不能以 . 开头: %s", segment)
	}
	return nil
}

func validateGoPackagePathSegment(segment string) error {
	if segment != strings.TrimSpace(segment) {
		return fmt.Errorf("路径片段不能包含首尾空格: %s", segment)
	}
	if err := validateBatchWritePathSegment(segment); err != nil {
		return err
	}
	if err := naming.ValidateGoPackageName(segment, "目录英文标识"); err != nil {
		return err
	}
	return nil
}

func validateBatchWritePackagePath(user, app, fullCodePath string) (string, error) {
	if fullCodePath != strings.TrimSpace(fullCodePath) {
		return "", fmt.Errorf("full_code_path 不能包含首尾空格: %s", fullCodePath)
	}
	trimmed := strings.Trim(fullCodePath, "/")
	if trimmed == "" {
		return "", fmt.Errorf("full_code_path 不能为空")
	}

	parts := strings.Split(trimmed, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("full_code_path 必须至少包含 user/app/directory")
	}
	if parts[0] != user || parts[1] != app {
		return "", fmt.Errorf("full_code_path 与目标应用不匹配: %s", fullCodePath)
	}

	packageParts := parts[2:]
	for _, part := range packageParts {
		if err := validateGoPackagePathSegment(part); err != nil {
			return "", fmt.Errorf("非法目录路径 %s: %w", fullCodePath, err)
		}
	}

	return strings.Join(packageParts, "/"), nil
}

func validateBatchWriteFileName(fileName string) error {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return fmt.Errorf("file_name 不能为空")
	}
	if fileName == "." || fileName == ".." {
		return fmt.Errorf("file_name 不能为 . 或 ..")
	}
	if strings.ContainsAny(fileName, `/\`) {
		return fmt.Errorf("file_name 不能包含路径分隔符: %s", fileName)
	}
	if strings.HasPrefix(fileName, ".") {
		return fmt.Errorf("file_name 不能以 . 开头: %s", fileName)
	}
	if fileName == "init_" {
		return fmt.Errorf("不允许修改 init_.go，该文件由目录创建流程自动维护")
	}
	return nil
}

func validateBatchWriteFileExt(fileType string) (string, error) {
	ext := getFileExtension(strings.TrimSpace(fileType))
	trimmed := strings.TrimPrefix(ext, ".")
	if trimmed == "" {
		return "", fmt.Errorf("file_type 不能为空")
	}
	if trimmed == "." || trimmed == ".." || strings.Contains(trimmed, "..") {
		return "", fmt.Errorf("非法 file_type: %s", fileType)
	}
	if strings.ContainsAny(trimmed, `/\`) {
		return "", fmt.Errorf("file_type 不能包含路径分隔符: %s", fileType)
	}
	if strings.TrimSpace(trimmed) != trimmed {
		return "", fmt.Errorf("file_type 不能包含首尾空格: %s", fileType)
	}
	return ext, nil
}

func ensurePathWithinBase(baseDir, targetPath string) error {
	rel, err := filepath.Rel(baseDir, targetPath)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("目标路径越界: %s", targetPath)
	}
	return nil
}

func resolveBatchWriteTarget(user, app, apiDir string, item *dto.FileWriteItem) (string, string, string, error) {
	if item == nil {
		return "", "", "", fmt.Errorf("文件项不能为空")
	}

	packagePath, err := validateBatchWritePackagePath(user, app, item.FullCodePath)
	if err != nil {
		return "", "", "", err
	}

	rawFileName := strings.TrimSpace(item.FileName)
	if rawFileName == "" {
		pathParts := strings.Split(strings.Trim(item.FullCodePath, "/"), "/")
		rawFileName = pathParts[len(pathParts)-1]
	}

	fileName := rawFileName
	fileExtFromName := filepath.Ext(rawFileName)
	if fileExtFromName != "" {
		fileName = strings.TrimSuffix(rawFileName, fileExtFromName)
	}
	if err := validateBatchWriteFileName(fileName); err != nil {
		return "", "", "", err
	}

	resolvedExt, err := validateBatchWriteFileExt(item.FileType)
	if err != nil {
		return "", "", "", err
	}
	if fileExtFromName != "" {
		if item.FileType != "" && resolvedExt != fileExtFromName {
			return "", "", "", fmt.Errorf("file_name 和 file_type 扩展名不一致: %s vs %s", fileExtFromName, resolvedExt)
		}
		resolvedExt = fileExtFromName
	}

	packageDir := filepath.Join(apiDir, packagePath)
	filePath := filepath.Join(packageDir, fileName+resolvedExt)
	if err := ensurePathWithinBase(apiDir, filePath); err != nil {
		return "", "", "", err
	}

	return packageDir, filePath, fileName, nil
}

func validateRelativePackagePath(packagePath string) (string, error) {
	if packagePath != strings.TrimSpace(packagePath) {
		return "", fmt.Errorf("package_path 不能包含首尾空格: %s", packagePath)
	}
	cleanPath := strings.Trim(packagePath, "/")
	if cleanPath == "" {
		return "", fmt.Errorf("package_path 不能为空")
	}

	parts := strings.Split(cleanPath, "/")
	for _, part := range parts {
		if err := validateGoPackagePathSegment(part); err != nil {
			return "", fmt.Errorf("非法 package_path %s: %w", packagePath, err)
		}
	}

	return strings.Join(parts, "/"), nil
}

func resolveDirectoryTarget(user, app, apiDir string, item *dto.DirectoryScaffoldItem) (string, string, error) {
	if item == nil {
		return "", "", fmt.Errorf("目录项不能为空")
	}

	packagePath, err := validateBatchWritePackagePath(user, app, item.FullCodePath)
	if err != nil {
		return "", "", err
	}

	packageDir := filepath.Join(apiDir, packagePath)
	if err := ensurePathWithinBase(apiDir, packageDir); err != nil {
		return "", "", err
	}

	return packagePath, packageDir, nil
}

func packageCodeFromPath(packagePath string) string {
	parts := strings.Split(strings.Trim(packagePath, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func writePackageInitFile(packageDir, packageCode, routerGroup, name, description string) error {
	content := fmt.Sprintf(`package %s

import (
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: %s,
	Name:        %s,
	Desc:        %s,
}
`, packageCode, strconv.Quote(routerGroup), strconv.Quote(name), strconv.Quote(description))

	initFilePath := filepath.Join(packageDir, "init_.go")
	if err := writeFileAtomic(initFilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入 init_.go 文件失败: %w", err)
	}

	return nil
}

func getFileExtension(fileType string) string {
	if fileType == "" {
		return ".go"
	}
	if strings.HasPrefix(fileType, ".") {
		return fileType
	}
	return "." + fileType
}

func sortDirectoryItemsByPath(items []*dto.DirectoryScaffoldItem) []*dto.DirectoryScaffoldItem {
	sorted := make([]*dto.DirectoryScaffoldItem, len(items))
	copy(sorted, items)

	sort.Slice(sorted, func(i, j int) bool {
		lenI := len(sorted[i].FullCodePath)
		lenJ := len(sorted[j].FullCodePath)
		if lenI != lenJ {
			return lenI < lenJ
		}
		return sorted[i].FullCodePath < sorted[j].FullCodePath
	})

	return sorted
}
