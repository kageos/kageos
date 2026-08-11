package v1

import (
	"strings"
	"testing"
)

func TestReadStandardAPIRequestBodyAcceptsBoundedPayload(t *testing.T) {
	data, err := readStandardAPIRequestBody(strings.NewReader(`{"name":"demo"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != `{"name":"demo"}` {
		t.Fatalf("body = %q", got)
	}
}

func TestReadStandardAPIRequestBodyRejectsOversizedPayload(t *testing.T) {
	_, err := readStandardAPIRequestBody(strings.NewReader(strings.Repeat("x", maxStandardAPIRequestBodyBytes+1)))
	if err == nil || !strings.Contains(err.Error(), "请求体超过") {
		t.Fatalf("error = %v, want request body size error", err)
	}
}
