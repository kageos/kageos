package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	msgmodel "github.com/kageos/kageos/core/message-server/model"
	msgrepo "github.com/kageos/kageos/core/message-server/repository"
	"github.com/kageos/kageos/core/message-server/service"
	"github.com/kageos/kageos/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSubmitPublicMessageActionReplySubmitsWorkspaceChat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newActionAPITestMessageRepo(t)
	entry, err := repo.Create(context.Background(), dto.MessageSendMeta{
		From:        "agent",
		SourcePath:  "/alice/ops/meeting_room",
		SourceTitle: "会议室",
		TraceID:     "trace-1",
	}, dto.MessageSendPayload{
		Title:   "会议快开始了",
		Content: "会议室 301 的会议将在 15 分钟后开始。",
	}, []string{"bob"})
	if err != nil {
		t.Fatalf("create message: %v", err)
	}
	rawToken, _, err := repo.CreateActionToken(context.Background(), msgrepo.CreateActionTokenInput{
		MessageID:         entry.ID,
		RecipientUsername: "bob",
		AllowedActions:    []string{"reply"},
		SourcePath:        entry.SourcePath,
		ThreadKey:         entry.ThreadKey,
		TraceID:           entry.TraceID,
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	runner := &fakeWorkspaceActionRunner{sessionID: "agent-session-2"}
	s := &Server{messageRepo: repo, workspaceActionRunner: runner}
	router := gin.New()
	router.POST("/message/api/v1/public/actions/:token/reply", s.submitPublicMessageActionReply)

	body := bytes.NewBufferString(`{"content":"帮我延迟到下午 5 点，并通知相关人。","action":"reply"}`)
	req := httptest.NewRequest(http.MethodPost, "/message/api/v1/public/actions/"+rawToken+"/reply", body)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var result struct {
		Code int                        `json:"code"`
		Msg  string                     `json:"msg"`
		Data dto.MessageActionReplyResp `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	if result.Code != 0 {
		t.Fatalf("response code=%d msg=%s", result.Code, result.Msg)
	}
	if !result.Data.AgentSubmitted || result.Data.WorkspaceSessionID != "agent-session-2" {
		t.Fatalf("reply response = %#v", result.Data)
	}
	if runner.req.RecipientUser != "bob" || runner.req.FullCodePath != "/alice/ops/meeting_room" {
		t.Fatalf("runner req = %#v", runner.req)
	}
	if runner.req.SessionID != "" {
		t.Fatalf("runner should create new session, got session_id=%q", runner.req.SessionID)
	}
	if !strings.Contains(runner.req.Content, "会议室 301") || !strings.Contains(runner.req.Content, "延迟到下午 5 点") {
		t.Fatalf("runner content = %q", runner.req.Content)
	}
	if !strings.Contains(runner.req.Content, "必须使用 Markdown 格式") || !strings.Contains(runner.req.Content, "content_type 使用 markdown") {
		t.Fatalf("runner content missing markdown reply guardrails = %q", runner.req.Content)
	}

	thread, total, err := repo.ListInbox(context.Background(), "bob", msgrepo.InboxListFilter{ThreadKey: entry.ThreadKey}, 0, 20)
	if err != nil {
		t.Fatalf("list thread: %v", err)
	}
	if total != 1 || len(thread) != 1 || thread[0].ID != entry.ID {
		t.Fatalf("message thread should not receive reply message: total=%d thread=%#v", total, thread)
	}
	view, err := repo.GetActionView(context.Background(), rawToken, "")
	if err != nil {
		t.Fatalf("get action view: %v", err)
	}
	if view.TokenStatus != string(dto.MessageActionTokenStatusSubmitted) || view.WorkspaceSession != "agent-session-2" {
		t.Fatalf("view after submit = %#v", view)
	}
}

func TestSubmitPublicMessageActionReplyRequiresAuthForRouteToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newActionAPITestMessageRepo(t)
	entry, err := repo.Create(context.Background(), dto.MessageSendMeta{
		From:        "agent",
		SourcePath:  "/alice/sales/orders",
		SourceTitle: "订单",
	}, dto.MessageSendPayload{
		Title:   "订单状态待确认",
		Content: "订单 A123 需要确认。",
	}, []string{"bob", "alice"})
	if err != nil {
		t.Fatalf("create message: %v", err)
	}
	rawToken, _, err := repo.CreateActionToken(context.Background(), msgrepo.CreateActionTokenInput{
		MessageID:         entry.ID,
		RecipientUsername: "bob",
		AuthorizedUsers:   []string{"bob", "alice"},
		RequireAuth:       true,
		AllowedActions:    []string{"reply"},
		SourcePath:        entry.SourcePath,
		ThreadKey:         entry.ThreadKey,
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	runner := &fakeWorkspaceActionRunner{sessionID: "route-session-1"}
	s := &Server{messageRepo: repo, workspaceActionRunner: runner}
	router := gin.New()
	router.POST("/message/api/v1/public/actions/:token/reply", s.submitPublicMessageActionReply)

	body := bytes.NewBufferString(`{"content":"我来处理这个订单。","action":"reply"}`)
	req := httptest.NewRequest(http.MethodPost, "/message/api/v1/public/actions/"+rawToken+"/reply", body)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d body=%s", recorder.Code, recorder.Body.String())
	}

	body = bytes.NewBufferString(`{"content":"我来处理这个订单。","action":"reply"}`)
	req = httptest.NewRequest(http.MethodPost, "/message/api/v1/public/actions/"+rawToken+"/reply", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-User", "alice")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("authenticated status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var result struct {
		Code int                        `json:"code"`
		Msg  string                     `json:"msg"`
		Data dto.MessageActionReplyResp `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	if result.Code != 0 || !result.Data.AgentSubmitted {
		t.Fatalf("response = %#v", result)
	}
	if runner.req.RecipientUser != "alice" {
		t.Fatalf("runner recipient user = %q, want alice", runner.req.RecipientUser)
	}
	if !strings.Contains(runner.req.Content, "处理用户：alice") {
		t.Fatalf("runner content should contain authenticated user, got %q", runner.req.Content)
	}
}

type fakeWorkspaceActionRunner struct {
	sessionID string
	req       service.WorkspaceActionRequest
}

func (f *fakeWorkspaceActionRunner) Submit(_ context.Context, req service.WorkspaceActionRequest) (*service.WorkspaceActionSubmitResult, error) {
	f.req = req
	return &service.WorkspaceActionSubmitResult{SessionID: f.sessionID, Accepted: true}, nil
}

func newActionAPITestMessageRepo(t *testing.T) *msgrepo.MessageRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := msgmodel.InitModels(db); err != nil {
		t.Fatalf("migrate message models: %v", err)
	}
	return msgrepo.NewMessageRepository(db)
}
