package scheduledsdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kageos/kageos/pkg/contextx"
)

func TestHTTPAdapterForwardsContextHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertHeader := func(name string, want string) {
			t.Helper()
			if got := r.Header.Get(name); got != want {
				t.Fatalf("%s = %q, want %q", name, got, want)
			}
		}
		assertHeader(contextx.TokenHeader, "token-1")
		assertHeader(contextx.TraceIdHeader, "trace-1")
		assertHeader(contextx.RequestUserHeader, "alice")
		assertHeader(contextx.DepartmentFullPathHeader, "/org/dev")
		assertHeader(contextx.CompanyCodeHeader, "acme")
		assertHeader(contextx.CompanyNameHeader, "Acme")
		assertHeader(contextx.ClientSourceHeader, contextx.ClientSourceAgent)
		assertHeader(contextx.SourceTypeHeader, contextx.SourceTypeAgentTool)
		assertHeader(contextx.SourceRefHeader, "session-1")

		_ = json.NewEncoder(w).Encode(ListTasksResponse{List: []*Task{}, Total: 0})
	}))
	defer server.Close()

	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		Token:              "token-1",
		TraceId:            "trace-1",
		RequestUser:        "alice",
		DepartmentFullPath: "/org/dev",
		CompanyCode:        "acme",
		CompanyName:        "Acme",
		ClientSource:       contextx.ClientSourceAgent,
		SourceType:         contextx.SourceTypeAgentTool,
		SourceRef:          "session-1",
	})

	client := NewClient(Options{BaseURL: server.URL})
	if _, err := client.ListTasks(ctx, ListTasksRequest{}); err != nil {
		t.Fatalf("ListTasks returned error: %v", err)
	}
}

func TestHTTPAdapterDeleteTaskUsesDeleteMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/tasks/42" {
			t.Fatalf("path = %s, want /tasks/42", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewClient(Options{BaseURL: server.URL})
	if err := client.DeleteTask(context.Background(), 42); err != nil {
		t.Fatalf("DeleteTask returned error: %v", err)
	}
}
