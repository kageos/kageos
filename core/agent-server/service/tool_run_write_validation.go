package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/widget"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
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
