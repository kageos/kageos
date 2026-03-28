package service

import (
	"context"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

func convertToGetServiceTreeRespImpl(
	s *ServiceTreeService,
	ctx context.Context,
	tree *model.ServiceTree,
	permissionsMap map[string]map[string]bool,
	isAdmin bool,
) *dto.GetServiceTreeResp {
	resp := &dto.GetServiceTreeResp{
		ID:              tree.ID,
		Name:            tree.Name,
		Code:            tree.Code,
		RefID:           tree.RefID,
		Type:            tree.Type,
		Description:     tree.Description,
		Tags:            tree.Tags,
		Admins:          tree.Admins,
		PendingCount:    tree.PendingCount,
		Owner:           tree.CreatedBy,
		AppID:           tree.AppID,
		FullCodePath:    tree.FullCodePath,
		TemplateType:    tree.TemplateType,
		Version:         tree.Version,
		VersionNum:      tree.VersionNum,
		HubFullCodePath: tree.HubFullCodePath,
		HubVersionNum:   tree.HubVersionNum,
		RunCount:        tree.RunCount,
		IsAdmin:         isAdmin,
	}

	if tree.FullCodePath != "" {
		if permissionsMap != nil {
			if nodePerms, ok := permissionsMap[tree.FullCodePath]; ok {
				resp.Permissions = nodePerms
			} else {
				resp.Permissions = make(map[string]bool)
			}
		} else {
			resp.Permissions = make(map[string]bool)
		}
	} else {
		resp.Permissions = make(map[string]bool)
	}

	if len(tree.Children) > 0 {
		for _, child := range tree.Children {
			childResp := convertToGetServiceTreeRespImpl(s, ctx, child, permissionsMap, isAdmin)
			resp.Children = append(resp.Children, childResp)
		}
	}

	if tree.Type == model.ServiceTreeTypePackage {
		resp.HasFunction = s.hasFunctionInDirectChildren(tree)
	}

	return resp
}

func calculateExpandedKeysImpl(trees []*dto.GetServiceTreeResp) []int64 {
	expandedKeysMap := make(map[int64]bool)

	for _, tree := range trees {
		segments := strings.Split(strings.Trim(tree.FullCodePath, "/"), "/")
		if tree.Type == "package" && len(segments) == 2 {
			expandedKeysMap[tree.ID] = true
		}
	}

	var findNodesWithPending func(nodes []*dto.GetServiceTreeResp, parentPath []int64)
	findNodesWithPending = func(nodes []*dto.GetServiceTreeResp, parentPath []int64) {
		for _, node := range nodes {
			currentPath := append(parentPath, node.ID)

			if node.PendingCount > 0 {
				for _, id := range currentPath {
					expandedKeysMap[id] = true
				}
			}

			if len(node.Children) > 0 {
				findNodesWithPending(node.Children, currentPath)
			}
		}
	}

	findNodesWithPending(trees, []int64{})

	expandedKeys := make([]int64, 0, len(expandedKeysMap))
	for id := range expandedKeysMap {
		expandedKeys = append(expandedKeys, id)
	}

	return expandedKeys
}
