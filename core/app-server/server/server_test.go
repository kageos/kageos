package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
)

func TestHandleSchedulerHealthReturnsUnavailableWhenServiceMissing(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	s.handleSchedulerHealth(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "scheduler service unavailable") {
		t.Fatalf("body = %q, want service unavailable message", body)
	}
}

func TestHandleSchedulerHealthReturnsUnavailableWhenHeartbeatStale(t *testing.T) {
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
	if body := recorder.Body.String(); !strings.Contains(body, "scheduler heartbeat stale") {
		t.Fatalf("body = %q, want stale heartbeat message", body)
	}
}
