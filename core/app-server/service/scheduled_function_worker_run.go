package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func (a *AppService) RunScheduledFunction(ctx context.Context, event scheduledsdk.ExecutionRequestedEvent) (*scheduledsdk.ExecutionResult, error) {
	payload, err := decodeScheduledFunctionPayload(event)
	if err != nil {
		return nil, err
	}
	logger.Infof(ctx, "[ScheduledFunctionWorker] start task_id=%d execution_id=%d full_code_path=%s action=%s",
		event.TaskID, event.ExecutionID, payload.FullCodePath, payload.Action)
	startedAt := time.Now()
	result, err := a.executeScheduledFunctionPayload(ctx, payload)
	durationMillis := time.Since(startedAt).Milliseconds()
	if !result.OperateLogRecorded {
		if logErr := a.recordScheduledFunctionOperateLog(ctx, payload, result, err, durationMillis); logErr != nil {
			logger.Warnf(ctx, "[ScheduledFunctionWorker] record scheduled function operate log failed: %v", logErr)
		}
	}
	executionResult := scheduledFunctionExecutionResult(result)
	if err != nil {
		return executionResult, err
	}
	if result.IsError {
		return executionResult, errors.New(result.Content)
	}
	logger.Infof(ctx, "[ScheduledFunctionWorker] done task_id=%d execution_id=%d full_code_path=%s",
		event.TaskID, event.ExecutionID, payload.FullCodePath)
	return executionResult, nil
}

func (a *AppService) recordScheduledFunctionOperateLog(ctx context.Context, payload scheduledFunctionPayload, result scheduledFunctionRunResult, runErr error, durationMillis int64) error {
	if a == nil || a.operateLogRepo == nil {
		return nil
	}
	user, app, router, err := parseScheduledFunctionFullCodePath(payload.FullCodePath)
	if err != nil {
		return err
	}
	status := "success"
	summary := fmt.Sprintf("定时任务执行函数成功: %s", payload.FullCodePath)
	responsePayload := dto.AppCallLogResponseBody{
		Code:          0,
		Result:        result.Data,
		TotalCostMill: durationMillis,
	}
	if runErr != nil || result.IsError {
		status = "failed"
		message := strings.TrimSpace(result.Content)
		if message == "" && runErr != nil {
			message = runErr.Error()
		}
		summary = fmt.Sprintf("定时任务执行函数失败: %s", message)
		responsePayload.Code = 1
		responsePayload.Message = message
		responsePayload.Error = message
	}
	responseBody, _ := json.Marshal(responsePayload)

	requestPayload := scheduledFunctionPayloadBodyBytes(payload.Payload)
	requestFields := a.sensitiveFieldSet(payload.FullCodePath, sensitiveFieldSectionRequest, sensitiveFieldSectionFields)
	responseFields := a.sensitiveFieldSet(payload.FullCodePath, sensitiveFieldSectionResponse)
	auditMeta := buildOperateLogAuditMetadata(ctx, "")
	details := dto.FunctionExecutionLogDetails{
		Router:          router,
		Method:          scheduledFunctionHTTPMethod(payload),
		TemplateType:    payload.TemplateType,
		ScheduledAction: payload.Action,
		DurationMillis:  durationMillis,
		SourceType:      auditMeta.SourceType,
		SourceRef:       auditMeta.SourceRef,
		RequestPayload:  rawMessageObjectWithFields(requestPayload, requestFields),
		ResponseBody:    rawMessageObjectWithFields(responseBody, responseFields),
	}
	actor := contextx.GetRequestUser(ctx)
	if actor == "" {
		actor = "system"
	}
	log := &model.OperateLog{
		TenantUser:    user,
		CompanyCode:   contextx.GetRequestCompanyCode(ctx),
		App:           app,
		ActorUser:     actor,
		Action:        "scheduled_function_execute",
		ResourceType:  "function",
		ResourcePath:  payload.FullCodePath,
		ResourceName:  router,
		Summary:       summary,
		DetailsJSON:   mustMarshalRaw(details),
		OldValuesJSON: sanitizeOperateLogRawMessageWithFields(requestPayload, requestFields),
		NewValuesJSON: sanitizeOperateLogRawMessageWithFields(responseBody, responseFields),
		Status:        status,
		TraceID:       contextx.GetTraceId(ctx),
	}
	applyOperateLogAuditMetadata(log, auditMeta)
	return a.operateLogRepo.CreateOperateLog(ctx, log)
}

func (a *AppService) executeScheduledFunctionPayload(ctx context.Context, payload scheduledFunctionPayload) (scheduledFunctionRunResult, error) {
	if err := a.requireScheduledFunctionAccess(ctx, payload.FullCodePath, payload.Action); err != nil {
		return scheduledFunctionErrorResult(err), err
	}

	switch payload.Action {
	case "table_create":
		return a.runScheduledTableCreate(ctx, payload)
	case "table_update":
		return a.runScheduledTableUpdate(ctx, payload)
	case "table_delete":
		return a.runScheduledTableDelete(ctx, payload)
	case "execute":
		switch payload.TemplateType {
		case "form":
			return a.runScheduledFormSubmit(ctx, payload)
		case "chart":
			return a.runScheduledReadFunction(ctx, payload)
		case "table":
			return a.runScheduledReadFunction(ctx, payload)
		default:
			err := fmt.Errorf("scheduled function template_type %q is not supported", payload.TemplateType)
			return scheduledFunctionErrorResult(err), err
		}
	default:
		err := fmt.Errorf("scheduled function action %q is not supported", payload.Action)
		return scheduledFunctionErrorResult(err), err
	}
}
