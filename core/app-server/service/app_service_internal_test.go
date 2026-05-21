package service

import (
	"context"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
)

func TestSyncUpdatedAppMetadataSkipsWhenDiffMissing(t *testing.T) {
	svc := &AppService{}

	if warning := svc.syncUpdatedAppMetadata(context.Background(), 1, nil); warning != "" {
		t.Fatalf("expected empty warning for nil diff, got %q", warning)
	}
}

func TestFinalizeReleasedAppMetadataRequiresApp(t *testing.T) {
	svc := &AppService{}

	_, err := svc.finalizeReleasedAppMetadata(context.Background(), "test", nil, "alice", "demo", "", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "应用不存在" {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestApplyFunctionNodeMetadataOverwritesStaleFields(t *testing.T) {
	svc := &AppService{}
	tree := &model.ServiceTree{
		Code:         "old_code",
		Name:         "Old Name",
		Description:  "old desc",
		TemplateType: "form",
		RefID:        1,
		Tags:         "legacy,stale",
	}
	api := &dto.ApiInfo{
		Code:         "ticket_list",
		Name:         "工单列表",
		Desc:         `新的描述`,
		TemplateType: "table",
	}

	svc.applyFunctionNodeMetadata(tree, api, 42)

	if tree.Code != "ticket_list" || tree.Name != "工单列表" {
		t.Fatalf("unexpected code/name after apply: %+v", tree)
	}
	if tree.Description != "新的描述" || tree.TemplateType != "table" {
		t.Fatalf("unexpected description/template type after apply: %+v", tree)
	}
	if tree.RefID != 42 {
		t.Fatalf("expected ref id to be updated, got %d", tree.RefID)
	}
	if tree.Tags != "" {
		t.Fatalf("expected stale tags to be cleared, got %q", tree.Tags)
	}
}
