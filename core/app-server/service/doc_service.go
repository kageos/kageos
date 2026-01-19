package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"gorm.io/gorm"
)

// DocService 文档服务
type DocService struct {
	docRepo        *repository.DocRepository
	serviceTreeRepo *repository.ServiceTreeRepository
	appRepo        *repository.AppRepository
}

// NewDocService 创建文档服务
func NewDocService(docRepo *repository.DocRepository, serviceTreeRepo *repository.ServiceTreeRepository, appRepo *repository.AppRepository) *DocService {
	return &DocService{
		docRepo:        docRepo,
		serviceTreeRepo: serviceTreeRepo,
		appRepo:        appRepo,
	}
}

// GetDoc 获取文档（根据 TreeID）
func (s *DocService) GetDoc(ctx context.Context, treeID int64) (*model.Doc, error) {
	// 1. 验证 ServiceTree 节点存在且为 docs 类型
	tree, err := s.serviceTreeRepo.GetByID(treeID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ServiceTree 节点不存在")
		}
		return nil, fmt.Errorf("获取 ServiceTree 节点失败: %w", err)
	}

	if !tree.IsDocs() {
		return nil, fmt.Errorf("节点类型不是 docs，当前类型: %s", tree.Type)
	}

	// 2. 根据 TreeID 获取文档
	doc, err := s.docRepo.GetByTreeID(treeID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("文档不存在")
		}
		return nil, fmt.Errorf("获取文档失败: %w", err)
	}

	return doc, nil
}

// CreateDoc 创建文档
func (s *DocService) CreateDoc(ctx context.Context, treeID int64, title, content, format string, summary ...string) (*model.Doc, error) {
	user := contextx.GetRequestUser(ctx)
	if user == "" {
		return nil, fmt.Errorf("请求用户信息不能为空")
	}

	// 1. 验证 ServiceTree 节点存在且为 docs 类型
	tree, err := s.serviceTreeRepo.GetByID(treeID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ServiceTree 节点不存在")
		}
		return nil, fmt.Errorf("获取 ServiceTree 节点失败: %w", err)
	}

	if !tree.IsDocs() {
		return nil, fmt.Errorf("节点类型不是 docs，当前类型: %s", tree.Type)
	}

	// 2. 检查文档是否已存在
	existingDoc, err := s.docRepo.GetByTreeID(treeID)
	if err == nil && existingDoc != nil {
		return nil, fmt.Errorf("文档已存在，请使用更新接口")
	}

	// 3. 设置默认格式
	if format == "" {
		format = "markdown"
	}

	// 4. 创建文档
	doc := &model.Doc{
		Title:   title,
		Content: content,
		Format:  format,
		AppID:   tree.AppID,
		TreeID:  treeID,
	}
	if len(summary) > 0 && summary[0] != "" {
		doc.Summary = summary[0]
	}
	doc.CreatedBy = user
	doc.UpdatedBy = user

	if err := s.docRepo.Create(doc); err != nil {
		return nil, fmt.Errorf("创建文档失败: %w", err)
	}

	// 5. 更新 ServiceTree 的 RefID
	tree.RefID = doc.ID
	if err := s.serviceTreeRepo.UpdateServiceTree(tree); err != nil {
		logger.Warnf(ctx, "[DocService] 更新 ServiceTree RefID 失败: %v", err)
		// 不返回错误，因为文档已创建成功
	}

	logger.Infof(ctx, "[DocService] 文档创建成功 - TreeID: %d, DocID: %d, Title: %s", treeID, doc.ID, title)
	return doc, nil
}

// UpdateDoc 更新文档
func (s *DocService) UpdateDoc(ctx context.Context, treeID int64, title, content, format string) (*model.Doc, error) {
	user := contextx.GetRequestUser(ctx)
	if user == "" {
		return nil, fmt.Errorf("请求用户信息不能为空")
	}

	// 1. 验证 ServiceTree 节点存在且为 docs 类型
	tree, err := s.serviceTreeRepo.GetByID(treeID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ServiceTree 节点不存在")
		}
		return nil, fmt.Errorf("获取 ServiceTree 节点失败: %w", err)
	}

	if !tree.IsDocs() {
		return nil, fmt.Errorf("节点类型不是 docs，当前类型: %s", tree.Type)
	}

	// 2. 获取文档
	doc, err := s.docRepo.GetByTreeID(treeID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("文档不存在")
		}
		return nil, fmt.Errorf("获取文档失败: %w", err)
	}

	// 3. 更新文档
	if title != "" {
		doc.Title = title
	}
	if content != "" {
		doc.Content = content
	}
	if format != "" {
		doc.Format = format
	}
	doc.UpdatedBy = user

	if err := s.docRepo.Update(doc); err != nil {
		return nil, fmt.Errorf("更新文档失败: %w", err)
	}

	logger.Infof(ctx, "[DocService] 文档更新成功 - TreeID: %d, DocID: %d, Title: %s", treeID, doc.ID, doc.Title)
	return doc, nil
}

// DeleteDoc 删除文档
func (s *DocService) DeleteDoc(ctx context.Context, treeID int64) error {
	// 1. 验证 ServiceTree 节点存在且为 docs 类型
	tree, err := s.serviceTreeRepo.GetByID(treeID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("ServiceTree 节点不存在")
		}
		return fmt.Errorf("获取 ServiceTree 节点失败: %w", err)
	}

	if !tree.IsDocs() {
		return fmt.Errorf("节点类型不是 docs，当前类型: %s", tree.Type)
	}

	// 2. 删除文档内容
	if err := s.docRepo.DeleteByTreeID(treeID); err != nil {
		return fmt.Errorf("删除文档失败: %w", err)
	}

	// 3. 删除 ServiceTree 节点（docs 类型的节点）
	if err := s.serviceTreeRepo.DeleteServiceTree(treeID); err != nil {
		logger.Warnf(ctx, "[DocService] 删除 ServiceTree 节点失败: %v", err)
		return fmt.Errorf("删除文档节点失败: %w", err)
	}

	logger.Infof(ctx, "[DocService] 文档及节点删除成功 - TreeID: %d", treeID)
	return nil
}

// GetDocsByFullCodePath 根据 FullCodePath 获取文档列表（用于知识库加载）
func (s *DocService) GetDocsByFullCodePath(ctx context.Context, fullCodePath string) ([]*model.Doc, error) {
	// 1. 根据 FullCodePath 获取 ServiceTree 节点
	tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(fullCodePath)
	if err != nil {
		return nil, fmt.Errorf("获取 ServiceTree 节点失败: %w", err)
	}

	// 2. 获取该节点及其所有子节点中的 docs 类型节点
	docsNodes, err := s.serviceTreeRepo.GetDocsNodesByParentID(tree.ID)
	if err != nil {
		return nil, fmt.Errorf("获取 docs 节点失败: %w", err)
	}

	if len(docsNodes) == 0 {
		return []*model.Doc{}, nil
	}

	// 3. 获取所有 TreeID（只获取有 RefID 的节点，即已创建文档的节点）
	treeIDs := make([]int64, 0, len(docsNodes))
	for _, node := range docsNodes {
		if node.RefID > 0 { // 只获取有 RefID 的节点（已创建文档的节点）
			treeIDs = append(treeIDs, node.ID)
		}
	}

	if len(treeIDs) == 0 {
		return []*model.Doc{}, nil
	}

	// 4. 批量获取文档内容
	docs, err := s.docRepo.ListByTreeIDs(treeIDs)
	if err != nil {
		return nil, fmt.Errorf("获取文档内容失败: %w", err)
	}

	return docs, nil
}
