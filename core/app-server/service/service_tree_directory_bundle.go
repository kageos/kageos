package service

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/naming"
	"gorm.io/gorm"
)

const (
	directoryBundlePolicyFailIfExists = "fail_if_exists"
	directoryBundlePolicySkipIfExists = "skip_if_exists"
)

type serviceTreeDirectoryBundleService struct {
	serviceTreeRepo  *repository.ServiceTreeRepository
	appRepo          *repository.AppRepository
	runtimeWorkspace *runtimeWorkspaceBridge
	appService       *AppService
}

func newServiceTreeDirectoryBundleService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	runtimeWorkspace *runtimeWorkspaceBridge,
	appService *AppService,
) *serviceTreeDirectoryBundleService {
	return &serviceTreeDirectoryBundleService{
		serviceTreeRepo:  serviceTreeRepo,
		appRepo:          appRepo,
		runtimeWorkspace: runtimeWorkspace,
		appService:       appService,
	}
}

func (s *serviceTreeDirectoryBundleService) ExportDirectoryBundle(ctx context.Context, req *dto.ExportDirectoryBundleReq) (*dto.DirectoryBundle, error) {
	if req == nil || strings.TrimSpace(req.SourceDirectoryPath) == "" {
		return nil, fmt.Errorf("source_directory_path 不能为空")
	}

	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源目录失败: %w", err)
	}
	if rootTree.Type != model.ServiceTreeTypePackage {
		return nil, fmt.Errorf("只能导出目录节点，当前类型: %s", rootTree.Type)
	}

	directoryFiles, err := readDirectoryFilesFromRuntimeRecursively(ctx, s.serviceTreeRepo, s.runtimeWorkspace, rootTree.AppID, rootTree.FullCodePath)
	if err != nil {
		return nil, fmt.Errorf("读取目录文件失败: %w", err)
	}

	descendants, err := s.serviceTreeRepo.GetDescendantDirectories(rootTree.AppID, rootTree.FullCodePath)
	if err != nil {
		return nil, fmt.Errorf("获取子目录失败: %w", err)
	}
	allDirectories := make([]*model.ServiceTree, 0, len(descendants)+1)
	allDirectories = append(allDirectories, rootTree)
	allDirectories = append(allDirectories, descendants...)

	root := buildDirectoryBundleNode(rootTree, allDirectories, directoryFiles)
	return &dto.DirectoryBundle{
		SchemaVersion: dto.DirectoryBundleSchemaVersion,
		Root:          root,
	}, nil
}

func (s *serviceTreeDirectoryBundleService) ImportDirectoryBundle(ctx context.Context, req *dto.ImportDirectoryBundleReq) (*dto.ImportDirectoryBundleResp, error) {
	if req == nil {
		return nil, fmt.Errorf("导入请求不能为空")
	}
	policy := normalizeDirectoryBundleConflictPolicy(req.ConflictPolicy)
	if policy != directoryBundlePolicyFailIfExists && policy != directoryBundlePolicySkipIfExists {
		return nil, fmt.Errorf("不支持的 conflict_policy: %s", req.ConflictPolicy)
	}
	if err := validateDirectoryBundle(req.Bundle); err != nil {
		return nil, err
	}

	targetTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.TargetDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取目标目录失败: %w", err)
	}
	if targetTree.Type != model.ServiceTreeTypePackage {
		return nil, fmt.Errorf("目标节点必须是目录，当前类型: %s", targetTree.Type)
	}

	targetApp, err := s.appRepo.GetAppByID(targetTree.AppID)
	if err != nil {
		return nil, fmt.Errorf("获取目标应用失败: %w", err)
	}

	targetRootPath := strings.TrimSuffix(targetTree.FullCodePath, "/") + "/" + req.Bundle.Root.Code
	existingRoot, err := s.serviceTreeRepo.GetServiceTreeByFullPath(targetRootPath)
	if err == nil && existingRoot != nil {
		if policy == directoryBundlePolicySkipIfExists {
			return &dto.ImportDirectoryBundleResp{
				Message:             "目标目录已存在，已跳过",
				TargetDirectoryPath: targetRootPath,
			}, nil
		}
		return nil, fmt.Errorf("目标目录已存在: %s", targetRootPath)
	}
	if err != nil && !isRecordNotFound(err) {
		return nil, fmt.Errorf("检查目标目录失败: %w", err)
	}

	directoryItems := make([]*dto.DirectoryScaffoldItem, 0)
	fileItems := make([]*dto.FileWriteItem, 0)
	if err := buildDirectoryBundleInstallItems(req.Bundle.Root, targetTree.FullCodePath, &directoryItems, &fileItems); err != nil {
		return nil, err
	}

	var createdPaths []string
	if len(directoryItems) > 0 {
		resp, err := executeBatchCreateDirectoryTree(ctx, s.serviceTreeRepo, s.runtimeWorkspace, &dto.BatchCreateDirectoryTreeReq{
			User:  targetApp.User,
			App:   targetApp.Code,
			Items: directoryItems,
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
	if len(fileItems) > 0 {
		resp, err := executeBatchWriteFiles(ctx, s.runtimeWorkspace, s.appService, &dto.BatchWriteFilesReq{
			User:  targetApp.User,
			App:   targetApp.Code,
			Files: fileItems,
		})
		if err != nil {
			return nil, fmt.Errorf("写入文件失败: %w", err)
		}
		if resp != nil {
			writtenPaths = resp.WrittenPaths
			oldVersion = resp.OldVersion
			newVersion = resp.NewVersion
		}
	}

	logger.Infof(ctx, "[DirectoryBundle] 导入完成: target=%s, directories=%d, files=%d", targetRootPath, len(directoryItems), len(fileItems))
	return &dto.ImportDirectoryBundleResp{
		Message:             fmt.Sprintf("目录导入成功，共创建 %d 个目录，写入 %d 个文件", len(directoryItems), len(fileItems)),
		DirectoryCount:      len(directoryItems),
		FileCount:           len(fileItems),
		TargetDirectoryPath: targetRootPath,
		CreatedPaths:        createdPaths,
		WrittenPaths:        writtenPaths,
		OldVersion:          oldVersion,
		NewVersion:          newVersion,
	}, nil
}

func normalizeDirectoryBundleConflictPolicy(policy string) string {
	switch strings.TrimSpace(policy) {
	case "", directoryBundlePolicyFailIfExists:
		return directoryBundlePolicyFailIfExists
	case directoryBundlePolicySkipIfExists:
		return directoryBundlePolicySkipIfExists
	default:
		return policy
	}
}

func buildDirectoryBundleNode(root *model.ServiceTree, allDirectories []*model.ServiceTree, directoryFiles map[string][]*model.FileSnapshot) *dto.DirectoryBundleNode {
	childrenByParent := make(map[string][]*model.ServiceTree)
	for _, dir := range allDirectories {
		if dir == nil || dir.FullCodePath == root.FullCodePath {
			continue
		}
		parentPath := dir.GetParentFullPath()
		childrenByParent[parentPath] = append(childrenByParent[parentPath], dir)
	}
	for parentPath := range childrenByParent {
		sort.Slice(childrenByParent[parentPath], func(i, j int) bool {
			return childrenByParent[parentPath][i].FullCodePath < childrenByParent[parentPath][j].FullCodePath
		})
	}

	var build func(*model.ServiceTree) *dto.DirectoryBundleNode
	build = func(dir *model.ServiceTree) *dto.DirectoryBundleNode {
		node := &dto.DirectoryBundleNode{
			Code:        dir.Code,
			Name:        dir.Name,
			Description: dir.Description,
		}
		for _, file := range directoryFiles[dir.FullCodePath] {
			if file == nil || isGeneratedDirectoryInitFile(file.RelativePath, file.FileName) {
				continue
			}
			filePath := directoryBundleFilePath(file)
			if filePath == "" {
				continue
			}
			node.Files = append(node.Files, &dto.DirectoryBundleFile{
				Path:    filePath,
				Content: file.Content,
			})
		}
		sort.Slice(node.Files, func(i, j int) bool {
			return node.Files[i].Path < node.Files[j].Path
		})

		for _, child := range childrenByParent[dir.FullCodePath] {
			node.Children = append(node.Children, build(child))
		}
		return node
	}
	return build(root)
}

func directoryBundleFilePath(file *model.FileSnapshot) string {
	if file == nil {
		return ""
	}
	if strings.TrimSpace(file.RelativePath) != "" {
		return strings.TrimSpace(file.RelativePath)
	}
	name := strings.TrimSpace(file.FileName)
	if name == "" {
		return ""
	}
	if path.Ext(name) != "" {
		return name
	}
	fileType := strings.Trim(strings.TrimSpace(file.FileType), ".")
	if fileType == "" {
		fileType = "go"
	}
	return name + "." + fileType
}

func validateDirectoryBundle(bundle *dto.DirectoryBundle) error {
	if bundle == nil {
		return fmt.Errorf("directory bundle 不能为空")
	}
	if bundle.SchemaVersion != dto.DirectoryBundleSchemaVersion {
		return fmt.Errorf("不支持的 directory bundle schema_version: %d", bundle.SchemaVersion)
	}
	if bundle.Root == nil {
		return fmt.Errorf("directory bundle 缺少 root")
	}
	return validateDirectoryBundleNode(bundle.Root, "root")
}

func validateDirectoryBundleNode(node *dto.DirectoryBundleNode, location string) error {
	if node == nil {
		return fmt.Errorf("%s 不能为空", location)
	}
	if err := naming.ValidateGoPackageName(node.Code, "目录 code"); err != nil {
		return fmt.Errorf("%s 目录 code 非法: %w", location, err)
	}

	seenChildCodes := make(map[string]struct{}, len(node.Children))
	for index, file := range node.Files {
		if err := validateDirectoryBundleFile(file, fmt.Sprintf("%s.files[%d]", location, index)); err != nil {
			return err
		}
	}
	for index, child := range node.Children {
		childLocation := fmt.Sprintf("%s.children[%d]", location, index)
		if child == nil {
			return fmt.Errorf("%s 不能为空", childLocation)
		}
		if _, exists := seenChildCodes[child.Code]; exists {
			return fmt.Errorf("%s 存在重复子目录 code: %s", location, child.Code)
		}
		seenChildCodes[child.Code] = struct{}{}
		if err := validateDirectoryBundleNode(child, childLocation); err != nil {
			return err
		}
	}
	return nil
}

func validateDirectoryBundleFile(file *dto.DirectoryBundleFile, location string) error {
	if file == nil {
		return fmt.Errorf("%s 不能为空", location)
	}
	filePath := strings.TrimSpace(file.Path)
	if filePath == "" {
		return fmt.Errorf("%s.path 不能为空", location)
	}
	if filePath != file.Path {
		return fmt.Errorf("%s.path 不能包含首尾空格: %s", location, file.Path)
	}
	if path.IsAbs(filePath) || strings.Contains(filePath, "\\") {
		return fmt.Errorf("%s.path 必须是目录内相对文件名: %s", location, filePath)
	}
	if strings.Contains(filePath, "/") {
		return fmt.Errorf("%s.path 第一版仅支持目录内直接文件: %s", location, filePath)
	}
	base := path.Base(filePath)
	if base == "." || base == ".." || strings.HasPrefix(base, ".") {
		return fmt.Errorf("%s.path 非法: %s", location, filePath)
	}
	if base == "init_.go" {
		return fmt.Errorf("%s.path 不允许包含 init_.go，该文件由目录脚手架生成", location)
	}
	if path.Ext(base) == "" {
		return fmt.Errorf("%s.path 必须包含文件扩展名: %s", location, filePath)
	}
	return nil
}

func buildDirectoryBundleInstallItems(
	node *dto.DirectoryBundleNode,
	targetParentPath string,
	directoryItems *[]*dto.DirectoryScaffoldItem,
	fileItems *[]*dto.FileWriteItem,
) error {
	if node == nil {
		return fmt.Errorf("目录节点不能为空")
	}
	currentPath := strings.TrimSuffix(targetParentPath, "/") + "/" + node.Code
	dirName := node.Name
	if strings.TrimSpace(dirName) == "" {
		dirName = node.Code
	}
	*directoryItems = append(*directoryItems, &dto.DirectoryScaffoldItem{
		FullCodePath: currentPath,
		Name:         dirName,
		Description:  node.Description,
	})

	for _, file := range node.Files {
		fileName, fileType, err := splitDirectoryBundleFilePath(file.Path)
		if err != nil {
			return err
		}
		*fileItems = append(*fileItems, &dto.FileWriteItem{
			FullCodePath: currentPath,
			FileName:     fileName,
			FileType:     fileType,
			Content:      file.Content,
			RelativePath: file.Path,
		})
	}
	for _, child := range node.Children {
		if err := buildDirectoryBundleInstallItems(child, currentPath, directoryItems, fileItems); err != nil {
			return err
		}
	}
	return nil
}

func splitDirectoryBundleFilePath(filePath string) (string, string, error) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return "", "", fmt.Errorf("文件路径不能为空")
	}
	ext := path.Ext(filePath)
	if ext == "" {
		return "", "", fmt.Errorf("文件路径必须包含扩展名: %s", filePath)
	}
	return strings.TrimSuffix(path.Base(filePath), ext), strings.TrimPrefix(ext, "."), nil
}

func isGeneratedDirectoryInitFile(relativePath, fileName string) bool {
	relativePath = strings.TrimSpace(relativePath)
	fileName = strings.TrimSpace(fileName)
	return path.Base(relativePath) == "init_.go" || fileName == "init_"
}

func isRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
