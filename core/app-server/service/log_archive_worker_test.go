package service

import (
	"context"
	"encoding/json"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRetireLogGroupSchedulesCancelsOnlyRetiredExecutors(t *testing.T) {
	lists, cancels := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/tasks" {
			key := r.URL.Query().Get("executor_key")
			if key != "platform.log_groups" && key != "platform.log_groups_reconcile" {
				t.Errorf("unexpected executor: %s", key)
			}
			if r.URL.Query().Get("resource_scope") != "system" {
				t.Error("not restricted to system tasks")
			}
			lists++
			_ = json.NewEncoder(w).Encode(scheduledsdk.ListTasksResponse{Total: 3, List: []*scheduledsdk.Task{{ID: 1, Status: scheduledsdk.TaskStatusPending}, {ID: 2, Status: scheduledsdk.TaskStatusDone}, {ID: 3, Status: scheduledsdk.TaskStatusCancelled}}})
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/tasks/1/cancel" {
			cancels++
			w.WriteHeader(http.StatusOK)
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	client := scheduledsdk.NewClient(scheduledsdk.Options{BaseURL: server.URL})
	if err := retireLogGroupSchedules(context.Background(), client); err != nil {
		t.Fatal(err)
	}
	if lists != 2 || cancels != 2 {
		t.Fatalf("lists=%d cancels=%d", lists, cancels)
	}
}
