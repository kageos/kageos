package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/widget"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/functionschema"
)

const (
	runWriteModeFormSubmit  = "form_submit"
	runWriteModeTableCreate = "table_create"
	runWriteModeTableUpdate = "table_update"
)

type runWriteValidationPayload struct {
	Label string
	Body  map[string]interface{}
}

type runWriteFieldValue struct {
	Path    string
	Field   *widget.Field
	Value   interface{}
	Request map[string]interface{}
}

type batchWidgetDataValidator interface {
	WidgetTypes() []string
	ValidateBatch(context.Context, []runWriteFieldValue) []runWriteValidationIssue
}

type runWriteValidationIssueKind string

const (
	runWriteIssueRequired     runWriteValidationIssueKind = "必填字段"
	runWriteIssueStaticChoice runWriteValidationIssueKind = "静态选项"
	runWriteIssueFuzzyChoice  runWriteValidationIssueKind = "动态选项"
	runWriteIssueUser         runWriteValidationIssueKind = "用户字段"
)

type runWriteValidationIssue struct {
	Kind    runWriteValidationIssueKind
	Message string
}

type runWriteUserRef struct {
	Username string
	Path     string
	Field    *widget.Field
}

type runWriteUserResolver func(context.Context, []string) (map[string]struct{}, error)

type runWriteFuzzyChoiceResolver func(context.Context, string, map[string]interface{}) (map[string]interface{}, error)

type runWriteValidationOptions struct {
	FullCodePath       string
	ResolveUsers       runWriteUserResolver
	ResolveFuzzyChoice runWriteFuzzyChoiceResolver
}

func runWritePreflight(ctx context.Context, toolName string, fullCodePath string, funcType string, mode string, payloads []runWriteValidationPayload) string {
	fn, err := apicall.GetFunctionInfo(ctx, funcType, fullCodePath)
	if err != nil {
		return fmt.Sprintf("%s 写入前校验失败：无法获取函数详情: %v。\n【给模型】先用 search(full_code_path=%q, resource_type=\"function\", schema_output=\"both\") 确认函数存在和字段 schema，再重新构造 body。", toolName, err, fullCodePath)
	}
	fields := runWriteFieldsForMode(fn, mode)
	if len(fields) == 0 {
		return ""
	}
	issues := validateRunWritePayloads(ctx, fields, payloads, mode != runWriteModeTableUpdate, runWriteValidationOptions{
		FullCodePath:       fullCodePath,
		ResolveUsers:       resolveRunWriteUsers,
		ResolveFuzzyChoice: resolveRunWriteFuzzyChoice,
	})
	if len(issues) == 0 {
		return ""
	}
	return formatRunWriteValidationFailure(ctx, toolName, issues)
}

func runWriteFieldsForMode(fn *dto.GetFunctionResp, mode string) []*widget.Field {
	if fn == nil || fn.Schema == nil {
		return nil
	}
	switch mode {
	case runWriteModeFormSubmit:
		if fn.Schema.Form == nil {
			return nil
		}
		return fn.Schema.Form.Request
	case runWriteModeTableCreate:
		if fn.Schema.Table == nil {
			return nil
		}
		return editableRunWriteFields(fn.Schema.Table.Fields, functionschema.SceneCreate)
	case runWriteModeTableUpdate:
		if fn.Schema.Table == nil {
			return nil
		}
		return editableRunWriteFields(fn.Schema.Table.Fields, functionschema.SceneUpdate)
	default:
		return nil
	}
}

func editableRunWriteFields(fields []*widget.Field, scene string) []*widget.Field {
	out := make([]*widget.Field, 0, len(fields))
	for _, field := range fields {
		if field == nil || !functionschema.VisibleInScene(field, scene) {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(field.Widget.Type), widget.TypeID) {
			continue
		}
		out = append(out, field)
	}
	return out
}

func validateRunWritePayloads(ctx context.Context, fields []*widget.Field, payloads []runWriteValidationPayload, enforceRequired bool, opts runWriteValidationOptions) []runWriteValidationIssue {
	var issues []runWriteValidationIssue
	validators := runWriteWidgetDataValidators(opts)
	validatorByWidgetType := runWriteWidgetValidatorIndex(validators)
	valuesByValidator := make(map[int][]runWriteFieldValue)

	for _, payload := range payloads {
		for _, field := range fields {
			if field == nil || strings.TrimSpace(field.Code) == "" {
				continue
			}
			path := runWriteFieldPath(payload.Label, field.Code)
			value, exists := payload.Body[field.Code]
			if enforceRequired && isRunWriteRequiredField(field) && (!exists || isRunWriteEmptyValue(value)) {
				issues = append(issues, runWriteValidationIssue{
					Kind:    runWriteIssueRequired,
					Message: fmt.Sprintf("%s (%s) 必填，不能省略或传空值。", runWriteFieldDisplayName(field), path),
				})
				continue
			}
			if !exists || isRunWriteEmptyValue(value) {
				continue
			}
			widgetType := strings.TrimSpace(field.Widget.Type)
			if validatorIndex, ok := validatorByWidgetType[widgetType]; ok {
				valuesByValidator[validatorIndex] = append(valuesByValidator[validatorIndex], runWriteFieldValue{
					Path:    path,
					Field:   field,
					Value:   value,
					Request: payload.Body,
				})
			}
		}
	}

	for i, validator := range validators {
		items := valuesByValidator[i]
		if len(items) == 0 {
			continue
		}
		issues = append(issues, validator.ValidateBatch(ctx, items)...)
	}

	return issues
}

func runWriteWidgetDataValidators(opts runWriteValidationOptions) []batchWidgetDataValidator {
	return []batchWidgetDataValidator{
		runWriteChoiceWidgetValidator{
			fullCodePath:       opts.FullCodePath,
			resolveFuzzyChoice: opts.ResolveFuzzyChoice,
		},
		runWriteUserWidgetValidator{resolveUsers: opts.ResolveUsers},
	}
}

func runWriteWidgetValidatorIndex(validators []batchWidgetDataValidator) map[string]int {
	out := make(map[string]int)
	for i, validator := range validators {
		for _, widgetType := range validator.WidgetTypes() {
			widgetType = strings.TrimSpace(widgetType)
			if widgetType == "" {
				continue
			}
			out[widgetType] = i
		}
	}
	return out
}

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

type runWriteUserWidgetValidator struct {
	resolveUsers runWriteUserResolver
}

func (runWriteUserWidgetValidator) WidgetTypes() []string {
	return []string{widget.TypeUser, widget.TypeUsers}
}

func (v runWriteUserWidgetValidator) ValidateBatch(ctx context.Context, items []runWriteFieldValue) []runWriteValidationIssue {
	var issues []runWriteValidationIssue
	userRefs := make([]runWriteUserRef, 0)
	for _, item := range items {
		values, invalid := runWriteStringValues(item.Value)
		if invalid {
			issues = append(issues, runWriteValidationIssue{
				Kind:    runWriteIssueUser,
				Message: fmt.Sprintf("%s (%s) 是用户字段，必须传真实 username 字符串；多用户字段使用逗号分隔 username。", runWriteFieldDisplayName(item.Field), item.Path),
			})
			continue
		}
		if strings.TrimSpace(item.Field.Widget.Type) == widget.TypeUser && len(values) > 1 {
			issues = append(issues, runWriteValidationIssue{
				Kind:    runWriteIssueUser,
				Message: fmt.Sprintf("%s (%s) 是单用户字段，只能传一个真实 username。", runWriteFieldDisplayName(item.Field), item.Path),
			})
			continue
		}
		for _, username := range values {
			userRefs = append(userRefs, runWriteUserRef{Username: username, Path: item.Path, Field: item.Field})
		}
	}

	if len(userRefs) == 0 || v.resolveUsers == nil {
		return issues
	}
	usernames := uniqueRunWriteUsernames(userRefs)
	existing, err := v.resolveUsers(ctx, usernames)
	if err != nil {
		issues = append(issues, runWriteValidationIssue{
			Kind:    runWriteIssueUser,
			Message: "用户字段校验失败：无法查询平台用户: " + err.Error(),
		})
		return issues
	}
	for _, ref := range userRefs {
		if _, ok := existing[ref.Username]; !ok {
			issues = append(issues, runWriteValidationIssue{
				Kind:    runWriteIssueUser,
				Message: fmt.Sprintf("%s (%s) 包含不存在或当前企业不可见的用户 %q。", runWriteFieldDisplayName(ref.Field), ref.Path, ref.Username),
			})
		}
	}
	return issues
}

func resolveRunWriteUsers(ctx context.Context, usernames []string) (map[string]struct{}, error) {
	users, err := apicall.GetUsersByUsernames(ctx, usernames)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(users))
	for _, user := range users {
		username := strings.TrimSpace(user.Username)
		if username != "" {
			out[username] = struct{}{}
		}
	}
	return out, nil
}

func resolveRunWriteFuzzyChoice(ctx context.Context, fullCodePath string, body map[string]interface{}) (map[string]interface{}, error) {
	return apicall.CallbackOnSelectFuzzy(ctx, fullCodePath, body)
}

func formatRunWriteValidationFailure(ctx context.Context, toolName string, issues []runWriteValidationIssue) string {
	var buf strings.Builder
	buf.WriteString(toolName)
	buf.WriteString(" 写入前校验失败，本次未提交任何数据：\n")
	grouped := groupRunWriteValidationIssues(issues)
	for _, kind := range orderedRunWriteValidationIssueKinds() {
		items := grouped[kind]
		if len(items) == 0 {
			continue
		}
		buf.WriteString("【")
		buf.WriteString(string(kind))
		buf.WriteString("】\n")
		for _, issue := range items {
			buf.WriteString("- ")
			buf.WriteString(issue.Message)
			buf.WriteString("\n")
		}
	}
	buf.WriteString("\n【给模型】\n")
	for _, kind := range orderedRunWriteValidationIssueKinds() {
		if len(grouped[kind]) == 0 {
			continue
		}
		buf.WriteString("- ")
		buf.WriteString(runWriteValidationKindGuidance(ctx, kind))
		buf.WriteString("\n")
	}
	return strings.TrimSpace(buf.String())
}

func orderedRunWriteValidationIssueKinds() []runWriteValidationIssueKind {
	return []runWriteValidationIssueKind{
		runWriteIssueRequired,
		runWriteIssueStaticChoice,
		runWriteIssueFuzzyChoice,
		runWriteIssueUser,
	}
}

func groupRunWriteValidationIssues(issues []runWriteValidationIssue) map[runWriteValidationIssueKind][]runWriteValidationIssue {
	grouped := make(map[runWriteValidationIssueKind][]runWriteValidationIssue)
	for _, issue := range issues {
		if issue.Message == "" {
			continue
		}
		grouped[issue.Kind] = append(grouped[issue.Kind], issue)
	}
	return grouped
}

func runWriteValidationKindGuidance(ctx context.Context, kind runWriteValidationIssueKind) string {
	switch kind {
	case runWriteIssueRequired:
		return "必填字段: 补齐非空值。"
	case runWriteIssueStaticChoice:
		return "静态选项: 只能填 schema options 中的值。"
	case runWriteIssueFuzzyChoice:
		return "动态选项: 用 run_on_select_fuzzy 查候选，填 items[].value；不支持 by_value/by_values 就修回调。"
	case runWriteIssueUser:
		if user := strings.TrimSpace(contextx.GetRequestUser(ctx)); user != "" {
			return "用户字段: 使用真实 username；测试时优先用当前请求用户 " + user + "。"
		}
		return "用户字段: 使用真实 username，不要填示例名或函数名。"
	default:
		return "按字段 schema 修正后重试。"
	}
}

func isRunWriteRequiredField(field *widget.Field) bool {
	for _, part := range strings.Split(field.Validation, ",") {
		part = strings.TrimSpace(part)
		if part == "required" {
			return true
		}
	}
	return false
}

func isRunWriteEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []interface{}:
		if len(v) == 0 {
			return true
		}
		values, invalid := runWriteStringValues(v)
		return !invalid && len(values) == 0
	case []string:
		return len(cleanRunWriteStrings(v)) == 0
	default:
		return false
	}
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

func isRunWriteUserWidget(field *widget.Field) bool {
	switch strings.TrimSpace(field.Widget.Type) {
	case widget.TypeUser, widget.TypeUsers:
		return true
	default:
		return false
	}
}

func runWriteStringValues(value interface{}) ([]string, bool) {
	switch v := value.(type) {
	case string:
		return splitRunWriteCSV(v), false
	case []string:
		return cleanRunWriteStrings(v), false
	case []interface{}:
		values := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, true
			}
			values = append(values, s)
		}
		return cleanRunWriteStrings(values), false
	default:
		return nil, true
	}
}

func splitRunWriteCSV(value string) []string {
	return cleanRunWriteStrings(strings.Split(value, ","))
}

func cleanRunWriteStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func uniqueRunWriteUsernames(refs []runWriteUserRef) []string {
	seen := map[string]struct{}{}
	for _, ref := range refs {
		username := strings.TrimSpace(ref.Username)
		if username == "" {
			continue
		}
		seen[username] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for username := range seen {
		out = append(out, username)
	}
	sort.Strings(out)
	return out
}

func runWriteFieldPath(label string, code string) string {
	if strings.TrimSpace(label) == "" {
		return code
	}
	return strings.TrimSpace(label) + "." + code
}

func runWriteFieldDisplayName(field *widget.Field) string {
	if field == nil {
		return "字段"
	}
	if strings.TrimSpace(field.Name) != "" {
		return field.Name
	}
	if strings.TrimSpace(field.FieldName) != "" {
		return field.FieldName
	}
	if strings.TrimSpace(field.Code) != "" {
		return field.Code
	}
	return "字段"
}

func runWriteConfigOptions(config interface{}) []string {
	m := runWriteConfigMap(config)
	if len(m) == 0 {
		return nil
	}
	return runWriteInterfaceStrings(m["options"])
}

func runWriteConfigBool(config interface{}, key string) bool {
	m := runWriteConfigMap(config)
	if len(m) == 0 {
		return false
	}
	switch v := m[key].(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}

func runWriteConfigMap(config interface{}) map[string]interface{} {
	if config == nil {
		return nil
	}
	if m, ok := config.(map[string]interface{}); ok {
		return m
	}
	data, err := json.Marshal(config)
	if err != nil {
		return nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
}

func runWriteInterfaceStrings(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return cleanRunWriteStrings(v)
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return cleanRunWriteStrings(out)
	case string:
		return splitRunWriteCSV(v)
	default:
		return nil
	}
}
