package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/apperror"
	"gorm.io/gorm"
)

func TestErrorUsesTypedKindAndStableCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "validation", err: apperror.InvalidArgument("请求参数错误", nil), wantStatus: http.StatusBadRequest, wantCode: CodeInvalidArgument},
		{name: "forbidden", err: apperror.PermissionDenied("无权限访问工作空间", nil), wantStatus: http.StatusForbidden, wantCode: CodePermissionDenied},
		{name: "not found", err: apperror.NotFound("应用不存在", nil), wantStatus: http.StatusNotFound, wantCode: CodeNotFound},
		{name: "conflict", err: apperror.Conflict("应用已存在", nil), wantStatus: http.StatusConflict, wantCode: CodeConflict},
		{name: "method not allowed", err: apperror.MethodNotAllowed("当前资源不支持该操作", nil), wantStatus: http.StatusMethodNotAllowed, wantCode: CodeMethodNotAllowed},
		{name: "rate limited", err: apperror.RateLimited("请求过于频繁", nil), wantStatus: http.StatusTooManyRequests, wantCode: CodeRateLimited},
		{name: "unavailable", err: apperror.Unavailable("依赖服务不可用", nil), wantStatus: http.StatusServiceUnavailable, wantCode: CodeUnavailable},
		{name: "gorm not found", err: gorm.ErrRecordNotFound, wantStatus: http.StatusNotFound, wantCode: CodeNotFound},
		{name: "typed internal", err: apperror.New(apperror.KindInternal, "database detail", nil), wantStatus: http.StatusInternalServerError, wantCode: CodeInternal},
		{name: "unknown", err: errors.New("database connection failed"), wantStatus: http.StatusInternalServerError, wantCode: CodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			Error(ctx, tt.err)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			var body Response
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code != tt.wantCode {
				t.Fatalf("code = %q, want %q", body.Code, tt.wantCode)
			}
			if tt.wantStatus == http.StatusInternalServerError && body.Message != "服务器内部错误" {
				t.Fatalf("internal message leaked: %q", body.Message)
			}
		})
	}
}

func TestOkUsesCanonicalEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	OkWithData(ctx, gin.H{"id": 1})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var body Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	code, ok := body.Code.(float64)
	if !ok || code != 0 || body.Message == "" {
		t.Fatalf("unexpected envelope: %#v", body)
	}
}
