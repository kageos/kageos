package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/subjects"
)

func TestValidateTimerRequestedPayloadRejectsLegacyToken(t *testing.T) {
	subject := subjects.TimerExecutionRequestedSubject("agent.session")
	event := scheduledsdk.ExecutionRequestedEvent{
		ExecutorKey: "agent.session",
		Token:       "eyJ.old-thirty-day-token",
	}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateTimerRequestedPayload(subject, payload); err == nil || !strings.Contains(err.Error(), "bearer token") {
		t.Fatalf("validate error = %v, want bearer token rejection", err)
	}
}

func TestValidateTimerRequestedPayloadAcceptsTokenlessRequest(t *testing.T) {
	event := scheduledsdk.ExecutionRequestedEvent{
		ExecutorKey: "agent.session",
	}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateTimerRequestedPayload(subjects.TimerExecutionRequestedSubject(event.ExecutorKey), payload); err != nil {
		t.Fatalf("validate tokenless request: %v", err)
	}
}
