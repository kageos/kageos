package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
)

type capabilityBundleInstallPlan struct {
	targetRootPath string
	directoryItems []*dto.DirectoryScaffoldItem
	fileItems      []*dto.FileWriteItem
	docItems       []*capabilityBundleDocInstallItem
}

func (s *serviceTreeCapabilityBundleService) InstallCapabilityBundle(ctx context.Context, opts *dto.InstallCapabilityOptions, bundle *dto.CapabilityBundle) (*dto.InstallCapabilityBundleResp, error) {
	if opts == nil {
		return nil, fmt.Errorf("安装选项不能为空")
	}
	if err := validateCapabilityBundle(bundle); err != nil {
		return nil, err
	}
	installBundle := bundle
	if strings.TrimSpace(opts.BundleSubpath) != "" {
		filtered, err := filterCapabilityBundleBySubpath(bundle, opts.BundleSubpath)
		if err != nil {
			return nil, err
		}
		installBundle = filtered
	}

	targetApp, targetRootPath, err := s.resolveCapabilityInstallTarget(ctx, opts)
	if err != nil {
		return nil, err
	}
	if _, err := s.runtimeWorkspace.requireRuntimeBinding(targetApp, "导入目录"); err != nil {
		return nil, err
	}

	plan, err := buildCapabilityBundleInstallPlan(targetRootPath, installBundle)
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
			return nil, fmt.Errorf("创建目录失败: %w", err)
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
			User:           targetApp.User,
			App:            targetApp.Code,
			Files:          plan.fileItems,
			ForceDiff:      opts.ForceDiff,
			OperationName:  "CapabilityBundleInstall",
			OperationLabel: "导入目录",
		})
		if err != nil {
			return nil, fmt.Errorf("写入目录文件失败: %w", err)
		}
		if resp != nil {
			writtenPaths = resp.WrittenPaths
			oldVersion = resp.OldVersion
			newVersion = resp.NewVersion
			warnings = resp.Warnings
		}
	}

	createdDocPaths := make([]string, 0)
	if len(plan.docItems) > 0 {
		createdDocPaths, err = s.installCapabilityBundleDocs(ctx, targetApp, plan.docItems, opts.Overwrite)
		if err != nil {
			return nil, fmt.Errorf("导入目录文档失败: %w", err)
		}
		createdPaths = append(createdPaths, createdDocPaths...)
	}

	createdAgentTaskRefs := make([]string, 0)
	if len(installBundle.AgentTasks) > 0 {
		createdAgentTaskRefs, err = s.installCapabilityBundleAgentTasks(ctx, plan.targetRootPath, installBundle.AgentTasks)
		if err != nil {
			return nil, fmt.Errorf("导入 Agent 任务失败: %w", err)
		}
	}

	if len(plan.fileItems) == 0 && s.appService != nil {
		resp, err := s.appService.UpdateApp(ctx, &dto.UpdateAppReq{
			ResourcePath:      fmt.Sprintf("/%s/%s", targetApp.User, targetApp.Code),
			ForceDiff:         opts.ForceDiff,
			ChangeDescription: "导入目录和文档",
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

	logger.Infof(ctx, "[CapabilityBundle] 安装完成: target=%s, directories=%d, files=%d, docs=%d", plan.targetRootPath, len(plan.directoryItems), len(plan.fileItems), len(plan.docItems))
	return &dto.InstallCapabilityBundleResp{
		Message:             fmt.Sprintf("目录导入成功，共创建 %d 个目录，写入 %d 个文件，导入 %d 份文档，安装 %d 个 Agent 任务", len(plan.directoryItems), len(plan.fileItems), len(plan.docItems), len(createdAgentTaskRefs)),
		DirectoryCount:      len(plan.directoryItems),
		FileCount:           len(plan.fileItems),
		DocCount:            len(plan.docItems),
		AgentTaskCount:      len(createdAgentTaskRefs),
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

func filterCapabilityBundleBySubpath(bundle *dto.CapabilityBundle, rawSubpath string) (*dto.CapabilityBundle, error) {
	if bundle == nil {
		return nil, fmt.Errorf("目录 JSON 不能为空")
	}
	subpath, err := validateCapabilityPackagePath(strings.TrimSpace(rawSubpath), "bundle_subpath", false)
	if err != nil {
		return nil, err
	}
	rootPath := path.Base(subpath)
	rebase := func(value string) (string, bool) {
		value = strings.Trim(value, "/")
		switch {
		case value == subpath:
			return rootPath, true
		case strings.HasPrefix(value, subpath+"/"):
			return path.Join(rootPath, strings.TrimPrefix(value, subpath+"/")), true
		default:
			return "", false
		}
	}

	filtered := &dto.CapabilityBundle{
		SchemaVersion: bundle.SchemaVersion,
		Name:          bundle.Name,
		TreeNodes:     make([]*dto.CapabilityBundleTreeNode, 0),
		Docs:          make([]*dto.CapabilityBundleDoc, 0),
		Packages:      make([]*dto.CapabilityBundlePackage, 0),
		Files:         make([]*dto.CapabilityBundleFile, 0),
		AgentTasks:    make([]*dto.CapabilityBundleAgentTask, 0),
		Extensions:    cloneCapabilityBundleExtensions(bundle.Extensions),
	}

	for _, pkg := range bundle.Packages {
		if pkg == nil {
			continue
		}
		rebasedPath, ok := rebase(pkg.Path)
		if !ok {
			continue
		}
		cp := *pkg
		cp.Path = rebasedPath
		if pkg.Path == subpath && strings.TrimSpace(pkg.Name) != "" {
			filtered.Name = strings.TrimSpace(pkg.Name)
		}
		filtered.Packages = append(filtered.Packages, &cp)
	}

	for _, file := range bundle.Files {
		if file == nil {
			continue
		}
		rebasedPath, ok := rebase(file.PackagePath)
		if !ok {
			continue
		}
		cp := *file
		cp.PackagePath = rebasedPath
		filtered.Files = append(filtered.Files, &cp)
	}

	for _, node := range bundle.TreeNodes {
		if node == nil {
			continue
		}
		rebasedPath, ok := rebase(node.RelativePath)
		if !ok {
			continue
		}
		cp := *node
		cp.RelativePath = rebasedPath
		cp.ParentPath = capabilityParentPath(rebasedPath)
		cp.Code = path.Base(rebasedPath)
		filtered.TreeNodes = append(filtered.TreeNodes, &cp)
	}

	for _, doc := range bundle.Docs {
		if doc == nil {
			continue
		}
		rebasedPath, ok := rebase(doc.RelativePath)
		if !ok {
			continue
		}
		cp := *doc
		cp.RelativePath = rebasedPath
		filtered.Docs = append(filtered.Docs, &cp)
	}

	for _, task := range bundle.AgentTasks {
		if task == nil {
			continue
		}
		rebasedPath, ok := rebase(task.RelativePath)
		if !ok {
			continue
		}
		cp := *task
		cp.RelativePath = rebasedPath
		filtered.AgentTasks = append(filtered.AgentTasks, &cp)
	}

	if len(filtered.Packages) == 0 && len(filtered.Files) == 0 && len(filtered.Docs) == 0 {
		return nil, fmt.Errorf("bundle_subpath 未匹配到可安装目录: %s", subpath)
	}
	if err := validateCapabilityBundle(filtered); err != nil {
		return nil, fmt.Errorf("bundle_subpath %s 过滤后目录 JSON 无效: %w", subpath, err)
	}
	return filtered, nil
}

func cloneCapabilityBundleExtensions(extensions map[string]interface{}) map[string]interface{} {
	if len(extensions) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(extensions))
	for key, value := range extensions {
		out[key] = value
	}
	return out
}

func readCapabilityBundleFile(filePath string) (*dto.CapabilityBundle, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取目录 JSON 失败: %w", err)
	}
	var bundle dto.CapabilityBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("解析目录 JSON 失败: file=%s: %w", filePath, err)
	}
	return &bundle, nil
}

func (s *serviceTreeCapabilityBundleService) resolveCapabilityInstallTarget(ctx context.Context, opts *dto.InstallCapabilityOptions) (*model.App, string, error) {
	if opts == nil {
		return nil, "", fmt.Errorf("安装选项不能为空")
	}
	targetPath, err := validateCapabilityTargetDirectoryPath(opts.TargetDirectoryPath)
	if err != nil {
		return nil, "", err
	}
	targetTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(ctx, targetPath)
	if err != nil {
		return nil, "", fmt.Errorf("获取目标目录失败: %w", err)
	}
	if targetTree.Type != model.ServiceTreeTypePackage {
		return nil, "", fmt.Errorf("目标节点必须是 package，当前类型: %s", targetTree.Type)
	}
	targetApp, err := s.appRepo.GetAppByIDContext(ctx, targetTree.AppID)
	if err != nil {
		return nil, "", fmt.Errorf("获取目标应用失败: %w", err)
	}
	return targetApp, targetTree.FullCodePath, nil
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
	treeNodeMeta := make(map[string]*dto.CapabilityBundleTreeNode, len(bundle.TreeNodes))
	targetPackagePaths := make(map[string]struct{}, len(bundle.Packages))

	for _, pkg := range bundle.Packages {
		packageMeta[pkg.Path] = pkg
		addCapabilityPackageAncestors(targetPackagePaths, pkg.Path)
	}
	for _, node := range bundle.TreeNodes {
		if node != nil {
			treeNodeMeta[node.RelativePath] = node
		}
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

	docItems := make([]*capabilityBundleDocInstallItem, 0, len(bundle.Docs))
	for _, doc := range bundle.Docs {
		relativePath, err := validateCapabilityTreeNodePath(doc.RelativePath, "doc.relative_path", false)
		if err != nil {
			return nil, err
		}
		parentRelativePath := capabilityParentPath(relativePath)
		if parentRelativePath != "" {
			addCapabilityPackageAncestors(targetPackagePaths, parentRelativePath)
		}
		meta := treeNodeMeta[relativePath]
		code := path.Base(relativePath)
		name := strings.TrimSpace(doc.Name)
		description := ""
		tags := ""
		if meta != nil {
			if name == "" {
				name = strings.TrimSpace(meta.Name)
			}
			description = meta.Description
			tags = strings.Join(meta.Tags, ",")
		}
		if name == "" {
			name = strings.TrimSuffix(code, codeSuffixDocs)
		}
		docItems = append(docItems, &capabilityBundleDocInstallItem{
			FullCodePath:       joinCapabilityFullCodePath(targetRootPath, relativePath),
			ParentFullCodePath: joinCapabilityFullCodePath(targetRootPath, parentRelativePath),
			Code:               code,
			Name:               name,
			Description:        description,
			Tags:               tags,
			Content:            doc.Content,
			Format:             doc.Format,
			Summary:            doc.Summary,
			Category:           doc.Category,
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
		docItems:       docItems,
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
