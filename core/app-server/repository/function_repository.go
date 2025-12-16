package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

type FunctionRepository struct {
	db *gorm.DB
}

func NewFunctionRepository(db *gorm.DB) *FunctionRepository {
	return &FunctionRepository{db: db}
}

// CreateFunctions 批量创建函数记录
func (r *FunctionRepository) CreateFunctions(functions []*model.Function) error {
	if len(functions) == 0 {
		return nil
	}
	return r.db.Create(&functions).Error
}

// UpdateFunctions 批量更新函数记录
func (r *FunctionRepository) UpdateFunctions(functions []*model.Function) error {
	if len(functions) == 0 {
		return nil
	}

		for _, function := range functions {
			updates := map[string]interface{}{
				"request":       function.Request,
				"response":      function.Response,
				"has_config":    function.HasConfig,
				"create_tables": function.CreateTables,
				"callbacks":     function.Callbacks,
				"template_type": function.TemplateType,
			}
			err := r.db.Model(&model.Function{}).
				Where("app_id = ? AND method = ? AND router = ?", function.AppID, function.Method, function.Router).
				Updates(updates).Error
			if err != nil {
				return err
			}
		}
	return nil
}

// DeleteFunctions 根据条件删除函数记录
func (r *FunctionRepository) DeleteFunctions(appID int64, routers []string, methods []string) error {
	if len(routers) == 0 || len(methods) == 0 || len(routers) != len(methods) {
		return nil
	}

	for i := 0; i < len(routers); i++ {
		err := r.db.Where("app_id = ? AND router = ? AND method = ?", appID, routers[i], methods[i]).
			Delete(&model.Function{}).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// GetFunctionByID 根据ID获取函数
func (r *FunctionRepository) GetFunctionByID(id int64) (*model.Function, error) {
	var function model.Function
	err := r.db.Where("id = ?", id).First(&function).Error
	if err != nil {
		return nil, err
	}
	return &function, nil
}

// GetFunctionsByAppID 获取应用的所有函数
func (r *FunctionRepository) GetFunctionsByAppID(appID int64) ([]*model.Function, error) {
	var functions []*model.Function
	err := r.db.Where("app_id = ?", appID).Find(&functions).Error
	return functions, err
}

// FunctionExists 检查函数是否存在
func (r *FunctionRepository) FunctionExists(appID int64, method, router string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Function{}).
		Where("app_id = ? AND method = ? AND router = ?", appID, method, router).
		Count(&count).Error
	return count > 0, err
}

// GetFunctionByKey 根据app_id、method、router获取函数
func (r *FunctionRepository) GetFunctionByKey(appID int64, method, router string) (*model.Function, error) {
	var function model.Function
	err := r.db.Where("app_id = ? AND method = ? AND router = ?", appID, method, router).
		First(&function).Error
	if err != nil {
		return nil, err
	}
	return &function, nil
}

