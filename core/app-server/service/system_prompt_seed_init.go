package service

import (
	"context"
	"os"
	"path"
	"strings"

	agentprompt "github.com/ai-agent-os/ai-agent-os/core/agent-server/prompt"
	appmodel "github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

var obsoletePromptSeedDocPaths = []string{
	"/system/prompt/doc/doc-catalog.docs",
	"/system/prompt/doc/doc-catalog/index.docs",
	"/system/prompt/doc/文档目录.docs",
	"/system/prompt/doc/文档目录/index.docs",
	"/system/prompt/doc/工作台环境模板.docs",
	"/system/prompt/doc/工作台环境模板/index.docs",
	"/system/prompt/平台介绍.docs",
	"/system/prompt/平台介绍/index.docs",
	"/system/prompt/平台的横切能力.docs",
	"/system/prompt/平台的横切能力/index.docs",
}

func initSystemPromptSeed(ctx context.Context, serviceTreeService *ServiceTreeService) error {
	if serviceTreeService == nil {
		return nil
	}
	deleteObsoletePromptSeedArtifacts(ctx, serviceTreeService)
	if !shouldSyncSystemPromptSeed(ctx, serviceTreeService) {
		logger.Infof(ctx, "[SystemWorkspace] APP_ENV 非 dev 且 prompt 已初始化，跳过 system/prompt upsert")
		return nil
	}

	seedPackages, err := agentprompt.ListSystemPromptSeedPackages()
	if err != nil {
		return err
	}
	for _, seedPackage := range seedPackages {
		if err := upsertPromptSeedPackage(ctx, serviceTreeService, seedPackage); err != nil {
			return err
		}
	}

	seedDocs, err := agentprompt.ListSystemPromptSeedDocs()
	if err != nil {
		return err
	}
	for _, seedDoc := range seedDocs {
		if err := upsertPromptSeedDoc(ctx, serviceTreeService, seedDoc); err != nil {
			return err
		}
	}
	logger.Infof(ctx, "[SystemWorkspace] system/prompt 已同步 %d 个目录、%d 篇种子文档", len(seedPackages), len(seedDocs))
	return nil
}

func shouldSyncSystemPromptSeed(ctx context.Context, serviceTreeService *ServiceTreeService) bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "dev") {
		return true
	}
	sentinelPath := agentprompt.PromptDocLeafPath(agentprompt.SystemPromptWorkspaceEnvTemplatePath)
	if sentinelPath == "" {
		return true
	}
	_, err := serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{FullCodePath: sentinelPath})
	return err != nil
}

func upsertPromptSeedDoc(ctx context.Context, serviceTreeService *ServiceTreeService, seedDoc agentprompt.PromptSeedDoc) error {
	if err := deleteLegacyPromptSeedDoc(ctx, serviceTreeService, seedDoc); err != nil {
		return err
	}

	detail, err := serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{FullCodePath: seedDoc.ActualPath})
	if err == nil && detail != nil {
		name := seedDoc.Name
		description := seedDoc.Description
		admins := SystemUsername
		content := seedDoc.Content
		format := seedDoc.Format
		return serviceTreeService.UpdateDocs(ctx, &dto.UpdateDocsReq{
			ID:          detail.ID,
			Name:        &name,
			Description: &description,
			Admins:      &admins,
			Content:     &content,
			Format:      &format,
		})
	}

	parentPath := path.Dir(seedDoc.ActualPath)
	if parentPath == "." {
		parentPath = ""
	}
	_, err = serviceTreeService.CreateDocs(ctx, &dto.CreateDocsReq{
		User:               SystemUsername,
		App:                "prompt",
		Name:               seedDoc.Name,
		Code:               path.Base(seedDoc.ActualPath),
		ParentFullCodePath: parentPath,
		Description:        seedDoc.Description,
		Content:            seedDoc.Content,
		Format:             seedDoc.Format,
		Admins:             SystemUsername,
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}
	return nil
}

func deleteObsoletePromptSeedArtifacts(ctx context.Context, serviceTreeService *ServiceTreeService) {
	existingDocs, err := getExistingPromptSeedDocsByPaths(serviceTreeService, obsoletePromptSeedDocPaths)
	if err != nil {
		logger.Warnf(ctx, "[SystemWorkspace] 查询废弃 prompt 文档失败: err=%v", err)
		return
	}
	for _, fullCodePath := range obsoletePromptSeedDocPaths {
		doc := existingDocs[fullCodePath]
		if doc == nil {
			continue
		}
		if err := serviceTreeService.DeleteDocs(ctx, doc.ID); err != nil {
			logger.Warnf(ctx, "[SystemWorkspace] 删除废弃 prompt 文档失败: path=%s err=%v", fullCodePath, err)
		}
	}
}

func getExistingPromptSeedDocsByPaths(serviceTreeService *ServiceTreeService, fullCodePaths []string) (map[string]*appmodel.ServiceTree, error) {
	result := make(map[string]*appmodel.ServiceTree, len(fullCodePaths))
	if serviceTreeService == nil || serviceTreeService.queryView == nil || serviceTreeService.queryView.serviceTreeRepo == nil {
		return result, nil
	}

	nodes, err := serviceTreeService.queryView.serviceTreeRepo.GetServiceTreesByFullPaths(fullCodePaths)
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		if node == nil || node.Type != appmodel.ServiceTreeTypeDocs {
			continue
		}
		result[node.FullCodePath] = node
	}
	return result, nil
}

func deleteLegacyPromptSeedDoc(ctx context.Context, serviceTreeService *ServiceTreeService, seedDoc agentprompt.PromptSeedDoc) error {
	legacyPath := agentprompt.PromptDocLeafPath(seedDoc.LogicalPath)
	if legacyPath == "" || legacyPath == seedDoc.ActualPath {
		return nil
	}
	existingDocs, err := getExistingPromptSeedDocsByPaths(serviceTreeService, []string{legacyPath})
	if err != nil {
		return err
	}
	detail := existingDocs[legacyPath]
	if detail == nil {
		return nil
	}
	return serviceTreeService.DeleteDocs(ctx, detail.ID)
}

func upsertPromptSeedPackage(ctx context.Context, serviceTreeService *ServiceTreeService, seedPackage agentprompt.PromptSeedPackage) error {
	detail, err := serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{FullCodePath: seedPackage.LogicalPath})
	if err == nil && detail != nil {
		name := seedPackage.Name
		description := seedPackage.Description
		admins := SystemUsername
		return serviceTreeService.UpdatePackage(ctx, &dto.UpdatePackageReq{
			ID:          detail.ID,
			Name:        &name,
			Description: &description,
			Admins:      &admins,
		})
	}

	parentPath := path.Dir(seedPackage.LogicalPath)
	if parentPath == "." {
		parentPath = ""
	}
	_, err = serviceTreeService.CreatePackage(ctx, &dto.CreatePackageReq{
		User:               SystemUsername,
		App:                "prompt",
		Name:               seedPackage.Name,
		Code:               seedPackage.Code,
		ParentFullCodePath: parentPath,
		Description:        seedPackage.Description,
		Admins:             SystemUsername,
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}
	return nil
}
