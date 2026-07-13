package repository

import (
	"context"
	"fmt"

	"github.com/kageos/kageos/core/app-server/model"
	"gorm.io/gorm"
)

type FunctionRepository struct {
	db *gorm.DB
}

func NewFunctionRepository(db *gorm.DB) *FunctionRepository {
	return &FunctionRepository{db: db}
}

// CreateFunctions 批量创建函数记录
func (r *FunctionRepository) CreateFunctions(ctx context.Context, functions []*model.Function) error {
	if len(functions) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, function := range functions {
			if function == nil {
				continue
			}
			if err := createOrUpdateActiveFunction(tx, function); err != nil {
				return err
			}
		}
		return nil
	})
}

func createOrUpdateActiveFunction(tx *gorm.DB, function *model.Function) error {
	var activeFunctions []model.Function
	if err := tx.
		Where("app_id = ? AND method = ? AND router = ?", function.AppID, function.Method, function.Router).
		Order("id DESC").
		Find(&activeFunctions).Error; err != nil {
		return fmt.Errorf("query active function: app_id=%d method=%s router=%s: %w", function.AppID, function.Method, function.Router, err)
	}

	if len(activeFunctions) == 0 {
		return tx.Create(function).Error
	}

	keep := activeFunctions[0]
	if len(activeFunctions) > 1 {
		duplicateIDs := make([]int64, 0, len(activeFunctions)-1)
		for _, duplicate := range activeFunctions[1:] {
			duplicateIDs = append(duplicateIDs, duplicate.ID)
		}
		if err := softDeleteDuplicateActiveFunctions(tx, keep.ID, duplicateIDs); err != nil {
			return err
		}
	}

	updates := map[string]interface{}{
		"schema":              function.Schema,
		"has_config":          function.HasConfig,
		"create_tables":       function.CreateTables,
		"connectors":          function.Connectors,
		"connector_endpoints": function.ConnectorEndpoints,
		"template_type":       function.TemplateType,
		"updated_by":          function.UpdatedBy,
	}
	if err := tx.Model(&model.Function{}).
		Where("id = ?", keep.ID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("update active function: id=%d app_id=%d method=%s router=%s: %w", keep.ID, function.AppID, function.Method, function.Router, err)
	}
	function.ID = keep.ID
	return nil
}

func softDeleteDuplicateActiveFunctions(tx *gorm.DB, keepID int64, duplicateIDs []int64) error {
	if len(duplicateIDs) == 0 {
		return nil
	}
	if tx.Migrator().HasTable(&model.ServiceTree{}) {
		if err := tx.Table((&model.ServiceTree{}).TableName()).
			Where("ref_id IN ?", duplicateIDs).
			Update("ref_id", keepID).Error; err != nil {
			return fmt.Errorf("repoint duplicate function service tree refs: %w", err)
		}
	}
	if err := tx.Where("id IN ?", duplicateIDs).Delete(&model.Function{}).Error; err != nil {
		return fmt.Errorf("soft delete duplicate active functions: %w", err)
	}
	return nil
}

// UpdateFunctions 批量更新函数记录
func (r *FunctionRepository) UpdateFunctions(ctx context.Context, functions []*model.Function) error {
	if len(functions) == 0 {
		return nil
	}

	for _, function := range functions {
		updates := map[string]interface{}{
			"schema":              function.Schema,
			"has_config":          function.HasConfig,
			"create_tables":       function.CreateTables,
			"connectors":          function.Connectors,
			"connector_endpoints": function.ConnectorEndpoints,
			"template_type":       function.TemplateType,
		}
		err := r.db.WithContext(ctx).Model(&model.Function{}).
			Where("app_id = ? AND method = ? AND router = ?", function.AppID, function.Method, function.Router).
			Updates(updates).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteFunctions 根据条件删除函数记录
func (r *FunctionRepository) DeleteFunctions(ctx context.Context, appID int64, routers []string, methods []string) error {
	if len(routers) == 0 || len(methods) == 0 || len(routers) != len(methods) {
		return nil
	}

	for i := 0; i < len(routers); i++ {
		err := r.db.WithContext(ctx).Where("app_id = ? AND router = ? AND method = ?", appID, routers[i], methods[i]).
			Delete(&model.Function{}).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// GetFunctionByID 根据ID获取函数
func (r *FunctionRepository) GetFunctionByID(ctx context.Context, id int64) (*model.Function, error) {
	var function model.Function
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&function).Error
	if err != nil {
		return nil, err
	}
	return &function, nil
}

// GetFunctionsByIDs 根据 ID 列表批量获取函数
func (r *FunctionRepository) GetFunctionsByIDs(ctx context.Context, ids []int64) ([]*model.Function, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var functions []*model.Function
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&functions).Error
	if err != nil {
		return nil, err
	}
	return functions, nil
}

// GetFunctionsByAppID 获取应用的所有函数
func (r *FunctionRepository) GetFunctionsByAppID(ctx context.Context, appID int64) ([]*model.Function, error) {
	var functions []*model.Function
	err := r.db.WithContext(ctx).Where("app_id = ?", appID).Find(&functions).Error
	return functions, err
}

// FunctionExists 检查函数是否存在
func (r *FunctionRepository) FunctionExists(ctx context.Context, appID int64, method, router string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Function{}).
		Where("app_id = ? AND method = ? AND router = ?", appID, method, router).
		Count(&count).Error
	return count > 0, err
}

// GetFunctionByKey 根据app_id、method、router获取函数
// ⚠️ 注意：router 存储的是 full-code-path，已经包含了 user 和 app 信息
// 但为了兼容性和明确性，仍然保留 appID 参数（虽然可以通过 router 推导出来）
func (r *FunctionRepository) GetFunctionByKey(ctx context.Context, appID int64, method, router string) (*model.Function, error) {
	var function model.Function
	err := r.db.WithContext(ctx).Where("app_id = ? AND method = ? AND router = ?", appID, method, router).
		First(&function).Error
	if err != nil {
		return nil, err
	}
	return &function, nil
}

// GetFunctionByFullCodePath 根据 full-code-path 获取函数
// fullCodePath 存储的是完整的路径（如 /luobei/operations/tools/pdftools/to_images），已经包含了 user 和 app 信息
// full-code-path 是全局唯一的，不需要 method 参数
func (r *FunctionRepository) GetFunctionByFullCodePath(ctx context.Context, fullCodePath string) (*model.Function, error) {
	var function model.Function
	err := r.db.WithContext(ctx).Where("router = ?", fullCodePath).
		First(&function).Error
	if err != nil {
		return nil, err
	}
	return &function, nil
}
