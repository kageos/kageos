package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/widget"
	"github.com/kageos/kageos/pkg/apicall"
)

type runWriteChoiceWidgetValidator struct {
	fullCodePath       string
	resolveFuzzyChoice runWriteFuzzyChoiceResolver
}

func (runWriteChoiceWidgetValidator) WidgetTypes() []string {
	return []string{widget.TypeSelect, widget.TypeRadio, widget.TypeMultiSelect, widget.TypeCheckbox}
}

func (v runWriteChoiceWidgetValidator) ValidateBatch(ctx context.Context, items []runWriteFieldValue) []runWriteValidationIssue {
	issues := make([]runWriteValidationIssue, 0)
	for _, item := range items {
		if issue := validateRunWriteChoiceValue(item.Path, item.Field, item.Value); issue != "" {
			issues = append(issues, runWriteValidationIssue{Kind: runWriteIssueStaticChoice, Message: issue})
			continue
		}
		if issue := v.validateFuzzyChoiceValue(ctx, item); issue.Message != "" {
			issues = append(issues, issue)
		}
	}
	return issues
}

func (v runWriteChoiceWidgetValidator) validateFuzzyChoiceValue(ctx context.Context, item runWriteFieldValue) runWriteValidationIssue {
	if !isRunWriteFuzzyChoiceWidget(item.Field) || runWriteConfigBool(item.Field.Widget.Config, "creatable") {
		return runWriteValidationIssue{}
	}
	if len(runWriteConfigOptions(item.Field.Widget.Config)) > 0 {
		return runWriteValidationIssue{}
	}
	if strings.TrimSpace(v.fullCodePath) == "" || v.resolveFuzzyChoice == nil {
		return runWriteValidationIssue{
			Kind:    runWriteIssueFuzzyChoice,
			Message: fmt.Sprintf("%s (%s) 是 OnSelectFuzzy 动态选项字段，写入前必须通过 by_value/by_values 反查候选；当前缺少校验入口。", runWriteFieldDisplayName(item.Field), item.Path),
		}
	}

	queryType, values, value, invalid := runWriteFuzzyChoiceQueryValue(item.Field, item.Value)
	if invalid {
		return runWriteValidationIssue{
			Kind:    runWriteIssueFuzzyChoice,
			Message: fmt.Sprintf("%s (%s) 是 OnSelectFuzzy 动态选项字段，select 必须传单个值，multiselect 必须传字符串、字符串数组或标量数组。", runWriteFieldDisplayName(item.Field), item.Path),
		}
	}

	body := map[string]interface{}{
		"code":       item.Field.Code,
		"type":       queryType,
		"value":      value,
		"request":    item.Request,
		"value_type": runWriteFieldValueType(item.Field),
	}
	result, err := v.resolveFuzzyChoice(ctx, v.fullCodePath, body)
	if err != nil {
		return runWriteFuzzyChoiceProtocolIssue(item, err.Error())
	}
	if errorMsg := strings.TrimSpace(runWriteFuzzyChoiceErrorMessage(result)); errorMsg != "" {
		return runWriteFuzzyChoiceProtocolIssue(item, errorMsg)
	}
	itemValues := runWriteFuzzyChoiceItemValues(result)
	if len(itemValues) == 0 {
		return runWriteValidationIssue{
			Kind:    runWriteIssueFuzzyChoice,
			Message: fmt.Sprintf("%s (%s) 的值 %q 不在 OnSelectFuzzy 候选内；请先用 run_on_select_fuzzy 按关键词查询真实候选值，不要猜值。", runWriteFieldDisplayName(item.Field), item.Path, runWriteValuesForMessage(values)),
		}
	}
	if missing := runWriteMissingFuzzyChoiceValues(values, itemValues); len(missing) > 0 {
		return runWriteValidationIssue{
			Kind:    runWriteIssueFuzzyChoice,
			Message: fmt.Sprintf("%s (%s) 的值 %q 不在 OnSelectFuzzy 候选内；请先用 run_on_select_fuzzy 按关键词查询真实候选值，不要猜值。", runWriteFieldDisplayName(item.Field), item.Path, runWriteValuesForMessage(missing)),
		}
	}
	return runWriteValidationIssue{}
}

func resolveRunWriteFuzzyChoice(ctx context.Context, fullCodePath string, body map[string]interface{}) (map[string]interface{}, error) {
	return apicall.CallbackOnSelectFuzzy(ctx, fullCodePath, body)
}

func validateRunWriteChoiceValue(path string, field *widget.Field, value interface{}) string {
	widgetType := strings.TrimSpace(field.Widget.Type)
	switch widgetType {
	case widget.TypeSelect, widget.TypeRadio, widget.TypeMultiSelect, widget.TypeCheckbox:
	default:
		return ""
	}
	if runWriteConfigBool(field.Widget.Config, "creatable") {
		return ""
	}
	options := runWriteConfigOptions(field.Widget.Config)
	if len(options) == 0 {
		return ""
	}
	values, invalid := runWriteStringValues(value)
	if invalid {
		return fmt.Sprintf("%s (%s) 必须使用字符串或字符串数组，且值必须来自可选项: %s。", runWriteFieldDisplayName(field), path, strings.Join(options, "、"))
	}
	allowed := make(map[string]struct{}, len(options))
	for _, option := range options {
		allowed[option] = struct{}{}
	}
	var bad []string
	for _, item := range values {
		if _, ok := allowed[item]; !ok {
			bad = append(bad, item)
		}
	}
	if len(bad) == 0 {
		return ""
	}
	return fmt.Sprintf("%s (%s) 的值 %q 不在允许选项内，可选: %s。", runWriteFieldDisplayName(field), path, strings.Join(bad, ","), strings.Join(options, "、"))
}

func isRunWriteFuzzyChoiceWidget(field *widget.Field) bool {
	if field == nil {
		return false
	}
	switch strings.TrimSpace(field.Widget.Type) {
	case widget.TypeSelect, widget.TypeMultiSelect:
	default:
		return false
	}
	for _, callback := range field.Callbacks {
		if strings.TrimSpace(callback) == "OnSelectFuzzy" {
			return true
		}
	}
	return false
}

func runWriteFuzzyChoiceQueryValue(field *widget.Field, value interface{}) (queryType string, values []interface{}, queryValue interface{}, invalid bool) {
	widgetType := strings.TrimSpace(field.Widget.Type)
	switch widgetType {
	case widget.TypeSelect:
		values, invalid = runWriteFuzzyChoiceValues(value)
		if invalid || len(values) != 1 {
			return "", nil, nil, true
		}
		return "by_value", values, values[0], false
	case widget.TypeMultiSelect:
		values, invalid = runWriteFuzzyChoiceValues(value)
		if invalid {
			return "", nil, nil, true
		}
		return "by_values", values, values, false
	default:
		return "", nil, nil, true
	}
}

func runWriteFuzzyChoiceValues(value interface{}) ([]interface{}, bool) {
	switch v := value.(type) {
	case string:
		parts := splitRunWriteCSV(v)
		out := make([]interface{}, 0, len(parts))
		for _, part := range parts {
			out = append(out, part)
		}
		return out, false
	case []string:
		values := cleanRunWriteStrings(v)
		out := make([]interface{}, 0, len(values))
		for _, item := range values {
			out = append(out, item)
		}
		return out, false
	case []interface{}:
		out := make([]interface{}, 0, len(v))
		for _, item := range v {
			if isRunWriteEmptyValue(item) {
				continue
			}
			out = append(out, item)
		}
		return out, false
	default:
		if value == nil {
			return nil, false
		}
		return []interface{}{value}, false
	}
}

func runWriteFieldValueType(field *widget.Field) string {
	if field != nil && field.Data != nil && strings.TrimSpace(field.Data.Type) != "" {
		return strings.TrimSpace(field.Data.Type)
	}
	if field != nil && strings.TrimSpace(field.Widget.Type) == widget.TypeMultiSelect {
		return widget.DataTypeStrings
	}
	return widget.DataTypeString
}

func runWriteFuzzyChoiceProtocolIssue(item runWriteFieldValue, detail string) runWriteValidationIssue {
	return runWriteValidationIssue{
		Kind:    runWriteIssueFuzzyChoice,
		Message: fmt.Sprintf("%s (%s) 是 OnSelectFuzzy 动态选项字段，写入前校验失败：%s。该字段回调必须支持 by_value/by_values 反查；请先用 run_on_select_fuzzy 按关键词查询真实候选，或修复回调协议后再写入。", runWriteFieldDisplayName(item.Field), item.Path, detail),
	}
}

func runWriteFuzzyChoiceErrorMessage(result map[string]interface{}) string {
	if result == nil {
		return ""
	}
	if msg, ok := result["error_msg"].(string); ok {
		return msg
	}
	return ""
}

func runWriteFuzzyChoiceItemValues(result map[string]interface{}) []interface{} {
	items, ok := result["items"]
	if !ok || items == nil {
		return nil
	}
	switch v := items.(type) {
	case []interface{}:
		out := make([]interface{}, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, m["value"])
			}
		}
		return out
	case []map[string]interface{}:
		out := make([]interface{}, 0, len(v))
		for _, item := range v {
			out = append(out, item["value"])
		}
		return out
	default:
		return nil
	}
}

func runWriteMissingFuzzyChoiceValues(values []interface{}, itemValues []interface{}) []interface{} {
	allowed := make(map[string]struct{}, len(itemValues))
	for _, value := range itemValues {
		allowed[runWriteComparableValue(value)] = struct{}{}
	}
	var missing []interface{}
	for _, value := range values {
		if _, ok := allowed[runWriteComparableValue(value)]; !ok {
			missing = append(missing, value)
		}
	}
	return missing
}

func runWriteComparableValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		data, err := json.Marshal(v)
		if err == nil {
			return string(data)
		}
		return fmt.Sprintf("%v", v)
	}
}

func runWriteValuesForMessage(values []interface{}) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%v", value))
	}
	return strings.Join(parts, ",")
}
