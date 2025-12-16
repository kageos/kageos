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
	serviceTreeRepo  *repository.ServiceTreeRepository
	appRepo          *repository.AppRepository
	appRuntime       *AppRuntime
	fileSnapshotRepo *repository.FileSnapshotRepository
	appService       *AppService
}

// NewServiceTreeService 创建服务目录服务
func NewServiceTreeService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	appRuntime *AppRuntime,
	fileSnapshotRepo *repository.FileSnapshotRepository,
	appService *AppService,
) *ServiceTreeService {
	return &ServiceTreeService{
		serviceTreeRepo:  serviceTreeRepo,
		appRepo:          appRepo,
		appRuntime:       appRuntime,
		fileSnapshotRepo: fileSnapshotRepo,
		appService:       appService,
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
func (s *ServiceTreeService) UpdateServiceTree(ctx context.Context, req *dto.UpdateServiceTreeReq) error {
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
// 返回：map[目录路径][]文件快照
func (s *ServiceTreeService) GetDirectorySnapshotsRecursively(ctx context.Context, appID int64, rootDirectoryPath string) (map[string][]*model.FileSnapshot, error) {
	// 1. 获取根目录的文件快照
	rootFiles, err := s.fileSnapshotRepo.GetCurrentVersionByDirectory(appID, rootDirectoryPath, s.serviceTreeRepo)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果根目录快照不存在，返回空 map（可能是新目录，还没有快照）
			logger.Warnf(ctx, "[ServiceTreeService] 根目录快照不存在: path=%s", rootDirectoryPath)
			return make(map[string][]*model.FileSnapshot), nil
		}
		return nil, fmt.Errorf("获取根目录快照失败: %w", err)
	}

	result := make(map[string][]*model.FileSnapshot)
	result[rootDirectoryPath] = rootFiles

	// 2. 递归查询所有子目录
	descendants, err := s.serviceTreeRepo.GetDescendantDirectories(appID, rootDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("查询子目录失败: %w", err)
	}

	// 3. 批量读取所有子目录的文件快照
	for _, dir := range descendants {
		files, err := s.fileSnapshotRepo.GetCurrentVersionByDirectory(appID, dir.FullCodePath, s.serviceTreeRepo)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				logger.Warnf(ctx, "[ServiceTreeService] 子目录快照不存在: path=%s", dir.FullCodePath)
				result[dir.FullCodePath] = []*model.FileSnapshot{} // 空列表
				continue                                           // 跳过没有快照的目录，继续处理其他目录
			}
			return nil, fmt.Errorf("获取子目录快照失败 (%s): %w", dir.FullCodePath, err)
		}
		result[dir.FullCodePath] = files
	}

	totalFiles := 0
	for _, files := range result {
		totalFiles += len(files)
	}

	logger.Infof(ctx, "[ServiceTreeService] 递归获取快照完成: 根目录=%s, 子目录数=%d, 总文件数=%d",
		rootDirectoryPath, len(descendants), totalFiles)

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

	// 创建目标目录的 ServiceTree 记录
	// 用户 copy 时，目标路径（包括子目录）一定不存在，直接递归创建即可
	targetTreeMap := make(map[string]*model.ServiceTree) // targetPath -> ServiceTree
	// 先将目标根目录放入 map，这样查找父目录时能找到它
	targetTreeMap[targetRootTree.FullCodePath] = targetRootTree
	currentVersionNum := extractVersionNumForServiceTree(targetApp.Version)

	for _, dirInfo := range dirsToCreate {
		// 计算父目录路径
		targetPathParts := strings.Split(strings.Trim(dirInfo.targetPath, "/"), "/")
		var parentTree *model.ServiceTree
		if len(targetPathParts) > 3 {
			// 有父目录，查找父目录的 ServiceTree（应该已经在 targetTreeMap 中）
			parentPath := "/" + strings.Join(targetPathParts[:len(targetPathParts)-1], "/")
			parentTree = targetTreeMap[parentPath]
			if parentTree == nil {
				return nil, fmt.Errorf("找不到父目录: path=%s, parentPath=%s", dirInfo.targetPath, parentPath)
			}
		} else {
			// 根目录，使用目标根目录作为父目录
			parentTree = targetRootTree
		}

		// 提取目录代码（路径的最后一部分）
		dirCode := targetPathParts[len(targetPathParts)-1]

		// 创建新的 ServiceTree 记录（保持源目录的名称、描述等信息）
		var parentID int64 = 0
		if parentTree != nil {
			parentID = parentTree.ID
		}
		newTree := &model.ServiceTree{
			Name:             dirInfo.sourceTree.Name, // 保持相同的名称
			Code:             dirCode,
			ParentID:         parentID,
			Type:             model.ServiceTreeTypePackage,
			Description:      dirInfo.sourceTree.Description, // 保持相同的描述
			Tags:             dirInfo.sourceTree.Tags,        // 保持相同的标签
			AppID:            targetApp.ID,
			FullCodePath:     dirInfo.targetPath,
			AddVersionNum:    currentVersionNum,
			UpdateVersionNum: 0,
		}

		// 保存到数据库
		if err := s.serviceTreeRepo.CreateServiceTreeWithParentPath(newTree, ""); err != nil {
			return nil, fmt.Errorf("创建目标目录失败 (%s): %w", dirInfo.targetPath, err)
		}

		// 发送NATS消息给app-runtime创建目录结构和 init_.go 文件
		if err := s.sendCreateServiceTreeMessage(ctx, targetApp.User, targetApp.Code, newTree); err != nil {
			logger.Errorf(ctx, "[ServiceTreeService] 发送创建目录消息失败: %v", err)
			return nil, fmt.Errorf("创建目录结构失败 (%s): %w", dirInfo.targetPath, err)
		}

		targetTreeMap[dirInfo.targetPath] = newTree
		logger.Infof(ctx, "[ServiceTreeService] 创建目标目录: source=%s, target=%s, name=%s",
			dirInfo.sourcePath, dirInfo.targetPath, newTree.Name)
	}

	// 更新目标路径（因为根目录会在目标根目录下创建同名目录）
	// 例如：从 /user/app/a/b 复制到 /user/app/e，实际目标路径应该是 /user/app/e/b
	if len(dirsToCreate) > 0 {
		// 使用第一个目录（根目录）的目标路径作为新的目标根路径
		targetRootPath = dirsToCreate[0].targetPath
	}

	// 9. 构建 CreateFunctions 请求（新版本：直接复制文件，不替换 package）
	// 因为新版本的 copy 是复制整个目录，package 路径已经正确了，不需要替换
	createFunctions := make([]*dto.CreateFunctionInfo, 0)
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

		// 提取目标 package 路径（去掉应用前缀）
		targetPackage := extractPackageFromPath(targetPath)

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

			// 构建 CreateFunctionInfo（直接使用快照中的代码，不替换 package）
			createFunctions = append(createFunctions, &dto.CreateFunctionInfo{
				Package:    targetPackage,
				GroupCode:  fileName,             // 使用 FileName 作为 GroupCode
				SourceCode: fileSnapshot.Content, // 直接使用快照中的代码，package 路径已经正确
			})
			totalFileCount++
		}

		logger.Infof(ctx, "[ServiceTreeService] 准备复制目录: source=%s, target=%s, fileCount=%d",
			sourcePath, targetPath, len(fileSnapshots))
	}

	// 10. 调用 UpdateApp（使用 CreateFunctions，不替换 package）
	logger.Infof(ctx, "[ServiceTreeService] 准备调用 UpdateApp（复制目录）: functionCount=%d", len(createFunctions))

	updateReq := &dto.UpdateAppReq{
		User:            targetApp.User,
		App:             targetApp.Code,
		CreateFunctions: createFunctions, // 使用 CreateFunctions，直接复制文件，不替换 package
	}

	_, err = s.appService.UpdateApp(ctx, updateReq)
	if err != nil {
		logger.Errorf(ctx, "[ServiceTreeService] UpdateApp（复制目录）失败: error=%v", err)
		return nil, fmt.Errorf("UpdateApp 失败: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] 复制目录完成: 目录数=%d, 文件数=%d", len(directoryFiles), totalFileCount)

	return &dto.CopyDirectoryResp{
		Message:        fmt.Sprintf("复制目录成功，共复制 %d 个目录，%d 个文件", len(directoryFiles), totalFileCount),
		DirectoryCount: len(directoryFiles),
		FileCount:      totalFileCount,
	}, nil
}
