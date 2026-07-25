package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/dto"
)

func TestWorkspaceActionRunnerSubmitsWorkspaceChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agent/api/v1/workspace/chat/stream" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Request-User"); got != "bob" {
			t.Fatalf("request user = %q, want bob", got)
		}
		if got := r.Header.Get("X-Client-Source"); got != WorkspaceActionClientSource {
			t.Fatalf("client source = %q, want %s", got, WorkspaceActionClientSource)
		}
		var body dto.WorkspaceChatReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Message.ContextUsage != dto.WorkspaceMessageContextCurrentTurn {
			t.Fatalf("context usage = %q, want %q", body.Message.ContextUsage, dto.WorkspaceMessageContextCurrentTurn)
		}
		if body.Message.Files != "kageos/pocket/meeting.pdf" {
			t.Fatalf("message files = %q", body.Message.Files)
		}
		if body.FullCodePath != "/alice/ops/meeting_room" || body.ResourceFullCodePath != "/alice/ops/meeting_room/notify.form" {
			t.Fatalf("workspace paths = directory %q resource %q", body.FullCodePath, body.ResourceFullCodePath)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "event: session\n")
		_, _ = fmt.Fprint(w, `data: {"session_id":"session-1"}`+"\n\n")
		_, _ = fmt.Fprint(w, "event: content\n")
		_, _ = fmt.Fprint(w, `data: {"content":"已处理完成"}`+"\n\n")
		_, _ = fmt.Fprint(w, "event: done\n")
		_, _ = fmt.Fprint(w, `data: {"session_id":"session-1","tool_calls":[]}`+"\n\n")
	}))
	defer server.Close()

	runner := NewWorkspaceActionRunner(server.URL)
	runner.startTimeout = time.Second
	runner.runTimeout = 3 * time.Second

	result, err := runner.Submit(context.Background(), WorkspaceActionRequest{
		RecipientUser: "bob",
		FullCodePath:  "/alice/ops/meeting_room",
		SourcePath:    "/alice/ops/meeting_room/notify.form",
		Content:       "帮我处理",
		Files:         "kageos/pocket/meeting.pdf",
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if result.SessionID != "session-1" || !result.Accepted {
		t.Fatalf("submit result = %#v", result)
	}
}

func TestWorkspaceActionRunnerStartTimeoutIsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-r.Context().Done()
	}))
	defer server.Close()

	runner := NewWorkspaceActionRunner(server.URL)
	runner.startTimeout = 30 * time.Millisecond
	runner.runTimeout = time.Second
	result, err := runner.Submit(context.Background(), WorkspaceActionRequest{
		RecipientUser: "bob",
		FullCodePath:  "/alice/demo",
		Content:       "继续处理",
	})
	if err == nil || result != nil || !strings.Contains(err.Error(), "超时") {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}
