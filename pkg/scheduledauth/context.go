package scheduledauth

import (
	"context"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

// WithExecutionToken creates a short-lived delegated token only after a worker
// starts processing an event. The event itself is never mutated, so the token
// cannot be written back to an outbox message or execution record.
func WithExecutionToken(parent context.Context, event scheduledsdk.ExecutionRequestedEvent, ttl time.Duration) (context.Context, error) {
	if ttl <= 0 {
		ttl = auth.DefaultScheduledTokenTTL
	}
	token, err := auth.NewJWTService().GenerateScheduledTokenWithContextTTL(auth.UserTokenContext{
		Username:           strings.TrimSpace(event.RequestUser),
		Email:              strings.TrimSpace(event.Metadata[scheduledsdk.MetadataRequestEmail]),
		DepartmentFullPath: strings.TrimSpace(event.RequestUserDept),
		LeaderUsername:     strings.TrimSpace(event.Metadata[scheduledsdk.MetadataLeaderUsername]),
	}, event.TaskID, event.ExecutionID, ttl)
	if err != nil {
		return nil, err
	}
	ctx := event.WithAuditContext(parent)
	return contextx.WithToken(ctx, token), nil
}
