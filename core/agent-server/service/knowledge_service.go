package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/utils"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// KnowledgeService 知识库服务
type KnowledgeService struct {
	repo *repository.KnowledgeRepository
}

// NewKnowledgeService 创建知识库服务
func NewKnowledgeService(repo *repository.KnowledgeRepository) *KnowledgeService {
	return &KnowledgeService{repo: repo}
}

// GetKnowledgeBase 获取知识库（根据 ID，保留兼容）
func (s *KnowledgeService) GetKnowledgeBase(ctx context.Context, id int64) (*model.KnowledgeBase, error) {
	kb, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("知识库不存在")
		}
		return nil, fmt.Errorf("获取知识库失败: %w", err)
	}
	return kb, nil
}


// ListKnowledgeBases 获取知识库列表
func (s *KnowledgeService) ListKnowledgeBases(ctx context.Context, scope string, page, pageSize int) ([]*model.KnowledgeBase, int64, error) {
	currentUser := contextx.GetRequestUser(ctx)
	offset := (page - 1) * pageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	return s.repo.List(scope, currentUser, offset, pageSize)
}

// CreateKnowledgeBase 创建知识库
func (s *KnowledgeService) CreateKnowledgeBase(ctx context.Context, kb *model.KnowledgeBase) error {
	// 获取用户信息
	user := contextx.GetRequestUser(ctx)
	kb.CreatedBy = user
	kb.UpdatedBy = user
	kb.User = user // 设置 User 字段

	// 验证必填字段
	if kb.Name == "" {
		return fmt.Errorf("知识库名称不能为空")
	}

	// 设置默认值
	if kb.Status == "" {
		kb.Status = "active"
	}
	if kb.DocumentCount == 0 {
		kb.DocumentCount = 0
	}

	// 设置默认管理员（如果为空，设置为创建用户）
	if kb.Admin == "" {
		kb.Admin = user
	}

	return s.repo.Create(kb)
}

// UpdateKnowledgeBase 更新知识库
func (s *KnowledgeService) UpdateKnowledgeBase(ctx context.Context, kb *model.KnowledgeBase) error {
	// 获取用户信息
	user := contextx.GetRequestUser(ctx)
	kb.UpdatedBy = user

	// 检查权限：只有管理员可以修改资源
	existing, err := s.repo.GetByID(kb.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("知识库不存在")
		}
		return fmt.Errorf("获取知识库失败: %w", err)
	}

	if !utils.IsAdmin(existing.Admin, user) {
		return fmt.Errorf("无权限：只有管理员可以修改此资源")
	}

	// 验证必填字段
	if kb.Name == "" {
		return fmt.Errorf("知识库名称不能为空")
	}

	return s.repo.Update(kb)
}

// DeleteKnowledgeBase 删除知识库（根据 ID，保留兼容）
func (s *KnowledgeService) DeleteKnowledgeBase(ctx context.Context, id int64) error {
	// 检查权限：只有管理员可以删除资源
	user := contextx.GetRequestUser(ctx)
	existing, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("知识库不存在")
		}
		return fmt.Errorf("获取知识库失败: %w", err)
	}

	if !utils.IsAdmin(existing.Admin, user) {
		return fmt.Errorf("无权限：只有管理员可以删除此资源")
	}

	return s.repo.Delete(id)
}


// AddDocument 添加文档
func (s *KnowledgeService) AddDocument(ctx context.Context, doc *model.KnowledgeDocument) error {
	// 获取用户信息
	user := contextx.GetRequestUser(ctx)
	doc.CreatedBy = user
	doc.UpdatedBy = user
	doc.User = user // 设置 User 字段

	// 验证必填字段
	if doc.KnowledgeBaseID == 0 {
		return fmt.Errorf("知识库ID不能为空")
	}
	if doc.Title == "" {
		return fmt.Errorf("文档标题不能为空")
	}
	if doc.Content == "" {
		return fmt.Errorf("文档内容不能为空")
	}

	// 自动生成 DocID（如果为空）
	if doc.DocID == "" {
		doc.DocID = uuid.New().String()
	}

	// 自动计算文件大小（根据 Content 的字节长度）
	if doc.FileSize == 0 {
		doc.FileSize = int64(len([]byte(doc.Content)))
	}

	// 设置默认值：文档写入数据库即表示已完成
	if doc.Status == "" {
		doc.Status = "completed"
	}

	// 确保所有文档都在根目录下（parent_id = 0）
	if doc.ParentID != 0 {
		doc.ParentID = 0
	}

	return s.repo.AddDocument(doc)
}

// ListDocuments 获取文档列表（根据 ID，保留兼容）
func (s *KnowledgeService) ListDocuments(ctx context.Context, kbID int64, page, pageSize int) ([]*model.KnowledgeDocument, int64, error) {
	offset := (page - 1) * pageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	return s.repo.GetDocumentsByKBID(kbID, offset, pageSize)
}

// GetDocument 获取文档（根据 ID）
func (s *KnowledgeService) GetDocument(ctx context.Context, id int64) (*model.KnowledgeDocument, error) {
	doc, err := s.repo.GetDocumentByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("文档不存在")
		}
		return nil, fmt.Errorf("获取文档失败: %w", err)
	}
	return doc, nil
}

// UpdateDocument 更新文档
func (s *KnowledgeService) UpdateDocument(ctx context.Context, doc *model.KnowledgeDocument) error {
	// 获取用户信息
	user := contextx.GetRequestUser(ctx)
	doc.UpdatedBy = user

	// 验证必填字段
	if doc.ID == 0 {
		return fmt.Errorf("文档ID不能为空")
	}
	if doc.Title == "" {
		return fmt.Errorf("文档标题不能为空")
	}
	if doc.Content == "" {
		return fmt.Errorf("文档内容不能为空")
	}

	// 自动计算文件大小（根据 Content 的字节长度）
	doc.FileSize = int64(len([]byte(doc.Content)))

	// 设置默认状态
	if doc.Status == "" {
		doc.Status = "completed"
	}

	return s.repo.UpdateDocument(doc)
}

// DeleteDocument 删除文档
func (s *KnowledgeService) DeleteDocument(ctx context.Context, id int64) error {
	if id == 0 {
		return fmt.Errorf("文档ID不能为空")
	}
	return s.repo.DeleteDocument(id)
}

// GetDocumentsTree 获取文档树（目录结构）
func (s *KnowledgeService) GetDocumentsTree(ctx context.Context, kbID int64) ([]*model.KnowledgeDocument, error) {
	if kbID == 0 {
		return nil, fmt.Errorf("知识库ID不能为空")
	}
	return s.repo.GetDocumentsTreeByKBID(kbID)
}

// UpdateDocumentsSort 批量更新文档排序
func (s *KnowledgeService) UpdateDocumentsSort(ctx context.Context, kbID int64, updates []struct {
	ID        int64
	ParentID  int64
	SortOrder int
	Path      string
}) error {
	if kbID == 0 {
		return fmt.Errorf("知识库ID不能为空")
	}
	if len(updates) == 0 {
		return fmt.Errorf("更新列表不能为空")
	}
	return s.repo.UpdateDocumentSort(updates)
}


