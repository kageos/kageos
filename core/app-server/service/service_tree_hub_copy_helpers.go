package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

func copyServiceTreeImpl(s *serviceTreeHubService, ctx context.Context, req *dto.CopyDirectoryReq) (*dto.CopyDirectoryResp, error) {
	targetApp, err := s.appRepo.GetAppByID(req.TargetAppID)
	if err != nil {
		return nil, fmt.Errorf("获取目标应用失败: %w", err)
	}

	if strings.HasPrefix(req.SourceDirectoryPath, "hub://") {
		return s.copyFromHub(ctx, req, targetApp)
	}

	return s.copyFromLocal(ctx, req, targetApp)
}

func copyFromLocalImpl(s *serviceTreeHubService, ctx context.Context, req *dto.CopyDirectoryReq, targetApp *model.App) (*dto.CopyDirectoryResp, error) {
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

func copyFromHubImpl(s *serviceTreeHubService, ctx context.Context, req *dto.CopyDirectoryReq, targetApp *model.App) (*dto.CopyDirectoryResp, error) {
	hubLinkInfo, err := ParseHubLink(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("解析 Hub 链接失败: %w", err)
	}

	logger.Infof(ctx, "[CopyServiceTree] 解析 Hub 链接成功: host=%s, fullCodePath=%s, version=%s",
		hubLinkInfo.Host, hubLinkInfo.FullCodePath, hubLinkInfo.Version)

	hubDetail, err := apicall.GetHubDirectoryDetailFromHost(ctx, &dto.GetHubDirectoryDetailFromHostReq{
		Host:         hubLinkInfo.Host,
		FullCodePath: hubLinkInfo.FullCodePath,
		Version:      hubLinkInfo.Version,
		IncludeTree:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("获取 Hub 目录详情失败: %w", err)
	}

	if hubDetail.DirectoryTree != nil {
		logger.Infof(ctx, "[CopyServiceTree] Hub 目录树根节点信息: Name=%s, Code=%s, Path=%s, Files数量=%d, Subdirectories数量=%d",
			hubDetail.DirectoryTree.Name, hubDetail.DirectoryTree.Code, hubDetail.DirectoryTree.Path,
			len(hubDetail.DirectoryTree.Files), len(hubDetail.DirectoryTree.Subdirectories))
		s.logDirectoryTree(ctx, hubDetail.DirectoryTree, 0)
	} else {
		logger.Warnf(ctx, "[CopyServiceTree] Hub 目录树为空")
	}

	if hubLinkInfo.Version != "" && hubDetail.Version != hubLinkInfo.Version {
		return nil, fmt.Errorf("版本不匹配：请求版本 %s，实际版本 %s", hubLinkInfo.Version, hubDetail.Version)
	}

	targetPath := req.TargetDirectoryPath
	if hubDetail.DirectoryTree == nil {
		return nil, fmt.Errorf("Hub 目录树为空")
	}
	if err := validateHubDirectoryTreeForInstallImpl(hubDetail.DirectoryTree); err != nil {
		return nil, fmt.Errorf("Hub 目录树校验失败: %w", err)
	}

	directoryItems := make([]*dto.DirectoryScaffoldItem, 0)
	fileItems := make([]*dto.FileWriteItem, 0)
	s.buildItemsFromTree(hubDetail.DirectoryTree, targetPath, &directoryItems, &fileItems)

	var directoryCount int
	if len(directoryItems) > 0 {
		batchCreateReq := &dto.BatchCreateDirectoryTreeReq{
			User:  targetApp.User,
			App:   targetApp.Code,
			Items: directoryItems,
		}

		batchCreateResp, err := s.batchCreateDirectoryTree(ctx, batchCreateReq)
		if err != nil {
			return nil, fmt.Errorf("批量创建目录失败: %w", err)
		}

		directoryCount = batchCreateResp.DirectoryCount
		logger.Infof(ctx, "[CopyServiceTree] 批量创建目录完成: directoryCount=%d", directoryCount)
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
			return nil, fmt.Errorf("批量写文件失败: %w", err)
		}

		fileCount = batchWriteResp.FileCount
		oldVersion = batchWriteResp.OldVersion
		newVersion = batchWriteResp.NewVersion
		gitCommitHash = batchWriteResp.GitCommitHash
		logger.Infof(ctx, "[CopyServiceTree] 批量写文件完成: fileCount=%d, oldVersion=%s, newVersion=%s, gitCommitHash=%s",
			fileCount, oldVersion, newVersion, gitCommitHash)
	}

	rootDirPath := targetPath
	if hubDetail.DirectoryTree != nil && hubDetail.DirectoryTree.Code != "" {
		rootDirPath = fmt.Sprintf("%s/%s", targetPath, hubDetail.DirectoryTree.Code)
	}
	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(rootDirPath)
	if err != nil {
		logger.Warnf(ctx, "[CopyServiceTree] 获取根目录 ServiceTree 失败: path=%s, error=%v", rootDirPath, err)
	}

	if rootTree != nil && hubDetail.FullCodePath != "" {
		rootTree.HubFullCodePath = hubDetail.FullCodePath
		rootTree.HubVersionNum = hubDetail.VersionNum
		if err := s.serviceTreeRepo.UpdateServiceTree(rootTree); err != nil {
			logger.Warnf(ctx, "[CopyServiceTree] 更新ServiceTree的Hub信息失败: treeID=%d, hubFullCodePath=%s, error=%v",
				rootTree.ID, hubDetail.FullCodePath, err)
		} else {
			logger.Infof(ctx, "[CopyServiceTree] 成功建立双向绑定: treeID=%d, hubFullCodePath=%s, hubVersion=%s", rootTree.ID, hubDetail.FullCodePath, fmt.Sprintf("v%d", rootTree.HubVersionNum))
		}
	}

	logger.Infof(ctx, "[CopyServiceTree] 从 Hub 复制目录完成: 目录数=%d, 文件数=%d, oldVersion=%s, newVersion=%s",
		directoryCount, fileCount, oldVersion, newVersion)

	if hubDetail.FullCodePath != "" {
		if errInc := apicall.IncrementDownloadCountOnHost(ctx, hubLinkInfo.Host, hubDetail.FullCodePath); errInc != nil {
			logger.Warnf(ctx, "[CopyServiceTree] 增加 Hub 下载次数失败: host=%s, path=%s, error=%v",
				hubLinkInfo.Host, hubDetail.FullCodePath, errInc)
		}
	}

	return &dto.CopyDirectoryResp{
		Message:        fmt.Sprintf("从 Hub 复制目录成功，共复制 %d 个目录，%d 个文件", directoryCount, fileCount),
		DirectoryCount: directoryCount,
		FileCount:      fileCount,
		OldVersion:     oldVersion,
		NewVersion:     newVersion,
		GitCommitHash:  gitCommitHash,
	}, nil
}

func publishDirectoryToHubImpl(s *serviceTreeHubService, ctx context.Context, req *dto.PublishDirectoryToHubReq) (*dto.PublishDirectoryToHubResp, error) {
	sourceApp, err := s.appRepo.GetAppByUserName(req.SourceUser, req.SourceApp)
	if err != nil {
		return nil, fmt.Errorf("获取源应用失败: %w", err)
	}

	sourceTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源目录信息失败: %w", err)
	}

	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取根目录节点失败: %w", err)
	}

	descendants, err := s.serviceTreeRepo.GetDescendantDirectories(sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("查询子目录失败: %w", err)
	}

	allTrees := make([]*model.ServiceTree, 0, len(descendants)+1)
	allTrees = append(allTrees, rootTree)
	allTrees = append(allTrees, descendants...)

	pathToTree := make(map[string]*model.ServiceTree)
	idToTree := make(map[int64]*model.ServiceTree)
	for _, tree := range allTrees {
		pathToTree[tree.FullCodePath] = tree
		idToTree[tree.ID] = tree
	}

	directoryFiles, err := s.getDirectoryFilesFromRuntimeRecursively(ctx, sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("从 runtime 读取目录文件失败: %w", err)
	}

	if len(directoryFiles) == 0 {
		return nil, fmt.Errorf("未找到任何目录，请确认源目录存在")
	}

	normalizedPath := strings.TrimSuffix(req.SourceDirectoryPath, "/") + "/"
	allFunctions, err := s.serviceTreeRepo.GetServiceTreesByAppIDAndType(sourceApp.ID, model.ServiceTreeTypeFunction)
	if err != nil {
		return nil, fmt.Errorf("查询函数节点失败: %w", err)
	}

	functionMap := make(map[int64][]*model.ServiceTree)
	for _, fn := range allFunctions {
		if strings.HasPrefix(fn.FullCodePath, normalizedPath) || fn.FullCodePath == req.SourceDirectoryPath {
			fnParentPath := fn.GetParentFullPath()
			if dirTree, exists := pathToTree[fnParentPath]; exists {
				if strings.HasPrefix(dirTree.FullCodePath, normalizedPath) || dirTree.FullCodePath == req.SourceDirectoryPath {
					functionMap[dirTree.ID] = append(functionMap[dirTree.ID], fn)
				}
			}
		}
	}

	refIDs := make([]int64, 0, len(allFunctions))
	for _, fn := range allFunctions {
		if fn.RefID > 0 {
			refIDs = append(refIDs, fn.RefID)
		}
	}
	refIDToFunction := make(map[int64]*model.Function)
	if len(refIDs) > 0 {
		functions, err := s.functionRepo.GetFunctionsByIDs(refIDs)
		if err == nil {
			for _, f := range functions {
				refIDToFunction[f.ID] = f
			}
		}
	}

	directoryTree := s.buildDirectoryTree(rootTree, allTrees, directoryFiles, idToTree, functionMap, refIDToFunction)
	if err := validateHubDirectoryTreeForPublishImpl(directoryTree); err != nil {
		return nil, fmt.Errorf("发布目录树校验失败: %w", err)
	}

	hubReq := &dto.PublishHubDirectoryReq{
		SourceUser:           req.SourceUser,
		SourceApp:            req.SourceApp,
		SourceDirectoryPath:  req.SourceDirectoryPath,
		Name:                 req.Name,
		Description:          req.Description,
		Category:             req.Category,
		Tags:                 req.Tags,
		ServiceFeePersonal:   req.ServiceFeePersonal,
		ServiceFeeEnterprise: req.ServiceFeeEnterprise,
		Version:              sourceTree.Version,
		DirectoryTree:        directoryTree,
	}

	var hubResp *dto.PublishHubDirectoryResp
	isRemote := req.RemoteHubURL != "" && req.PubKey != ""
	if isRemote {
		hubResp, err = apicall.PublishDirectoryToRemoteHub(ctx, req.RemoteHubURL, req.PubKey, hubReq)
	} else {
		hubResp, err = apicall.PublishDirectoryToHub(ctx, hubReq)
	}
	if err != nil {
		return nil, fmt.Errorf("调用 Hub API 失败: %w", err)
	}

	hubDetail, err := apicall.GetHubDirectoryDetail(ctx, &dto.GetHubDirectoryDetailReq{
		FullCodePath: req.SourceDirectoryPath,
		Version:      "",
		IncludeTree:  false,
	})
	if err != nil {
		logger.Warnf(ctx, "[PublishDirectoryToHub] 获取Hub目录详情失败，无法记录版本信息: error=%v", err)
		rootTree.HubFullCodePath = req.SourceDirectoryPath
	} else {
		rootTree.HubFullCodePath = hubDetail.FullCodePath
		rootTree.HubVersionNum = hubDetail.VersionNum
	}

	if err := s.serviceTreeRepo.UpdateServiceTree(rootTree); err != nil {
		logger.Warnf(ctx, "[PublishDirectoryToHub] 更新ServiceTree的Hub信息失败: treeID=%d, hubFullCodePath=%s, error=%v",
			rootTree.ID, rootTree.HubFullCodePath, err)
	} else {
		logger.Infof(ctx, "[PublishDirectoryToHub] 成功建立双向绑定: treeID=%d, hubFullCodePath=%s, hubVersion=%s", rootTree.ID, rootTree.HubFullCodePath, fmt.Sprintf("v%d", rootTree.HubVersionNum))
	}

	return &dto.PublishDirectoryToHubResp{
		HubFullCodePath: rootTree.HubFullCodePath,
		DirectoryCount:  hubResp.DirectoryCount,
		FileCount:       hubResp.FileCount,
	}, nil
}

func pushDirectoryToHubImpl(s *serviceTreeHubService, ctx context.Context, req *dto.PushDirectoryToHubReq) (*dto.PushDirectoryToHubResp, error) {
	sourceApp, err := s.appRepo.GetAppByUserName(req.SourceUser, req.SourceApp)
	if err != nil {
		return nil, fmt.Errorf("获取源应用失败: %w", err)
	}

	sourceTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源目录信息失败: %w", err)
	}

	if sourceTree.HubFullCodePath == "" {
		return nil, fmt.Errorf("目录尚未发布到 Hub，请先使用 PublishDirectoryToHub 发布")
	}

	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取根目录节点失败: %w", err)
	}

	descendants, err := s.serviceTreeRepo.GetDescendantDirectories(sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("查询子目录失败: %w", err)
	}

	allTrees := make([]*model.ServiceTree, 0, len(descendants)+1)
	allTrees = append(allTrees, rootTree)
	allTrees = append(allTrees, descendants...)

	idToTree := make(map[int64]*model.ServiceTree)
	pathToTreeLocal := make(map[string]*model.ServiceTree)
	for _, tree := range allTrees {
		idToTree[tree.ID] = tree
		pathToTreeLocal[tree.FullCodePath] = tree
	}

	directoryFiles, err := s.getDirectoryFilesFromRuntimeRecursively(ctx, sourceApp.ID, req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("从 runtime 读取目录文件失败: %w", err)
	}

	if len(directoryFiles) == 0 {
		return nil, fmt.Errorf("未找到任何目录，请确认源目录存在")
	}

	normalizedPath := strings.TrimSuffix(req.SourceDirectoryPath, "/") + "/"
	allFunctions, err := s.serviceTreeRepo.GetServiceTreesByAppIDAndType(sourceApp.ID, model.ServiceTreeTypeFunction)
	if err != nil {
		return nil, fmt.Errorf("查询函数节点失败: %w", err)
	}

	functionMap := make(map[int64][]*model.ServiceTree)
	for _, fn := range allFunctions {
		if strings.HasPrefix(fn.FullCodePath, normalizedPath) || fn.FullCodePath == req.SourceDirectoryPath {
			fnParentPath := fn.GetParentFullPath()
			if dirTree, exists := pathToTreeLocal[fnParentPath]; exists {
				if strings.HasPrefix(dirTree.FullCodePath, normalizedPath) || dirTree.FullCodePath == req.SourceDirectoryPath {
					functionMap[dirTree.ID] = append(functionMap[dirTree.ID], fn)
				}
			}
		}
	}

	refIDs := make([]int64, 0, len(allFunctions))
	for _, fn := range allFunctions {
		if fn.RefID > 0 {
			refIDs = append(refIDs, fn.RefID)
		}
	}
	refIDToFunction := make(map[int64]*model.Function)
	if len(refIDs) > 0 {
		functions, err := s.functionRepo.GetFunctionsByIDs(refIDs)
		if err == nil {
			for _, f := range functions {
				refIDToFunction[f.ID] = f
			}
		}
	}

	directoryTree := s.buildDirectoryTree(rootTree, allTrees, directoryFiles, idToTree, functionMap, refIDToFunction)
	if err := validateHubDirectoryTreeForPublishImpl(directoryTree); err != nil {
		return nil, fmt.Errorf("更新目录树校验失败: %w", err)
	}

	detailForID, errDetail := apicall.GetHubDirectoryDetail(ctx, &dto.GetHubDirectoryDetailReq{
		FullCodePath: sourceTree.HubFullCodePath,
		IncludeTree:  false,
	})
	if errDetail != nil || detailForID == nil {
		return nil, fmt.Errorf("无法获取 Hub 目录详情，请确认目录已发布到 Hub: %w", errDetail)
	}
	hubDirectoryIDForUpdate := detailForID.ID

	nextVersion := req.Version
	if nextVersion == "" {
		nextVersion = fmt.Sprintf("v%d", sourceTree.HubVersionNum+1)
	}

	hubReq := &dto.UpdateHubDirectoryReq{
		APIKey:               req.APIKey,
		HubDirectoryID:       hubDirectoryIDForUpdate,
		SourceDirectoryPath:  req.SourceDirectoryPath,
		Name:                 req.Name,
		Description:          req.Description,
		Category:             req.Category,
		Tags:                 req.Tags,
		ServiceFeePersonal:   req.ServiceFeePersonal,
		ServiceFeeEnterprise: req.ServiceFeeEnterprise,
		Version:              nextVersion,
		UpdateDescription:    req.UpdateDescription,
		DirectoryTree:        directoryTree,
	}

	var hubResp *dto.UpdateHubDirectoryResp
	isRemote := req.RemoteHubURL != "" && req.PubKey != ""
	if isRemote {
		hubResp, err = apicall.UpdateDirectoryToRemoteHub(ctx, req.RemoteHubURL, req.PubKey, hubReq)
	} else {
		hubResp, err = apicall.UpdateDirectoryToHub(ctx, hubReq)
	}
	if err != nil {
		return nil, fmt.Errorf("调用 Hub API 失败: %w", err)
	}

	rootTree.HubVersionNum = extractVersionNumForServiceTree(hubResp.NewVersion)
	if err := s.serviceTreeRepo.UpdateServiceTree(rootTree); err != nil {
		logger.Warnf(ctx, "[PushDirectoryToHub] 更新ServiceTree的Hub版本信息失败: treeID=%d, hubFullCodePath=%s, error=%v",
			rootTree.ID, rootTree.HubFullCodePath, err)
	} else {
		logger.Infof(ctx, "[PushDirectoryToHub] 成功更新Hub版本: treeID=%d, hubFullCodePath=%s, oldVersion=%s, newVersion=%s",
			rootTree.ID, rootTree.HubFullCodePath, hubResp.OldVersion, hubResp.NewVersion)
	}

	return &dto.PushDirectoryToHubResp{
		HubFullCodePath: rootTree.HubFullCodePath,
		DirectoryCount:  hubResp.DirectoryCount,
		FileCount:       hubResp.FileCount,
		OldVersion:      hubResp.OldVersion,
		NewVersion:      hubResp.NewVersion,
	}, nil
}

func getHubPushFormInfoImpl(s *serviceTreeHubService, ctx context.Context, req *dto.GetHubPushFormInfoReq) (*dto.GetHubPushFormInfoResp, error) {
	sourceTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.SourceDirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取源目录信息失败: %w", err)
	}
	if sourceTree.HubFullCodePath == "" {
		return nil, fmt.Errorf("目录尚未发布到 Hub，请先使用发布到应用中心")
	}
	detail, err := apicall.GetHubDirectoryDetail(ctx, &dto.GetHubDirectoryDetailReq{
		FullCodePath: sourceTree.HubFullCodePath,
		IncludeTree:  false,
	})
	if err != nil {
		return nil, fmt.Errorf("获取 Hub 目录详情失败: %w", err)
	}
	if detail == nil {
		return nil, fmt.Errorf("Hub 目录详情为空")
	}
	nextVer := fmt.Sprintf("v%d", detail.VersionNum+1)
	return &dto.GetHubPushFormInfoResp{
		Name:                 detail.Name,
		Description:          detail.Description,
		Category:             detail.Category,
		Tags:                 detail.Tags,
		ServiceFeePersonal:   detail.ServiceFeePersonal,
		ServiceFeeEnterprise: detail.ServiceFeeEnterprise,
		CurrentVersion:       detail.Version,
		NextVersion:          nextVer,
	}, nil
}

func buildDirectoryTreeImpl(s *serviceTreeHubService, rootTree *model.ServiceTree, allTrees []*model.ServiceTree, directoryFiles map[string][]*model.FileSnapshot, idToTree map[int64]*model.ServiceTree, functionMap map[int64][]*model.ServiceTree, refIDToFunction map[int64]*model.Function) *dto.DirectoryTreeNode {
	return s.buildDirectoryTreeNode(rootTree, allTrees, directoryFiles, idToTree, functionMap, refIDToFunction)
}

func buildDirectoryTreeNodeImpl(s *serviceTreeHubService, tree *model.ServiceTree, allTrees []*model.ServiceTree, directoryFiles map[string][]*model.FileSnapshot, idToTree map[int64]*model.ServiceTree, functionMap map[int64][]*model.ServiceTree, refIDToFunction map[int64]*model.Function) *dto.DirectoryTreeNode {
	files := make([]*dto.FileSnapshotInfo, 0)
	if fileSnapshots, exists := directoryFiles[tree.FullCodePath]; exists {
		for _, file := range fileSnapshots {
			if file.FileName == "init_" || file.RelativePath == "init_.go" || strings.HasSuffix(file.RelativePath, "/init_.go") {
				continue
			}
			files = append(files, &dto.FileSnapshotInfo{
				FileName:     file.FileName,
				RelativePath: file.RelativePath,
				Content:      file.Content,
				FileType:     file.FileType,
				FileVersion:  file.FileVersion,
			})
		}
	}

	functions := make([]*dto.HubFunctionInfo, 0)
	if functionList, exists := functionMap[tree.ID]; exists {
		for _, fn := range functionList {
			info := &dto.HubFunctionInfo{
				ID:           fn.ID,
				Name:         fn.Name,
				Code:         fn.Code,
				FullCodePath: fn.FullCodePath,
				Description:  fn.Description,
				TemplateType: fn.TemplateType,
				Tags:         fn.GetTagsSlice(),
				RefID:        fn.RefID,
				Version:      fn.Version,
				VersionNum:   fn.VersionNum,
			}
			if refIDToFunction != nil {
				if refFn, ok := refIDToFunction[fn.RefID]; ok && refFn != nil {
					info.Method = refFn.Method
					info.Router = refFn.Router
					info.CreateTables = refFn.CreateTables
					info.Callbacks = refFn.Callbacks
					schemaObj := map[string]interface{}{
						"request":  refFn.Request,
						"response": refFn.Response,
					}
					if schemaBytes, err := json.Marshal(schemaObj); err == nil {
						info.Schema = schemaBytes
					}
				}
			}
			functions = append(functions, info)
		}
	}

	treePrefix := strings.TrimSuffix(tree.FullCodePath, "/") + "/"
	treeDepth := tree.GetDepth()
	subdirectories := make([]*dto.DirectoryTreeNode, 0)
	for _, childTree := range allTrees {
		if strings.HasPrefix(childTree.FullCodePath, treePrefix) && childTree.GetDepth() == treeDepth+1 {
			childNode := s.buildDirectoryTreeNode(childTree, allTrees, directoryFiles, idToTree, functionMap, refIDToFunction)
			subdirectories = append(subdirectories, childNode)
		}
	}

	return &dto.DirectoryTreeNode{
		Type:           "package",
		Name:           tree.Name,
		Code:           tree.Code,
		Path:           tree.FullCodePath,
		Files:          files,
		Functions:      functions,
		Subdirectories: subdirectories,
	}
}

// HubLinkInfo Hub 链接信息
type HubLinkInfo struct {
	Host         string
	FullCodePath string
	Version      string
}

// ParseHubLink 解析 Hub 链接
func ParseHubLink(hubLink string) (*HubLinkInfo, error) {
	if !strings.HasPrefix(hubLink, "hub://") {
		return nil, fmt.Errorf("无效的 Hub 链接格式，必须以 hub:// 开头")
	}

	link := strings.TrimPrefix(hubLink, "hub://")

	var version string
	if idx := strings.LastIndex(link, "@"); idx != -1 {
		version = link[idx+1:]
		link = link[:idx]
	}

	parts := strings.SplitN(link, "/", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("无效的 Hub 链接格式，缺少 full-code-path")
	}

	host := parts[0]
	fullCodePath := "/" + parts[1]

	pathParts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("无效的 full-code-path 格式，应该至少包含 /user/app/...")
	}

	return &HubLinkInfo{
		Host:         host,
		FullCodePath: fullCodePath,
		Version:      version,
	}, nil
}

func installDirectoryTreeFromHubSnapshotImpl(s *serviceTreeHubService, ctx context.Context, tree *dto.DirectoryTreeNode, targetApp *model.App, targetPath, hubFullCodePath string, hubVersionNum int, hubDirectoryName, successMessagePrefix string) (*dto.PullDirectoryFromHubResp, error) {
	if tree == nil {
		return nil, fmt.Errorf("目录树为空")
	}

	directoryItems := make([]*dto.DirectoryScaffoldItem, 0)
	fileItems := make([]*dto.FileWriteItem, 0)
	s.buildItemsFromTree(tree, targetPath, &directoryItems, &fileItems)

	if len(directoryItems) > 0 {
		batchCreateReq := &dto.BatchCreateDirectoryTreeReq{
			User:  targetApp.User,
			App:   targetApp.Code,
			Items: directoryItems,
		}
		batchCreateResp, err := s.batchCreateDirectoryTree(ctx, batchCreateReq)
		if err != nil {
			return nil, fmt.Errorf("批量创建目录失败: %w", err)
		}
		logger.Infof(ctx, "[%s] 批量创建目录完成: directoryCount=%d", successMessagePrefix, batchCreateResp.DirectoryCount)
	}

	if len(fileItems) > 0 {
		batchWriteReq := &dto.BatchWriteFilesReq{
			User:  targetApp.User,
			App:   targetApp.Code,
			Files: fileItems,
		}
		batchWriteResp, err := s.batchWriteFiles(ctx, batchWriteReq)
		if err != nil {
			return nil, fmt.Errorf("批量写文件失败: %w", err)
		}
		logger.Infof(ctx, "[%s] 批量写文件完成: fileCount=%d", successMessagePrefix, batchWriteResp.FileCount)
	}

	rootDirPath := targetPath
	if tree.Code != "" {
		rootDirPath = fmt.Sprintf("%s/%s", targetPath, tree.Code)
	}
	rootTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(rootDirPath)
	if err != nil {
		logger.Warnf(ctx, "[%s] 获取根目录 ServiceTree 失败: path=%s, error=%v", successMessagePrefix, rootDirPath, err)
	}

	if rootTree != nil && hubFullCodePath != "" {
		rootTree.HubFullCodePath = hubFullCodePath
		rootTree.HubVersionNum = hubVersionNum
		if err := s.serviceTreeRepo.UpdateServiceTree(rootTree); err != nil {
			logger.Warnf(ctx, "[%s] 更新ServiceTree的Hub信息失败: treeID=%d, hubFullCodePath=%s, error=%v",
				successMessagePrefix, rootTree.ID, hubFullCodePath, err)
		} else {
			logger.Infof(ctx, "[%s] 成功建立双向绑定: treeID=%d, hubFullCodePath=%s, hubVersion=%s",
				successMessagePrefix, rootTree.ID, hubFullCodePath, fmt.Sprintf("v%d", rootTree.HubVersionNum))
		}
	}

	displayName := hubDirectoryName
	if displayName == "" {
		displayName = tree.Name
	}

	var serviceTreeID int64
	if rootTree != nil {
		serviceTreeID = rootTree.ID
	}

	return &dto.PullDirectoryFromHubResp{
		Message:             fmt.Sprintf("%s，共安装 %d 个目录，%d 个文件", successMessagePrefix, len(directoryItems), len(fileItems)),
		DirectoryCount:      len(directoryItems),
		FileCount:           len(fileItems),
		TargetDirectoryPath: rootDirPath,
		ServiceTreeID:       serviceTreeID,
		HubDirectoryName:    displayName,
		HubVersionNum:       hubVersionNum,
	}, nil
}

func pullDirectoryFromHubImpl(s *serviceTreeHubService, ctx context.Context, req *dto.PullDirectoryFromHubReq) (*dto.PullDirectoryFromHubResp, error) {
	hubLinkInfo, err := ParseHubLink(req.HubLink)
	if err != nil {
		return nil, fmt.Errorf("解析 Hub 链接失败: %w", err)
	}

	logger.Infof(ctx, "[PullDirectoryFromHub] 解析 Hub 链接成功: host=%s, fullCodePath=%s, version=%s",
		hubLinkInfo.Host, hubLinkInfo.FullCodePath, hubLinkInfo.Version)

	hubDetail, err := apicall.GetHubDirectoryDetailFromHost(ctx, &dto.GetHubDirectoryDetailFromHostReq{
		Host:         hubLinkInfo.Host,
		FullCodePath: hubLinkInfo.FullCodePath,
		Version:      hubLinkInfo.Version,
		IncludeTree:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("获取 Hub 目录详情失败: %w", err)
	}

	if hubLinkInfo.Version != "" && hubDetail.Version != hubLinkInfo.Version {
		return nil, fmt.Errorf("版本不匹配：请求版本 %s，实际版本 %s", hubLinkInfo.Version, hubDetail.Version)
	}

	targetApp, err := s.appRepo.GetAppByUserName(req.TargetUser, req.TargetApp)
	if err != nil {
		return nil, fmt.Errorf("获取目标应用失败: %w", err)
	}

	targetPath := req.TargetDirectoryPath
	if targetPath == "" {
		targetPath = fmt.Sprintf("/%s/%s", targetApp.User, targetApp.Code)
	}

	if hubDetail.DirectoryTree == nil {
		return nil, fmt.Errorf("Hub 目录树为空")
	}
	if err := validateHubDirectoryTreeForInstallImpl(hubDetail.DirectoryTree); err != nil {
		return nil, fmt.Errorf("Hub 目录树校验失败: %w", err)
	}

	return s.installDirectoryTreeFromHubSnapshot(ctx, hubDetail.DirectoryTree, targetApp, targetPath,
		hubDetail.FullCodePath, hubDetail.VersionNum, hubDetail.Name,
		"从 Hub 安装目录成功")
}

func importHubDirectoryBundleImpl(s *serviceTreeHubService, ctx context.Context, req *dto.ImportHubDirectoryBundleReq) (*dto.PullDirectoryFromHubResp, error) {
	targetApp, err := s.appRepo.GetAppByUserName(req.TargetUser, req.TargetApp)
	if err != nil {
		return nil, fmt.Errorf("获取目标应用失败: %w", err)
	}
	targetPath := req.TargetDirectoryPath
	if targetPath == "" {
		targetPath = fmt.Sprintf("/%s/%s", targetApp.User, targetApp.Code)
	}
	if err := validateHubDirectoryInstallBundleForImportImpl(req.Bundle); err != nil {
		return nil, err
	}
	return s.installDirectoryTreeFromHubSnapshot(ctx, req.Bundle.DirectoryTree, targetApp, targetPath,
		req.Bundle.HubFullCodePath, req.Bundle.HubVersionNum, req.Bundle.HubDirectoryName,
		"从离线包安装目录成功")
}

func countFilesInTreeImpl(s *serviceTreeHubService, node *dto.DirectoryTreeNode) int {
	count := len(node.Files)
	for _, subdir := range node.Subdirectories {
		count += s.countFilesInTree(subdir)
	}
	return count
}

func logDirectoryTreeImpl(s *serviceTreeHubService, ctx context.Context, node *dto.DirectoryTreeNode, level int) {
	indent := strings.Repeat("  ", level)
	logger.Infof(ctx, "%s[logDirectoryTree] 节点: Name=%s, Code=%s, Path=%s, Files数量=%d, Subdirectories数量=%d",
		indent, node.Name, node.Code, node.Path, len(node.Files), len(node.Subdirectories))

	for i, file := range node.Files {
		logger.Infof(ctx, "%s  [文件%d] FileName=%s, RelativePath=%s, FileType=%s, Content长度=%d",
			indent, i+1, file.FileName, file.RelativePath, file.FileType, len(file.Content))
	}

	for i, subdir := range node.Subdirectories {
		logger.Infof(ctx, "%s  [子目录%d]", indent, i+1)
		s.logDirectoryTree(ctx, subdir, level+1)
	}
}

func buildItemsFromTreeImpl(s *serviceTreeHubService, node *dto.DirectoryTreeNode, targetBasePath string, directoryItems *[]*dto.DirectoryScaffoldItem, fileItems *[]*dto.FileWriteItem) {
	dirCode := node.Code
	logger.Infof(context.Background(), "[buildItemsFromTree] 处理节点: Name=%s, Code=%s, Path=%s, Files数量=%d",
		node.Name, node.Code, node.Path, len(node.Files))

	if dirCode == "" {
		logger.Warnf(context.Background(), "[buildItemsFromTree] ⚠️ Code 字段为空！Name=%s, Path=%s", node.Name, node.Path)
	}

	currentTargetPath := fmt.Sprintf("%s/%s", targetBasePath, dirCode)

	dirName := node.Name
	if dirName == "" {
		dirName = dirCode
	}
	*directoryItems = append(*directoryItems, &dto.DirectoryScaffoldItem{
		FullCodePath: currentTargetPath,
		Name:         dirName,
		Description:  "",
		Tags:         "",
	})

	logger.Infof(context.Background(), "[buildItemsFromTree] 处理目录 %s，文件数量: %d", currentTargetPath, len(node.Files))
	if len(node.Files) == 0 {
		logger.Warnf(context.Background(), "[buildItemsFromTree] ⚠️ 目录 %s 没有文件！Name=%s, Code=%s, Path=%s", currentTargetPath, node.Name, node.Code, node.Path)
	}
	for i, file := range node.Files {
		logger.Infof(context.Background(), "[buildItemsFromTree] 处理文件[%d]: FileName=%s, RelativePath=%s, FileType=%s, Content长度=%d",
			i+1, file.FileName, file.RelativePath, file.FileType, len(file.Content))
		fileName := file.FileName
		if fileName == "" {
			pathParts := strings.Split(file.RelativePath, "/")
			fileName = pathParts[len(pathParts)-1]
			if ext := strings.LastIndex(fileName, "."); ext != -1 {
				fileName = fileName[:ext]
			}
		}

		if file.Content == "" {
			logger.Warnf(context.Background(), "[buildItemsFromTree] 文件 %s 内容为空", fileName)
		}

		*fileItems = append(*fileItems, &dto.FileWriteItem{
			FullCodePath: currentTargetPath,
			FileName:     fileName,
			FileType:     file.FileType,
			Content:      file.Content,
			RelativePath: file.RelativePath,
		})
		logger.Infof(context.Background(), "[buildItemsFromTree] 添加文件: %s, 内容长度: %d", fileName, len(file.Content))
	}

	for _, subdir := range node.Subdirectories {
		s.buildItemsFromTree(subdir, currentTargetPath, directoryItems, fileItems)
	}
}

func getHubInfoImpl(s *serviceTreeHubService, ctx context.Context, req *dto.GetHubInfoReq) (*dto.GetHubInfoResp, error) {
	tree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(req.FullCodePath)
	if err != nil {
		return nil, fmt.Errorf("获取目录信息失败: %w", err)
	}

	if tree.HubFullCodePath == "" {
		return nil, fmt.Errorf("目录未发布到 Hub")
	}

	hubDetail, err := apicall.GetHubDirectoryDetail(ctx, &dto.GetHubDirectoryDetailReq{
		FullCodePath: tree.HubFullCodePath,
		Version:      "",
		IncludeTree:  false,
	})
	if err != nil {
		logger.Warnf(ctx, "[GetHubInfo] 获取 Hub 目录详情失败: fullCodePath=%s, error=%v", req.FullCodePath, err)
		return &dto.GetHubInfoResp{
			HubFullCodePath: tree.HubFullCodePath,
			PublishedAt:     "",
		}, nil
	}

	return &dto.GetHubInfoResp{
		HubFullCodePath: hubDetail.FullCodePath,
		PublishedAt:     hubDetail.PublishedAt,
	}, nil
}
