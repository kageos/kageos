package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/appcall"
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

func (s *ServiceTreeService) assignAdminRoleToUser(ctx context.Context, user, app, username, resourcePath string) error {
	return assignDirectoryAdminRoleToUser(ctx, user, app, username, resourcePath)
}

func (s *ServiceTreeService) removeAdminRoleFromUser(ctx context.Context, resourcePath, username string) error {
	return removeDirectoryAdminRoleFromUser(ctx, resourcePath, username)
}

func (s *ServiceTreeService) removeAdminRoleFromUserWithUserApp(ctx context.Context, user, app, username, resourcePath string) error {
	return removeDirectoryAdminRoleFromUserWithUserApp(ctx, user, app, username, resourcePath)
}

type ServiceTreeService struct {
	serviceTreeRepo    *repository.ServiceTreeRepository
	functionRepo       *repository.FunctionRepository // 用于 SearchFunctions 查函数表并返回 Request/Response
	appRepo            *repository.AppRepository
	fileSnapshotRepo   *repository.FileSnapshotRepository
	runtimeWorkspace   *runtimeWorkspaceBridge
	queryView          *serviceTreeQueryView
	hubService         *serviceTreeHubService
	mutationService    *serviceTreeMutationService
	specialNodeService *serviceTreeSpecialNodeService
	functionService    *serviceTreeFunctionService
	packageService     *PackageService
	appService         *AppService
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
	runtimeWorkspace := newRuntimeWorkspaceBridge(appRepo, appCall)
	queryView := newServiceTreeQueryView(serviceTreeRepo, appRepo, permissionService)
	hubService := newServiceTreeHubService(serviceTreeRepo, functionRepo, appRepo, runtimeWorkspace, appService)
	mutationService := newServiceTreeMutationService(serviceTreeRepo, appRepo, runtimeWorkspace, docService, boardPostRepo)
	specialNodeService := newServiceTreeSpecialNodeService(serviceTreeRepo, appRepo, docService)
	functionService := newServiceTreeFunctionService(serviceTreeRepo, appRepo, appService)
	serviceTreeService := &ServiceTreeService{
		serviceTreeRepo:    serviceTreeRepo,
		functionRepo:       functionRepo,
		appRepo:            appRepo,
		fileSnapshotRepo:   fileSnapshotRepo,
		runtimeWorkspace:   runtimeWorkspace,
		queryView:          queryView,
		hubService:         hubService,
		mutationService:    mutationService,
		specialNodeService: specialNodeService,
		functionService:    functionService,
		appService:         appService,
	}

	serviceTreeService.packageService = NewPackageService(serviceTreeRepo, appRepo, runtimeWorkspace)
	return serviceTreeService
}

// CreatePackage 创建 package 类型节点（专门的接口）
func (s *ServiceTreeService) CreatePackage(ctx context.Context, req *dto.CreatePackageReq) (*dto.CreatePackageResp, error) {
	return s.packageService.CreatePackage(ctx, req)
}

// CreateFunction 创建 function 类型节点（专门的接口）
func (s *ServiceTreeService) CreateFunction(ctx context.Context, req *dto.CreateFunctionReq) (*dto.CreateFunctionResp, error) {
	return s.functionService.CreateFunction(ctx, req)
}

// getServiceTreeByAppModel 根据 appModel 获取服务目录树（内部方法，避免重复获取 appModel）
// ⭐ 新架构：工作空间根节点也在 service_tree 表中，统一查询和权限处理
func (s *ServiceTreeService) getServiceTreeByAppModel(ctx context.Context, appModel *model.App, nodeType string) ([]*dto.GetServiceTreeResp, error) {
	return s.queryView.getServiceTreeByAppModel(ctx, appModel, nodeType)
}

// filterTreeByPermission 按权限过滤服务树：只保留有权限的节点，无权限的父节点不展示，有权限的深层节点提升到根下。
// 根节点始终保留；其直接子节点 = 递归时「有权限则保留该节点并递归其子，无权限则丢弃该节点本身、将其下有权限的节点提升上来」。
func (s *ServiceTreeService) filterTreeByPermission(rootResp *dto.GetServiceTreeResp) *dto.GetServiceTreeResp {
	return s.queryView.filterTreeByPermission(rootResp)
}

// collectVisibleChildren 对一组子节点做过滤并提升：有权限的节点保留且递归其子；无权限的节点不保留，其下有权限的节点提升到当前层。
// 返回应挂到父节点下的节点列表。
func (s *ServiceTreeService) collectVisibleChildren(children []*dto.GetServiceTreeResp) []*dto.GetServiceTreeResp {
	return s.queryView.collectVisibleChildren(children)
}

// collectVisibleFromNode 对单个节点：若有权限则返回 [该节点]（其 Children 已递归过滤并提升）；若无权限则返回其子树中所有可见节点（提升到当前层）。
func (s *ServiceTreeService) collectVisibleFromNode(node *dto.GetServiceTreeResp) []*dto.GetServiceTreeResp {
	return s.queryView.collectVisibleFromNode(node)
}

// calculateTotalPendingCount 递归计算节点及其所有子节点的 pending_count 总和
func calculateTotalPendingCount(node *dto.GetServiceTreeResp) int {
	total := node.PendingCount
	for _, child := range node.Children {
		total += calculateTotalPendingCount(child)
	}
	return total
}

// GetAppWithServiceTree 获取应用详情和服务目录树（合并接口，减少请求次数）
// 这个方法放在 ServiceTreeService 中，因为：
// 1. ServiceTreeService 已经有 appService 依赖，可以直接调用
// 2. 避免 AppService 和 ServiceTreeService 之间的循环依赖
// 3. 职责清晰：ServiceTreeService 负责服务目录树相关的所有操作，包括组合操作
// 优化：只获取一次 appModel，避免重复查询数据库
func (s *ServiceTreeService) GetAppWithServiceTree(ctx context.Context, req *dto.GetAppWithServiceTreeReq) (*dto.GetAppWithServiceTreeResp, error) {
	return s.queryView.GetAppWithServiceTree(ctx, req)
}

// GetServiceTreeDetail 获取服务目录详情（包含权限信息）
// ⭐ 优化：按需查询权限，只在获取详情时查询
func (s *ServiceTreeService) GetServiceTreeDetail(ctx context.Context, req *dto.GetServiceTreeDetailReq) (*dto.GetServiceTreeDetailResp, error) {
	return s.queryView.GetServiceTreeDetail(ctx, req)
}

// convertToGetServiceTreeResp 转换为响应格式（包含权限信息）
// ⭐ 优化：在服务树中直接返回权限信息，一次性获取所有权限（只需要8ms）
// ⭐ 父子关系由 FullCodePath 推导，无需 ParentID
func (s *ServiceTreeService) convertToGetServiceTreeResp(ctx context.Context, tree *model.ServiceTree, permissionsMap map[string]map[string]bool, isAdmin bool) *dto.GetServiceTreeResp {
	return s.queryView.convertToGetServiceTreeResp(ctx, tree, permissionsMap, isAdmin)
}

// calculateExpandedKeys 计算需要自动展开的节点ID列表
// ⭐ 包含所有 pending_count > 0 的节点及其所有父节点
func (s *ServiceTreeService) calculateExpandedKeys(trees []*dto.GetServiceTreeResp) []int64 {
	return s.queryView.calculateExpandedKeys(trees)
}

// calculatePermissions 计算权限（内部方法）
// ⭐ 优先检查 app.Admins 字段，如果当前用户在管理员列表中，直接返回所有权限
// ⭐ 否则使用工作空间权限记录 + 继承规则自顶向下计算节点权限
func (s *ServiceTreeService) calculatePermissions(ctx context.Context, user, app string, trees []*model.ServiceTree, admins string, username string) (map[string]map[string]bool, error) {
	return s.queryView.calculatePermissions(ctx, user, app, trees, admins, username)
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
	s.queryView.applyPermissionInheritance(nodeType, templateType, parentPerms, nodePerms)
}

// hasFunctionInDirectChildren 只检查直接子节点是否有 function 类型（不递归）
func (s *ServiceTreeService) hasFunctionInDirectChildren(node *model.ServiceTree) bool {
	return s.queryView.hasFunctionInDirectChildren(node)
}

// getPermissionActionsForNode 根据节点类型和模板类型，获取需要检查的权限点
// ⭐ 优化：使用公共函数，避免代码重复
func (s *ServiceTreeService) getPermissionActionsForNode(nodeType string, templateType string) []string {
	return s.queryView.getPermissionActionsForNode(nodeType, templateType)
}

// GetServiceTreeByFullPath 根据完整路径获取服务目录（用于权限检查）
func (s *ServiceTreeService) GetServiceTreeByFullPath(ctx context.Context, fullPath string) (*model.ServiceTree, error) {
	return s.queryView.GetServiceTreeByFullPath(fullPath)
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

// CopyServiceTree 复制服务目录（递归复制目录及其所有子目录）
// 支持两种模式：
// 1. 本地复制：从本地工作空间复制目录
// 2. Hub 复制：从 Hub 链接复制目录（自动检测 hub:// 前缀）
func (s *ServiceTreeService) CopyServiceTree(ctx context.Context, req *dto.CopyDirectoryReq) (*dto.CopyDirectoryResp, error) {
	return s.hubService.CopyServiceTree(ctx, req)
}

// PublishDirectoryToHub 发布目录到 Hub
func (s *ServiceTreeService) PublishDirectoryToHub(ctx context.Context, req *dto.PublishDirectoryToHubReq) (*dto.PublishDirectoryToHubResp, error) {
	return s.hubService.PublishDirectoryToHub(ctx, req)
}

// PushDirectoryToHub 推送目录到 Hub（更新已发布的目录，类似 git push）
func (s *ServiceTreeService) PushDirectoryToHub(ctx context.Context, req *dto.PushDirectoryToHubReq) (*dto.PushDirectoryToHubResp, error) {
	return s.hubService.PushDirectoryToHub(ctx, req)
}

// GetHubPushFormInfo 获取推送表单信息（当前已发布信息 + 下一版本号，用于推送对话框预填）
func (s *ServiceTreeService) GetHubPushFormInfo(ctx context.Context, req *dto.GetHubPushFormInfoReq) (*dto.GetHubPushFormInfoResp, error) {
	return s.hubService.GetHubPushFormInfo(ctx, req)
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
	return s.functionService.AddFunctions(ctx, req)
}

// ProcessFunctionGenResult 处理函数生成结果（接收 agent-server 处理后的结构化数据）
// 目录由 full_code_path 指定，文件名由 file_name 指定（或 fallback 为目录 Code）；不再解析代码内元数据
func (s *ServiceTreeService) ProcessFunctionGenResult(ctx context.Context, req *dto.AddFunctionsReq) error {
	return s.functionService.ProcessFunctionGenResult(ctx, req)
}

// PullDirectoryFromHub 从 Hub 拉取目录到工作空间（类似 git pull）
func (s *ServiceTreeService) PullDirectoryFromHub(ctx context.Context, req *dto.PullDirectoryFromHubReq) (*dto.PullDirectoryFromHubResp, error) {
	return s.hubService.PullDirectoryFromHub(ctx, req)
}

// ImportHubDirectoryBundle 从离线 JSON 包安装目录（不访问 Hub，不增加下载次数）
func (s *ServiceTreeService) ImportHubDirectoryBundle(ctx context.Context, req *dto.ImportHubDirectoryBundleReq) (*dto.PullDirectoryFromHubResp, error) {
	return s.hubService.ImportHubDirectoryBundle(ctx, req)
}

// BatchWriteFiles 批量写文件（app-server 端，调用 runtime）
func (s *ServiceTreeService) BatchWriteFiles(ctx context.Context, req *dto.BatchWriteFilesReq) (*dto.BatchWriteFilesResp, error) {
	return batchWriteFilesImpl(s, ctx, req)
}

// GetHubInfo 获取目录的 Hub 信息
func (s *ServiceTreeService) GetHubInfo(ctx context.Context, req *dto.GetHubInfoReq) (*dto.GetHubInfoResp, error) {
	return s.hubService.GetHubInfo(ctx, req)
}

// SearchFunctions 搜索函数：查 ServiceTree（type=function），预加载 App、Function，返回带请求/响应参数的结果
// 当有关键词且为第一页时，会按关键词相关度重排，把最符合的结果排在前面
func (s *ServiceTreeService) SearchFunctions(ctx context.Context, req *dto.SearchFunctionsReq) (*dto.SearchFunctionsResp, error) {
	return searchFunctionsImpl(s, ctx, req)
}
