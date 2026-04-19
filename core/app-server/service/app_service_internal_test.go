package service

import (
	"context"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
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

func TestExtractVersionNum(t *testing.T) {
	tests := []struct {
		version string
		want    int
	}{
		{version: "", want: 0},
		{version: "v1", want: 1},
		{version: "V20", want: 20},
		{version: "3", want: 3},
		{version: "vx", want: 0},
	}

	for _, tt := range tests {
		if got := extractVersionNum(tt.version); got != tt.want {
			t.Fatalf("version %q: want %d, got %d", tt.version, tt.want, got)
		}
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
		Desc:         "新的描述",
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
