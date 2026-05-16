package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
)

func TestAPICallPropagatesRequestContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/service-tree/search" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get(contextx.TokenHeader); got != "token-1" {
			t.Fatalf("token header = %q, want token-1", got)
		}
		if got := r.Header.Get(contextx.TraceIdHeader); got != "trace-1" {
			t.Fatalf("trace header = %q, want trace-1", got)
		}
		if got := r.Header.Get(contextx.RequestUserHeader); got != "alice" {
			t.Fatalf("request user header = %q, want alice", got)
		}
		if got := r.Header.Get(contextx.DepartmentFullPathHeader); got != "/org/dev" {
			t.Fatalf("department header = %q, want /org/dev", got)
		}
		if got := r.Header.Get(contextx.ClientSourceHeader); got != "agent" {
			t.Fatalf("client source header = %q, want agent", got)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["keyword"] != "ticket" {
			t.Fatalf("keyword = %v, want ticket", body["keyword"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{"matched":true,"name":"ticket"}}`))
	}))
	defer server.Close()
	t.Setenv("GATEWAY_URL", server.URL)

	a := &App{}
	ctx, err := a.NewContext(context.Background(), &dto.RequestAppReq{
		TraceId:         "trace-1",
		RequestUser:     "alice",
		RequestUserDept: "/org/dev",
		Token:           "token-1",
		ClientSource:    "agent",
		Method:          "POST",
		Router:          "/workspace/search.form",
	})
	if err != nil {
		t.Fatalf("NewContext returned error: %v", err)
	}

	var resp struct {
		Matched bool   `json:"matched"`
		Name    string `json:"name"`
	}
	err = ctx.APICall(http.MethodPost, "/api/v1/service-tree/search", map[string]interface{}{
		"keyword": "ticket",
	}, &resp)
	if err != nil {
		t.Fatalf("APICall returned error: %v", err)
	}
	if !resp.Matched || resp.Name != "ticket" {
		t.Fatalf("resp = %+v", resp)
	}
}
