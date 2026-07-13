package service

import (
	"encoding/json"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

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
