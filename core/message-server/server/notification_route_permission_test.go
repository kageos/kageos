package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	apiresponse "github.com/kageos/kageos/pkg/ginx/response"
)

type stubNotificationRoutePermissionResolver struct {
	response     *dto.MyPermissionsResp
	err          error
	resourcePath string
}

func (s *stubNotificationRoutePermissionResolver) MyPermissions(_ context.Context, resourcePath string) (*dto.MyPermissionsResp, error) {
	s.resourcePath = resourcePath
	return s.response, s.err
}

func TestRequireNotificationRouteAdminAllowsAdmin(t *testing.T) {
	resolver := &stubNotificationRoutePermissionResolver{
		response: &dto.MyPermissionsResp{
			Permissions: map[access.Action]bool{access.ActionAdmin: true},
		},
	}
	server := &Server{notificationRouteAuth: resolver}

	if err := server.requireNotificationRouteAdmin(context.Background(), "alice/sales/orders/"); err != nil {
		t.Fatalf("requireNotificationRouteAdmin() error = %v", err)
	}
	if resolver.resourcePath != "/alice/sales/orders" {
		t.Fatalf("resource path = %q", resolver.resourcePath)
	}
}

func TestRequireNotificationRouteAdminRejectsMember(t *testing.T) {
	server := &Server{notificationRouteAuth: &stubNotificationRoutePermissionResolver{
		response: &dto.MyPermissionsResp{
			Permissions: map[access.Action]bool{
				access.ActionRead:  true,
				access.ActionWrite: true,
			},
		},
	}}

	err := server.requireNotificationRouteAdmin(context.Background(), "/alice/sales/orders")
	if err == nil || !strings.Contains(err.Error(), "需要 admin 权限") {
		t.Fatalf("requireNotificationRouteAdmin() error = %v", err)
	}
}

func TestRequireNotificationRouteAdminFailsClosed(t *testing.T) {
	t.Run("resolver error", func(t *testing.T) {
		server := &Server{notificationRouteAuth: &stubNotificationRoutePermissionResolver{err: errors.New("workspace unavailable")}}
		err := server.requireNotificationRouteAdmin(context.Background(), "/alice/sales/orders")
		if err == nil || !strings.Contains(err.Error(), "workspace unavailable") {
			t.Fatalf("requireNotificationRouteAdmin() error = %v", err)
		}
	})

	t.Run("resolver missing", func(t *testing.T) {
		err := (&Server{}).requireNotificationRouteAdmin(context.Background(), "/alice/sales/orders")
		if err == nil || !strings.Contains(err.Error(), "权限服务未初始化") {
			t.Fatalf("requireNotificationRouteAdmin() error = %v", err)
		}
	})
}

func TestNotificationRouteMutationsRequireAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := &Server{notificationRouteAuth: &stubNotificationRoutePermissionResolver{
		response: &dto.MyPermissionsResp{
			Permissions: map[access.Action]bool{
				access.ActionRead:  true,
				access.ActionWrite: true,
			},
		},
	}}

	tests := []struct {
		name   string
		method string
		target string
		body   string
		run    func(*gin.Context)
	}{
		{
			name:   "upsert",
			method: http.MethodPut,
			target: "/message/api/v1/notification_routes/wecom",
			body:   `{"scope_path":"/alice/sales/orders","webhook_url":"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test"}`,
			run:    server.upsertNotificationRoute,
		},
		{
			name:   "delete",
			method: http.MethodDelete,
			target: "/message/api/v1/notification_routes/wecom?scope_path=/alice/sales/orders",
			run:    server.deleteNotificationRoute,
		},
		{
			name:   "test",
			method: http.MethodPost,
			target: "/message/api/v1/notification_routes/wecom/test?scope_path=/alice/sales/orders",
			run:    server.testNotificationRoute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(tt.method, tt.target, bytes.NewBufferString(tt.body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			ctx.Request.Header.Set(contextx.RequestUserHeader, "bob")
			ctx.Params = gin.Params{{Key: "channel", Value: "wecom"}}

			tt.run(ctx)

			var result apiresponse.Response
			if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
				t.Fatalf("decode response: %v; body=%s", err, recorder.Body.String())
			}
			if result.Code != apiresponse.ERROR || !strings.Contains(result.Msg, "需要 admin 权限") {
				t.Fatalf("unexpected response: %#v", result)
			}
		})
	}
}
