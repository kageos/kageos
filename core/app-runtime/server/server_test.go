package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHealthReturnsOK(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	s.handleHealth(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if body := recorder.Body.String(); body != "ok" {
		t.Fatalf("body = %q, want %q", body, "ok")
	}
}
