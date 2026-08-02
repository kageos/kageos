package scheduledsdk

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
)

func TestExecutionRequestedEventIssueTokenKeepsTokenOutOfJSON(t *testing.T) {
	event := ExecutionRequestedEvent{
		TaskID:          10,
		ExecutionID:     20,
		RequestUser:     "alice",
		RequestUserDept: "/org/dev",
		Metadata: map[string]string{
			MetadataRequestEmail: "alice@example.com",
		},
	}
	ttl := 4*time.Hour + 30*time.Minute
	if err := event.IssueToken(ttl); err != nil {
		t.Fatal(err)
	}
	claims, err := auth.NewJWTService().ValidateScheduledToken(event.Token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.TaskID != 10 || claims.ExecutionID != 20 || claims.UserID != 0 || claims.Username != "alice" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if got := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time); got != ttl {
		t.Fatalf("token ttl = %s, want %s", got, ttl)
	}
	ctx := event.WithAuditContext(context.Background())
	if got := contextx.GetToken(ctx); got != event.Token {
		t.Fatal("issued token must stay available for the whole worker context")
	}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "token") || strings.Contains(string(payload), event.Token) {
		t.Fatalf("runtime token must not be serialized: %s", payload)
	}
}

func TestExecutionRequestedEventIssueTokenDoesNotRequireUserID(t *testing.T) {
	event := ExecutionRequestedEvent{TaskID: 10, ExecutionID: 20, RequestUser: "alice"}
	if err := event.IssueToken(time.Hour); err != nil {
		t.Fatalf("username-only scheduled token issuance failed: %v", err)
	}
	claims, err := auth.NewJWTService().ValidateScheduledToken(event.Token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 0 || claims.Username != "alice" {
		t.Fatalf("unexpected username-only claims: %+v", claims)
	}
}
