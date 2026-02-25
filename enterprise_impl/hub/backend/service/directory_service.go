package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/hub/backend/dto"
	"github.com/ai-agent-os/hub/backend/model"
	"github.com/ai-agent-os/hub/backend/repository"
)

// HubDirectoryService Hub 目录服务（两表方案：仅目录 + 快照；星星单独表）
type HubDirectoryService struct {
	directoryRepo *repository.HubDirectoryRepository
	snapshotRepo  *repository.HubSnapshotRepository
	starRepo      *repository.HubDirectoryStarRepository
}

// NewHubDirectoryService 创建 Hub 目录服务（依赖注入）
func NewHubDirectoryService(
	directoryRepo *repository.HubDirectoryRepository,
	snapshotRepo *repository.HubSnapshotRepository,
	starRepo *repository.HubDirectoryStarRepository,
) *HubDirectoryService {
	return &HubDirectoryService{
		directoryRepo: directoryRepo,
		snapshotRepo:  snapshotRepo,
		starRepo:      starRepo,
	}
}

// PublishDirectory 发布目录到 Hub（发布者从 ctx 获取，与 app-server 一致）
func (s *HubDirectoryService) PublishDirectory(ctx context.Context, req *dto.PublishHubDirectoryRequest) (*dto.PublishHubDirectoryResponse, error) {
	publisherUsername := contextx.GetRequestUser(ctx)
	if publisherUsername == "" {
		return nil, fmt.Errorf("未获取到发布者信息，请确认已登录")
	}
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
		Status:                model.HubDirectoryStatusActive,
		Name:                  req.Name,
		Description:           req.Description,
		Category:              req.Category,
		Tags:                  strings.Join(req.Tags, ","),
		PackagePath:           packagePath,
		FullCodePath:          rootPath,
		ParentDirID:           0, // 根目录
		SourceUser:            req.SourceUser,
		SourceApp:             req.SourceApp,
		SourceDirectoryPath:   req.SourceDirectoryPath,
		PublisherUsername:     publisherUsername,
		PublishedAt:           &now,
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

	// 8. 拆成三份：结构(展示)、文件(复制)、函数定义(预览)
	treeJSON, filesJSON, defsJSON, err := splitDirectoryTreeIntoSnapshotParts(req.DirectoryTree)
	if err != nil {
		return nil, fmt.Errorf("拆分快照三字段失败: %w", err)
	}
	// 9. 创建快照（三字段 + 兼容旧端的 SnapshotData + 该版本详情）
	snapshot := &model.HubSnapshot{
		HubDirectoryID:       directory.ID,
		Version:              version,
		VersionNum:           versionNum,
		SnapshotAt:           now,
		DirectoryCount:       totalDirectories - 1,
		FileCount:            totalFiles,
		FunctionCount:        totalFunctions,
		Name:                 req.Name,
		DetailDescription:    req.Description,
		Category:             req.Category,
		Tags:                 strings.Join(req.Tags, ","),
		ServiceFeePersonal:   req.ServiceFeePersonal,
		ServiceFeeEnterprise: req.ServiceFeeEnterprise,
		PublisherUsername:    publisherUsername,
		SnapshotData:         string(directoryTreeJSON),
		SnapshotTree:         treeJSON,
		SnapshotFiles:        filesJSON,
		SnapshotFunctionDefs:  defsJSON,
		IsCurrent:            true,
	}
	if err := s.snapshotRepo.Create(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("创建快照失败: %w", err)
	}

	// 10. 构建响应（只返回 hub_full_code_path，前端用此拼详情 URL）
	return &dto.PublishHubDirectoryResponse{
		HubFullCodePath: directory.FullCodePath,
		DirectoryCount:  totalDirectories,
		FileCount:       totalFiles,
	}, nil
}

// UpdateDirectory 更新目录到 Hub（用于 push；发布者从 ctx 获取）
func (s *HubDirectoryService) UpdateDirectory(ctx context.Context, req *dto.UpdateHubDirectoryRequest) (*dto.UpdateHubDirectoryResponse, error) {
	publisherUsername := contextx.GetRequestUser(ctx)
	if publisherUsername == "" {
		return nil, fmt.Errorf("未获取到发布者信息，请确认已登录")
	}
	// 1. 获取现有目录
	existingDirectory, err := s.directoryRepo.GetByID(ctx, req.HubDirectoryID)
	if err != nil {
		return nil, fmt.Errorf("获取目录信息失败: %w", err)
	}

	// 2. 版本号：未传或未大于当前版本时由后端自动递增为 v{N+1}
	newVersion := req.Version
	newVersionNum := extractVersionNum(newVersion)
	if newVersion == "" || newVersionNum <= existingDirectory.VersionNum {
		newVersion = fmt.Sprintf("v%d", existingDirectory.VersionNum+1)
		newVersionNum = existingDirectory.VersionNum + 1
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

	// 7. 更新目录记录
	if err := s.directoryRepo.Update(ctx, existingDirectory); err != nil {
		return nil, fmt.Errorf("更新目录记录失败: %w", err)
	}

	// 9. 将旧快照的 is_current 设为 false
	if err := s.snapshotRepo.SetCurrent(ctx, existingDirectory.ID, 0); err != nil {
		return nil, fmt.Errorf("更新旧快照状态失败: %w", err)
	}

	// 10. 拆成三份并创建新快照（含本版本更新说明）；已改为两表方案，不再写 hub_service_tree / hub_file_snapshots
	treeJSON, filesJSON, defsJSON, err := splitDirectoryTreeIntoSnapshotParts(req.DirectoryTree)
	if err != nil {
		return nil, fmt.Errorf("拆分快照三字段失败: %w", err)
	}
	snapshot := &model.HubSnapshot{
		HubDirectoryID:       existingDirectory.ID,
		Version:              newVersion,
		VersionNum:           newVersionNum,
		SnapshotAt:            now,
		DirectoryCount:       totalDirectories - 1,
		FileCount:            totalFiles,
		FunctionCount:        stats.FunctionCount,
		Name:                 existingDirectory.Name,
		DetailDescription:    existingDirectory.Description,
		Category:             existingDirectory.Category,
		Tags:                 existingDirectory.Tags,
		ServiceFeePersonal:   existingDirectory.ServiceFeePersonal,
		ServiceFeeEnterprise: existingDirectory.ServiceFeeEnterprise,
		PublisherUsername:    publisherUsername,
		SnapshotData:         string(directoryTreeJSON),
		SnapshotTree:         treeJSON,
		SnapshotFiles:        filesJSON,
		SnapshotFunctionDefs:  defsJSON,
		IsCurrent:            true,
		Description:          req.UpdateDescription, // 本版本更新说明，便于在 Hub 查看历史时看到「这个版本加了什么」
	}
	if err := s.snapshotRepo.Create(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("创建快照失败: %w", err)
	}

	// 11. 构建响应（只返回 hub_full_code_path，前端用此拼详情 URL）
	return &dto.UpdateHubDirectoryResponse{
		HubFullCodePath: existingDirectory.FullCodePath,
		DirectoryCount:  totalDirectories,
		FileCount:       totalFiles,
		OldVersion:      oldVersion,
		NewVersion:      newVersion,
	}, nil
}

// GetDirectoryList 获取目录列表；host 用于生成每条记录的 copy_url
// feeType: 空=全部，free=免费，paid=收费；orderBy: 空或 latest=最新，hot=热门
func (s *HubDirectoryService) GetDirectoryList(ctx context.Context, page, pageSize int, search, category, publisherUsername, feeType, orderBy string, host string) (*dto.HubDirectoryListResponse, error) {
	directories, total, err := s.directoryRepo.GetList(ctx, page, pageSize, search, category, publisherUsername, feeType, orderBy)
	if err != nil {
		return nil, fmt.Errorf("获取目录列表失败: %w", err)
	}

	// 转换为 DTO（含 copy_url；star_count 取自目录表冗余字段）
	items := make([]*dto.HubDirectoryDTO, len(directories))
	for i, dir := range directories {
		items[i] = s.toDirectoryDTO(dir, host)
	}

	return &dto.HubDirectoryListResponse{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

// GetDirectoryDetail 获取目录详情（支持通过 ID 或 full-code-path 查询，支持版本号）
// host 用于生成 copy_url；includeTree: 是否包含目录树结构；当前用户从 ctx 获取（与 app-server 一致，网关已带 X-Request-User），有则填充 has_starred
func (s *HubDirectoryService) GetDirectoryDetail(ctx context.Context, hubDirectoryID int64, fullCodePath string, version string, includeTree bool, host string) (*dto.HubDirectoryDetailDTO, error) {
	username := contextx.GetRequestUser(ctx)
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

	// 3. 构建详情 DTO（star_count 取自目录表；has_starred 需查 star 表）
	detail := &dto.HubDirectoryDetailDTO{
		HubDirectoryDTO: *s.toDirectoryDTO(directory, host),
	}
	if s.starRepo != nil && username != "" {
		if ok, _ := s.starRepo.HasStarred(ctx, directory.ID, username); ok {
			detail.HasStarred = true
		}
	}
	if snapshot != nil && version != "" {
		detail.Version = snapshot.Version
		detail.VersionNum = snapshot.VersionNum
		// 该版本的详情从快照取（切换版本时展示当时 name/description 等）
		if snapshot.Name != "" {
			detail.Name = snapshot.Name
		}
		if snapshot.DetailDescription != "" {
			detail.Description = snapshot.DetailDescription
		}
		if snapshot.Category != "" {
			detail.Category = snapshot.Category
		}
		if snapshot.Tags != "" {
			detail.Tags = strings.Split(snapshot.Tags, ",")
		}
		// 该版本时的服务费（旧快照无此字段时为 0，展示 0 即可）
		detail.ServiceFeePersonal = snapshot.ServiceFeePersonal
		detail.ServiceFeeEnterprise = snapshot.ServiceFeeEnterprise
		if snapshot.PublisherUsername != "" {
			detail.PublisherUsername = snapshot.PublisherUsername
		}
	}
	// 当前查看版本的更新说明（推送时填的「本版本更新说明」）
	if snapshot != nil && snapshot.Description != "" {
		detail.VersionDescription = snapshot.Description // Description 存的是本版本更新说明
	}

	// 4. 目录树结构：优先用快照三字段（结构+文件合并），否则 SnapshotData，再兜底服务树/目录 DirectoryTree
	if includeTree {
		if snapshot != nil {
			// 优先：SnapshotTree + SnapshotFiles 合并（展示用结构 + 复制用文件内容）
			if snapshot.SnapshotTree != "" {
				tree, mergeErr := mergeSnapshotPartsIntoTree(snapshot.SnapshotTree, snapshot.SnapshotFiles)
				if mergeErr == nil && tree != nil {
					detail.DirectoryTree = tree
				}
			}
			if detail.DirectoryTree == nil && snapshot.SnapshotData != "" {
				var tree dto.DirectoryTreeNode
				if err := json.Unmarshal([]byte(snapshot.SnapshotData), &tree); err == nil {
					detail.DirectoryTree = &tree
				}
			}
			// 已改为两表方案，不再从 hub_service_tree / hub_file_snapshots 兜底
		} else {
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

// ListDirectoryVersions 获取目录的版本列表（用于详情页右侧展示历史版本）
func (s *HubDirectoryService) ListDirectoryVersions(ctx context.Context, hubDirectoryID int64) (*dto.GetHubDirectoryVersionsResponse, error) {
	if hubDirectoryID <= 0 {
		return nil, fmt.Errorf("hub_directory_id 必须大于 0")
	}
	snapshots, err := s.snapshotRepo.GetByHubDirectoryID(ctx, hubDirectoryID)
	if err != nil {
		return nil, fmt.Errorf("获取版本列表失败: %w", err)
	}
	items := make([]*dto.HubDirectoryVersionItem, 0, len(snapshots))
	for _, sn := range snapshots {
		items = append(items, &dto.HubDirectoryVersionItem{
			Version:           sn.Version,
			VersionNum:        sn.VersionNum,
			SnapshotAt:        sn.SnapshotAt.Format(time.RFC3339),
			IsCurrent:         sn.IsCurrent,
			Description:       sn.Description,
			PublisherUsername: sn.PublisherUsername,
		})
	}
	return &dto.GetHubDirectoryVersionsResponse{Items: items}, nil
}

// toDirectoryDTO 转换为目录 DTO
// toDirectoryDTO 将模型转为 DTO；host 用于生成 copy_url（格式 hub://host/full_code_path@version），为空则不填 copy_url
func (s *HubDirectoryService) toDirectoryDTO(dir *model.HubDirectory, host string) *dto.HubDirectoryDTO {
	tags := []string{}
	if dir.Tags != "" {
		tags = strings.Split(dir.Tags, ",")
	}

	publishedAt := ""
	if dir.PublishedAt != nil {
		publishedAt = dir.PublishedAt.Format(time.RFC3339)
	}

	copyURL := ""
	if host != "" && dir.FullCodePath != "" {
		path := strings.TrimPrefix(dir.FullCodePath, "/")
		copyURL = fmt.Sprintf("hub://%s/%s@%s", host, path, dir.Version)
	}

	return &dto.HubDirectoryDTO{
		ID:                   dir.ID,
		CreatedAt:            dir.CreatedAt,
		UpdatedAt:            dir.UpdatedAt,
		Status:               dir.Status,
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
		CopyURL:              copyURL,
		StarCount:            dir.StarCount,
	}
}

// splitDirectoryTreeIntoSnapshotParts 从目录树拆出三份：结构(展示)、文件(复制)、函数定义(预览)
func splitDirectoryTreeIntoSnapshotParts(tree *dto.DirectoryTreeNode) (treeJSON, filesJSON, defsJSON string, err error) {
	if tree == nil {
		return "", "", "", nil
	}
	// 1. 结构：深拷贝树，去掉文件 content，用于展示
	treeOnly := cloneTreeStripFileContent(tree)
	treeBytes, err := json.Marshal(treeOnly)
	if err != nil {
		return "", "", "", fmt.Errorf("序列化快照结构失败: %w", err)
	}
	// 2. 文件：平铺列表，用于复制
	filesList := flattenFilesFromTree(tree)
	filesBytes, err := json.Marshal(filesList)
	if err != nil {
		return "", "", "", fmt.Errorf("序列化快照文件失败: %w", err)
	}
	// 3. 函数定义：平铺列表，用于预览
	defsList := collectFunctionDefsFromTree(tree)
	defsBytes, err := json.Marshal(defsList)
	if err != nil {
		return "", "", "", fmt.Errorf("序列化快照函数定义失败: %w", err)
	}
	return string(treeBytes), string(filesBytes), string(defsBytes), nil
}

func cloneTreeStripFileContent(node *dto.DirectoryTreeNode) *dto.DirectoryTreeNode {
	if node == nil {
		return nil
	}
	files := make([]*dto.FileSnapshotInfo, 0, len(node.Files))
	for _, f := range node.Files {
		files = append(files, &dto.FileSnapshotInfo{
			FileName:     f.FileName,
			RelativePath: f.RelativePath,
			Content:      "", // 展示用不存 content
			FileType:     f.FileType,
			FileVersion:  f.FileVersion,
		})
	}
	subdirs := make([]*dto.DirectoryTreeNode, 0, len(node.Subdirectories))
	for _, sub := range node.Subdirectories {
		subdirs = append(subdirs, cloneTreeStripFileContent(sub))
	}
	return &dto.DirectoryTreeNode{
		Type:           node.Type,
		Name:           node.Name,
		Code:           node.Code,
		Path:           node.Path,
		Files:          files,
		Functions:      node.Functions, // 展示用保留函数列表（名称/路径等），详情在 SnapshotFunctionDefs
		Subdirectories: subdirs,
	}
}

func flattenFilesFromTree(node *dto.DirectoryTreeNode) []*dto.SnapshotFileEntry {
	out := make([]*dto.SnapshotFileEntry, 0)
	var walk func(*dto.DirectoryTreeNode)
	walk = func(n *dto.DirectoryTreeNode) {
		if n == nil {
			return
		}
		for _, f := range n.Files {
			if f == nil {
				continue
			}
			out = append(out, &dto.SnapshotFileEntry{
				RelativePath: f.RelativePath,
				Content:      f.Content,
				FileType:     f.FileType,
			})
		}
		for _, sub := range n.Subdirectories {
			walk(sub)
		}
	}
	walk(node)
	return out
}

func collectFunctionDefsFromTree(node *dto.DirectoryTreeNode) []*dto.HubFunctionInfo {
	out := make([]*dto.HubFunctionInfo, 0)
	var walk func(*dto.DirectoryTreeNode)
	walk = func(n *dto.DirectoryTreeNode) {
		if n == nil {
			return
		}
		for _, fn := range n.Functions {
			if fn != nil {
				out = append(out, fn)
			}
		}
		for _, sub := range n.Subdirectories {
			walk(sub)
		}
	}
	walk(node)
	return out
}

// mergeSnapshotPartsIntoTree 用快照的「结构 + 文件」合并成带内容的目录树（用于详情/复制）
func mergeSnapshotPartsIntoTree(snapshotTreeJSON, snapshotFilesJSON string) (*dto.DirectoryTreeNode, error) {
	if snapshotTreeJSON == "" {
		return nil, nil
	}
	var tree dto.DirectoryTreeNode
	if err := json.Unmarshal([]byte(snapshotTreeJSON), &tree); err != nil {
		return nil, fmt.Errorf("解析快照结构失败: %w", err)
	}
	if snapshotFilesJSON != "" {
		var filesList []*dto.SnapshotFileEntry
		if err := json.Unmarshal([]byte(snapshotFilesJSON), &filesList); err != nil {
			return nil, fmt.Errorf("解析快照文件失败: %w", err)
		}
		contentByPath := make(map[string]string)
		for _, e := range filesList {
			if e != nil {
				contentByPath[e.RelativePath] = e.Content
			}
		}
		fillTreeFileContent(&tree, contentByPath)
	}
	return &tree, nil
}

func fillTreeFileContent(node *dto.DirectoryTreeNode, contentByPath map[string]string) {
	if node == nil {
		return
	}
	for _, f := range node.Files {
		if f != nil && f.RelativePath != "" {
			if c, ok := contentByPath[f.RelativePath]; ok {
				f.Content = c
			}
		}
	}
	for _, sub := range node.Subdirectories {
		fillTreeFileContent(sub, contentByPath)
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

// IncrementDownloadCount 复制/下载时增加该目录的下载次数（按 full_code_path 定位）
func (s *HubDirectoryService) IncrementDownloadCount(ctx context.Context, fullCodePath string) error {
	return s.directoryRepo.IncrementDownloadCount(ctx, fullCodePath)
}

// Star 为目录加星（类似 GitHub star）
func (s *HubDirectoryService) Star(ctx context.Context, hubDirectoryID int64, username string) error {
	if s.starRepo == nil {
		return fmt.Errorf("star 功能未启用")
	}
	created, err := s.starRepo.Star(ctx, hubDirectoryID, username)
	if err != nil {
		return err
	}
	if created {
		return s.directoryRepo.IncrementStarCount(ctx, hubDirectoryID)
	}
	return nil
}

// Unstar 取消星星
func (s *HubDirectoryService) Unstar(ctx context.Context, hubDirectoryID int64, username string) error {
	if s.starRepo == nil {
		return fmt.Errorf("star 功能未启用")
	}
	if err := s.starRepo.Unstar(ctx, hubDirectoryID, username); err != nil {
		return err
	}
	return s.directoryRepo.DecrementStarCount(ctx, hubDirectoryID)
}

// DeleteDirectory 删除应用（软删除：只改状态为 deleted，数据保留，通过链接仍可访问；仅发布者可操作）
func (s *HubDirectoryService) DeleteDirectory(ctx context.Context, hubDirectoryID int64, username string) error {
	if hubDirectoryID <= 0 {
		return fmt.Errorf("目录 ID 无效")
	}
	if username == "" {
		return fmt.Errorf("请先登录")
	}
	dir, err := s.directoryRepo.GetByID(ctx, hubDirectoryID)
	if err != nil || dir == nil {
		return fmt.Errorf("目录不存在")
	}
	if dir.PublisherUsername != username {
		return fmt.Errorf("仅发布者可下架该应用")
	}
	if dir.Status == model.HubDirectoryStatusDeleted {
		return nil // 已下架，幂等
	}
	dir.Status = model.HubDirectoryStatusDeleted
	return s.directoryRepo.Update(ctx, dir)
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

