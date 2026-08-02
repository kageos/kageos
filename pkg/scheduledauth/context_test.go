package scheduledauth

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func TestWithExecutionTokenKeepsTokenInWorkerContext(t *testing.T) {
	event := scheduledsdk.ExecutionRequestedEvent{
		TaskID:          10,
		ExecutionID:     20,
		TraceID:         "trace-1",
		RequestUser:     "alice",
		RequestUserDept: "/org/dev",
		Metadata: map[string]string{
			scheduledsdk.MetadataRequestEmail:   "alice@example.com",
			scheduledsdk.MetadataLeaderUsername: "leader",
		},
	}
	ttl := 4*time.Hour + 30*time.Minute
	ctx, err := WithExecutionToken(context.Background(), event, ttl)
	if err != nil {
		t.Fatal(err)
	}
	token := contextx.GetToken(ctx)
	claims, err := auth.NewJWTService().ValidateScheduledToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.TaskID != 10 || claims.ExecutionID != 20 || claims.UserID != 0 || claims.Username != "alice" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.DepartmentFullPath == nil || *claims.DepartmentFullPath != "/org/dev" {
		t.Fatalf("unexpected department claims: %+v", claims)
	}
	if got := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time); got != ttl {
		t.Fatalf("token ttl = %s, want %s", got, ttl)
	}
	if contextx.GetTraceId(ctx) != "trace-1" || contextx.GetRequestUser(ctx) != "alice" {
		t.Fatal("scheduled audit context must be preserved")
	}
	if event.Token != "" {
		t.Fatal("worker token must not mutate the execution event")
	}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), `"token"`) || strings.Contains(string(payload), token) {
		t.Fatalf("worker token must not enter the event payload: %s", payload)
	}
}

func TestWithExecutionTokenUsesDefaultTTL(t *testing.T) {
	event := scheduledsdk.ExecutionRequestedEvent{TaskID: 10, ExecutionID: 20, RequestUser: "alice"}
	ctx, err := WithExecutionToken(context.Background(), event, 0)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.NewJWTService().ValidateScheduledToken(contextx.GetToken(ctx))
	if err != nil {
		t.Fatal(err)
	}
	if got := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time); got != auth.DefaultScheduledTokenTTL {
		t.Fatalf("token ttl = %s, want %s", got, auth.DefaultScheduledTokenTTL)
	}
}
