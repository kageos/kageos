package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/hub/backend/dto"
	"github.com/ai-agent-os/hub/backend/model"
	"github.com/ai-agent-os/hub/backend/repository"
	"github.com/ai-agent-os/hub/backend/utils"
)

// HubDirectoryService Hub 目录服务
type HubDirectoryService struct {
	directoryRepo    *repository.HubDirectoryRepository
	serviceTreeRepo  *repository.HubServiceTreeRepository
	snapshotRepo     *repository.HubSnapshotRepository
	fileSnapshotRepo *repository.HubFileSnapshotRepository
}

// NewHubDirectoryService 创建 Hub 目录服务（依赖注入）
func NewHubDirectoryService(
	directoryRepo *repository.HubDirectoryRepository,
	serviceTreeRepo *repository.HubServiceTreeRepository,
	snapshotRepo *repository.HubSnapshotRepository,
	fileSnapshotRepo *repository.HubFileSnapshotRepository,
) *HubDirectoryService {
	return &HubDirectoryService{
		directoryRepo:    directoryRepo,
		serviceTreeRepo:  serviceTreeRepo,
		snapshotRepo:     snapshotRepo,
		fileSnapshotRepo: fileSnapshotRepo,
	}
}

// PublishDirectory 发布目录到 Hub
func (s *HubDirectoryService) PublishDirectory(ctx context.Context, req *dto.PublishHubDirectoryRequest, publisherUsername string) (*dto.PublishHubDirectoryResponse, error) {
	// 1. 解析版本号
	version := req.Version
	if version == "" {
		version = "v1"
	}
	versionNum := extractVersionNum(version)

	// 2. 验证目录树
	if req.DirectoryTree == nil {
		return nil, fmt.Errorf("目录树不能为空")
	}

	// 3. 序列化目录树（JSON格式）
	directoryTreeJSON, err := json.Marshal(req.DirectoryTree)
	if err != nil {
		return nil, fmt.Errorf("序列化目录树失败: %w", err)
	}

	// 4. 统计信息（递归统计，包含函数）
	stats := s.countDirectoryTreeStats(req.DirectoryTree)
	totalDirectories := stats.DirectoryCount
	totalFiles := stats.FileCount
	totalFunctions := stats.FunctionCount

	// 5. 获取根目录信息
	rootPath := req.DirectoryTree.Path

	// 提取 package_path（去掉 user/app 前缀）
	packagePath := extractPackagePath(rootPath, req.SourceUser, req.SourceApp)

	// 6. 创建 Hub 目录记录
	now := time.Now()
	directory := &model.HubDirectory{
		Name:                 req.Name,
		Description:          req.Description,
		Category:             req.Category,
		Tags:                 strings.Join(req.Tags, ","),
		PackagePath:          packagePath,
		FullCodePath:         rootPath,
		ParentDirID:          0, // 根目录
		SourceUser:           req.SourceUser,
		SourceApp:            req.SourceApp,
		SourceDirectoryPath:  req.SourceDirectoryPath,
		PublisherUsername:    publisherUsername,
		PublishedAt:          &now,
		ServiceFeePersonal:   req.ServiceFeePersonal,
		ServiceFeeEnterprise: req.ServiceFeeEnterprise,
		Version:              version,
		VersionNum:           versionNum,
		DirectoryTree:        string(directoryTreeJSON),
		DirectoryCount:       totalDirectories - 1, // 减去根目录
		FileCount:            totalFiles,
		FunctionCount:        totalFunctions,
	}

	// 7. 保存目录记录
	if err := s.directoryRepo.Create(ctx, directory); err != nil {
		return nil, fmt.Errorf("创建目录记录失败: %w", err)
	}

	// 8. 创建快照
	snapshot := &model.HubSnapshot{
		HubDirectoryID: directory.ID,
		Version:        version,
		VersionNum:     versionNum,
		SnapshotAt:     now,
		DirectoryCount: totalDirectories - 1,
		FileCount:      totalFiles,
		FunctionCount:  totalFunctions,
		SnapshotData:   string(directoryTreeJSON),
		IsCurrent:      true,
	}
	if err := s.snapshotRepo.Create(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("创建快照失败: %w", err)
	}

	// 9. 构建服务树节点和文件快照（递归处理）
	if err := s.buildServiceTreeAndSnapshots(ctx, req.DirectoryTree, directory.ID, snapshot.ID, rootPath, 0, 0); err != nil {
		return nil, fmt.Errorf("构建服务树和快照失败: %w", err)
	}

	// 10. 构建响应
	hubDirectoryURL := fmt.Sprintf("/hub/directories/%d", directory.ID)
	return &dto.PublishHubDirectoryResponse{
		HubDirectoryID:  directory.ID,
		HubDirectoryURL: hubDirectoryURL,
		DirectoryCount:  totalDirectories,
		FileCount:       totalFiles,
	}, nil
}

// UpdateDirectory 更新目录到 Hub（用于 push）
func (s *HubDirectoryService) UpdateDirectory(ctx context.Context, req *dto.UpdateHubDirectoryRequest, publisherUsername string) (*dto.UpdateHubDirectoryResponse, error) {
	// 1. 获取现有目录
	existingDirectory, err := s.directoryRepo.GetByID(ctx, req.HubDirectoryID)
	if err != nil {
		return nil, fmt.Errorf("获取目录信息失败: %w", err)
	}

	// 2. 验证版本号（新版本必须大于当前版本）
	newVersion := req.Version
	if newVersion == "" {
		return nil, fmt.Errorf("版本号不能为空")
	}
	newVersionNum := extractVersionNum(newVersion)
	if newVersionNum <= existingDirectory.VersionNum {
		return nil, fmt.Errorf("新版本号必须大于当前版本号：当前版本 %s (v%d)，新版本 %s (v%d)",
			existingDirectory.Version, existingDirectory.VersionNum, newVersion, newVersionNum)
	}

	// 3. 验证目录树
	if req.DirectoryTree == nil {
		return nil, fmt.Errorf("目录树不能为空")
	}

	// 4. 序列化目录树（JSON格式）
	directoryTreeJSON, err := json.Marshal(req.DirectoryTree)
	if err != nil {
		return nil, fmt.Errorf("序列化目录树失败: %w", err)
	}

	// 5. 统计信息（递归统计）
	stats := s.countDirectoryTreeStats(req.DirectoryTree)
	totalDirectories := stats.DirectoryCount
	totalFiles := stats.FileCount

	// 6. 更新目录记录（只更新需要更新的字段）
	oldVersion := existingDirectory.Version
	if req.Name != "" {
		existingDirectory.Name = req.Name
	}
	if req.Description != "" {
		existingDirectory.Description = req.Description
	}
	if req.Category != "" {
		existingDirectory.Category = req.Category
	}
	if len(req.Tags) > 0 {
		existingDirectory.Tags = strings.Join(req.Tags, ",")
	}
	if req.ServiceFeePersonal > 0 {
		existingDirectory.ServiceFeePersonal = req.ServiceFeePersonal
	}
	if req.ServiceFeeEnterprise > 0 {
		existingDirectory.ServiceFeeEnterprise = req.ServiceFeeEnterprise
	}
	existingDirectory.Version = newVersion
	existingDirectory.VersionNum = newVersionNum
	existingDirectory.DirectoryTree = string(directoryTreeJSON)
	existingDirectory.DirectoryCount = totalDirectories - 1 // 减去根目录
	existingDirectory.FileCount = totalFiles
	existingDirectory.FunctionCount = stats.FunctionCount
	existingDirectory.PublisherUsername = publisherUsername
	now := time.Now()
	existingDirectory.PublishedAt = &now

	// 7. 获取根目录路径
	rootPath := req.DirectoryTree.Path

	// 8. 更新目录记录
	if err := s.directoryRepo.Update(ctx, existingDirectory); err != nil {
		return nil, fmt.Errorf("更新目录记录失败: %w", err)
	}

	// 9. 将旧快照的 is_current 设为 false
	if err := s.snapshotRepo.SetCurrent(ctx, existingDirectory.ID, 0); err != nil {
		return nil, fmt.Errorf("更新旧快照状态失败: %w", err)
	}

	// 10. 创建新快照
	snapshot := &model.HubSnapshot{
		HubDirectoryID: existingDirectory.ID,
		Version:        newVersion,
		VersionNum:     newVersionNum,
		SnapshotAt:     now,
		DirectoryCount: totalDirectories - 1,
		FileCount:      totalFiles,
		FunctionCount:  stats.FunctionCount,
		SnapshotData:   string(directoryTreeJSON),
		IsCurrent:      true,
	}
	if err := s.snapshotRepo.Create(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("创建快照失败: %w", err)
	}

	// 11. 删除旧的服务树节点
	if err := s.serviceTreeRepo.DeleteByHubDirectoryID(ctx, existingDirectory.ID); err != nil {
		return nil, fmt.Errorf("删除旧服务树节点失败: %w", err)
	}

	// 12. 构建新的服务树节点和文件快照
	if err := s.buildServiceTreeAndSnapshots(ctx, req.DirectoryTree, existingDirectory.ID, snapshot.ID, rootPath, 0, 0); err != nil {
		return nil, fmt.Errorf("构建服务树和快照失败: %w", err)
	}

	// 13. 构建响应
	hubDirectoryURL := fmt.Sprintf("/hub/directories/%d", existingDirectory.ID)
	return &dto.UpdateHubDirectoryResponse{
		HubDirectoryID:  existingDirectory.ID,
		HubDirectoryURL: hubDirectoryURL,
		DirectoryCount:  totalDirectories,
		FileCount:       totalFiles,
		OldVersion:      oldVersion,
		NewVersion:      newVersion,
	}, nil
}

// GetDirectoryList 获取目录列表
func (s *HubDirectoryService) GetDirectoryList(ctx context.Context, page, pageSize int, search, category, publisherUsername string) (*dto.HubDirectoryListResponse, error) {
	directories, total, err := s.directoryRepo.GetList(ctx, page, pageSize, search, category, publisherUsername)
	if err != nil {
		return nil, fmt.Errorf("获取目录列表失败: %w", err)
	}

	// 转换为 DTO
	items := make([]*dto.HubDirectoryDTO, len(directories))
	for i, dir := range directories {
		items[i] = s.toDirectoryDTO(dir)
	}

	return &dto.HubDirectoryListResponse{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

// GetDirectoryDetail 获取目录详情（支持通过 ID 或 full-code-path 查询，支持版本号）
// includeTree: 是否包含目录树结构（如果为 true，DirectoryTree 中的 Files 字段会包含文件内容）
func (s *HubDirectoryService) GetDirectoryDetail(ctx context.Context, hubDirectoryID int64, fullCodePath string, version string, includeTree bool) (*dto.HubDirectoryDetailDTO, error) {
	// 1. 获取目录信息（优先使用 full-code-path，如果为空则使用 ID）
	var directory *model.HubDirectory
	var err error
	if fullCodePath != "" {
		directory, err = s.directoryRepo.GetByFullCodePath(ctx, fullCodePath)
		if err != nil {
			return nil, fmt.Errorf("根据 full-code-path 获取目录信息失败: %w", err)
		}
	} else if hubDirectoryID > 0 {
		directory, err = s.directoryRepo.GetByID(ctx, hubDirectoryID)
		if err != nil {
			return nil, fmt.Errorf("获取目录信息失败: %w", err)
		}
	} else {
		return nil, fmt.Errorf("必须提供 hub_directory_id 或 full_code_path 之一")
	}

	// 2. 获取快照（如果指定了版本号，使用指定版本；否则使用当前版本）
	var snapshot *model.HubSnapshot
	if version != "" {
		snapshot, err = s.snapshotRepo.GetByVersion(ctx, directory.ID, version)
		if err != nil {
			return nil, fmt.Errorf("获取版本快照失败: version=%s, error=%w", version, err)
		}
	} else {
		snapshot, err = s.snapshotRepo.GetCurrent(ctx, directory.ID)
		if err != nil {
			// 如果没有当前快照，尝试使用最新版本
			snapshots, err := s.snapshotRepo.GetByHubDirectoryID(ctx, directory.ID)
			if err == nil && len(snapshots) > 0 {
				snapshot = snapshots[0] // 使用最新版本（已按版本号倒序）
			}
		}
	}

	// 3. 构建详情 DTO（如果指定了版本，使用快照中的版本信息）
	detail := &dto.HubDirectoryDetailDTO{
		HubDirectoryDTO: *s.toDirectoryDTO(directory),
	}

	// 如果指定了版本，更新版本信息
	if snapshot != nil && version != "" {
		detail.Version = snapshot.Version
		detail.VersionNum = snapshot.VersionNum
	}

	// 4. 如果需要目录树结构，从快照和服务树构建
	if includeTree {
		if snapshot != nil {
			// 优先从服务树和文件快照构建（确保数据完整）
			tree, err := s.buildTreeFromServiceTree(ctx, directory.ID, snapshot.ID)
			if err == nil && tree != nil {
				detail.DirectoryTree = tree
			} else {
				// 如果从服务树构建失败，尝试从快照数据中获取（JSON格式，快速）
				if snapshot.SnapshotData != "" {
					var tree dto.DirectoryTreeNode
					if err := json.Unmarshal([]byte(snapshot.SnapshotData), &tree); err == nil {
						detail.DirectoryTree = &tree
					}
				}
			}
		} else {
			// 如果没有快照，尝试从目录的 DirectoryTree 字段获取（兼容旧数据）
			if directory.DirectoryTree != "" {
				var tree dto.DirectoryTreeNode
				if err := json.Unmarshal([]byte(directory.DirectoryTree), &tree); err == nil {
					detail.DirectoryTree = &tree
				}
			}
		}
	}

	// 文件内容已通过 DirectoryTree.Files 字段返回（当 includeTree=true 时）
	// copyFromHub 等接口通过 DirectoryTree.Files 获取文件内容

	return detail, nil
}

// toDirectoryDTO 转换为目录 DTO
func (s *HubDirectoryService) toDirectoryDTO(dir *model.HubDirectory) *dto.HubDirectoryDTO {
	tags := []string{}
	if dir.Tags != "" {
		tags = strings.Split(dir.Tags, ",")
	}

	publishedAt := ""
	if dir.PublishedAt != nil {
		publishedAt = dir.PublishedAt.Format(time.RFC3339)
	}

	return &dto.HubDirectoryDTO{
		ID:                   dir.ID,
		CreatedAt:            dir.CreatedAt.String(),
		UpdatedAt:            dir.UpdatedAt.String(),
		Name:                 dir.Name,
		Description:          dir.Description,
		Category:             dir.Category,
		Tags:                 tags,
		PackagePath:          dir.PackagePath,
		FullCodePath:         dir.FullCodePath,
		SourceUser:           dir.SourceUser,
		SourceApp:            dir.SourceApp,
		SourceDirectoryPath:  dir.SourceDirectoryPath,
		PublisherUsername:    dir.PublisherUsername,
		PublishedAt:          publishedAt,
		ServiceFeePersonal:   dir.ServiceFeePersonal,
		ServiceFeeEnterprise: dir.ServiceFeeEnterprise,
		DownloadCount:        dir.DownloadCount,
		TrialCount:           dir.TrialCount,
		Rating:               dir.Rating,
		Version:              dir.Version,
		VersionNum:           dir.VersionNum,
		DirectoryCount:       dir.DirectoryCount,
		FileCount:            dir.FileCount,
		FunctionCount:        dir.FunctionCount,
	}
}

// countDirectoryTreeStats 递归统计目录树信息（包含函数）
type directoryTreeStats struct {
	DirectoryCount int
	FileCount      int
	FunctionCount  int
}

func (s *HubDirectoryService) countDirectoryTreeStats(node *dto.DirectoryTreeNode) directoryTreeStats {
	// 统计文件数量时，排除 init_.go 文件
	fileCount := 0
	for _, file := range node.Files {
		if file.FileName != "init_" && file.FileName != "init_.go" && !strings.HasSuffix(file.RelativePath, "/init_.go") {
			fileCount++
		}
	}

	stats := directoryTreeStats{
		DirectoryCount: 1,         // 当前目录
		FileCount:      fileCount, // 排除 init_.go 后的文件数量
		FunctionCount:  len(node.Functions),
	}

	// 递归统计子目录
	for _, subdir := range node.Subdirectories {
		subStats := s.countDirectoryTreeStats(subdir)
		stats.DirectoryCount += subStats.DirectoryCount
		stats.FileCount += subStats.FileCount
		stats.FunctionCount += subStats.FunctionCount
	}

	return stats
}

// buildServiceTreeAndSnapshots 递归构建服务树节点和文件快照
func (s *HubDirectoryService) buildServiceTreeAndSnapshots(
	ctx context.Context,
	node *dto.DirectoryTreeNode,
	hubDirectoryID int64,
	snapshotID int64,
	basePath string,
	parentID int64,
	level int,
) error {
	// 1. 创建目录节点（package 类型）
	dirNode := &model.HubServiceTree{
		HubDirectoryID:     hubDirectoryID,
		Name:               node.Name,
		Code:               node.Code, // 直接使用 Code 字段（英文标识）
		ParentID:           parentID,
		Type:               model.HubServiceTreeTypePackage,
		FullCodePath:       node.Path,
		SnapshotVersion:    "", // 将在快照中设置
		SnapshotVersionNum: 0,
		Level:              level,
	}
	if err := s.serviceTreeRepo.Create(ctx, dirNode); err != nil {
		return fmt.Errorf("创建目录节点失败: %w", err)
	}

	// 2. 创建函数节点（function 类型）
	for _, fn := range node.Functions {
		fnNode := &model.HubServiceTree{
			HubDirectoryID:     hubDirectoryID,
			Name:               fn.Name,
			Code:               fn.Code,
			ParentID:           dirNode.ID,
			Type:               model.HubServiceTreeTypeFunction,
			Description:        fn.Description,
			FullCodePath:       fn.FullCodePath,
			RefID:              fn.RefID,
			Version:            fn.Version,
			VersionNum:         fn.VersionNum,
			TemplateType:       fn.TemplateType,
			Tags:               strings.Join(fn.Tags, ","),
			SnapshotVersion:    "", // 将在快照中设置
			SnapshotVersionNum: 0,
			Level:              level + 1,
		}
		if err := s.serviceTreeRepo.Create(ctx, fnNode); err != nil {
			return fmt.Errorf("创建函数节点失败: %w", err)
		}
	}

	// 3. 创建文件快照（init_.go 已在 runtime 层过滤，这里不需要再过滤）
	logger.Infof(ctx, "[buildServiceTreeAndSnapshots] 目录节点 %s (ID=%d) 有 %d 个文件", node.Name, dirNode.ID, len(node.Files))
	for i, fileInfo := range node.Files {
		logger.Infof(ctx, "[buildServiceTreeAndSnapshots] 处理文件[%d]: FileName=%s, RelativePath=%s, FileType=%s, Content长度=%d",
			i+1, fileInfo.FileName, fileInfo.RelativePath, fileInfo.FileType, len(fileInfo.Content))
		fileSnapshot := &model.HubFileSnapshot{
			HubSnapshotID:    snapshotID,
			HubServiceTreeID: dirNode.ID,
			FileName:         fileInfo.FileName,
			RelativePath:     fileInfo.RelativePath,
			FileType:         fileInfo.FileType,
			Content:          fileInfo.Content,
			FileVersion:      fileInfo.FileVersion,
			FileVersionNum:   extractVersionNum(fileInfo.FileVersion),
			FileSize:         len(fileInfo.Content),
			ContentHash:      utils.HashString(fileInfo.Content),
		}
		if err := s.fileSnapshotRepo.Create(ctx, fileSnapshot); err != nil {
			return fmt.Errorf("创建文件快照失败: %w", err)
		}
		logger.Infof(ctx, "[buildServiceTreeAndSnapshots] 文件快照创建成功: HubServiceTreeID=%d, FileName=%s", dirNode.ID, fileInfo.FileName)
	}

	// 4. 递归处理子目录
	for _, subdir := range node.Subdirectories {
		if err := s.buildServiceTreeAndSnapshots(ctx, subdir, hubDirectoryID, snapshotID, basePath, dirNode.ID, level+1); err != nil {
			return err
		}
	}

	return nil
}

// extractPackagePath 从完整路径提取 package 路径
func extractPackagePath(fullPath, user, app string) string {
	// 格式：/user/app/package1/package2
	// 返回：package1/package2
	prefix := fmt.Sprintf("/%s/%s", user, app)
	if strings.HasPrefix(fullPath, prefix) {
		return strings.TrimPrefix(fullPath, prefix+"/")
	}
	return fullPath
}

// extractVersionNum 从版本号字符串提取数字部分
func extractVersionNum(version string) int {
	// 格式：v1, v2, v101
	version = strings.TrimPrefix(version, "v")
	num, err := strconv.Atoi(version)
	if err != nil {
		return 1 // 默认版本号
	}
	return num
}

// buildTreeFromServiceTree 从服务树和文件快照构建目录树
func (s *HubDirectoryService) buildTreeFromServiceTree(ctx context.Context, hubDirectoryID int64, snapshotID int64) (*dto.DirectoryTreeNode, error) {
	// 1. 获取所有服务树节点
	allNodes, err := s.serviceTreeRepo.GetByHubDirectoryID(ctx, hubDirectoryID)
	if err != nil {
		return nil, fmt.Errorf("获取服务树节点失败: %w", err)
	}

	// 2. 获取所有文件快照
	fileSnapshots, err := s.fileSnapshotRepo.GetBySnapshotID(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("获取文件快照失败: %w", err)
	}

	// 3. 构建节点映射
	nodeMap := make(map[int64]*model.HubServiceTree)
	fileMap := make(map[int64][]*model.HubFileSnapshot) // HubServiceTreeID -> []FileSnapshot

	logger.Infof(ctx, "[buildTreeFromServiceTree] 开始构建树: 节点数量=%d, 文件快照数量=%d", len(allNodes), len(fileSnapshots))

	for _, node := range allNodes {
		nodeMap[node.ID] = node
		logger.Infof(ctx, "[buildTreeFromServiceTree] 节点: ID=%d, Name=%s, Code=%s, Type=%s, FullCodePath=%s",
			node.ID, node.Name, node.Code, node.Type, node.FullCodePath)
	}

	for _, file := range fileSnapshots {
		logger.Infof(ctx, "[buildTreeFromServiceTree] 文件快照: HubServiceTreeID=%d, FileName=%s, RelativePath=%s, Content长度=%d",
			file.HubServiceTreeID, file.FileName, file.RelativePath, len(file.Content))
		fileMap[file.HubServiceTreeID] = append(fileMap[file.HubServiceTreeID], file)
	}

	logger.Infof(ctx, "[buildTreeFromServiceTree] 文件映射构建完成: fileMap大小=%d", len(fileMap))

	// 4. 找到根节点（ParentID = 0 的 package 节点）
	var rootNode *model.HubServiceTree
	for _, node := range allNodes {
		if node.ParentID == 0 && node.Type == model.HubServiceTreeTypePackage {
			rootNode = node
			break
		}
	}

	if rootNode == nil {
		return nil, fmt.Errorf("未找到根节点")
	}

	// 5. 递归构建树
	return s.buildTreeNode(rootNode, allNodes, fileMap, nodeMap), nil
}

// buildTreeNode 递归构建树节点
func (s *HubDirectoryService) buildTreeNode(
	node *model.HubServiceTree,
	allNodes []*model.HubServiceTree,
	fileMap map[int64][]*model.HubFileSnapshot,
	nodeMap map[int64]*model.HubServiceTree,
) *dto.DirectoryTreeNode {

	// 构建函数列表
	functions := make([]*dto.HubFunctionInfo, 0)
	for _, n := range allNodes {
		if n.ParentID == node.ID && n.Type == model.HubServiceTreeTypeFunction {
			tags := []string{}
			if n.Tags != "" {
				tags = strings.Split(n.Tags, ",")
			}
			functions = append(functions, &dto.HubFunctionInfo{
				ID:           n.ID,
				Name:         n.Name,
				Code:         n.Code,
				FullCodePath: n.FullCodePath,
				Description:  n.Description,
				TemplateType: n.TemplateType,
				Tags:         tags,
				RefID:        n.RefID,
				Version:      n.Version,
				VersionNum:   n.VersionNum,
			})
		}
	}

	// 构建子目录列表
	subdirectories := make([]*dto.DirectoryTreeNode, 0)
	for _, n := range allNodes {
		if n.ParentID == node.ID && n.Type == model.HubServiceTreeTypePackage {
			subdir := s.buildTreeNode(n, allNodes, fileMap, nodeMap)
			subdirectories = append(subdirectories, subdir)
		}
	}

	return &dto.DirectoryTreeNode{
		Type:           "package", // DirectoryTreeNode 始终是 package 类型
		Name:           node.Name, // 目录名称（中文显示名称）
		Code:           node.Code, // 目录代码（英文标识）
		Path:           node.FullCodePath,
		Files:          nil, // ⭐ 设为 nil，使用 omitempty 后 JSON 序列化时不会包含 files 字段
		Functions:      functions,
		Subdirectories: subdirectories,
	}
}
