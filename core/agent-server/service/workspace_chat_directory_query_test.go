package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kageos/kageos/dto"
)

func TestGetWorkspaceServiceTreeDetailsBatchUsesAppServerQueryAPI(t *testing.T) {
	wantPaths := []string{"/alice/demo", "/alice/demo/sweep.form"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/workspace/api/v1/directory-queries" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var req dto.BatchGetServiceTreeDetailsReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.FullCodePaths) != len(wantPaths) {
			t.Fatalf("full_code_paths = %#v", req.FullCodePaths)
		}
		for i, want := range wantPaths {
			if req.FullCodePaths[i] != want {
				t.Fatalf("full_code_paths[%d] = %q, want %q", i, req.FullCodePaths[i], want)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{"items":[{"name":"Demo","code":"demo","full_code_path":"/alice/demo"}]}}`))
	}))
	defer server.Close()
	t.Setenv("GATEWAY_URL", server.URL)

	resp, err := getWorkspaceServiceTreeDetailsBatch(context.Background(), wantPaths)
	if err != nil {
		t.Fatalf("getWorkspaceServiceTreeDetailsBatch: %v", err)
	}
	if resp == nil || len(resp.Items) != 1 || resp.Items[0].Name != "Demo" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}
