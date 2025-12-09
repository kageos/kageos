package v1

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Knowledge 知识库 API 处理器
type Knowledge struct {
	service *service.KnowledgeService
}

// NewKnowledge 创建知识库 API 处理器
func NewKnowledge(service *service.KnowledgeService) *Knowledge {
	return &Knowledge{service: service}
}

// List 获取知识库列表
// @Summary 获取知识库列表
// @Description 获取所有知识库列表
// @Tags 知识库管理
// @Accept json
// @Produce json
// @Param page query int true "页码" default(1)
// @Param page_size query int true "每页数量" default(10)
// @Success 200 {object} dto.KnowledgeListResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/knowledge/list [get]
func (h *Knowledge) List(c *gin.Context) {
	var req dto.KnowledgeListReq
	var resp *dto.KnowledgeListResp
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Knowledge.List req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	kbs, total, err := h.service.ListKnowledgeBases(ctx, req.Page, req.PageSize)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 转换为响应格式
	kbInfos := make([]dto.KnowledgeInfo, 0, len(kbs))
	for _, kb := range kbs {
		kbInfos = append(kbInfos, dto.KnowledgeInfo{
			ID:            kb.ID,
			Name:          kb.Name,
			Description:   kb.Description,
			Status:        kb.Status,
			DocumentCount: kb.DocumentCount,
			ContentHash:   kb.ContentHash,
			User:          kb.User,
			CreatedAt:     time.Time(kb.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:     time.Time(kb.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		})
	}

	resp = &dto.KnowledgeListResp{
		KnowledgeBases: kbInfos,
		Total:          total,
	}
	response.OkWithData(c, resp)
}

// Get 获取知识库详情
// @Summary 获取知识库详情
// @Description 根据ID获取知识库详细信息
// @Tags 知识库管理
// @Accept json
// @Produce json
// @Param id query int true "知识库ID"
// @Success 200 {object} dto.KnowledgeGetResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/knowledge/get [get]
func (h *Knowledge) Get(c *gin.Context) {
	var req dto.KnowledgeGetReq
	var resp *dto.KnowledgeGetResp
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Knowledge.Get req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	kb, err := h.service.GetKnowledgeBase(ctx, req.ID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp = &dto.KnowledgeGetResp{
		KnowledgeInfo: dto.KnowledgeInfo{
			ID:            kb.ID,
			Name:          kb.Name,
			Description:   kb.Description,
			Status:        kb.Status,
			DocumentCount: kb.DocumentCount,
			ContentHash:   kb.ContentHash,
			User:          kb.User,
			CreatedAt:     time.Time(kb.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:     time.Time(kb.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		},
	}
	response.OkWithData(c, resp)
}

// Create 创建知识库
// @Summary 创建知识库
// @Description 创建新的知识库
// @Tags 知识库管理
// @Accept json
// @Produce json
// @Param request body dto.KnowledgeCreateReq true "创建知识库请求"
// @Success 200 {object} dto.KnowledgeCreateResp "创建成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/knowledge/create [post]
func (h *Knowledge) Create(c *gin.Context) {
	var req dto.KnowledgeCreateReq
	var resp *dto.KnowledgeCreateResp
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Knowledge.Create req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	kb := &model.KnowledgeBase{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
	}

	if err := h.service.CreateKnowledgeBase(ctx, kb); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp = &dto.KnowledgeCreateResp{ID: kb.ID}
	response.OkWithData(c, resp)
}

// Update 更新知识库
// @Summary 更新知识库
// @Description 更新知识库信息
// @Tags 知识库管理
// @Accept json
// @Produce json
// @Param request body dto.KnowledgeUpdateReq true "更新知识库请求"
// @Success 200 {object} dto.KnowledgeUpdateResp "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/knowledge/update [post]
func (h *Knowledge) Update(c *gin.Context) {
	var req dto.KnowledgeUpdateReq
	var resp *dto.KnowledgeUpdateResp
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Knowledge.Update req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	
	// 先获取现有知识库
	kb, err := h.service.GetKnowledgeBase(ctx, req.ID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 更新字段
	kb.Name = req.Name
	kb.Description = req.Description
	kb.Status = req.Status

	if err := h.service.UpdateKnowledgeBase(ctx, kb); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp = &dto.KnowledgeUpdateResp{ID: kb.ID}
	response.OkWithData(c, resp)
}

// Delete 删除知识库
// @Summary 删除知识库
// @Description 删除知识库
// @Tags 知识库管理
// @Accept json
// @Produce json
// @Param id query int true "知识库ID"
// @Success 200 "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/knowledge/delete [post]
func (h *Knowledge) Delete(c *gin.Context) {
	var req dto.KnowledgeDeleteReq
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Knowledge.Delete req:%+v err:%v", req, err)
	}()

	ctx := contextx.ToContext(c)
	if err := h.service.DeleteKnowledgeBase(ctx, req.ID); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "删除成功")
}

// AddDocument 添加文档
// @Summary 添加文档
// @Description 向知识库添加文档
// @Tags 知识库管理
// @Accept json
// @Produce json
// @Param request body dto.KnowledgeAddDocumentReq true "添加文档请求"
// @Success 200 {object} dto.KnowledgeAddDocumentResp "添加成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/knowledge/add_document [post]
func (h *Knowledge) AddDocument(c *gin.Context) {
	var req dto.KnowledgeAddDocumentReq
	var resp *dto.KnowledgeAddDocumentResp
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Knowledge.AddDocument req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	doc := &model.KnowledgeDocument{
		KnowledgeBaseID: req.KnowledgeBaseID,
		ParentID:        req.ParentID,
		Title:           req.Title,
		Content:         req.Content,
		FileType:        req.FileType,
		SortOrder:       req.SortOrder,
		// DocID 和 FileSize 由 Service 层自动生成和计算
	}

	if err := h.service.AddDocument(ctx, doc); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp = &dto.KnowledgeAddDocumentResp{DocID: doc.DocID}
	response.OkWithData(c, resp)
}

// ListDocuments 获取文档列表
// @Summary 获取文档列表
// @Description 获取知识库的文档列表
// @Tags 知识库管理
// @Accept json
// @Produce json
// @Param knowledge_base_id query int true "知识库ID"
// @Param page query int true "页码" default(1)
// @Param page_size query int true "每页数量" default(10)
// @Success 200 {object} dto.KnowledgeListDocumentsResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/knowledge/list_documents [get]
func (h *Knowledge) ListDocuments(c *gin.Context) {
	var req dto.KnowledgeListDocumentsReq
	var resp *dto.KnowledgeListDocumentsResp
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Knowledge.ListDocuments req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	docs, total, err := h.service.ListDocuments(ctx, req.KnowledgeBaseID, req.Page, req.PageSize)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 转换为响应格式
	docInfos := make([]dto.DocumentInfo, 0, len(docs))
	for _, doc := range docs {
		docInfos = append(docInfos, dto.DocumentInfo{
			ID:              doc.ID,
			KnowledgeBaseID: doc.KnowledgeBaseID,
			ParentID:        doc.ParentID,
			DocID:           doc.DocID,
			Title:           doc.Title,
			Content:         doc.Content,
			FileType:        doc.FileType,
			FileSize:        doc.FileSize,
			Status:          doc.Status,
			SortOrder:       doc.SortOrder,
			Path:            doc.Path,
			User:            doc.User,
			CreatedAt:       time.Time(doc.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:       time.Time(doc.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		})
	}

	resp = &dto.KnowledgeListDocumentsResp{
		Documents: docInfos,
		Total:     total,
	}
	response.OkWithData(c, resp)
}

// GetDocument 获取文档详情
// @Summary 获取文档详情
// @Description 根据ID获取文档详细信息
// @Tags 知识库管理
// @Accept json
// @Produce json
// @Param id query int true "文档ID"
// @Success 200 {object} dto.KnowledgeGetDocumentResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/knowledge/get_document [get]
func (h *Knowledge) GetDocument(c *gin.Context) {
	var req dto.KnowledgeGetDocumentReq
	var resp *dto.KnowledgeGetDocumentResp
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Knowledge.GetDocument req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	doc, err := h.service.GetDocument(ctx, req.ID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp = &dto.KnowledgeGetDocumentResp{
		DocumentInfo: dto.DocumentInfo{
			ID:              doc.ID,
			KnowledgeBaseID: doc.KnowledgeBaseID,
			ParentID:        doc.ParentID,
			DocID:           doc.DocID,
			Title:           doc.Title,
			Content:         doc.Content,
			FileType:        doc.FileType,
			FileSize:        doc.FileSize,
			Status:          doc.Status,
			SortOrder:       doc.SortOrder,
			Path:            doc.Path,
			User:            doc.User,
			CreatedAt:       time.Time(doc.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:       time.Time(doc.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		},
	}
	response.OkWithData(c, resp)
}

// UpdateDocument 更新文档
// @Summary 更新文档
// @Description 更新文档信息
// @Tags 知识库管理
// @Accept json
// @Produce json
// @Param request body dto.KnowledgeUpdateDocumentReq true "更新文档请求"
// @Success 200 {object} dto.KnowledgeUpdateDocumentResp "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/knowledge/update_document [post]
func (h *Knowledge) UpdateDocument(c *gin.Context) {
	var req dto.KnowledgeUpdateDocumentReq
	var resp *dto.KnowledgeUpdateDocumentResp
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Knowledge.UpdateDocument req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	
	// 先获取现有文档
	doc, err := h.service.GetDocument(ctx, req.ID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 更新字段
	doc.Title = req.Title
	doc.Content = req.Content
	doc.FileType = req.FileType
	doc.ParentID = req.ParentID
	doc.SortOrder = req.SortOrder
	if req.Status != "" {
		doc.Status = req.Status
	}

	if err := h.service.UpdateDocument(ctx, doc); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp = &dto.KnowledgeUpdateDocumentResp{ID: doc.ID}
	response.OkWithData(c, resp)
}

// DeleteDocument 删除文档
// @Summary 删除文档
// @Description 删除文档
// @Tags 知识库管理
// @Accept json
// @Produce json
// @Param request body dto.KnowledgeDeleteDocumentReq true "文档ID"
// @Success 200 "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/knowledge/delete_document [post]
func (h *Knowledge) DeleteDocument(c *gin.Context) {
	var req dto.KnowledgeDeleteDocumentReq
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Knowledge.DeleteDocument req:%+v err:%v", req, err)
	}()

	ctx := contextx.ToContext(c)
	if err := h.service.DeleteDocument(ctx, req.ID); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "删除成功")
}

// GetDocumentsTree 获取文档树
// @Summary 获取文档树
// @Description 获取知识库的文档树（目录结构）
// @Tags 知识库管理
// @Accept json
// @Produce json
// @Param knowledge_base_id query int true "知识库ID"
// @Success 200 {object} dto.KnowledgeGetDocumentsTreeResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/knowledge/get_documents_tree [get]
func (h *Knowledge) GetDocumentsTree(c *gin.Context) {
	var req dto.KnowledgeGetDocumentsTreeReq
	var resp *dto.KnowledgeGetDocumentsTreeResp
	var err error

	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Knowledge.GetDocumentsTree req:%+v resp:%+v err:%v", req, resp, err)
	}()

	ctx := contextx.ToContext(c)
	docs, err := h.service.GetDocumentsTree(ctx, req.KnowledgeBaseID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 构建树形结构
	docMap := make(map[int64]*dto.DocumentInfo)
	docInfos := make([]dto.DocumentInfo, 0, len(docs))

	// 第一遍：创建所有节点
	for _, doc := range docs {
		docInfo := dto.DocumentInfo{
			ID:              doc.ID,
			KnowledgeBaseID: doc.KnowledgeBaseID,
			ParentID:        doc.ParentID,
			DocID:           doc.DocID,
			Title:           doc.Title,
			Content:         doc.Content,
			FileType:        doc.FileType,
			FileSize:        doc.FileSize,
			Status:          doc.Status,
			SortOrder:       doc.SortOrder,
			Path:            doc.Path,
			User:            doc.User,
			CreatedAt:       time.Time(doc.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:       time.Time(doc.UpdatedAt).Format("2006-01-02T15:04:05Z"),
			Children:        []dto.DocumentInfo{},
		}
		docMap[doc.ID] = &docInfo
		docInfos = append(docInfos, docInfo)
	}

	// 第二遍：构建父子关系
	var rootNodes []dto.DocumentInfo
	for i := range docInfos {
		if docInfos[i].ParentID == 0 {
			// 根节点
			rootNodes = append(rootNodes, docInfos[i])
		} else {
			// 子节点，添加到父节点的 Children 中
			if parent, ok := docMap[docInfos[i].ParentID]; ok {
				parent.Children = append(parent.Children, docInfos[i])
			}
		}
	}

	// 递归排序（按 SortOrder 和 ID）
	var sortTree func(nodes *[]dto.DocumentInfo)
	sortTree = func(nodes *[]dto.DocumentInfo) {
		// 排序当前层级
		for i := 0; i < len(*nodes); i++ {
			for j := i + 1; j < len(*nodes); j++ {
				if (*nodes)[i].SortOrder > (*nodes)[j].SortOrder ||
					((*nodes)[i].SortOrder == (*nodes)[j].SortOrder && (*nodes)[i].ID > (*nodes)[j].ID) {
					(*nodes)[i], (*nodes)[j] = (*nodes)[j], (*nodes)[i]
				}
			}
			// 递归排序子节点
			if len((*nodes)[i].Children) > 0 {
				sortTree(&(*nodes)[i].Children)
			}
		}
	}
	sortTree(&rootNodes)

	resp = &dto.KnowledgeGetDocumentsTreeResp{
		Documents: rootNodes,
	}
	response.OkWithData(c, resp)
}

// UpdateDocumentsSort 批量更新文档排序
// @Summary 批量更新文档排序
// @Description 批量更新文档的排序、父节点和路径
// @Tags 知识库管理
// @Accept json
// @Produce json
// @Param request body dto.KnowledgeUpdateDocumentsSortReq true "更新排序请求"
// @Success 200 "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Router /api/v1/agent/knowledge/update_documents_sort [post]
func (h *Knowledge) UpdateDocumentsSort(c *gin.Context) {
	var req dto.KnowledgeUpdateDocumentsSortReq
	var err error

	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "参数错误: "+err.Error())
		return
	}

	defer func() {
		logger.Infof(c, "Knowledge.UpdateDocumentsSort req:%+v err:%v", req, err)
	}()

	ctx := contextx.ToContext(c)

	// 转换为 repository 需要的格式
	updates := make([]struct {
		ID        int64
		ParentID  int64
		SortOrder int
		Path      string
	}, len(req.Updates))
	for i, update := range req.Updates {
		updates[i] = struct {
			ID        int64
			ParentID  int64
			SortOrder int
			Path      string
		}{
			ID:        update.ID,
			ParentID:  update.ParentID,
			SortOrder: update.SortOrder,
			Path:      update.Path,
		}
	}

	if err := h.service.UpdateDocumentsSort(ctx, req.KnowledgeBaseID, updates); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "更新成功")
}

