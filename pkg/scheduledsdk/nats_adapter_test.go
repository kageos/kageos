package scheduledsdk

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/nats-io/nats.go"
)

func TestNATSAdapterUnsupportedControlPlaneOperations(t *testing.T) {
	adapter := NewNATSAdapter(nil, NATSAdapterOptions{})
	if _, err := adapter.CreateTask(context.Background(), CreateTaskRequest{}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("CreateTask error = %v, want ErrUnsupported", err)
	}
}

func TestNATSAdapterRequiresConnectionForExecutionUpdates(t *testing.T) {
	adapter := NewNATSAdapter(nil, NATSAdapterOptions{})
	err := adapter.MarkExecutionStarted(context.Background(), MarkExecutionStartedRequest{TaskID: 1, ExecutionID: 2})
	if err == nil || !strings.Contains(err.Error(), "nats connection") {
		t.Fatalf("MarkExecutionStarted error = %v, want nats connection error", err)
	}
}

func TestNATSAdapterRequestContextKeepsExistingDeadline(t *testing.T) {
	parent, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	got, gotCancel := natsAdapterRequestContext(parent, time.Second)
	defer gotCancel()
	if got != parent {
		t.Fatal("natsAdapterRequestContext should keep context with existing deadline")
	}
}

func TestApplyNATSContextHeaders(t *testing.T) {
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		TraceId:            "trace-1",
		RequestUser:        "alice",
		UserID:             "42",
		UserEmail:          "alice@example.com",
		LeaderUsername:     "bob",
		Token:              "must-not-enter-timer-control-nats",
		ClientSource:       contextx.ClientSourceScheduledTask,
		SourceType:         contextx.SourceTypeScheduledTask,
		SourceRef:          "timer_task:1:execution:2",
		WorkspaceSessionID: "session-1",
		WorkspaceRole:      "app_developer",
	})
	msg := nats.NewMsg("timer.test")
	applyNATSContextHeaders(msg, ctx)

	if got := msg.Header.Get(contextx.TraceIdHeader); got != "trace-1" {
		t.Fatalf("trace header = %q", got)
	}
	if got := msg.Header.Get(contextx.RequestUserHeader); got != "alice" {
		t.Fatalf("request user header = %q", got)
	}
	if got := msg.Header.Get(contextx.SourceRefHeader); got != "timer_task:1:execution:2" {
		t.Fatalf("source ref header = %q", got)
	}
	if got := msg.Header.Get(contextx.UserIDHeader); got != "42" {
		t.Fatalf("user id header = %q", got)
	}
	if got := msg.Header.Get(contextx.UserEmailHeader); got != "alice@example.com" {
		t.Fatalf("user email header = %q", got)
	}
	if got := msg.Header.Get(contextx.LeaderUsernameHeader); got != "bob" {
		t.Fatalf("leader username header = %q", got)
	}
	if got := msg.Header.Get(contextx.WorkspaceSessionIDHeader); got != "session-1" {
		t.Fatalf("workspace session header = %q", got)
	}
	if got := msg.Header.Get(contextx.TokenHeader); got != "" {
		t.Fatalf("timer control NATS header leaked token %q", got)
	}
}

func TestNATSAdapterResponseMustBeSignedByScheduler(t *testing.T) {
	secret := strings.Repeat("s", 32)
	workerAuth, err := NewWorkerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	schedulerAuth, err := NewSchedulerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	adapter := NewNATSAdapter(nil, NATSAdapterOptions{ResponseVerifier: workerAuth.ResponseVerifier})
	const requestNonce = "request-nonce-1"
	const requestSubject = "timer.v1.cmd.execution.started"

	unsigned := nats.NewMsg("_INBOX.timer-test")
	unsigned.Data = []byte(`{"ok":true,"request_nonce":"request-nonce-1","request_subject":"timer.v1.cmd.execution.started"}`)
	wrongNonce := signedNATSAdapterResponse(t, schedulerAuth.ResponseSigner, natsCommandResponse{
		OK:             true,
		RequestNonce:   "other-request",
		RequestSubject: requestSubject,
	})
	wrongSubject := signedNATSAdapterResponse(t, schedulerAuth.ResponseSigner, natsCommandResponse{
		OK:             true,
		RequestNonce:   requestNonce,
		RequestSubject: "timer.v1.cmd.execution.finished",
	})
	valid := signedNATSAdapterResponse(t, schedulerAuth.ResponseSigner, natsCommandResponse{
		OK:             true,
		RequestNonce:   requestNonce,
		RequestSubject: requestSubject,
	})
	responses := []*nats.Msg{unsigned, wrongNonce, wrongSubject, valid}
	nextCalls := 0
	next := func(context.Context) (*nats.Msg, error) {
		if nextCalls >= len(responses) {
			return nil, context.DeadlineExceeded
		}
		msg := responses[nextCalls]
		nextCalls++
		return msg, nil
	}
	if err := adapter.waitForResponse(context.Background(), requestNonce, requestSubject, next); err != nil {
		t.Fatalf("waitForResponse() error = %v", err)
	}
	if nextCalls != len(responses) {
		t.Fatalf("response reads = %d, want %d", nextCalls, len(responses))
	}
}

func signedNATSAdapterResponse(t *testing.T, signer *controlauth.Signer, body natsCommandResponse) *nats.Msg {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	msg := nats.NewMsg("_INBOX.timer-test")
	msg.Data = data
	if err := controlauth.SignNATSMessage(msg, signer); err != nil {
		t.Fatal(err)
	}
	return msg
}
