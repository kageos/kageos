package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/functionschema"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/nats-io/nats.go"
)

const ScheduledFunctionExecutorKey = "app.function"

type scheduledFunctionPayload struct {
	FullCodePath string          `json:"full_code_path"`
	TemplateType string          `json:"template_type,omitempty"`
	Action       string          `json:"action,omitempty"`
	Method       string          `json:"method,omitempty"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	Body         json.RawMessage `json:"body,omitempty"`
}

type scheduledFunctionRunResult struct {
	Content            string
	Data               interface{}
	IsError            bool
	OperateLogRecorded bool
}

type scheduledCallbackRequestEnvelope struct {
	Method string `json:"method"`
	Router string `json:"router"`
	Body   []byte `json:"body"`
	Type   string `json:"type"`
}

func NewScheduledFunctionWorker(natsConn *nats.Conn, appService *AppService) (*scheduledsdk.Worker, error) {
	if natsConn == nil {
		return nil, fmt.Errorf("scheduled function worker requires nats connection")
	}
	if appService == nil {
		return nil, fmt.Errorf("scheduled function worker requires app service")
	}
	client := scheduledsdk.NewClient(scheduledsdk.Options{
		Adapter: scheduledsdk.NewNATSAdapter(natsConn, scheduledsdk.NATSAdapterOptions{}),
	})
	return scheduledsdk.NewWorker(scheduledsdk.WorkerOptions{
		Client:      client,
		NATSConn:    natsConn,
		ExecutorKey: ScheduledFunctionExecutorKey,
		Handler:     appService.RunScheduledFunction,
		OnError: func(ctx context.Context, err error) {
			logger.Warnf(ctx, "[ScheduledFunctionWorker] %v", err)
		},
	})
}

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
	details := dto.FunctionExecutionLogDetails{
		Router:          router,
		Method:          scheduledFunctionHTTPMethod(payload),
		TemplateType:    payload.TemplateType,
		ScheduledAction: payload.Action,
		DurationMillis:  durationMillis,
		SourceType:      contextx.GetSourceType(ctx),
		SourceRef:       contextx.GetSourceRef(ctx),
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
		Source:        contextx.GetAuditClientSource(ctx),
		TraceID:       contextx.GetTraceId(ctx),
	}
	return a.operateLogRepo.CreateOperateLog(ctx, log)
}

func decodeScheduledFunctionPayload(event scheduledsdk.ExecutionRequestedEvent) (scheduledFunctionPayload, error) {
	var payload scheduledFunctionPayload
	if len(event.ExecutorPayload) == 0 {
		return payload, fmt.Errorf("scheduled function executor_payload is empty")
	}
	if err := json.Unmarshal(event.ExecutorPayload, &payload); err != nil {
		return payload, fmt.Errorf("decode scheduled function payload: %w", err)
	}
	payload.FullCodePath = access.NormalizeResourcePath(payload.FullCodePath)
	payload.TemplateType = strings.TrimSpace(payload.TemplateType)
	payload.Action = strings.TrimSpace(payload.Action)
	if payload.Action == "" {
		payload.Action = "execute"
	}
	payload.Method = strings.TrimSpace(payload.Method)
	if len(payload.Payload) == 0 && len(payload.Body) > 0 {
		payload.Payload = payload.Body
	}
	if len(payload.Payload) == 0 {
		payload.Payload = json.RawMessage(`{}`)
	}
	if payload.FullCodePath == "" {
		return payload, fmt.Errorf("scheduled function payload requires full_code_path")
	}
	if payload.TemplateType == "" {
		payload.TemplateType = scheduledFunctionTemplateType(payload.FullCodePath)
	}
	if !isScheduledFunctionAction(payload.Action) {
		return payload, fmt.Errorf("scheduled function action %q is not supported", payload.Action)
	}
	return payload, nil
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

	idStr, ok := getScheduledBodyID(bodyData)
	if !ok || idStr == "" {
		return nil, nil, fmt.Errorf("请求体缺少有效 id，无法自动填充 old_values")
	}
	user, app, router, err := parseScheduledFunctionFullCodePath(fullCodePath)
	if err != nil {
		return nil, nil, err
	}
	searchReq := buildScheduledBaseRequestAppReq(ctx, user, app, router, http.MethodGet)
	searchReq.UrlQuery = "eq=id:" + url.QueryEscape(idStr) + "&page=1&page_size=1"
	searchResp, err := a.RequestApp(ctx, searchReq)
	if err != nil {
		return nil, nil, fmt.Errorf("自动查询当前行失败: %w", err)
	}
	if searchResp == nil {
		return nil, nil, fmt.Errorf("自动查询当前行失败: app-runtime 返回空响应")
	}
	if searchResp.Error != "" {
		return nil, nil, fmt.Errorf("自动查询当前行失败: %s", searchResp.Error)
	}
	oldRow := extractScheduledFirstItemFromSearchResult(searchResp.Result)
	if oldRow == nil {
		return nil, nil, fmt.Errorf("记录不存在（eq=id 未查到数据），无法填充 old_values")
	}
	bodyData["old_values"] = oldRow
	newBody, err := json.Marshal(bodyData)
	if err != nil {
		return nil, nil, fmt.Errorf("构造 old_values 后序列化失败: %w", err)
	}
	return newBody, bodyData, nil
}

func (a *AppService) requireScheduledFunctionAccess(ctx context.Context, fullCodePath, action string) error {
	if a == nil || a.teamAccess == nil {
		return nil
	}
	resourcePath := access.NormalizeResourcePath(fullCodePath)
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		return err
	}
	return a.teamAccess.Check(ctx, tenantUser, app, contextx.GetRequestUser(ctx), resourcePath, scheduledFunctionRequiredAccessAction(fullCodePath, action))
}

func (a *AppService) ensureScheduledTableCallbackEnabled(ctx context.Context, fullCodePath, callbackType, denyMessage string) error {
	function, err := a.GetFunctionByFullCodePath(ctx, fullCodePath)
	if err != nil {
		return err
	}
	if functionschema.Type(function.Schema) != functionschema.TypeTable {
		return fmt.Errorf("目标函数不是 Table 类型，不支持该操作")
	}
	if !function.HasCallback(callbackType) {
		return fmt.Errorf("%s", denyMessage)
	}
	return nil
}

func buildScheduledRequestAppReq(ctx context.Context, fullCodePath, method string, body []byte, rawQuery string) (*dto.RequestAppReq, error) {
	user, app, router, err := parseScheduledFunctionFullCodePath(fullCodePath)
	if err != nil {
		return nil, err
	}
	req := buildScheduledBaseRequestAppReq(ctx, user, app, router, method)
	req.Body = body
	req.UrlQuery = rawQuery
	return req, nil
}

func buildScheduledCallbackAppReq(ctx context.Context, fullCodePath, method, callbackType string, body []byte, rawQuery string) (*dto.RequestAppReq, error) {
	user, app, router, err := parseScheduledFunctionFullCodePath(fullCodePath)
	if err != nil {
		return nil, err
	}
	req := buildScheduledBaseRequestAppReq(ctx, user, app, "/_callback", method)
	req.UrlQuery = rawQuery
	envelope := scheduledCallbackRequestEnvelope{
		Method: method,
		Router: router,
		Body:   body,
		Type:   callbackType,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	req.Body = data
	return req, nil
}

func buildScheduledBaseRequestAppReq(ctx context.Context, user, app, router, method string) *dto.RequestAppReq {
	return &dto.RequestAppReq{
		User:                  user,
		App:                   app,
		Router:                router,
		Method:                method,
		TraceId:               contextx.GetTraceId(ctx),
		RequestUser:           contextx.GetRequestUser(ctx),
		RequestUserDept:       contextx.GetRequestDepartmentFullPath(ctx),
		Token:                 contextx.GetToken(ctx),
		ClientSource:          contextx.GetClientSource(ctx),
		SourceType:            contextx.GetSourceType(ctx),
		SourceRef:             contextx.GetSourceRef(ctx),
		SourcePath:            contextx.GetSourcePath(ctx),
		SourceTitle:           contextx.GetSourceTitle(ctx),
		SourceParentPath:      contextx.GetSourceParentPath(ctx),
		SourceParentTitle:     contextx.GetSourceParentTitle(ctx),
		SourceTemplateType:    contextx.GetSourceTemplateType(ctx),
		SourceIcon:            contextx.GetSourceIcon(ctx),
		SourceColor:           contextx.GetSourceColor(ctx),
		SourceParentIcon:      contextx.GetSourceParentIcon(ctx),
		SourceParentColor:     contextx.GetSourceParentColor(ctx),
		WorkspaceSessionID:    contextx.GetWorkspaceSessionID(ctx),
		WorkspaceSessionTitle: contextx.GetWorkspaceSessionTitle(ctx),
		WorkspaceRole:         contextx.GetWorkspaceRole(ctx),
	}
}

func parseScheduledFunctionFullCodePath(fullCodePath string) (user, app, router string, err error) {
	fullCodePath = strings.TrimPrefix(access.NormalizeResourcePath(fullCodePath), "/")
	parts := strings.Split(fullCodePath, "/")
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("full-code-path 格式错误，至少需要包含 user/app/function")
	}
	return parts[0], parts[1], strings.Join(parts[2:], "/"), nil
}

func isScheduledFunctionAction(action string) bool {
	switch strings.TrimSpace(action) {
	case "execute", "table_create", "table_update", "table_delete":
		return true
	default:
		return false
	}
}

func scheduledFunctionRequiredAccessAction(fullCodePath, action string) access.Action {
	switch strings.TrimSpace(action) {
	case "table_update":
		return access.ActionUpdate
	case "table_delete":
		return access.ActionDelete
	case "execute":
		switch scheduledFunctionTemplateType(fullCodePath) {
		case "chart", "table":
			return access.ActionRead
		default:
			return access.ActionWrite
		}
	default:
		return access.ActionWrite
	}
}

func scheduledFunctionHTTPMethod(payload scheduledFunctionPayload) string {
	if method := strings.TrimSpace(payload.Method); method != "" {
		return strings.ToUpper(method)
	}
	switch strings.TrimSpace(payload.Action) {
	case "table_update":
		return http.MethodPut
	case "table_delete":
		return http.MethodDelete
	case "execute":
		switch payload.TemplateType {
		case "table", "chart":
			return http.MethodGet
		default:
			return http.MethodPost
		}
	default:
		return http.MethodPost
	}
}

func scheduledFunctionTemplateType(fullCodePath string) string {
	path := strings.ToLower(strings.TrimSpace(fullCodePath))
	switch {
	case strings.Contains(path, ".form"):
		return "form"
	case strings.Contains(path, ".table"):
		return "table"
	case strings.Contains(path, ".chart"):
		return "chart"
	default:
		return ""
	}
}

func scheduledFunctionPayloadBodyBytes(raw json.RawMessage) []byte {
	raw = compactScheduledRawJSON(raw)
	if len(raw) == 0 {
		return []byte("{}")
	}
	return []byte(raw)
}

func scheduledFunctionPayloadURLQuery(raw json.RawMessage) (string, error) {
	raw = compactScheduledRawJSON(raw)
	if len(raw) == 0 || string(raw) == "{}" {
		return "", nil
	}
	var queryString string
	if err := json.Unmarshal(raw, &queryString); err == nil {
		return strings.TrimPrefix(strings.TrimSpace(queryString), "?"), nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("scheduled function query payload must be JSON object or query string: %w", err)
	}
	values := url.Values{}
	for key, value := range payload {
		key = strings.TrimSpace(key)
		if key == "" || value == nil {
			continue
		}
		switch typed := value.(type) {
		case []interface{}:
			for _, item := range typed {
				if item != nil {
					values.Add(key, fmt.Sprint(item))
				}
			}
		default:
			values.Set(key, fmt.Sprint(typed))
		}
	}
	return values.Encode(), nil
}

func compactScheduledRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var out interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return raw
	}
	if out == nil {
		return nil
	}
	data, err := json.Marshal(out)
	if err != nil {
		return raw
	}
	return data
}

func needFillScheduledOldValues(bodyData map[string]interface{}) bool {
	if bodyData == nil {
		return false
	}
	if _, hasID := bodyData["id"]; !hasID {
		return false
	}
	oldValues, ok := bodyData["old_values"].(map[string]interface{})
	return !ok || len(oldValues) == 0
}

func getScheduledBodyID(bodyData map[string]interface{}) (string, bool) {
	id, ok := getScheduledBodyIDInt64(bodyData)
	return strconv.FormatInt(id, 10), ok
}

func getScheduledBodyIDInt64(bodyData map[string]interface{}) (int64, bool) {
	if bodyData == nil {
		return 0, false
	}
	switch v := bodyData["id"].(type) {
	case float64:
		return int64(v), true
	case int:
		return int64(v), true
	case int64:
		return v, true
	default:
		return 0, false
	}
}

func extractScheduledFirstItemFromSearchResult(result interface{}) map[string]interface{} {
	if result == nil {
		return nil
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		return nil
	}
	items, ok := m["items"].([]interface{})
	if !ok || len(items) == 0 {
		return nil
	}
	first, ok := items[0].(map[string]interface{})
	if !ok {
		return nil
	}
	return first
}

func scheduledRowIDsFromDeleteBody(bodyData map[string]interface{}) []int64 {
	ids, ok := bodyData["ids"].([]interface{})
	if !ok {
		return nil
	}
	rowIDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		switch v := id.(type) {
		case float64:
			rowIDs = append(rowIDs, int64(v))
		case int:
			rowIDs = append(rowIDs, int64(v))
		case int64:
			rowIDs = append(rowIDs, v)
		case string:
			parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			if err == nil {
				rowIDs = append(rowIDs, parsed)
			}
		}
	}
	return rowIDs
}

func scheduledFunctionResultFromResponse(resp *dto.RequestAppResp, err error) scheduledFunctionRunResult {
	if err != nil {
		return scheduledFunctionErrorResult(err)
	}
	if resp == nil {
		return scheduledFunctionRunResult{Content: "函数执行完成", Data: nil}
	}
	if resp.Error != "" {
		return scheduledFunctionRunResult{Content: resp.Error, Data: resp.Result, IsError: true}
	}
	content := "函数执行完成"
	if resp.Result != nil {
		if data, marshalErr := json.Marshal(resp.Result); marshalErr == nil && len(data) > 0 {
			content = string(data)
		}
	}
	return scheduledFunctionRunResult{Content: content, Data: resp.Result}
}

func scheduledFunctionErrorResult(err error) scheduledFunctionRunResult {
	if err == nil {
		return scheduledFunctionRunResult{}
	}
	return scheduledFunctionRunResult{Content: err.Error(), IsError: true}
}

func scheduledFunctionExecutionResult(result scheduledFunctionRunResult) *scheduledsdk.ExecutionResult {
	resultPayload, _ := json.Marshal(map[string]interface{}{
		"is_error": result.IsError,
		"data":     result.Data,
	})
	return &scheduledsdk.ExecutionResult{
		OutputSummary: compactScheduledFunctionSummary(result.Content),
		ResultPayload: resultPayload,
	}
}

func compactScheduledFunctionSummary(content string) string {
	content = strings.Join(strings.Fields(content), " ")
	const max = 240
	if len(content) <= max {
		return content
	}
	return content[:max] + "..."
}

func scheduledFormOperateLogResponseBody(resp *dto.RequestAppResp, err error, totalCostMill int64) json.RawMessage {
	payload := dto.AppCallLogResponseBody{
		Code:          0,
		TotalCostMill: totalCostMill,
	}
	switch {
	case resp != nil:
		payload.Code = resp.ErrCode
		payload.ErrCode = resp.ErrCode
		payload.TraceID = resp.TraceId
		payload.Version = resp.Version
		payload.Result = resp.Result
		if resp.Error != "" {
			payload.Message = resp.Error
			payload.Error = resp.Error
		}
	case err != nil:
		payload.Code = 1
		payload.Message = err.Error()
		payload.Error = err.Error()
	}
	data, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		fallback := dto.AppCallLogResponseBody{
			Code:          1,
			Message:       "marshal form operate log response failed",
			TotalCostMill: totalCostMill,
		}
		if err != nil {
			fallback.Message = err.Error()
			fallback.Error = err.Error()
		}
		data, _ = json.Marshal(fallback)
	}
	return data
}

func fillScheduledTableActionLogResult(logReq *dto.RecordTableActionLogReq, resp *dto.RequestAppResp, err error, durationMillis int64) {
	if logReq == nil {
		return
	}
	logReq.DurationMillis = durationMillis
	logReq.Status = "success"
	if resp != nil && resp.Version != "" {
		logReq.Version = resp.Version
	}
	payload := dto.AppCallLogResponseBody{
		Code:          0,
		TotalCostMill: durationMillis,
	}
	if err != nil {
		logReq.Status = "failed"
		payload.Code = 1
		payload.Message = err.Error()
		payload.Error = err.Error()
		logReq.Summary = err.Error()
	} else if resp != nil {
		payload.Code = resp.ErrCode
		payload.ErrCode = resp.ErrCode
		payload.TraceID = resp.TraceId
		payload.Version = resp.Version
		payload.Result = resp.Result
		payload.Error = resp.Error
		if resp.Error != "" {
			logReq.Status = "failed"
			payload.Message = resp.Error
			logReq.Summary = resp.Error
		}
	}
	raw, marshalErr := json.Marshal(payload)
	if marshalErr == nil {
		logReq.ResponseBody = raw
	}
}
