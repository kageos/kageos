package apicall

import (
	"net/http"
	"strings"
	"testing"
)

func TestFormatHTTPErrorIncludesForbiddenBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Status:     "403 Forbidden",
	}
	body := []byte(`{"msg":"访问被拒绝"}`)

	err := formatHTTPError(resp, body)
	if err == nil {
		t.Fatal("expected error")
	}
	message := err.Error()
	for _, want := range []string{
		"HTTP错误: 403 403 Forbidden",
		`{"msg":"访问被拒绝"}`,
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
