package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// CreateDocs 创建 docs 类型节点（专门的接口）
func (s *ServiceTreeService) CreateDocs(ctx context.Context, req *dto.CreateDocsReq) (*dto.CreateDocsResp, error) {
	// 转换为通用请求格式
	createReq := &dto.CreateServiceTreeReq{
		User:               req.User,
		App:                req.App,
		Name:               req.Name,
		Code:               req.Code,
		ParentFullCodePath: req.ParentFullCodePath,
		Type:               model.ServiceTreeTypeDocs,
		Description:        req.Description,
		Tags:               req.Tags,
		Admins:             req.Admins,
		DocContent:         req.Content,
		DocFormat:          req.Format,
		DocSummary:         req.Summary,
	}

	// 调用通用创建方法
	resp, err := s.CreateDocsNode(ctx, createReq)
	if err != nil {
		return nil, err
	}

	// 获取创建的文档记录ID（从 ServiceTree.RefID 获取）
	docID := resp.RefID

	// 转换为专门的响应格式
	return &dto.CreateDocsResp{
		ID:           resp.ID,
		Name:         resp.Name,
		Code:         resp.Code,
		Type:         resp.Type,
		Description:  resp.Description,
		Tags:         resp.Tags,
		AppID:        resp.AppID,
		FullCodePath: resp.FullCodePath,
		Admins:       resp.Admins,
		DocID:        docID,
	}, nil
}

// docs 类型 code 后缀（与 form/table/chart 一致，便于路由与识别）
const codeSuffixDocs = ".docs"

// CreateDocsNode 创建文档节点（docs 类型）
// ⭐ 专门用于创建文档节点，不创建文件系统目录
func (s *ServiceTreeService) CreateDocsNode(ctx context.Context, req *dto.CreateServiceTreeReq) (*dto.CreateServiceTreeResp, error) {
	// 兜底：code 未带类型后缀时自动补全
	if req.Code != "" && !strings.HasSuffix(req.Code, codeSuffixDocs) {
		req.Code = req.Code + codeSuffixDocs
	}
	// 获取应用信息
	app, err := s.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	var parentTree *model.ServiceTree

	if req.ParentFullCodePath != "" {
		parentTree, err = s.serviceTreeRepo.GetServiceTreeByFullPath(req.ParentFullCodePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent node: %w", err)
		}
	}

	// 构建完整路径
	fullCodePath := fmt.Sprintf("/%s/%s/%s", app.User, app.Code, req.Code)
	if parentTree != nil {
		fullCodePath = parentTree.FullCodePath + "/" + req.Code
	}

	// 检查节点是否已存在
	docsParentPath := ""
	if parentTree != nil {
		docsParentPath = parentTree.FullCodePath
	}
	exists, err := s.serviceTreeRepo.CheckNameExistsByPath(docsParentPath, req.Code, app.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check name exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("docs node %s already exists", req.Code)
	}

	// 提取当前版本号数字
	currentVersionNum := extractVersionNumForServiceTree(app.Version)

	// 获取创建者用户名
	requestUser := contextx.GetRequestUser(ctx)

	// 创建文档节点记录（docs 类型）
	serviceTree := &model.ServiceTree{
		Name:             req.Name,
		Code:             req.Code,
		Type:             model.ServiceTreeTypeDocs,
		Description:      req.Description,
		Tags:             req.Tags,
		Admins:           req.Admins,
		AppID:            app.ID,
		FullCodePath:     fullCodePath,
		AddVersionNum:    currentVersionNum,
		UpdateVersionNum: 0,
	}

	// 设置创建者
	if requestUser != "" {
		serviceTree.CreatedBy = requestUser
	}

	// 保存到数据库
	if err := s.serviceTreeRepo.CreateServiceTreeWithParentPath(serviceTree, ""); err != nil {
		return nil, fmt.Errorf("failed to create docs node: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Created docs node: %s/%s/%s", req.User, req.App, req.Code)

	// ⭐ 自动给创建者和管理员分配管理员角色（拥有 directory:admin 权限）
	if requestUser != "" {
		if err := s.assignAdminRoleToUser(ctx, req.User, req.App, requestUser, serviceTree.FullCodePath); err != nil {
			logger.Warnf(ctx, "[ServiceTreeService] 自动添加创建者管理员角色失败: user=%s, app=%s, username=%s, resource=%s, error=%v",
				req.User, req.App, requestUser, serviceTree.FullCodePath, err)
		}
	}

	if req.Admins != "" {
		admins := strings.Split(req.Admins, ",")
		for _, admin := range admins {
			admin = strings.TrimSpace(admin)
			if admin != "" && admin != requestUser {
				if err := s.assignAdminRoleToUser(ctx, req.User, req.App, admin, serviceTree.FullCodePath); err != nil {
					logger.Warnf(ctx, "[ServiceTreeService] 自动添加管理员角色失败: user=%s, app=%s, username=%s, resource=%s, error=%v",
						req.User, req.App, admin, serviceTree.FullCodePath, err)
				}
			}
		}
	}

	// ⭐ docs 类型不需要创建文件系统目录，只创建数据库记录
	if req.DocContent != "" {
		docFormat := req.DocFormat
		if docFormat == "" {
			docFormat = "markdown"
		}
		docReq := &dto.CreateDocReq{
			FullCodePath: serviceTree.FullCodePath,
			Content:      req.DocContent,
			Format:       docFormat,
			Summary:      req.DocSummary,
		}
		doc, err := s.docService.CreateDoc(ctx, docReq)
		if err != nil {
			logger.Warnf(ctx, "[ServiceTreeService] 创建文档内容失败: %v", err)
		} else {
			if serviceTree.RefID == 0 {
				serviceTree.RefID = doc.ID
				if err := s.serviceTreeRepo.UpdateServiceTree(serviceTree); err != nil {
					logger.Warnf(ctx, "[ServiceTreeService] 更新 ServiceTree RefID 失败: %v", err)
				}
			}
			logger.Infof(ctx, "[ServiceTreeService] 文档内容创建成功 - FullCodePath: %s, DocID: %d", serviceTree.FullCodePath, doc.ID)
		}
	}

	return &dto.CreateServiceTreeResp{
		ID:           serviceTree.ID,
		Name:         serviceTree.Name,
		Code:         serviceTree.Code,
		Type:         serviceTree.Type,
		Description:  serviceTree.Description,
		Tags:         serviceTree.Tags,
		AppID:        serviceTree.AppID,
		FullCodePath: serviceTree.FullCodePath,
		Version:      serviceTree.Version,
		VersionNum:   serviceTree.VersionNum,
		Status:       "created",
	}, nil
}

// CreateBoard 创建版块（board）类型节点（专门接口）
func (s *ServiceTreeService) CreateBoard(ctx context.Context, req *dto.CreateBoardReq) (*dto.CreateBoardResp, error) {
	createReq := &dto.CreateServiceTreeReq{
		User:               req.User,
		App:                req.App,
		Name:               req.Name,
		Code:               req.Code,
		ParentFullCodePath: req.ParentFullCodePath,
		Type:               model.ServiceTreeTypeBoard,
		Description:        req.Description,
		Tags:               req.Tags,
		Admins:             req.Admins,
	}
	resp, err := s.CreateBoardNode(ctx, createReq)
	if err != nil {
		return nil, err
	}
	return &dto.CreateBoardResp{
		ID:           resp.ID,
		Name:         resp.Name,
		Code:         resp.Code,
		Type:         resp.Type,
		Description:  resp.Description,
		Tags:         resp.Tags,
		AppID:        resp.AppID,
		FullCodePath: resp.FullCodePath,
		Admins:       resp.Admins,
	}, nil
}

// board 类型 code 后缀（与 form/table/chart 一致）
const codeSuffixBoard = ".board"

// CreateBoardNode 创建版块节点（board 类型，不建 RefID，帖子在 board_posts 表）
func (s *ServiceTreeService) CreateBoardNode(ctx context.Context, req *dto.CreateServiceTreeReq) (*dto.CreateServiceTreeResp, error) {
	if req.Code != "" && !strings.HasSuffix(req.Code, codeSuffixBoard) {
		req.Code = req.Code + codeSuffixBoard
	}
	app, err := s.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}
	var parentTree *model.ServiceTree
	if req.ParentFullCodePath != "" {
		parentTree, err = s.serviceTreeRepo.GetServiceTreeByFullPath(req.ParentFullCodePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent node: %w", err)
		}
	}
	fullCodePath := fmt.Sprintf("/%s/%s/%s", app.User, app.Code, req.Code)
	if parentTree != nil {
		fullCodePath = parentTree.FullCodePath + "/" + req.Code
	}
	boardParentPath := ""
	if parentTree != nil {
		boardParentPath = parentTree.FullCodePath
	}
	exists, err := s.serviceTreeRepo.CheckNameExistsByPath(boardParentPath, req.Code, app.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check name exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("board node %s already exists", req.Code)
	}
	currentVersionNum := extractVersionNumForServiceTree(app.Version)
	requestUser := contextx.GetRequestUser(ctx)
	serviceTree := &model.ServiceTree{
		Name:             req.Name,
		Code:             req.Code,
		Type:             model.ServiceTreeTypeBoard,
		Description:      req.Description,
		Tags:             req.Tags,
		Admins:           req.Admins,
		AppID:            app.ID,
		FullCodePath:     fullCodePath,
		AddVersionNum:    currentVersionNum,
		UpdateVersionNum: 0,
	}
	if requestUser != "" {
		serviceTree.CreatedBy = requestUser
	}
	if err := s.serviceTreeRepo.CreateServiceTreeWithParentPath(serviceTree, ""); err != nil {
		return nil, fmt.Errorf("failed to create board node: %w", err)
	}
	logger.Infof(ctx, "[ServiceTreeService] Created board node: %s/%s/%s", req.User, req.App, req.Code)
	if requestUser != "" {
		if err := s.assignAdminRoleToUser(ctx, req.User, req.App, requestUser, serviceTree.FullCodePath); err != nil {
			logger.Warnf(ctx, "[ServiceTreeService] 自动添加创建者管理员角色失败: user=%s, app=%s, username=%s, resource=%s, error=%v",
				req.User, req.App, requestUser, serviceTree.FullCodePath, err)
		}
	}
	if req.Admins != "" {
		admins := strings.Split(req.Admins, ",")
		for _, admin := range admins {
			admin = strings.TrimSpace(admin)
			if admin != "" && admin != requestUser {
				if err := s.assignAdminRoleToUser(ctx, req.User, req.App, admin, serviceTree.FullCodePath); err != nil {
					logger.Warnf(ctx, "[ServiceTreeService] 自动添加管理员角色失败: user=%s, app=%s, username=%s, resource=%s, error=%v",
						req.User, req.App, admin, serviceTree.FullCodePath, err)
				}
			}
		}
	}
	return &dto.CreateServiceTreeResp{
		ID:           serviceTree.ID,
		Name:         serviceTree.Name,
		Code:         serviceTree.Code,
		Type:         serviceTree.Type,
		Description:  serviceTree.Description,
		Tags:         serviceTree.Tags,
		AppID:        serviceTree.AppID,
		FullCodePath: serviceTree.FullCodePath,
		Version:      serviceTree.Version,
		VersionNum:   serviceTree.VersionNum,
		Status:       "created",
	}, nil
}
