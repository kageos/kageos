package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/pkg/contextx"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
)

// RecordTableActionLog 记录 Table 操作日志（OnTableAddRow, OnTableUpdateRow, OnTableDeleteRows）。
func (a *AppService) RecordTableActionLog(ctx context.Context, req *dto.RecordTableActionLogReq) error {
	if req == nil {
		return nil
	}
	if req.TenantUser == "" || req.App == "" {
		return fmt.Errorf("记录 Table 操作日志失败: tenant_user 和 app 不能为空")
	}

	// 获取应用信息（用于获取版本号）
	app, err := a.appRepo.GetAppByUserNameContext(ctx, req.TenantUser, req.App)
	if err != nil {
		return fmt.Errorf("获取应用信息失败: %w", err)
	}
	version := strings.TrimSpace(req.Version)
	if version == "" {
		version = app.Version
	}

	resourceID := fmt.Sprintf("%s/%s/%s", req.TenantUser, req.App, strings.TrimPrefix(req.Router, "/"))
	writeCtx := context.WithoutCancel(ctx)

	// 根据操作类型处理不同的记录逻辑
	switch req.Action {
	case "OnTableAddRow":
		log := a.buildTableActionOperateLog(ctx, req, resourceID, req.RowID, req.Body, nil, version)
		go func(ctx context.Context, log *model.OperateLog) {
			if err := a.operateLogRepo.CreateOperateLog(ctx, log); err != nil {
				logger.Warnf(ctx, "[RecordTableActionLog] 记录 Table 新增操作日志失败: %v", err)
			}
		}(writeCtx, log)

	case "OnTableUpdateRow":
		// 更新操作：记录 updates 和 old_values
		log := a.buildTableActionOperateLog(ctx, req, resourceID, req.RowID, req.Updates, req.OldValues, version)
		go func(ctx context.Context, log *model.OperateLog) {
			if err := a.operateLogRepo.CreateOperateLog(ctx, log); err != nil {
				logger.Warnf(ctx, "[RecordTableActionLog] 记录 Table 更新操作日志失败: %v", err)
			}
		}(writeCtx, log)

	case "OnTableDeleteRows":
		// 删除操作：为每个删除的记录创建一条日志
		for _, rowID := range req.RowIDs {
			log := a.buildTableActionOperateLog(ctx, req, resourceID, rowID, nil, nil, version)
			go func(ctx context.Context, log *model.OperateLog) {
				if err := a.operateLogRepo.CreateOperateLog(ctx, log); err != nil {
					logger.Warnf(ctx, "[RecordTableActionLog] 记录 Table 删除操作日志失败: %v", err)
				}
			}(writeCtx, log)
		}
	}

	return nil
}

func (a *AppService) buildTableActionOperateLog(ctx context.Context, req *dto.RecordTableActionLogReq, resourceID string, rowID int64, updates, oldValues []byte, version string) *model.OperateLog {
	fullCodePath := buildTableActionLogFullCodePath(req, resourceID)
	tableFields := a.sensitiveFieldSet(fullCodePath, sensitiveFieldSectionRequest, sensitiveFieldSectionFields)
	responseFields := a.sensitiveFieldSet(fullCodePath, sensitiveFieldSectionResponse)
	auditMeta := buildOperateLogAuditMetadata(ctx, req.Source)
	details := dto.TableActionLogDetails{
		RowID:      rowID,
		Version:    version,
		SourceType: auditMeta.SourceType,
		SourceRef:  auditMeta.SourceRef,
	}
	if len(req.RowIDs) > 0 {
		details.RowIDs = req.RowIDs
	}
	if req.DurationMillis > 0 {
		details.DurationMillis = req.DurationMillis
	}
	if len(req.ResponseBody) > 0 {
		details.ResponseBody = rawMessageObjectWithFields(req.ResponseBody, responseFields)
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "success"
	}
	summary := strings.TrimSpace(req.Summary)
	if summary == "" {
		summary = buildTableActionLogSummary(req.RequestUser, req.Action, fullCodePath, rowID, status)
	}
	log := &model.OperateLog{
		TenantUser:    req.TenantUser,
		CompanyCode:   contextx.GetRequestCompanyCode(ctx),
		App:           req.App,
		ActorUser:     req.RequestUser,
		Action:        req.Action,
		ResourceType:  "table",
		ResourcePath:  fullCodePath,
		ResourceName:  strings.TrimPrefix(req.Router, "/"),
		TargetID:      fmt.Sprintf("%d", rowID),
		Summary:       summary,
		DetailsJSON:   mustMarshalRaw(details),
		OldValuesJSON: sanitizeOperateLogRawMessageWithFields(oldValues, tableFields),
		NewValuesJSON: sanitizeOperateLogRawMessageWithFields(updates, tableFields),
		Status:        status,
		IPAddress:     req.IPAddress,
		UserAgent:     req.UserAgent,
		TraceID:       req.TraceID,
	}
	applyOperateLogAuditMetadata(log, auditMeta)
	return log
}

func (a *AppService) sensitiveFieldSet(fullCodePath string, sections ...string) map[string]struct{} {
	if a == nil || a.sensitiveFields == nil {
		return nil
	}
	return a.sensitiveFields.SensitiveFieldSet(fullCodePath, sections...)
}

func buildTableActionLogSummary(requestUser, action, fullCodePath string, rowID int64, status string) string {
	if status == "failed" {
		switch action {
		case "OnTableAddRow":
			return fmt.Sprintf("%s failed to create row on %s", requestUser, fullCodePath)
		case "OnTableUpdateRow":
			return fmt.Sprintf("%s failed to update row #%d on %s", requestUser, rowID, fullCodePath)
		case "OnTableDeleteRows":
			return fmt.Sprintf("%s failed to delete row #%d on %s", requestUser, rowID, fullCodePath)
		default:
			return fmt.Sprintf("%s failed to execute %s on %s", requestUser, action, fullCodePath)
		}
	}
	switch action {
	case "OnTableAddRow":
		return fmt.Sprintf("%s created row on %s", requestUser, fullCodePath)
	case "OnTableUpdateRow":
		return fmt.Sprintf("%s updated row #%d on %s", requestUser, rowID, fullCodePath)
	case "OnTableDeleteRows":
		return fmt.Sprintf("%s deleted row #%d on %s", requestUser, rowID, fullCodePath)
	default:
		return fmt.Sprintf("%s executed %s on %s", requestUser, action, fullCodePath)
	}
}

func buildTableActionLogFullCodePath(req *dto.RecordTableActionLogReq, resourceID string) string {
	parts := strings.Split(resourceID, "/")
	fullCodePath := fmt.Sprintf("/%s/%s", req.TenantUser, req.App)
	if len(parts) > 2 {
		fullCodePath = fmt.Sprintf("/%s/%s/%s", req.TenantUser, req.App, strings.Join(parts[2:], "/"))
	}
	return fullCodePath
}
