package maintenance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func TestManagementClientRegistersAndTriggersViaHTTP(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if r.Header.Get(contextx.RequestUserHeader) != "system" {
			t.Error("system identity missing")
		}
		if r.Header.Get(contextx.ClientSourceHeader) != "app_manifest" {
			t.Error("managed client source missing")
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/timer/api/v1/tasks" {
			var req scheduledsdk.CreateTaskRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error(err)
			}
			if req.ExecutorKey != "platform.log_groups" {
				t.Error("wrong executor")
			}
			_, _ = w.Write([]byte(`{"id":42,"status":"paused"}`))
		} else {
			_, _ = w.Write([]byte(`{"id":7,"task_id":42}`))
		}
	}))
	defer server.Close()
	client := newManagementClient(server.URL + "/timer/api/v1")
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{RequestUser: "system", ClientSource: "app_manifest"})
	task, err := client.CreateTask(ctx, scheduledsdk.CreateTaskRequest{ExecutorKey: "platform.log_groups", ResourceScope: "system"})
	if err != nil {
		t.Fatal(err)
	}
	if task.ID != 42 || task.Status != scheduledsdk.TaskStatusPaused {
		t.Fatalf("unexpected task: %+v", task)
	}
	execution, err := client.RunNow(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if execution.ID != 7 {
		t.Fatalf("unexpected execution: %+v", execution)
	}
	if len(calls) != 2 || calls[0] != "POST /timer/api/v1/tasks" || calls[1] != "POST /timer/api/v1/tasks/42/run_now" {
		t.Fatalf("unexpected routes: %v", calls)
	}
}

func TestManagementURLUsesInternalBackend(t *testing.T) {
	got, err := managementURL([]config.RouteConfig{{ServiceName: "timer", Path: "/timer", Targets: []config.BackendConfig{{URL: "http://scheduler.internal:9098/"}}}})
	if err != nil || got != "http://scheduler.internal:9098/timer/api/v1" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := managementURL(nil); err == nil {
		t.Fatal("must not fall back to the public gateway")
	}
}
