package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/kageos/kageos-sdk/agent-app/widget"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/functionschema"
)

func TestBusinessRejectionKeepsReasonButNotLegacySQL(t *testing.T) {
	for _, reason := range []string{"不能删除进行中的会议，请等待会议结束后再删除", "开始时间不能是过去时间", "当前记录无权删除"} {
		got := publicToolBackendError(context.Background(), "delete", fmt.Errorf("wrapped: %w", &apicall.BusinessError{Code: -1, Message: reason}))
		if got != "business_rejected："+reason {
			t.Fatalf("lost business refusal: %s", got)
		}
	}
	for _, reason := range []string{"Error 1054 (42S22): Unknown column 'status' in 'field list'", "open /srv/private/main.go: denied", "SELECT password FROM users"} {
		got := publicToolBackendError(context.Background(), "update", &apicall.BusinessError{Code: -1, Message: reason})
		if !strings.HasPrefix(got, "service_error：") || strings.Contains(got, reason) {
			t.Fatalf("leaked internal error: %s", got)
		}
	}
}

func TestWriteValidationRejectsHiddenAndUnknownFields(t *testing.T) {
	status := testRunWriteField("status", "预约状态", widget.TypeInput, nil, "")
	status.Hide = &widget.FieldHide{Scenes: []string{functionschema.SceneCreate, functionschema.SceneUpdate}}
	title := testRunWriteField("title", "主题", widget.TypeInput, nil, "")
	fields := editableRunWriteFields([]*widget.Field{status, title}, functionschema.SceneUpdate)
	for _, body := range []map[string]interface{}{{"status": "已结束"}, {"invented": "x"}, {"title": "会议", "status": "已结束"}} {
		issues := validateRunWritePayloads(context.Background(), fields, []runWriteValidationPayload{{Label: "[0]", Body: body}}, false, runWriteValidationOptions{})
		got := formatRunWriteValidationFailure(context.Background(), "run_table_update", issues)
		if len(issues) == 0 || !strings.Contains(got, "不可写字段") || !strings.Contains(got, "本次未提交任何数据") {
			t.Fatalf("unknown fields accepted: %s", got)
		}
	}
	if issues := validateRunWritePayloads(context.Background(), fields, []runWriteValidationPayload{{Body: map[string]interface{}{"title": "会议"}}}, false, runWriteValidationOptions{}); len(issues) != 0 {
		t.Fatalf("valid field rejected: %#v", issues)
	}
	if issues := validateRunWritePayloads(context.Background(), nil, []runWriteValidationPayload{{Body: map[string]interface{}{"status": "已结束"}}}, false, runWriteValidationOptions{}); len(issues) != 1 {
		t.Fatal("empty writable field list must reject nonempty input")
	}
}

func TestTableBatchResultReportsPartialAndTotalFailures(t *testing.T) {
	for _, tc := range []struct {
		success, failed int
		status          string
		isError         bool
	}{{2, 0, "success", false}, {1, 1, "partial_success", true}, {0, 2, "failed", true}} {
		out := map[string]interface{}{"updated_count": tc.success, "failed_count": tc.failed, "errors": []string{"reason"}}
		result := tableBatchWriteResult(out, tc.success, tc.failed, "")
		if result.IsError != tc.isError || out["status"] != tc.status || !strings.Contains(result.Content, "updated_count") || !strings.Contains(result.Content, "reason") {
			t.Fatalf("incorrect batch result: %#v", result)
		}
	}
}
