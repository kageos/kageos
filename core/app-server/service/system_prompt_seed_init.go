package service

import (
	"context"
	"os"
	"path"
	"strings"

	agentprompt "github.com/ai-agent-os/ai-agent-os/core/agent-server/prompt"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func initSystemPromptSeed(ctx context.Context, serviceTreeService *ServiceTreeService) error {
	if serviceTreeService == nil {
		return nil
	}
	if !shouldSyncSystemPromptSeed(ctx, serviceTreeService) {
		prunedCount := pruneRetiredSystemPromptSeedNodes(ctx, serviceTreeService)
		logger.Infof(ctx, "[SystemWorkspace] APP_ENV 非 dev 且 prompt 已初始化，跳过 system/prompt upsert，清理 %d 个废弃节点", prunedCount)
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
	prunedCount := pruneRetiredSystemPromptSeedNodes(ctx, serviceTreeService)
	logger.Infof(ctx, "[SystemWorkspace] system/prompt 已同步 %d 个目录、%d 篇种子文档，清理 %d 个废弃节点", len(seedPackages), len(seedDocs), prunedCount)
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

func pruneRetiredSystemPromptSeedNodes(ctx context.Context, serviceTreeService *ServiceTreeService) int {
	if serviceTreeService == nil {
		return 0
	}
	pruned := 0
	for _, fullCodePath := range retiredSystemPromptSeedPathsForPruneOnly() {
		fullCodePath = strings.TrimSpace(fullCodePath)
		if fullCodePath == "" {
			continue
		}
		detail, err := serviceTreeService.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{FullCodePath: fullCodePath})
		if err != nil || detail == nil {
			cleanupRetiredSystemPromptRuntimePackage(ctx, serviceTreeService, fullCodePath)
			continue
		}
		if err := serviceTreeService.DeleteServiceTree(ctx, detail.ID); err != nil {
			logger.Warnf(ctx, "[SystemWorkspace] 清理废弃 system/prompt 节点失败: path=%s, id=%d, error=%v", fullCodePath, detail.ID, err)
			cleanupRetiredSystemPromptRuntimePackage(ctx, serviceTreeService, fullCodePath)
			continue
		}
		cleanupRetiredSystemPromptRuntimePackage(ctx, serviceTreeService, fullCodePath)
		pruned++
	}
	return pruned
}

func cleanupRetiredSystemPromptRuntimePackage(ctx context.Context, serviceTreeService *ServiceTreeService, fullCodePath string) {
	packagePath := retiredSystemPromptRuntimePackagePath(fullCodePath)
	if packagePath == "" || serviceTreeService == nil || serviceTreeService.mutationService == nil || serviceTreeService.mutationService.runtimeWorkspace == nil {
		return
	}
	runtimeWorkspace := serviceTreeService.mutationService.runtimeWorkspace
	appModel, err := runtimeWorkspace.getRuntimeBoundAppByUserApp(SystemUsername, "prompt", "清理废弃 system/prompt 目录脚手架")
	if err != nil {
		logger.Warnf(ctx, "[SystemWorkspace] 清理废弃 runtime prompt 目录失败: path=%s, package=%s, error=%v", fullCodePath, packagePath, err)
		return
	}
	_, resp, err := runtimeWorkspace.deleteDirectoryScaffold(ctx, appModel.ID, packagePath)
	if err != nil {
		logger.Warnf(ctx, "[SystemWorkspace] 清理废弃 runtime prompt 目录失败: path=%s, package=%s, error=%v", fullCodePath, packagePath, err)
		return
	}
	if resp == nil || !resp.Success {
		errText := ""
		if resp != nil {
			errText = resp.Error
		}
		logger.Warnf(ctx, "[SystemWorkspace] 清理废弃 runtime prompt 目录失败: path=%s, package=%s, error=%s", fullCodePath, packagePath, errText)
	}
}

func retiredSystemPromptRuntimePackagePath(fullCodePath string) string {
	fullCodePath = strings.TrimRight(strings.TrimSpace(fullCodePath), "/")
	switch fullCodePath {
	case agentprompt.SystemPromptRootPath + "/workspace":
		return "workspace"
	default:
		return ""
	}
}

// retiredSystemPromptSeedPathsForPruneOnly lists historical system prompt nodes that should be
// deleted from the service tree after the local seed has been simplified. It is not an injection
// list and must not be used by agent prompts.
func retiredSystemPromptSeedPathsForPruneOnly() []string {
	return []string{
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/platform-overview"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/platform-function-architecture"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/platform-cross-cutting-capabilities"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/sdk/widget-reference"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/sdk/workbench-tools-sdk-relationship"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/sdk/widget-system"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/sdk/form-submit-basic"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/sdk/table-crud-basic"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/sdk/combo-table-form"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/sdk/combo-table-form-chart"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/sdk/form-table-chart-reference"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/sdk/sdk"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/sdk/build-validation-reference"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/sdk/common-runtime-capabilities"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/sdk/platform-api-reference"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/doc/workbench-all-in-one-system-prompt"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/intents/publish-hub"),
		agentprompt.PromptDocLeafPath(agentprompt.SystemPromptRootPath + "/mode/dev/first_assistant"),
		agentprompt.PromptDocIndexPath(agentprompt.SystemPromptRootPath + "/mode"),
		agentprompt.SystemPromptRootPath + "/mode/agent",
		agentprompt.SystemPromptRootPath + "/mode/execute",
		agentprompt.SystemPromptRootPath + "/mode/modify",
		agentprompt.SystemPromptRootPath + "/mode/qa",
		agentprompt.SystemPromptRootPath + "/workspace",
	}
}
