package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"gorm.io/gorm"
)

type copyDirectorySource struct {
	treesByPath    map[string]*model.ServiceTree
	directoryFiles map[string][]*model.FileSnapshot
	runtimeBundle  *dto.CapabilityBundle
}

type copyDirectoryPlan struct {
	directoryItems       []*dto.DirectoryScaffoldItem
	fileItems            []*dto.FileWriteItem
	docItems             []*capabilityBundleDocInstallItem
	agentTasks           []*dto.CapabilityBundleAgentTask
	targetRootPath       string
	runtimeAssetBasePath string
}

type copyDirectoryTarget struct {
	sourcePath string
	targetPath string
	sourceTree *model.ServiceTree
}

func copyServiceTreeImpl(s *serviceTreeCopyService, ctx context.Context, req *dto.CopyDirectoryReq) (*dto.CopyDirectoryResp, error) {
	targetApp, err := s.appRepo.GetAppByID(req.TargetAppID)
	if err != nil {
		return nil, fmt.Errorf("获取目标应用失败: %w", err)
	}

	return s.copyFromLocal(ctx, req, targetApp)
}

func copyFromLocalImpl(s *serviceTreeCopyService, ctx context.Context, req *dto.CopyDirectoryReq, targetApp *model.App) (*dto.CopyDirectoryResp, error) {
	source, err := loadCopyDirectorySource(ctx, s, req.SourceDirectoryPath)
	if err != nil {
		return nil, err
	}

	targetRootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.TargetDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取目标目录信息失败: %w", err)
	}

	plan, err := buildCopyDirectoryPlan(req.SourceDirectoryPath, targetRootTree.FullCodePath, req.TargetDirectoryName, source)
	if err != nil {
		return nil, err
	}
	if err := validateCopyDirectoryPlacement(req.SourceDirectoryPath, targetRootTree.FullCodePath, plan.targetRootPath); err != nil {
		return nil, err
	}

	targetExists, err := copyTargetDirectoryExists(s, targetApp.ID, plan.targetRootPath)
	if err != nil {
		return nil, err
	}
	if targetExists {
		if !req.ReplaceExisting {
			return nil, fmt.Errorf("目标目录下已存在同名目录 %s；如需更新它，请使用覆盖同名目录", plan.targetRootPath)
		}
		return replaceCopyDirectory(ctx, s, targetApp, plan)
	}

	logger.Infof(ctx, "[CopyServiceTree] 开始复制目录: source=%s, targetRoot=%s, directoryCount=%d, fileDirectoryCount=%d",
		req.SourceDirectoryPath, plan.targetRootPath, len(plan.directoryItems), len(source.directoryFiles))

	directoryCount, err := createCopyDirectories(ctx, s, targetApp, plan.directoryItems)
	if err != nil {
		return nil, err
	}

	fileCount, oldVersion, newVersion, gitCommitHash, err := writeCopyFiles(ctx, s, targetApp, plan.fileItems)
	if err != nil {
		return nil, err
	}
	docCount, agentTaskCount, err := syncCopiedRuntimeAssets(ctx, s, targetApp, plan, false)
	if err != nil {
		return nil, err
	}

	logger.Infof(ctx, "[CopyServiceTree] 复制目录完成: 目录数=%d, 文件数=%d, 文档数=%d, Agent任务数=%d, oldVersion=%s, newVersion=%s",
		directoryCount, fileCount, docCount, agentTaskCount, oldVersion, newVersion)

	return &dto.CopyDirectoryResp{
		Message:             fmt.Sprintf("复制目录成功，共复制 %d 个目录，%d 个文件，%d 份文档，%d 个 Agent 任务", directoryCount, fileCount, docCount, agentTaskCount),
		DirectoryCount:      directoryCount,
		FileCount:           fileCount,
		DocCount:            docCount,
		AgentTaskCount:      agentTaskCount,
		TargetDirectoryPath: plan.targetRootPath,
		OldVersion:          oldVersion,
		NewVersion:          newVersion,
		GitCommitHash:       gitCommitHash,
	}, nil
}

func validateCopyDirectoryPlacement(sourceRootPath, targetParentPath, targetRootPath string) error {
	sourceRootPath = strings.TrimRight(sourceRootPath, "/")
	targetParentPath = strings.TrimRight(targetParentPath, "/")
	targetRootPath = strings.TrimRight(targetRootPath, "/")
	if sourceRootPath == "" || targetParentPath == "" || targetRootPath == "" {
		return fmt.Errorf("复制路径不能为空")
	}
	if sourceRootPath == targetRootPath {
		return fmt.Errorf("不能把目录粘贴回自己；如需备份，请选择其他父目录")
	}
	if strings.HasPrefix(targetParentPath, sourceRootPath+"/") {
		return fmt.Errorf("不能粘贴到自己的子目录")
	}
	if strings.HasPrefix(sourceRootPath, targetRootPath+"/") {
		return fmt.Errorf("不能用子目录覆盖父目录，请先把副本放到兄弟目录或其他目录")
	}
	return nil
}

func copyTargetDirectoryExists(s *serviceTreeCopyService, appID int64, targetRootPath string) (bool, error) {
	existingTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(targetRootPath)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("查询目标同名目录失败: %w", err)
	}
	if existingTree.AppID != appID || existingTree.Type != model.ServiceTreeTypePackage {
		return false, fmt.Errorf("目标路径已存在但类型或应用不匹配: %s", targetRootPath)
	}
	return true, nil
}

func replaceCopyDirectory(
	ctx context.Context,
	s *serviceTreeCopyService,
	targetApp *model.App,
	plan *copyDirectoryPlan,
) (*dto.CopyDirectoryResp, error) {
	logger.Infof(ctx, "[CopyServiceTree] 开始覆盖同名目录: targetRoot=%s, directoryCount=%d, fileCount=%d",
		plan.targetRootPath, len(plan.directoryItems), len(plan.fileItems))

	_, replaceResp, err := s.replaceDirectoryTree(ctx, &dto.ReplaceDirectoryTreeReq{
		User:                   targetApp.User,
		App:                    targetApp.Code,
		TargetRootFullCodePath: plan.targetRootPath,
		Items:                  plan.directoryItems,
		Files:                  plan.fileItems,
		ForceDiff:              true,
		OperationName:          "CopyDirectoryReplace",
		OperationLabel:         "覆盖复制目录",
	})
	if err != nil {
		return nil, err
	}

	if err := syncCopiedDirectoryMetadata(ctx, s, targetApp, plan.directoryItems); err != nil {
		return nil, err
	}
	docCount, agentTaskCount, err := syncCopiedRuntimeAssets(ctx, s, targetApp, plan, true)
	if err != nil {
		return nil, err
	}
	if err := cleanupReplacedDirectoryMetadata(ctx, s, targetApp, plan.targetRootPath, plan.directoryItems); err != nil {
		return nil, err
	}

	directoryCount := len(plan.directoryItems)
	fileCount := len(plan.fileItems)
	var oldVersion, newVersion, gitCommitHash string
	if replaceResp != nil {
		directoryCount = replaceResp.DirectoryCount
		fileCount = replaceResp.FileCount
		oldVersion = replaceResp.OldVersion
		newVersion = replaceResp.NewVersion
		gitCommitHash = replaceResp.GitCommitHash
	}

	return &dto.CopyDirectoryResp{
		Message:             fmt.Sprintf("覆盖目录成功，目标 %s 已替换为复制内容，共恢复 %d 个目录，写入 %d 个文件，复制 %d 份文档，%d 个 Agent 任务", plan.targetRootPath, directoryCount, fileCount, docCount, agentTaskCount),
		DirectoryCount:      directoryCount,
		FileCount:           fileCount,
		DocCount:            docCount,
		AgentTaskCount:      agentTaskCount,
		Replaced:            true,
		TargetDirectoryPath: plan.targetRootPath,
		OldVersion:          oldVersion,
		NewVersion:          newVersion,
		GitCommitHash:       gitCommitHash,
	}, nil
}

func loadCopyDirectorySource(ctx context.Context, s *serviceTreeCopyService, sourceDirectoryPath string) (*copyDirectorySource, error) {
	sourceParts := strings.Split(strings.Trim(sourceDirectoryPath, "/"), "/")
	if len(sourceParts) < 3 {
		return nil, fmt.Errorf("源目录路径格式错误: %s", sourceDirectoryPath)
	}
	sourceUser := sourceParts[0]
	sourceAppCode := sourceParts[1]

	sourceApp, err := s.appRepo.GetAppByUserName(sourceUser, sourceAppCode)
	if err != nil {
		return nil, fmt.Errorf("获取源应用失败: %w", err)
	}

	sourceRootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(sourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源目录信息失败: %w", err)
	}

	sourceDescendants, err := s.serviceTreeRepo.GetDescendantDirectories(sourceApp.ID, sourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源子目录失败: %w", err)
	}

	sourceTrees := make(map[string]*model.ServiceTree)
	sourceTrees[sourceRootTree.FullCodePath] = sourceRootTree
	for _, desc := range sourceDescendants {
		sourceTrees[desc.FullCodePath] = desc
	}

	directoryFiles, err := s.getDirectoryFilesFromRuntimeRecursively(ctx, sourceApp.ID, sourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("从 runtime 读取目录文件失败: %w", err)
	}

	if len(directoryFiles) == 0 {
		return nil, fmt.Errorf("未找到任何目录，请确认源目录存在")
	}
	if s.capabilityBundle == nil {
		return nil, fmt.Errorf("目录能力包服务未初始化，无法复制文档和 Agent 任务")
	}
	runtimeBundle, err := s.capabilityBundle.ExportCapabilityBundle(ctx, &dto.ExportCapabilityBundleReq{
		SourceDirectoryPath: sourceDirectoryPath,
	})
	if err != nil {
		return nil, fmt.Errorf("读取源目录文档和 Agent 任务失败: %w", err)
	}

	return &copyDirectorySource{
		treesByPath:    sourceTrees,
		directoryFiles: directoryFiles,
		runtimeBundle:  runtimeBundle,
	}, nil
}

func buildCopyDirectoryPlan(sourceRootPath, targetParentPath, targetRootName string, source *copyDirectorySource) (*copyDirectoryPlan, error) {
	targets, err := planCopyDirectoryTargets(sourceRootPath, targetParentPath, source.treesByPath)
	if err != nil {
		return nil, err
	}

	targetRootPath := targetParentPath
	if len(targets) > 0 {
		targetRootPath = targets[0].targetPath
	}

	targetRootName = strings.TrimSpace(targetRootName)
	directoryItems := make([]*dto.DirectoryScaffoldItem, 0, len(targets))
	for _, target := range targets {
		name := target.sourceTree.Name
		if target.sourcePath == sourceRootPath && targetRootName != "" {
			name = targetRootName
		}
		directoryItems = append(directoryItems, &dto.DirectoryScaffoldItem{
			FullCodePath: target.targetPath,
			Name:         name,
			Description:  target.sourceTree.Description,
			Tags:         target.sourceTree.Tags,
		})
	}

	fileItems := buildCopyFileItems(sourceRootPath, targetRootPath, source.directoryFiles)
	var docItems []*capabilityBundleDocInstallItem
	var agentTasks []*dto.CapabilityBundleAgentTask
	if source.runtimeBundle != nil {
		runtimePlan, err := buildCapabilityBundleInstallPlan(targetParentPath, source.runtimeBundle)
		if err != nil {
			return nil, fmt.Errorf("生成文档和 Agent 任务复制计划失败: %w", err)
		}
		docItems = runtimePlan.docItems
		agentTasks = source.runtimeBundle.AgentTasks
	}

	return &copyDirectoryPlan{
		directoryItems:       directoryItems,
		fileItems:            fileItems,
		docItems:             docItems,
		agentTasks:           agentTasks,
		targetRootPath:       targetRootPath,
		runtimeAssetBasePath: targetParentPath,
	}, nil
}

func planCopyDirectoryTargets(sourceRootPath, targetParentPath string, sourceTrees map[string]*model.ServiceTree) ([]copyDirectoryTarget, error) {
	sourceDirName, err := lastPathSegment(sourceRootPath)
	if err != nil {
		return nil, fmt.Errorf("源目录路径格式错误: %s", sourceRootPath)
	}

	targets := make([]copyDirectoryTarget, 0, len(sourceTrees))
	for sourcePath, sourceTree := range sourceTrees {
		relativePath := strings.TrimPrefix(sourcePath, sourceRootPath)
		relativePath = strings.TrimPrefix(relativePath, "/")

		targetPath := targetParentPath + "/" + sourceDirName
		if relativePath != "" {
			targetPath += "/" + relativePath
		}

		targets = append(targets, copyDirectoryTarget{
			sourcePath: sourcePath,
			targetPath: targetPath,
			sourceTree: sourceTree,
		})
	}

	sort.Slice(targets, func(i, j int) bool {
		return len(targets[i].targetPath) < len(targets[j].targetPath)
	})

	return targets, nil
}

func buildCopyFileItems(sourceRootPath, targetRootPath string, directoryFiles map[string][]*model.FileSnapshot) []*dto.FileWriteItem {
	fileItems := make([]*dto.FileWriteItem, 0)
	for sourcePath, fileSnapshots := range directoryFiles {
		relativePath := strings.TrimPrefix(sourcePath, sourceRootPath)
		relativePath = strings.TrimPrefix(relativePath, "/")

		var targetPath string
		if relativePath == "" {
			targetPath = targetRootPath
		} else {
			targetPath = targetRootPath + "/" + relativePath
		}

		for _, fileSnapshot := range fileSnapshots {
			fileName := fileNameForCopy(fileSnapshot)
			if shouldSkipCopyFile(fileSnapshot, fileName) {
				continue
			}

			fileItems = append(fileItems, &dto.FileWriteItem{
				FullCodePath: targetPath,
				FileName:     fileName,
				FileType:     fileSnapshot.FileType,
				Content:      fileSnapshot.Content,
				RelativePath: fileSnapshot.RelativePath,
			})
		}
	}
	return fileItems
}

func fileNameForCopy(fileSnapshot *model.FileSnapshot) string {
	if fileSnapshot.FileName != "" {
		return fileSnapshot.FileName
	}

	fileName := strings.TrimSuffix(fileSnapshot.RelativePath, ".go")
	if lastSlash := strings.LastIndex(fileName, "/"); lastSlash >= 0 {
		fileName = fileName[lastSlash+1:]
	}
	return fileName
}

func shouldSkipCopyFile(fileSnapshot *model.FileSnapshot, fileName string) bool {
	return fileName == "init_" || fileSnapshot.RelativePath == "init_.go" || strings.HasSuffix(fileSnapshot.RelativePath, "/init_.go")
}

func createCopyDirectories(ctx context.Context, s *serviceTreeCopyService, targetApp *model.App, directoryItems []*dto.DirectoryScaffoldItem) (int, error) {
	if len(directoryItems) == 0 {
		return 0, nil
	}

	batchCreateReq := &dto.BatchCreateDirectoryTreeReq{
		User:  targetApp.User,
		App:   targetApp.Code,
		Items: directoryItems,
	}

	batchCreateResp, err := s.batchCreateDirectoryTree(ctx, batchCreateReq)
	if err != nil {
		logger.Errorf(ctx, "[CopyServiceTree] 批量创建目录失败: error=%v", err)
		return 0, fmt.Errorf("批量创建目录失败: %w", err)
	}

	logger.Infof(ctx, "[CopyServiceTree] 批量创建目录完成: directoryCount=%d", batchCreateResp.DirectoryCount)
	return batchCreateResp.DirectoryCount, nil
}

func writeCopyFiles(ctx context.Context, s *serviceTreeCopyService, targetApp *model.App, fileItems []*dto.FileWriteItem) (int, string, string, string, error) {
	if len(fileItems) == 0 {
		return 0, "", "", "", nil
	}

	batchWriteReq := &dto.BatchWriteFilesReq{
		User:  targetApp.User,
		App:   targetApp.Code,
		Files: fileItems,
	}

	batchWriteResp, err := s.batchWriteFiles(ctx, batchWriteReq)
	if err != nil {
		logger.Errorf(ctx, "[CopyServiceTree] 批量写文件失败: error=%v", err)
		return 0, "", "", "", fmt.Errorf("批量写文件失败: %w", err)
	}

	logger.Infof(ctx, "[CopyServiceTree] 批量写文件完成: fileCount=%d, oldVersion=%s, newVersion=%s, gitCommitHash=%s",
		batchWriteResp.FileCount, batchWriteResp.OldVersion, batchWriteResp.NewVersion, batchWriteResp.GitCommitHash)
	return batchWriteResp.FileCount, batchWriteResp.OldVersion, batchWriteResp.NewVersion, batchWriteResp.GitCommitHash, nil
}

func syncCopiedRuntimeAssets(
	ctx context.Context,
	s *serviceTreeCopyService,
	targetApp *model.App,
	plan *copyDirectoryPlan,
	replaceExisting bool,
) (int, int, error) {
	if s.capabilityBundle == nil {
		return 0, 0, fmt.Errorf("目录能力包服务未初始化，无法复制文档和 Agent 任务")
	}
	if replaceExisting {
		if err := cleanupReplacedDocs(ctx, s, targetApp, plan); err != nil {
			return 0, 0, err
		}
		if err := cleanupReplacedAgentTasks(ctx, plan); err != nil {
			return 0, 0, err
		}
	}

	if len(plan.docItems) > 0 {
		if _, err := s.capabilityBundle.installCapabilityBundleDocs(ctx, targetApp, plan.docItems, replaceExisting); err != nil {
			return 0, 0, fmt.Errorf("复制目录文档失败: %w", err)
		}
	}
	agentTaskRefs, err := s.capabilityBundle.installCapabilityBundleAgentTasks(ctx, plan.runtimeAssetBasePath, plan.agentTasks, replaceExisting)
	if err != nil {
		return len(plan.docItems), 0, fmt.Errorf("复制 Agent 任务失败: %w", err)
	}
	return len(plan.docItems), len(agentTaskRefs), nil
}

func cleanupReplacedDocs(
	ctx context.Context,
	s *serviceTreeCopyService,
	targetApp *model.App,
	plan *copyDirectoryPlan,
) error {
	if s.capabilityBundle.docService == nil {
		return fmt.Errorf("文档服务未初始化")
	}
	intended := make(map[string]struct{}, len(plan.docItems))
	for _, item := range plan.docItems {
		if item != nil {
			intended[strings.TrimRight(item.FullCodePath, "/")] = struct{}{}
		}
	}
	nodes, err := s.serviceTreeRepo.GetDescendantNodes(targetApp.ID, plan.targetRootPath)
	if err != nil {
		return fmt.Errorf("查询待清理旧文档失败: %w", err)
	}
	for _, node := range nodes {
		if node == nil || node.Type != model.ServiceTreeTypeDocs {
			continue
		}
		if _, keep := intended[strings.TrimRight(node.FullCodePath, "/")]; keep {
			continue
		}
		if err := s.capabilityBundle.docService.DeleteDoc(ctx, node.FullCodePath); err != nil {
			return fmt.Errorf("清理旧文档失败: path=%s: %w", node.FullCodePath, err)
		}
	}
	return nil
}

func cleanupReplacedAgentTasks(ctx context.Context, plan *copyDirectoryPlan) error {
	client := newAppScheduleClient()
	managedCtx := contextx.WithClientSource(ctx, scheduledTaskSourceBundle)
	deleteClient, ok := client.(interface {
		DeleteTask(context.Context, int64) error
	})
	if !ok {
		return fmt.Errorf("定时任务客户端不支持删除，无法完全替换 Agent 任务")
	}

	intended := make(map[string]struct{}, len(plan.agentTasks))
	for _, task := range plan.agentTasks {
		if task == nil {
			continue
		}
		targetFullCodePath := joinCapabilityFullCodePath(plan.runtimeAssetBasePath, task.RelativePath)
		intended[capabilityAgentTaskKey(targetFullCodePath, task.Code)] = struct{}{}
	}

	taskIDs := make([]int64, 0)
	for page := 1; ; page++ {
		resp, err := client.ListTasks(managedCtx, scheduledsdk.ListTasksRequest{
			ExecutorKey:       ScheduledAgentSessionExecutorKey,
			Category:          "scheduled_agent_session",
			ResourceScope:     "workspace_directory",
			ResourceKeyPrefix: plan.targetRootPath,
			Page:              page,
			PageSize:          capabilityBundleAgentTaskPageSize,
		})
		if err != nil {
			return fmt.Errorf("查询待清理旧 Agent 任务失败: %w", err)
		}
		if resp == nil || len(resp.List) == 0 {
			break
		}
		for _, task := range resp.List {
			if task == nil {
				continue
			}
			if strings.TrimSpace(task.Metadata["managed_by"]) != scheduledTaskSourceBundle {
				continue
			}
			key := capabilityAgentTaskKey(scheduledAgentTaskResourcePath(task), capabilityBundleAgentTaskCode(task))
			if _, keep := intended[key]; !keep {
				taskIDs = append(taskIDs, task.ID)
			}
		}
		if len(resp.List) < capabilityBundleAgentTaskPageSize {
			break
		}
	}
	for _, taskID := range taskIDs {
		if err := deleteClient.DeleteTask(managedCtx, taskID); err != nil {
			return fmt.Errorf("清理旧 Agent 任务失败: task_id=%d: %w", taskID, err)
		}
	}
	return nil
}

func syncCopiedDirectoryMetadata(
	ctx context.Context,
	s *serviceTreeCopyService,
	targetApp *model.App,
	directoryItems []*dto.DirectoryScaffoldItem,
) error {
	if len(directoryItems) == 0 {
		return nil
	}
	sortedItems := make([]*dto.DirectoryScaffoldItem, 0, len(directoryItems))
	for _, item := range directoryItems {
		if item != nil {
			sortedItems = append(sortedItems, item)
		}
	}
	sort.Slice(sortedItems, func(i, j int) bool {
		return len(sortedItems[i].FullCodePath) < len(sortedItems[j].FullCodePath)
	})

	currentVersionNum := extractVersionNumForServiceTree(targetApp.Version)
	for _, item := range sortedItems {
		pathParts := strings.Split(strings.Trim(item.FullCodePath, "/"), "/")
		if len(pathParts) < 3 {
			continue
		}
		dirCode := pathParts[len(pathParts)-1]
		existingTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(item.FullCodePath)
		if err == nil && existingTree != nil {
			if existingTree.AppID != targetApp.ID || existingTree.Type != model.ServiceTreeTypePackage {
				return fmt.Errorf("目标目录路径已存在但类型或应用不匹配: %s", item.FullCodePath)
			}
			existingTree.Name = item.Name
			existingTree.Description = item.Description
			existingTree.Tags = item.Tags
			existingTree.UpdateVersionNum = currentVersionNum
			if err := s.serviceTreeRepo.UpdateServiceTree(existingTree); err != nil {
				return fmt.Errorf("更新目录元数据失败: path=%s: %w", item.FullCodePath, err)
			}
			continue
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("查询目标目录失败: path=%s: %w", item.FullCodePath, err)
		}

		newTree := &model.ServiceTree{
			Name:             item.Name,
			Code:             dirCode,
			Type:             model.ServiceTreeTypePackage,
			Description:      item.Description,
			Tags:             item.Tags,
			AppID:            targetApp.ID,
			FullCodePath:     item.FullCodePath,
			AddVersionNum:    currentVersionNum,
			UpdateVersionNum: 0,
		}
		if err := s.serviceTreeRepo.CreateServiceTreeWithParentPath(newTree, ""); err != nil {
			return fmt.Errorf("创建目录元数据失败: path=%s: %w", item.FullCodePath, err)
		}
		logger.Infof(ctx, "[CopyServiceTree] 已同步目录元数据: %s", item.FullCodePath)
	}
	return nil
}

func cleanupReplacedDirectoryMetadata(
	ctx context.Context,
	s *serviceTreeCopyService,
	targetApp *model.App,
	targetRootPath string,
	directoryItems []*dto.DirectoryScaffoldItem,
) error {
	intended := make(map[string]struct{}, len(directoryItems))
	for _, item := range directoryItems {
		if item != nil {
			intended[strings.TrimRight(item.FullCodePath, "/")] = struct{}{}
		}
	}
	if _, ok := intended[strings.TrimRight(targetRootPath, "/")]; !ok {
		intended[strings.TrimRight(targetRootPath, "/")] = struct{}{}
	}

	descendants, err := s.serviceTreeRepo.GetDescendantDirectories(targetApp.ID, targetRootPath)
	if err != nil {
		return fmt.Errorf("查询待清理旧目录失败: %w", err)
	}
	sort.Slice(descendants, func(i, j int) bool {
		return len(descendants[i].FullCodePath) > len(descendants[j].FullCodePath)
	})
	for _, desc := range descendants {
		if desc == nil {
			continue
		}
		if _, keep := intended[strings.TrimRight(desc.FullCodePath, "/")]; keep {
			continue
		}
		if err := s.serviceTreeRepo.DeleteServiceTree(desc.ID); err != nil {
			return fmt.Errorf("清理旧目录元数据失败: path=%s: %w", desc.FullCodePath, err)
		}
		logger.Infof(ctx, "[CopyServiceTree] 已清理替换后不存在的旧目录元数据: %s", desc.FullCodePath)
	}
	return nil
}

func lastPathSegment(path string) (string, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid service tree path")
	}
	return parts[len(parts)-1], nil
}
