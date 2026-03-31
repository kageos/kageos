package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/appcall"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
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

// assignAdminRoleToUser 给用户分配管理员角色（目录节点）
// ⭐ 使用角色系统，分配"admin"角色（拥有 directory:admin 权限）
func (s *ServiceTreeService) assignAdminRoleToUser(ctx context.Context, user, app, username, resourcePath string) error {
	// 检查权限功能是否启用（企业版）
	licenseMgr := license.GetManager()
	if !licenseMgr.HasFeature(enterprise.FeaturePermission) {
		// 权限功能未启用，跳过
		return nil
	}

	// 获取权限服务
	permissionService := enterprise.GetPermissionService()
	if permissionService == nil {
		return fmt.Errorf("权限服务未初始化")
	}

	// ⭐ 使用角色系统，分配"admin"角色（拥有 directory:admin 权限）
	// 目录节点使用 directory 资源类型
	assignReq := &dto.AssignRoleToUserReq{
		Username:     username,
		RoleCode:     "admin",     // 管理员角色
		ResourceType: "directory", // ⭐ 目录节点使用 directory 资源类型
		ResourcePath: resourcePath,
		StartTime:    nil, // 永久权限
		EndTime:      nil, // 永久权限
	}

	_, err := permissionService.AssignRoleToUser(ctx, assignReq)
	if err != nil {
		return fmt.Errorf("分配管理员角色失败: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] 分配管理员角色成功: user=%s, app=%s, username=%s, resource=%s",
		user, app, username, resourcePath)
	return nil
}

// removeAdminRoleFromUser 移除用户的管理员角色（目录节点）
// ⭐ 从 resourcePath 自动解析 user 和 app
func (s *ServiceTreeService) removeAdminRoleFromUser(ctx context.Context, resourcePath, username string) error {
	// 从 resourcePath 解析 user 和 app
	parts := strings.Split(strings.Trim(resourcePath, "/"), "/")
	if len(parts) < 2 {
		return fmt.Errorf("无法从资源路径解析 user 和 app: %s", resourcePath)
	}
	user := parts[0]
	app := parts[1]

	return s.removeAdminRoleFromUserWithUserApp(ctx, user, app, username, resourcePath)
}

// removeAdminRoleFromUserWithUserApp 移除用户的管理员角色（目录节点，带 user 和 app 参数）
func (s *ServiceTreeService) removeAdminRoleFromUserWithUserApp(ctx context.Context, user, app, username, resourcePath string) error {
	// 检查权限功能是否启用（企业版）
	licenseMgr := license.GetManager()
	if !licenseMgr.HasFeature(enterprise.FeaturePermission) {
		// 权限功能未启用，跳过
		return nil
	}

	// 获取权限服务
	permissionService := enterprise.GetPermissionService()
	if permissionService == nil {
		return fmt.Errorf("权限服务未初始化")
	}

	// ⭐ 使用角色系统，移除"admin"角色
	// 目录节点使用 directory 资源类型
	removeReq := &dto.RemoveRoleFromUserReq{
		Username:     username,
		RoleCode:     "admin",     // 管理员角色
		ResourceType: "directory", // ⭐ 目录节点使用 directory 资源类型
		ResourcePath: resourcePath,
	}

	err := permissionService.RemoveRoleFromUser(ctx, removeReq)
	if err != nil {
		return fmt.Errorf("移除管理员角色失败: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] 移除管理员角色成功: user=%s, app=%s, username=%s, resource=%s",
		user, app, username, resourcePath)
	return nil
}

type ServiceTreeService struct {
	serviceTreeRepo   *repository.ServiceTreeRepository
	functionRepo      *repository.FunctionRepository // 用于 SearchFunctions 查函数表并返回 Request/Response
	appRepo           *repository.AppRepository
	appCall           *appcall.Client
	fileSnapshotRepo  *repository.FileSnapshotRepository
	packageService    *PackageService
	appService        *AppService
	permissionService *PermissionService              // ⭐ 添加 PermissionService 依赖，用于查询权限
	docService        *DocService                     // ⭐ 添加 DocService 依赖，用于创建文档内容
	boardPostRepo     *repository.BoardPostRepository // 版块帖子，删版块时需先删帖子
}

// NewServiceTreeService 创建服务目录服务
func NewServiceTreeService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	functionRepo *repository.FunctionRepository,
	appRepo *repository.AppRepository,
	appCall *appcall.Client,
	fileSnapshotRepo *repository.FileSnapshotRepository,
	appService *AppService,
	permissionService *PermissionService, // ⭐ 新增 PermissionService 依赖
	docService *DocService, // ⭐ 新增 DocService 依赖
	boardPostRepo *repository.BoardPostRepository, // 版块帖子，删版块时需先删帖子
) *ServiceTreeService {
	serviceTreeService := &ServiceTreeService{
		serviceTreeRepo:   serviceTreeRepo,
		functionRepo:      functionRepo,
		appRepo:           appRepo,
		appCall:           appCall,
		fileSnapshotRepo:  fileSnapshotRepo,
		appService:        appService,
		permissionService: permissionService,
		docService:        docService,
		boardPostRepo:     boardPostRepo,
	}

	serviceTreeService.packageService = NewPackageService(serviceTreeRepo, appRepo, appCall)
	return serviceTreeService
}

// CreateServiceTree 创建服务目录（package 类型）
func (s *ServiceTreeService) CreateServiceTree(ctx context.Context, req *dto.CreateServiceTreeReq) (*dto.CreateServiceTreeResp, error) {
	// ⭐ 如果指定了类型为 docs，则调用专门的方法
	if req.Type == model.ServiceTreeTypeDocs {
		return s.CreateDocsNode(ctx, req)
	}
	// ⭐ 如果指定了类型为 board，则调用专门的方法
	if req.Type == model.ServiceTreeTypeBoard {
		return s.CreateBoardNode(ctx, req)
	}

	packageResp, err := s.packageService.CreatePackage(ctx, &dto.CreatePackageReq{
		User:               req.User,
		App:                req.App,
		Name:               req.Name,
		Code:               req.Code,
		ParentFullCodePath: req.ParentFullCodePath,
		Description:        req.Description,
		Tags:               req.Tags,
		Admins:             req.Admins,
	})
	if err != nil {
		return nil, err
	}

	return &dto.CreateServiceTreeResp{
		ID:           packageResp.ID,
		Name:         packageResp.Name,
		Code:         packageResp.Code,
		Type:         packageResp.Type,
		Description:  packageResp.Description,
		Tags:         packageResp.Tags,
		Admins:       packageResp.Admins,
		AppID:        packageResp.AppID,
		FullCodePath: packageResp.FullCodePath,
		Version:      packageResp.Version,
		VersionNum:   packageResp.VersionNum,
		Status:       "created",
	}, nil
}

// CreatePackage 创建 package 类型节点（专门的接口）
func (s *ServiceTreeService) CreatePackage(ctx context.Context, req *dto.CreatePackageReq) (*dto.CreatePackageResp, error) {
	return s.packageService.CreatePackage(ctx, req)
}

// CreateFunction 创建 function 类型节点（专门的接口）
func (s *ServiceTreeService) CreateFunction(ctx context.Context, req *dto.CreateFunctionReq) (*dto.CreateFunctionResp, error) {
	// 根据 directory_path 获取父目录
	parentTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.DirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取父目录失败: %w", err)
	}

	// 预加载 App 信息（如果还没有加载）
	if parentTree.App == nil {
		app, err := s.appRepo.GetAppByID(parentTree.AppID)
		if err != nil {
			return nil, fmt.Errorf("获取应用信息失败: %w", err)
		}
		parentTree.App = app
	}

	// 构建 AddFunctionsReq（复用现有逻辑）；租户由 AddFunctions 内从 full_code_path 解析
	addFunctionsReq := &dto.AddFunctionsReq{
		FullCodePath: req.DirectoryPath,
		FileName:     req.Code, // 使用 code 作为文件名
		SourceCode:   req.SourceCode,
	}

	// 调用 AddFunctions 方法
	addResp, err := s.AddFunctions(ctx, addFunctionsReq)
	if err != nil {
		return nil, fmt.Errorf("创建函数失败: %w", err)
	}

	if !addResp.Success {
		return nil, fmt.Errorf("创建函数失败: %s", addResp.Error)
	}

	// 获取创建的 function 节点（通过路径查找）
	expectedPath := req.DirectoryPath + "/" + req.Code
	functionTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(expectedPath)
	if err != nil {
		// 如果找不到，返回基本信息（函数可能已创建但 ServiceTree 记录可能还未同步）
		// 这种情况可能发生在异步处理时，返回基本信息让调用方知道函数已创建
		logger.Warnf(ctx, "[CreateFunction] 无法通过路径查找函数节点: %s, error: %v，返回基本信息", expectedPath, err)
		return &dto.CreateFunctionResp{
			ID:           0,
			Name:         req.Name,
			Code:         req.Code,
			Type:         model.ServiceTreeTypeFunction,
			TemplateType: req.TemplateType,
			Description:  req.Description,
			Tags:         req.Tags,
			AppID:        parentTree.AppID,
			FullCodePath: expectedPath,
			Version:      "v1",
			VersionNum:   1,
		}, nil
	}

	// 转换为专门的响应格式
	fnResp := &dto.CreateFunctionResp{
		ID:           functionTree.ID,
		Name:         functionTree.Name,
		Code:         functionTree.Code,
		Type:         functionTree.Type,
		TemplateType: functionTree.TemplateType,
		Description:  functionTree.Description,
		Tags:         functionTree.Tags,
		AppID:        functionTree.AppID,
		RefID:        functionTree.RefID,
		FullCodePath: functionTree.FullCodePath,
		Version:      functionTree.Version,
		VersionNum:   functionTree.VersionNum,
	}
	return fnResp, nil
}

// getServiceTreeByAppModel 根据 appModel 获取服务目录树（内部方法，避免重复获取 appModel）
// ⭐ 新架构：工作空间根节点也在 service_tree 表中，统一查询和权限处理
func (s *ServiceTreeService) getServiceTreeByAppModel(ctx context.Context, appModel *model.App, nodeType string) ([]*dto.GetServiceTreeResp, error) {
	return getServiceTreeByAppModelImpl(s, ctx, appModel, nodeType)
}

// filterTreeByPermission 按权限过滤服务树：只保留有权限的节点，无权限的父节点不展示，有权限的深层节点提升到根下。
// 根节点始终保留；其直接子节点 = 递归时「有权限则保留该节点并递归其子，无权限则丢弃该节点本身、将其下有权限的节点提升上来」。
func (s *ServiceTreeService) filterTreeByPermission(rootResp *dto.GetServiceTreeResp) *dto.GetServiceTreeResp {
	// 根节点（路径段数=2，即 /user/app）始终保留；其 Children 由 collectVisibleChildren 对每个子节点收集并合并
	rootResp.Children = s.collectVisibleChildren(rootResp.Children)
	return rootResp
}

// collectVisibleChildren 对一组子节点做过滤并提升：有权限的节点保留且递归其子；无权限的节点不保留，其下有权限的节点提升到当前层。
// 返回应挂到父节点下的节点列表。
func (s *ServiceTreeService) collectVisibleChildren(children []*dto.GetServiceTreeResp) []*dto.GetServiceTreeResp {
	out := make([]*dto.GetServiceTreeResp, 0, len(children))
	for _, child := range children {
		out = append(out, s.collectVisibleFromNode(child)...)
	}
	return out
}

// collectVisibleFromNode 对单个节点：若有权限则返回 [该节点]（其 Children 已递归过滤并提升）；若无权限则返回其子树中所有可见节点（提升到当前层）。
func (s *ServiceTreeService) collectVisibleFromNode(node *dto.GetServiceTreeResp) []*dto.GetServiceTreeResp {
	hasAnyTrue := false
	if node.Permissions != nil {
		for _, v := range node.Permissions {
			if v {
				hasAnyTrue = true
				break
			}
		}
	}
	if !hasAnyTrue {
		// 无权限：不保留当前节点，把其子节点中可见的提升上来
		return s.collectVisibleChildren(node.Children)
	}
	// 有权限：保留当前节点，其 Children 为子节点过滤并提升后的结果
	node.Children = s.collectVisibleChildren(node.Children)
	return []*dto.GetServiceTreeResp{node}
}

// calculateTotalPendingCount 递归计算节点及其所有子节点的 pending_count 总和
func calculateTotalPendingCount(node *dto.GetServiceTreeResp) int {
	total := node.PendingCount
	for _, child := range node.Children {
		total += calculateTotalPendingCount(child)
	}
	return total
}

// GetServiceTree 获取服务目录
func (s *ServiceTreeService) GetServiceTree(ctx context.Context, user, app string, nodeType string) ([]*dto.GetServiceTreeResp, error) {
	appModel, err := s.appRepo.GetAppByUserName(user, app)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}
	return s.getServiceTreeByAppModel(ctx, appModel, nodeType)
}

// GetAppWithServiceTree 获取应用详情和服务目录树（合并接口，减少请求次数）
// 这个方法放在 ServiceTreeService 中，因为：
// 1. ServiceTreeService 已经有 appService 依赖，可以直接调用
// 2. 避免 AppService 和 ServiceTreeService 之间的循环依赖
// 3. 职责清晰：ServiceTreeService 负责服务目录树相关的所有操作，包括组合操作
// 优化：只获取一次 appModel，避免重复查询数据库
func (s *ServiceTreeService) GetAppWithServiceTree(ctx context.Context, req *dto.GetAppWithServiceTreeReq) (*dto.GetAppWithServiceTreeResp, error) {
	return getAppWithServiceTreeImpl(s, ctx, req)
}

// GetServiceTreeDetail 获取服务目录详情（包含权限信息）
// ⭐ 优化：按需查询权限，只在获取详情时查询
func (s *ServiceTreeService) GetServiceTreeDetail(ctx context.Context, req *dto.GetServiceTreeDetailReq) (*dto.GetServiceTreeDetailResp, error) {
	return getServiceTreeDetailImpl(s, ctx, req)
}

// GetPackageInfo 获取目录信息（仅用于获取目录权限，不包含函数）
// ⭐ 优化：专门用于获取目录权限，函数权限从函数详情接口获取
func (s *ServiceTreeService) GetPackageInfo(ctx context.Context, req *dto.GetPackageInfoReq) (*dto.GetPackageInfoResp, error) {
	return getPackageInfoImpl(s, ctx, req)
}

// convertToGetServiceTreeResp 转换为响应格式（包含权限信息）
// ⭐ 优化：在服务树中直接返回权限信息，一次性获取所有权限（只需要8ms）
// ⭐ 父子关系由 FullCodePath 推导，无需 ParentID
func (s *ServiceTreeService) convertToGetServiceTreeResp(ctx context.Context, tree *model.ServiceTree, permissionsMap map[string]map[string]bool, isAdmin bool) *dto.GetServiceTreeResp {
	return convertToGetServiceTreeRespImpl(s, ctx, tree, permissionsMap, isAdmin)
}

// calculateExpandedKeys 计算需要自动展开的节点ID列表
// ⭐ 包含所有 pending_count > 0 的节点及其所有父节点
func (s *ServiceTreeService) calculateExpandedKeys(trees []*dto.GetServiceTreeResp) []int64 {
	return calculateExpandedKeysImpl(trees)
}

// calculatePermissions 计算权限（内部方法）
// ⭐ 优先检查 app.Admins 字段，如果当前用户在管理员列表中，直接返回所有权限
// ⭐ 否则使用工作空间权限记录 + 继承规则自顶向下计算节点权限
func (s *ServiceTreeService) calculatePermissions(ctx context.Context, user, app string, trees []*model.ServiceTree, admins string, username string) (map[string]map[string]bool, error) {
	return calculatePermissionsImpl(s, ctx, user, app, trees, admins, username)
}

// applyPermissionInheritance 应用权限继承规则
// ⭐ 新权限系统实现权限继承逻辑
// applyPermissionInheritance 应用权限继承规则（新格式：resource_type:action_type）
// ⭐ 权限点格式：resource_type:action_type（如 directory:read, table:write）
// 目录权限继承：directory:read -> table:read（需要转换）
func (s *ServiceTreeService) applyPermissionInheritance(
	nodeType string,
	templateType string,
	parentPerms map[string]bool, // 父目录的权限（格式：actionCode -> true，如 directory:read -> true）
	nodePerms map[string]bool, // 子节点的权限（格式：actionCode -> true，如 table:read -> true）
) {
	applyPermissionInheritanceImpl(nodeType, templateType, parentPerms, nodePerms)
}

// hasFunctionInDirectChildren 只检查直接子节点是否有 function 类型（不递归）
func (s *ServiceTreeService) hasFunctionInDirectChildren(node *model.ServiceTree) bool {
	if node == nil {
		return false
	}

	// 只检查直接子节点，不递归检查子目录的子节点
	for _, child := range node.Children {
		if child.Type == model.ServiceTreeTypeFunction {
			return true
		}
	}

	return false
}

// getPermissionActionsForNode 根据节点类型和模板类型，获取需要检查的权限点
// ⭐ 优化：使用公共函数，避免代码重复
func (s *ServiceTreeService) getPermissionActionsForNode(nodeType string, templateType string) []string {
	return getPermissionActionsForNodeImpl(nodeType, templateType)
}

// GetServiceTreeByFullPath 根据完整路径获取服务目录（用于权限检查）
func (s *ServiceTreeService) GetServiceTreeByFullPath(ctx context.Context, fullPath string) (*model.ServiceTree, error) {
	return s.serviceTreeRepo.GetServiceTreeByFullPath(fullPath)
}

// GetDirectorySnapshotsRecursively 递归获取目录及其所有子目录的文件快照
// GetDirectorySnapshotsRecursively 递归获取目录及其所有子目录的当前版本文件快照
// 优化：使用 ServiceTreeID 和 IsCurrent 字段，性能更好
// 返回：map[目录路径][]文件快照
func (s *ServiceTreeService) GetDirectorySnapshotsRecursively(ctx context.Context, appID int64, rootDirectoryPath string) (map[string][]*model.FileSnapshot, error) {
	return getDirectorySnapshotsRecursivelyImpl(s, ctx, appID, rootDirectoryPath)
}

// GetDirectoryFilesFromRuntimeRecursively 从 app-runtime 实时递归读取目录及其所有子目录下的文件（不依赖快照表）
// 返回：map[目录完整路径][]*FileSnapshot（仅填充 FileName、RelativePath、Content、FileType、FileVersion，用于 buildDirectoryTree）
func (s *ServiceTreeService) GetDirectoryFilesFromRuntimeRecursively(ctx context.Context, appID int64, rootDirectoryPath string) (map[string][]*model.FileSnapshot, error) {
	return getDirectoryFilesFromRuntimeRecursivelyImpl(s, ctx, appID, rootDirectoryPath)
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
// 支持两种模式：
// 1. 本地复制：从本地工作空间复制目录
// 2. Hub 复制：从 Hub 链接复制目录（自动检测 hub:// 前缀）
func (s *ServiceTreeService) CopyServiceTree(ctx context.Context, req *dto.CopyDirectoryReq) (*dto.CopyDirectoryResp, error) {
	return copyServiceTreeImpl(s, ctx, req)
}

// copyFromLocal 从本地工作空间复制目录
func (s *ServiceTreeService) copyFromLocal(ctx context.Context, req *dto.CopyDirectoryReq, targetApp *model.App) (*dto.CopyDirectoryResp, error) {
	return copyFromLocalImpl(s, ctx, req, targetApp)
}

// copyFromHub 从 Hub 链接复制目录
func (s *ServiceTreeService) copyFromHub(ctx context.Context, req *dto.CopyDirectoryReq, targetApp *model.App) (*dto.CopyDirectoryResp, error) {
	return copyFromHubImpl(s, ctx, req, targetApp)
}

// PublishDirectoryToHub 发布目录到 Hub
func (s *ServiceTreeService) PublishDirectoryToHub(ctx context.Context, req *dto.PublishDirectoryToHubReq) (*dto.PublishDirectoryToHubResp, error) {
	return publishDirectoryToHubImpl(s, ctx, req)
}

// PushDirectoryToHub 推送目录到 Hub（更新已发布的目录，类似 git push）
func (s *ServiceTreeService) PushDirectoryToHub(ctx context.Context, req *dto.PushDirectoryToHubReq) (*dto.PushDirectoryToHubResp, error) {
	return pushDirectoryToHubImpl(s, ctx, req)
}

// GetHubPushFormInfo 获取推送表单信息（当前已发布信息 + 下一版本号，用于推送对话框预填）
func (s *ServiceTreeService) GetHubPushFormInfo(ctx context.Context, req *dto.GetHubPushFormInfoReq) (*dto.GetHubPushFormInfoResp, error) {
	return getHubPushFormInfoImpl(s, ctx, req)
}

// BatchCreateDirectoryTree 批量创建目录树（用于 copy 和 pull from hub）
func (s *ServiceTreeService) BatchCreateDirectoryTree(
	ctx context.Context,
	req *dto.BatchCreateDirectoryTreeReq,
) (*dto.BatchCreateDirectoryTreeResp, error) {
	return batchCreateDirectoryTreeImpl(s, ctx, req)
}

// AddFunctions 向服务目录添加函数（同步处理）
// 目录由 full_code_path 指定，文件名由 file_name 指定（或 fallback 为目录 Code）；不再解析代码内元数据，由调用方（工作台/模型）负责目录与文件名
func (s *ServiceTreeService) AddFunctions(ctx context.Context, req *dto.AddFunctionsReq) (*dto.AddFunctionsResp, error) {
	return addFunctionsImpl(s, ctx, req)
}

// ProcessFunctionGenResult 处理函数生成结果（接收 agent-server 处理后的结构化数据）
// 目录由 full_code_path 指定，文件名由 file_name 指定（或 fallback 为目录 Code）；不再解析代码内元数据
func (s *ServiceTreeService) ProcessFunctionGenResult(ctx context.Context, req *dto.AddFunctionsReq) error {
	return processFunctionGenResultImpl(s, ctx, req)
}

// buildDirectoryTree 构建目录树结构（递归，包含函数及 request/response 等完整信息）
// rootTree: 根目录节点
// allTrees: 所有目录节点（包括根目录和子目录）
// directoryFiles: 目录路径到文件快照的映射
// idToTree: ServiceTreeID 到 ServiceTree 的映射
// functionMap: 目录ID到函数列表的映射
// refIDToFunction: RefID -> *Function，用于填充 request/response 等
func (s *ServiceTreeService) buildDirectoryTree(
	rootTree *model.ServiceTree,
	allTrees []*model.ServiceTree,
	directoryFiles map[string][]*model.FileSnapshot,
	idToTree map[int64]*model.ServiceTree,
	functionMap map[int64][]*model.ServiceTree,
	refIDToFunction map[int64]*model.Function,
) *dto.DirectoryTreeNode {
	return buildDirectoryTreeImpl(s, rootTree, allTrees, directoryFiles, idToTree, functionMap, refIDToFunction)
}

// buildDirectoryTreeNode 递归构建目录树节点（包含函数及 request/response 等完整信息）
func (s *ServiceTreeService) buildDirectoryTreeNode(
	tree *model.ServiceTree,
	allTrees []*model.ServiceTree,
	directoryFiles map[string][]*model.FileSnapshot,
	idToTree map[int64]*model.ServiceTree,
	functionMap map[int64][]*model.ServiceTree,
	refIDToFunction map[int64]*model.Function,
) *dto.DirectoryTreeNode {
	return buildDirectoryTreeNodeImpl(s, tree, allTrees, directoryFiles, idToTree, functionMap, refIDToFunction)
}

// installDirectoryTreeFromHubSnapshot 将 Hub 快照目录树安装到工作空间（批量建目录、写文件、可选 Hub 绑定）
func (s *ServiceTreeService) installDirectoryTreeFromHubSnapshot(
	ctx context.Context,
	tree *dto.DirectoryTreeNode,
	targetApp *model.App,
	targetPath string,
	hubFullCodePath string,
	hubVersionNum int,
	hubDirectoryName string,
	successMessagePrefix string,
) (*dto.PullDirectoryFromHubResp, error) {
	return installDirectoryTreeFromHubSnapshotImpl(s, ctx, tree, targetApp, targetPath, hubFullCodePath, hubVersionNum, hubDirectoryName, successMessagePrefix)
}

// PullDirectoryFromHub 从 Hub 拉取目录到工作空间（类似 git pull）
func (s *ServiceTreeService) PullDirectoryFromHub(ctx context.Context, req *dto.PullDirectoryFromHubReq) (*dto.PullDirectoryFromHubResp, error) {
	return pullDirectoryFromHubImpl(s, ctx, req)
}

// ImportHubDirectoryBundle 从离线 JSON 包安装目录（不访问 Hub，不增加下载次数）
func (s *ServiceTreeService) ImportHubDirectoryBundle(ctx context.Context, req *dto.ImportHubDirectoryBundleReq) (*dto.PullDirectoryFromHubResp, error) {
	return importHubDirectoryBundleImpl(s, ctx, req)
}

// BatchWriteFiles 批量写文件（app-server 端，调用 runtime）
func (s *ServiceTreeService) BatchWriteFiles(ctx context.Context, req *dto.BatchWriteFilesReq) (*dto.BatchWriteFilesResp, error) {
	return batchWriteFilesImpl(s, ctx, req)
}

// countFilesInTree 递归统计目录树中的文件数量（用于调试）
func (s *ServiceTreeService) countFilesInTree(node *dto.DirectoryTreeNode) int {
	return countFilesInTreeImpl(s, node)
}

// logDirectoryTree 递归打印目录树详细信息（用于调试）
func (s *ServiceTreeService) logDirectoryTree(ctx context.Context, node *dto.DirectoryTreeNode, level int) {
	logDirectoryTreeImpl(s, ctx, node, level)
}

// buildItemsFromTree 递归构建目录和文件项列表
func (s *ServiceTreeService) buildItemsFromTree(
	node *dto.DirectoryTreeNode,
	targetBasePath string,
	directoryItems *[]*dto.DirectoryTreeItem,
	fileItems *[]*dto.DirectoryTreeItem,
) {
	buildItemsFromTreeImpl(s, node, targetBasePath, directoryItems, fileItems)
}

// GetHubInfo 获取目录的 Hub 信息
func (s *ServiceTreeService) GetHubInfo(ctx context.Context, req *dto.GetHubInfoReq) (*dto.GetHubInfoResp, error) {
	return getHubInfoImpl(s, ctx, req)
}

// SearchFunctions 搜索函数：查 ServiceTree（type=function），预加载 App、Function，返回带请求/响应参数的结果
// 当有关键词且为第一页时，会按关键词相关度重排，把最符合的结果排在前面
func (s *ServiceTreeService) SearchFunctions(ctx context.Context, req *dto.SearchFunctionsReq) (*dto.SearchFunctionsResp, error) {
	return searchFunctionsImpl(s, ctx, req)
}
