package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// GetWorkspaceContext 获取工作台环境信息（用于构建 LLM 上下文）
func (s *ServiceTreeService) GetWorkspaceContext(ctx context.Context, req *dto.GetWorkspaceContextReq) (*dto.GetWorkspaceContextResp, error) {
	detail, err := s.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{
		FullCodePath: req.FullCodePath,
	})
	if err != nil {
		return nil, fmt.Errorf("获取目录详情失败: %w", err)
	}

	children, err := s.serviceTreeRepo.GetServiceTreeChildren(detail.ID)
	if err != nil {
		return nil, fmt.Errorf("获取子节点列表失败: %w", err)
	}

	childrenNodes := make([]dto.WorkspaceContextNode, 0, len(children))
	for _, child := range children {
		childrenNodes = append(childrenNodes, dto.WorkspaceContextNode{
			ID:           child.ID,
			Name:         child.Name,
			Code:         child.Code,
			Type:         child.Type,
			Description:  child.Description,
			FullCodePath: child.FullCodePath,
			TemplateType: child.TemplateType,
		})
	}

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
				lineCount := 0
				if f.Content != "" {
					lines := strings.Split(f.Content, "\n")
					lineCount = len(lines)
					if lineCount > 0 && lines[lineCount-1] == "" {
						lineCount--
					}
				}
				files = append(files, dto.WorkspaceContextFile{
					FileName:      f.FileName,
					RelativePath:  f.RelativePath,
					FileType:      "go",
					Content:       f.Content,
					ContentLength: len(f.Content),
					LineCount:     lineCount,
				})
			}
			logger.Infof(ctx, "[GetWorkspaceContext] 从 runtime 读取目录文件: fullCodePath=%s, fileCount=%d", req.FullCodePath, len(files))
		}
		if files == nil {
			files = []dto.WorkspaceContextFile{}
		}
	}

	publishedToHub := detail.HubFullCodePath != ""
	return &dto.GetWorkspaceContextResp{
		User:                   username,
		DepartmentFullPath:     departmentFullPath,
		DepartmentFullNamePath: departmentFullNamePath,
		Directory: dto.WorkspaceContextDirectory{
			ID:              detail.ID,
			Name:            detail.Name,
			Code:            detail.Code,
			FullCodePath:    detail.FullCodePath,
			Description:     detail.Description,
			Type:            detail.Type,
			PublishedToHub:  publishedToHub,
			HubFullCodePath: detail.HubFullCodePath,
		},
		Children: childrenNodes,
		Files:    files,
	}, nil
}

// ReplaceFileContent 工作台文件 search-replace（统一批量：调 app-runtime 内存替换、全部校验通过才落盘）
func (s *ServiceTreeService) ReplaceFileContent(ctx context.Context, req *dto.ReplaceFileContentReq) (*dto.ReplaceFileContentResp, error) {
	if len(req.Replacements) == 0 {
		return &dto.ReplaceFileContentResp{Success: false, Message: "replacements 不能为空"}, nil
	}
	detail, err := s.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{FullCodePath: req.FullCodePath})
	if err != nil {
		return nil, fmt.Errorf("获取目录详情失败: %w", err)
	}
	if detail.AppID <= 0 {
		return nil, fmt.Errorf("该目录不属于应用，无法替换文件")
	}
	appModel, err := s.runtimeWorkspace.getRuntimeBoundAppByID(detail.AppID, "替换文件")
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

// DeleteFile 工作台删除文件：仅调用 runtime 删除磁盘上的 Go 文件（与 read_go_file 一致：目录 + file_name 定位，不涉及节点）
func (s *ServiceTreeService) DeleteFile(ctx context.Context, req *dto.DeleteFileReq) (*dto.DeleteFileResp, error) {
	detail, err := s.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{FullCodePath: req.FullCodePath})
	if err != nil {
		return nil, fmt.Errorf("获取目录详情失败: %w", err)
	}
	if detail.AppID <= 0 {
		return nil, fmt.Errorf("该目录不属于应用")
	}
	appModel, err := s.runtimeWorkspace.getRuntimeBoundAppByID(detail.AppID, "删除文件")
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

// ReadAppLog 读取应用日志（支持 version、关键词检索）
func (s *ServiceTreeService) ReadAppLog(ctx context.Context, req *dto.ReadAppLogReq) (*dto.ReadAppLogResp, error) {
	detail, err := s.GetServiceTreeDetail(ctx, &dto.GetServiceTreeDetailReq{FullCodePath: req.FullCodePath})
	if err != nil {
		return nil, fmt.Errorf("获取目录详情失败: %w", err)
	}
	if detail.AppID <= 0 {
		return nil, fmt.Errorf("该目录不属于应用")
	}
	appModel, err := s.runtimeWorkspace.getRuntimeBoundAppByID(detail.AppID, "读取日志")
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
