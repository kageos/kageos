package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
)

type schedulerHealthResponseForTest struct {
	Status              string `json:"status"`
	Service             string `json:"service"`
	Healthy             bool   `json:"healthy"`
	Message             string `json:"message"`
	HeartbeatAgeSeconds int64  `json:"heartbeat_age_seconds"`
}

func TestHandleSchedulerHealthReturnsUnavailableWhenServiceMissing(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	s.handleSchedulerHealth(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("content-type = %q, want %q", contentType, "application/json")
	}

	var resp schedulerHealthResponseForTest
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Status != "unavailable" || resp.Service != "app-scheduler" || resp.Healthy {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Message != "scheduler service unavailable" {
		t.Fatalf("message = %q, want %q", resp.Message, "scheduler service unavailable")
	}
}

func TestHandleSchedulerHealthReturnsStartingWhenHeartbeatNotObservedYet(t *testing.T) {
	s := &Server{
		scheduledTaskService: service.NewScheduledTaskService(
			nil,
			nil,
			nil,
			nil,
			service.ScheduledTaskServiceOptions{
				HeartbeatMaxAge: 30 * time.Second,
			},
		),
	}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	s.handleSchedulerHealth(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}

	var resp schedulerHealthResponseForTest
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Status != "starting" {
		t.Fatalf("status = %q, want %q", resp.Status, "starting")
	}
	if resp.Message != "scheduler heartbeat not observed yet" {
		t.Fatalf("message = %q, want %q", resp.Message, "scheduler heartbeat not observed yet")
	}
	if resp.HeartbeatAgeSeconds != 0 {
		t.Fatalf("heartbeat age = %d, want 0", resp.HeartbeatAgeSeconds)
	}
}
