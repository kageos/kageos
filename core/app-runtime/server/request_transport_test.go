package server

import (
	"encoding/json"
	"testing"

	"github.com/kageos/kageos/core/app-runtime/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/appinvoke"
	appconfig "github.com/kageos/kageos/pkg/config"
)

func newTestAppRequestTransport(t *testing.T) *AppRequestTransport {
	t.Helper()
	appDatabaseService, err := newTestAppDatabaseService()
	if err != nil {
		t.Fatalf("NewAppDatabaseService: %v", err)
	}
	return &AppRequestTransport{appDatabaseService: appDatabaseService}
}

func newTestAppDatabaseService() (*service.AppDatabaseService, error) {
	return service.NewAppDatabaseService(nil, appconfig.AppDatabaseConfig{
		Enabled:       true,
		Dialect:       "mysql",
		Host:          "127.0.0.1",
		Port:          3306,
		AdminUser:     "root",
		AdminPassword: "password",
		SecretKey:     "test-secret",
	})
}

func TestAppDatabaseCapabilityUsesCallbackTargetRouter(t *testing.T) {
	envelope := struct {
		Method string `json:"method"`
		Router string `json:"router"`
		Body   []byte `json:"body"`
		Type   string `json:"type"`
	}{
		Method: "POST",
		Router: "sales/leads.table",
		Body:   []byte(`{"name":"Ada"}`),
		Type:   "OnTableAddRow",
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	req := &dto.RequestAppReq{
		User:         "alice",
		App:          "crm",
		Version:      "v1",
		Method:       "POST",
		Router:       "/_callback",
		TargetRouter: "sales/leads.table",
		Body:         body,
	}
	msg, err := appinvoke.BuildRuntimeRequestMsg(req)
	if err != nil {
		t.Fatalf("BuildRuntimeRequestMsg: %v", err)
	}
	meta, err := appinvoke.ParseRuntimeRequest(msg)
	if err != nil {
		t.Fatalf("ParseRuntimeRequest: %v", err)
	}

	out, err := newTestAppRequestTransport(t).withAppDatabaseCapability(meta, msg.Data)
	if err != nil {
		t.Fatalf("withAppDatabaseCapability: %v", err)
	}

	var got dto.RequestAppReq
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if got.Router != "/_callback" {
		t.Fatalf("request router = %q, want /_callback", got.Router)
	}
	if got.DBCapability == nil {
		t.Fatal("DBCapability is nil")
	}
	if got.DBCapability.Router != "sales/leads.table" {
		t.Fatalf("capability router = %q, want sales/leads.table", got.DBCapability.Router)
	}
}

func TestAppDatabaseCapabilityUsesDirectRouter(t *testing.T) {
	req := &dto.RequestAppReq{
		User:    "alice",
		App:     "crm",
		Version: "v1",
		Method:  "POST",
		Router:  "sales/leads.form",
		Body:    []byte(`{"name":"Ada"}`),
	}
	msg, err := appinvoke.BuildRuntimeRequestMsg(req)
	if err != nil {
		t.Fatalf("BuildRuntimeRequestMsg: %v", err)
	}
	meta, err := appinvoke.ParseRuntimeRequest(msg)
	if err != nil {
		t.Fatalf("ParseRuntimeRequest: %v", err)
	}

	out, err := newTestAppRequestTransport(t).withAppDatabaseCapability(meta, msg.Data)
	if err != nil {
		t.Fatalf("withAppDatabaseCapability: %v", err)
	}

	var got dto.RequestAppReq
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if got.DBCapability == nil {
		t.Fatal("DBCapability is nil")
	}
	if got.DBCapability.Router != "sales/leads.form" {
		t.Fatalf("capability router = %q, want sales/leads.form", got.DBCapability.Router)
	}
}

func TestAppDatabaseCapabilityRejectsCallbackWithoutTargetRouter(t *testing.T) {
	req := &dto.RequestAppReq{
		User:    "alice",
		App:     "crm",
		Version: "v1",
		Method:  "POST",
		Router:  "/_callback",
		Body:    []byte(`{"type":"OnTableAddRow"}`),
	}
	msg, err := appinvoke.BuildRuntimeRequestMsg(req)
	if err != nil {
		t.Fatalf("BuildRuntimeRequestMsg: %v", err)
	}
	meta, err := appinvoke.ParseRuntimeRequest(msg)
	if err != nil {
		t.Fatalf("ParseRuntimeRequest: %v", err)
	}

	if _, err := newTestAppRequestTransport(t).withAppDatabaseCapability(meta, msg.Data); err == nil {
		t.Fatal("expected missing callback target router error")
	}
}
