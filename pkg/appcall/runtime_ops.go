package appcall

import (
	"context"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/subjects"
)

// CreateApp 创建应用（subject: runtime.v1.cmd.app.create）
func (c *Client) CreateApp(ctx context.Context, hostID int64, req *dto.CreateAppReq) (*dto.CreateAppResp, error) {
	var resp dto.CreateAppResp
	if err := c.runtime.requestByHost(ctx, hostID, subjects.RuntimeAppCreateCommandSubject, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateApp 更新应用（subject: runtime.v1.cmd.app.update）
func (c *Client) UpdateApp(ctx context.Context, hostID int64, req *dto.UpdateAppRuntimeReq) (*dto.UpdateAppResp, error) {
	var resp dto.UpdateAppResp
	if err := c.runtime.requestByHost(ctx, hostID, subjects.RuntimeAppUpdateCommandSubject, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteApp 删除应用（subject: runtime.v1.cmd.app.delete）
func (c *Client) DeleteApp(ctx context.Context, hostID int64, req *dto.DeleteAppRuntimeReq) (*dto.DeleteAppResp, error) {
	var resp dto.DeleteAppResp
	if err := c.runtime.requestByHost(ctx, hostID, subjects.RuntimeAppDeleteCommandSubject, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteServiceTree 删除服务目录（subject: runtime.v1.cmd.service-tree.delete，删磁盘并从 main.go 移除 import）
func (c *Client) DeleteServiceTree(ctx context.Context, hostID int64, req *dto.DeleteServiceTreeRuntimeReq) (*dto.DeleteServiceTreeRuntimeResp, error) {
	var resp dto.DeleteServiceTreeRuntimeResp
	if err := c.runtime.requestByHost(ctx, hostID, subjects.RuntimeServiceTreeDeleteCommandSubject, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ReadDirectoryFiles 读取目录文件（subject: runtime.v1.query.directory-files.read）
func (c *Client) ReadDirectoryFiles(ctx context.Context, hostID int64, req *dto.ReadDirectoryFilesRuntimeReq) (*dto.ReadDirectoryFilesRuntimeResp, error) {
	var resp dto.ReadDirectoryFilesRuntimeResp
	if err := c.runtime.requestByHost(ctx, hostID, subjects.RuntimeDirectoryFilesReadQuerySubject, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ReplaceInFileBatch 文件批量 search-replace（subject: runtime.v1.cmd.file.replace-batch）
func (c *Client) ReplaceInFileBatch(ctx context.Context, hostID int64, req *dto.ReplaceInFileBatchReq) (*dto.ReplaceInFileBatchResp, error) {
	var resp dto.ReplaceInFileBatchResp
	if err := c.runtime.requestByHost(ctx, hostID, subjects.RuntimeFileReplaceBatchCommandSubject, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteFile 删除磁盘文件（subject: runtime.v1.cmd.file.delete）
func (c *Client) DeleteFile(ctx context.Context, hostID int64, req *dto.DeleteFileRuntimeReq) (*dto.DeleteFileRuntimeResp, error) {
	var resp dto.DeleteFileRuntimeResp
	if err := c.runtime.requestByHost(ctx, hostID, subjects.RuntimeFileDeleteCommandSubject, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ReadAppLog 读取应用日志（subject: runtime.v1.query.app-log.read）
func (c *Client) ReadAppLog(ctx context.Context, hostID int64, req *dto.ReadAppLogRuntimeReq) (*dto.ReadAppLogRuntimeResp, error) {
	var resp dto.ReadAppLogRuntimeResp
	if err := c.runtime.requestByHost(ctx, hostID, subjects.RuntimeAppLogReadQuerySubject, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// BatchCreateDirectoryTree 批量创建目录树（subject: runtime.v1.cmd.directory-tree.batch-create）
func (c *Client) BatchCreateDirectoryTree(ctx context.Context, hostID int64, req *dto.BatchCreateDirectoryTreeRuntimeReq) (*dto.BatchCreateDirectoryTreeRuntimeResp, error) {
	var resp dto.BatchCreateDirectoryTreeRuntimeResp
	if err := c.runtime.requestByHost(ctx, hostID, subjects.RuntimeDirectoryTreeBatchCreateCommandSubject, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// BatchWriteFiles 批量写文件（subject: runtime.v1.cmd.file.batch-write）
func (c *Client) BatchWriteFiles(ctx context.Context, hostID int64, req *dto.BatchWriteFilesRuntimeReq) (*dto.BatchWriteFilesRuntimeResp, error) {
	var resp dto.BatchWriteFilesRuntimeResp
	if err := c.runtime.requestByHost(ctx, hostID, subjects.RuntimeFileBatchWriteCommandSubject, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ReplaceDirectoryTree 完全替换同名目录（subject: runtime.v1.cmd.directory-tree.replace）
func (c *Client) ReplaceDirectoryTree(ctx context.Context, hostID int64, req *dto.ReplaceDirectoryTreeRuntimeReq) (*dto.ReplaceDirectoryTreeRuntimeResp, error) {
	var resp dto.ReplaceDirectoryTreeRuntimeResp
	if err := c.runtime.requestByHost(ctx, hostID, subjects.RuntimeDirectoryTreeReplaceCommandSubject, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
