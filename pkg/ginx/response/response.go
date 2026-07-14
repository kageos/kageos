package response

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/apperror"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/gorm"
)

const CodeOK = 0

const (
	CodeInvalidArgument  = "invalid_argument"
	CodeUnauthenticated  = "unauthenticated"
	CodePermissionDenied = "permission_denied"
	CodeNotFound         = "not_found"
	CodeConflict         = "conflict"
	CodeMethodNotAllowed = "method_not_allowed"
	CodeRateLimited      = "rate_limited"
	CodeInternal         = "internal"
	CodeUnavailable      = "unavailable"
)

type Response struct {
	Code     interface{}            `json:"code"`
	Data     interface{}            `json:"data,omitempty"`
	Message  string                 `json:"message"`
	Details  interface{}            `json:"details,omitempty"`
	TraceID  string                 `json:"trace_id,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func Result(status int, code interface{}, data interface{}, message string, c *gin.Context, metadata ...map[string]interface{}) {
	result := Response{
		Code:    code,
		Data:    data,
		Message: message,
		TraceID: contextx.GetTraceId(c),
	}
	if len(metadata) > 0 {
		result.Metadata = metadata[0]
	}
	c.JSON(status, result)
}

func Ok(c *gin.Context) {
	Result(http.StatusOK, CodeOK, map[string]interface{}{}, "操作成功", c)
}

func OkWithMessage(c *gin.Context, message string) {
	Result(http.StatusOK, CodeOK, map[string]interface{}{}, message, c)
}

func OkWithData(c *gin.Context, data interface{}, metadata ...map[string]interface{}) {
	Result(http.StatusOK, CodeOK, data, "成功", c, metadata...)
}

func OkWithDetailed(c *gin.Context, data interface{}, message string) {
	Result(http.StatusOK, CodeOK, data, message, c)
}

func Fail(c *gin.Context) { Internal(c, "操作失败") }

func NoAuth(c *gin.Context, message string) {
	Result(http.StatusUnauthorized, CodeUnauthenticated, nil, message, c)
}

// Error writes a typed application error. Unknown errors are deliberately
// treated as internal failures: the adapter never guesses semantics from text.
func Error(c *gin.Context, err error, metadata ...map[string]interface{}) {
	if err == nil {
		return
	}
	appErr, ok := apperror.As(err)
	if !ok {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			NotFound(c, "资源不存在")
			return
		}
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			Conflict(c, "资源已存在")
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			Unavailable(c, err.Error(), metadata...)
			return
		}
		logger.Errorf(c, "request failed: %v", err)
		Result(http.StatusInternalServerError, CodeInternal, nil, "服务器内部错误", c, metadata...)
		return
	}

	status, code := transportStatus(appErr.Kind())
	message := appErr.Message()
	if status >= http.StatusInternalServerError {
		logger.Errorf(c, "request failed: %v", err)
		if appErr.Kind() == apperror.KindInternal {
			message = "服务器内部错误"
		} else if message == "" {
			message = "服务暂时不可用"
		}
	}
	result := Response{
		Code:     code,
		Message:  message,
		Details:  appErr.Details(),
		TraceID:  contextx.GetTraceId(c),
		Metadata: firstMetadata(metadata),
	}
	c.JSON(status, result)
}

func BadRequest(c *gin.Context, message string) {
	Result(http.StatusBadRequest, CodeInvalidArgument, nil, message, c)
}

func Forbidden(c *gin.Context, message string) {
	Result(http.StatusForbidden, CodePermissionDenied, nil, message, c)
}

func NotFound(c *gin.Context, message string) {
	Result(http.StatusNotFound, CodeNotFound, nil, message, c)
}

func Conflict(c *gin.Context, message string) {
	Result(http.StatusConflict, CodeConflict, nil, message, c)
}

func MethodNotAllowed(c *gin.Context, message string) {
	Result(http.StatusMethodNotAllowed, CodeMethodNotAllowed, nil, message, c)
}

func TooManyRequests(c *gin.Context, message string) {
	Result(http.StatusTooManyRequests, CodeRateLimited, nil, message, c)
}

func Internal(c *gin.Context, cause string, metadata ...map[string]interface{}) {
	logger.Errorf(c, "request failed: %s", cause)
	Result(http.StatusInternalServerError, CodeInternal, nil, "服务器内部错误", c, metadata...)
}

func Unavailable(c *gin.Context, cause string, metadata ...map[string]interface{}) {
	logger.Errorf(c, "dependency unavailable: %s", cause)
	Result(http.StatusServiceUnavailable, CodeUnavailable, nil, "服务暂时不可用", c, metadata...)
}

// ApplicationError 保留工作区应用自己的错误码，但不再把它当作平台 HTTP 状态码。
func ApplicationError(c *gin.Context, appCode int, message string, metadata ...map[string]interface{}) {
	status := http.StatusUnprocessableEntity
	code := "application_error"
	if appCode > 0 {
		status = http.StatusBadGateway
		code = "application_failure"
	}
	result := Response{
		Code:    code,
		Message: message,
		Details: map[string]interface{}{"application_code": appCode},
		TraceID: contextx.GetTraceId(c),
	}
	if len(metadata) > 0 {
		result.Metadata = metadata[0]
	}
	c.JSON(status, result)
}

func transportStatus(kind apperror.Kind) (int, string) {
	switch kind {
	case apperror.KindInvalidArgument:
		return http.StatusBadRequest, CodeInvalidArgument
	case apperror.KindUnauthenticated:
		return http.StatusUnauthorized, CodeUnauthenticated
	case apperror.KindPermissionDenied:
		return http.StatusForbidden, CodePermissionDenied
	case apperror.KindNotFound:
		return http.StatusNotFound, CodeNotFound
	case apperror.KindConflict:
		return http.StatusConflict, CodeConflict
	case apperror.KindMethodNotAllowed:
		return http.StatusMethodNotAllowed, CodeMethodNotAllowed
	case apperror.KindRateLimited:
		return http.StatusTooManyRequests, CodeRateLimited
	case apperror.KindUnavailable:
		return http.StatusServiceUnavailable, CodeUnavailable
	default:
		return http.StatusInternalServerError, CodeInternal
	}
}

func firstMetadata(values []map[string]interface{}) map[string]interface{} {
	if len(values) == 0 {
		return nil
	}
	return values[0]
}
