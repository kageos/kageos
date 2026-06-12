package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"gorm.io/gorm"
)

type ServiceTreeRepository struct {
	db *gorm.DB
}

func NewServiceTreeRepository(db *gorm.DB) *ServiceTreeRepository {
	return &ServiceTreeRepository{db: db}
}

// Create 创建服务树节点（通用方法）
func (r *ServiceTreeRepository) Create(serviceTree *model.ServiceTree) error {
	return r.db.Create(serviceTree).Error
}

// GetDocsNodesByPathPrefix 根据路径前缀获取所有 docs 类型的后代节点
func (r *ServiceTreeRepository) GetDocsNodesByPathPrefix(appID int64, parentPath string) ([]*model.ServiceTree, error) {
	prefix := strings.TrimSuffix(parentPath, "/") + "/"
	var nodes []*model.ServiceTree
	err := r.db.Where("app_id = ? AND full_code_path LIKE ? AND type = ?",
		appID, prefix+"%", model.ServiceTreeTypeDocs).
		Order("created_at ASC").
		Find(&nodes).Error
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// CreateServiceTreeWithParentPath 创建服务目录
func (r *ServiceTreeRepository) CreateServiceTreeWithParentPath(serviceTree *model.ServiceTree, parentFullIDPath string) error {
	// 直接创建，不再计算FullIDPath
	return r.db.Create(serviceTree).Error
}

// CreateServiceTreeWithAppPrefix 创建带有用户应用前缀的服务目录
func (r *ServiceTreeRepository) CreateServiceTreeWithAppPrefix(serviceTree *model.ServiceTree, userAppPrefix string) error {
	// 先保存到数据库获取ID
	if err := r.db.Create(serviceTree).Error; err != nil {
		return err
	}

	// 然后计算包含用户应用前缀的路径信息
	if err := r.calculatePathsWithAppPrefix(serviceTree, userAppPrefix); err != nil {
		return fmt.Errorf("failed to calculate paths with app prefix: %w", err)
	}

	// 更新路径信息到数据库
	return r.db.Save(serviceTree).Error
}

// GetServiceTreeByID 根据ID获取服务目录
func (r *ServiceTreeRepository) GetServiceTreeByID(id int64) (*model.ServiceTree, error) {
	var serviceTree model.ServiceTree
	err := r.db.Where("id = ?", id).First(&serviceTree).Error
	if err != nil {
		return nil, err
	}
	return &serviceTree, nil
}

// GetServiceTreesByAppID 根据应用ID获取所有服务目录
func (r *ServiceTreeRepository) GetServiceTreesByAppID(appID int64) ([]*model.ServiceTree, error) {
	var serviceTrees []*model.ServiceTree
	err := r.db.Where("app_id = ?", appID).Order("id ASC").Find(&serviceTrees).Error
	if err != nil {
		return nil, err
	}
	return serviceTrees, nil
}

// GetRootNodeByAppID 获取应用的根节点
// 根节点特征：ref_id = app_id（不再依赖 parent_id）
func (r *ServiceTreeRepository) GetRootNodeByAppID(appID int64) (*model.ServiceTree, error) {
	var root model.ServiceTree
	err := r.db.Where("app_id = ? AND ref_id = ?", appID, appID).
		Preload("App").
		First(&root).Error
	if err != nil {
		return nil, err
	}
	return &root, nil
}

// GetServiceTreesByAppIDAndType 根据应用ID和类型获取服务目录
func (r *ServiceTreeRepository) GetServiceTreesByAppIDAndType(appID int64, nodeType string) ([]*model.ServiceTree, error) {
	var serviceTrees []*model.ServiceTree
	query := r.db.Where("app_id = ?", appID)
	if nodeType != "" {
		query = query.Where("type = ?", nodeType)
	}
	err := query.Order("id ASC").Find(&serviceTrees).Error
	if err != nil {
		return nil, err
	}
	return serviceTrees, nil
}

// GetServiceTreeChildren 获取直接子节点（基于路径前缀，只取下一层）
func (r *ServiceTreeRepository) GetServiceTreeChildren(parentID int64) ([]*model.ServiceTree, error) {
	parent, err := r.GetServiceTreeByID(parentID)
	if err != nil {
		return nil, fmt.Errorf("查询父节点失败: %w", err)
	}
	return r.GetDirectChildrenByPath(parent.AppID, parent.FullCodePath)
}

// GetDirectChildrenByPath 根据路径获取直接子节点（只有下一层深度）
func (r *ServiceTreeRepository) GetDirectChildrenByPath(appID int64, parentPath string) ([]*model.ServiceTree, error) {
	prefix := strings.TrimSuffix(parentPath, "/") + "/"
	var all []*model.ServiceTree
	err := r.db.Where("app_id = ? AND full_code_path LIKE ?", appID, prefix+"%").
		Preload("Function").
		Order("id ASC").
		Find(&all).Error
	if err != nil {
		return nil, err
	}
	// 过滤：只保留直接子节点（parentPath 段数 + 1）
	parentDepth := len(strings.Split(strings.Trim(parentPath, "/"), "/"))
	var children []*model.ServiceTree
	for _, node := range all {
		nodeDepth := len(strings.Split(strings.Trim(node.FullCodePath, "/"), "/"))
		if nodeDepth == parentDepth+1 {
			children = append(children, node)
		}
	}
	return dedupeServiceTreesByFullPath(children), nil
}

// BuildServiceTree 构建树形结构
func (r *ServiceTreeRepository) BuildServiceTree(appID int64) ([]*model.ServiceTree, error) {
	// 获取所有服务目录
	allTrees, err := r.GetServiceTreesByAppID(appID)
	if err != nil {
		return nil, err
	}
	return r.buildTreeFromNodes(allTrees), nil
}

// BuildServiceTreeByType 根据类型构建树形结构
func (r *ServiceTreeRepository) BuildServiceTreeByType(appID int64, nodeType string) ([]*model.ServiceTree, error) {
	// 获取指定类型的服务目录
	allTrees, err := r.GetServiceTreesByAppIDAndType(appID, nodeType)
	if err != nil {
		return nil, err
	}
	return r.buildTreeFromNodes(allTrees), nil
}

// BuildServiceTreeByVersion 根据版本号构建树形结构（用于版本回滚）
// versionNum: 目标版本号数字（如 19），只返回 add_version_num <= versionNum 且 (update_version_num = 0 或 update_version_num <= versionNum) 的节点
func (r *ServiceTreeRepository) BuildServiceTreeByVersion(appID int64, versionNum int) ([]*model.ServiceTree, error) {
	// 查询符合条件的节点：add_version_num <= versionNum 且 (update_version_num = 0 或 update_version_num <= versionNum)
	var allTrees []*model.ServiceTree
	err := r.db.Where("app_id = ? AND add_version_num <= ? AND (update_version_num = 0 OR update_version_num <= ?)",
		appID, versionNum, versionNum).
		Order("id ASC").
		Find(&allTrees).Error
	if err != nil {
		return nil, err
	}
	return r.buildTreeFromNodes(allTrees), nil
}

// buildTreeFromNodes 从节点列表构建树形结构（基于 FullCodePath）
func (r *ServiceTreeRepository) buildTreeFromNodes(allTrees []*model.ServiceTree) []*model.ServiceTree {
	allTrees = dedupeServiceTreesByFullPath(allTrees)

	pathMap := make(map[string]*model.ServiceTree, len(allTrees))
	var rootNodes []*model.ServiceTree

	for _, tree := range allTrees {
		pathMap[tree.FullCodePath] = tree
	}

	for _, tree := range allTrees {
		parentPath := tree.GetParentFullPath()
		if parentPath == "" {
			rootNodes = append(rootNodes, tree)
		} else if parent, exists := pathMap[parentPath]; exists {
			parent.Children = append(parent.Children, tree)
		} else {
			rootNodes = append(rootNodes, tree)
		}
	}

	return rootNodes
}

func dedupeServiceTreesByFullPath(allTrees []*model.ServiceTree) []*model.ServiceTree {
	if len(allTrees) == 0 {
		return allTrees
	}

	seen := make(map[string]struct{}, len(allTrees))
	deduped := make([]*model.ServiceTree, 0, len(allTrees))
	for _, tree := range allTrees {
		if tree == nil {
			continue
		}
		key := normalizeFullCodePath(tree.FullCodePath)
		if key == "" {
			key = tree.FullCodePath
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		tree.Children = nil
		deduped = append(deduped, tree)
	}
	return deduped
}

// UpdateServiceTree 更新服务目录
func (r *ServiceTreeRepository) UpdateServiceTree(serviceTree *model.ServiceTree) error {
	return r.db.Save(serviceTree).Error
}

// DeleteServiceTree 删除服务目录（级联删除子目录）
func (r *ServiceTreeRepository) DeleteServiceTree(id int64) error {
	// 先删除所有子目录
	children, err := r.GetServiceTreeChildren(id)
	if err != nil {
		return err
	}

	for _, child := range children {
		if err := r.DeleteServiceTree(child.ID); err != nil {
			return err
		}
	}

	// 删除当前目录
	return r.db.Delete(&model.ServiceTree{}, id).Error
}

// calculatePathsWithAppPrefix 计算带有用户应用前缀的路径信息
func (r *ServiceTreeRepository) calculatePathsWithAppPrefix(serviceTree *model.ServiceTree, userAppPrefix string) error {
	// FullCodePath使用预加载的app信息计算
	if serviceTree.App != nil {
		// 使用预加载的App对象构建路径
		appPrefix := fmt.Sprintf("/%s/%s", serviceTree.App.User, serviceTree.App.Code)
		serviceTree.FullCodePath = fmt.Sprintf("%s/%s", appPrefix, serviceTree.Code)
	} else {
		// 回退到传入的用户应用前缀
		serviceTree.FullCodePath = fmt.Sprintf("%s/%s", userAppPrefix, serviceTree.Code)
	}

	return nil
}

// normalizeFullCodePath 规范化 full_code_path：去首尾空格、去尾斜杠、保证以单个 / 开头（便于与 DB 一致匹配）
func normalizeFullCodePath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.TrimSuffix(p, "/")
	if p != "" && !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

// GetServiceTreeByFullPath 根据完整路径获取服务目录（full_code_path全局唯一）
func (r *ServiceTreeRepository) GetServiceTreeByFullPath(fullPath string) (*model.ServiceTree, error) {
	fullPath = normalizeFullCodePath(fullPath)
	if fullPath == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var serviceTree model.ServiceTree
	err := r.db.Where("full_code_path = ?", fullPath).Order("id ASC").First(&serviceTree).Error
	if err != nil {
		return nil, err
	}
	return &serviceTree, nil
}

// IncrementRunCountByFullCodePath 将指定 full_code_path 的 function 节点运行次数 +1（用于 search 按热度排序）
func (r *ServiceTreeRepository) IncrementRunCountByFullCodePath(ctx context.Context, fullPath string) error {
	fullPath = normalizeFullCodePath(fullPath)
	if fullPath == "" {
		return nil
	}
	res := r.db.WithContext(ctx).Model(&model.ServiceTree{}).
		Where("full_code_path = ? AND type = ?", fullPath, model.ServiceTreeTypeFunction).
		Update("run_count", gorm.Expr("run_count + 1"))
	if res.Error != nil {
		return res.Error
	}
	return nil
}

// GetNodeByPath 根据路径查询节点（带 context）。
func (r *ServiceTreeRepository) GetNodeByPath(ctx context.Context, resourcePath string) (*model.ServiceTree, error) {
	return r.GetServiceTreeByFullPath(resourcePath)
}

// GetNodeAdmins 获取节点的管理员列表
func (r *ServiceTreeRepository) GetNodeAdmins(ctx context.Context, resourcePath string) ([]string, error) {
	var node model.ServiceTree
	err := r.db.WithContext(ctx).
		Where("full_code_path = ?", resourcePath).
		Select("admins").
		First(&node).Error
	if err != nil {
		return nil, err
	}

	// 解析逗号分隔的管理员列表
	if node.Admins == "" {
		return []string{}, nil
	}

	admins := strings.Split(node.Admins, ",")
	result := make([]string, 0, len(admins))
	for _, admin := range admins {
		admin = strings.TrimSpace(admin)
		if admin != "" {
			result = append(result, admin)
		}
	}

	return result, nil
}

// AddNodeAdmin 添加节点管理员
func (r *ServiceTreeRepository) AddNodeAdmin(ctx context.Context, resourcePath string, adminUsername string) error {
	// 获取当前管理员列表
	admins, err := r.GetNodeAdmins(ctx, resourcePath)
	if err != nil {
		return err
	}

	// 检查是否已存在
	for _, admin := range admins {
		if admin == adminUsername {
			return nil // 已存在，静默成功
		}
	}

	// 添加新管理员
	admins = append(admins, adminUsername)
	adminsStr := strings.Join(admins, ",")

	return r.db.WithContext(ctx).
		Model(&model.ServiceTree{}).
		Where("full_code_path = ?", resourcePath).
		Update("admins", adminsStr).Error
}

// RemoveNodeAdmin 删除节点管理员
func (r *ServiceTreeRepository) RemoveNodeAdmin(ctx context.Context, resourcePath string, adminUsername string) error {
	// 获取当前管理员列表
	admins, err := r.GetNodeAdmins(ctx, resourcePath)
	if err != nil {
		return err
	}

	// 移除管理员
	newAdmins := make([]string, 0, len(admins))
	for _, admin := range admins {
		if admin != adminUsername {
			newAdmins = append(newAdmins, admin)
		}
	}

	adminsStr := strings.Join(newAdmins, ",")

	return r.db.WithContext(ctx).
		Model(&model.ServiceTree{}).
		Where("full_code_path = ?", resourcePath).
		Update("admins", adminsStr).Error
}

// GetServiceTreeByFullPaths 批量根据完整路径获取服务目录
func (r *ServiceTreeRepository) GetServiceTreeByFullPaths(fullPaths []string) (map[string]*model.ServiceTree, error) {
	if len(fullPaths) == 0 {
		return make(map[string]*model.ServiceTree), nil
	}

	var serviceTrees []*model.ServiceTree
	err := r.db.Where("full_code_path IN ?", fullPaths).Order("id ASC").Find(&serviceTrees).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]*model.ServiceTree)
	for _, tree := range serviceTrees {
		if _, exists := result[tree.FullCodePath]; exists {
			continue
		}
		result[tree.FullCodePath] = tree
	}
	return result, nil
}

// CheckNameExists 检查名称是否已存在（在同一父目录下和同一应用内）
// parentID 参数保留签名兼容，内部先查 parent 再走路径逻辑
func (r *ServiceTreeRepository) CheckNameExists(parentID int64, code string, appID int64) (bool, error) {
	if parentID == 0 {
		return false, nil
	}
	parent, err := r.GetServiceTreeByID(parentID)
	if err != nil {
		return false, fmt.Errorf("查询父节点失败: %w", err)
	}
	return r.CheckNameExistsByPath(parent.FullCodePath, code, appID)
}

// CheckNameExistsByPath 基于路径检查同级是否已存在同 code 节点
func (r *ServiceTreeRepository) CheckNameExistsByPath(parentPath string, code string, appID int64) (bool, error) {
	targetPath := strings.TrimSuffix(parentPath, "/") + "/" + code
	var count int64
	err := r.db.Model(&model.ServiceTree{}).
		Where("full_code_path = ? AND app_id = ?", targetPath, appID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *ServiceTreeRepository) GetByID(parentId int64) (*model.ServiceTree, error) {
	var tree model.ServiceTree
	err := r.db.Model(&model.ServiceTree{}).Where("id = ?", parentId).First(&tree).Error
	if err != nil {
		return nil, err
	}
	return &tree, nil
}

// GetDescendantDirectories 递归获取所有子目录（包括嵌套）
// 使用路径前缀匹配，一次查询获取所有子目录
func (r *ServiceTreeRepository) GetDescendantDirectories(appID int64, parentFullCodePath string) ([]*model.ServiceTree, error) {
	// 标准化路径（确保以 / 结尾，用于前缀匹配）
	normalizedPath := strings.TrimSuffix(parentFullCodePath, "/") + "/"

	var descendants []*model.ServiceTree
	err := r.db.Where("app_id = ? AND full_code_path LIKE ? AND type = ?",
		appID, normalizedPath+"%", model.ServiceTreeTypePackage).
		Order("full_code_path ASC").
		Find(&descendants).Error

	if err != nil {
		return nil, err
	}

	// 过滤：只返回真正的子目录（路径必须以 parentFullCodePath/ 开头）
	result := make([]*model.ServiceTree, 0)
	for _, dir := range descendants {
		if strings.HasPrefix(dir.FullCodePath, normalizedPath) {
			result = append(result, dir)
		}
	}

	return result, nil
}

// GetDescendantNodes 递归获取所有子节点（包括目录、函数和文档）。
func (r *ServiceTreeRepository) GetDescendantNodes(appID int64, parentFullCodePath string) ([]*model.ServiceTree, error) {
	normalizedPath := strings.TrimSuffix(parentFullCodePath, "/") + "/"

	var descendants []*model.ServiceTree
	err := r.db.Where("app_id = ? AND full_code_path LIKE ?",
		appID, normalizedPath+"%").
		Order("full_code_path ASC").
		Find(&descendants).Error
	if err != nil {
		return nil, err
	}

	result := make([]*model.ServiceTree, 0, len(descendants))
	for _, node := range descendants {
		if strings.HasPrefix(node.FullCodePath, normalizedPath) {
			result = append(result, node)
		}
	}
	return result, nil
}

// splitSearchKeywords 将 keyword 按竖线 | 拆成多个关键词并去空，如 "视频|video|流媒体" -> ["视频","video","流媒体"]
func splitSearchKeywords(keyword string) []string {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil
	}
	parts := strings.Split(keyword, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// SearchFunctions 搜索函数节点：只查 ServiceTree（type=function），按 code/name/description/tags/full_code_path 匹配，预加载 App、Function
// user 非空时先查 app 表该 user 的 app id 列表，再用 app_id IN 限定，避免 JOIN 导致查不到
func (r *ServiceTreeRepository) SearchFunctions(currentUser, user, app, keyword, fullCodePath, templateType string, page, pageSize int) ([]*model.ServiceTree, int64, error) {
	query := r.db.Model(&model.ServiceTree{}).
		Where("service_tree.type = ?", model.ServiceTreeTypeFunction)

	// user 非空时：先查 app 表得到该用户下的 app id，用 app_id IN 限定（保证能命中 system 等）
	if user != "" {
		subq := r.db.Model(&model.App{}).Select("id").Where("user = ?", user)
		if app != "" {
			subq = subq.Where("code = ?", app)
		}
		query = query.Where("service_tree.app_id IN (?)", subq)
	} else if currentUser != "" && app != "" {
		subq := r.db.Model(&model.App{}).Select("id").Where("code = ?", app)
		query = query.Where("service_tree.app_id IN (?)", subq)
	}
	if templateType != "" {
		query = query.Where("service_tree.template_type = ?", templateType)
	}
	if fullCodePath != "" {
		fullCodePath = normalizeFullCodePath(fullCodePath)
		query = query.Where("(service_tree.full_code_path = ? OR service_tree.full_code_path LIKE ?)", fullCodePath, fullCodePath+"/%")
	}
	if keyword != "" {
		keywords := splitSearchKeywords(keyword)
		if len(keywords) > 0 {
			var orConditions []string
			var args []interface{}
			for _, k := range keywords {
				pattern := "%" + k + "%"
				orConditions = append(orConditions, "(service_tree.code LIKE ? OR service_tree.name LIKE ? OR service_tree.description LIKE ? OR service_tree.tags LIKE ? OR service_tree.full_code_path LIKE ?)")
				args = append(args, pattern, pattern, pattern, pattern, pattern)
			}
			query = query.Where(strings.Join(orConditions, " OR "), args...)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	offset := (page - 1) * pageSize
	var list []*model.ServiceTree
	if err := query.
		Preload("App").
		Preload("Function").
		Offset(offset).
		Limit(pageSize).
		Order("service_tree.run_count DESC, service_tree.created_at DESC").
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// SearchResources 搜索服务树资源节点：目录、函数、文档。
// 文档节点额外 JOIN docs 表，支持按文档摘要/正文命中。
func (r *ServiceTreeRepository) SearchResources(currentUser, user, app, keyword, fullCodePath string, nodeTypes []string, page, pageSize int) ([]*model.ServiceTree, int64, error) {
	query := r.db.Model(&model.ServiceTree{}).
		Joins("LEFT JOIN docs ON docs.tree_id = service_tree.id")

	if len(nodeTypes) > 0 {
		query = query.Where("service_tree.type IN ?", nodeTypes)
	}

	if user != "" {
		subq := r.db.Model(&model.App{}).Select("id").Where("user = ?", user)
		if app != "" {
			subq = subq.Where("code = ?", app)
		}
		query = query.Where("service_tree.app_id IN (?)", subq)
	} else if currentUser != "" && app != "" {
		subq := r.db.Model(&model.App{}).Select("id").Where("code = ?", app)
		query = query.Where("service_tree.app_id IN (?)", subq)
	}
	if fullCodePath != "" {
		fullCodePath = normalizeFullCodePath(fullCodePath)
		query = query.Where("(service_tree.full_code_path = ? OR service_tree.full_code_path LIKE ?)", fullCodePath, fullCodePath+"/%")
	}

	if keyword != "" {
		keywords := splitSearchKeywords(keyword)
		if len(keywords) > 0 {
			var orConditions []string
			var args []interface{}
			for _, k := range keywords {
				pattern := "%" + k + "%"
				orConditions = append(orConditions, `(
					service_tree.code LIKE ?
					OR service_tree.name LIKE ?
					OR service_tree.description LIKE ?
					OR service_tree.tags LIKE ?
					OR service_tree.full_code_path LIKE ?
					OR docs.name LIKE ?
					OR docs.summary LIKE ?
					OR docs.content LIKE ?
					OR docs.category LIKE ?
				)`)
				args = append(args, pattern, pattern, pattern, pattern, pattern, pattern, pattern, pattern, pattern)
			}
			query = query.Where(strings.Join(orConditions, " OR "), args...)
		}
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Distinct("service_tree.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	offset := (page - 1) * pageSize
	var list []*model.ServiceTree
	if err := query.
		Select("service_tree.*, docs.summary AS search_doc_summary").
		Preload("App").
		Offset(offset).
		Limit(pageSize).
		Order("service_tree.run_count DESC, service_tree.updated_at DESC, service_tree.created_at DESC").
		Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}
