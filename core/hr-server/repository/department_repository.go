package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/hr-server/model"
	"gorm.io/gorm"
)

type DepartmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) *DepartmentRepository {
	return &DepartmentRepository{db: db}
}

// CreateDepartment 创建部门
func (r *DepartmentRepository) CreateDepartment(department *model.Department) error {
	return r.db.Create(department).Error
}

// GetDepartmentByID 根据ID获取部门
func (r *DepartmentRepository) GetDepartmentByID(id int64) (*model.Department, error) {
	var department model.Department
	err := r.db.Where("id = ?", id).First(&department).Error
	if err != nil {
		return nil, err
	}
	return &department, nil
}

// GetDepartmentByCode 根据编码获取部门
func (r *DepartmentRepository) GetDepartmentByCode(code string) (*model.Department, error) {
	var department model.Department
	err := r.db.Where("code = ?", code).First(&department).Error
	if err != nil {
		return nil, err
	}
	return &department, nil
}

// GetDepartmentByFullCodePath 根据完整路径获取部门
func (r *DepartmentRepository) GetDepartmentByFullCodePath(fullCodePath string) (*model.Department, error) {
	var department model.Department
	err := r.db.Where("full_code_path = ?", fullCodePath).First(&department).Error
	if err != nil {
		return nil, err
	}
	return &department, nil
}

// GetAllDepartments 获取所有部门
func (r *DepartmentRepository) GetAllDepartments() ([]*model.Department, error) {
	var departments []*model.Department
	err := r.db.Order("sort ASC, id ASC").Find(&departments).Error
	if err != nil {
		return nil, err
	}
	return departments, nil
}

// GetDepartmentsByParentID 根据父部门ID获取子部门列表
func (r *DepartmentRepository) GetDepartmentsByParentID(parentID *int64) ([]*model.Department, error) {
	var departments []*model.Department
	if parentID == nil {
		err := r.db.Where("parent_id IS NULL").Order("sort ASC, id ASC").Find(&departments).Error
		if err != nil {
			return nil, err
		}
	} else {
		err := r.db.Where("parent_id = ?", *parentID).Order("sort ASC, id ASC").Find(&departments).Error
		if err != nil {
			return nil, err
		}
	}
	return departments, nil
}

// UpdateDepartment 更新部门
func (r *DepartmentRepository) UpdateDepartment(department *model.Department) error {
	return r.db.Save(department).Error
}

// DeleteDepartment 删除部门（软删除）
func (r *DepartmentRepository) DeleteDepartment(id int64) error {
	return r.db.Delete(&model.Department{}, id).Error
}

// GetDepartmentTree 获取部门树（递归查询）
func (r *DepartmentRepository) GetDepartmentTree() ([]*model.Department, error) {
	var departments []*model.Department
	err := r.db.Where("parent_id IS NULL").Order("sort ASC, id ASC").Find(&departments).Error
	if err != nil {
		return nil, err
	}

	// 递归加载子部门
	for _, dept := range departments {
		if err := r.loadChildren(dept); err != nil {
			return nil, err
		}
	}

	return departments, nil
}

// loadChildren 递归加载子部门
func (r *DepartmentRepository) loadChildren(department *model.Department) error {
	parentID := &department.ID
	children, err := r.GetDepartmentsByParentID(parentID)
	if err != nil {
		return err
	}

	department.Children = children

	// 递归加载子部门的子部门
	for _, child := range children {
		if err := r.loadChildren(child); err != nil {
			return err
		}
	}

	return nil
}

