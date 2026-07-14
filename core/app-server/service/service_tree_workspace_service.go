package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/functionschema"
	"github.com/kageos/kageos/pkg/logger"
)

type serviceTreeWorkspaceService struct {
	serviceTreeRepo  *repository.ServiceTreeRepository
	fileSnapshotRepo *repository.FileSnapshotRepository
	runtimeWorkspace *runtimeWorkspaceBridge
	queryView        *serviceTreeQueryView
}

func newServiceTreeWorkspaceService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	fileSnapshotRepo *repository.FileSnapshotRepository,
	runtimeWorkspace *runtimeWorkspaceBridge,
	queryView *serviceTreeQueryView,
) *serviceTreeWorkspaceService {
	return &serviceTreeWorkspaceService{
		serviceTreeRepo:  serviceTreeRepo,
		fileSnapshotRepo: fileSnapshotRepo,
		runtimeWorkspace: runtimeWorkspace,
		queryView:        queryView,
	}
}

func (s *serviceTreeWorkspaceService) GetWorkspaceContext(ctx context.Context, req *dto.GetWorkspaceContextReq) (*dto.GetWorkspaceContextResp, error) {
	detail, err := s.queryView.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{
		FullCodePath: req.FullCodePath,
	})
	if err != nil {
		return nil, fmt.Errorf("获取目录详情失败: %w", err)
	}

	children, err := s.serviceTreeRepo.GetServiceTreeChildren(ctx, detail.ID)
	if err != nil {
		return nil, fmt.Errorf("获取子节点列表失败: %w", err)
	}

	childrenNodes := make([]dto.WorkspaceContextNode, 0, len(children))
	for _, child := range children {
		callbacks := []string(nil)
		var schema *functionschema.FunctionSchema
		if child.Function != nil {
			callbacks = child.Function.GetCallbacks()
			if child.Function.Connectors != "" {
				child.Connectors = child.Function.Connectors
			}
			if child.Function.ConnectorEndpoints != "" {
				child.ConnectorEndpoints = child.Function.ConnectorEndpoints
			}
			if parsed, err := functionschema.Parse(child.Function.Schema); err == nil {
				schema = parsed
			}
		}
		childrenNodes = append(childrenNodes, dto.WorkspaceContextNode{
			ID:                 child.ID,
			Name:               child.Name,
			Code:               child.Code,
			Type:               child.Type,
			Description:        child.Description,
			FullCodePath:       child.FullCodePath,
			TemplateType:       child.TemplateType,
			Callbacks:          callbacks,
			Connectors:         splitConnectorCodes(child.Connectors),
			ConnectorEndpoints: splitConnectorEndpoints(child.ConnectorEndpoints),
			Schema:             schema,
		})
	}
	sort.SliceStable(childrenNodes, func(i, j int) bool {
		left, right := childrenNodes[i], childrenNodes[j]
		if left.FullCodePath != right.FullCodePath {
			return left.FullCodePath < right.FullCodePath
		}
		if left.Type != right.Type {
			return left.Type < right.Type
		}
		return left.ID < right.ID
	})

	username := contextx.GetRequestUser(ctx)
	departmentFullPath := contextx.GetRequestDepartmentFullPath(ctx)
	departmentFullNamePath := ""
	if departmentFullPath != "" {
		deptResp, errDept := apicall.GetDepartmentsByPaths(ctx, []string{departmentFullPath})
		if errDept == nil && deptResp != nil && len(deptResp.Departments) > 0 && deptResp.Departments[0].FullNamePath != "" {
			departmentFullNamePath = deptResp.Departments[0].FullNamePath
		}
	}

	var files []dto.WorkspaceContextFile
	if detail.AppID > 0 {
		_, runtimeResp, errRt := s.runtimeWorkspace.readDirectoryFiles(ctx, detail.AppID, req.FullCodePath)
		if errRt == nil && runtimeResp != nil && runtimeResp.Success {
			files = make([]dto.WorkspaceContextFile, 0, len(runtimeResp.Files))
			for _, f := range runtimeResp.Files {
				if isInternalWorkspaceManifestFile(f.RelativePath, f.FileName) {
					continue
				}
				lineCount := 0
				if f.Content != "" {
					lines := strings.Split(f.Content, "\n")
					lineCount = len(lines)
					if lineCount > 0 && lines[lineCount-1] == "" {
						lineCount--
					}
				}
				fileType := strings.TrimSpace(f.FileType)
				if fileType == "" {
					fileType = "go"
				}
				files = append(files, dto.WorkspaceContextFile{
					FileName:      f.FileName,
					RelativePath:  f.RelativePath,
					FileType:      fileType,
					Content:       f.Content,
					ContentLength: len(f.Content),
					LineCount:     lineCount,
				})
			}
			sort.SliceStable(files, func(i, j int) bool {
				if files[i].RelativePath != files[j].RelativePath {
					return files[i].RelativePath < files[j].RelativePath
				}
				return files[i].FileName < files[j].FileName
			})
			logger.Infof(ctx, "[GetWorkspaceContext] 从 runtime 读取目录文件: fullCodePath=%s, fileCount=%d", req.FullCodePath, len(files))
		}
		if files == nil {
			files = []dto.WorkspaceContextFile{}
		}
	}

	return &dto.GetWorkspaceContextResp{
		User:                   username,
		DepartmentFullPath:     departmentFullPath,
		DepartmentFullNamePath: departmentFullNamePath,
		Directory: dto.WorkspaceContextDirectory{
			ID:           detail.ID,
			Name:         detail.Name,
			Code:         detail.Code,
			FullCodePath: detail.FullCodePath,
			Description:  detail.Description,
			Type:         detail.Type,
		},
		Children: childrenNodes,
		Files:    files,
	}, nil
}

func (s *serviceTreeWorkspaceService) WriteFileContent(ctx context.Context, req *dto.WriteFileContentReq) (*dto.WriteFileContentResp, error) {
	detail, err := s.queryView.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{FullCodePath: req.FullCodePath})
	if err != nil {
		return nil, fmt.Errorf("获取目录详情失败: %w", err)
	}
	if detail.AppID <= 0 {
		return nil, fmt.Errorf("该目录不属于应用，无法写入文件")
	}
	appModel, err := s.runtimeWorkspace.getRuntimeBoundAppByID(ctx, detail.AppID, "写入文件")
	if err != nil {
		return nil, err
	}

	runtimeReq := &dto.WriteFileRuntimeReq{
		User:          appModel.User,
		App:           appModel.Code,
		DirectoryPath: req.FullCodePath,
		FileName:      req.FileName,
		FileType:      req.FileType,
		Content:       req.Content,
	}
	resp, err := s.runtimeWorkspace.writeFile(ctx, appModel, runtimeReq)
	if err != nil {
		return nil, fmt.Errorf("写入文件失败: %w", err)
	}
	if resp == nil {
		return &dto.WriteFileContentResp{Success: true, Message: "写入成功"}, nil
	}
	return &dto.WriteFileContentResp{
		Success:      resp.Success,
		Message:      resp.Message,
		RelativePath: resp.RelativePath,
		FileType:     resp.FileType,
	}, nil
}

func (s *serviceTreeWorkspaceService) ReplaceFileContent(ctx context.Context, req *dto.ReplaceFileContentReq) (*dto.ReplaceFileContentResp, error) {
	if len(req.Replacements) == 0 {
		return &dto.ReplaceFileContentResp{Success: false, Message: "replacements 不能为空"}, nil
	}
	detail, err := s.queryView.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{FullCodePath: req.FullCodePath})
	if err != nil {
		return nil, fmt.Errorf("获取目录详情失败: %w", err)
	}
	if detail.AppID <= 0 {
		return nil, fmt.Errorf("该目录不属于应用，无法替换文件")
	}
	appModel, err := s.runtimeWorkspace.getRuntimeBoundAppByID(ctx, detail.AppID, "替换文件")
	if err != nil {
		return nil, err
	}

	items := make([]dto.ReplaceItemRuntime, 0, len(req.Replacements))
	for _, r := range req.Replacements {
		items = append(items, dto.ReplaceItemRuntime{
			SearchString:  r.SearchString,
			ReplaceString: r.ReplaceString,
			ExpectedCount: r.ExpectedCount,
		})
	}
	allOrNothing := req.AllOrNothing
	if !allOrNothing {
		allOrNothing = true
	}
	runtimeReq := &dto.ReplaceInFileBatchReq{
		User:              appModel.User,
		App:               appModel.Code,
		DirectoryPath:     req.FullCodePath,
		FileName:          req.FileName,
		Replacements:      items,
		AllOrNothing:      allOrNothing,
		ReturnFullContent: req.ReturnFullContent,
	}
	resp, err := s.runtimeWorkspace.replaceInFileBatch(ctx, appModel, runtimeReq)
	if err != nil {
		return nil, fmt.Errorf("替换文件失败: %w", err)
	}
	if !resp.Success {
		details := make([]dto.ReplaceItemResult, 0, len(resp.Details))
		for _, d := range resp.Details {
			details = append(details, dto.ReplaceItemResult{Index: d.Index, ExpectedCount: d.ExpectedCount, ActualCount: d.ActualCount})
		}
		return &dto.ReplaceFileContentResp{Success: false, Message: resp.Message, Details: details}, nil
	}
	out := &dto.ReplaceFileContentResp{Success: true, Message: resp.Message, ReplaceCount: resp.ReplaceCount}
	if req.ReturnFullContent && resp.FullContent != "" {
		out.FullContent = resp.FullContent
	}
	return out, nil
}

func (s *serviceTreeWorkspaceService) DeleteFile(ctx context.Context, req *dto.DeleteFileReq) (*dto.DeleteFileResp, error) {
	detail, err := s.queryView.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{FullCodePath: req.FullCodePath})
	if err != nil {
		return nil, fmt.Errorf("获取目录详情失败: %w", err)
	}
	if detail.AppID <= 0 {
		return nil, fmt.Errorf("该目录不属于应用")
	}
	appModel, err := s.runtimeWorkspace.getRuntimeBoundAppByID(ctx, detail.AppID, "删除文件")
	if err != nil {
		return nil, err
	}
	runtimeReq := &dto.DeleteFileRuntimeReq{
		User:          appModel.User,
		App:           appModel.Code,
		DirectoryPath: req.FullCodePath,
		FileName:      req.FileName,
	}
	_, err = s.runtimeWorkspace.deleteFile(ctx, appModel, runtimeReq)
	if err != nil {
		logger.Warnf(ctx, "[DeleteFile] runtime 删文件失败: %v", err)
		return nil, fmt.Errorf("删除文件失败: %w", err)
	}
	return &dto.DeleteFileResp{Success: true, Message: "已删除"}, nil
}

func (s *serviceTreeWorkspaceService) ReadAppLog(ctx context.Context, req *dto.ReadAppLogReq) (*dto.ReadAppLogResp, error) {
	detail, err := s.queryView.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{FullCodePath: req.FullCodePath})
	if err != nil {
		return nil, fmt.Errorf("获取目录详情失败: %w", err)
	}
	if detail.AppID <= 0 {
		return nil, fmt.Errorf("该目录不属于应用")
	}
	appModel, err := s.runtimeWorkspace.getRuntimeBoundAppByID(ctx, detail.AppID, "读取日志")
	if err != nil {
		return nil, err
	}

	version := strings.TrimSpace(req.Version)
	if version == "" {
		version = strings.TrimSpace(appModel.Version)
	}
	runtimeReq := &dto.ReadAppLogRuntimeReq{
		User:         appModel.User,
		App:          appModel.Code,
		Version:      version,
		Lines:        req.Lines,
		Keyword:      req.Keyword,
		ContextLines: req.ContextLines,
		MaxMatches:   req.MaxMatches,
		IgnoreCase:   req.IgnoreCase,
	}
	resp, err := s.runtimeWorkspace.readAppLog(ctx, appModel, runtimeReq)
	if err != nil {
		return nil, fmt.Errorf("读取日志失败: %w", err)
	}
	return &dto.ReadAppLogResp{
		Success:         resp.Success,
		Message:         resp.Message,
		ResolvedVersion: resp.ResolvedVersion,
		LogFile:         resp.LogFile,
		TotalLines:      resp.TotalLines,
		ReturnedLines:   resp.ReturnedLines,
		MatchCount:      resp.MatchCount,
		Truncated:       resp.Truncated,
		Content:         resp.Content,
	}, nil
}

func (s *serviceTreeWorkspaceService) GetDirectorySnapshotsRecursively(ctx context.Context, appID int64, rootDirectoryPath string) (map[string][]*model.FileSnapshot, error) {
	return readDirectorySnapshotsRecursively(ctx, s.serviceTreeRepo, s.fileSnapshotRepo, appID, rootDirectoryPath)
}

func (s *serviceTreeWorkspaceService) GetDirectoryFilesFromRuntimeRecursively(ctx context.Context, appID int64, rootDirectoryPath string) (map[string][]*model.FileSnapshot, error) {
	return readDirectoryFilesFromRuntimeRecursively(ctx, s.serviceTreeRepo, s.runtimeWorkspace, appID, rootDirectoryPath)
}
