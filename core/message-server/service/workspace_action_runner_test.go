package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/controlauth"
)

func TestWorkspaceActionRunnerRejectsMissingInternalSigner(t *testing.T) {
	runner := NewWorkspaceActionRunner("http://127.0.0.1:1", nil)
	runner.startTimeout = time.Second
	_, err := runner.Submit(context.Background(), WorkspaceActionRequest{
		RecipientUser: "bob",
		FullCodePath:  "/alice/ops/meeting_room",
		Content:       "帮我处理",
	})
	if err == nil || !strings.Contains(err.Error(), "signer is not configured") {
		t.Fatalf("missing signer error = %v", err)
	}
}

func TestWorkspaceActionRunnerSubmitsWorkspaceChat(t *testing.T) {
	const secret = "0123456789abcdef0123456789abcdef"
	signer, err := controlauth.NewSigner(secret, controlauth.HTTPWorkspaceActionScope)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := controlauth.NewVerifier(secret, controlauth.HTTPWorkspaceActionScope, controlauth.VerifierOptions{})
	if err != nil {
		t.Fatal(err)
	}
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
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := controlauth.VerifyHTTPRequest(r, bodyBytes, workspaceActionSignedHeaders(), verifier); err != nil {
			t.Fatalf("verify internal request signature: %v", err)
		}
		var body dto.WorkspaceChatReq
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Message.ContextUsage != dto.WorkspaceMessageContextCurrentTurn {
			t.Fatalf("context usage = %q, want %q", body.Message.ContextUsage, dto.WorkspaceMessageContextCurrentTurn)
		}
		if body.Message.Files != "kageos/pocket/meeting.pdf" {
			t.Fatalf("message files = %q", body.Message.Files)
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

	runner := NewWorkspaceActionRunner(server.URL, signer)
	runner.startTimeout = time.Second
	runner.runTimeout = 3 * time.Second

	result, err := runner.Submit(context.Background(), WorkspaceActionRequest{
		RecipientUser: "bob",
		FullCodePath:  "/alice/ops/meeting_room",
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

func TestWorkspaceActionRunnerDoesNotForwardSignedIdentityAcrossRedirect(t *testing.T) {
	const secret = "0123456789abcdef0123456789abcdef"
	signer, err := controlauth.NewSigner(secret, controlauth.HTTPWorkspaceActionScope)
	if err != nil {
		t.Fatal(err)
	}
	var redirectedRequests atomic.Int32
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectedRequests.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer redirectTarget.Close()
	redirectSource := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTarget.URL+"/capture", http.StatusTemporaryRedirect)
	}))
	defer redirectSource.Close()

	runner := NewWorkspaceActionRunner(redirectSource.URL, signer)
	runner.startTimeout = time.Second
	runner.runTimeout = time.Second
	_, err = runner.Submit(context.Background(), WorkspaceActionRequest{
		RecipientUser: "bob",
		FullCodePath:  "/alice/ops/meeting_room",
		Content:       "帮我处理",
	})
	if err == nil || !strings.Contains(err.Error(), "HTTP 307") {
		t.Fatalf("redirect submit error = %v, want HTTP 307", err)
	}
	if got := redirectedRequests.Load(); got != 0 {
		t.Fatalf("redirect target received %d signed requests, want 0", got)
	}
}
