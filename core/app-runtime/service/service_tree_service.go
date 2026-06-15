package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
)

// WorkspaceChangeService 管理目录脚手架、批量文件写入和发布前后的回滚编排。
type WorkspaceChangeService struct {
	config               *config.AppManageServiceConfig
	appManageService     *AppManageService       // 用于编译和获取 diff
	workspaceFileService *WorkspaceFileService   // 用于文件写盘和回滚
	packageScaffold      *PackageScaffoldService // 用于目录脚手架维护
}

// NewWorkspaceChangeService 创建工作区变更编排服务。
func NewWorkspaceChangeService(
	config *config.AppManageServiceConfig,
	appManageService *AppManageService,
	workspaceFileService *WorkspaceFileService,
) *WorkspaceChangeService {
	return &WorkspaceChangeService{
		config:               config,
		appManageService:     appManageService,
		workspaceFileService: workspaceFileService,
		packageScaffold:      NewPackageScaffoldService(config),
	}
}

func (s *WorkspaceChangeService) SetAppDatabaseService(appDatabaseService *AppDatabaseService) {
	if s != nil && s.packageScaffold != nil {
		s.packageScaffold.SetAppDatabaseService(appDatabaseService)
	}
}

// DeleteServiceTree 删除服务目录（删磁盘目录，并从 main.go 移除该包的 import）
func (s *WorkspaceChangeService) DeleteServiceTree(ctx context.Context, user, app, serviceTreeName string) error {
	return s.packageScaffold.DeleteServiceTree(ctx, user, app, serviceTreeName)
}

// DeleteServiceTreeByReq 按请求删除服务目录（供 NATS 调用）
func (s *WorkspaceChangeService) DeleteServiceTreeByReq(ctx context.Context, req *dto.DeleteServiceTreeRuntimeReq) (*dto.DeleteServiceTreeRuntimeResp, error) {
	if err := s.DeleteServiceTree(ctx, req.User, req.App, req.PackagePath); err != nil {
		return &dto.DeleteServiceTreeRuntimeResp{Success: false, Error: err.Error()}, nil
	}
	return &dto.DeleteServiceTreeRuntimeResp{Success: true}, nil
}

// BatchCreateDirectoryTree 批量创建目录树（只处理目录，不处理文件）
// 文件写入请使用 BatchWriteFiles 方法
func (s *WorkspaceChangeService) BatchCreateDirectoryTree(
	ctx context.Context,
	req *dto.BatchCreateDirectoryTreeRuntimeReq,
) (*dto.BatchCreateDirectoryTreeRuntimeResp, error) {
	return s.packageScaffold.BatchCreateDirectoryTree(ctx, req)
}

// BatchWriteFiles 批量写文件（批量写文件，编译，返回 diff）
func (s *WorkspaceChangeService) BatchWriteFiles(
	ctx context.Context,
	req *dto.BatchWriteFilesRuntimeReq,
) (*dto.BatchWriteFilesRuntimeResp, error) {
	operationName := strings.TrimSpace(req.OperationName)
	if operationName == "" {
		operationName = "BatchWriteFiles"
	}
	operationLabel := strings.TrimSpace(req.OperationLabel)
	if operationLabel == "" {
		operationLabel = "批量写文件"
	}

	logger.Infof(ctx, "[WorkspaceChangeService] 开始%s: user=%s, app=%s, fileCount=%d",
		operationLabel, req.User, req.App, len(req.Files))

	if len(req.Files) == 0 {
		return nil, fmt.Errorf("没有需要写入的文件")
	}
	if s.workspaceFileService == nil {
		return nil, fmt.Errorf("workspaceFileService 未设置，无法写入文件")
	}
	if s.appManageService == nil {
		return nil, fmt.Errorf("appManageService 未设置，无法编译应用")
	}

	state, err := s.workspaceFileService.writeDirectoryTreeFiles(ctx, req.User, req.App, req.Files)
	if err != nil {
		return nil, err
	}

	appPaths := newRuntimeAppPaths(s.config.GetBasePath(), req.User, req.App)
	result, err := s.appManageService.finalizeWrittenAppChanges(ctx, req.User, req.App, appPaths, req.ForceDiff, operationName)
	if err != nil {
		s.appManageService.rollbackWrittenFilesAfterFailedBuild(ctx, operationName, state)
		return nil, err
	}

	logger.Infof(ctx, "[WorkspaceChangeService] %s并编译完成: oldVersion=%s, newVersion=%s", operationLabel, result.oldVersion, result.newVersion)

	return s.buildBatchWriteFilesRuntimeResp(state, result), nil
}

func (s *WorkspaceChangeService) buildBatchWriteFilesRuntimeResp(
	state *batchWriteState,
	release *appReleaseResult,
) *dto.BatchWriteFilesRuntimeResp {
	fileCount := 0
	writtenPaths := []string{}
	if state != nil {
		fileCount = len(state.writtenPaths)
		writtenPaths = state.writtenPaths
	}

	return &dto.BatchWriteFilesRuntimeResp{
		FileCount:     fileCount,
		WrittenPaths:  writtenPaths,
		Diff:          release.diff,
		OldVersion:    release.oldVersion,
		NewVersion:    release.newVersion,
		GitCommitHash: release.gitCommitHash,
	}
}
