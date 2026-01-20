package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
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
func (s *DocService) CreateDoc(ctx context.Context, req *dto.CreateDocReq) (*model.Doc, error) {
	user := contextx.GetRequestUser(ctx)
	if user == "" {
		return nil, fmt.Errorf("请求用户信息不能为空")
	}

	// 1. 验证 ServiceTree 节点存在且为 docs 类型
	tree, err := s.serviceTreeRepo.GetByID(req.TreeID)
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
	existingDoc, err := s.docRepo.GetByTreeID(req.TreeID)
	if err == nil && existingDoc != nil {
		return nil, fmt.Errorf("文档已存在，请使用更新接口")
	}

	// 3. 设置默认格式
	format := req.Format
	if format == "" {
		format = "markdown"
	}

	// 4. 创建文档
	doc := &model.Doc{
		Title:   req.Title,
		Content: req.Content,
		Format:  format,
		Summary: req.Summary,
		AppID:   tree.AppID,
		TreeID:  req.TreeID,
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

	logger.Infof(ctx, "[DocService] 文档创建成功 - TreeID: %d, DocID: %d, Title: %s", req.TreeID, doc.ID, req.Title)
	return doc, nil
}

// UpdateDoc 更新文档（基于 TreeID）
func (s *DocService) UpdateDoc(ctx context.Context, req *dto.UpdateDocReq) (*model.Doc, error) {
	user := contextx.GetRequestUser(ctx)
	if user == "" {
		return nil, fmt.Errorf("请求用户信息不能为空")
	}

	// 如果提供了 FullCodePath，则使用基于路径的更新
	if req.FullCodePath != "" {
		return s.updateDocByPath(ctx, req)
	}

	// 否则使用基于 TreeID 的更新
	if req.TreeID == 0 {
		return nil, fmt.Errorf("TreeID 和 FullCodePath 必须提供其中一个")
	}

	// 1. 验证 ServiceTree 节点存在且为 docs 类型
	tree, err := s.serviceTreeRepo.GetByID(req.TreeID)
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
	doc, err := s.docRepo.GetByTreeID(req.TreeID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("文档不存在")
		}
		return nil, fmt.Errorf("获取文档失败: %w", err)
	}

	// 3. 更新文档（只更新非空字段）
	if req.Title != "" {
		doc.Title = req.Title
	}
	if req.Content != "" {
		doc.Content = req.Content
	}
	if req.Format != "" {
		doc.Format = req.Format
	}
	if req.Summary != "" {
		doc.Summary = req.Summary
	}
	doc.UpdatedBy = user

	if err := s.docRepo.Update(doc); err != nil {
		return nil, fmt.Errorf("更新文档失败: %w", err)
	}

	logger.Infof(ctx, "[DocService] 文档更新成功 - TreeID: %d, DocID: %d, Title: %s", req.TreeID, doc.ID, doc.Title)
	return doc, nil
}

// updateDocByPath 基于路径更新文档（内部方法）
func (s *DocService) updateDocByPath(ctx context.Context, req *dto.UpdateDocReq) (*model.Doc, error) {
	// 1. 根据路径获取 ServiceTree 节点
	tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.FullCodePath)
	if err != nil {
		return nil, fmt.Errorf("获取服务树节点失败: %w", err)
	}

	// 2. 验证节点类型为 docs
	if !tree.IsDocs() {
		return nil, fmt.Errorf("节点类型错误，期望 docs，实际 %s", tree.Type)
	}

	// 3. 填充 TreeID 并调用基于 ID 的更新逻辑
	req.TreeID = tree.ID
	req.FullCodePath = "" // 清空路径，避免死循环
	return s.UpdateDoc(ctx, req)
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

// ==================== 基于路径的文档操作（新接口，用于 /doc/*full-code-path） ====================

// GetDocByPath 根据完整路径获取文档
func (s *DocService) GetDocByPath(ctx context.Context, fullCodePath string) (*model.Doc, error) {
	// 1. 根据路径获取 ServiceTree 节点
	tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(fullCodePath)
	if err != nil {
		return nil, fmt.Errorf("获取服务树节点失败: %w", err)
	}

	// 2. 验证节点类型为 docs
	if !tree.IsDocs() {
		return nil, fmt.Errorf("节点类型错误，期望 docs，实际 %s", tree.Type)
	}

	// 3. 获取文档内容
	return s.GetDoc(ctx, tree.ID)
}

// UpdateDocByPath 根据完整路径更新文档（兼容性方法）
func (s *DocService) UpdateDocByPath(ctx context.Context, req *dto.UpdateDocReq) (*model.Doc, error) {
	// 直接调用 UpdateDoc，逻辑已整合
	return s.UpdateDoc(ctx, req)
}

// DeleteDocByPath 根据完整路径删除文档
func (s *DocService) DeleteDocByPath(ctx context.Context, fullCodePath string) error {
	// 1. 根据路径获取 ServiceTree 节点
	tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(fullCodePath)
	if err != nil {
		return fmt.Errorf("获取服务树节点失败: %w", err)
	}

	// 2. 验证节点类型为 docs
	if !tree.IsDocs() {
		return fmt.Errorf("节点类型错误，期望 docs，实际 %s", tree.Type)
	}

	// 3. 删除文档内容和节点
	return s.DeleteDoc(ctx, tree.ID)
}
