package repository

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"gorm.io/gorm"
)

type SourceCodeRepository struct {
	db *gorm.DB
}

func NewSourceCodeRepository(db *gorm.DB) *SourceCodeRepository {
	return &SourceCodeRepository{db: db}
}

// GetOrCreate 根据 FullGroupCode 获取或创建 SourceCode 记录
// 如果存在则更新，不存在则创建
func (r *SourceCodeRepository) GetOrCreate(sourceCode *model.SourceCode) (*model.SourceCode, error) {
	ctx := context.Background()
	logger.Infof(ctx, "[SourceCodeRepository.GetOrCreate] 开始: fullGroupCode=%s, appID=%d, version=%s, contentLength=%d",
		sourceCode.FullGroupCode, sourceCode.AppID, sourceCode.Version, len(sourceCode.Content))

	var existing model.SourceCode
	err := r.db.Where("full_group_code = ?", sourceCode.FullGroupCode).First(&existing).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 不存在，创建新记录
			logger.Infof(ctx, "[SourceCodeRepository.GetOrCreate] 记录不存在，创建新记录: fullGroupCode=%s", sourceCode.FullGroupCode)
			if err := r.db.Create(sourceCode).Error; err != nil {
				logger.Errorf(ctx, "[SourceCodeRepository.GetOrCreate] 创建失败: fullGroupCode=%s, error=%v", sourceCode.FullGroupCode, err)
				return nil, err
			}
			logger.Infof(ctx, "[SourceCodeRepository.GetOrCreate] 创建成功: ID=%d, fullGroupCode=%s, contentLength=%d",
				sourceCode.ID, sourceCode.FullGroupCode, len(sourceCode.Content))
			return sourceCode, nil
		}
		logger.Errorf(ctx, "[SourceCodeRepository.GetOrCreate] 查询失败: fullGroupCode=%s, error=%v", sourceCode.FullGroupCode, err)
		return nil, err
	}

	// 存在，更新内容
	logger.Infof(ctx, "[SourceCodeRepository.GetOrCreate] 记录已存在，更新: ID=%d, fullGroupCode=%s, oldContentLength=%d, newContentLength=%d",
		existing.ID, existing.FullGroupCode, len(existing.Content), len(sourceCode.Content))

	existing.Content = sourceCode.Content
	existing.Version = sourceCode.Version
	existing.FullPath = sourceCode.FullPath
	existing.GroupCode = sourceCode.GroupCode

	if err := r.db.Save(&existing).Error; err != nil {
		logger.Errorf(ctx, "[SourceCodeRepository.GetOrCreate] 更新失败: ID=%d, fullGroupCode=%s, error=%v", existing.ID, existing.FullGroupCode, err)
		return nil, err
	}

	logger.Infof(ctx, "[SourceCodeRepository.GetOrCreate] 更新成功: ID=%d, fullGroupCode=%s, contentLength=%d",
		existing.ID, existing.FullGroupCode, len(existing.Content))

	return &existing, nil
}

// GetByFullGroupCode 根据 FullGroupCode 获取 SourceCode 记录
func (r *SourceCodeRepository) GetByFullGroupCode(fullGroupCode string) (*model.SourceCode, error) {
	var sourceCode model.SourceCode
	err := r.db.Where("full_group_code = ?", fullGroupCode).First(&sourceCode).Error
	if err != nil {
		return nil, err
	}
	return &sourceCode, nil
}

// GetByID 根据 ID 获取 SourceCode 记录
func (r *SourceCodeRepository) GetByID(id int64) (*model.SourceCode, error) {
	var sourceCode model.SourceCode
	err := r.db.Where("id = ?", id).First(&sourceCode).Error
	if err != nil {
		return nil, err
	}
	return &sourceCode, nil
}

// GetByFunctionID 根据 Function ID 获取 SourceCode 记录
func (r *SourceCodeRepository) GetByFunctionID(functionID int64) (*model.SourceCode, error) {
	var function model.Function
	err := r.db.Where("id = ?", functionID).First(&function).Error
	if err != nil {
		return nil, err
	}

	if function.SourceCodeID == nil || *function.SourceCodeID == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return r.GetByID(*function.SourceCodeID)
}

// GetByGroupCode 根据 AppID 和 GroupCode 获取 SourceCode 记录
func (r *SourceCodeRepository) GetByGroupCode(appID int64, fullPath, groupCode string) (*model.SourceCode, error) {
	// 构建 FullGroupCode：{full_path}/{group_code}
	fullGroupCode := fullPath + "/" + groupCode
	return r.GetByFullGroupCode(fullGroupCode)
}
