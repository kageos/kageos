package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/naming"
)

type capabilityBundleInstallPlan struct {
	targetRootPath string
	directoryItems []*dto.DirectoryScaffoldItem
	fileItems      []*dto.FileWriteItem
}

func (s *serviceTreeCapabilityBundleService) ExportCapabilityBundle(ctx context.Context, req *dto.ExportCapabilityBundleReq) (*dto.CapabilityBundle, error) {
	sourcePaths := normalizeCapabilityExportSourcePaths(req)
	if len(sourcePaths) == 0 {
		return nil, fmt.Errorf("source_directory_path 不能为空")
	}

	bundle := &dto.CapabilityBundle{
		SchemaVersion: dto.CapabilityBundleSchemaVersion,
		Name:          strings.TrimSpace(req.Name),
		Files:         make([]*dto.CapabilityBundleFile, 0),
		Packages:      make([]*dto.CapabilityBundlePackage, 0),
	}

	seenFiles := make(map[string]struct{})
	seenPackages := make(map[string]struct{})
	var sourceAppID int64
	var sourceRoot *model.ServiceTree
	if strings.TrimSpace(req.SourceRootPath) != "" {
		var err error
		sourceRoot, err = s.serviceTreeRepo.GetServiceTreeByFullPath(strings.TrimSpace(req.SourceRootPath))
		if err != nil {
			return nil, fmt.Errorf("获取导出根目录失败 %s: %w", req.SourceRootPath, err)
		}
		if sourceRoot.Type != model.ServiceTreeTypePackage {
			return nil, fmt.Errorf("source_root_path 必须是 package 节点，当前类型: %s", sourceRoot.Type)
		}
		sourceAppID = sourceRoot.AppID
		if bundle.Name == "" {
			bundle.Name = sourceRoot.Name
		}
	}

	for _, sourcePath := range sourcePaths {
		rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("获取源目录失败 %s: %w", sourcePath, err)
		}
		if sourceAppID == 0 {
			sourceAppID = rootTree.AppID
		} else if sourceAppID != rootTree.AppID {
			return nil, fmt.Errorf("一次能力包导出只能选择同一个应用内的目录")
		}
		if sourceRoot != nil {
			if err := ensureCapabilitySourceWithinRoot(sourceRoot, rootTree); err != nil {
				return nil, err
			}
		}

		if bundle.Name == "" {
			bundle.Name = rootTree.Name
		}

		switch rootTree.Type {
		case model.ServiceTreeTypePackage:
			baseTree := rootTree
			includeBaseCode := true
			if sourceRoot != nil && sourceRoot.FullCodePath != rootTree.FullCodePath {
				baseTree = sourceRoot
				includeBaseCode = false
			}
			if err := s.appendCapabilityBundleRoot(ctx, bundle, baseTree, rootTree, includeBaseCode, seenPackages, seenFiles); err != nil {
				return nil, err
			}
		case model.ServiceTreeTypeFunction:
			if err := s.appendCapabilityBundleFunction(ctx, bundle, sourceRoot, rootTree, sourceRoot == nil, seenPackages, seenFiles); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("只能导出 package 或 function 节点，当前类型: %s", rootTree.Type)
		}
	}

	sort.Slice(bundle.Packages, func(i, j int) bool {
		return bundle.Packages[i].Path < bundle.Packages[j].Path
	})
	sort.Slice(bundle.Files, func(i, j int) bool {
		left := capabilityFileKey(bundle.Files[i].PackagePath, bundle.Files[i].Path)
		right := capabilityFileKey(bundle.Files[j].PackagePath, bundle.Files[j].Path)
		return left < right
	})

	if err := validateCapabilityBundle(bundle); err != nil {
		return nil, err
	}
	return bundle, nil
}

func (s *serviceTreeCapabilityBundleService) appendCapabilityBundleRoot(
	ctx context.Context,
	bundle *dto.CapabilityBundle,
	baseTree *model.ServiceTree,
	rootTree *model.ServiceTree,
	includeBaseCode bool,
	seenPackages map[string]struct{},
	seenFiles map[string]struct{},
) error {
	directoryFiles, err := readDirectoryFilesFromRuntimeRecursively(ctx, s.serviceTreeRepo, s.runtimeWorkspace, rootTree.AppID, rootTree.FullCodePath)
	if err != nil {
		return fmt.Errorf("读取目录文件失败: %w", err)
	}

	descendants, err := s.serviceTreeRepo.GetDescendantDirectories(rootTree.AppID, rootTree.FullCodePath)
	if err != nil {
		return fmt.Errorf("获取子目录失败: %w", err)
	}
	allDirectories := make([]*model.ServiceTree, 0, len(descendants)+1)
	allDirectories = append(allDirectories, rootTree)
	allDirectories = append(allDirectories, descendants...)
	sort.Slice(allDirectories, func(i, j int) bool {
		return allDirectories[i].FullCodePath < allDirectories[j].FullCodePath
	})

	for _, dir := range allDirectories {
		relativeDir, err := capabilityRelativePackagePath(baseTree, dir, includeBaseCode)
		if err != nil {
			return err
		}
		if relativeDir != "" {
			if err := s.addCapabilityBundlePackageWithAncestors(ctx, bundle, baseTree, dir, relativeDir, includeBaseCode, seenPackages); err != nil {
				return err
			}
		}

		for _, file := range directoryFiles[dir.FullCodePath] {
			if file == nil || isGeneratedDirectoryInitFile(file.RelativePath, file.FileName) {
				continue
			}
			filePath := capabilitySourceFilePath(file)
			if filePath == "" {
				continue
			}
			fileKey := capabilityFileKey(relativeDir, filePath)
			if _, exists := seenFiles[fileKey]; exists {
				return fmt.Errorf("能力包存在重复文件路径: %s", fileKey)
			}
			seenFiles[fileKey] = struct{}{}
			bundle.Files = append(bundle.Files, &dto.CapabilityBundleFile{
				PackagePath: relativeDir,
				Path:        filePath,
				Content:     file.Content,
			})
		}
	}

	return nil
}

func (s *serviceTreeCapabilityBundleService) appendCapabilityBundleFunction(
	ctx context.Context,
	bundle *dto.CapabilityBundle,
	baseTree *model.ServiceTree,
	functionTree *model.ServiceTree,
	includeBaseCode bool,
	seenPackages map[string]struct{},
	seenFiles map[string]struct{},
) error {
	parentPath := functionTree.GetParentFullPath()
	parentTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(parentPath)
	if err != nil {
		return fmt.Errorf("获取函数父目录失败: %w", err)
	}
	if parentTree.Type != model.ServiceTreeTypePackage {
		return fmt.Errorf("函数父节点必须是 package，当前类型: %s", parentTree.Type)
	}
	if baseTree == nil {
		baseTree = parentTree
	}

	relativeDir, err := capabilityRelativePackagePath(baseTree, parentTree, includeBaseCode)
	if err != nil {
		return err
	}
	if relativeDir != "" {
		if err := s.addCapabilityBundlePackageWithAncestors(ctx, bundle, baseTree, parentTree, relativeDir, includeBaseCode, seenPackages); err != nil {
			return err
		}
	}

	_, runtimeResp, err := s.runtimeWorkspace.readDirectoryFiles(ctx, functionTree.AppID, parentPath)
	if err != nil {
		return fmt.Errorf("读取函数源码文件失败: %w", err)
	}
	if runtimeResp == nil {
		return fmt.Errorf("读取函数源码文件失败: runtime 响应为空")
	}

	for _, file := range runtimeResp.Files {
		filePath := strings.TrimSpace(file.RelativePath)
		if filePath == "" && strings.TrimSpace(file.FileName) != "" {
			filePath = strings.TrimSpace(file.FileName) + ".go"
		}
		if path.Base(filePath) != functionTree.Code+".go" {
			continue
		}

		fileKey := capabilityFileKey(relativeDir, filePath)
		if _, exists := seenFiles[fileKey]; exists {
			return fmt.Errorf("能力包存在重复文件路径: %s", fileKey)
		}
		seenFiles[fileKey] = struct{}{}
		bundle.Files = append(bundle.Files, &dto.CapabilityBundleFile{
			PackagePath: relativeDir,
			Path:        filePath,
			Content:     file.Content,
		})
		return nil
	}

	return fmt.Errorf("未找到函数源码文件: %s/%s.go", parentPath, functionTree.Code)
}

func (s *serviceTreeCapabilityBundleService) addCapabilityBundlePackageWithAncestors(
	ctx context.Context,
	bundle *dto.CapabilityBundle,
	baseTree *model.ServiceTree,
	leafTree *model.ServiceTree,
	relativeDir string,
	includeBaseCode bool,
	seenPackages map[string]struct{},
) error {
	parts := strings.Split(strings.Trim(relativeDir, "/"), "/")
	for i := range parts {
		packagePath := strings.Join(parts[:i+1], "/")
		if packagePath == "" {
			continue
		}
		if _, exists := seenPackages[packagePath]; exists {
			continue
		}

		metaTree := leafTree
		if packagePath != relativeDir {
			fullPath := capabilityFullCodePathForRelativePackage(baseTree, packagePath, includeBaseCode)
			if fullPath != "" {
				if tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(fullPath); err == nil && tree != nil {
					metaTree = tree
				} else {
					logger.Warnf(ctx, "[CapabilityBundle] 获取父 package 元数据失败: path=%s, error=%v", fullPath, err)
					metaTree = nil
				}
			}
		}

		pkg := &dto.CapabilityBundlePackage{
			Path: packagePath,
			Name: path.Base(packagePath),
		}
		if metaTree != nil {
			pkg.Name = metaTree.Name
			pkg.Description = metaTree.Description
			pkg.Tags = metaTree.Tags
		}
		seenPackages[packagePath] = struct{}{}
		bundle.Packages = append(bundle.Packages, pkg)
	}
	return nil
}

func capabilityFullCodePathForRelativePackage(baseTree *model.ServiceTree, relativePackagePath string, includeBaseCode bool) string {
	if baseTree == nil {
		return ""
	}
	relativePackagePath = strings.Trim(relativePackagePath, "/")
	if relativePackagePath == "" {
		return baseTree.FullCodePath
	}
	if includeBaseCode {
		rootCode := strings.Trim(baseTree.Code, "/")
		if relativePackagePath == rootCode {
			return baseTree.FullCodePath
		}
		return strings.TrimRight(baseTree.FullCodePath, "/") + "/" + strings.TrimPrefix(strings.TrimPrefix(relativePackagePath, rootCode), "/")
	}
	return strings.TrimRight(baseTree.FullCodePath, "/") + "/" + relativePackagePath
}

func capabilityFileKey(packagePath, filePath string) string {
	if strings.TrimSpace(packagePath) == "" {
		return filePath
	}
	return path.Join(packagePath, filePath)
}

func (s *serviceTreeCapabilityBundleService) InstallCapabilityBundle(ctx context.Context, opts *dto.InstallCapabilityOptions, bundle *dto.CapabilityBundle) (*dto.InstallCapabilityBundleResp, error) {
	if err := validateCapabilityBundle(bundle); err != nil {
		return nil, err
	}

	targetApp, targetRootPath, err := s.resolveCapabilityInstallTarget(opts)
	if err != nil {
		return nil, err
	}
	if _, err := s.runtimeWorkspace.requireRuntimeBinding(targetApp, "安装能力包"); err != nil {
		return nil, err
	}

	plan, err := buildCapabilityBundleInstallPlan(targetRootPath, bundle)
	if err != nil {
		return nil, err
	}
	if !opts.Overwrite {
		if err := s.ensureNoCapabilityFileConflicts(ctx, targetApp, plan); err != nil {
			return nil, err
		}
	}

	var createdPaths []string
	if len(plan.directoryItems) > 0 {
		resp, err := executeBatchCreateDirectoryTree(ctx, s.serviceTreeRepo, s.runtimeWorkspace, &dto.BatchCreateDirectoryTreeReq{
			User:  targetApp.User,
			App:   targetApp.Code,
			Items: plan.directoryItems,
		})
		if err != nil {
			return nil, fmt.Errorf("创建能力包目录失败: %w", err)
		}
		if resp != nil {
			createdPaths = resp.CreatedPaths
		}
	}

	var writtenPaths []string
	var oldVersion, newVersion string
	var warnings []string
	if len(plan.fileItems) > 0 {
		resp, err := executeBatchWriteFiles(ctx, s.runtimeWorkspace, s.appService, &dto.BatchWriteFilesReq{
			User:      targetApp.User,
			App:       targetApp.Code,
			Files:     plan.fileItems,
			ForceDiff: opts.ForceDiff,
		})
		if err != nil {
			return nil, fmt.Errorf("写入能力包文件失败: %w", err)
		}
		if resp != nil {
			writtenPaths = resp.WrittenPaths
			oldVersion = resp.OldVersion
			newVersion = resp.NewVersion
			warnings = resp.Warnings
		}
	} else if s.appService != nil {
		resp, err := s.appService.UpdateApp(ctx, &dto.UpdateAppReq{
			User:              targetApp.User,
			App:               targetApp.Code,
			ForceDiff:         opts.ForceDiff,
			ChangeDescription: "安装能力包目录",
		})
		if err != nil {
			return nil, fmt.Errorf("触发目标应用更新失败: %w", err)
		}
		if resp != nil {
			oldVersion = resp.OldVersion
			newVersion = resp.NewVersion
			warnings = resp.Warnings
		}
	}

	logger.Infof(ctx, "[CapabilityBundle] 安装完成: target=%s, directories=%d, files=%d", plan.targetRootPath, len(plan.directoryItems), len(plan.fileItems))
	return &dto.InstallCapabilityBundleResp{
		Message:             fmt.Sprintf("能力包安装成功，共创建 %d 个目录，写入 %d 个文件", len(plan.directoryItems), len(plan.fileItems)),
		DirectoryCount:      len(plan.directoryItems),
		FileCount:           len(plan.fileItems),
		TargetDirectoryPath: plan.targetRootPath,
		CreatedPaths:        createdPaths,
		WrittenPaths:        writtenPaths,
		OldVersion:          oldVersion,
		NewVersion:          newVersion,
		Warnings:            warnings,
	}, nil
}

func (s *serviceTreeCapabilityBundleService) InstallCapabilityBundleFromFile(ctx context.Context, opts *dto.InstallCapabilityOptions, filePath string) (*dto.InstallCapabilityBundleResp, error) {
	bundle, err := readCapabilityBundleFile(filePath)
	if err != nil {
		return nil, err
	}
	return s.InstallCapabilityBundle(ctx, opts, bundle)
}

func readCapabilityBundleFile(filePath string) (*dto.CapabilityBundle, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取能力包 JSON 失败: %w", err)
	}
	var bundle dto.CapabilityBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("解析能力包 JSON 失败: file=%s: %w", filePath, err)
	}
	return &bundle, nil
}

func (s *serviceTreeCapabilityBundleService) resolveCapabilityInstallTarget(opts *dto.InstallCapabilityOptions) (*model.App, string, error) {
	if opts == nil {
		return nil, "", fmt.Errorf("安装选项不能为空")
	}
	targetPath, err := validateCapabilityTargetDirectoryPath(opts.TargetDirectoryPath)
	if err != nil {
		return nil, "", err
	}
	targetTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(targetPath)
	if err != nil {
		return nil, "", fmt.Errorf("获取目标目录失败: %w", err)
	}
	if targetTree.Type != model.ServiceTreeTypePackage {
		return nil, "", fmt.Errorf("目标节点必须是 package，当前类型: %s", targetTree.Type)
	}
	targetApp, err := s.appRepo.GetAppByID(targetTree.AppID)
	if err != nil {
		return nil, "", fmt.Errorf("获取目标应用失败: %w", err)
	}
	return targetApp, targetTree.FullCodePath, nil
}

func normalizeCapabilityExportSourcePaths(req *dto.ExportCapabilityBundleReq) []string {
	if req == nil {
		return nil
	}
	sourcePaths := make([]string, 0, len(req.SourceDirectoryPaths)+1)
	if strings.TrimSpace(req.SourceDirectoryPath) != "" {
		sourcePaths = append(sourcePaths, strings.TrimSpace(req.SourceDirectoryPath))
	}
	for _, sourcePath := range req.SourceDirectoryPaths {
		sourcePath = strings.TrimSpace(sourcePath)
		if sourcePath != "" {
			sourcePaths = append(sourcePaths, sourcePath)
		}
	}

	seen := make(map[string]struct{}, len(sourcePaths))
	out := make([]string, 0, len(sourcePaths))
	for _, sourcePath := range sourcePaths {
		if _, exists := seen[sourcePath]; exists {
			continue
		}
		seen[sourcePath] = struct{}{}
		out = append(out, sourcePath)
	}
	return out
}

func ensureCapabilitySourceWithinRoot(rootTree, sourceTree *model.ServiceTree) error {
	if rootTree == nil || sourceTree == nil {
		return fmt.Errorf("目录不能为空")
	}
	if sourceTree.FullCodePath == rootTree.FullCodePath {
		return nil
	}
	prefix := strings.TrimRight(rootTree.FullCodePath, "/") + "/"
	if !strings.HasPrefix(sourceTree.FullCodePath, prefix) {
		return fmt.Errorf("导出节点 %s 不在导出根目录 %s 下", sourceTree.FullCodePath, rootTree.FullCodePath)
	}
	return nil
}

func capabilityRelativePackagePath(rootTree, dir *model.ServiceTree, includeRootCode bool) (string, error) {
	if rootTree == nil || dir == nil {
		return "", fmt.Errorf("目录不能为空")
	}
	prefix := strings.TrimRight(rootTree.FullCodePath, "/") + "/"
	if dir.FullCodePath != rootTree.FullCodePath && !strings.HasPrefix(dir.FullCodePath, prefix) {
		return "", fmt.Errorf("目录 %s 不在源目录 %s 下", dir.FullCodePath, rootTree.FullCodePath)
	}
	if !includeRootCode {
		if dir.FullCodePath == rootTree.FullCodePath {
			return "", nil
		}
		return strings.TrimPrefix(dir.FullCodePath, prefix), nil
	}

	rootCode := strings.Trim(rootTree.Code, "/")
	if rootCode == "" {
		return "", fmt.Errorf("源目录 code 不能为空: %s", rootTree.FullCodePath)
	}
	if dir.FullCodePath == rootTree.FullCodePath {
		return rootCode, nil
	}
	return path.Join(rootCode, strings.TrimPrefix(dir.FullCodePath, prefix)), nil
}

func validateCapabilityBundle(bundle *dto.CapabilityBundle) error {
	if bundle == nil {
		return fmt.Errorf("capability bundle 不能为空")
	}
	if bundle.SchemaVersion != dto.CapabilityBundleSchemaVersion {
		return fmt.Errorf("不支持的 capability bundle schema_version: %s", bundle.SchemaVersion)
	}
	if len(bundle.Files) == 0 && len(bundle.Packages) == 0 {
		return fmt.Errorf("capability bundle 必须包含 files 或 packages")
	}

	seenPackages := make(map[string]struct{}, len(bundle.Packages))
	for index, pkg := range bundle.Packages {
		if pkg == nil {
			return fmt.Errorf("packages[%d] 不能为空", index)
		}
		normalized, err := validateCapabilityPackagePath(pkg.Path, fmt.Sprintf("packages[%d].path", index), false)
		if err != nil {
			return err
		}
		if normalized != pkg.Path {
			return fmt.Errorf("packages[%d].path 必须使用规范相对路径: %s", index, pkg.Path)
		}
		if _, exists := seenPackages[normalized]; exists {
			return fmt.Errorf("capability bundle 存在重复 package 路径: %s", normalized)
		}
		seenPackages[normalized] = struct{}{}
	}
	seenFiles := make(map[string]struct{}, len(bundle.Files))
	for index, file := range bundle.Files {
		if file == nil {
			return fmt.Errorf("files[%d] 不能为空", index)
		}
		packagePath, err := validateCapabilityPackagePath(file.PackagePath, fmt.Sprintf("files[%d].package_path", index), true)
		if err != nil {
			return err
		}
		if packagePath != file.PackagePath {
			return fmt.Errorf("files[%d].package_path 必须使用规范相对路径: %s", index, file.PackagePath)
		}
		if packagePath != "" {
			if _, exists := seenPackages[packagePath]; !exists {
				return fmt.Errorf("files[%d].package_path 未在 packages 中声明: %s", index, packagePath)
			}
		}

		filePath, err := validateCapabilityFilePath(file.Path, fmt.Sprintf("files[%d].path", index))
		if err != nil {
			return err
		}
		if filePath != file.Path {
			return fmt.Errorf("files[%d].path 必须使用规范相对文件名: %s", index, file.Path)
		}
		key := capabilityFileKey(packagePath, filePath)
		if _, exists := seenFiles[key]; exists {
			return fmt.Errorf("capability bundle 存在重复文件路径: %s", key)
		}
		seenFiles[key] = struct{}{}
	}

	return nil
}

func buildCapabilityBundleInstallPlan(targetRootPath string, bundle *dto.CapabilityBundle) (*capabilityBundleInstallPlan, error) {
	if err := validateCapabilityBundle(bundle); err != nil {
		return nil, err
	}
	targetRootPath, err := validateCapabilityTargetDirectoryPath(targetRootPath)
	if err != nil {
		return nil, err
	}
	packageMeta := make(map[string]*dto.CapabilityBundlePackage, len(bundle.Packages))
	targetPackagePaths := make(map[string]struct{}, len(bundle.Packages))

	for _, pkg := range bundle.Packages {
		packageMeta[pkg.Path] = pkg
		addCapabilityPackageAncestors(targetPackagePaths, pkg.Path)
	}

	fileItems := make([]*dto.FileWriteItem, 0, len(bundle.Files))
	for _, file := range bundle.Files {
		packagePath, err := validateCapabilityPackagePath(file.PackagePath, "file.package_path", true)
		if err != nil {
			return nil, err
		}
		filePath, err := validateCapabilityFilePath(file.Path, "file.path")
		if err != nil {
			return nil, err
		}
		if packagePath != "" {
			addCapabilityPackageAncestors(targetPackagePaths, packagePath)
		}

		fileName, fileType, err := splitCapabilityBundleFilePath(filePath)
		if err != nil {
			return nil, err
		}
		fileItems = append(fileItems, &dto.FileWriteItem{
			FullCodePath: joinCapabilityFullCodePath(targetRootPath, packagePath),
			FileName:     fileName,
			FileType:     fileType,
			Content:      file.Content,
			RelativePath: filePath,
		})
	}

	packagePaths := make([]string, 0, len(targetPackagePaths))
	for packagePath := range targetPackagePaths {
		packagePaths = append(packagePaths, packagePath)
	}
	sort.Slice(packagePaths, func(i, j int) bool {
		leftDepth := strings.Count(packagePaths[i], "/")
		rightDepth := strings.Count(packagePaths[j], "/")
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		return packagePaths[i] < packagePaths[j]
	})

	directoryItems := make([]*dto.DirectoryScaffoldItem, 0, len(packagePaths))
	for _, packagePath := range packagePaths {
		meta := packageMeta[packagePath]
		name := path.Base(packagePath)
		description := ""
		tags := ""
		if meta != nil {
			if strings.TrimSpace(meta.Name) != "" {
				name = strings.TrimSpace(meta.Name)
			}
			description = meta.Description
			tags = meta.Tags
		}
		directoryItems = append(directoryItems, &dto.DirectoryScaffoldItem{
			FullCodePath: joinCapabilityFullCodePath(targetRootPath, packagePath),
			Name:         name,
			Description:  description,
			Tags:         tags,
		})
	}

	return &capabilityBundleInstallPlan{
		targetRootPath: targetRootPath,
		directoryItems: directoryItems,
		fileItems:      fileItems,
	}, nil
}

func (s *serviceTreeCapabilityBundleService) ensureNoCapabilityFileConflicts(ctx context.Context, targetApp *model.App, plan *capabilityBundleInstallPlan) error {
	filesByDir := make(map[string]map[string]struct{})
	for _, item := range plan.fileItems {
		fileType := strings.TrimPrefix(strings.TrimSpace(item.FileType), ".")
		if fileType != "go" {
			continue
		}
		fileName := strings.TrimSpace(item.FileName)
		if fileName == "" {
			continue
		}
		if filesByDir[item.FullCodePath] == nil {
			filesByDir[item.FullCodePath] = make(map[string]struct{})
		}
		filesByDir[item.FullCodePath][fileName+".go"] = struct{}{}
	}

	for fullCodePath, targetFiles := range filesByDir {
		_, resp, err := s.runtimeWorkspace.readDirectoryFiles(ctx, targetApp.ID, fullCodePath)
		if err != nil {
			return fmt.Errorf("检查目标文件冲突失败: %w", err)
		}
		if resp == nil {
			continue
		}
		for _, existing := range resp.Files {
			existingPath := strings.TrimSpace(existing.RelativePath)
			if existingPath == "" && strings.TrimSpace(existing.FileName) != "" {
				existingPath = strings.TrimSpace(existing.FileName) + ".go"
			}
			if _, exists := targetFiles[existingPath]; exists {
				return fmt.Errorf("目标文件已存在: %s/%s", fullCodePath, existingPath)
			}
		}
	}
	return nil
}

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
		if err := naming.ValidateGoPackageName(part, "package 路径片段"); err != nil {
			return "", fmt.Errorf("%s 包含非法 Go package 名称 %q: %w", field, part, err)
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
			if err := naming.ValidateGoPackageName(part, "目标 package 路径片段"); err != nil {
				return "", fmt.Errorf("target_directory_path 包含非法 Go package 名称 %q: %w", part, err)
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
