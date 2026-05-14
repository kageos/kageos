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

func copyServiceTreeImpl(s *serviceTreeCopyService, ctx context.Context, req *dto.CopyDirectoryReq) (*dto.CopyDirectoryResp, error) {
	targetApp, err := s.appRepo.GetAppByID(req.TargetAppID)
	if err != nil {
		return nil, fmt.Errorf("获取目标应用失败: %w", err)
	}

	if strings.HasPrefix(req.SourceDirectoryPath, "hub://") {
		return nil, fmt.Errorf("copy service tree 不再支持 hub:// 链接，请使用本地目录路径或能力包导入")
	}

	return s.copyFromLocal(ctx, req, targetApp)
}

func copyFromLocalImpl(s *serviceTreeCopyService, ctx context.Context, req *dto.CopyDirectoryReq, targetApp *model.App) (*dto.CopyDirectoryResp, error) {
	sourceParts := strings.Split(strings.Trim(req.SourceDirectoryPath, "/"), "/")
	if len(sourceParts) < 3 {
		return nil, fmt.Errorf("源目录路径格式错误: %s", req.SourceDirectoryPath)
	}
	sourceUser := sourceParts[0]
	sourceAppCode := sourceParts[1]

	sourceApp, err := s.appRepo.GetAppByUserName(sourceUser, sourceAppCode)
	if err != nil {
		return nil, fmt.Errorf("获取源应用失败: %w", err)
	}

	sourceRootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源目录信息失败: %w", err)
	}

	sourceDescendants, err := s.serviceTreeRepo.GetDescendantDirectories(sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源子目录失败: %w", err)
	}

	sourceTrees := make(map[string]*model.ServiceTree)
	sourceTrees[sourceRootTree.FullCodePath] = sourceRootTree
	for _, desc := range sourceDescendants {
		sourceTrees[desc.FullCodePath] = desc
	}

	directoryFiles, err := s.getDirectoryFilesFromRuntimeRecursively(ctx, sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("从 runtime 读取目录文件失败: %w", err)
	}

	if len(directoryFiles) == 0 {
		return nil, fmt.Errorf("未找到任何目录，请确认源目录存在")
	}

	targetRootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.TargetDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取目标目录信息失败: %w", err)
	}

	targetRootPath := targetRootTree.FullCodePath

	type dirInfo struct {
		sourcePath string
		targetPath string
		sourceTree *model.ServiceTree
	}
	dirsToCreate := make([]dirInfo, 0, len(sourceTrees))

	for sourcePath, sourceTree := range sourceTrees {
		relativePath := strings.TrimPrefix(sourcePath, req.SourceDirectoryPath)
		relativePath = strings.TrimPrefix(relativePath, "/")

		var targetPath string
		if relativePath == "" {
			sourcePathParts := strings.Split(strings.Trim(req.SourceDirectoryPath, "/"), "/")
			if len(sourcePathParts) < 3 {
				return nil, fmt.Errorf("源目录路径格式错误: %s", req.SourceDirectoryPath)
			}
			dirName := sourcePathParts[len(sourcePathParts)-1]
			targetPath = targetRootPath + "/" + dirName
		} else {
			sourcePathParts := strings.Split(strings.Trim(req.SourceDirectoryPath, "/"), "/")
			if len(sourcePathParts) < 3 {
				return nil, fmt.Errorf("源目录路径格式错误: %s", req.SourceDirectoryPath)
			}
			dirName := sourcePathParts[len(sourcePathParts)-1]
			targetPath = targetRootPath + "/" + dirName + "/" + relativePath
		}

		dirsToCreate = append(dirsToCreate, dirInfo{
			sourcePath: sourcePath,
			targetPath: targetPath,
			sourceTree: sourceTree,
		})
	}

	sort.Slice(dirsToCreate, func(i, j int) bool {
		return len(dirsToCreate[i].targetPath) < len(dirsToCreate[j].targetPath)
	})

	if len(dirsToCreate) > 0 {
		targetRootPath = dirsToCreate[0].targetPath
	}

	directoryItems := make([]*dto.DirectoryScaffoldItem, 0)
	fileItems := make([]*dto.FileWriteItem, 0)

	for _, dirInfo := range dirsToCreate {
		directoryItems = append(directoryItems, &dto.DirectoryScaffoldItem{
			FullCodePath: dirInfo.targetPath,
			Name:         dirInfo.sourceTree.Name,
			Description:  dirInfo.sourceTree.Description,
			Tags:         dirInfo.sourceTree.Tags,
		})
	}

	totalFileCount := 0
	logger.Infof(ctx, "[CopyServiceTree] 开始处理文件快照: 源目录=%s, 目标根路径=%s, 目录数=%d",
		req.SourceDirectoryPath, targetRootPath, len(directoryFiles))

	for sourcePath, fileSnapshots := range directoryFiles {
		logger.Infof(ctx, "[CopyServiceTree] 处理目录文件快照: sourcePath=%s, fileCount=%d",
			sourcePath, len(fileSnapshots))

		relativePath := strings.TrimPrefix(sourcePath, req.SourceDirectoryPath)
		relativePath = strings.TrimPrefix(relativePath, "/")

		var targetPath string
		if relativePath == "" {
			targetPath = targetRootPath
		} else {
			targetPath = targetRootPath + "/" + relativePath
		}

		logger.Infof(ctx, "[CopyServiceTree] 目录路径映射: sourcePath=%s -> targetPath=%s, relativePath=%s",
			sourcePath, targetPath, relativePath)

		for _, fileSnapshot := range fileSnapshots {
			fileName := fileSnapshot.FileName
			if fileName == "" {
				fileName = strings.TrimSuffix(fileSnapshot.RelativePath, ".go")
				if lastSlash := strings.LastIndex(fileName, "/"); lastSlash >= 0 {
					fileName = fileName[lastSlash+1:]
				}
			}

			if fileName == "init_" || fileSnapshot.RelativePath == "init_.go" || strings.HasSuffix(fileSnapshot.RelativePath, "/init_.go") {
				logger.Infof(ctx, "[ServiceTreeService] 跳过 init_.go 文件: %s", fileSnapshot.RelativePath)
				continue
			}

			fileItems = append(fileItems, &dto.FileWriteItem{
				FullCodePath: targetPath,
				FileName:     fileName,
				FileType:     fileSnapshot.FileType,
				Content:      fileSnapshot.Content,
				RelativePath: fileSnapshot.RelativePath,
			})
			totalFileCount++
		}

		logger.Infof(ctx, "[ServiceTreeService] 准备复制目录: source=%s, target=%s, fileCount=%d",
			sourcePath, targetPath, len(fileSnapshots))
	}

	var directoryCount int
	if len(directoryItems) > 0 {
		batchCreateReq := &dto.BatchCreateDirectoryTreeReq{
			User:  targetApp.User,
			App:   targetApp.Code,
			Items: directoryItems,
		}

		batchCreateResp, err := s.batchCreateDirectoryTree(ctx, batchCreateReq)
		if err != nil {
			logger.Errorf(ctx, "[ServiceTreeService] 批量创建目录失败: error=%v", err)
			return nil, fmt.Errorf("批量创建目录失败: %w", err)
		}

		directoryCount = batchCreateResp.DirectoryCount
		logger.Infof(ctx, "[ServiceTreeService] 批量创建目录完成: directoryCount=%d", directoryCount)
	}

	var fileCount int
	var oldVersion, newVersion, gitCommitHash string
	if len(fileItems) > 0 {
		batchWriteReq := &dto.BatchWriteFilesReq{
			User:  targetApp.User,
			App:   targetApp.Code,
			Files: fileItems,
		}

		batchWriteResp, err := s.batchWriteFiles(ctx, batchWriteReq)
		if err != nil {
			logger.Errorf(ctx, "[ServiceTreeService] 批量写文件失败: error=%v", err)
			return nil, fmt.Errorf("批量写文件失败: %w", err)
		}

		fileCount = batchWriteResp.FileCount
		oldVersion = batchWriteResp.OldVersion
		newVersion = batchWriteResp.NewVersion
		gitCommitHash = batchWriteResp.GitCommitHash
		logger.Infof(ctx, "[ServiceTreeService] 批量写文件完成: fileCount=%d, oldVersion=%s, newVersion=%s, gitCommitHash=%s",
			fileCount, oldVersion, newVersion, gitCommitHash)
	}

	logger.Infof(ctx, "[ServiceTreeService] 复制目录完成: 目录数=%d, 文件数=%d, oldVersion=%s, newVersion=%s",
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
