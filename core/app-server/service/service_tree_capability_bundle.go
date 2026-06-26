package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/naming"
	"gorm.io/gorm"
)

const maxRemoteCapabilityBundleBytes = 32 << 20

type capabilityBundleInstallPlan struct {
	targetRootPath string
	directoryItems []*dto.DirectoryScaffoldItem
	fileItems      []*dto.FileWriteItem
	docItems       []*capabilityBundleDocInstallItem
}

type capabilityBundleDocInstallItem struct {
	FullCodePath       string
	ParentFullCodePath string
	Code               string
	Name               string
	Description        string
	Tags               string
	Content            string
	Format             string
	Summary            string
	Category           string
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
		Docs:          make([]*dto.CapabilityBundleDoc, 0),
	}

	seenFiles := make(map[string]struct{})
	seenPackages := make(map[string]struct{})
	seenTreeNodes := make(map[string]struct{})
	seenDocs := make(map[string]struct{})
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
			if err := s.appendCapabilityBundleRoot(ctx, bundle, baseTree, rootTree, includeBaseCode, seenPackages, seenFiles, seenTreeNodes, seenDocs); err != nil {
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
	descendantNodes, err := s.serviceTreeRepo.GetDescendantNodes(rootTree.AppID, rootTree.FullCodePath)
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

func (s *serviceTreeCapabilityBundleService) appendCapabilityBundleDocs(
	ctx context.Context,
	bundle *dto.CapabilityBundle,
	baseTree *model.ServiceTree,
	nodes []*model.ServiceTree,
	includeBaseCode bool,
	seenDocs map[string]struct{},
) error {
	if s.docService == nil || s.docService.docRepo == nil {
		return nil
	}

	docNodeIDs := make([]int64, 0)
	docNodesByID := make(map[int64]*model.ServiceTree)
	for _, node := range nodes {
		if node == nil || node.Type != model.ServiceTreeTypeDocs {
			continue
		}
		docNodeIDs = append(docNodeIDs, node.ID)
		docNodesByID[node.ID] = node
	}
	if len(docNodeIDs) == 0 {
		return nil
	}

	docs, err := s.docService.docRepo.ListByTreeIDs(docNodeIDs)
	if err != nil {
		return fmt.Errorf("获取文档内容失败: %w", err)
	}
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		node := docNodesByID[doc.TreeID]
		if node == nil {
			continue
		}
		relativePath, err := capabilityRelativeTreeNodePath(baseTree, node, includeBaseCode)
		if err != nil {
			return err
		}
		if relativePath == "" {
			continue
		}
		if _, exists := seenDocs[relativePath]; exists {
			continue
		}
		seenDocs[relativePath] = struct{}{}
		bundle.Docs = append(bundle.Docs, &dto.CapabilityBundleDoc{
			RelativePath: relativePath,
			Name:         doc.Name,
			Content:      doc.Content,
			Format:       doc.Format,
			Summary:      doc.Summary,
			Category:     doc.Category,
		})
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
	function, err := s.appService.functionRepo.GetFunctionByID(node.RefID)
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
	if _, err := s.runtimeWorkspace.requireRuntimeBinding(targetApp, "导入目录"); err != nil {
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
		Message:             fmt.Sprintf("目录导入成功，共创建 %d 个目录，写入 %d 个文件，导入 %d 份文档", len(plan.directoryItems), len(plan.fileItems), len(plan.docItems)),
		DirectoryCount:      len(plan.directoryItems),
		FileCount:           len(plan.fileItems),
		DocCount:            len(plan.docItems),
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

func (s *serviceTreeCapabilityBundleService) InstallCapabilityBundleFromURL(ctx context.Context, opts *dto.InstallCapabilityOptions, bundleURL, installKey string) (*dto.InstallCapabilityBundleResp, error) {
	bundle, err := downloadCapabilityBundle(ctx, bundleURL, installKey)
	if err != nil {
		return nil, err
	}
	return s.InstallCapabilityBundle(ctx, opts, bundle)
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

func downloadCapabilityBundle(ctx context.Context, rawURL, installKey string) (*dto.CapabilityBundle, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("目录 URL 不能为空")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("目录 URL 无效: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("目录 URL 仅支持 http/https")
	}
	if installKey == "" {
		rawPath := strings.Trim(parsed.EscapedPath(), "/")
		parts := strings.Split(rawPath, "/")
		if len(parts) >= 2 && parts[len(parts)-2] == "bundle" {
			key, err := url.PathUnescape(parts[len(parts)-1])
			if err != nil {
				return nil, fmt.Errorf("解析 URL 中的安装密钥失败: %w", err)
			}
			installKey = key
			parts = parts[:len(parts)-1]
			parsed.Path = "/" + strings.Join(parts, "/")
			parsed.RawPath = ""
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建目录下载请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if installKey = strings.TrimSpace(installKey); installKey != "" {
		req.Header.Set("X-Install-Key", installKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载目录失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("下载目录失败: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxRemoteCapabilityBundleBytes+1))
	if err != nil {
		return nil, fmt.Errorf("读取目录响应失败: %w", err)
	}
	if len(data) > maxRemoteCapabilityBundleBytes {
		return nil, fmt.Errorf("目录 JSON 过大，最大支持 %d MB", maxRemoteCapabilityBundleBytes>>20)
	}

	var bundle dto.CapabilityBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("解析目录 JSON 失败: %w", err)
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
		return fmt.Errorf("目录 JSON 不能为空")
	}
	if bundle.SchemaVersion != dto.CapabilityBundleSchemaVersion {
		return fmt.Errorf("不支持的目录 JSON schema_version: %s", bundle.SchemaVersion)
	}
	if len(bundle.Files) == 0 && len(bundle.Packages) == 0 {
		if len(bundle.Docs) == 0 {
			return fmt.Errorf("目录 JSON 必须包含 files、packages 或 docs")
		}
	}
	if err := validateCapabilityBundleTreeNodes(bundle.TreeNodes); err != nil {
		return err
	}
	if err := validateCapabilityBundleDocs(bundle.Docs, bundle.TreeNodes); err != nil {
		return err
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
			return fmt.Errorf("目录 JSON 存在重复 package 路径: %s", normalized)
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
			return fmt.Errorf("目录 JSON 存在重复文件路径: %s", key)
		}
		seenFiles[key] = struct{}{}
	}
	return nil
}

func validateCapabilityBundleDocs(docs []*dto.CapabilityBundleDoc, treeNodes []*dto.CapabilityBundleTreeNode) error {
	nodesByPath := make(map[string]*dto.CapabilityBundleTreeNode, len(treeNodes))
	for _, node := range treeNodes {
		if node != nil {
			nodesByPath[node.RelativePath] = node
		}
	}

	seen := make(map[string]struct{}, len(docs))
	for index, doc := range docs {
		if doc == nil {
			return fmt.Errorf("docs[%d] 不能为空", index)
		}
		relativePath, err := validateCapabilityTreeNodePath(doc.RelativePath, fmt.Sprintf("docs[%d].relative_path", index), false)
		if err != nil {
			return err
		}
		if relativePath != doc.RelativePath {
			return fmt.Errorf("docs[%d].relative_path 必须使用规范相对路径: %s", index, doc.RelativePath)
		}
		if _, exists := seen[relativePath]; exists {
			return fmt.Errorf("目录 JSON 存在重复 docs 路径: %s", relativePath)
		}
		seen[relativePath] = struct{}{}
		if len(nodesByPath) > 0 {
			node, exists := nodesByPath[relativePath]
			if !exists {
				return fmt.Errorf("docs[%d].relative_path 未在 tree_nodes 中声明: %s", index, relativePath)
			}
			if node.Type != model.ServiceTreeTypeDocs {
				return fmt.Errorf("docs[%d].relative_path 对应的 tree node 不是 docs: %s", index, relativePath)
			}
		}
	}
	return nil
}

func validateCapabilityBundleTreeNodes(nodes []*dto.CapabilityBundleTreeNode) error {
	seen := make(map[string]struct{}, len(nodes))
	for index, node := range nodes {
		if node == nil {
			return fmt.Errorf("tree_nodes[%d] 不能为空", index)
		}
		relativePath, err := validateCapabilityTreeNodePath(node.RelativePath, fmt.Sprintf("tree_nodes[%d].relative_path", index), false)
		if err != nil {
			return err
		}
		if relativePath != node.RelativePath {
			return fmt.Errorf("tree_nodes[%d].relative_path 必须使用规范相对路径: %s", index, node.RelativePath)
		}
		if _, exists := seen[relativePath]; exists {
			return fmt.Errorf("目录 JSON 存在重复 tree node 路径: %s", relativePath)
		}
		seen[relativePath] = struct{}{}
		if node.Type != model.ServiceTreeTypePackage && node.Type != model.ServiceTreeTypeFunction && node.Type != model.ServiceTreeTypeDocs {
			return fmt.Errorf("tree_nodes[%d].type 不支持: %s", index, node.Type)
		}
		if strings.TrimSpace(node.Code) == "" {
			return fmt.Errorf("tree_nodes[%d].code 不能为空", index)
		}
		if path.Base(relativePath) != node.Code {
			return fmt.Errorf("tree_nodes[%d].code 必须等于 relative_path 的最后一段: want=%s got=%s", index, path.Base(relativePath), node.Code)
		}
		parentPath, err := validateCapabilityTreeNodePath(node.ParentPath, fmt.Sprintf("tree_nodes[%d].parent_path", index), true)
		if err != nil {
			return err
		}
		expectedParent := capabilityParentPath(relativePath)
		if parentPath != expectedParent {
			return fmt.Errorf("tree_nodes[%d].parent_path 必须等于 relative_path 的父路径: want=%s got=%s", index, expectedParent, parentPath)
		}
	}
	for index, node := range nodes {
		parentPath := strings.TrimSpace(node.ParentPath)
		if parentPath == "" {
			continue
		}
		if _, exists := seen[parentPath]; !exists {
			return fmt.Errorf("tree_nodes[%d].parent_path 未在 tree_nodes 中声明: %s", index, parentPath)
		}
	}
	return nil
}

func validateCapabilityTreeNodePath(nodePath string, field string, allowEmpty bool) (string, error) {
	if nodePath != strings.TrimSpace(nodePath) {
		return "", fmt.Errorf("%s 不能包含首尾空格: %s", field, nodePath)
	}
	if strings.HasPrefix(nodePath, "/") || strings.HasSuffix(nodePath, "/") {
		return "", fmt.Errorf("%s 必须是相对节点路径: %s", field, nodePath)
	}
	nodePath = strings.Trim(nodePath, "/")
	if nodePath == "" {
		if allowEmpty {
			return "", nil
		}
		return "", fmt.Errorf("%s 不能为空", field)
	}
	if strings.Contains(nodePath, "\\") || path.IsAbs(nodePath) {
		return "", fmt.Errorf("%s 必须是相对节点路径: %s", field, nodePath)
	}
	if cleaned := path.Clean(nodePath); cleaned != nodePath || cleaned == "." {
		return "", fmt.Errorf("%s 必须使用规范相对路径: %s", field, nodePath)
	}
	parts := strings.Split(nodePath, "/")
	if err := rejectWorkspaceBoundCapabilityPath(parts, field, nodePath); err != nil {
		return "", err
	}
	for _, part := range parts {
		if part == "" || strings.HasPrefix(part, ".") {
			return "", fmt.Errorf("%s 包含非法路径片段: %s", field, nodePath)
		}
	}
	return nodePath, nil
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

func (s *serviceTreeCapabilityBundleService) installCapabilityBundleDocs(
	ctx context.Context,
	targetApp *model.App,
	items []*capabilityBundleDocInstallItem,
	overwrite bool,
) ([]string, error) {
	if s.docService == nil || s.docService.docRepo == nil {
		return nil, fmt.Errorf("文档服务未初始化")
	}

	createdPaths := make([]string, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(item.FullCodePath)
		switch {
		case err == nil:
			if tree.Type != model.ServiceTreeTypeDocs {
				return nil, fmt.Errorf("目标路径已存在且不是 docs: %s", item.FullCodePath)
			}
			if !overwrite {
				return nil, fmt.Errorf("目标文档已存在: %s", item.FullCodePath)
			}
			if err := s.updateCapabilityBundleDocTree(ctx, tree, item); err != nil {
				return nil, err
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			tree, err = s.createCapabilityBundleDocTree(ctx, targetApp, item)
			if err != nil {
				return nil, err
			}
			createdPaths = append(createdPaths, tree.FullCodePath)
		default:
			return nil, fmt.Errorf("检查目标文档失败: %w", err)
		}

		if err := s.upsertCapabilityBundleDocContent(ctx, tree, item); err != nil {
			return nil, err
		}
	}
	return createdPaths, nil
}

func (s *serviceTreeCapabilityBundleService) createCapabilityBundleDocTree(
	ctx context.Context,
	targetApp *model.App,
	item *capabilityBundleDocInstallItem,
) (*model.ServiceTree, error) {
	if targetApp == nil {
		return nil, fmt.Errorf("目标应用不能为空")
	}
	if _, err := s.serviceTreeRepo.GetServiceTreeByFullPath(item.ParentFullCodePath); err != nil {
		return nil, fmt.Errorf("目标文档父目录不存在: %s: %w", item.ParentFullCodePath, err)
	}
	requestUser := contextx.GetRequestUser(ctx)
	tree := &model.ServiceTree{
		Name:             item.Name,
		Code:             item.Code,
		Type:             model.ServiceTreeTypeDocs,
		Description:      item.Description,
		Tags:             item.Tags,
		AppID:            targetApp.ID,
		FullCodePath:     item.FullCodePath,
		AddVersionNum:    extractVersionNumForServiceTree(targetApp.Version),
		UpdateVersionNum: 0,
	}
	if requestUser != "" {
		tree.CreatedBy = requestUser
		tree.UpdatedBy = requestUser
	}
	if err := s.serviceTreeRepo.CreateServiceTreeWithParentPath(tree, ""); err != nil {
		return nil, fmt.Errorf("创建文档节点失败: %w", err)
	}
	return tree, nil
}

func (s *serviceTreeCapabilityBundleService) updateCapabilityBundleDocTree(
	ctx context.Context,
	tree *model.ServiceTree,
	item *capabilityBundleDocInstallItem,
) error {
	if tree == nil || item == nil {
		return nil
	}
	tree.Name = item.Name
	tree.Description = item.Description
	tree.Tags = item.Tags
	if requestUser := contextx.GetRequestUser(ctx); requestUser != "" {
		tree.UpdatedBy = requestUser
	}
	if err := s.serviceTreeRepo.UpdateServiceTree(tree); err != nil {
		return fmt.Errorf("更新文档节点失败: %w", err)
	}
	return nil
}

func (s *serviceTreeCapabilityBundleService) upsertCapabilityBundleDocContent(
	ctx context.Context,
	tree *model.ServiceTree,
	item *capabilityBundleDocInstallItem,
) error {
	if tree == nil || item == nil {
		return nil
	}
	doc, err := s.docService.docRepo.GetByTreeID(tree.ID)
	switch {
	case err == nil:
		doc.Name = tree.Name
		doc.Content = item.Content
		doc.Format = defaultCapabilityBundleDocFormat(item.Format)
		doc.Summary = item.Summary
		doc.Category = item.Category
		doc.AppID = tree.AppID
		doc.FullCodePath = tree.FullCodePath
		if requestUser := contextx.GetRequestUser(ctx); requestUser != "" {
			doc.UpdatedBy = requestUser
		}
		if err := s.docService.docRepo.Update(doc); err != nil {
			return fmt.Errorf("更新文档内容失败: %w", err)
		}
		tree.RefID = doc.ID
	case errors.Is(err, gorm.ErrRecordNotFound):
		doc = &model.Docs{
			Name:         tree.Name,
			Content:      item.Content,
			Format:       defaultCapabilityBundleDocFormat(item.Format),
			Summary:      item.Summary,
			Category:     item.Category,
			AppID:        tree.AppID,
			TreeID:       tree.ID,
			FullCodePath: tree.FullCodePath,
		}
		if requestUser := contextx.GetRequestUser(ctx); requestUser != "" {
			doc.CreatedBy = requestUser
			doc.UpdatedBy = requestUser
		}
		if err := s.docService.docRepo.Create(doc); err != nil {
			return fmt.Errorf("创建文档内容失败: %w", err)
		}
		tree.RefID = doc.ID
	default:
		return fmt.Errorf("获取文档内容失败: %w", err)
	}
	if err := s.serviceTreeRepo.UpdateServiceTree(tree); err != nil {
		logger.Warnf(ctx, "[CapabilityBundle] 更新文档节点 RefID 失败: %v", err)
	}
	return nil
}

func defaultCapabilityBundleDocFormat(format string) string {
	if strings.TrimSpace(format) == "" {
		return "markdown"
	}
	return strings.TrimSpace(format)
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
