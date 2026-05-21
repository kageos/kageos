package v1

import (
	"encoding/json"

	"github.com/kageos/kageos/dto"
)

func fillTableActionLogResult(logReq *dto.RecordTableActionLogReq, resp *dto.RequestAppResp, err error, durationMillis int64) {
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
	logReq.ResponseBody = mustMarshalTableActionLogPayload(payload)
}

func mustMarshalTableActionLogPayload(payload dto.AppCallLogResponseBody) json.RawMessage {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return raw
}
