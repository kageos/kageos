package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type copyDirectorySource struct {
	treesByPath    map[string]*model.ServiceTree
	directoryFiles map[string][]*model.FileSnapshot
}

type copyDirectoryPlan struct {
	directoryItems []*dto.DirectoryScaffoldItem
	fileItems      []*dto.FileWriteItem
	targetRootPath string
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

	plan, err := buildCopyDirectoryPlan(req.SourceDirectoryPath, targetRootTree.FullCodePath, source)
	if err != nil {
		return nil, err
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

	logger.Infof(ctx, "[CopyServiceTree] 复制目录完成: 目录数=%d, 文件数=%d, oldVersion=%s, newVersion=%s",
		directoryCount, fileCount, oldVersion, newVersion)

	return &dto.CopyDirectoryResp{
		Message:        fmt.Sprintf("复制目录成功，共复制 %d 个目录，%d 个文件", directoryCount, fileCount),
		DirectoryCount: directoryCount,
		FileCount:      fileCount,
		OldVersion:     oldVersion,
		NewVersion:     newVersion,
		GitCommitHash:  gitCommitHash,
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

	return &copyDirectorySource{
		treesByPath:    sourceTrees,
		directoryFiles: directoryFiles,
	}, nil
}

func buildCopyDirectoryPlan(sourceRootPath, targetParentPath string, source *copyDirectorySource) (*copyDirectoryPlan, error) {
	targets, err := planCopyDirectoryTargets(sourceRootPath, targetParentPath, source.treesByPath)
	if err != nil {
		return nil, err
	}

	targetRootPath := targetParentPath
	if len(targets) > 0 {
		targetRootPath = targets[0].targetPath
	}

	directoryItems := make([]*dto.DirectoryScaffoldItem, 0, len(targets))
	for _, target := range targets {
		directoryItems = append(directoryItems, &dto.DirectoryScaffoldItem{
			FullCodePath: target.targetPath,
			Name:         target.sourceTree.Name,
			Description:  target.sourceTree.Description,
			Tags:         target.sourceTree.Tags,
		})
	}

	fileItems := buildCopyFileItems(sourceRootPath, targetRootPath, source.directoryFiles)

	return &copyDirectoryPlan{
		directoryItems: directoryItems,
		fileItems:      fileItems,
		targetRootPath: targetRootPath,
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

func lastPathSegment(path string) (string, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid service tree path")
	}
	return parts[len(parts)-1], nil
}
