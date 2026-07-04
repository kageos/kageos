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

const docSeedPolicyCreateIfMissing = "create_if_missing"

type packageDocSeedItem struct {
	FullCodePath       string
	ParentFullCodePath string
	Code               string
	Name               string
	Description        string
	Tags               string
	Content            string
	Format             string
	Summary            string
	Policy             string
}

func (a *AppService) reconcilePackageDocs(ctx context.Context, state *appMetadataSyncState, packages []*dto.PackageInfo) error {
	if len(packages) == 0 {
		return nil
	}
	if !packagesHaveDocs(packages) {
		return nil
	}
	if a.serviceTreeRepo == nil || a.docService == nil || a.docService.docRepo == nil {
		return fmt.Errorf("文档种子同步需要 serviceTreeRepo 和 docService")
	}
	for _, pkg := range packages {
		if pkg == nil || len(pkg.Docs) == 0 {
			continue
		}
		for _, docConfig := range pkg.Docs {
			item, err := buildPackageDocSeedItem(pkg, docConfig)
			if err != nil {
				return err
			}
			if err := a.upsertPackageDocSeed(ctx, state, item); err != nil {
				return err
			}
		}
	}
	return nil
}

func packagesHaveDocs(packages []*dto.PackageInfo) bool {
	for _, pkg := range packages {
		if pkg != nil && len(pkg.Docs) > 0 {
			return true
		}
	}
	return false
}

func buildPackageDocSeedItem(pkg *dto.PackageInfo, docConfig dto.DocSeedConfig) (*packageDocSeedItem, error) {
	if pkg == nil {
		return nil, fmt.Errorf("文档种子缺少 package")
	}
	parentPath := strings.TrimRight(strings.TrimSpace(pkg.FullPath), "/")
	if parentPath == "" {
		return nil, fmt.Errorf("文档种子缺少 package full_path")
	}
	code, err := normalizePackageDocSeedCode(docConfig.Code)
	if err != nil {
		return nil, err
	}
	content := docConfig.Content
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("文档种子 %s/%s content 不能为空", parentPath, code)
	}
	policy := strings.TrimSpace(docConfig.Policy)
	if policy == "" {
		policy = docSeedPolicyCreateIfMissing
	}
	if policy != docSeedPolicyCreateIfMissing {
		return nil, fmt.Errorf("文档种子 %s/%s policy 不支持: %s", parentPath, code, policy)
	}
	name := strings.TrimSpace(docConfig.Name)
	if name == "" {
		name = strings.TrimSuffix(code, codeSuffixDocs)
	}
	format := strings.TrimSpace(docConfig.Format)
	if format == "" {
		format = "markdown"
	}
	return &packageDocSeedItem{
		FullCodePath:       parentPath + "/" + code,
		ParentFullCodePath: parentPath,
		Code:               code,
		Name:               name,
		Description:        strings.TrimSpace(docConfig.Description),
		Tags:               strings.TrimSpace(docConfig.Tags),
		Content:            content,
		Format:             format,
		Summary:            strings.TrimSpace(docConfig.Summary),
		Policy:             policy,
	}, nil
}

func normalizePackageDocSeedCode(code string) (string, error) {
	code = strings.Trim(strings.TrimSpace(code), "/")
	if code == "" {
		return "", fmt.Errorf("文档种子 code 不能为空")
	}
	if strings.ContainsAny(code, `/\`) {
		return "", fmt.Errorf("文档种子 code 必须是单段路径: %s", code)
	}
	if !strings.HasSuffix(code, codeSuffixDocs) {
		code += codeSuffixDocs
	}
	return code, nil
}

func (a *AppService) upsertPackageDocSeed(ctx context.Context, state *appMetadataSyncState, item *packageDocSeedItem) error {
	tree, err := a.serviceTreeRepo.GetServiceTreeByFullPath(item.FullCodePath)
	switch {
	case err == nil:
		if tree.Type != model.ServiceTreeTypeDocs {
			return fmt.Errorf("文档种子目标路径已存在且不是 docs: %s", item.FullCodePath)
		}
		if _, err := a.docService.docRepo.GetByTreeID(tree.ID); err == nil {
			logger.Infof(ctx, "[PackageDocs] skip existing docs full_code_path=%s", item.FullCodePath)
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("查询文档种子内容失败 %s: %w", item.FullCodePath, err)
		}
		return a.createPackageDocSeedContent(ctx, state, tree, item)
	case errors.Is(err, gorm.ErrRecordNotFound):
		parent, parentErr := a.serviceTreeRepo.GetServiceTreeByFullPath(item.ParentFullCodePath)
		if parentErr != nil {
			return fmt.Errorf("文档种子父目录不存在 %s: %w", item.ParentFullCodePath, parentErr)
		}
		tree = &model.ServiceTree{
			AppID:            parent.AppID,
			Type:             model.ServiceTreeTypeDocs,
			Code:             item.Code,
			Name:             item.Name,
			Description:      item.Description,
			Tags:             item.Tags,
			FullCodePath:     item.FullCodePath,
			AddVersionNum:    state.currentVersionNum,
			UpdateVersionNum: 0,
		}
		requestUser := requestUserForPackageDoc(ctx, state)
		if requestUser != "" {
			tree.CreatedBy = requestUser
			tree.UpdatedBy = requestUser
			tree.Admins = requestUser
		}
		if err := a.serviceTreeRepo.CreateServiceTreeWithParentPath(tree, ""); err != nil {
			return fmt.Errorf("创建文档种子节点失败 %s: %w", item.FullCodePath, err)
		}
		return a.createPackageDocSeedContent(ctx, state, tree, item)
	default:
		return fmt.Errorf("检查文档种子目标失败 %s: %w", item.FullCodePath, err)
	}
}

func (a *AppService) createPackageDocSeedContent(ctx context.Context, state *appMetadataSyncState, tree *model.ServiceTree, item *packageDocSeedItem) error {
	if tree == nil || item == nil {
		return nil
	}
	requestUser := requestUserForPackageDoc(ctx, state)
	doc := &model.Docs{
		Name:         tree.Name,
		Content:      item.Content,
		Format:       item.Format,
		Summary:      item.Summary,
		AppID:        tree.AppID,
		TreeID:       tree.ID,
		FullCodePath: tree.FullCodePath,
	}
	if requestUser != "" {
		doc.CreatedBy = requestUser
		doc.UpdatedBy = requestUser
	}
	if err := a.docService.docRepo.Create(doc); err != nil {
		return fmt.Errorf("创建文档种子内容失败 %s: %w", item.FullCodePath, err)
	}
	tree.RefID = doc.ID
	if err := a.serviceTreeRepo.UpdateServiceTree(tree); err != nil {
		logger.Warnf(ctx, "[PackageDocs] 更新文档种子 RefID 失败 full_code_path=%s err=%v", item.FullCodePath, err)
	}
	logger.Infof(ctx, "[PackageDocs] created docs seed full_code_path=%s doc_id=%d", item.FullCodePath, doc.ID)
	return nil
}

func requestUserForPackageDoc(ctx context.Context, state *appMetadataSyncState) string {
	requestUser := strings.TrimSpace(contextx.GetRequestUser(ctx))
	if requestUser == "" && state != nil && state.requestUser != "" {
		requestUser = state.requestUser
	}
	if requestUser == "" && state != nil && state.app != nil {
		requestUser = state.app.User
	}
	if requestUser == "" {
		requestUser = "system"
	}
	return requestUser
}
