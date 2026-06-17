package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/appcall"
)

type runtimeWorkspaceBridge struct {
	appRepo *repository.AppRepository
	appCall runtimeWorkspaceClient
}

type runtimeWorkspaceClient interface {
	BatchCreateDirectoryTree(ctx context.Context, hostID int64, req *dto.BatchCreateDirectoryTreeRuntimeReq) (*dto.BatchCreateDirectoryTreeRuntimeResp, error)
	BatchWriteFiles(ctx context.Context, hostID int64, req *dto.BatchWriteFilesRuntimeReq) (*dto.BatchWriteFilesRuntimeResp, error)
	DeleteServiceTree(ctx context.Context, hostID int64, req *dto.DeleteServiceTreeRuntimeReq) (*dto.DeleteServiceTreeRuntimeResp, error)
	ReadDirectoryFiles(ctx context.Context, hostID int64, req *dto.ReadDirectoryFilesRuntimeReq) (*dto.ReadDirectoryFilesRuntimeResp, error)
	ReplaceInFileBatch(ctx context.Context, hostID int64, req *dto.ReplaceInFileBatchReq) (*dto.ReplaceInFileBatchResp, error)
	DeleteFile(ctx context.Context, hostID int64, req *dto.DeleteFileRuntimeReq) (*dto.DeleteFileRuntimeResp, error)
	ReadAppLog(ctx context.Context, hostID int64, req *dto.ReadAppLogRuntimeReq) (*dto.ReadAppLogRuntimeResp, error)
}

var _ runtimeWorkspaceClient = (*appcall.Client)(nil)

func newRuntimeWorkspaceBridge(appRepo *repository.AppRepository, appCall runtimeWorkspaceClient) *runtimeWorkspaceBridge {
	return &runtimeWorkspaceBridge{
		appRepo: appRepo,
		appCall: appCall,
	}
}

func (b *runtimeWorkspaceBridge) getAppByUserApp(user, app string) (*model.App, error) {
	appModel, err := b.appRepo.GetAppByUserName(user, app)
	if err != nil {
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}
	return appModel, nil
}

func (b *runtimeWorkspaceBridge) getRuntimeBoundAppByUserApp(user, app, action string) (*model.App, error) {
	appModel, err := b.getAppByUserApp(user, app)
	if err != nil {
		return nil, err
	}
	return b.requireRuntimeBinding(appModel, action)
}

func (b *runtimeWorkspaceBridge) getRuntimeBoundAppByID(appID int64, action string) (*model.App, error) {
	appModel, err := b.appRepo.GetAppByID(appID)
	if err != nil {
		return nil, fmt.Errorf("获取应用失败: %w", err)
	}
	return b.requireRuntimeBinding(appModel, action)
}

func (b *runtimeWorkspaceBridge) requireRuntimeBinding(appModel *model.App, action string) (*model.App, error) {
	if appModel == nil {
		return nil, fmt.Errorf("应用不存在")
	}
	if appModel.HostID <= 0 {
		return nil, fmt.Errorf("应用未关联 runtime，无法%s", action)
	}
	return appModel, nil
}

func (b *runtimeWorkspaceBridge) createDirectoryScaffold(
	ctx context.Context,
	user, app string,
	serviceTree *model.ServiceTree,
) error {
	appModel, err := b.getRuntimeBoundAppByUserApp(user, app, "创建目录脚手架")
	if err != nil {
		return err
	}

	req := newSingleDirectoryScaffoldRuntimeReq(user, app, serviceTree)
	if _, err := b.appCall.BatchCreateDirectoryTree(ctx, appModel.HostID, req); err != nil {
		return fmt.Errorf("failed to create directory scaffold via app-runtime: %w", err)
	}
	return nil
}

func (b *runtimeWorkspaceBridge) deleteDirectoryScaffold(
	ctx context.Context,
	appID int64,
	packagePath string,
) (*model.App, *dto.DeleteServiceTreeRuntimeResp, error) {
	appModel, err := b.getRuntimeBoundAppByID(appID, "删除目录脚手架")
	if err != nil {
		return nil, nil, err
	}

	req := &dto.DeleteServiceTreeRuntimeReq{
		User:        appModel.User,
		App:         appModel.Code,
		PackagePath: packagePath,
	}
	resp, err := b.appCall.DeleteServiceTree(ctx, appModel.HostID, req)
	if err != nil {
		return appModel, nil, err
	}
	return appModel, resp, nil
}

func (b *runtimeWorkspaceBridge) batchCreateDirectoryTree(
	ctx context.Context,
	req *dto.BatchCreateDirectoryTreeReq,
) (*model.App, *dto.BatchCreateDirectoryTreeRuntimeResp, error) {
	appModel, err := b.getRuntimeBoundAppByUserApp(req.User, req.App, "批量创建目录树")
	if err != nil {
		return nil, nil, err
	}

	runtimeReq := &dto.BatchCreateDirectoryTreeRuntimeReq{
		User:  req.User,
		App:   req.App,
		Items: req.Items,
	}
	runtimeResp, err := b.appCall.BatchCreateDirectoryTree(ctx, appModel.HostID, runtimeReq)
	if err != nil {
		return nil, nil, fmt.Errorf("批量创建目录树失败: %w", err)
	}

	return appModel, runtimeResp, nil
}

func (b *runtimeWorkspaceBridge) batchWriteFiles(
	ctx context.Context,
	req *dto.BatchWriteFilesReq,
) (*model.App, *dto.BatchWriteFilesRuntimeResp, error) {
	operationLabel := strings.TrimSpace(req.OperationLabel)
	if operationLabel == "" {
		operationLabel = "批量写文件"
	}

	appModel, err := b.getRuntimeBoundAppByUserApp(req.User, req.App, operationLabel)
	if err != nil {
		return nil, nil, err
	}

	runtimeReq := &dto.BatchWriteFilesRuntimeReq{
		User:           req.User,
		App:            req.App,
		Files:          req.Files,
		ForceDiff:      req.ForceDiff,
		OperationName:  req.OperationName,
		OperationLabel: operationLabel,
	}
	runtimeResp, err := b.appCall.BatchWriteFiles(ctx, appModel.HostID, runtimeReq)
	if err != nil {
		return nil, nil, fmt.Errorf("%s失败: %w", operationLabel, err)
	}

	return appModel, runtimeResp, nil
}

func (b *runtimeWorkspaceBridge) readDirectoryFiles(
	ctx context.Context,
	appID int64,
	directoryPath string,
) (*model.App, *dto.ReadDirectoryFilesRuntimeResp, error) {
	appModel, err := b.getRuntimeBoundAppByID(appID, "读取目录文件")
	if err != nil {
		return nil, nil, err
	}
	resp, err := b.readDirectoryFilesFromApp(ctx, appModel, directoryPath)
	if err != nil {
		return appModel, nil, err
	}
	return appModel, resp, nil
}

func (b *runtimeWorkspaceBridge) readDirectoryFilesFromApp(
	ctx context.Context,
	appModel *model.App,
	directoryPath string,
) (*dto.ReadDirectoryFilesRuntimeResp, error) {
	runtimeReq := &dto.ReadDirectoryFilesRuntimeReq{
		User:          appModel.User,
		App:           appModel.Code,
		DirectoryPath: directoryPath,
	}
	return b.appCall.ReadDirectoryFiles(ctx, appModel.HostID, runtimeReq)
}

func (b *runtimeWorkspaceBridge) replaceInFileBatch(
	ctx context.Context,
	appModel *model.App,
	runtimeReq *dto.ReplaceInFileBatchReq,
) (*dto.ReplaceInFileBatchResp, error) {
	return b.appCall.ReplaceInFileBatch(ctx, appModel.HostID, runtimeReq)
}

func (b *runtimeWorkspaceBridge) deleteFile(
	ctx context.Context,
	appModel *model.App,
	runtimeReq *dto.DeleteFileRuntimeReq,
) (*dto.DeleteFileRuntimeResp, error) {
	return b.appCall.DeleteFile(ctx, appModel.HostID, runtimeReq)
}

func (b *runtimeWorkspaceBridge) readAppLog(
	ctx context.Context,
	appModel *model.App,
	runtimeReq *dto.ReadAppLogRuntimeReq,
) (*dto.ReadAppLogRuntimeResp, error) {
	return b.appCall.ReadAppLog(ctx, appModel.HostID, runtimeReq)
}
