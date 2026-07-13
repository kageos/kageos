package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/pkg/logger"
)

// DepartmentService 部门服务
type DepartmentService struct {
	deptRepo *repository.DepartmentRepository
	userRepo *repository.UserRepository

	// 内存缓存：fullCodePath -> Department
	deptCache     map[string]*model.Department
	deptCacheMu   sync.RWMutex
	deptCacheInit bool
}

// NewDepartmentService 创建部门服务
func NewDepartmentService(deptRepo *repository.DepartmentRepository, userRepo *repository.UserRepository) *DepartmentService {
	return &DepartmentService{
		deptRepo:      deptRepo,
		userRepo:      userRepo,
		deptCache:     make(map[string]*model.Department),
		deptCacheInit: false,
	}
}

// loadDepartmentCache 加载部门缓存（懒加载，首次调用时加载）
func (s *DepartmentService) loadDepartmentCache(ctx context.Context) error {
	s.deptCacheMu.Lock()
	defer s.deptCacheMu.Unlock()

	// 如果已经初始化，直接返回
	if s.deptCacheInit {
		return nil
	}

	// 从数据库加载所有部门
	departments, err := s.deptRepo.GetAllDepartments(ctx)
	if err != nil {
		return fmt.Errorf("加载部门缓存失败: %w", err)
	}

	// 构建缓存：fullCodePath -> Department
	s.deptCache = make(map[string]*model.Department, len(departments))
	for _, dept := range departments {
		s.deptCache[dept.FullCodePath] = dept
	}

	s.deptCacheInit = true
	logger.Infof(ctx, "[DepartmentService] 部门缓存已加载，共 %d 个部门", len(s.deptCache))
	return nil
}

// GetDepartmentByFullCodePath 根据完整路径获取部门（从缓存读取）
func (s *DepartmentService) GetDepartmentByFullCodePath(ctx context.Context, fullCodePath string) (*model.Department, error) {
	// 确保缓存已加载
	if err := s.loadDepartmentCache(ctx); err != nil {
		return nil, err
	}

	s.deptCacheMu.RLock()
	defer s.deptCacheMu.RUnlock()

	dept, ok := s.deptCache[fullCodePath]
	if !ok {
		return nil, fmt.Errorf("部门不存在: %s", fullCodePath)
	}

	return dept, nil
}

// GetDepartmentsByFullCodePaths 批量获取部门信息（从缓存读取）
func (s *DepartmentService) GetDepartmentsByFullCodePaths(ctx context.Context, fullCodePaths []string) (map[string]*model.Department, error) {
	// 确保缓存已加载
	if err := s.loadDepartmentCache(ctx); err != nil {
		return nil, err
	}

	s.deptCacheMu.RLock()
	defer s.deptCacheMu.RUnlock()

	result := make(map[string]*model.Department)
	for _, path := range fullCodePaths {
		if path == "" {
			continue
		}
		if dept, ok := s.deptCache[path]; ok {
			result[path] = dept
		}
	}

	return result, nil
}

// InvalidateDepartmentCache 使部门缓存失效（在创建、更新、删除部门后调用）
func (s *DepartmentService) InvalidateDepartmentCache() {
	s.deptCacheMu.Lock()
	defer s.deptCacheMu.Unlock()

	s.deptCacheInit = false
	s.deptCache = make(map[string]*model.Department)
}

// InitDefaultDepartments 初始化默认组织（根节点、未分配组织、虚拟组织/测试组）
// 如果已存在则不重复创建
func (s *DepartmentService) InitDefaultDepartments(ctx context.Context) error {
	// 检查根节点是否已存在
	rootDept, err := s.deptRepo.GetDepartmentByCode(ctx, "org")
	if err == nil && rootDept != nil {
		// 根节点已存在，检查未分配组织
		unassignedDept, err := s.deptRepo.GetDepartmentByCode(ctx, "unassigned")
		if err == nil && unassignedDept != nil {
			// 根与未分配都已存在，仍需确保虚拟组织/测试组存在
			if err := s.createVirtualTestDepartment(ctx, rootDept.ID); err != nil {
				return err
			}
			s.InvalidateDepartmentCache()
			logger.Infof(ctx, "[DepartmentService] 默认组织已存在，已确保虚拟组织/测试组")
			return nil
		}
		// 根节点存在但未分配组织不存在，创建未分配组织后再确保虚拟组织/测试组
		if err := s.createUnassignedDepartment(ctx, rootDept.ID); err != nil {
			return err
		}
		if err := s.createVirtualTestDepartment(ctx, rootDept.ID); err != nil {
			return err
		}
		s.InvalidateDepartmentCache()
		return nil
	}

	// 创建根节点（默认组织）
	rootDept = &model.Department{
		Name:            "默认组织",
		Code:            "org",
		ParentID:        nil, // 根节点
		FullCodePath:    "/org",
		FullNamePath:    "默认组织",
		Description:     "系统默认根组织",
		Status:          "active",
		Sort:            0,
		IsSystemDefault: true, // 标记为系统默认组织
	}

	if err := s.deptRepo.CreateDepartment(ctx, rootDept); err != nil {
		return fmt.Errorf("创建根节点失败: %w", err)
	}

	logger.Infof(ctx, "[DepartmentService] 根节点创建成功: %s", rootDept.Name)

	// 创建未分配组织
	if err := s.createUnassignedDepartment(ctx, rootDept.ID); err != nil {
		return err
	}

	// 创建虚拟组织/测试组
	if err := s.createVirtualTestDepartment(ctx, rootDept.ID); err != nil {
		return err
	}

	// 使缓存失效
	s.InvalidateDepartmentCache()

	logger.Infof(ctx, "[DepartmentService] 默认组织初始化完成")
	return nil
}

// createUnassignedDepartment 创建未分配组织
func (s *DepartmentService) createUnassignedDepartment(ctx context.Context, rootID int64) error {
	// 检查未分配组织是否已存在
	unassignedDept, err := s.deptRepo.GetDepartmentByCode(ctx, "unassigned")
	if err == nil && unassignedDept != nil {
		// 已存在，更新为系统默认组织（如果之前不是）
		if !unassignedDept.IsSystemDefault {
			unassignedDept.IsSystemDefault = true
			if err := s.deptRepo.UpdateDepartment(ctx, unassignedDept); err != nil {
				logger.Warnf(ctx, "[DepartmentService] 更新未分配组织标记失败: %v", err)
			}
		}
		return nil
	}

	// 获取根节点信息（用于构建 FullNamePath）
	rootDept, err := s.deptRepo.GetDepartmentByID(ctx, rootID)
	if err != nil {
		return fmt.Errorf("获取根节点失败: %w", err)
	}

	unassignedDept = &model.Department{
		Name:            "未分配",
		Code:            "unassigned",
		ParentID:        &rootID,
		FullCodePath:    "/org/unassigned",
		FullNamePath:    fmt.Sprintf("%s/未分配", rootDept.FullNamePath),
		Description:     "未分配用户的默认组织",
		Status:          "active",
		Sort:            0,
		IsSystemDefault: true, // 标记为系统默认组织
	}

	if err := s.deptRepo.CreateDepartment(ctx, unassignedDept); err != nil {
		return fmt.Errorf("创建未分配组织失败: %w", err)
	}

	logger.Infof(ctx, "[DepartmentService] 未分配组织创建成功: %s", unassignedDept.Name)
	return nil
}

// GetUnassignedDepartmentPath 获取未分配组织的完整路径
func (s *DepartmentService) GetUnassignedDepartmentPath() string {
	return "/org/unassigned"
}

// VirtualTestDepartmentPath 虚拟组织/测试组的完整路径（用于 test_user 等测试账号）
func (s *DepartmentService) VirtualTestDepartmentPath() string {
	return "/org/virtual/test"
}

// createVirtualTestDepartment 创建虚拟组织与测试组（org -> 虚拟组织 -> 测试组），若已存在则跳过
func (s *DepartmentService) createVirtualTestDepartment(ctx context.Context, rootID int64) error {
	rootDept, err := s.deptRepo.GetDepartmentByID(ctx, rootID)
	if err != nil || rootDept == nil {
		return fmt.Errorf("获取根节点失败: %w", err)
	}
	// 虚拟组织
	virtualDept, err := s.deptRepo.GetDepartmentByFullCodePath(ctx, "/org/virtual")
	if err != nil || virtualDept == nil {
		virtualDept = &model.Department{
			Name:            "虚拟组织",
			Code:            "virtual",
			ParentID:        &rootID,
			FullCodePath:    "/org/virtual",
			FullNamePath:    fmt.Sprintf("%s/虚拟组织", rootDept.FullNamePath),
			Description:     "虚拟/测试用组织",
			Status:          "active",
			Sort:            10,
			IsSystemDefault: true,
		}
		if err := s.deptRepo.CreateDepartment(ctx, virtualDept); err != nil {
			return fmt.Errorf("创建虚拟组织失败: %w", err)
		}
		logger.Infof(ctx, "[DepartmentService] 虚拟组织创建成功: %s", virtualDept.Name)
	}
	// 测试组
	testDept, err := s.deptRepo.GetDepartmentByFullCodePath(ctx, "/org/virtual/test")
	if err != nil || testDept == nil {
		testDept := &model.Department{
			Name:            "测试组",
			Code:            "test",
			ParentID:        &virtualDept.ID,
			FullCodePath:    "/org/virtual/test",
			FullNamePath:    fmt.Sprintf("%s/测试组", virtualDept.FullNamePath),
			Description:     "测试用户默认归属",
			Status:          "active",
			Sort:            0,
			IsSystemDefault: true,
		}
		if err := s.deptRepo.CreateDepartment(ctx, testDept); err != nil {
			return fmt.Errorf("创建测试组失败: %w", err)
		}
		logger.Infof(ctx, "[DepartmentService] 测试组创建成功: %s", testDept.Name)
	}
	return nil
}

// CreateDepartment 创建部门
func (s *DepartmentService) CreateDepartment(ctx context.Context, name, code string, parentID int64, description string, managers string) (*model.Department, error) {
	// 构建 FullCodePath
	fullCodePath, err := s.buildFullCodePath(ctx, code, parentID)
	if err != nil {
		return nil, fmt.Errorf("构建部门路径失败: %w", err)
	}

	// 检查编码是否已存在
	existing, err := s.deptRepo.GetDepartmentByCode(ctx, code)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("部门编码 %s 已存在", code)
	}

	// 处理 parentID：0 表示根部门，使用 nil
	var parentIDPtr *int64
	if parentID != 0 {
		parentIDPtr = &parentID
	}

	// 构建 FullNamePath
	fullNamePath, err := s.buildFullNamePath(ctx, name, parentID)
	if err != nil {
		return nil, fmt.Errorf("构建部门名称路径失败: %w", err)
	}

	department := &model.Department{
		Name:         name,
		Code:         code,
		ParentID:     parentIDPtr,
		FullCodePath: fullCodePath,
		FullNamePath: fullNamePath,
		Description:  description,
		Managers:     managers,
		Status:       "active",
		Sort:         0,
	}

	if err := s.deptRepo.CreateDepartment(ctx, department); err != nil {
		return nil, fmt.Errorf("创建部门失败: %w", err)
	}

	// ⭐ 如果部门名称发生变化，需要更新所有子部门的 FullNamePath
	if err := s.updateChildrenFullNamePath(ctx, department); err != nil {
		logger.Warnf(ctx, "[DepartmentService] 更新子部门名称路径失败: %v", err)
		// 不返回错误，因为主操作已成功
	}

	// 使缓存失效
	s.InvalidateDepartmentCache()

	logger.Infof(ctx, "[DepartmentService] Department created: %s (path: %s)", name, fullCodePath)
	return department, nil
}

// UpdateDepartment 更新部门
func (s *DepartmentService) UpdateDepartment(ctx context.Context, id int64, name, description, managers string, status string, sort int) (*model.Department, error) {
	department, err := s.deptRepo.GetDepartmentByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("部门不存在: %w", err)
	}

	nameChanged := false
	if name != "" && name != department.Name {
		department.Name = name
		nameChanged = true
	}
	if description != "" {
		department.Description = description
	}
	if managers != "" {
		department.Managers = managers
	}
	if status != "" {
		department.Status = status
	}
	if sort >= 0 {
		department.Sort = sort
	}

	// 如果部门名称发生变化，需要重新计算 FullNamePath
	if nameChanged {
		parentID := int64(0)
		if department.ParentID != nil {
			parentID = *department.ParentID
		}
		fullNamePath, err := s.buildFullNamePath(ctx, department.Name, parentID)
		if err != nil {
			return nil, fmt.Errorf("构建部门名称路径失败: %w", err)
		}
		department.FullNamePath = fullNamePath
	}

	if err := s.deptRepo.UpdateDepartment(ctx, department); err != nil {
		return nil, fmt.Errorf("更新部门失败: %w", err)
	}

	// 使缓存失效
	s.InvalidateDepartmentCache()

	// ⭐ 如果部门名称发生变化，需要更新所有子部门的 FullNamePath
	if nameChanged {
		if err := s.updateChildrenFullNamePath(ctx, department); err != nil {
			logger.Warnf(ctx, "[DepartmentService] 更新子部门名称路径失败: %v", err)
			// 不返回错误，因为主操作已成功
		}
	}

	logger.Infof(ctx, "[DepartmentService] Department updated: %s", department.Name)
	return department, nil
}

// GetDepartmentByID 根据ID获取部门
func (s *DepartmentService) GetDepartmentByID(ctx context.Context, id int64) (*model.Department, error) {
	return s.deptRepo.GetDepartmentByID(ctx, id)
}

// GetDepartmentTree 获取部门树
func (s *DepartmentService) GetDepartmentTree(ctx context.Context) ([]*model.Department, error) {
	return s.deptRepo.GetDepartmentTree(ctx)
}

// DeleteDepartment 删除部门
func (s *DepartmentService) DeleteDepartment(ctx context.Context, id int64) error {
	// 获取部门信息
	department, err := s.deptRepo.GetDepartmentByID(ctx, id)
	if err != nil {
		return fmt.Errorf("部门不存在: %w", err)
	}

	// ⭐ 检查是否为系统默认组织（不可删除）
	if department.IsSystemDefault {
		return fmt.Errorf("系统默认组织不可删除")
	}

	// 检查是否有子部门
	parentID := &id
	children, err := s.deptRepo.GetDepartmentsByParentID(ctx, parentID)
	if err != nil {
		return fmt.Errorf("查询子部门失败: %w", err)
	}
	if len(children) > 0 {
		return fmt.Errorf("该部门下存在子部门，无法删除")
	}

	// 检查是否有用户（通过 DepartmentFullPath 查询）
	userCount, err := s.userRepo.CountUsersByDepartmentFullPath(ctx, department.FullCodePath)
	if err != nil {
		return fmt.Errorf("查询部门下用户失败: %w", err)
	}
	if userCount > 0 {
		return fmt.Errorf("该部门下存在用户，无法删除")
	}

	if err := s.deptRepo.DeleteDepartment(ctx, id); err != nil {
		return err
	}

	// 使缓存失效
	s.InvalidateDepartmentCache()

	return nil
}

// buildFullCodePath 构建部门完整路径
func (s *DepartmentService) buildFullCodePath(ctx context.Context, code string, parentID int64) (string, error) {
	if parentID == 0 {
		// 根部门
		return fmt.Sprintf("/%s", code), nil
	}

	// 获取父部门
	parent, err := s.deptRepo.GetDepartmentByID(ctx, parentID)
	if err != nil {
		return "", fmt.Errorf("父部门不存在: %w", err)
	}

	// 拼接路径
	return fmt.Sprintf("%s/%s", parent.FullCodePath, code), nil
}

// buildFullNamePath 构建部门完整名称路径（如：技术部/后端组）
func (s *DepartmentService) buildFullNamePath(ctx context.Context, name string, parentID int64) (string, error) {
	if parentID == 0 {
		// 根部门
		return name, nil
	}

	// 获取父部门
	parent, err := s.deptRepo.GetDepartmentByID(ctx, parentID)
	if err != nil {
		return "", fmt.Errorf("父部门不存在: %w", err)
	}

	// 如果父部门有 FullNamePath，则拼接；否则使用父部门名称
	if parent.FullNamePath != "" {
		return fmt.Sprintf("%s/%s", parent.FullNamePath, name), nil
	}
	// 如果父部门没有 FullNamePath（可能是旧数据），则递归构建
	parentFullNamePath, err := s.buildFullNamePath(ctx, parent.Name, func() int64 {
		if parent.ParentID == nil {
			return 0
		}
		return *parent.ParentID
	}())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", parentFullNamePath, name), nil
}

// updateChildrenFullNamePath 更新所有子部门的 FullNamePath（当父部门名称变化时）
func (s *DepartmentService) updateChildrenFullNamePath(ctx context.Context, parent *model.Department) error {
	// 获取所有子部门
	children, err := s.deptRepo.GetDepartmentsByParentID(ctx, &parent.ID)
	if err != nil {
		return fmt.Errorf("查询子部门失败: %w", err)
	}

	// 递归更新每个子部门
	for _, child := range children {
		// 重新计算子部门的 FullNamePath
		fullNamePath, err := s.buildFullNamePath(ctx, child.Name, parent.ID)
		if err != nil {
			logger.Warnf(ctx, "[DepartmentService] 计算子部门 %s 的名称路径失败: %v", child.Name, err)
			continue
		}

		// 更新子部门
		child.FullNamePath = fullNamePath
		if err := s.deptRepo.UpdateDepartment(ctx, child); err != nil {
			logger.Warnf(ctx, "[DepartmentService] 更新子部门 %s 失败: %v", child.Name, err)
			continue
		}

		// 递归更新子部门的子部门
		if err := s.updateChildrenFullNamePath(ctx, child); err != nil {
			logger.Warnf(ctx, "[DepartmentService] 更新子部门 %s 的子部门失败: %v", child.Name, err)
			// 继续处理其他子部门
		}
	}

	return nil
}
