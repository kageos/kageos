package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/gorm"
)

type capabilityBundleDocInstallItem struct {
	FullCodePath       string
	ParentFullCodePath string
	Code               string
	Name               string
	Description        string
	Tags               string
	Content            string
	Format             string
	Summary            string
	Category           string
}

func (s *serviceTreeCapabilityBundleService) appendCapabilityBundleDocs(
	ctx context.Context,
	bundle *dto.CapabilityBundle,
	baseTree *model.ServiceTree,
	nodes []*model.ServiceTree,
	includeBaseCode bool,
	seenDocs map[string]struct{},
) error {
	if s.docService == nil || s.docService.docRepo == nil {
		return nil
	}

	docNodeIDs := make([]int64, 0)
	docNodesByID := make(map[int64]*model.ServiceTree)
	for _, node := range nodes {
		if node == nil || node.Type != model.ServiceTreeTypeDocs {
			continue
		}
		docNodeIDs = append(docNodeIDs, node.ID)
		docNodesByID[node.ID] = node
	}
	if len(docNodeIDs) == 0 {
		return nil
	}

	docs, err := s.docService.docRepo.ListByTreeIDs(ctx, docNodeIDs)
	if err != nil {
		return fmt.Errorf("获取文档内容失败: %w", err)
	}
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		node := docNodesByID[doc.TreeID]
		if node == nil {
			continue
		}
		relativePath, err := capabilityRelativeTreeNodePath(baseTree, node, includeBaseCode)
		if err != nil {
			return err
		}
		if relativePath == "" {
			continue
		}
		if _, exists := seenDocs[relativePath]; exists {
			continue
		}
		seenDocs[relativePath] = struct{}{}
		bundle.Docs = append(bundle.Docs, &dto.CapabilityBundleDoc{
			RelativePath: relativePath,
			Name:         doc.Name,
			Content:      doc.Content,
			Format:       doc.Format,
			Summary:      doc.Summary,
			Category:     doc.Category,
		})
	}
	return nil
}

func (s *serviceTreeCapabilityBundleService) installCapabilityBundleDocs(
	ctx context.Context,
	targetApp *model.App,
	items []*capabilityBundleDocInstallItem,
	overwrite bool,
) ([]string, error) {
	if s.docService == nil || s.docService.docRepo == nil {
		return nil, fmt.Errorf("文档服务未初始化")
	}

	createdPaths := make([]string, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(ctx, item.FullCodePath)
		switch {
		case err == nil:
			if tree.Type != model.ServiceTreeTypeDocs {
				return nil, fmt.Errorf("目标路径已存在且不是 docs: %s", item.FullCodePath)
			}
			if !overwrite {
				return nil, fmt.Errorf("目标文档已存在: %s", item.FullCodePath)
			}
			if err := s.updateCapabilityBundleDocTree(ctx, tree, item); err != nil {
				return nil, err
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			tree, err = s.createCapabilityBundleDocTree(ctx, targetApp, item)
			if err != nil {
				return nil, err
			}
			createdPaths = append(createdPaths, tree.FullCodePath)
		default:
			return nil, fmt.Errorf("检查目标文档失败: %w", err)
		}

		if err := s.upsertCapabilityBundleDocContent(ctx, tree, item); err != nil {
			return nil, err
		}
	}
	return createdPaths, nil
}

func (s *serviceTreeCapabilityBundleService) createCapabilityBundleDocTree(
	ctx context.Context,
	targetApp *model.App,
	item *capabilityBundleDocInstallItem,
) (*model.ServiceTree, error) {
	if targetApp == nil {
		return nil, fmt.Errorf("目标应用不能为空")
	}
	if _, err := s.serviceTreeRepo.GetServiceTreeByFullPath(ctx, item.ParentFullCodePath); err != nil {
		return nil, fmt.Errorf("目标文档父目录不存在: %s: %w", item.ParentFullCodePath, err)
	}
	requestUser := contextx.GetRequestUser(ctx)
	tree := &model.ServiceTree{
		Name:             item.Name,
		Code:             item.Code,
		Type:             model.ServiceTreeTypeDocs,
		Description:      item.Description,
		Tags:             item.Tags,
		AppID:            targetApp.ID,
		FullCodePath:     item.FullCodePath,
		AddVersionNum:    extractVersionNumForServiceTree(targetApp.Version),
		UpdateVersionNum: 0,
	}
	if requestUser != "" {
		tree.CreatedBy = requestUser
		tree.UpdatedBy = requestUser
	}
	if err := s.serviceTreeRepo.CreateServiceTreeWithParentPath(ctx, tree, ""); err != nil {
		return nil, fmt.Errorf("创建文档节点失败: %w", err)
	}
	return tree, nil
}

func (s *serviceTreeCapabilityBundleService) updateCapabilityBundleDocTree(
	ctx context.Context,
	tree *model.ServiceTree,
	item *capabilityBundleDocInstallItem,
) error {
	if tree == nil || item == nil {
		return nil
	}
	tree.Name = item.Name
	tree.Description = item.Description
	tree.Tags = item.Tags
	if requestUser := contextx.GetRequestUser(ctx); requestUser != "" {
		tree.UpdatedBy = requestUser
	}
	if err := s.serviceTreeRepo.UpdateServiceTree(ctx, tree); err != nil {
		return fmt.Errorf("更新文档节点失败: %w", err)
	}
	return nil
}

func (s *serviceTreeCapabilityBundleService) upsertCapabilityBundleDocContent(
	ctx context.Context,
	tree *model.ServiceTree,
	item *capabilityBundleDocInstallItem,
) error {
	if tree == nil || item == nil {
		return nil
	}
	doc, err := s.docService.docRepo.GetByTreeID(ctx, tree.ID)
	switch {
	case err == nil:
		doc.Name = tree.Name
		doc.Content = item.Content
		doc.Format = defaultCapabilityBundleDocFormat(item.Format)
		doc.Summary = item.Summary
		doc.Category = item.Category
		doc.AppID = tree.AppID
		doc.FullCodePath = tree.FullCodePath
		if requestUser := contextx.GetRequestUser(ctx); requestUser != "" {
			doc.UpdatedBy = requestUser
		}
		if err := s.docService.docRepo.Update(ctx, doc); err != nil {
			return fmt.Errorf("更新文档内容失败: %w", err)
		}
		tree.RefID = doc.ID
	case errors.Is(err, gorm.ErrRecordNotFound):
		doc = &model.Docs{
			Name:         tree.Name,
			Content:      item.Content,
			Format:       defaultCapabilityBundleDocFormat(item.Format),
			Summary:      item.Summary,
			Category:     item.Category,
			AppID:        tree.AppID,
			TreeID:       tree.ID,
			FullCodePath: tree.FullCodePath,
		}
		if requestUser := contextx.GetRequestUser(ctx); requestUser != "" {
			doc.CreatedBy = requestUser
			doc.UpdatedBy = requestUser
		}
		if err := s.docService.docRepo.Create(ctx, doc); err != nil {
			return fmt.Errorf("创建文档内容失败: %w", err)
		}
		tree.RefID = doc.ID
	default:
		return fmt.Errorf("获取文档内容失败: %w", err)
	}
	if err := s.serviceTreeRepo.UpdateServiceTree(ctx, tree); err != nil {
		logger.Warnf(ctx, "[CapabilityBundle] 更新文档节点 RefID 失败: %v", err)
	}
	return nil
}

func defaultCapabilityBundleDocFormat(format string) string {
	if strings.TrimSpace(format) == "" {
		return "markdown"
	}
	return strings.TrimSpace(format)
}
