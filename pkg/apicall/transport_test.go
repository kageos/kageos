package apicall

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/kageos/kageos/pkg/serviceconfig"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

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

func TestTransportFailureInvalidatesGatewayWithoutReplayingRequest(t *testing.T) {
	var healthHits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		healthHits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	t.Setenv("GATEWAY_URL", strings.Replace(server.URL, "127.0.0.1", "localhost", 1))

	_ = serviceconfig.GetGatewayURL()
	firstHealthHits := healthHits.Load()

	previousClient := httpClient
	t.Cleanup(func() { httpClient = previousClient })
	var attempts atomic.Int32
	httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts.Add(1)
		return nil, errors.New("connection failed")
	})}

	req, err := http.NewRequest(http.MethodPost, "http://gateway.invalid/write", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doAPIRequest[any](req); err == nil {
		t.Fatal("expected transport failure")
	}
	if attempts.Load() != 1 {
		t.Fatalf("request was replayed %d times", attempts.Load())
	}

	_ = serviceconfig.GetGatewayURL()
	if healthHits.Load() <= firstHealthHits {
		t.Fatal("transport failure did not invalidate gateway resolution")
	}
}
