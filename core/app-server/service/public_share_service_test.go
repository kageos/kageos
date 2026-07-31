package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPublicShareListSubmissionsOnlyReturnsCurrentAnonymousActorAndShare(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.OperateLog{}); err != nil {
		t.Fatalf("migrate operate logs: %v", err)
	}

	repo := repository.NewOperateLogRepository(db)
	svc := NewPublicShareService(nil, nil, nil, repo)
	ctx := context.Background()
	share := &model.PublicShare{
		ShareID:      "ps_target",
		TenantUser:   "owner",
		App:          "ops",
		FullCodePath: "/owner/ops/report.form",
	}
	targetActor := "guest_anon_target"

	createLog := func(actor, resourcePath, sourceType, sourceRef string, details json.RawMessage) {
		t.Helper()
		if err := repo.CreateOperateLog(ctx, &model.OperateLog{
			TenantUser:   "owner",
			App:          "ops",
			ActorUser:    actor,
			Action:       "form_submit",
			ResourceType: "form",
			ResourcePath: resourcePath,
			Summary:      "公开表单提交成功",
			DetailsJSON:  details,
			Status:       "success",
			SourceType:   sourceType,
			SourceRef:    sourceRef,
			TraceID:      "trace-" + actor + "-" + sourceRef,
		}); err != nil {
			t.Fatalf("create operate log: %v", err)
		}
	}

	createLog(
		targetActor,
		"/owner/ops/report.form",
		contextx.SourceTypePublicShare,
		share.ShareID,
		json.RawMessage(`{"duration_millis":18,"request_body":{"name":"Alice"},"response_body":{"code":0}}`),
	)
	createLog(
		"guest_anon_other",
		"/owner/ops/report.form",
		contextx.SourceTypePublicShare,
		share.ShareID,
		json.RawMessage(`{"request_body":{"name":"Other actor"}}`),
	)
	createLog(
		targetActor,
		"/owner/ops/report.form",
		contextx.SourceTypePublicShare,
		"ps_other",
		json.RawMessage(`{"request_body":{"name":"Other share"}}`),
	)
	createLog(
		targetActor,
		"/owner/ops/other.form",
		contextx.SourceTypePublicShare,
		share.ShareID,
		json.RawMessage(`{"request_body":{"name":"Other route"}}`),
	)

	resp, err := svc.ListSubmissions(ctx, share, targetActor, 1, 20)
	if err != nil {
		t.Fatalf("ListSubmissions() error = %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("expected one isolated submission, total=%d items=%+v", resp.Total, resp.Items)
	}
	item := resp.Items[0]
	if item.DurationMillis != 18 {
		t.Fatalf("duration_millis = %d, want 18", item.DurationMillis)
	}
	request, ok := item.RequestBody.(map[string]interface{})
	if !ok || request["name"] != "Alice" {
		t.Fatalf("request_body = %#v, want current actor payload", item.RequestBody)
	}
	if item.TraceID == "" || item.CreatedAt == "" {
		t.Fatalf("trace_id and created_at should be available: %+v", item)
	}
}
