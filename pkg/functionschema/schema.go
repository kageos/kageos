package functionschema

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/widget"
)

const Version = 1

const (
	TypeForm  = "form"
	TypeTable = "table"
	TypeChart = "chart"
)

const (
	// SceneList 表示表格列表场景。
	SceneList = "list"
	// SceneCreate 表示新增表单场景。
	SceneCreate = "create"
	// SceneUpdate 表示编辑表单场景。
	SceneUpdate = "update"
)

type FunctionSchema struct {
	Version   int          `json:"version"`
	Type      string       `json:"type"`
	Form      *FormSchema  `json:"form,omitempty"`
	Table     *TableSchema `json:"table,omitempty"`
	Chart     *ChartSchema `json:"chart,omitempty"`
	Callbacks []string     `json:"callbacks,omitempty"`
}

type FormSchema struct {
	Request  []*widget.Field `json:"request"`
	Response []*widget.Field `json:"response"`
}

type TableSchema struct {
	Request []*widget.Field `json:"request"`
	Fields  []*widget.Field `json:"fields"`
}

type ChartSchema struct {
	ChartType string          `json:"chart_type,omitempty"`
	Request   []*widget.Field `json:"request"`
	Response  []*widget.Field `json:"response,omitempty"`
}

func NewForm(request, response []*widget.Field, callbacks []string) *FunctionSchema {
	return &FunctionSchema{
		Version: Version,
		Type:    TypeForm,
		Form: &FormSchema{
			Request:  nonNilFields(request),
			Response: nonNilFields(response),
		},
		Callbacks: normalizeCallbacks(callbacks),
	}
}

func NewTable(request, fields []*widget.Field, callbacks []string) *FunctionSchema {
	return &FunctionSchema{
		Version: Version,
		Type:    TypeTable,
		Table: &TableSchema{
			Request: nonNilFields(request),
			Fields:  nonNilFields(fields),
		},
		Callbacks: normalizeCallbacks(callbacks),
	}
}

func NewChart(request, response []*widget.Field, callbacks []string) *FunctionSchema {
	return NewChartWithType("", request, response, callbacks)
}

func NewChartWithType(chartType string, request, response []*widget.Field, callbacks []string) *FunctionSchema {
	return &FunctionSchema{
		Version: Version,
		Type:    TypeChart,
		Chart: &ChartSchema{
			ChartType: strings.TrimSpace(chartType),
			Request:   nonNilFields(request),
			Response:  nonNilFields(response),
		},
		Callbacks: normalizeCallbacks(callbacks),
	}
}

func Marshal(schema *FunctionSchema) (json.RawMessage, error) {
	if err := Validate(schema); err != nil {
		return nil, err
	}
	data, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func Parse(raw json.RawMessage) (*FunctionSchema, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("function schema is empty")
	}
	var schema FunctionSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, err
	}
	if err := Validate(&schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

func Validate(schema *FunctionSchema) error {
	if schema == nil {
		return fmt.Errorf("function schema is nil")
	}
	normalizeSchema(schema)
	var errs []error
	if schema.Version != Version {
		errs = append(errs, fmt.Errorf("unsupported function schema version: %d", schema.Version))
	}
	switch schema.Type {
	case TypeForm:
		if schema.Form == nil {
			errs = append(errs, fmt.Errorf("form schema is required"))
			break
		}
		if err := validateFields(append(nonNilFields(schema.Form.Request), nonNilFields(schema.Form.Response)...)); err != nil {
			errs = append(errs, err)
		}
	case TypeTable:
		if schema.Table == nil {
			errs = append(errs, fmt.Errorf("table schema is required"))
			break
		}
		if err := validateFields(append(nonNilFields(schema.Table.Request), nonNilFields(schema.Table.Fields)...)); err != nil {
			errs = append(errs, err)
		}
	case TypeChart:
		if schema.Chart == nil {
			errs = append(errs, fmt.Errorf("chart schema is required"))
			break
		}
		if err := validateFields(append(nonNilFields(schema.Chart.Request), nonNilFields(schema.Chart.Response)...)); err != nil {
			errs = append(errs, err)
		}
	default:
		errs = append(errs, fmt.Errorf("unsupported function schema type: %s", schema.Type))
	}
	return errors.Join(errs...)
}

func normalizeSchema(schema *FunctionSchema) {
	if schema == nil {
		return
	}
	schema.Callbacks = normalizeCallbacks(schema.Callbacks)
	if schema.Form != nil {
		schema.Form.Request = nonNilFields(schema.Form.Request)
		schema.Form.Response = nonNilFields(schema.Form.Response)
		widget.NormalizeFieldCodes(schema.Form.Request)
		widget.NormalizeFieldCodes(schema.Form.Response)
	}
	if schema.Table != nil {
		schema.Table.Request = nonNilFields(schema.Table.Request)
		schema.Table.Fields = nonNilFields(schema.Table.Fields)
		widget.NormalizeFieldCodes(schema.Table.Request)
		widget.NormalizeFieldCodes(schema.Table.Fields)
	}
	if schema.Chart != nil {
		schema.Chart.Request = nonNilFields(schema.Chart.Request)
		schema.Chart.Response = nonNilFields(schema.Chart.Response)
		widget.NormalizeFieldCodes(schema.Chart.Request)
		widget.NormalizeFieldCodes(schema.Chart.Response)
	}
}

func Type(raw json.RawMessage) string {
	schema, err := Parse(raw)
	if err != nil {
		return ""
	}
	return schema.Type
}

func Callbacks(raw json.RawMessage) []string {
	schema, err := Parse(raw)
	if err != nil {
		return nil
	}
	return normalizeCallbacks(schema.Callbacks)
}

func HasCallback(raw json.RawMessage, target string) bool {
	for _, callback := range Callbacks(raw) {
		if callback == target {
			return true
		}
	}
	return false
}

func FormRequestFields(raw json.RawMessage) []*widget.Field {
	schema, err := Parse(raw)
	if err != nil || schema.Form == nil {
		return nil
	}
	return nonNilFields(schema.Form.Request)
}

func FormResponseFields(raw json.RawMessage) []*widget.Field {
	schema, err := Parse(raw)
	if err != nil || schema.Form == nil {
		return nil
	}
	return nonNilFields(schema.Form.Response)
}

func ChartRequestFields(raw json.RawMessage) []*widget.Field {
	schema, err := Parse(raw)
	if err != nil || schema.Chart == nil {
		return nil
	}
	return nonNilFields(schema.Chart.Request)
}

func TableRequestFields(raw json.RawMessage) []*widget.Field {
	schema, err := Parse(raw)
	if err != nil || schema.Table == nil {
		return nil
	}
	return nonNilFields(schema.Table.Request)
}

func TableListFields(raw json.RawMessage) []*widget.Field {
	schema, err := Parse(raw)
	if err != nil || schema.Table == nil {
		return nil
	}
	return filterVisibleInScene(schema.Table.Fields, SceneList)
}

func TableCreateFields(raw json.RawMessage) []*widget.Field {
	schema, err := Parse(raw)
	if err != nil || schema.Table == nil {
		return nil
	}
	return filterEditableFields(filterVisibleInScene(schema.Table.Fields, SceneCreate))
}

func TableUpdateFields(raw json.RawMessage) []*widget.Field {
	schema, err := Parse(raw)
	if err != nil || schema.Table == nil {
		return nil
	}
	return filterEditableFields(filterVisibleInScene(schema.Table.Fields, SceneUpdate))
}

func TableSearchFields(raw json.RawMessage) []*widget.Field {
	schema, err := Parse(raw)
	if err != nil || schema.Table == nil {
		return nil
	}
	fields := make([]*widget.Field, 0, len(schema.Table.Request))
	seen := make(map[string]struct{})
	for _, field := range schema.Table.Request {
		if field == nil || strings.TrimSpace(field.Code) == "" {
			continue
		}
		if _, ok := seen[field.Code]; ok {
			continue
		}
		fields = append(fields, field)
	}
	return fields
}

func VisibleInScene(field *widget.Field, scene string) bool {
	if field == nil {
		return false
	}
	// table/form 是表单容器组件，不作为表格列表列渲染。
	if scene == SceneList && isContainerWidget(field) {
		return false
	}
	if field.Hide == nil {
		return true
	}
	for _, item := range field.Hide.Scenes {
		if item == scene {
			return false
		}
	}
	return true
}

func validateFields(fields []*widget.Field) error {
	var errs []error
	for _, field := range fields {
		if field == nil {
			continue
		}
		if err := validateFieldWidget(field); err != nil {
			errs = append(errs, err)
		}
		if err := validateFieldHide(field); err != nil {
			errs = append(errs, err)
		}
		if err := validateFields(field.Children); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func validateFieldWidget(field *widget.Field) error {
	widgetType := strings.TrimSpace(field.Widget.Type)
	if widgetType == "" {
		return nil
	}
	if !widget.IsSupportedType(widgetType) {
		return fmt.Errorf("field %q has unsupported widget type: %s", field.Code, widgetType)
	}
	return nil
}

func validateFieldHide(field *widget.Field) error {
	if field.Hide == nil {
		return nil
	}
	var errs []error
	if len(field.Hide.Scenes) == 0 {
		errs = append(errs, fmt.Errorf("field %q hide.scenes must not be empty", field.Code))
	}
	seen := make(map[string]struct{}, len(field.Hide.Scenes))
	for _, scene := range field.Hide.Scenes {
		switch scene {
		case SceneList, SceneCreate, SceneUpdate:
		default:
			errs = append(errs, fmt.Errorf("field %q has unsupported hide scene: %s", field.Code, scene))
			continue
		}
		if _, exists := seen[scene]; exists {
			errs = append(errs, fmt.Errorf("field %q has duplicated hide scene: %s", field.Code, scene))
			continue
		}
		seen[scene] = struct{}{}
	}
	return errors.Join(errs...)
}

func isContainerWidget(field *widget.Field) bool {
	if field == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(field.Widget.Type)) {
	case widget.TypeTable, widget.TypeForm:
		return true
	default:
		return false
	}
}

func filterVisibleInScene(fields []*widget.Field, scene string) []*widget.Field {
	result := make([]*widget.Field, 0, len(fields))
	for _, field := range fields {
		if VisibleInScene(field, scene) {
			result = append(result, field)
		}
	}
	return result
}

func filterEditableFields(fields []*widget.Field) []*widget.Field {
	result := make([]*widget.Field, 0, len(fields))
	for _, field := range fields {
		if field == nil {
			continue
		}
		if strings.EqualFold(field.Widget.Type, widget.TypeID) {
			continue
		}
		result = append(result, field)
	}
	return result
}

func nonNilFields(fields []*widget.Field) []*widget.Field {
	if fields == nil {
		return []*widget.Field{}
	}
	return fields
}

func normalizeCallbacks(callbacks []string) []string {
	if len(callbacks) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(callbacks))
	seen := make(map[string]struct{}, len(callbacks))
	for _, callback := range callbacks {
		callback = strings.TrimSpace(callback)
		if callback == "" {
			continue
		}
		if _, ok := seen[callback]; ok {
			continue
		}
		seen[callback] = struct{}{}
		result = append(result, callback)
	}
	return result
}
