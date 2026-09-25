package service

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/logger"
)

// Backend exceptions are diagnostics, not part of the model's business contract.
// Local field validation remains at the call site and is not passed here.
func publicToolBackendError(ctx context.Context, operation string, err error) string {
	logger.Errorf(ctx, "[ToolBackend] operation=%s error=%v", operation, err)
	var business *apicall.BusinessError
	if errors.As(err, &business) {
		if message := publicBusinessRejection(business.Message); message != "" {
			return "business_rejected：" + message
		}
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "operation_timeout：操作超时，执行结果尚未确认。"
	case errors.Is(err, context.Canceled):
		return "operation_canceled：操作已中断，执行结果尚未确认。"
	case errors.Is(err, os.ErrPermission):
		return "permission_denied：当前操作权限不足。"
	default:
		return "service_error：服务未能完成请求，执行结果尚未确认。"
	}
}

var diagnosticPathPattern = regexp.MustCompile(`(?:[A-Za-z]:[\\/]|/)[^\s"'<>，；：）)\]]+`)
var diagnosticANSI = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// Logs are a diagnostic view, not a file browser. Keep business messages and
// script line numbers while removing transport, SQL and physical location data.
func publicDiagnosticText(raw, workspace string) string {
	var lines []string
	for _, line := range strings.Split(diagnosticANSI.ReplaceAllString(raw, ""), "\n") {
		lower := strings.ToLower(line)
		internal := false
		for _, marker := range []string{"/_runtime/", "nats:", "nats://", "podman", "docker", "runtime.v1.", "app.v1.", "select ", "insert into ", "update `", "delete from ", "goroutine ", "database/sql", "gorm.io", "github.com/kageos/kageos/core/"} {
			if strings.Contains(lower, marker) {
				internal = true
				break
			}
		}
		if internal {
			continue
		}
		line = diagnosticPathPattern.ReplaceAllStringFunc(line, func(ref string) string {
			if workspace != "" && (ref == workspace || strings.HasPrefix(ref, strings.TrimRight(workspace, "/")+"/")) && !strings.Contains(ref, "/code/") {
				return ref
			}
			return "[路径]"
		})
		// Relative generated paths can also appear in stack frames.
		if strings.Contains(line, "namespace/") || strings.Contains(line, "code/cmd/") || strings.Contains(line, "workplace/") {
			continue
		}
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return "没有可展示的业务诊断信息。"
	}
	return strings.Join(lines, "\n")
}

// Legacy business envelopes can contain SQL or runtime errors. Keep the business
// reason only when it has no known implementation details; never expose raw SQL.
func publicBusinessRejection(message string) string {
	message = strings.TrimSpace(message)
	lower := strings.ToLower(message)
	for _, marker := range []string{"sqlstate", "unknown column", "error 1054", "syntax error", "select ", "insert into ", "update `", "delete from ", "gorm", "nats", "podman", "docker", "namespace/", "code/cmd/", "code/api/", "/_runtime/", "traceback", "goroutine", "panic:", "connection refused", "dial tcp"} {
		if strings.Contains(lower, marker) {
			return ""
		}
	}
	if message == "" || strings.ContainsAny(message, "\n\r") || diagnosticPathPattern.MatchString(message) {
		return ""
	}
	return message
}
