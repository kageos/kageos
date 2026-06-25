package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos-sdk/agent-app/widget"
	"github.com/kageos/kageos/pkg/contextx"
)

func TestValidateRunWritePayloadsRejectsUnknownUsers(t *testing.T) {
	ctx := contextx.WithRequestUser(context.Background(), "beiluo")
	fields := []*widget.Field{
		testRunWriteField("owner", "负责人", widget.TypeUser, nil, "required"),
		testRunWriteField("reviewers", "评审人", widget.TypeUsers, nil, ""),
	}
	payloads := []runWriteValidationPayload{{
		Body: map[string]interface{}{
			"owner":     "zhangsan",
			"reviewers": "beiluo,test_user",
		},
	}}

	issues := validateRunWritePayloads(ctx, fields, payloads, true, runWriteValidationOptions{ResolveUsers: fakeRunWriteUsers("beiluo")})
	joined := joinRunWriteIssueMessages(issues)
	for _, want := range []string{`负责人 (owner) 包含不存在`, `zhangsan`, `评审人 (reviewers) 包含不存在`, `test_user`} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected user validation issue %q, got:\n%s", want, joined)
		}
	}

	msg := formatRunWriteValidationFailure(ctx, "run_form_submit", issues)
	for _, want := range []string{"本次未提交任何数据", "【用户字段】", "用户字段: 使用真实 username", "当前请求用户 beiluo"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected failure guidance %q, got:\n%s", want, msg)
		}
	}
}

func TestValidateRunWritePayloadsRejectsRequiredAndEnumViolations(t *testing.T) {
	fields := []*widget.Field{
		testRunWriteField("title", "标题", widget.TypeInput, nil, "required"),
		testRunWriteField("status", "状态", widget.TypeSelect, map[string]interface{}{
			"options": []interface{}{"待处理", "已完成"},
		}, "required"),
	}
	payloads := []runWriteValidationPayload{{
		Label: "[0]",
		Body: map[string]interface{}{
			"status": "随便写",
		},
	}}

	issues := validateRunWritePayloads(context.Background(), fields, payloads, true, runWriteValidationOptions{})
	joined := joinRunWriteIssueMessages(issues)
	for _, want := range []string{"标题 ([0].title) 必填", `状态 ([0].status) 的值 "随便写" 不在允许选项内`} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected validation issue %q, got:\n%s", want, joined)
		}
	}

	msg := formatRunWriteValidationFailure(context.Background(), "run_table_create", issues)
	for _, want := range []string{"【必填字段】", "【静态选项】", "- 必填字段: 补齐非空值。", "- 静态选项: 只能填 schema options 中的值。"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected grouped guidance %q, got:\n%s", want, msg)
		}
	}
}

func TestValidateRunWritePayloadsRejectsCleanedEmptyRequiredValues(t *testing.T) {
	fields := []*widget.Field{
		testRunWriteField("title", "标题", widget.TypeInput, nil, "required"),
		testRunWriteField("status", "状态", widget.TypeSelect, map[string]interface{}{
			"options": []interface{}{"待处理", "已完成"},
		}, "required"),
	}
	payloads := []runWriteValidationPayload{{
		Body: map[string]interface{}{
			"title":  []interface{}{" ", ""},
			"status": []string{" "},
		},
	}}

	issues := validateRunWritePayloads(context.Background(), fields, payloads, true, runWriteValidationOptions{})
	joined := joinRunWriteIssueMessages(issues)
	for _, want := range []string{"标题 (title) 必填", "状态 (status) 必填"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected cleaned empty validation issue %q, got:\n%s", want, joined)
		}
	}
}

func TestValidateRunWritePayloadsRejectsMultipleValuesForSingleUserField(t *testing.T) {
	fields := []*widget.Field{
		testRunWriteField("owner", "负责人", widget.TypeUser, nil, "required"),
	}
	payloads := []runWriteValidationPayload{{
		Body: map[string]interface{}{
			"owner": "beiluo,test_user",
		},
	}}

	issues := validateRunWritePayloads(context.Background(), fields, payloads, true, runWriteValidationOptions{ResolveUsers: fakeRunWriteUsers("beiluo", "test_user")})
	joined := joinRunWriteIssueMessages(issues)
	if !strings.Contains(joined, "负责人 (owner) 是单用户字段，只能传一个真实 username") {
		t.Fatalf("expected single user validation issue, got:\n%s", joined)
	}
}

func TestValidateRunWritePayloadsRejectsMissingOnSelectFuzzyValue(t *testing.T) {
	productField := testRunWriteField("product_id", "商品", widget.TypeSelect, nil, "required")
	productField.Callbacks = []string{"OnSelectFuzzy"}
	productField.Data = &widget.FieldData{Type: widget.DataTypeInt}
	fields := []*widget.Field{productField}
	payloads := []runWriteValidationPayload{{
		Body: map[string]interface{}{
			"product_id": 1001,
		},
	}}

	var gotBody map[string]interface{}
	resolver := func(_ context.Context, fullCodePath string, body map[string]interface{}) (map[string]interface{}, error) {
		if fullCodePath != "/beiluo/shop/products.form" {
			t.Fatalf("fullCodePath = %q", fullCodePath)
		}
		gotBody = body
		return map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"value": 2002, "label": "另一个商品"},
			},
		}, nil
	}

	issues := validateRunWritePayloads(context.Background(), fields, payloads, true, runWriteValidationOptions{
		FullCodePath:       "/beiluo/shop/products.form",
		ResolveFuzzyChoice: resolver,
	})
	if gotBody["type"] != "by_value" {
		t.Fatalf("fuzzy query type = %v, want by_value", gotBody["type"])
	}
	if gotBody["value_type"] != widget.DataTypeInt {
		t.Fatalf("fuzzy value_type = %v, want %s", gotBody["value_type"], widget.DataTypeInt)
	}
	joined := joinRunWriteIssueMessages(issues)
	for _, want := range []string{"商品 (product_id) 的值", "1001", "OnSelectFuzzy 候选", "run_on_select_fuzzy"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected fuzzy validation issue %q, got:\n%s", want, joined)
		}
	}

	msg := formatRunWriteValidationFailure(contextx.WithRequestUser(context.Background(), "system"), "run_form_submit", issues)
	for _, want := range []string{"【动态选项】", "动态选项:", "items[].value", "by_value/by_values"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected fuzzy guidance %q, got:\n%s", want, msg)
		}
	}
	for _, noise := range []string{"user/users 字段", "zhangsan", "test_user", "当前请求用户 system"} {
		if strings.Contains(msg, noise) {
			t.Fatalf("fuzzy guidance should not contain user noise %q, got:\n%s", noise, msg)
		}
	}
}

func TestValidateRunWritePayloadsAcceptsOnSelectFuzzyMultiSelectValues(t *testing.T) {
	productField := testRunWriteField("product_ids", "商品", widget.TypeMultiSelect, nil, "required")
	productField.Callbacks = []string{"OnSelectFuzzy"}
	productField.Data = &widget.FieldData{Type: widget.DataTypeInts}
	fields := []*widget.Field{productField}
	payloads := []runWriteValidationPayload{{
		Body: map[string]interface{}{
			"product_ids": []interface{}{float64(1001), float64(1002)},
		},
	}}

	callCount := 0
	resolver := func(_ context.Context, _ string, body map[string]interface{}) (map[string]interface{}, error) {
		callCount++
		if body["type"] != "by_values" {
			t.Fatalf("fuzzy query type = %v, want by_values", body["type"])
		}
		if body["value_type"] != widget.DataTypeInts {
			t.Fatalf("fuzzy value_type = %v, want %s", body["value_type"], widget.DataTypeInts)
		}
		return map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"value": float64(1001), "label": "商品A"},
				map[string]interface{}{"value": float64(1002), "label": "商品B"},
			},
		}, nil
	}

	issues := validateRunWritePayloads(context.Background(), fields, payloads, true, runWriteValidationOptions{
		FullCodePath:       "/beiluo/shop/products.form",
		ResolveFuzzyChoice: resolver,
	})
	if len(issues) != 0 {
		t.Fatalf("expected no validation issues, got %v", issues)
	}
	if callCount != 1 {
		t.Fatalf("expected one fuzzy resolver call, got %d", callCount)
	}
}

func TestValidateRunWritePayloadsKeepsCheckboxStatic(t *testing.T) {
	field := testRunWriteField("tags", "标签", widget.TypeCheckbox, map[string]interface{}{
		"options": []interface{}{"前端", "后端"},
	}, "required")
	field.Callbacks = []string{"OnSelectFuzzy"}
	fields := []*widget.Field{field}
	payloads := []runWriteValidationPayload{{
		Body: map[string]interface{}{
			"tags": []interface{}{"测试"},
		},
	}}

	resolver := func(context.Context, string, map[string]interface{}) (map[string]interface{}, error) {
		t.Fatal("checkbox should not call OnSelectFuzzy validator")
		return nil, nil
	}
	issues := validateRunWritePayloads(context.Background(), fields, payloads, true, runWriteValidationOptions{
		FullCodePath:       "/beiluo/shop/products.form",
		ResolveFuzzyChoice: resolver,
	})
	joined := joinRunWriteIssueMessages(issues)
	if !strings.Contains(joined, `标签 (tags) 的值 "测试" 不在允许选项内`) {
		t.Fatalf("expected static checkbox options issue, got:\n%s", joined)
	}
}

func TestValidateRunWritePayloadsReportsOnSelectFuzzyProtocolFailure(t *testing.T) {
	field := testRunWriteField("member_id", "会员", widget.TypeSelect, nil, "required")
	field.Callbacks = []string{"OnSelectFuzzy"}
	fields := []*widget.Field{field}
	payloads := []runWriteValidationPayload{{
		Body: map[string]interface{}{
			"member_id": "m_001",
		},
	}}
	resolver := func(context.Context, string, map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"error_msg": "unsupported type by_value"}, nil
	}

	issues := validateRunWritePayloads(context.Background(), fields, payloads, true, runWriteValidationOptions{
		FullCodePath:       "/beiluo/shop/members.form",
		ResolveFuzzyChoice: resolver,
	})
	joined := joinRunWriteIssueMessages(issues)
	for _, want := range []string{"写入前校验失败", "by_value/by_values", "修复回调协议"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected protocol issue %q, got:\n%s", want, joined)
		}
	}
}

func TestValidateRunWritePayloadsReportsAllIssueKindsInOnePass(t *testing.T) {
	itemField := testRunWriteField("item_id", "拍品", widget.TypeSelect, nil, "required")
	itemField.Callbacks = []string{"OnSelectFuzzy"}
	ownerField := testRunWriteField("owner", "负责人", widget.TypeUser, nil, "required")
	fields := []*widget.Field{itemField, ownerField}
	payloads := []runWriteValidationPayload{{
		Body: map[string]interface{}{
			"item_id": 999,
			"owner":   "zhangsan",
		},
	}}
	fuzzyResolver := func(context.Context, string, map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{"items": []interface{}{}}, nil
	}

	issues := validateRunWritePayloads(context.Background(), fields, payloads, true, runWriteValidationOptions{
		FullCodePath:       "/system/x_world/auction/bid_submit.form",
		ResolveFuzzyChoice: fuzzyResolver,
		ResolveUsers:       fakeRunWriteUsers("system"),
	})
	joined := joinRunWriteIssueMessages(issues)
	for _, want := range []string{"拍品 (item_id) 的值", "负责人 (owner) 包含不存在"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected issue %q, got:\n%s", want, joined)
		}
	}

	msg := formatRunWriteValidationFailure(contextx.WithRequestUser(context.Background(), "system"), "run_form_submit", issues)
	for _, want := range []string{"【动态选项】", "【用户字段】", "- 动态选项:", "- 用户字段:"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected grouped issue/guidance %q, got:\n%s", want, msg)
		}
	}
}

func TestValidateRunWritePayloadsBatchesUserResolverOnce(t *testing.T) {
	fields := []*widget.Field{
		testRunWriteField("owner", "负责人", widget.TypeUser, nil, "required"),
		testRunWriteField("reviewers", "评审人", widget.TypeUsers, nil, ""),
	}
	payloads := []runWriteValidationPayload{
		{
			Label: "[0]",
			Body: map[string]interface{}{
				"owner":     "beiluo",
				"reviewers": "test_user,eric",
			},
		},
		{
			Label: "[1]",
			Body: map[string]interface{}{
				"owner":     "eric",
				"reviewers": "beiluo",
			},
		},
	}
	callCount := 0
	var requested []string
	resolver := func(_ context.Context, usernames []string) (map[string]struct{}, error) {
		callCount++
		requested = append([]string(nil), usernames...)
		out := make(map[string]struct{}, len(usernames))
		for _, username := range usernames {
			out[username] = struct{}{}
		}
		return out, nil
	}

	issues := validateRunWritePayloads(context.Background(), fields, payloads, true, runWriteValidationOptions{ResolveUsers: resolver})
	if len(issues) != 0 {
		t.Fatalf("expected no validation issues, got %v", issues)
	}
	if callCount != 1 {
		t.Fatalf("expected one batched user resolver call, got %d", callCount)
	}
	if got, want := strings.Join(requested, ","), "beiluo,eric,test_user"; got != want {
		t.Fatalf("batched usernames = %q, want %q", got, want)
	}
}

func TestValidateRunWritePayloadsSkipsEmptyOptionalUserFields(t *testing.T) {
	fields := []*widget.Field{
		testRunWriteField("owner", "负责人", widget.TypeUser, nil, ""),
		testRunWriteField("reviewers", "评审人", widget.TypeUsers, nil, ""),
	}
	payloads := []runWriteValidationPayload{
		{Label: "[0]", Body: map[string]interface{}{}},
		{Label: "[1]", Body: map[string]interface{}{"owner": nil, "reviewers": nil}},
		{Label: "[2]", Body: map[string]interface{}{"owner": "", "reviewers": "  "}},
		{Label: "[3]", Body: map[string]interface{}{"owner": []interface{}{}, "reviewers": []interface{}{"", " "}}},
		{Label: "[4]", Body: map[string]interface{}{"owner": []string{" "}, "reviewers": []string{}}},
	}
	callCount := 0
	resolver := func(context.Context, []string) (map[string]struct{}, error) {
		callCount++
		return map[string]struct{}{}, nil
	}

	issues := validateRunWritePayloads(context.Background(), fields, payloads, true, runWriteValidationOptions{ResolveUsers: resolver})
	if len(issues) != 0 {
		t.Fatalf("expected no validation issues, got %v", issues)
	}
	if callCount != 0 {
		t.Fatalf("expected empty optional user fields to skip user resolver, got %d calls", callCount)
	}
}

func TestValidateRunWritePayloadsSkipsUserResolverForEmptyRequiredUserFields(t *testing.T) {
	fields := []*widget.Field{
		testRunWriteField("owner", "负责人", widget.TypeUser, nil, "required"),
	}
	payloads := []runWriteValidationPayload{{
		Body: map[string]interface{}{"owner": ""},
	}}
	callCount := 0
	resolver := func(context.Context, []string) (map[string]struct{}, error) {
		callCount++
		return map[string]struct{}{}, nil
	}

	issues := validateRunWritePayloads(context.Background(), fields, payloads, true, runWriteValidationOptions{ResolveUsers: resolver})
	joined := joinRunWriteIssueMessages(issues)
	if !strings.Contains(joined, "负责人 (owner) 必填") {
		t.Fatalf("expected required issue, got:\n%s", joined)
	}
	if callCount != 0 {
		t.Fatalf("expected empty required user field to skip user resolver, got %d calls", callCount)
	}
}

func TestValidateRunWritePayloadsAcceptsValidUsersAndEnums(t *testing.T) {
	fields := []*widget.Field{
		testRunWriteField("owner", "负责人", widget.TypeUser, nil, "required"),
		testRunWriteField("status", "状态", widget.TypeSelect, map[string]interface{}{
			"options": []interface{}{"待处理", "已完成"},
		}, "required"),
	}
	payloads := []runWriteValidationPayload{{
		Body: map[string]interface{}{
			"owner":  "beiluo",
			"status": "待处理",
		},
	}}

	issues := validateRunWritePayloads(context.Background(), fields, payloads, true, runWriteValidationOptions{ResolveUsers: fakeRunWriteUsers("beiluo")})
	if len(issues) != 0 {
		t.Fatalf("expected no validation issues, got %v", issues)
	}
}

func testRunWriteField(code, name, widgetType string, config interface{}, validation string) *widget.Field {
	field := &widget.Field{
		Code:       code,
		Name:       name,
		Validation: validation,
	}
	field.Widget.Type = widgetType
	field.Widget.Config = config
	return field
}

func fakeRunWriteUsers(existing ...string) runWriteUserResolver {
	return func(context.Context, []string) (map[string]struct{}, error) {
		out := make(map[string]struct{}, len(existing))
		for _, username := range existing {
			out[username] = struct{}{}
		}
		return out, nil
	}
}

func joinRunWriteIssueMessages(issues []runWriteValidationIssue) string {
	messages := make([]string, 0, len(issues))
	for _, issue := range issues {
		messages = append(messages, issue.Message)
	}
	return strings.Join(messages, "\n")
}
