package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func TestTimerHTTPCreateAndUpdateUseOnlyVerifiedIdentity(t *testing.T) {
	service := newTimerHTTPTestService(t)
	router := NewRouter(service)
	aliceToken := timerHTTPAccessToken(t, auth.UserTokenContext{
		UserID:             101,
		Username:           "alice",
		Email:              "alice@example.com",
		CompanyCode:        "acme",
		CompanyName:        "Acme",
		CompanyLogoURL:     "https://acme.example/logo.png",
		DepartmentFullPath: "/acme/engineering",
		LeaderUsername:     "alice-lead",
	})

	created := createTimerHTTPTask(t, router, aliceToken, scheduledsdk.CreateTaskRequest{
		Title:           "Alice task",
		ExecutorKey:     "agent.session",
		Schedule:        scheduledsdk.Every(3600),
		IdempotencyKey:  "shared-client-key",
		CreatedBy:       "bob",
		RequestUser:     "bob",
		RequestUserDept: "/forged/admin",
		Metadata: map[string]string{
			"custom":                            "preserved",
			scheduledsdk.MetadataRequestEmail:   "bob@example.com",
			scheduledsdk.MetadataCompanyCode:    "evil",
			scheduledsdk.MetadataCompanyName:    "Evil Corp",
			scheduledsdk.MetadataCompanyLogoURL: "https://evil.example/logo.png",
			scheduledsdk.MetadataLeaderUsername: "mallory",
		},
	})
	assertTimerTaskIdentity(t, created, "alice", "/acme/engineering", map[string]string{
		scheduledsdk.MetadataRequestEmail:   "alice@example.com",
		scheduledsdk.MetadataCompanyCode:    "acme",
		scheduledsdk.MetadataCompanyName:    "Acme",
		scheduledsdk.MetadataCompanyLogoURL: "https://acme.example/logo.png",
		scheduledsdk.MetadataLeaderUsername: "alice-lead",
		"custom":                            "preserved",
	})
	if created.IdempotencyKey == "shared-client-key" || !strings.HasPrefix(created.IdempotencyKey, "u1-") || len(created.IdempotencyKey) != 67 {
		t.Fatalf("scoped idempotency key = %q", created.IdempotencyKey)
	}

	update := scheduledsdk.UpdateTaskRequest{
		RequestUser:     stringPointer("bob"),
		RequestUserDept: stringPointer("/forged/admin"),
		Metadata: &map[string]string{
			"custom":                            "updated",
			scheduledsdk.MetadataRequestEmail:   "bob@example.com",
			scheduledsdk.MetadataCompanyCode:    "evil",
			scheduledsdk.MetadataLeaderUsername: "mallory",
		},
	}
	recorder := timerHTTPJSONRequest(t, router, aliceToken, http.MethodPut, fmt.Sprintf("/timer/api/v1/tasks/%d", created.ID), update)
	if recorder.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	var updated scheduledsdk.Task
	if err := json.Unmarshal(recorder.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	assertTimerTaskIdentity(t, &updated, "alice", "/acme/engineering", map[string]string{
		scheduledsdk.MetadataRequestEmail:   "alice@example.com",
		scheduledsdk.MetadataCompanyCode:    "acme",
		scheduledsdk.MetadataCompanyName:    "Acme",
		scheduledsdk.MetadataCompanyLogoURL: "https://acme.example/logo.png",
		scheduledsdk.MetadataLeaderUsername: "alice-lead",
		"custom":                            "updated",
	})
}

func TestTimerHTTPIdempotencyKeysAreScopedByAuthenticatedUser(t *testing.T) {
	service := newTimerHTTPTestService(t)
	router := NewRouter(service)
	aliceToken := timerHTTPAccessToken(t, auth.UserTokenContext{UserID: 101, Username: "alice", Email: "alice@example.com"})
	bobToken := timerHTTPAccessToken(t, auth.UserTokenContext{UserID: 202, Username: "bob", Email: "bob@example.com"})
	request := scheduledsdk.CreateTaskRequest{
		ExecutorKey:    "agent.session",
		Schedule:       scheduledsdk.Every(3600),
		IdempotencyKey: "same-client-key",
	}

	alice := createTimerHTTPTask(t, router, aliceToken, request)
	bob := createTimerHTTPTask(t, router, bobToken, request)
	aliceAgain := createTimerHTTPTask(t, router, aliceToken, request)
	if alice.ID == bob.ID || alice.IdempotencyKey == bob.IdempotencyKey {
		t.Fatalf("cross-user idempotency collision: alice=%#v bob=%#v", alice, bob)
	}
	if aliceAgain.ID != alice.ID || aliceAgain.IdempotencyKey != alice.IdempotencyKey {
		t.Fatalf("same-user idempotency changed: first=%#v again=%#v", alice, aliceAgain)
	}
}

func TestTimerHTTPRejectsCrossUserTaskAccess(t *testing.T) {
	service := newTimerHTTPTestService(t)
	router := NewRouter(service)
	aliceToken := timerHTTPAccessToken(t, auth.UserTokenContext{UserID: 101, Username: "alice", Email: "alice@example.com"})
	bobToken := timerHTTPAccessToken(t, auth.UserTokenContext{UserID: 202, Username: "bob", Email: "bob@example.com"})
	alice := createTimerHTTPTask(t, router, aliceToken, scheduledsdk.CreateTaskRequest{
		Title:       "Alice private task",
		ExecutorKey: "agent.session",
		Schedule:    scheduledsdk.Every(3600),
	})

	listRecorder := timerHTTPJSONRequest(t, router, bobToken, http.MethodGet, "/timer/api/v1/tasks?created_by=alice", nil)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("bob list status = %d; body=%s", listRecorder.Code, listRecorder.Body.String())
	}
	var list scheduledsdk.ListTasksResponse
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Total != 0 || len(list.List) != 0 {
		t.Fatalf("bob list leaked alice task: %#v", list)
	}

	taskPath := fmt.Sprintf("/timer/api/v1/tasks/%d", alice.ID)
	tests := []struct {
		method string
		path   string
		body   interface{}
	}{
		{method: http.MethodGet, path: taskPath},
		{method: http.MethodPut, path: taskPath, body: scheduledsdk.UpdateTaskRequest{RequestUser: stringPointer("bob")}},
		{method: http.MethodDelete, path: taskPath},
		{method: http.MethodPost, path: taskPath + "/pause"},
		{method: http.MethodPost, path: taskPath + "/resume"},
		{method: http.MethodPost, path: taskPath + "/cancel"},
		{method: http.MethodPost, path: taskPath + "/run_now"},
		{method: http.MethodGet, path: taskPath + "/executions"},
		{method: http.MethodGet, path: taskPath + "/executions/1"},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			recorder := timerHTTPJSONRequest(t, router, bobToken, tt.method, tt.path, tt.body)
			if recorder.Code != http.StatusNotFound {
				t.Fatalf("cross-user status = %d, want 404; body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}

	aliceGet := timerHTTPJSONRequest(t, router, aliceToken, http.MethodGet, taskPath, nil)
	if aliceGet.Code != http.StatusOK {
		t.Fatalf("alice task was changed/deleted by bob, status=%d body=%s", aliceGet.Code, aliceGet.Body.String())
	}
	aliceExecutions := timerHTTPJSONRequest(t, router, aliceToken, http.MethodGet, taskPath+"/executions", nil)
	if aliceExecutions.Code != http.StatusOK {
		t.Fatalf("alice executions status=%d body=%s", aliceExecutions.Code, aliceExecutions.Body.String())
	}
	var executions scheduledsdk.ListExecutionsResponse
	if err := json.Unmarshal(aliceExecutions.Body.Bytes(), &executions); err != nil {
		t.Fatal(err)
	}
	if executions.Total != 0 {
		t.Fatalf("bob run_now created an execution: %#v", executions)
	}
}

func timerHTTPAccessToken(t *testing.T, identity auth.UserTokenContext) string {
	t.Helper()
	token, err := auth.NewJWTService().GenerateAccessTokenWithContext(identity)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func createTimerHTTPTask(t *testing.T, router http.Handler, token string, req scheduledsdk.CreateTaskRequest) *scheduledsdk.Task {
	t.Helper()
	recorder := timerHTTPJSONRequest(t, router, token, http.MethodPost, "/timer/api/v1/tasks", req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("create task status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	var task scheduledsdk.Task
	if err := json.Unmarshal(recorder.Body.Bytes(), &task); err != nil {
		t.Fatal(err)
	}
	return &task
}

func timerHTTPJSONRequest(t *testing.T, router http.Handler, token, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set(contextx.TokenHeader, token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func assertTimerTaskIdentity(t *testing.T, task *scheduledsdk.Task, user, dept string, metadata map[string]string) {
	t.Helper()
	if task == nil {
		t.Fatal("task is nil")
	}
	if task.CreatedBy != user || task.RequestUser != user || task.RequestUserDept != dept {
		t.Fatalf("task identity = created_by:%q request_user:%q dept:%q, want %q/%q", task.CreatedBy, task.RequestUser, task.RequestUserDept, user, dept)
	}
	for key, want := range metadata {
		if got := task.Metadata[key]; got != want {
			t.Fatalf("task metadata[%q] = %q, want %q; all=%#v", key, got, want, task.Metadata)
		}
	}
}

func stringPointer(value string) *string {
	return &value
}

func TestScopedTimerIdempotencyKeyIsStableAndUserBound(t *testing.T) {
	alice := scopedTimerIdempotencyKey("alice", "key")
	if alice != scopedTimerIdempotencyKey("alice", "key") {
		t.Fatal("same user/key must be stable")
	}
	if alice == scopedTimerIdempotencyKey("bob", "key") {
		t.Fatal("different users must not share scoped key")
	}
	if got := scopedTimerIdempotencyKey("alice", ""); got != "" {
		t.Fatalf("empty key = %q, want empty", got)
	}
	if len(strings.TrimPrefix(alice, "u1-")) != 64 {
		t.Fatalf("scoped key length = %d", len(alice))
	}
}

func TestTimerTaskOwnerFallsBackToLegacyRequestUser(t *testing.T) {
	if got := timerTaskOwner(&scheduledsdk.Task{RequestUser: " alice "}); got != "alice" {
		t.Fatalf("legacy owner = %q, want alice", got)
	}
	if got := timerTaskOwner(&scheduledsdk.Task{CreatedBy: "bob", RequestUser: "alice"}); got != "bob" {
		t.Fatalf("explicit owner = %q, want bob", got)
	}
}
