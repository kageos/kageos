package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

func (s *ServiceTreeService) GetWorkspaceContext(ctx context.Context, req *dto.GetWorkspaceContextReq) (*dto.GetWorkspaceContextResp, error) {
	return s.workspaceService.GetWorkspaceContext(ctx, req)
}

func (s *ServiceTreeService) ReplaceFileContent(ctx context.Context, req *dto.ReplaceFileContentReq) (*dto.ReplaceFileContentResp, error) {
	return s.workspaceService.ReplaceFileContent(ctx, req)
}

func (s *ServiceTreeService) DeleteFile(ctx context.Context, req *dto.DeleteFileReq) (*dto.DeleteFileResp, error) {
	return s.workspaceService.DeleteFile(ctx, req)
}

func (s *ServiceTreeService) ReadAppLog(ctx context.Context, req *dto.ReadAppLogReq) (*dto.ReadAppLogResp, error) {
	return s.workspaceService.ReadAppLog(ctx, req)
}

func (s *ServiceTreeService) GetDirectorySnapshotsRecursively(ctx context.Context, appID int64, rootDirectoryPath string) (map[string][]*model.FileSnapshot, error) {
	return s.workspaceService.GetDirectorySnapshotsRecursively(ctx, appID, rootDirectoryPath)
}

func (s *ServiceTreeService) GetDirectoryFilesFromRuntimeRecursively(ctx context.Context, appID int64, rootDirectoryPath string) (map[string][]*model.FileSnapshot, error) {
	return s.workspaceService.GetDirectoryFilesFromRuntimeRecursively(ctx, appID, rootDirectoryPath)
}
