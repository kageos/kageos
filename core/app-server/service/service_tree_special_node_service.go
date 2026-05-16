package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

const (
	codeSuffixDocs = ".docs"
)

type serviceTreeSpecialNodeService struct {
	serviceTreeRepo *repository.ServiceTreeRepository
	appRepo         *repository.AppRepository
	docService      *DocService
}

func newServiceTreeSpecialNodeService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	docService *DocService,
) *serviceTreeSpecialNodeService {
	return &serviceTreeSpecialNodeService{
		serviceTreeRepo: serviceTreeRepo,
		appRepo:         appRepo,
		docService:      docService,
	}
}

func (s *serviceTreeSpecialNodeService) CreateDocs(ctx context.Context, req *dto.CreateDocsReq) (*dto.CreateDocsResp, error) {
	resp, err := s.CreateDocsNode(ctx, &dto.CreateServiceTreeReq{
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
	})
	if err != nil {
		return nil, err
	}

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
		DocID:        resp.RefID,
	}, nil
}

func (s *serviceTreeSpecialNodeService) CreateDocsNode(ctx context.Context, req *dto.CreateServiceTreeReq) (*dto.CreateServiceTreeResp, error) {
	serviceTree, err := s.createSpecialNode(ctx, req, model.ServiceTreeTypeDocs, codeSuffixDocs)
	if err != nil {
		return nil, err
	}

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

	return toCreateServiceTreeResp(serviceTree), nil
}

func (s *serviceTreeSpecialNodeService) createSpecialNode(
	ctx context.Context,
	req *dto.CreateServiceTreeReq,
	nodeType string,
	codeSuffix string,
) (*model.ServiceTree, error) {
	if req.Code != "" && !strings.HasSuffix(req.Code, codeSuffix) {
		req.Code = req.Code + codeSuffix
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
	parentPath := ""
	if parentTree != nil {
		fullCodePath = parentTree.FullCodePath + "/" + req.Code
		parentPath = parentTree.FullCodePath
	}

	exists, err := s.serviceTreeRepo.CheckNameExistsByPath(parentPath, req.Code, app.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check name exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("%s node %s already exists", nodeType, req.Code)
	}

	requestUser := contextx.GetRequestUser(ctx)
	serviceTree := &model.ServiceTree{
		Name:             req.Name,
		Code:             req.Code,
		Type:             nodeType,
		Description:      req.Description,
		Tags:             req.Tags,
		Admins:           req.Admins,
		AppID:            app.ID,
		FullCodePath:     fullCodePath,
		AddVersionNum:    extractVersionNumForServiceTree(app.Version),
		UpdateVersionNum: 0,
	}
	if requestUser != "" {
		serviceTree.CreatedBy = requestUser
	}

	if err := s.serviceTreeRepo.CreateServiceTreeWithParentPath(serviceTree, ""); err != nil {
		return nil, fmt.Errorf("failed to create %s node: %w", nodeType, err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Created %s node: %s/%s/%s", nodeType, req.User, req.App, req.Code)
	s.assignNodeAdmins(ctx, req.User, req.App, requestUser, req.Admins, serviceTree.FullCodePath)

	return serviceTree, nil
}

func (s *serviceTreeSpecialNodeService) assignNodeAdmins(
	ctx context.Context,
	user string,
	app string,
	requestUser string,
	admins string,
	resourcePath string,
) {
	if requestUser != "" {
		if err := assignDirectoryAdminRoleToUser(ctx, user, app, requestUser, resourcePath); err != nil {
			logger.Warnf(ctx, "[ServiceTreeService] 自动添加创建者管理员角色失败: user=%s, app=%s, username=%s, resource=%s, error=%v",
				user, app, requestUser, resourcePath, err)
		}
	}

	for admin := range parseAdminUserSet(admins) {
		if admin == requestUser {
			continue
		}
		if err := assignDirectoryAdminRoleToUser(ctx, user, app, admin, resourcePath); err != nil {
			logger.Warnf(ctx, "[ServiceTreeService] 自动添加管理员角色失败: user=%s, app=%s, username=%s, resource=%s, error=%v",
				user, app, admin, resourcePath, err)
		}
	}
}

func toCreateServiceTreeResp(serviceTree *model.ServiceTree) *dto.CreateServiceTreeResp {
	return &dto.CreateServiceTreeResp{
		ID:           serviceTree.ID,
		Name:         serviceTree.Name,
		Code:         serviceTree.Code,
		Type:         serviceTree.Type,
		Description:  serviceTree.Description,
		Tags:         serviceTree.Tags,
		AppID:        serviceTree.AppID,
		RefID:        serviceTree.RefID,
		FullCodePath: serviceTree.FullCodePath,
		Version:      serviceTree.Version,
		VersionNum:   serviceTree.VersionNum,
		Admins:       serviceTree.Admins,
		Status:       "created",
	}
}
