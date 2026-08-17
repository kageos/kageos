package service

import (
	"context"
	"fmt"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/gorm"
)

// DocService 文档服务
type DocService struct {
	docRepo         *repository.DocRepository
	serviceTreeRepo *repository.ServiceTreeRepository
	appRepo         *repository.AppRepository
	permission      *PermissionService
}

// NewDocService 创建文档服务
func NewDocService(docRepo *repository.DocRepository, serviceTreeRepo *repository.ServiceTreeRepository, appRepo *repository.AppRepository, permission *PermissionService) *DocService {
	return &DocService{
		docRepo:         docRepo,
		serviceTreeRepo: serviceTreeRepo,
		appRepo:         appRepo,
		permission:      permission,
	}
}

// GetDoc 获取文档（基于完整路径）
func (s *DocService) GetDoc(ctx context.Context, fullCodePath string) (*model.Docs, error) {
	// 1. 根据完整路径获取 ServiceTree 节点
	tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(fullCodePath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ServiceTree 节点不存在: %s", fullCodePath)
		}
		return nil, fmt.Errorf("获取 ServiceTree 节点失败: %w", err)
	}

	// 2. 验证节点类型为 docs
	if !tree.IsDocs() {
		return nil, fmt.Errorf("节点类型不是 docs，当前类型: %s", tree.Type)
	}

	// 3. 根据 TreeID 获取文档
	doc, err := s.docRepo.GetByTreeID(tree.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("文档不存在")
		}
		return nil, fmt.Errorf("获取文档失败: %w", err)
	}

	return doc, nil
}

// CreateDoc 创建文档（基于完整路径）
func (s *DocService) CreateDoc(ctx context.Context, req *dto.CreateDocReq) (*model.Docs, error) {
	user := contextx.GetRequestUser(ctx)
	if user == "" {
		return nil, fmt.Errorf("请求用户信息不能为空")
	}

	// 1. 根据完整路径获取 ServiceTree 节点
	tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.FullCodePath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ServiceTree 节点不存在: %s", req.FullCodePath)
		}
		return nil, fmt.Errorf("获取 ServiceTree 节点失败: %w", err)
	}

	// 2. 验证节点类型为 docs
	if !tree.IsDocs() {
		return nil, fmt.Errorf("节点类型不是 docs，当前类型: %s", tree.Type)
	}

	// 3. 检查文档是否已存在
	existingDoc, err := s.docRepo.GetByTreeID(tree.ID)
	if err == nil && existingDoc != nil {
		return nil, fmt.Errorf("文档已存在，请使用更新接口")
	}

	// 4. 设置默认格式
	format := req.Format
	if format == "" {
		format = "markdown"
	}

	// 5. 创建文档（名称从 ServiceTree 获取）
	doc := &model.Docs{
		Name:         tree.Name, // 使用 ServiceTree 的名称
		Content:      req.Content,
		Format:       format,
		Summary:      req.Summary,
		AppID:        tree.AppID,
		TreeID:       tree.ID,
		FullCodePath: tree.FullCodePath, // 同步完整路径
	}
	doc.CreatedBy = user
	doc.UpdatedBy = user

	if err := s.docRepo.Create(doc); err != nil {
		return nil, fmt.Errorf("创建文档失败: %w", err)
	}

	// 6. 更新 ServiceTree 的 RefID
	tree.RefID = doc.ID
	if err := s.serviceTreeRepo.UpdateServiceTree(tree); err != nil {
		logger.Warnf(ctx, "[DocService] 更新 ServiceTree RefID 失败: %v", err)
		// 不返回错误，因为文档已创建成功
	}

	logger.Infof(ctx, "[DocService] 文档创建成功 - FullCodePath: %s, DocID: %d, Name: %s", req.FullCodePath, doc.ID, doc.Name)
	return doc, nil
}

// UpdateDoc 更新文档（基于完整路径）
func (s *DocService) UpdateDoc(ctx context.Context, req *dto.UpdateDocReq) (*model.Docs, error) {
	user := contextx.GetRequestUser(ctx)
	if user == "" {
		return nil, fmt.Errorf("请求用户信息不能为空")
	}

	// 1. 根据完整路径获取 ServiceTree 节点
	tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.FullCodePath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("ServiceTree 节点不存在: %s", req.FullCodePath)
		}
		return nil, fmt.Errorf("获取 ServiceTree 节点失败: %w", err)
	}

	// 2. 验证节点类型为 docs
	if !tree.IsDocs() {
		return nil, fmt.Errorf("节点类型不是 docs，当前类型: %s", tree.Type)
	}

	// 3. 获取文档
	doc, err := s.docRepo.GetByTreeID(tree.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("文档不存在")
		}
		return nil, fmt.Errorf("获取文档失败: %w", err)
	}

	// 4. 更新文档（只更新非空字段）
	// 注意：文档名称应该通过更新 ServiceTree 来修改，这里不更新 Name
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

	logger.Infof(ctx, "[DocService] 文档更新成功 - FullCodePath: %s, DocID: %d, Name: %s", req.FullCodePath, doc.ID, doc.Name)
	return doc, nil
}

// DeleteDoc 删除文档（基于完整路径）
func (s *DocService) DeleteDoc(ctx context.Context, fullCodePath string) error {
	// 1. 根据完整路径获取 ServiceTree 节点
	tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(fullCodePath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("ServiceTree 节点不存在: %s", fullCodePath)
		}
		return fmt.Errorf("获取 ServiceTree 节点失败: %w", err)
	}

	// 2. 验证节点类型为 docs
	if !tree.IsDocs() {
		return fmt.Errorf("节点类型不是 docs，当前类型: %s", tree.Type)
	}

	// 3. 删除文档内容
	if err := s.docRepo.DeleteByTreeID(tree.ID); err != nil {
		return fmt.Errorf("删除文档失败: %w", err)
	}

	// 4. 删除 ServiceTree 节点（docs 类型的节点）
	if err := s.serviceTreeRepo.DeleteServiceTree(tree.ID); err != nil {
		logger.Warnf(ctx, "[DocService] 删除 ServiceTree 节点失败: %v", err)
		return fmt.Errorf("删除文档节点失败: %w", err)
	}

	logger.Infof(ctx, "[DocService] 文档及节点删除成功 - FullCodePath: %s", fullCodePath)
	return nil
}

// GetDocsByFullCodePath 根据 FullCodePath 获取文档列表（用于知识库加载）
func (s *DocService) GetDocsByFullCodePath(ctx context.Context, fullCodePath string) ([]*model.Docs, error) {
	// 1. 根据 FullCodePath 获取 ServiceTree 节点
	tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(fullCodePath)
	if err != nil {
		return nil, fmt.Errorf("获取 ServiceTree 节点失败: %w", err)
	}

	// 2. 获取该节点及其所有子节点中的 docs 类型节点
	docsNodes, err := s.serviceTreeRepo.GetDocsNodesByPathPrefix(tree.AppID, tree.FullCodePath)
	if err != nil {
		return nil, fmt.Errorf("获取 docs 节点失败: %w", err)
	}

	if len(docsNodes) == 0 {
		return []*model.Docs{}, nil
	}

	// 3. 获取所有 TreeID（只获取有 RefID 的节点，即已创建文档的节点）
	treeIDs := make([]int64, 0, len(docsNodes))
	for _, node := range docsNodes {
		if node.RefID > 0 { // 只获取有 RefID 的节点（已创建文档的节点）
			treeIDs = append(treeIDs, node.ID)
		}
	}

	if len(treeIDs) == 0 {
		return []*model.Docs{}, nil
	}

	// 4. 批量获取文档内容
	docs, err := s.docRepo.ListByTreeIDs(treeIDs)
	if err != nil {
		return nil, fmt.Errorf("获取文档内容失败: %w", err)
	}

	return docs, nil
}

// SearchDocs 搜索文档（模糊搜索）
func (s *DocService) SearchDocs(ctx context.Context, req *dto.SearchDocsReq) (*dto.SearchDocsResp, error) {
	// 验证分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100 // 限制最大每页数量
	}

	logger.Infof(ctx, "[DocService] 搜索文档 - Keyword: %s, Page: %d, PageSize: %d, IncludeContent: %v",
		req.Keyword, req.Page, req.PageSize, req.IncludeContent)

	// 调用 Repository 搜索文档
	docs, total, err := s.docRepo.SearchDocs(req.Keyword, req.Page, req.PageSize)
	if err != nil {
		logger.Errorf(ctx, "[DocService] 搜索文档失败 - Keyword: %s, Error: %v", req.Keyword, err)
		return nil, fmt.Errorf("搜索文档失败: %w", err)
	}

	// 转换为 DTO
	docItems := make([]*dto.DocItem, 0, len(docs))
	for _, doc := range docs {
		if !s.canReadPath(ctx, doc.FullCodePath) {
			continue
		}
		item := &dto.DocItem{
			ID:           doc.ID,
			Name:         doc.Name,
			Format:       doc.Format,
			FullCodePath: doc.FullCodePath,
			Summary:      doc.Summary,
			Category:     doc.Category,
		}

		// 根据 include_content 决定是否包含内容（默认 true）
		if req.IncludeContent {
			item.Content = doc.Content
		}

		docItems = append(docItems, item)
	}

	logger.Infof(ctx, "[DocService] 搜索文档成功 - Keyword: %s, Total: %d, DocsCount: %d, IncludeContent: %v",
		req.Keyword, total, len(docItems), req.IncludeContent)

	return &dto.SearchDocsResp{
		Docs:     docItems,
		Total:    int64(len(docItems)),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// BatchGetDocs 批量获取文档（精确查询）
func (s *DocService) BatchGetDocs(ctx context.Context, req *dto.BatchGetDocsReq) (*dto.BatchGetDocsResp, error) {
	logger.Infof(ctx, "[DocService] 批量获取文档 - Paths: %v, IncludeContent: %v", req.Paths, req.IncludeContent)

	// 直接根据 full_code_path 批量查询文档
	docs, err := s.docRepo.GetByFullCodePaths(req.Paths)
	if err != nil {
		logger.Errorf(ctx, "[DocService] 批量获取文档失败 - Paths: %v, Error: %v", req.Paths, err)
		return nil, fmt.Errorf("批量获取文档失败: %w", err)
	}

	// 转换为 DTO
	docItems := make([]*dto.DocItem, 0, len(docs))
	for _, doc := range docs {
		if !s.canReadPath(ctx, doc.FullCodePath) {
			continue
		}
		item := &dto.DocItem{
			ID:           doc.ID,
			Name:         doc.Name,
			Format:       doc.Format,
			FullCodePath: doc.FullCodePath,
			Summary:      doc.Summary,
			Category:     doc.Category,
		}

		// 根据 include_content 决定是否包含内容（默认 true）
		if req.IncludeContent {
			item.Content = doc.Content
		}

		docItems = append(docItems, item)
	}

	logger.Infof(ctx, "[DocService] 批量获取文档成功 - Paths: %v, DocsCount: %d, IncludeContent: %v",
		req.Paths, len(docItems), req.IncludeContent)

	return &dto.BatchGetDocsResp{Docs: docItems}, nil
}

func (s *DocService) canReadPath(ctx context.Context, fullCodePath string) bool {
	if s.permission == nil {
		return true
	}
	ok, err := s.permission.HasPermission(ctx, contextx.GetRequestUser(ctx), fullCodePath, access.ActionRead)
	return err == nil && ok
}

// ==================== 基于路径的文档操作（新接口，用于 /docs/*full-code-path） ====================
