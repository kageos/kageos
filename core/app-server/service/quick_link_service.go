package service

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// QuickLinkService 快链服务
type QuickLinkService struct {
	quickLinkRepo *repository.QuickLinkRepository
}

// NewQuickLinkService 创建快链服务（依赖注入）
func NewQuickLinkService(quickLinkRepo *repository.QuickLinkRepository) *QuickLinkService {
	return &QuickLinkService{
		quickLinkRepo: quickLinkRepo,
	}
}

// CreateQuickLink 创建快链
func (s *QuickLinkService) CreateQuickLink(username string, name string, functionRouter string, functionMethod string, templateType string, requestParams map[string]interface{}, fieldMetadata map[string]interface{}, metadata map[string]interface{}) (*model.QuickLink, error) {
	quickLink := &model.QuickLink{
		CreatedBy:      username,
		Name:           name,
		FunctionRouter: functionRouter,
		FunctionMethod: functionMethod,
		TemplateType:   templateType,
	}

	// 设置请求参数
	if err := quickLink.SetRequestParams(requestParams); err != nil {
		logger.Errorf(nil, "[QuickLinkService] Failed to set request params: %v", err)
		return nil, err
	}

	// 设置字段元数据
	if fieldMetadata != nil && len(fieldMetadata) > 0 {
		if err := quickLink.SetFieldMetadata(fieldMetadata); err != nil {
			logger.Errorf(nil, "[QuickLinkService] Failed to set field metadata: %v", err)
			return nil, err
		}
	}

	// 设置其他元数据
	if metadata != nil && len(metadata) > 0 {
		if err := quickLink.SetMetadata(metadata); err != nil {
			logger.Errorf(nil, "[QuickLinkService] Failed to set metadata: %v", err)
			return nil, err
		}
	}

	// 保存到数据库
	if err := s.quickLinkRepo.Create(quickLink); err != nil {
		logger.Errorf(nil, "[QuickLinkService] Failed to create quick link: %v", err)
		return nil, err
	}

	return quickLink, nil
}

// GetQuickLink 获取快链（不验证用户，用于公开访问）
func (s *QuickLinkService) GetQuickLink(id int64) (*model.QuickLink, error) {
	return s.quickLinkRepo.GetByID(id)
}

// GetQuickLinkByUser 获取快链（验证用户，确保用户只能访问自己的快链）
func (s *QuickLinkService) GetQuickLinkByUser(id int64, username string) (*model.QuickLink, error) {
	return s.quickLinkRepo.GetByIDAndUser(id, username)
}

// ListQuickLinks 获取快链列表
func (s *QuickLinkService) ListQuickLinks(username string, functionRouter string, page, pageSize int) ([]*model.QuickLink, int64, error) {
	// 限制分页参数
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return s.quickLinkRepo.ListByUser(username, functionRouter, page, pageSize)
}

// UpdateQuickLink 更新快链
func (s *QuickLinkService) UpdateQuickLink(id int64, username string, name string, requestParams map[string]interface{}, fieldMetadata map[string]interface{}, metadata map[string]interface{}) (*model.QuickLink, error) {
	// 获取快链（验证用户）
	quickLink, err := s.quickLinkRepo.GetByIDAndUser(id, username)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if name != "" {
		quickLink.Name = name
	}

	if requestParams != nil {
		if err := quickLink.SetRequestParams(requestParams); err != nil {
			return nil, err
		}
	}

	if fieldMetadata != nil {
		if err := quickLink.SetFieldMetadata(fieldMetadata); err != nil {
			return nil, err
		}
	}

	if metadata != nil {
		if err := quickLink.SetMetadata(metadata); err != nil {
			return nil, err
		}
	}

	// 保存更新
	if err := s.quickLinkRepo.Update(quickLink); err != nil {
		return nil, err
	}

	return quickLink, nil
}

// DeleteQuickLink 删除快链
func (s *QuickLinkService) DeleteQuickLink(id int64, username string) error {
	return s.quickLinkRepo.Delete(id, username)
}

