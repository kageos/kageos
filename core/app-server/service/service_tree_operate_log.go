package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
)

func (s *ServiceTreeService) getServiceTreeForAudit(ctx context.Context, id int64) *model.ServiceTree {
	if id <= 0 || s == nil || s.mutationService == nil || s.mutationService.serviceTreeRepo == nil {
		return nil
	}
	serviceTree, err := s.mutationService.serviceTreeRepo.GetServiceTreeByID(ctx, id)
	if err != nil {
		return nil
	}
	return serviceTree
}

func (s *ServiceTreeService) getServiceTreeForAuditByPath(ctx context.Context, fullCodePath string) *model.ServiceTree {
	if fullCodePath == "" || s == nil || s.mutationService == nil || s.mutationService.serviceTreeRepo == nil {
		return nil
	}
	serviceTree, err := s.mutationService.serviceTreeRepo.GetServiceTreeByFullPath(ctx, fullCodePath)
	if err != nil {
		return nil
	}
	return serviceTree
}

func (s *ServiceTreeService) resolveAddFunctionsAuditPath(ctx context.Context, req *dto.AddFunctionsReq) string {
	if req == nil || strings.TrimSpace(req.FullCodePath) == "" {
		return ""
	}
	targetTree := s.getServiceTreeForAuditByPath(ctx, strings.TrimSpace(req.FullCodePath))
	fallbackFileName := ""
	if targetTree != nil {
		fallbackFileName = targetTree.Code
	}
	fileName, err := normalizeAddFunctionsGoFileName(req.FileName, fallbackFileName)
	if err != nil {
		logger.Warnf(ctx, "[ServiceTreeAudit] resolve add_functions audit path failed: %v", err)
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(req.FullCodePath), "/") + "/" + fileName
}

func (s *ServiceTreeService) writeServiceTreeOperateLog(ctx context.Context, action string, oldNode, newNode *model.ServiceTree) {
	if s == nil || s.teamAccessService == nil {
		return
	}
	node := newNode
	if node == nil {
		node = oldNode
	}
	if node == nil || node.FullCodePath == "" {
		return
	}
	tenantUser, app, err := access.ParseUserApp(node.FullCodePath)
	if err != nil {
		logger.Warnf(ctx, "[ServiceTreeAudit] parse resource path failed: path=%s err=%v", node.FullCodePath, err)
		return
	}

	nodeType := normalizeServiceTreeAuditResourceType(node.Type)
	actor := contextx.GetRequestUser(ctx)
	var oldValues interface{}
	if oldNode != nil {
		oldValues = serviceTreeNodeLogValues(oldNode)
	}
	var newValues interface{}
	if newNode != nil {
		newValues = serviceTreeNodeLogValues(newNode)
	}
	s.teamAccessService.writeOperateLog(ctx, operateLogInput{
		TenantUser:   tenantUser,
		App:          app,
		ActorUser:    actor,
		Action:       action,
		ResourceType: nodeType,
		ResourcePath: node.FullCodePath,
		ResourceName: node.Name,
		TargetID:     fmt.Sprintf("%d", node.ID),
		Summary:      buildServiceTreeAuditSummary(actor, action, node),
		Status:       "success",
		Details: dto.ServiceTreeNodeLogDetails{
			NodeID:       node.ID,
			NodeType:     node.Type,
			FullCodePath: node.FullCodePath,
		},
		OldValues: oldValues,
		NewValues: newValues,
	})
}

func serviceTreeNodeLogValues(node *model.ServiceTree) *dto.ServiceTreeNodeLogValues {
	if node == nil {
		return nil
	}
	return &dto.ServiceTreeNodeLogValues{
		ID:           node.ID,
		Type:         node.Type,
		Name:         node.Name,
		Code:         node.Code,
		Description:  node.Description,
		Tags:         node.Tags,
		Admins:       node.Admins,
		AppID:        node.AppID,
		RefID:        node.RefID,
		FullCodePath: node.FullCodePath,
		TemplateType: node.TemplateType,
		Version:      node.Version,
		VersionNum:   node.VersionNum,
	}
}

func normalizeServiceTreeAuditResourceType(nodeType string) string {
	switch nodeType {
	case model.ServiceTreeTypePackage:
		return "directory"
	case model.ServiceTreeTypeFunction:
		return "function"
	case model.ServiceTreeTypeDocs:
		return "docs"
	default:
		return nodeType
	}
}

func buildServiceTreeAuditSummary(actor, action string, node *model.ServiceTree) string {
	if actor == "" {
		actor = "unknown"
	}
	switch action {
	case "service_tree.node.created":
		return fmt.Sprintf("%s created %s %s", actor, node.Type, node.FullCodePath)
	case "service_tree.node.updated":
		return fmt.Sprintf("%s updated %s %s", actor, node.Type, node.FullCodePath)
	case "service_tree.node.deleted":
		return fmt.Sprintf("%s deleted %s %s", actor, node.Type, node.FullCodePath)
	default:
		return fmt.Sprintf("%s executed %s on %s", actor, action, node.FullCodePath)
	}
}
