package repository

import (
	"context"

	"github.com/kageos/kageos/core/hr-server/model"
	"gorm.io/gorm"
)

type DepartmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) *DepartmentRepository {
	return &DepartmentRepository{db: db}
}

// CreateDepartment 创建部门
func (r *DepartmentRepository) CreateDepartment(ctx context.Context, department *model.Department) error {
	return r.db.WithContext(ctx).Create(department).Error
}

// GetDepartmentByID 根据ID获取部门
func (r *DepartmentRepository) GetDepartmentByID(ctx context.Context, id int64) (*model.Department, error) {
	var department model.Department
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&department).Error
	if err != nil {
		return nil, err
	}
	return &department, nil
}

// GetDepartmentByCode 根据编码获取部门
func (r *DepartmentRepository) GetDepartmentByCode(ctx context.Context, code string) (*model.Department, error) {
	var department model.Department
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&department).Error
	if err != nil {
		return nil, err
	}
	return &department, nil
}

// GetDepartmentByFullCodePath 根据完整路径获取部门
func (r *DepartmentRepository) GetDepartmentByFullCodePath(ctx context.Context, fullCodePath string) (*model.Department, error) {
	var department model.Department
	err := r.db.WithContext(ctx).Where("full_code_path = ?", fullCodePath).First(&department).Error
	if err != nil {
		return nil, err
	}
	return &department, nil
}

// GetAllDepartments 获取所有部门
func (r *DepartmentRepository) GetAllDepartments(ctx context.Context) ([]*model.Department, error) {
	var departments []*model.Department
	err := r.db.WithContext(ctx).Order("sort ASC, id ASC").Find(&departments).Error
	if err != nil {
		return nil, err
	}
	return departments, nil
}

// GetDepartmentsByParentID 根据父部门ID获取子部门列表
func (r *DepartmentRepository) GetDepartmentsByParentID(ctx context.Context, parentID *int64) ([]*model.Department, error) {
	var departments []*model.Department
	if parentID == nil {
		err := r.db.WithContext(ctx).Where("parent_id IS NULL").Order("sort ASC, id ASC").Find(&departments).Error
		if err != nil {
			return nil, err
		}
	} else {
		err := r.db.WithContext(ctx).Where("parent_id = ?", *parentID).Order("sort ASC, id ASC").Find(&departments).Error
		if err != nil {
			return nil, err
		}
	}
	return departments, nil
}

// UpdateDepartment 更新部门
func (r *DepartmentRepository) UpdateDepartment(ctx context.Context, department *model.Department) error {
	return r.db.WithContext(ctx).Save(department).Error
}

// DeleteDepartment 删除部门（软删除）
func (r *DepartmentRepository) DeleteDepartment(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.Department{}, id).Error
}

// GetDepartmentTree 获取部门树（递归查询）
func (r *DepartmentRepository) GetDepartmentTree(ctx context.Context) ([]*model.Department, error) {
	var departments []*model.Department
	err := r.db.WithContext(ctx).Where("parent_id IS NULL").Order("sort ASC, id ASC").Find(&departments).Error
	if err != nil {
		return nil, err
	}

	// 递归加载子部门
	for _, dept := range departments {
		if err := r.loadChildren(ctx, dept); err != nil {
			return nil, err
		}
	}

	return departments, nil
}

// loadChildren 递归加载子部门
func (r *DepartmentRepository) loadChildren(ctx context.Context, department *model.Department) error {
	parentID := &department.ID
	children, err := r.GetDepartmentsByParentID(ctx, parentID)
	if err != nil {
		return err
	}

	department.Children = children

	// 递归加载子部门的子部门
	for _, child := range children {
		if err := r.loadChildren(ctx, child); err != nil {
			return err
		}
	}

	return nil
}
