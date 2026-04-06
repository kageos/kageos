package service

import (
	"encoding/json"
	"testing"
)

func TestExtractFormOperateLogStatus(t *testing.T) {
	cases := []struct {
		name string
		body map[string]interface{}
		code int
		msg  string
	}{
		{
			name: "success",
			body: map[string]interface{}{
				"code":   0,
				"result": map[string]interface{}{"id": 1},
			},
			code: 0,
			msg:  "",
		},
		{
			name: "biz error",
			body: map[string]interface{}{
				"code": -1,
				"msg":  "参数校验失败",
			},
			code: -1,
			msg:  "参数校验失败",
		},
		{
			name: "system error compatible keys",
			body: map[string]interface{}{
				"err_code": 1,
				"error":    "系统错误",
			},
			code: 1,
			msg:  "系统错误",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatalf("marshal body failed: %v", err)
			}
			code, msg := extractFormOperateLogStatus(raw)
			if code != tc.code || msg != tc.msg {
				t.Fatalf("extractFormOperateLogStatus() = (%d, %q), want (%d, %q)", code, msg, tc.code, tc.msg)
			}
		})
	}
}
