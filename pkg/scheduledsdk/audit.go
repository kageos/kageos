package scheduledsdk

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/kageos/kageos/pkg/contextx"
)

const (
	MetadataRequestEmail   = "request_email"
	MetadataLeaderUsername = "leader_username"
)

// AuditSourceRef returns the stable provenance reference used by operate logs.
func (e ExecutionRequestedEvent) AuditSourceRef() string {
	switch {
	case e.TaskID > 0 && e.ExecutionID > 0:
		return fmt.Sprintf("timer_task:%d:execution:%d", e.TaskID, e.ExecutionID)
	case e.ExecutionID > 0:
		return fmt.Sprintf("timer_execution:%d", e.ExecutionID)
	case e.TaskID > 0:
		return fmt.Sprintf("timer_task:%d", e.TaskID)
	default:
		eventID := strings.TrimSpace(e.EventID)
		if eventID != "" {
			return "timer_event:" + eventID
		}
		return ""
	}
}

// WithAuditContext stamps scheduled-task provenance on a context before the
// worker calls appserver or other business services.
func (e ExecutionRequestedEvent) WithAuditContext(parent context.Context) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return contextx.WithRequestInfo(parent, contextx.RequestInfo{
		TraceId:            e.TraceID,
		RequestUser:        e.RequestUser,
		Token:              e.Token,
		DepartmentFullPath: e.RequestUserDept,
		ClientSource:       contextx.ClientSourceScheduledTask,
		SourceType:         contextx.SourceTypeScheduledTask,
		SourceRef:          e.AuditSourceRef(),
	})
}

// ApplyAuditHeaders stamps scheduled-task provenance on direct HTTP calls.
func (e ExecutionRequestedEvent) ApplyAuditHeaders(header http.Header) {
	if header == nil {
		return
	}
	header.Set(contextx.ClientSourceHeader, contextx.ClientSourceScheduledTask)
	header.Set(contextx.SourceTypeHeader, contextx.SourceTypeScheduledTask)
	if sourceRef := e.AuditSourceRef(); sourceRef != "" {
		header.Set(contextx.SourceRefHeader, sourceRef)
	}
	if traceID := strings.TrimSpace(e.TraceID); traceID != "" {
		header.Set(contextx.TraceIdHeader, traceID)
	}
	if requestUser := strings.TrimSpace(e.RequestUser); requestUser != "" {
		header.Set(contextx.RequestUserHeader, requestUser)
	}
	if token := strings.TrimSpace(e.Token); token != "" {
		header.Set(contextx.TokenHeader, token)
	}
	if requestUserDept := strings.TrimSpace(e.RequestUserDept); requestUserDept != "" {
		header.Set(contextx.DepartmentFullPathHeader, requestUserDept)
	}
}
