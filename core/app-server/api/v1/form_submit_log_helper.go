package v1

import (
	"encoding/json"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func buildFormOperateLogResponseBody(resp *dto.RequestAppResp, err error, totalCostMill int64) json.RawMessage {
	payload := map[string]interface{}{
		"code": 0,
	}
	if totalCostMill >= 0 {
		payload["total_cost_mill"] = totalCostMill
	}

	switch {
	case resp != nil:
		payload["code"] = resp.ErrCode
		if resp.TraceId != "" {
			payload["trace_id"] = resp.TraceId
		}
		if resp.Version != "" {
			payload["version"] = resp.Version
		}
		if resp.Result != nil {
			payload["result"] = resp.Result
		}
		if resp.Error != "" {
			payload["msg"] = resp.Error
			payload["error"] = resp.Error
		}
	case err != nil:
		payload["code"] = 1
		payload["msg"] = err.Error()
		payload["error"] = err.Error()
	}

	data, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		fallback := map[string]interface{}{
			"code": 1,
			"msg":  "marshal form operate log response failed",
		}
		if err != nil {
			fallback["msg"] = err.Error()
			fallback["error"] = err.Error()
		}
		data, _ = json.Marshal(fallback)
	}
	return data
}
