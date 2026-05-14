package apicall

import (
	"net/http"
	"strings"
	"testing"
)

func TestFormatHTTPErrorFormatsPermissionDenied(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Status:     "403 Forbidden",
	}
	body := []byte(`{
		"code": 7,
		"data": {
			"action": "form:write",
			"action_display": "表单提交",
			"apply_url": "/permissions/apply?resource=/system/openapi/message/send.form&action=form:write",
			"error_message": "无权限提交该表单",
			"resource_path": "/system/openapi/message/send.form"
		},
		"msg": "无权限提交该表单"
	}`)

	err := formatHTTPError(resp, body)
	if err == nil {
		t.Fatal("expected error")
	}
	message := err.Error()
	for _, want := range []string{
		"权限不足",
		"无权限提交该表单",
		"资源: /system/openapi/message/send.form",
		"操作: 表单提交(form:write)",
		"申请链接: /permissions/apply?resource=/system/openapi/message/send.form&action=form:write",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("expected %q in %q", want, message)
		}
	}
}

func TestFormatHTTPErrorFallsBackForNonPermissionErrors(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Status:     "500 Internal Server Error",
	}

	err := formatHTTPError(resp, []byte(`{"msg":"boom"}`))
	if err == nil {
		t.Fatal("expected error")
	}
	message := err.Error()
	if !strings.Contains(message, "HTTP错误: 500 500 Internal Server Error") {
		t.Fatalf("expected HTTP status in %q", message)
	}
	if !strings.Contains(message, `{"msg":"boom"}`) {
		t.Fatalf("expected raw body in %q", message)
	}
}
