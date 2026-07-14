package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/apperror"
	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/gorm"
)

// RequestApp 请求应用
func (a *AppService) RequestApp(ctx context.Context, req *dto.RequestAppReq) (*dto.RequestAppResp, error) {
	start := time.Now()
	app, err := a.appRepo.GetAppByUserNameContext(ctx, req.User, req.App)
	if err != nil {
		logger.Errorf(ctx, "[AppService:RequestApp] GetAppByUserName failed: user=%s, app=%s, traceId=%s, err=%v, elapsed=%s",
			req.User, req.App, req.TraceId, err, time.Since(start).Truncate(time.Millisecond))
		return nil, err
	}
	dbElapsed := time.Since(start)
	req.Version = app.Version
	a.applyRequestSourceContext(ctx, req)
	logger.Debugf(ctx, "[AppService:RequestApp] start: traceId=%s, %s/%s/%s, method=%s, router=%s, natsId=%d, dbElapsed=%s",
		req.TraceId, req.User, req.App, req.Version, req.Method, req.Router, app.NatsID, dbElapsed.Truncate(time.Millisecond))

	if err := a.requireFunctionConnectors(ctx, req); err != nil {
		logger.Warnf(ctx, "[AppService:RequestApp] connector dependency not ready: traceId=%s, user=%s, app=%s, router=%s, err=%v",
			req.TraceId, req.User, req.App, req.Router, err)
		return nil, err
	}

	resp, err := a.appCall.RequestApp(ctx, app.NatsID, req)
	totalElapsed := time.Since(start)
	if err != nil {
		logger.Errorf(ctx, "[AppService:RequestApp] appCall failed: traceId=%s, %s/%s/%s, err=%v, totalElapsed=%s",
			req.TraceId, req.User, req.App, req.Version, err, totalElapsed.Truncate(time.Millisecond))
		return nil, err
	}
	logger.Debugf(ctx, "[AppService:RequestApp] done: traceId=%s, %s/%s/%s, hasError=%v, totalElapsed=%s",
		req.TraceId, req.User, req.App, req.Version, resp.Error != "", totalElapsed.Truncate(time.Millisecond))
	resp.Version = req.Version
	return resp, nil
}

func (a *AppService) applyRequestSourceContext(ctx context.Context, req *dto.RequestAppReq) {
	if a == nil || a.serviceTreeRepo == nil || req == nil {
		return
	}
	if req.SourcePath == "" {
		req.SourcePath = requestAppFullCodePath(req)
	}
	if req.SourcePath == "" {
		return
	}
	if req.SourceTitle != "" && req.SourceParentTitle != "" && req.SourceTemplateType != "" {
		return
	}

	sourcePath := access.NormalizeResourcePath(req.SourcePath)
	parentPath := strings.TrimSpace(req.SourceParentPath)
	if parentPath == "" {
		parentPath = parentFullCodePath(sourcePath)
	}
	paths := []string{sourcePath}
	if parentPath != "" {
		paths = append(paths, parentPath)
	}
	nodes, err := a.serviceTreeRepo.GetServiceTreeByFullPaths(ctx, paths)
	if err != nil {
		logger.Warnf(ctx, "[AppService:RequestApp] resolve source display failed: source_path=%s err=%v", sourcePath, err)
		return
	}
	if source := nodes[sourcePath]; source != nil {
		if req.SourceTitle == "" {
			req.SourceTitle = source.Name
		}
		if req.SourceTemplateType == "" {
			req.SourceTemplateType = source.TemplateType
		}
	}
	if parentPath != "" {
		req.SourceParentPath = parentPath
		if parent := nodes[parentPath]; parent != nil {
			if req.SourceParentTitle == "" {
				req.SourceParentTitle = parent.Name
			}
		}
	}
}

func requestAppFullCodePath(req *dto.RequestAppReq) string {
	if req == nil {
		return ""
	}
	router := strings.Trim(strings.TrimSpace(req.Router), "/")
	if router == "" || strings.HasPrefix(router, "_") {
		return ""
	}
	return access.NormalizeResourcePath(fmt.Sprintf("/%s/%s/%s", strings.TrimSpace(req.User), strings.TrimSpace(req.App), router))
}

func parentFullCodePath(fullCodePath string) string {
	fullCodePath = strings.Trim(strings.TrimSpace(fullCodePath), "/")
	if fullCodePath == "" {
		return ""
	}
	parts := strings.Split(fullCodePath, "/")
	if len(parts) <= 2 {
		return ""
	}
	return "/" + strings.Join(parts[:len(parts)-1], "/")
}

func (a *AppService) requireFunctionConnectors(ctx context.Context, req *dto.RequestAppReq) error {
	if a == nil || a.functionRepo == nil {
		return nil
	}
	fullCodePath := requestFunctionFullCodePath(req)
	if fullCodePath == "" {
		return nil
	}
	function, err := a.functionRepo.GetFunctionByFullCodePath(ctx, fullCodePath)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("检查函数连接器依赖失败: %w", err)
	}
	connectors := splitConnectorCodes(function.Connectors)
	if len(connectors) == 0 {
		return nil
	}
	endpoints := splitConnectorEndpoints(function.ConnectorEndpoints)
	statuses := functionConnectorStatuses(ctx, fullCodePath, connectors, endpoints)
	missing := missingConnectorProviders(statuses)
	if len(missing) > 0 {
		if err := connectorDependencyError(statuses); err != nil {
			return err
		}
		return fmt.Errorf("函数依赖连接器 %s，请先完成连接或补充授权后再执行", strings.Join(missing, "、"))
	}
	return nil
}

// IncrementFunctionRunCount 将指定 full_code_path 的 function 运行次数 +1（成功执行 Form/Table/Chart 后调用，用于 search 按热度排序）
func (a *AppService) IncrementFunctionRunCount(ctx context.Context, fullCodePath string) {
	if fullCodePath == "" {
		return
	}
	if err := a.serviceTreeRepo.IncrementRunCountByFullCodePath(ctx, fullCodePath); err != nil {
		logger.Warnf(ctx, "[AppService] IncrementFunctionRunCount failed: full_code_path=%s, err=%v", fullCodePath, err)
	}
}

// GetFunctionByFullCodePath 根据 full-code-path 获取函数信息
// fullCodePath 参数应该是完整的路径（如 /luobei/operations/tools/pdftools/to_images）
// full-code-path 是全局唯一的，不需要 method 参数
func (a *AppService) GetFunctionByFullCodePath(ctx context.Context, fullCodePath string) (*model.Function, error) {
	function, err := a.functionRepo.GetFunctionByFullCodePath(ctx, fullCodePath)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("函数不存在", err)
		}
		return nil, apperror.Internal(fmt.Errorf("获取函数信息失败: %w", err))
	}

	return function, nil
}
