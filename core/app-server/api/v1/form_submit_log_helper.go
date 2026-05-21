package v1

import (
	"encoding/json"

	"github.com/kageos/kageos/dto"
)

func buildFormOperateLogResponseBody(resp *dto.RequestAppResp, err error, totalCostMill int64) json.RawMessage {
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
