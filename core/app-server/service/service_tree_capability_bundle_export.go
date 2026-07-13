package service

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
)

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
		Docs:          make([]*dto.CapabilityBundleDoc, 0),
		AgentTasks:    make([]*dto.CapabilityBundleAgentTask, 0),
	}

	seenFiles := make(map[string]struct{})
	seenPackages := make(map[string]struct{})
	seenTreeNodes := make(map[string]struct{})
	seenDocs := make(map[string]struct{})
	seenAgentTasks := make(map[string]struct{})
	var sourceAppID int64
	var sourceRoot *model.ServiceTree
	if strings.TrimSpace(req.SourceRootPath) != "" {
		var err error
		sourceRoot, err = s.serviceTreeRepo.GetServiceTreeByFullPath(ctx, strings.TrimSpace(req.SourceRootPath))
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
		rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(ctx, sourcePath)
		if err != nil {
			return nil, fmt.Errorf("获取源目录失败 %s: %w", sourcePath, err)
		}
		if sourceAppID == 0 {
			sourceAppID = rootTree.AppID
		} else if sourceAppID != rootTree.AppID {
			return nil, fmt.Errorf("一次目录导出只能选择同一个应用内的目录")
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
			if err := s.appendCapabilityBundleRoot(ctx, bundle, baseTree, rootTree, includeBaseCode, seenPackages, seenFiles, seenTreeNodes, seenDocs, seenAgentTasks); err != nil {
				return nil, err
			}
		case model.ServiceTreeTypeFunction:
			if err := s.appendCapabilityBundleFunction(ctx, bundle, sourceRoot, rootTree, sourceRoot == nil, seenPackages, seenFiles, seenTreeNodes); err != nil {
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
	sort.Slice(bundle.TreeNodes, func(i, j int) bool {
		return bundle.TreeNodes[i].RelativePath < bundle.TreeNodes[j].RelativePath
	})
	sort.Slice(bundle.Docs, func(i, j int) bool {
		return bundle.Docs[i].RelativePath < bundle.Docs[j].RelativePath
	})
	sort.Slice(bundle.AgentTasks, func(i, j int) bool {
		left := capabilityAgentTaskKey(bundle.AgentTasks[i].RelativePath, bundle.AgentTasks[i].Code)
		right := capabilityAgentTaskKey(bundle.AgentTasks[j].RelativePath, bundle.AgentTasks[j].Code)
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
	seenTreeNodes map[string]struct{},
	seenDocs map[string]struct{},
	seenAgentTasks map[string]struct{},
) error {
	directoryFiles, err := readDirectoryFilesFromRuntimeRecursively(ctx, s.serviceTreeRepo, s.runtimeWorkspace, rootTree.AppID, rootTree.FullCodePath)
	if err != nil {
		return fmt.Errorf("读取目录文件失败: %w", err)
	}

	descendants, err := s.serviceTreeRepo.GetDescendantDirectories(ctx, rootTree.AppID, rootTree.FullCodePath)
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
			if file == nil || isExcludedCapabilitySourceFile(file.RelativePath, file.FileName) {
				continue
			}
			filePath := capabilitySourceFilePath(file)
			if filePath == "" {
				continue
			}
			fileKey := capabilityFileKey(relativeDir, filePath)
			if _, exists := seenFiles[fileKey]; exists {
				return fmt.Errorf("目录 JSON 存在重复文件路径: %s", fileKey)
			}
			seenFiles[fileKey] = struct{}{}
			bundle.Files = append(bundle.Files, &dto.CapabilityBundleFile{
				PackagePath: relativeDir,
				Path:        filePath,
				Content:     file.Content,
			})
		}
	}

	treeNodes := make([]*model.ServiceTree, 0)
	treeNodes = append(treeNodes, rootTree)
	descendantNodes, err := s.serviceTreeRepo.GetDescendantNodes(ctx, rootTree.AppID, rootTree.FullCodePath)
	if err != nil {
		return fmt.Errorf("获取子节点失败: %w", err)
	}
	treeNodes = append(treeNodes, descendantNodes...)
	if err := s.appendCapabilityBundleTreeNodes(ctx, bundle, baseTree, treeNodes, includeBaseCode, seenTreeNodes); err != nil {
		return err
	}
	if err := s.appendCapabilityBundleDocs(ctx, bundle, baseTree, treeNodes, includeBaseCode, seenDocs); err != nil {
		return err
	}
	if err := s.appendCapabilityBundleAgentTasks(ctx, bundle, baseTree, rootTree, includeBaseCode, seenAgentTasks); err != nil {
		return err
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
	seenTreeNodes map[string]struct{},
) error {
	parentPath := functionTree.GetParentFullPath()
	parentTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(ctx, parentPath)
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

	foundSourceFile := false
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
			return fmt.Errorf("目录 JSON 存在重复文件路径: %s", fileKey)
		}
		seenFiles[fileKey] = struct{}{}
		bundle.Files = append(bundle.Files, &dto.CapabilityBundleFile{
			PackagePath: relativeDir,
			Path:        filePath,
			Content:     file.Content,
		})
		foundSourceFile = true
		break
	}
	if !foundSourceFile {
		return fmt.Errorf("未找到函数源码文件: %s/%s.go", parentPath, functionTree.Code)
	}
	if err := s.appendCapabilityBundleTreeNodes(ctx, bundle, baseTree, []*model.ServiceTree{functionTree}, includeBaseCode, seenTreeNodes); err != nil {
		return err
	}
	return nil
}

func (s *serviceTreeCapabilityBundleService) appendCapabilityBundleTreeNodes(
	ctx context.Context,
	bundle *dto.CapabilityBundle,
	baseTree *model.ServiceTree,
	nodes []*model.ServiceTree,
	includeBaseCode bool,
	seenTreeNodes map[string]struct{},
) error {
	for _, node := range nodes {
		if node == nil || strings.TrimSpace(node.FullCodePath) == "" {
			continue
		}
		relativePath, err := capabilityRelativeTreeNodePath(baseTree, node, includeBaseCode)
		if err != nil {
			return err
		}
		if relativePath == "" {
			continue
		}
		if _, exists := seenTreeNodes[relativePath]; exists {
			continue
		}
		treeNode := &dto.CapabilityBundleTreeNode{
			RelativePath: relativePath,
			ParentPath:   capabilityParentPath(relativePath),
			Type:         node.Type,
			Code:         node.Code,
			Name:         node.Name,
			Description:  node.Description,
			Tags:         splitCapabilityTags(node.Tags),
		}
		if node.Type == model.ServiceTreeTypeFunction {
			treeNode.TemplateType = node.TemplateType
			if function := s.getFunctionForBundleTreeNode(ctx, node); function != nil {
				treeNode.Method = function.Method
				treeNode.Router = function.Router
				if treeNode.TemplateType == "" {
					treeNode.TemplateType = function.TemplateType
				}
			}
		}
		seenTreeNodes[relativePath] = struct{}{}
		bundle.TreeNodes = append(bundle.TreeNodes, treeNode)
	}
	return nil
}

func (s *serviceTreeCapabilityBundleService) getFunctionForBundleTreeNode(ctx context.Context, node *model.ServiceTree) *model.Function {
	if node == nil {
		return nil
	}
	if node.Function != nil {
		return node.Function
	}
	if s == nil || s.appService == nil || s.appService.functionRepo == nil || node.RefID <= 0 {
		return nil
	}
	function, err := s.appService.functionRepo.GetFunctionByID(ctx, node.RefID)
	if err != nil {
		logger.Warnf(ctx, "[CapabilityBundle] 获取函数元数据失败: full_code_path=%s ref_id=%d error=%v", node.FullCodePath, node.RefID, err)
		return nil
	}
	return function
}

func capabilityRelativeTreeNodePath(baseTree *model.ServiceTree, node *model.ServiceTree, includeBaseCode bool) (string, error) {
	if baseTree == nil || node == nil {
		return "", nil
	}
	if node.Type == model.ServiceTreeTypePackage {
		return capabilityRelativePackagePath(baseTree, node, includeBaseCode)
	}
	parentPath := strings.TrimRight(node.GetParentFullPath(), "/")
	parentTree := &model.ServiceTree{
		Code:         path.Base(parentPath),
		FullCodePath: parentPath,
	}
	parentRelativePath, err := capabilityRelativePackagePath(baseTree, parentTree, includeBaseCode)
	if err != nil {
		return "", err
	}
	return path.Join(parentRelativePath, strings.TrimSpace(node.Code)), nil
}

func capabilityParentPath(relativePath string) string {
	relativePath = strings.Trim(relativePath, "/")
	if relativePath == "" || !strings.Contains(relativePath, "/") {
		return ""
	}
	return path.Dir(relativePath)
}

func splitCapabilityTags(tags string) []string {
	parts := strings.FieldsFunc(tags, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；'
	})
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
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
				if tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(ctx, fullPath); err == nil && tree != nil {
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
