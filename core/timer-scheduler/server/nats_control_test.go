package server

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/kageos/kageos/pkg/controlauth"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/nats-io/nats.go"
)

func TestTimerNATSControlRejectsUnsignedAndReplayedCommands(t *testing.T) {
	secret := strings.Repeat("s", 32)
	schedulerAuth, err := scheduledsdk.NewSchedulerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	workerAuth, err := scheduledsdk.NewWorkerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(scheduledsdk.MarkExecutionStartedRequest{TaskID: 1, ExecutionID: 2, WorkerID: "worker-1"})
	if err != nil {
		t.Fatal(err)
	}
	runCalls := 0
	run := func(context.Context) error { runCalls++; return nil }

	unsigned := nats.NewMsg(subjects.TimerExecutionStartedCommandSubject)
	unsigned.Data = payload
	var unsignedReq scheduledsdk.MarkExecutionStartedRequest
	handleTimerNATSCommand(unsigned, schedulerAuth.CommandVerifier, schedulerAuth.ResponseSigner, &unsignedReq, run)
	if runCalls != 0 {
		t.Fatalf("unsigned command executed %d times", runCalls)
	}

	signed := nats.NewMsg(subjects.TimerExecutionStartedCommandSubject)
	signed.Data = payload
	if err := controlauth.SignNATSMessage(signed, workerAuth.CommandSigner); err != nil {
		t.Fatal(err)
	}
	var signedReq scheduledsdk.MarkExecutionStartedRequest
	handleTimerNATSCommand(signed, schedulerAuth.CommandVerifier, schedulerAuth.ResponseSigner, &signedReq, run)
	if runCalls != 1 {
		t.Fatalf("signed command run calls = %d, want 1", runCalls)
	}
	handleTimerNATSCommand(signed, schedulerAuth.CommandVerifier, schedulerAuth.ResponseSigner, &signedReq, run)
	if runCalls != 1 {
		t.Fatalf("replayed command run calls = %d, want 1", runCalls)
	}
}

func TestTimerNATSControlDoesNotRespondToUnauthenticatedReply(t *testing.T) {
	secret := strings.Repeat("s", 32)
	schedulerAuth, err := scheduledsdk.NewSchedulerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	msg := nats.NewMsg(subjects.TimerExecutionStartedCommandSubject)
	msg.Reply = "victim.reply.subject"
	msg.Data = []byte(`{"task_id":1,"execution_id":2}`)
	responded := false
	handleTimerNATSCommandWithResponder(
		msg,
		schedulerAuth.CommandVerifier,
		schedulerAuth.ResponseSigner,
		&scheduledsdk.MarkExecutionStartedRequest{},
		func(context.Context) error { t.Fatal("unauthenticated command executed"); return nil },
		func(*nats.Msg, *controlauth.Signer, string, string, error) error {
			responded = true
			return nil
		},
	)
	if responded {
		t.Fatal("unauthenticated command caused a signed response")
	}
}

func TestTimerNATSControlResponseIsSigned(t *testing.T) {
	secret := strings.Repeat("s", 32)
	schedulerAuth, err := scheduledsdk.NewSchedulerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	workerAuth, err := scheduledsdk.NewWorkerNATSAuth(secret)
	if err != nil {
		t.Fatal(err)
	}
	response, err := buildTimerNATSCommandResponse(
		"_INBOX.timer-test",
		schedulerAuth.ResponseSigner,
		"request-nonce-1",
		subjects.TimerExecutionStartedCommandSubject,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := controlauth.VerifyNATSMessage(response, workerAuth.ResponseVerifier); err != nil {
		t.Fatalf("VerifyNATSMessage(response) error = %v", err)
	}
	var body timerNATSCommandResponse
	if err := json.Unmarshal(response.Data, &body); err != nil {
		t.Fatal(err)
	}
	if !body.OK || body.Error != "" || body.RequestNonce != "request-nonce-1" || body.RequestSubject != subjects.TimerExecutionStartedCommandSubject {
		t.Fatalf("unexpected response: %#v", body)
	}

	response.Data = []byte(`{"ok":false}`)
	otherVerifier, err := controlauth.NewVerifier(secret, scheduledsdk.TimerSchedulerResponseAuthScope, controlauth.VerifierOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := controlauth.VerifyNATSMessage(response, otherVerifier); !errors.Is(err, controlauth.ErrInvalidSignature) {
		t.Fatalf("tampered response error = %v, want ErrInvalidSignature", err)
	}
}
