package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
)

// RecordFormOperateLog 记录 Form 提交操作日志，用于审计提交参数、响应结果和失败原因。
func (a *AppService) RecordFormOperateLog(ctx context.Context, req *dto.RecordFormOperateLogReq) error {
	if req == nil {
		return nil
	}
	if req.TenantUser == "" || req.App == "" {
		return fmt.Errorf("记录 Form 操作日志失败: tenant_user 和 app 不能为空")
	}

	version := strings.TrimSpace(req.Version)
	if version == "" {
		app, err := a.appRepo.GetAppByUserName(req.TenantUser, req.App)
		if err != nil {
			return fmt.Errorf("获取应用信息失败: %w", err)
		}
		version = app.Version
	}

	router := strings.TrimPrefix(strings.TrimSpace(req.Router), "/")
	resourcePath := fmt.Sprintf("/%s/%s", req.TenantUser, req.App)
	if router != "" {
		resourcePath = fmt.Sprintf("%s/%s", resourcePath, router)
	}
	requestFields := a.sensitiveFieldSet(resourcePath, sensitiveFieldSectionRequest)
	responseFields := a.sensitiveFieldSet(resourcePath, sensitiveFieldSectionResponse)
	auditMeta := buildOperateLogAuditMetadata(ctx, "")

	action := strings.TrimSpace(req.Action)
	if action == "" {
		action = "form_submit"
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "success"
	}
	summary := strings.TrimSpace(req.Summary)
	if summary == "" {
		if status == "success" {
			summary = "表单提交成功"
		} else {
			summary = "表单提交失败"
		}
	}

	details := dto.FormOperateLogDetails{
		Router:         router,
		Method:         req.FunctionMethod,
		Version:        version,
		DurationMillis: req.DurationMillis,
		SourceType:     auditMeta.SourceType,
		SourceRef:      auditMeta.SourceRef,
		RequestBody:    rawMessageObjectWithFields(req.RequestBody, requestFields),
		ResponseBody:   rawMessageObjectWithFields(req.ResponseBody, responseFields),
	}

	log := &model.OperateLog{
		TenantUser:    req.TenantUser,
		CompanyCode:   contextx.GetRequestCompanyCode(ctx),
		App:           req.App,
		ActorUser:     req.RequestUser,
		Action:        action,
		ResourceType:  "form",
		ResourcePath:  resourcePath,
		ResourceName:  router,
		Summary:       summary,
		DetailsJSON:   mustMarshalRaw(details),
		OldValuesJSON: sanitizeOperateLogRawMessageWithFields(req.RequestBody, requestFields),
		NewValuesJSON: sanitizeOperateLogRawMessageWithFields(req.ResponseBody, responseFields),
		Status:        status,
		IPAddress:     req.IPAddress,
		UserAgent:     req.UserAgent,
		TraceID:       req.TraceID,
	}
	applyOperateLogAuditMetadata(log, auditMeta)

	writeCtx := context.WithoutCancel(ctx)
	go func(ctx context.Context, log *model.OperateLog) {
		if err := a.operateLogRepo.CreateOperateLog(ctx, log); err != nil {
			logger.Warnf(ctx, "[RecordFormOperateLog] 记录 Form 操作日志失败: %v", err)
		}
	}(writeCtx, log)

	return nil
}

func normalizeRawMessage(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return sanitizeOperateLogRawMessage(raw)
}

func rawMessageObject(raw json.RawMessage) interface{} {
	return rawMessageObjectWithFields(raw, nil)
}

func rawMessageObjectWithFields(raw json.RawMessage, sensitiveFields map[string]struct{}) interface{} {
	if len(raw) == 0 {
		return nil
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return redactOperateLogValueWithFields(v, sensitiveFields)
}
