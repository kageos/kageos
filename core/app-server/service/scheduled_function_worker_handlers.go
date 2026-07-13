package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
)

func (a *AppService) runScheduledFormSubmit(ctx context.Context, payload scheduledFunctionPayload) (scheduledFunctionRunResult, error) {
	body := scheduledFunctionPayloadBodyBytes(payload.Payload)
	req, err := buildScheduledRequestAppReq(ctx, payload.FullCodePath, http.MethodPost, body, "")
	if err != nil {
		return scheduledFunctionErrorResult(err), err
	}

	now := time.Now()
	resp, err := a.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	formLogReq := &dto.RecordFormOperateLogReq{
		TenantUser:     req.User,
		RequestUser:    req.RequestUser,
		App:            req.App,
		Router:         req.Router,
		Action:         "form_submit",
		FunctionMethod: req.Method,
		RequestBody:    req.Body,
		ResponseBody:   scheduledFormOperateLogResponseBody(resp, err, mill),
		TraceID:        req.TraceId,
		DurationMillis: mill,
		Status:         "success",
		Summary:        "表单提交成功",
	}
	if resp != nil {
		formLogReq.Version = resp.Version
	}
	if err != nil || (resp != nil && resp.Error != "") {
		formLogReq.Status = "failed"
		formLogReq.Summary = "表单提交失败"
	}
	operateLogRecorded := true
	if logErr := a.RecordFormOperateLog(ctx, formLogReq); logErr != nil {
		operateLogRecorded = false
		logger.Warnf(ctx, "[ScheduledFunctionWorker] record scheduled form operate log failed: %v", logErr)
	}

	result := scheduledFunctionResultFromResponse(resp, err)
	result.OperateLogRecorded = operateLogRecorded
	if err == nil && !result.IsError {
		a.IncrementFunctionRunCount(ctx, payload.FullCodePath)
	}
	return result, err
}

func (a *AppService) runScheduledReadFunction(ctx context.Context, payload scheduledFunctionPayload) (scheduledFunctionRunResult, error) {
	query, err := scheduledFunctionPayloadURLQuery(payload.Payload)
	if err != nil {
		return scheduledFunctionErrorResult(err), err
	}
	req, err := buildScheduledRequestAppReq(ctx, payload.FullCodePath, http.MethodGet, nil, query)
	if err != nil {
		return scheduledFunctionErrorResult(err), err
	}
	resp, err := a.RequestApp(ctx, req)
	result := scheduledFunctionResultFromResponse(resp, err)
	if err == nil && !result.IsError {
		a.IncrementFunctionRunCount(ctx, payload.FullCodePath)
	}
	return result, err
}

func (a *AppService) runScheduledTableCreate(ctx context.Context, payload scheduledFunctionPayload) (scheduledFunctionRunResult, error) {
	if err := a.ensureScheduledTableCallbackEnabled(ctx, payload.FullCodePath, "OnTableAddRow", "该表未开启新增能力，通常是只读查询表，不支持新增"); err != nil {
		return scheduledFunctionErrorResult(err), err
	}

	body := scheduledFunctionPayloadBodyBytes(payload.Payload)
	req, err := buildScheduledCallbackAppReq(ctx, payload.FullCodePath, http.MethodPost, "OnTableAddRow", body, "")
	if err != nil {
		return scheduledFunctionErrorResult(err), err
	}
	user, app, router, _ := parseScheduledFunctionFullCodePath(payload.FullCodePath)
	logReq := &dto.RecordTableActionLogReq{
		TenantUser:  user,
		RequestUser: req.RequestUser,
		App:         app,
		Router:      router,
		Action:      "OnTableAddRow",
		Source:      contextx.GetAuditClientSource(ctx),
		Body:        body,
		TraceID:     req.TraceId,
	}

	now := time.Now()
	resp, err := a.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()
	fillScheduledTableActionLogResult(logReq, resp, err, mill)
	operateLogRecorded := true
	if logErr := a.RecordTableActionLog(ctx, logReq); logErr != nil {
		operateLogRecorded = false
		logger.Warnf(ctx, "[ScheduledFunctionWorker] record scheduled table create log failed: %v", logErr)
	}

	result := scheduledFunctionResultFromResponse(resp, err)
	result.OperateLogRecorded = operateLogRecorded
	if err == nil && !result.IsError {
		a.IncrementFunctionRunCount(ctx, payload.FullCodePath)
	}
	return result, err
}

func (a *AppService) runScheduledTableUpdate(ctx context.Context, payload scheduledFunctionPayload) (scheduledFunctionRunResult, error) {
	if err := a.ensureScheduledTableCallbackEnabled(ctx, payload.FullCodePath, "OnTableUpdateRow", "该表未开启编辑能力，通常是只读查询表，不支持更新"); err != nil {
		return scheduledFunctionErrorResult(err), err
	}

	body, bodyData, err := a.prepareScheduledTableUpdateBody(ctx, payload.FullCodePath, payload.Payload)
	if err != nil {
		return scheduledFunctionErrorResult(err), err
	}
	req, err := buildScheduledCallbackAppReq(ctx, payload.FullCodePath, http.MethodPut, "OnTableUpdateRow", body, "")
	if err != nil {
		return scheduledFunctionErrorResult(err), err
	}

	user, app, router, _ := parseScheduledFunctionFullCodePath(payload.FullCodePath)
	logReq := &dto.RecordTableActionLogReq{
		TenantUser:  user,
		RequestUser: req.RequestUser,
		App:         app,
		Router:      router,
		Action:      "OnTableUpdateRow",
		Source:      contextx.GetAuditClientSource(ctx),
		TraceID:     req.TraceId,
	}
	if id, ok := getScheduledBodyIDInt64(bodyData); ok {
		logReq.RowID = id
	}
	if updatesData, ok := bodyData["updates"].(map[string]interface{}); ok {
		logReq.Updates, _ = json.Marshal(updatesData)
	}
	if oldValuesData, ok := bodyData["old_values"].(map[string]interface{}); ok {
		logReq.OldValues, _ = json.Marshal(oldValuesData)
	}

	now := time.Now()
	resp, err := a.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()
	fillScheduledTableActionLogResult(logReq, resp, err, mill)
	operateLogRecorded := true
	if logErr := a.RecordTableActionLog(ctx, logReq); logErr != nil {
		operateLogRecorded = false
		logger.Warnf(ctx, "[ScheduledFunctionWorker] record scheduled table update log failed: %v", logErr)
	}

	result := scheduledFunctionResultFromResponse(resp, err)
	result.OperateLogRecorded = operateLogRecorded
	if err == nil && !result.IsError {
		a.IncrementFunctionRunCount(ctx, payload.FullCodePath)
	}
	return result, err
}

func (a *AppService) runScheduledTableDelete(ctx context.Context, payload scheduledFunctionPayload) (scheduledFunctionRunResult, error) {
	if err := a.ensureScheduledTableCallbackEnabled(ctx, payload.FullCodePath, "OnTableDeleteRows", "该表未开启删除能力，不支持删除"); err != nil {
		return scheduledFunctionErrorResult(err), err
	}

	body := scheduledFunctionPayloadBodyBytes(payload.Payload)
	req, err := buildScheduledCallbackAppReq(ctx, payload.FullCodePath, http.MethodDelete, "OnTableDeleteRows", body, "")
	if err != nil {
		return scheduledFunctionErrorResult(err), err
	}

	user, app, router, _ := parseScheduledFunctionFullCodePath(payload.FullCodePath)
	logReq := &dto.RecordTableActionLogReq{
		TenantUser:  user,
		RequestUser: req.RequestUser,
		App:         app,
		Router:      router,
		Action:      "OnTableDeleteRows",
		Source:      contextx.GetAuditClientSource(ctx),
		TraceID:     req.TraceId,
	}
	var bodyData map[string]interface{}
	if err := json.Unmarshal(body, &bodyData); err == nil {
		logReq.RowIDs = scheduledRowIDsFromDeleteBody(bodyData)
	}

	now := time.Now()
	resp, err := a.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()
	fillScheduledTableActionLogResult(logReq, resp, err, mill)
	operateLogRecorded := true
	if logErr := a.RecordTableActionLog(ctx, logReq); logErr != nil {
		operateLogRecorded = false
		logger.Warnf(ctx, "[ScheduledFunctionWorker] record scheduled table delete log failed: %v", logErr)
	}

	result := scheduledFunctionResultFromResponse(resp, err)
	result.OperateLogRecorded = operateLogRecorded
	if err == nil && !result.IsError {
		a.IncrementFunctionRunCount(ctx, payload.FullCodePath)
	}
	return result, err
}

func (a *AppService) prepareScheduledTableUpdateBody(ctx context.Context, fullCodePath string, raw json.RawMessage) ([]byte, map[string]interface{}, error) {
	body := scheduledFunctionPayloadBodyBytes(raw)
	var bodyData map[string]interface{}
	if err := json.Unmarshal(body, &bodyData); err != nil {
		return nil, nil, fmt.Errorf("请求体必须是合法 JSON")
	}
	if !needFillScheduledOldValues(bodyData) {
		return body, bodyData, nil
	}

	id, ok := getScheduledBodyIDInt64(bodyData)
	if !ok {
		return nil, nil, fmt.Errorf("请求体缺少有效 id，无法自动填充 old_values")
	}
	rows, err := a.fetchScheduledTableRowsByIDs(ctx, fullCodePath, []int64{id})
	if err != nil {
		return nil, nil, fmt.Errorf("自动查询当前行失败: %w", err)
	}
	oldRow := findScheduledTableRowByID(rows, id)
	if oldRow == nil {
		return nil, nil, fmt.Errorf("记录不存在（id 未查到数据），无法填充 old_values")
	}
	bodyData["old_values"] = oldRow
	newBody, err := json.Marshal(bodyData)
	if err != nil {
		return nil, nil, fmt.Errorf("构造 old_values 后序列化失败: %w", err)
	}
	return newBody, bodyData, nil
}

func (a *AppService) fetchScheduledTableRowsByIDs(ctx context.Context, fullCodePath string, ids []int64) ([]map[string]interface{}, error) {
	body, err := json.Marshal(scheduledTableGetRowsCallbackReq{IDs: ids})
	if err != nil {
		return nil, fmt.Errorf("构造 %s 请求失败: %w", internalTableGetRowsCallback, err)
	}
	req, err := buildScheduledCallbackAppReq(ctx, fullCodePath, tableGetRowsCallbackHTTPMethod, internalTableGetRowsCallback, body, "")
	if err != nil {
		return nil, err
	}
	resp, err := a.RequestApp(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("%s 返回空响应", internalTableGetRowsCallback)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("%s 失败: %s", internalTableGetRowsCallback, resp.Error)
	}
	return extractScheduledTableGetRowsCallbackRows(resp.Result)
}
