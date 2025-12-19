package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// extractVersionNum 从版本号字符串中提取数字部分（如 "v1" -> 1, "v20" -> 20）
func extractVersionNumForServiceTree(version string) int {
	if version == "" {
		return 0
	}
	// 去掉 "v" 前缀
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	// 提取数字部分
	num, err := strconv.Atoi(version)
	if err != nil {
		return 0
	}
	return num
}

type ServiceTreeService struct {
	serviceTreeRepo    *repository.ServiceTreeRepository
	appRepo            *repository.AppRepository
	appRuntime         *AppRuntime
	fileSnapshotRepo   *repository.FileSnapshotRepository
	appService         *AppService
	functionGenService *FunctionGenService // 用于异步处理和回调
}

// NewServiceTreeService 创建服务目录服务
func NewServiceTreeService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	appRuntime *AppRuntime,
	fileSnapshotRepo *repository.FileSnapshotRepository,
	appService *AppService,
	functionGenService *FunctionGenService,
) *ServiceTreeService {
	return &ServiceTreeService{
		serviceTreeRepo:    serviceTreeRepo,
		appRepo:            appRepo,
		appRuntime:         appRuntime,
		fileSnapshotRepo:   fileSnapshotRepo,
		appService:         appService,
		functionGenService: functionGenService,
	}
}

// CreateServiceTree 创建服务目录
func (s *ServiceTreeService) CreateServiceTree(ctx context.Context, req *dto.CreateServiceTreeReq) (*dto.CreateServiceTreeResp, error) {
	// 获取应用信息
	app, err := s.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	var parentTree *model.ServiceTree

	if req.ParentID != 0 {
		// 检查名称是否已存在
		parentTree, err = s.serviceTreeRepo.GetByID(req.ParentID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to check name exists: %s", err)
		}
	}

	fullCodePath := fmt.Sprintf("/%s/%s/%s", app.User, app.Code, req.Code)
	if parentTree != nil {
		fullCodePath = parentTree.FullCodePath + "/" + req.Code
	}
	exists, err := s.serviceTreeRepo.CheckNameExists(req.ParentID, req.Code, app.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check name exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("directory %s already exists", req.Code)
	}

	// 提取当前版本号数字
	currentVersionNum := extractVersionNumForServiceTree(app.Version)

	// 创建服务目录记录
	serviceTree := &model.ServiceTree{
		Name:             req.Name,
		Code:             req.Code,
		ParentID:         req.ParentID,
		Type:             model.ServiceTreeTypePackage,
		Description:      req.Description,
		Tags:             req.Tags,
		AppID:            app.ID,
		FullCodePath:     fullCodePath,
		AddVersionNum:    currentVersionNum, // 设置添加版本号
		UpdateVersionNum: 0,                 // 新增节点，更新版本号为0
	}

	// 保存到数据库
	if err := s.serviceTreeRepo.CreateServiceTreeWithParentPath(serviceTree, ""); err != nil {
		return nil, fmt.Errorf("failed to create service tree: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Created service tree: %s/%s/%s", req.User, req.App, req.Code)

	// 发送NATS消息给app-runtime创建目录结构
	if err := s.sendCreateServiceTreeMessage(ctx, req.User, req.App, serviceTree); err != nil {
		logger.Warnf(ctx, "[ServiceTreeService] Failed to send NATS message: %v", err)
		// 不返回错误，因为数据库记录已创建成功
	}

	// 返回响应
	resp := &dto.CreateServiceTreeResp{
		ID:           serviceTree.ID,
		Name:         serviceTree.Name,
		Code:         serviceTree.Code,
		ParentID:     serviceTree.ParentID,
		Type:         serviceTree.Type,
		Description:  serviceTree.Description,
		Tags:         serviceTree.Tags,
		AppID:        serviceTree.AppID,
		FullCodePath: serviceTree.FullCodePath,
		Version:      serviceTree.Version,
		VersionNum:   serviceTree.VersionNum,
		Status:       "created",
	}

	return resp, nil
}

// GetServiceTree 获取服务目录
func (s *ServiceTreeService) GetServiceTree(ctx context.Context, user, app string, nodeType string) ([]*dto.GetServiceTreeResp, error) {
	// 获取应用信息
	appModel, err := s.appRepo.GetAppByUserName(user, app)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	// 构建树形结构（如果指定了类型，则只返回该类型的节点）
	var trees []*model.ServiceTree
	if nodeType != "" {
		trees, err = s.serviceTreeRepo.BuildServiceTreeByType(appModel.ID, nodeType)
	} else {
		trees, err = s.serviceTreeRepo.BuildServiceTree(appModel.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build service tree: %w", err)
	}

	// 转换为响应格式
	var resp []*dto.GetServiceTreeResp
	for _, tree := range trees {
		resp = append(resp, s.convertToGetServiceTreeResp(tree))
	}

	return resp, nil
}

// UpdateServiceTree 更新服务目录
func (s *ServiceTreeService) UpdateServiceTreeMetadata(ctx context.Context, req *dto.UpdateServiceTreeMetadataReq) error {
	// 获取服务目录
	serviceTree, err := s.serviceTreeRepo.GetServiceTreeByID(req.ID)
	if err != nil {
		return fmt.Errorf("failed to get service tree: %w", err)
	}

	// 更新字段
	if req.Name != "" {
		serviceTree.Name = req.Name
	}
	if req.Code != "" {
		// 检查新名称是否已存在
		exists, err := s.serviceTreeRepo.CheckNameExists(serviceTree.ParentID, req.Code, serviceTree.AppID)
		if err != nil {
			return fmt.Errorf("failed to check name exists: %w", err)
		}
		if exists {
			return fmt.Errorf("service tree name '%s' already exists in this parent directory", req.Code)
		}
		serviceTree.Code = req.Code
	}
	if req.Description != "" {
		serviceTree.Description = req.Description
	}
	if req.Tags != "" {
		serviceTree.Tags = req.Tags
	}

	// 保存更新
	if err := s.serviceTreeRepo.UpdateServiceTree(serviceTree); err != nil {
		return fmt.Errorf("failed to update service tree: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Updated service tree: ID=%d", req.ID)
	return nil
}

// DeleteServiceTree 删除服务目录
func (s *ServiceTreeService) DeleteServiceTree(ctx context.Context, id int64) error {
	// 获取服务目录信息（用于日志）
	serviceTree, err := s.serviceTreeRepo.GetServiceTreeByID(id)
	if err != nil {
		return fmt.Errorf("failed to get service tree: %w", err)
	}

	// 删除服务目录（级联删除子目录）
	if err := s.serviceTreeRepo.DeleteServiceTree(id); err != nil {
		return fmt.Errorf("failed to delete service tree: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Deleted service tree: ID=%d, Code=%s", id, serviceTree.Code)
	return nil
}

// convertToGetServiceTreeResp 转换为响应格式
func (s *ServiceTreeService) convertToGetServiceTreeResp(tree *model.ServiceTree) *dto.GetServiceTreeResp {
	resp := &dto.GetServiceTreeResp{
		ID:            tree.ID,
		Name:          tree.Name,
		Code:          tree.Code,
		ParentID:      tree.ParentID,
		RefID:         tree.RefID,
		Type:          tree.Type,
		FullGroupCode: tree.FullGroupCode,
		GroupName:     tree.GroupName,
		Description:   tree.Description,
		Tags:          tree.Tags,
		AppID:         tree.AppID,
		FullCodePath:  tree.FullCodePath,
		TemplateType:  tree.TemplateType,
		Version:       tree.Version,
		VersionNum:    tree.VersionNum,
	}

	// 递归处理子节点
	if len(tree.Children) > 0 {
		for _, child := range tree.Children {
			resp.Children = append(resp.Children, s.convertToGetServiceTreeResp(child))
		}
	}

	return resp
}

// sendCreateServiceTreeMessage 发送创建服务目录的NATS消息
func (s *ServiceTreeService) sendCreateServiceTreeMessage(ctx context.Context, user, app string, serviceTree *model.ServiceTree) error {
	// 获取应用信息以获取HostID
	appModel, err := s.appRepo.GetAppByUserName(user, app)
	if err != nil {
		return fmt.Errorf("failed to get app info: %w", err)
	}

	// 构建消息
	req := dto.CreateServiceTreeRuntimeReq{
		User: user,
		App:  app,
		ServiceTree: &dto.ServiceTreeRuntimeData{
			ID:           serviceTree.ID,
			Name:         serviceTree.Name,
			Code:         serviceTree.Code,
			ParentID:     serviceTree.ParentID,
			Type:         serviceTree.Type,
			Description:  serviceTree.Description,
			Tags:         serviceTree.Tags,
			AppID:        serviceTree.AppID,
			FullCodePath: serviceTree.FullCodePath,
		},
	}

	// 调用 app-runtime 创建服务目录，使用应用所属的 HostID
	_, err = s.appRuntime.CreateServiceTree(ctx, appModel.HostID, &req)
	if err != nil {
		return fmt.Errorf("failed to create service tree via app-runtime: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Service tree created successfully via app-runtime: %s/%s/%s",
		user, app, serviceTree.Code)

	return nil
}

// GetDirectorySnapshotsRecursively 递归获取目录及其所有子目录的文件快照
// GetDirectorySnapshotsRecursively 递归获取目录及其所有子目录的当前版本文件快照
// 优化：使用 ServiceTreeID 和 IsCurrent 字段，性能更好
// 返回：map[目录路径][]文件快照
func (s *ServiceTreeService) GetDirectorySnapshotsRecursively(ctx context.Context, appID int64, rootDirectoryPath string) (map[string][]*model.FileSnapshot, error) {
	// 1. 获取根目录节点（ServiceTree）
	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(rootDirectoryPath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warnf(ctx, "[ServiceTreeService] 根目录节点不存在: path=%s", rootDirectoryPath)
			return make(map[string][]*model.FileSnapshot), nil
		}
		return nil, fmt.Errorf("获取根目录节点失败: %w", err)
	}

	// 2. 递归查询所有子目录节点（包括根目录）
	descendants, err := s.serviceTreeRepo.GetDescendantDirectories(appID, rootDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("查询子目录失败: %w", err)
	}

	// 3. 构建所有目录节点的列表（包括根目录）
	allTrees := make([]*model.ServiceTree, 0, len(descendants)+1)
	allTrees = append(allTrees, rootTree)
	allTrees = append(allTrees, descendants...)

	// 4. 收集所有目录的 ServiceTreeID
	treeIDs := make([]int64, 0, len(allTrees))
	treeIDToPath := make(map[int64]string) // ServiceTreeID -> FullCodePath
	for _, tree := range allTrees {
		treeIDs = append(treeIDs, tree.ID)
		treeIDToPath[tree.ID] = tree.FullCodePath
	}

	// 5. 批量查询所有目录的当前版本快照（使用 ServiceTreeID 和 IsCurrent）
	// 一次性查询所有目录的快照，性能更好
	allSnapshots, err := s.fileSnapshotRepo.GetCurrentSnapshotsByServiceTreeIDs(treeIDs)
	if err != nil {
		return nil, fmt.Errorf("批量查询文件快照失败: %w", err)
	}

	// 6. 按目录路径分组快照
	result := make(map[string][]*model.FileSnapshot)
	for _, tree := range allTrees {
		// 初始化每个目录的空列表（即使没有文件也要包含）
		result[tree.FullCodePath] = make([]*model.FileSnapshot, 0)
	}

	// 7. 将快照按目录分组
	for _, snapshot := range allSnapshots {
		path := treeIDToPath[snapshot.ServiceTreeID]
		if path != "" {
			result[path] = append(result[path], snapshot)
		}
	}

	totalFiles := 0
	for _, files := range result {
		totalFiles += len(files)
	}

	logger.Infof(ctx, "[ServiceTreeService] 递归获取快照完成: 根目录=%s, 目录数=%d, 总文件数=%d",
		rootDirectoryPath, len(allTrees), totalFiles)

	return result, nil
}

// extractPackageFromPath 从完整路径提取 package 路径（去掉应用前缀）
func extractPackageFromPath(fullCodePath string) string {
	// 格式：/user/app/package1/package2
	// 返回：package1/package2
	parts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(parts) < 3 {
		return ""
	}
	return strings.Join(parts[2:], "/")
}

// CopyServiceTree 复制服务目录（递归复制目录及其所有子目录）
func (s *ServiceTreeService) CopyServiceTree(ctx context.Context, req *dto.CopyDirectoryReq) (*dto.CopyDirectoryResp, error) {
	// 1. 获取目标应用信息
	targetApp, err := s.appRepo.GetAppByID(req.TargetAppID)
	if err != nil {
		return nil, fmt.Errorf("获取目标应用失败: %w", err)
	}

	// 2. 解析源目录信息
	sourceParts := strings.Split(strings.Trim(req.SourceDirectoryPath, "/"), "/")
	if len(sourceParts) < 3 {
		return nil, fmt.Errorf("源目录路径格式错误: %s", req.SourceDirectoryPath)
	}
	sourceUser := sourceParts[0]
	sourceAppCode := sourceParts[1]

	// 3. 获取源应用信息
	sourceApp, err := s.appRepo.GetAppByUserName(sourceUser, sourceAppCode)
	if err != nil {
		return nil, fmt.Errorf("获取源应用失败: %w", err)
	}

	// 4. 获取源目录的 ServiceTree 信息（包括所有子目录）
	sourceRootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源目录信息失败: %w", err)
	}

	// 获取所有子目录（package 类型）
	sourceDescendants, err := s.serviceTreeRepo.GetDescendantDirectories(sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源子目录失败: %w", err)
	}

	// 构建源目录映射：fullCodePath -> ServiceTree
	sourceTrees := make(map[string]*model.ServiceTree)
	sourceTrees[sourceRootTree.FullCodePath] = sourceRootTree
	for _, desc := range sourceDescendants {
		sourceTrees[desc.FullCodePath] = desc
	}

	// 5. 递归获取所有目录的文件快照（包括根目录和所有子目录）
	directoryFiles, err := s.GetDirectorySnapshotsRecursively(ctx, sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取目录快照失败: %w", err)
	}

	if len(directoryFiles) == 0 {
		return nil, fmt.Errorf("未找到任何目录快照，请确保源目录已创建快照")
	}

	// 6. 获取目标根目录的 ServiceTree（用于确定父目录ID和完整路径）
	targetRootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.TargetDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取目标目录信息失败: %w", err)
	}

	// 使用目标根目录的完整路径作为基础路径
	targetRootPath := targetRootTree.FullCodePath

	// 8. 按层级顺序创建目标目录的 ServiceTree 记录
	// 先按路径长度排序，确保先创建父目录，再创建子目录
	type dirInfo struct {
		sourcePath string
		targetPath string
		sourceTree *model.ServiceTree
	}
	dirsToCreate := make([]dirInfo, 0, len(sourceTrees))

	for sourcePath, sourceTree := range sourceTrees {
		// 计算相对路径（相对于源根目录）
		relativePath := strings.TrimPrefix(sourcePath, req.SourceDirectoryPath)
		relativePath = strings.TrimPrefix(relativePath, "/")

		// 构建目标路径
		var targetPath string
		if relativePath == "" {
			// 根目录：直接在目标根目录下创建同名目录
			// 例如：从 /user/app/a/b 复制到 /user/app/e，应该在 /user/app/e/b 创建目录
			// 提取源目录的最后一部分作为目录名
			sourcePathParts := strings.Split(strings.Trim(req.SourceDirectoryPath, "/"), "/")
			if len(sourcePathParts) < 3 {
				return nil, fmt.Errorf("源目录路径格式错误: %s", req.SourceDirectoryPath)
			}
			dirName := sourcePathParts[len(sourcePathParts)-1] // 获取最后一部分（b）
			targetPath = targetRootPath + "/" + dirName
		} else {
			// 子目录：保持相对路径结构
			// 例如：从 /user/app/a/b/c 复制到 /user/app/e，应该在 /user/app/e/b/c 创建目录
			sourcePathParts := strings.Split(strings.Trim(req.SourceDirectoryPath, "/"), "/")
			if len(sourcePathParts) < 3 {
				return nil, fmt.Errorf("源目录路径格式错误: %s", req.SourceDirectoryPath)
			}
			dirName := sourcePathParts[len(sourcePathParts)-1] // 获取源根目录名（b）
			targetPath = targetRootPath + "/" + dirName + "/" + relativePath
		}

		dirsToCreate = append(dirsToCreate, dirInfo{
			sourcePath: sourcePath,
			targetPath: targetPath,
			sourceTree: sourceTree,
		})
	}

	// 按路径长度排序（短的在前，确保先创建父目录）
	sort.Slice(dirsToCreate, func(i, j int) bool {
		return len(dirsToCreate[i].targetPath) < len(dirsToCreate[j].targetPath)
	})

	// 更新目标路径（因为根目录会在目标根目录下创建同名目录）
	// 例如：从 /user/app/a/b 复制到 /user/app/e，实际目标路径应该是 /user/app/e/b
	if len(dirsToCreate) > 0 {
		// 使用第一个目录（根目录）的目标路径作为新的目标根路径
		targetRootPath = dirsToCreate[0].targetPath
	}

	// 8. 构建批量创建请求（优化版本：使用批量创建接口）
	batchItems := make([]*dto.DirectoryTreeItem, 0)

	// 8.1 添加目录项
	for _, dirInfo := range dirsToCreate {
		batchItems = append(batchItems, &dto.DirectoryTreeItem{
			FullCodePath: dirInfo.targetPath,
			Type:         "directory",
			Name:         dirInfo.sourceTree.Name,
			Description:  dirInfo.sourceTree.Description,
			Tags:         dirInfo.sourceTree.Tags,
		})
	}

	// 8.2 添加文件项
	totalFileCount := 0
	for sourcePath, fileSnapshots := range directoryFiles {
		// 计算相对路径（相对于源根目录）
		relativePath := strings.TrimPrefix(sourcePath, req.SourceDirectoryPath)
		relativePath = strings.TrimPrefix(relativePath, "/")

		// 构建目标路径
		var targetPath string
		if relativePath == "" {
			// 根目录
			targetPath = targetRootPath
		} else {
			// 子目录：保持相对路径结构
			targetPath = targetRootPath + "/" + relativePath
		}

		for _, fileSnapshot := range fileSnapshots {
			// 使用 FileName，如果为空则从相对路径提取
			fileName := fileSnapshot.FileName
			if fileName == "" {
				// 从相对路径提取文件名
				fileName = strings.TrimSuffix(fileSnapshot.RelativePath, ".go")
				if lastSlash := strings.LastIndex(fileName, "/"); lastSlash >= 0 {
					fileName = fileName[lastSlash+1:]
				}
			}

			// 忽略 init_.go 文件（init_.go 是创建目录时动态生成的，不能复制）
			if fileName == "init_" || fileSnapshot.RelativePath == "init_.go" || strings.HasSuffix(fileSnapshot.RelativePath, "/init_.go") {
				logger.Infof(ctx, "[ServiceTreeService] 跳过 init_.go 文件: %s", fileSnapshot.RelativePath)
				continue
			}

			// 构建文件完整路径
			fileFullPath := targetPath + "/" + fileName
			if fileSnapshot.FileType != "" && fileSnapshot.FileType != "go" {
				fileFullPath = targetPath + "/" + fileName + "." + fileSnapshot.FileType
			} else {
				fileFullPath = targetPath + "/" + fileName + ".go"
			}

			// 添加文件项
			batchItems = append(batchItems, &dto.DirectoryTreeItem{
				FullCodePath: fileFullPath,
				Type:         "file",
				FileName:     fileName,
				FileType:     fileSnapshot.FileType,
				Content:      fileSnapshot.Content,
				RelativePath: fileSnapshot.RelativePath,
			})
			totalFileCount++
		}

		logger.Infof(ctx, "[ServiceTreeService] 准备复制目录: source=%s, target=%s, fileCount=%d",
			sourcePath, targetPath, len(fileSnapshots))
	}

	// 9. 批量创建目录和文件（一次性调用 runtime）
	logger.Infof(ctx, "[ServiceTreeService] 准备批量创建目录树: directoryCount=%d, fileCount=%d",
		len(dirsToCreate), totalFileCount)

	batchReq := &dto.BatchCreateDirectoryTreeReq{
		User:  targetApp.User,
		App:   targetApp.Code,
		Items: batchItems,
	}

	batchResp, err := s.BatchCreateDirectoryTree(ctx, batchReq)
	if err != nil {
		logger.Errorf(ctx, "[ServiceTreeService] 批量创建目录树失败: error=%v", err)
		return nil, fmt.Errorf("批量创建目录树失败: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] 批量创建目录树完成: directoryCount=%d, fileCount=%d",
		batchResp.DirectoryCount, batchResp.FileCount)

	// 10. 调用 UpdateApp（只用于编译和启动，文件已经通过批量创建了）
	// 注意：这里仍然需要调用 UpdateApp 来触发编译和启动，但不需要 CreateFunctions
	logger.Infof(ctx, "[ServiceTreeService] 准备调用 UpdateApp（编译和启动）")

	updateReq := &dto.UpdateAppReq{
		User: targetApp.User,
		App:  targetApp.Code,
		// 不传递 CreateFunctions，因为文件已经通过批量创建了
	}

	_, err = s.appService.UpdateApp(ctx, updateReq)
	if err != nil {
		logger.Errorf(ctx, "[ServiceTreeService] UpdateApp（编译和启动）失败: error=%v", err)
		return nil, fmt.Errorf("UpdateApp 失败: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] 复制目录完成: 目录数=%d, 文件数=%d", batchResp.DirectoryCount, batchResp.FileCount)

	return &dto.CopyDirectoryResp{
		Message:        fmt.Sprintf("复制目录成功，共复制 %d 个目录，%d 个文件", batchResp.DirectoryCount, batchResp.FileCount),
		DirectoryCount: batchResp.DirectoryCount,
		FileCount:      batchResp.FileCount,
	}, nil
}

// PublishDirectoryToHub 发布目录到 Hub
func (s *ServiceTreeService) PublishDirectoryToHub(ctx context.Context, req *dto.PublishDirectoryToHubReq) (*dto.PublishDirectoryToHubResp, error) {
	// 1. 获取应用信息
	sourceApp, err := s.appRepo.GetAppByUserName(req.SourceUser, req.SourceApp)
	if err != nil {
		return nil, fmt.Errorf("获取源应用失败: %w", err)
	}

	// 2. 验证源目录是否存在
	sourceTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源目录信息失败: %w", err)
	}

	// 3. 获取根目录节点和所有子目录节点（用于构建父子关系）
	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取根目录节点失败: %w", err)
	}

	descendants, err := s.serviceTreeRepo.GetDescendantDirectories(sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("查询子目录失败: %w", err)
	}

	// 构建所有目录节点的列表和映射
	allTrees := make([]*model.ServiceTree, 0, len(descendants)+1)
	allTrees = append(allTrees, rootTree)
	allTrees = append(allTrees, descendants...)

	// 构建路径到 ServiceTree 的映射，用于快速查找父目录
	pathToTree := make(map[string]*model.ServiceTree)
	idToTree := make(map[int64]*model.ServiceTree)
	for _, tree := range allTrees {
		pathToTree[tree.FullCodePath] = tree
		idToTree[tree.ID] = tree
	}

	// 4. 递归获取所有目录的文件快照（包括根目录和所有子目录）
	directoryFiles, err := s.GetDirectorySnapshotsRecursively(ctx, sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取目录快照失败: %w", err)
	}

	if len(directoryFiles) == 0 {
		return nil, fmt.Errorf("未找到任何目录快照，请确保源目录已创建快照")
	}

	// 5. 获取所有函数节点（function 类型，属于当前目录树下的）
	// 使用路径前缀匹配，只查询属于当前目录的函数
	normalizedPath := strings.TrimSuffix(req.SourceDirectoryPath, "/") + "/"
	allFunctions, err := s.serviceTreeRepo.GetServiceTreesByAppIDAndType(sourceApp.ID, model.ServiceTreeTypeFunction)
	if err != nil {
		return nil, fmt.Errorf("查询函数节点失败: %w", err)
	}

	// 构建函数映射：ParentID -> []Function
	functionMap := make(map[int64][]*model.ServiceTree)
	for _, fn := range allFunctions {
		// 只包含属于当前目录树下的函数（路径前缀匹配）
		if strings.HasPrefix(fn.FullCodePath, normalizedPath) || fn.FullCodePath == req.SourceDirectoryPath {
			// 找到函数所属的目录节点（通过 ParentID 匹配）
			if dirTree, exists := idToTree[fn.ParentID]; exists {
				// 确保目录节点也在当前目录树下
				if strings.HasPrefix(dirTree.FullCodePath, normalizedPath) || dirTree.FullCodePath == req.SourceDirectoryPath {
					functionMap[dirTree.ID] = append(functionMap[dirTree.ID], fn)
				}
			}
		}
	}

	// 6. 构建树形结构（包含函数）
	directoryTree := s.buildDirectoryTree(rootTree, allTrees, directoryFiles, idToTree, functionMap)

	// 6. 构建 Hub 请求
	hubReq := &dto.PublishHubDirectoryReq{
		SourceUser:           req.SourceUser,
		SourceApp:            req.SourceApp,
		SourceDirectoryPath:  req.SourceDirectoryPath,
		Name:                 req.Name,
		Description:          req.Description,
		Category:             req.Category,
		Tags:                 req.Tags,
		ServiceFeePersonal:   req.ServiceFeePersonal,
		ServiceFeeEnterprise: req.ServiceFeeEnterprise,
		Version:              sourceTree.Version,
		DirectoryTree:        directoryTree,
	}

	// 7. 调用 Hub API
	header := &apicall.Header{
		TraceID:     contextx.GetTraceId(ctx),
		RequestUser: contextx.GetRequestUser(ctx),
		Token:       contextx.GetToken(ctx),
	}

	hubResp, err := apicall.PublishDirectoryToHub(header, hubReq)
	if err != nil {
		return nil, fmt.Errorf("调用 Hub API 失败: %w", err)
	}

	// 8. 建立双向绑定：更新根目录节点的 HubDirectoryID 和版本信息
	// 需要查询 Hub 获取版本信息（因为 hubResp 可能不包含版本）
		hubDetail, err := apicall.GetHubDirectoryDetail(header, req.SourceDirectoryPath, "", false, false)
	if err != nil {
		logger.Warnf(ctx, "[PublishDirectoryToHub] 获取Hub目录详情失败，无法记录版本信息: hubDirectoryID=%d, error=%v", hubResp.HubDirectoryID, err)
		// 即使获取详情失败，也记录 HubDirectoryID
		rootTree.HubDirectoryID = hubResp.HubDirectoryID
	} else {
		// 记录 Hub 信息（ID 和版本）
		rootTree.HubDirectoryID = hubDetail.ID
		rootTree.HubVersion = hubDetail.Version
		rootTree.HubVersionNum = hubDetail.VersionNum
	}

	if err := s.serviceTreeRepo.UpdateServiceTree(rootTree); err != nil {
		logger.Warnf(ctx, "[PublishDirectoryToHub] 更新ServiceTree的Hub信息失败: treeID=%d, hubDirectoryID=%d, hubVersion=%s, error=%v",
			rootTree.ID, rootTree.HubDirectoryID, rootTree.HubVersion, err)
		// 不返回错误，因为发布已经成功，只是绑定失败
	} else {
		logger.Infof(ctx, "[PublishDirectoryToHub] 成功建立双向绑定: treeID=%d, hubDirectoryID=%d, hubVersion=%s", rootTree.ID, rootTree.HubDirectoryID, rootTree.HubVersion)
	}

	// 9. 返回结果
	return &dto.PublishDirectoryToHubResp{
		HubDirectoryID:  hubResp.HubDirectoryID,
		HubDirectoryURL: hubResp.HubDirectoryURL,
		DirectoryCount:  hubResp.DirectoryCount,
		FileCount:       hubResp.FileCount,
	}, nil
}

// PushDirectoryToHub 推送目录到 Hub（更新已发布的目录，类似 git push）
func (s *ServiceTreeService) PushDirectoryToHub(ctx context.Context, req *dto.PushDirectoryToHubReq) (*dto.PushDirectoryToHubResp, error) {
	// 1. 获取应用信息
	sourceApp, err := s.appRepo.GetAppByUserName(req.SourceUser, req.SourceApp)
	if err != nil {
		return nil, fmt.Errorf("获取源应用失败: %w", err)
	}

	// 2. 验证源目录是否存在
	sourceTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源目录信息失败: %w", err)
	}

	// 3. 检查目录是否已发布到 Hub
	if sourceTree.HubDirectoryID == 0 {
		return nil, fmt.Errorf("目录尚未发布到 Hub，请先使用 PublishDirectoryToHub 发布")
	}

	// 4. 获取根目录节点和所有子目录节点（用于构建父子关系）
	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取根目录节点失败: %w", err)
	}

	descendants, err := s.serviceTreeRepo.GetDescendantDirectories(sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("查询子目录失败: %w", err)
	}

	// 构建所有目录节点的列表和映射
	allTrees := make([]*model.ServiceTree, 0, len(descendants)+1)
	allTrees = append(allTrees, rootTree)
	allTrees = append(allTrees, descendants...)

	// 构建路径到 ServiceTree 的映射，用于快速查找父目录
	idToTree := make(map[int64]*model.ServiceTree)
	for _, tree := range allTrees {
		idToTree[tree.ID] = tree
	}

	// 5. 递归获取所有目录的文件快照（包括根目录和所有子目录）
	directoryFiles, err := s.GetDirectorySnapshotsRecursively(ctx, sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取目录快照失败: %w", err)
	}

	if len(directoryFiles) == 0 {
		return nil, fmt.Errorf("未找到任何目录快照，请确保源目录已创建快照")
	}

	// 6. 获取所有函数节点（function 类型，属于当前目录树下的）
	// 使用路径前缀匹配，只查询属于当前目录的函数
	normalizedPath := strings.TrimSuffix(req.SourceDirectoryPath, "/") + "/"
	allFunctions, err := s.serviceTreeRepo.GetServiceTreesByAppIDAndType(sourceApp.ID, model.ServiceTreeTypeFunction)
	if err != nil {
		return nil, fmt.Errorf("查询函数节点失败: %w", err)
	}

	// 构建函数映射：ParentID -> []Function
	functionMap := make(map[int64][]*model.ServiceTree)
	for _, fn := range allFunctions {
		// 只包含属于当前目录树下的函数（路径前缀匹配）
		if strings.HasPrefix(fn.FullCodePath, normalizedPath) || fn.FullCodePath == req.SourceDirectoryPath {
			// 找到函数所属的目录节点（通过 ParentID 匹配）
			if dirTree, exists := idToTree[fn.ParentID]; exists {
				// 确保目录节点也在当前目录树下
				if strings.HasPrefix(dirTree.FullCodePath, normalizedPath) || dirTree.FullCodePath == req.SourceDirectoryPath {
					functionMap[dirTree.ID] = append(functionMap[dirTree.ID], fn)
				}
			}
		}
	}

	// 7. 构建树形结构（包含函数）
	directoryTree := s.buildDirectoryTree(rootTree, allTrees, directoryFiles, idToTree, functionMap)

	// 7. 构建 Hub 请求
	hubReq := &dto.UpdateHubDirectoryReq{
		APIKey:               req.APIKey,
		HubDirectoryID:       sourceTree.HubDirectoryID,
		SourceDirectoryPath:  req.SourceDirectoryPath,
		Name:                 req.Name,
		Description:          req.Description,
		Category:             req.Category,
		Tags:                 req.Tags,
		ServiceFeePersonal:   req.ServiceFeePersonal,
		ServiceFeeEnterprise: req.ServiceFeeEnterprise,
		Version:              req.Version,
		DirectoryTree:        directoryTree,
	}

	// 8. 调用 Hub API
	header := &apicall.Header{
		TraceID:     contextx.GetTraceId(ctx),
		RequestUser: contextx.GetRequestUser(ctx),
		Token:       contextx.GetToken(ctx),
	}

	hubResp, err := apicall.UpdateDirectoryToHub(header, hubReq)
	if err != nil {
		return nil, fmt.Errorf("调用 Hub API 失败: %w", err)
	}

	// 9. 更新根目录节点的版本信息
	rootTree.HubVersion = hubResp.NewVersion
	rootTree.HubVersionNum = extractVersionNumForServiceTree(hubResp.NewVersion)
	if err := s.serviceTreeRepo.UpdateServiceTree(rootTree); err != nil {
		logger.Warnf(ctx, "[PushDirectoryToHub] 更新ServiceTree的Hub版本信息失败: treeID=%d, hubDirectoryID=%d, newVersion=%s, error=%v",
			rootTree.ID, hubResp.HubDirectoryID, hubResp.NewVersion, err)
		// 不返回错误，因为推送已经成功，只是更新版本信息失败
	} else {
		logger.Infof(ctx, "[PushDirectoryToHub] 成功更新Hub版本: treeID=%d, hubDirectoryID=%d, oldVersion=%s, newVersion=%s",
			rootTree.ID, hubResp.HubDirectoryID, hubResp.OldVersion, hubResp.NewVersion)
	}

	// 10. 返回结果
	return &dto.PushDirectoryToHubResp{
		HubDirectoryID:  hubResp.HubDirectoryID,
		HubDirectoryURL: hubResp.HubDirectoryURL,
		DirectoryCount:  hubResp.DirectoryCount,
		FileCount:       hubResp.FileCount,
		OldVersion:      hubResp.OldVersion,
		NewVersion:      hubResp.NewVersion,
	}, nil
}

// BatchCreateDirectoryTree 批量创建目录树（用于 copy 和 pull from hub）
func (s *ServiceTreeService) BatchCreateDirectoryTree(
	ctx context.Context,
	req *dto.BatchCreateDirectoryTreeReq,
) (*dto.BatchCreateDirectoryTreeResp, error) {
	// 1. 获取应用信息
	app, err := s.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	// 2. 构建 runtime 请求
	runtimeReq := &dto.BatchCreateDirectoryTreeRuntimeReq{
		User:  req.User,
		App:   req.App,
		Items: req.Items,
	}

	// 3. 调用 runtime 批量创建
	runtimeResp, err := s.appRuntime.BatchCreateDirectoryTree(ctx, app.HostID, runtimeReq)
	if err != nil {
		return nil, fmt.Errorf("批量创建目录树失败: %w", err)
	}

	// 4. 创建 ServiceTree 记录（批量创建数据库记录）
	// 这里需要根据 Items 创建对应的 ServiceTree 记录
	// 先按路径排序，确保先创建父目录
	sortedItems := make([]*dto.DirectoryTreeItem, len(req.Items))
	copy(sortedItems, req.Items)

	// 按路径长度排序
	sort.Slice(sortedItems, func(i, j int) bool {
		return len(sortedItems[i].FullCodePath) < len(sortedItems[j].FullCodePath)
	})

	// 创建路径到 ServiceTree 的映射
	pathToTree := make(map[string]*model.ServiceTree)
	currentVersionNum := extractVersionNumForServiceTree(app.Version)

	// 遍历所有项，创建 ServiceTree 记录（只处理目录）
	for _, item := range sortedItems {
		// 只处理目录，文件不在这里处理
		if item.Type != "directory" {
			continue
		}

		// 提取目录代码（路径的最后一部分）
		pathParts := strings.Split(strings.Trim(item.FullCodePath, "/"), "/")
		if len(pathParts) < 3 {
			continue // 跳过无效路径
		}
		dirCode := pathParts[len(pathParts)-1]

		// 查找父目录
		var parentID int64 = 0
		parentPath := getParentPath(item.FullCodePath)
		if parentPath != "" {
			if parentTree, exists := pathToTree[parentPath]; exists {
				parentID = parentTree.ID
			}
		}

		// 创建 ServiceTree 记录
		newTree := &model.ServiceTree{
			Name:             item.Name,
			Code:             dirCode,
			ParentID:         parentID,
			Type:             model.ServiceTreeTypePackage,
			Description:      item.Description,
			Tags:             item.Tags,
			AppID:            app.ID,
			FullCodePath:     item.FullCodePath,
			AddVersionNum:    currentVersionNum,
			UpdateVersionNum: 0,
		}

		// 保存到数据库
		if err := s.serviceTreeRepo.CreateServiceTreeWithParentPath(newTree, ""); err != nil {
			logger.Warnf(ctx, "[BatchCreateDirectoryTree] 创建 ServiceTree 记录失败: path=%s, error=%v",
				item.FullCodePath, err)
			// 不返回错误，因为目录已经创建成功
		} else {
			pathToTree[item.FullCodePath] = newTree
		}
	}

	return &dto.BatchCreateDirectoryTreeResp{
		DirectoryCount: runtimeResp.DirectoryCount,
		FileCount:      runtimeResp.FileCount,
		CreatedPaths:   runtimeResp.CreatedPaths,
	}, nil
}

// getParentPath 获取父目录路径（辅助函数）
func getParentPath(fullCodePath string) string {
	pathParts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(pathParts) <= 3 {
		return "" // 没有父目录（已经是根目录）
	}
	// 去掉最后一部分，重新组合
	parentParts := pathParts[:len(pathParts)-1]
	return "/" + strings.Join(parentParts, "/")
}

// AddFunctions 向服务目录添加函数（同步处理）
func (s *ServiceTreeService) AddFunctions(ctx context.Context, req *dto.AddFunctionsReq) (*dto.AddFunctionsResp, error) {
	// 1. 根据 TreeID 获取 ServiceTree（需要预加载 App）
	serviceTree, err := s.serviceTreeRepo.GetByID(req.TreeID)
	if err != nil {
		logger.Errorf(ctx, "[ServiceTreeService] 获取 ServiceTree 失败: TreeID=%d, error=%v", req.TreeID, err)
		return &dto.AddFunctionsResp{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// 预加载 App 信息（如果还没有加载）
	if serviceTree.App == nil {
		app, err := s.appRepo.GetAppByID(serviceTree.AppID)
		if err != nil {
			logger.Errorf(ctx, "[ServiceTreeService] 获取 App 失败: AppID=%d, error=%v", serviceTree.AppID, err)
			return &dto.AddFunctionsResp{
				Success: false,
				Error:   err.Error(),
			}, err
		}
		serviceTree.App = app
	}

	// 2. 从 ServiceTree 中提取 package 路径（使用 model 方法）
	packagePath := serviceTree.GetPackagePathForFileCreation()

	// 3. 使用 agent-server 处理后的结构化数据
	fileName := req.FileName
	if fileName == "" {
		logger.Warnf(ctx, "[ServiceTreeService] agent-server 未提取到文件名，使用 ServiceTree.Code 作为 fallback: %s", serviceTree.Code)
		fileName = serviceTree.Code
	}

	sourceCode := req.SourceCode
	if sourceCode == "" {
		logger.Warnf(ctx, "[ServiceTreeService] agent-server 未处理代码，使用原始代码")
		sourceCode = req.Code
	}

	logger.Infof(ctx, "[ServiceTreeService] 添加函数: Package=%s, FileName=%s, SourceCodeLength=%d", packagePath, fileName, len(sourceCode))

	// 4. 构建 CreateFunctionInfo
	createFunction := &dto.CreateFunctionInfo{
		Package:    packagePath,
		GroupCode:  fileName,
		SourceCode: sourceCode,
	}

	// 5. 调用 AppService.UpdateApp
	updateReq := &dto.UpdateAppReq{
		User:            req.User,
		App:             serviceTree.App.Code,
		CreateFunctions: []*dto.CreateFunctionInfo{createFunction},
	}

	updateResp, err := s.appService.UpdateApp(ctx, updateReq)
	if err != nil {
		logger.Errorf(ctx, "[ServiceTreeService] AppService.UpdateApp 失败: error=%v", err)
		return &dto.AddFunctionsResp{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// 6. 获取新增的 FullGroupCodes
	fullGroupCodes := make([]string, 0)
	if updateResp.Diff != nil {
		fullGroupCodes = updateResp.Diff.GetAddFullGroupCodes()
		logger.Infof(ctx, "[ServiceTreeService] 添加函数完成 - Count: %d, FullGroupCodes: %v", len(fullGroupCodes), fullGroupCodes)
	}

	// 7. 返回同步结果（不发送回调）
	return &dto.AddFunctionsResp{
		Success:        true,
		FullGroupCodes: fullGroupCodes,
		AppID:          serviceTree.App.ID,
		AppCode:        serviceTree.App.Code,
	}, nil
}

// buildDirectoryTree 构建目录树结构（递归，包含函数）
// rootTree: 根目录节点
// allTrees: 所有目录节点（包括根目录和子目录）
// directoryFiles: 目录路径到文件快照的映射
// idToTree: ServiceTreeID 到 ServiceTree 的映射
// functionMap: 目录ID到函数列表的映射
func (s *ServiceTreeService) buildDirectoryTree(
	rootTree *model.ServiceTree,
	allTrees []*model.ServiceTree,
	directoryFiles map[string][]*model.FileSnapshot,
	idToTree map[int64]*model.ServiceTree,
	functionMap map[int64][]*model.ServiceTree,
) *dto.DirectoryTreeNode {
	return s.buildDirectoryTreeNode(rootTree, allTrees, directoryFiles, idToTree, functionMap)
}

// buildDirectoryTreeNode 递归构建目录树节点（包含函数）
func (s *ServiceTreeService) buildDirectoryTreeNode(
	tree *model.ServiceTree,
	allTrees []*model.ServiceTree,
	directoryFiles map[string][]*model.FileSnapshot,
	idToTree map[int64]*model.ServiceTree,
	functionMap map[int64][]*model.ServiceTree,
) *dto.DirectoryTreeNode {
	// 获取目录名称（路径的最后一部分）
	pathParts := strings.Split(strings.Trim(tree.FullCodePath, "/"), "/")
	dirName := pathParts[len(pathParts)-1]

	// 构建文件列表
	files := make([]*dto.FileSnapshotInfo, 0)
	if fileSnapshots, exists := directoryFiles[tree.FullCodePath]; exists {
		for _, file := range fileSnapshots {
			files = append(files, &dto.FileSnapshotInfo{
				FileName:     file.FileName,
				RelativePath: file.RelativePath,
				Content:      file.Content,
				FileType:     file.FileType,
				FileVersion:  file.FileVersion,
			})
		}
	}

	// 构建函数列表
	functions := make([]*dto.HubFunctionInfo, 0)
	if functionList, exists := functionMap[tree.ID]; exists {
		for _, fn := range functionList {
			functions = append(functions, &dto.HubFunctionInfo{
				ID:           fn.ID,
				Name:         fn.Name,
				Code:         fn.Code,
				FullCodePath: fn.FullCodePath,
				Description:  fn.Description,
				TemplateType: fn.TemplateType,
				Tags:         fn.GetTagsSlice(),
				RefID:        fn.RefID,
				Version:      fn.Version,
				VersionNum:   fn.VersionNum,
			})
		}
	}

	// 查找所有直接子目录
	subdirectories := make([]*dto.DirectoryTreeNode, 0)
	for _, childTree := range allTrees {
		if childTree.ParentID == tree.ID {
			// 递归构建子目录节点
			childNode := s.buildDirectoryTreeNode(childTree, allTrees, directoryFiles, idToTree, functionMap)
			subdirectories = append(subdirectories, childNode)
		}
	}

	return &dto.DirectoryTreeNode{
		Name:           dirName,
		Path:           tree.FullCodePath,
		Files:          files,
		Functions:      functions,
		Subdirectories: subdirectories,
	}
}

// HubLinkInfo Hub 链接信息
type HubLinkInfo struct {
	Host         string // Hub 主机地址（如 hub.example.com:8080）
	FullCodePath string // 目录完整路径（如 /user/app/plugins/cashier）
	Version      string // 版本号（可选）
}

// ParseHubLink 解析 Hub 链接
// 格式：hub://{host}/{full_code_path} 或 hub://{host}/{full_code_path}@v1.0.0
// 例如：hub://hub.example.com/luobei/demo/plugins/cashier 或 hub://hub.example.com/luobei/demo/plugins/cashier@v1.0.0
func ParseHubLink(hubLink string) (*HubLinkInfo, error) {
	// 移除协议前缀
	if !strings.HasPrefix(hubLink, "hub://") {
		return nil, fmt.Errorf("无效的 Hub 链接格式，必须以 hub:// 开头")
	}

	link := strings.TrimPrefix(hubLink, "hub://")

	// 解析版本号（可选）
	var version string
	if idx := strings.LastIndex(link, "@"); idx != -1 {
		version = link[idx+1:]
		link = link[:idx]
	}

	// 解析主机和 full-code-path
	parts := strings.SplitN(link, "/", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("无效的 Hub 链接格式，缺少 full-code-path")
	}

	host := parts[0]
	fullCodePath := "/" + parts[1] // 确保以 / 开头

	// 验证 full-code-path 格式（应该至少包含 /user/app/...）
	pathParts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("无效的 full-code-path 格式，应该至少包含 /user/app/...")
	}

	return &HubLinkInfo{
		Host:         host,
		FullCodePath: fullCodePath,
		Version:      version,
	}, nil
}

// PullDirectoryFromHub 从 Hub 拉取目录到工作空间（类似 git pull）
func (s *ServiceTreeService) PullDirectoryFromHub(ctx context.Context, req *dto.PullDirectoryFromHubReq) (*dto.PullDirectoryFromHubResp, error) {
	// 1. 解析 Hub 链接
	hubLinkInfo, err := ParseHubLink(req.HubLink)
	if err != nil {
		return nil, fmt.Errorf("解析 Hub 链接失败: %w", err)
	}

	logger.Infof(ctx, "[PullDirectoryFromHub] 解析 Hub 链接成功: host=%s, fullCodePath=%s, version=%s",
		hubLinkInfo.Host, hubLinkInfo.FullCodePath, hubLinkInfo.Version)

	// 2. 从 Hub 获取目录详情（包含目录树和文件内容，如果指定了版本号则查询指定版本）
	hubDetail, err := apicall.GetHubDirectoryDetailFromHost(hubLinkInfo.Host, hubLinkInfo.FullCodePath, hubLinkInfo.Version, true, true)
	if err != nil {
		return nil, fmt.Errorf("获取 Hub 目录详情失败: %w", err)
	}

	// 3. 如果指定了版本号，验证版本是否匹配
	if hubLinkInfo.Version != "" && hubDetail.Version != hubLinkInfo.Version {
		return nil, fmt.Errorf("版本不匹配：请求版本 %s，实际版本 %s", hubLinkInfo.Version, hubDetail.Version)
	}

	// 4. 获取目标应用信息
	targetApp, err := s.appRepo.GetAppByUserName(req.TargetUser, req.TargetApp)
	if err != nil {
		return nil, fmt.Errorf("获取目标应用失败: %w", err)
	}

	// 5. 确定目标目录路径
	targetPath := req.TargetDirectoryPath
	if targetPath == "" {
		targetPath = fmt.Sprintf("/%s/%s", targetApp.User, targetApp.Code)
	}

	// 6. 从 DirectoryTree 构建批量创建请求
	if hubDetail.DirectoryTree == nil {
		return nil, fmt.Errorf("Hub 目录树为空")
	}

	// 6.1 构建目录项列表
	directoryItems := make([]*dto.DirectoryTreeItem, 0)
	fileItems := make([]*dto.DirectoryTreeItem, 0)

	// 递归遍历目录树，构建目录和文件项
	s.buildItemsFromTree(hubDetail.DirectoryTree, targetPath, &directoryItems, &fileItems)

	// 7. 先批量创建目录
	if len(directoryItems) > 0 {
		batchCreateReq := &dto.BatchCreateDirectoryTreeReq{
			User:  req.TargetUser,
			App:   req.TargetApp,
			Items: directoryItems,
		}

		batchCreateResp, err := s.BatchCreateDirectoryTree(ctx, batchCreateReq)
		if err != nil {
			return nil, fmt.Errorf("批量创建目录失败: %w", err)
		}

		logger.Infof(ctx, "[PullDirectoryFromHub] 批量创建目录完成: directoryCount=%d", batchCreateResp.DirectoryCount)
	}

	// 8. 再批量写文件（会编译并返回 diff）
	if len(fileItems) > 0 {
		batchWriteReq := &dto.BatchWriteFilesReq{
			User:  req.TargetUser,
			App:   req.TargetApp,
			Files: fileItems,
		}

		batchWriteResp, err := s.BatchWriteFiles(ctx, batchWriteReq)
		if err != nil {
			return nil, fmt.Errorf("批量写文件失败: %w", err)
		}
		logger.Infof(ctx, "[PullDirectoryFromHub] 批量写文件完成: fileCount=%d", batchWriteResp.FileCount)
	}

	// 9. 获取根目录的 ServiceTree ID（用于建立双向绑定）
	// 注意：根目录路径应该是 targetPath + 目录名称（从 Hub 目录树的第一层目录名）
	rootDirPath := targetPath
	if hubDetail.DirectoryTree != nil {
		rootDirPath = fmt.Sprintf("%s/%s", targetPath, hubDetail.DirectoryTree.Name)
	}
	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(rootDirPath)
	if err != nil {
		logger.Warnf(ctx, "[PullDirectoryFromHub] 获取根目录 ServiceTree 失败: path=%s, error=%v", targetPath, err)
	}

	// 10. 建立双向绑定：更新根目录节点的 HubDirectoryID 和版本信息
	if rootTree != nil && hubDetail.ID > 0 {
		rootTree.HubDirectoryID = hubDetail.ID
		rootTree.HubVersion = hubDetail.Version
		rootTree.HubVersionNum = hubDetail.VersionNum
		if err := s.serviceTreeRepo.UpdateServiceTree(rootTree); err != nil {
			logger.Warnf(ctx, "[PullDirectoryFromHub] 更新ServiceTree的Hub信息失败: treeID=%d, hubDirectoryID=%d, hubVersion=%s, error=%v",
				rootTree.ID, hubDetail.ID, hubDetail.Version, err)
		} else {
			logger.Infof(ctx, "[PullDirectoryFromHub] 成功建立双向绑定: treeID=%d, hubDirectoryID=%d, hubVersion=%s", rootTree.ID, hubDetail.ID, hubDetail.Version)
		}
	}

	return &dto.PullDirectoryFromHubResp{
		Message:             fmt.Sprintf("从 Hub 安装目录成功，共安装 %d 个目录，%d 个文件", len(directoryItems), len(fileItems)),
		DirectoryCount:      len(directoryItems),
		FileCount:           len(fileItems),
		TargetDirectoryPath: rootDirPath,
		ServiceTreeID: func() int64 {
			if rootTree != nil {
				return rootTree.ID
			} else {
				return 0
			}
		}(),
		HubDirectoryID:   hubDetail.ID,
		HubDirectoryName: hubDetail.Name,
		HubVersion:       hubDetail.Version,
		HubVersionNum:    hubDetail.VersionNum,
	}, nil
}

// BatchWriteFiles 批量写文件（app-server 端，调用 runtime）
func (s *ServiceTreeService) BatchWriteFiles(ctx context.Context, req *dto.BatchWriteFilesReq) (*dto.BatchWriteFilesResp, error) {
	// 1. 获取应用信息
	app, err := s.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	// 2. 构建 runtime 请求
	runtimeReq := &dto.BatchWriteFilesRuntimeReq{
		User:  req.User,
		App:   req.App,
		Files: req.Files,
	}

	// 3. 调用 runtime 批量写文件
	runtimeResp, err := s.appRuntime.BatchWriteFiles(ctx, app.HostID, runtimeReq)
	if err != nil {
		return nil, fmt.Errorf("批量写文件失败: %w", err)
	}

	// 4. 处理 diff，更新 ServiceTree 的 function 节点（通过 processAPIDiff）
	if runtimeResp.Diff != nil {
		// 构建 UpdateAppReq 用于 processAPIDiff（兼容旧接口）
		updateAppReq := &dto.UpdateAppReq{
			User: req.User,
			App:  req.App,
		}
		// 调用 appService.processAPIDiff 处理 API 差异
		if err := s.appService.processAPIDiff(ctx, app.ID, runtimeResp.Diff, updateAppReq, 0, runtimeResp.GitCommitHash); err != nil {
			logger.Warnf(ctx, "[BatchWriteFiles] 处理 API diff 失败: %v", err)
			// 不返回错误，因为文件已经写入成功
		}
	}

	return &dto.BatchWriteFilesResp{
		FileCount:     runtimeResp.FileCount,
		WrittenPaths:  runtimeResp.WrittenPaths,
		Diff:          runtimeResp.Diff,
		OldVersion:    runtimeResp.OldVersion,
		NewVersion:    runtimeResp.NewVersion,
		GitCommitHash: runtimeResp.GitCommitHash,
	}, nil
}

// buildItemsFromTree 递归构建目录和文件项列表
func (s *ServiceTreeService) buildItemsFromTree(
	node *dto.DirectoryTreeNode,
	targetBasePath string,
	directoryItems *[]*dto.DirectoryTreeItem,
	fileItems *[]*dto.DirectoryTreeItem,
) {
	// 计算当前目录的目标路径
	dirName := node.Name
	currentTargetPath := fmt.Sprintf("%s/%s", targetBasePath, dirName)

	// 添加目录项
	*directoryItems = append(*directoryItems, &dto.DirectoryTreeItem{
		FullCodePath: currentTargetPath,
		Type:         "directory",
		Name:         dirName,
		Description:  "", // Hub 目录树可能没有描述
		Tags:         "", // Hub 目录树可能没有标签
	})

	// 添加文件项
	for _, file := range node.Files {
		// 从 RelativePath 提取文件名（不含扩展名）
		fileName := file.FileName
		if fileName == "" {
			// 从 RelativePath 提取
			pathParts := strings.Split(file.RelativePath, "/")
			fileName = pathParts[len(pathParts)-1]
			if ext := strings.LastIndex(fileName, "."); ext != -1 {
				fileName = fileName[:ext]
			}
		}

		fileFullPath := fmt.Sprintf("%s/%s", currentTargetPath, fileName)
		*fileItems = append(*fileItems, &dto.DirectoryTreeItem{
			FullCodePath: fileFullPath,
			Type:         "file",
			FileName:     fileName,
			FileType:     file.FileType,
			Content:      file.Content, // 文件内容
			RelativePath: file.RelativePath,
		})
	}

	// 递归处理子目录
	for _, subdir := range node.Subdirectories {
		s.buildItemsFromTree(subdir, currentTargetPath, directoryItems, fileItems)
	}
}

// GetHubInfo 获取目录的 Hub 信息
func (s *ServiceTreeService) GetHubInfo(ctx context.Context, req *dto.GetHubInfoReq) (*dto.GetHubInfoResp, error) {
	// 1. 根据 FullCodePath 获取 ServiceTree 节点
	tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.FullCodePath)
	if err != nil {
		return nil, fmt.Errorf("获取目录信息失败: %w", err)
	}

	// 2. 检查是否已发布到 Hub
	if tree.HubDirectoryID == 0 {
		return nil, fmt.Errorf("目录未发布到 Hub")
	}

	// 3. 调用 Hub API 获取目录信息（用于获取 URL 和发布时间）
	header := &apicall.Header{
		TraceID:     contextx.GetTraceId(ctx),
		RequestUser: contextx.GetRequestUser(ctx),
		Token:       contextx.GetToken(ctx),
	}

		hubDetail, err := apicall.GetHubDirectoryDetail(header, req.FullCodePath, "", false, false)
	if err != nil {
		logger.Warnf(ctx, "[GetHubInfo] 获取 Hub 目录详情失败: fullCodePath=%s, error=%v", req.FullCodePath, err)
		// 即使获取详情失败，也返回基本信息
		return &dto.GetHubInfoResp{
			HubDirectoryID:  tree.HubDirectoryID,
			HubDirectoryURL: fmt.Sprintf("/hub/directory/%d", tree.HubDirectoryID),
			PublishedAt:     "", // 无法获取发布时间
		}, nil
	}

	// 4. 构建 Hub URL（使用 full-code-path）
	hubURL := fmt.Sprintf("/hub/directory/%s", req.FullCodePath)

	return &dto.GetHubInfoResp{
		HubDirectoryID:  hubDetail.ID,
		HubDirectoryURL: hubURL,
		PublishedAt:     hubDetail.PublishedAt,
	}, nil
}
