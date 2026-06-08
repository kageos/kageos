package widget

import (
	"encoding/json"
	"fmt"
	"strings"
)

type SummaryMode string

const (
	SummaryCompact SummaryMode = "compact"
	SummaryFull    SummaryMode = "full"
)

type SummaryOptions struct {
	Mode     SummaryMode
	MaxDepth int
}

type SemanticFact struct {
	Key   string
	Value string
}

const llmRequiredLabel = "【必填】"
const llmUIDefaultLabel = "渲染默认值"

type FieldSemanticProvider interface {
	LLMFacts(opts SummaryOptions) []SemanticFact
	LLMSummaryLines(opts SummaryOptions) []string
}

type WidgetSemanticProvider interface {
	WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact
}

func DecodeFields(raw []interface{}) ([]*Field, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var fields []*Field
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func (f *Field) LLMFacts(opts SummaryOptions) []SemanticFact {
	if f == nil {
		return nil
	}
	opts = normalizeSummaryOptions(opts)
	facts := make([]SemanticFact, 0, 12)

	if widgetType := strings.TrimSpace(f.Widget.Type); widgetType != "" {
		facts = appendUniqueFact(facts, SemanticFact{Key: "widget", Value: widgetType})
	}
	if f.Data != nil && strings.TrimSpace(f.Data.Type) != "" {
		facts = appendUniqueFact(facts, SemanticFact{Key: "type", Value: f.Data.Type})
	}
	if format := fieldInputFormat(f); format != "" {
		facts = appendUniqueFact(facts, SemanticFact{Key: "format", Value: format})
	}
	if fieldIsRequired(f) {
		facts = appendUniqueFact(facts, SemanticFact{Key: llmRequiredLabel})
	}
	if len(f.Children) > 0 {
		facts = appendUniqueFact(facts, SemanticFact{Key: "fields", Value: fmt.Sprintf("%d", len(f.Children))})
	}

	facts = appendWidgetSemanticFacts(facts, f, opts)
	facts = appendValidationFacts(facts, f, opts)
	facts = appendCallbackFacts(facts, f, opts)

	if example := fieldExample(f); example != "" {
		facts = appendUniqueFact(facts, SemanticFact{Key: "example", Value: example})
	}
	if strings.TrimSpace(f.DependOn) != "" && opts.Mode == SummaryFull {
		facts = appendUniqueFact(facts, SemanticFact{Key: "depend_on", Value: strings.TrimSpace(f.DependOn)})
	}
	return facts
}

func (f *Field) LLMSummaryLines(opts SummaryOptions) []string {
	opts = normalizeSummaryOptions(opts)
	return f.llmSummaryLines(opts, 0)
}

func (f *Field) llmSummaryLines(opts SummaryOptions, depth int) []string {
	if f == nil {
		return nil
	}
	line := "- " + fieldDisplayName(f) + ": " + renderFacts(f.LLMFacts(opts))
	lines := []string{strings.Repeat("  ", depth) + line}
	if len(f.Children) == 0 {
		return lines
	}
	if opts.MaxDepth >= 0 && depth >= opts.MaxDepth {
		return lines
	}
	for _, child := range f.Children {
		lines = append(lines, child.llmSummaryLines(opts, depth+1)...)
	}
	return lines
}

func normalizeSummaryOptions(opts SummaryOptions) SummaryOptions {
	if opts.Mode == "" {
		opts.Mode = SummaryCompact
	}
	if opts.MaxDepth == 0 {
		opts.MaxDepth = 1
	}
	return opts
}

func fieldDisplayName(f *Field) string {
	code := strings.TrimSpace(f.Code)
	name := strings.TrimSpace(f.Name)
	switch {
	case code != "" && name != "" && code != name:
		return fmt.Sprintf("%s（%s）", code, name)
	case code != "":
		return code
	case name != "":
		return name
	default:
		return f.FieldName
	}
}

func renderFacts(facts []SemanticFact) string {
	if len(facts) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(facts))
	for _, fact := range facts {
		if fact.Key == "" {
			continue
		}
		if fact.Value == "" {
			parts = append(parts, fact.Key)
			continue
		}
		parts = append(parts, fact.Key+"="+fact.Value)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}

func appendUniqueFact(facts []SemanticFact, fact SemanticFact) []SemanticFact {
	for _, item := range facts {
		if item.Key == fact.Key && item.Value == fact.Value {
			return facts
		}
	}
	return append(facts, fact)
}

func placeholderFact(value string) (SemanticFact, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return SemanticFact{}, false
	}
	return SemanticFact{Key: "placeholder", Value: quoteExampleValue(value)}, true
}

func fieldIsRequired(f *Field) bool {
	for _, token := range splitValidationTokens(f.Validation) {
		if token == "required" {
			return true
		}
	}
	return false
}

func splitValidationTokens(validation string) []string {
	validation = strings.TrimSpace(validation)
	if validation == "" {
		return nil
	}
	parts := strings.Split(validation, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func appendValidationFacts(facts []SemanticFact, f *Field, opts SummaryOptions) []SemanticFact {
	for _, token := range splitValidationTokens(f.Validation) {
		switch {
		case strings.HasPrefix(token, "oneof=") && !hasFactKey(facts, "enum"):
			values := strings.Fields(strings.TrimPrefix(token, "oneof="))
			if len(values) > 0 {
				facts = appendUniqueFact(facts, SemanticFact{Key: "enum", Value: strings.Join(values, "|")})
			}
		case opts.Mode == SummaryFull && (strings.HasPrefix(token, "min=") || strings.HasPrefix(token, "max=")):
			parts := strings.SplitN(token, "=", 2)
			if len(parts) == 2 && parts[1] != "" {
				facts = appendUniqueFact(facts, SemanticFact{Key: parts[0], Value: parts[1]})
			}
		case opts.Mode == SummaryFull && strings.HasPrefix(token, "required_if="):
			facts = appendUniqueFact(facts, SemanticFact{Key: "required_if", Value: strings.TrimPrefix(token, "required_if=")})
		}
	}
	return facts
}

func appendCallbackFacts(facts []SemanticFact, f *Field, opts SummaryOptions) []SemanticFact {
	for _, callback := range f.Callbacks {
		callback = strings.TrimSpace(callback)
		if callback == "" {
			continue
		}
		facts = appendUniqueFact(facts, SemanticFact{Key: "callback", Value: callback})
		switch callback {
		case "OnSelectFuzzy":
			facts = appendUniqueFact(facts, SemanticFact{Key: "value_source", Value: "fuzzy search result"})
			facts = appendUniqueFact(facts, SemanticFact{Key: "候选值工具", Value: "run_on_select_fuzzy"})
			if code := strings.TrimSpace(f.Code); code != "" {
				facts = appendUniqueFact(facts, SemanticFact{
					Key:   "调用示例",
					Value: fmt.Sprintf(`run_on_select_fuzzy(full_code_path=<当前函数路径>, code=%q, keyword=<关键词>)`, code),
				})
			}
			facts = appendUniqueFact(facts, SemanticFact{Key: "example", Value: "<from OnSelectFuzzy result>"})
		}
	}
	return facts
}

func hasFactKey(facts []SemanticFact, key string) bool {
	for _, fact := range facts {
		if fact.Key == key {
			return true
		}
	}
	return false
}

func fieldInputFormat(f *Field) string {
	if f == nil || f.Data == nil {
		return ""
	}
	if s := strings.TrimSpace(f.Data.Format); s != "" {
		return s
	}
	switch strings.TrimSpace(f.Widget.Type) {
	case TypeUser:
		return "username"
	case TypeUsers:
		return "comma-separated usernames"
	case TypeDepartment:
		return "department full_code_path"
	case TypeDepartments:
		return "comma-separated department full_code_path"
	case TypeDatetime:
		return "YYYY-MM-DD HH:mm:ss"
	}
	switch strings.TrimSpace(f.Data.Type) {
	case DataTypeStrings, DataTypeInts, DataTypeFloats:
		return "json array"
	case DataTypeStruct:
		return "object"
	case DataTypeStructs:
		return "array of objects"
	default:
		return ""
	}
}

func fieldExample(f *Field) string {
	if f == nil || f.Data == nil {
		return ""
	}
	if hasCallbackKey(f, "OnSelectFuzzy") {
		return ""
	}
	if s := strings.TrimSpace(f.Data.Example); s != "" {
		return s
	}
	switch strings.TrimSpace(f.Widget.Type) {
	case TypeUser:
		return `"beiluo"`
	case TypeUsers:
		return `"beiluo,zhangsan"`
	case TypeDepartment:
		return `"/org/hr"`
	case TypeDepartments:
		return `"/org/hr,/org/finance"`
	case TypeDatetime:
		return `"2026-04-21 16:30:00"`
	}
	switch strings.TrimSpace(f.Data.Type) {
	case DataTypeStrings:
		return `["a","b"]`
	case DataTypeInts:
		return `[1,2]`
	case DataTypeFloats:
		return `[1.5,2.5]`
	default:
		return ""
	}
}

func hasCallbackKey(f *Field, target string) bool {
	for _, callback := range f.Callbacks {
		if strings.TrimSpace(callback) == target {
			return true
		}
	}
	return false
}

func quoteExampleValue(v string) string {
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, `"`) || strings.HasPrefix(v, "'") {
		return v
	}
	return fmt.Sprintf("%q", v)
}

func quoteJSONArrayExample(values []string) string {
	if len(values) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, quoteExampleValue(value))
	}
	return "[" + strings.Join(quoted, ",") + "]"
}

func appendWidgetSemanticFacts(facts []SemanticFact, f *Field, opts SummaryOptions) []SemanticFact {
	provider := decodeWidgetSemanticProvider(f)
	if provider == nil {
		return facts
	}
	for _, fact := range provider.WidgetLLMFacts(f, opts) {
		facts = appendUniqueFact(facts, fact)
	}
	return facts
}

func decodeWidgetSemanticProvider(f *Field) WidgetSemanticProvider {
	if f == nil {
		return nil
	}
	switch strings.TrimSpace(f.Widget.Type) {
	case TypeSelect:
		if cfg, ok := decodeWidgetConfig[Select](f.Widget.Config); ok {
			return cfg
		}
	case TypeMultiSelect:
		if cfg, ok := decodeWidgetConfig[MultiSelect](f.Widget.Config); ok {
			return cfg
		}
	case TypeCheckbox:
		if cfg, ok := decodeWidgetConfig[Checkbox](f.Widget.Config); ok {
			return cfg
		}
	case TypeRadio:
		if cfg, ok := decodeWidgetConfig[Radio](f.Widget.Config); ok {
			return cfg
		}
	case TypeUser:
		if cfg, ok := decodeWidgetConfig[User](f.Widget.Config); ok {
			return cfg
		}
	case TypeUsers:
		if cfg, ok := decodeWidgetConfig[Users](f.Widget.Config); ok {
			return cfg
		}
	case TypeDepartment:
		if cfg, ok := decodeWidgetConfig[Department](f.Widget.Config); ok {
			return cfg
		}
	case TypeDepartments:
		if cfg, ok := decodeWidgetConfig[Departments](f.Widget.Config); ok {
			return cfg
		}
	case TypeDatetime:
		if cfg, ok := decodeWidgetConfig[DateTime](f.Widget.Config); ok {
			return cfg
		}
	case TypeFiles:
		if cfg, ok := decodeWidgetConfig[Files](f.Widget.Config); ok {
			return cfg
		}
	case TypeInput:
		if cfg, ok := decodeWidgetConfig[Input](f.Widget.Config); ok {
			return cfg
		}
	case TypeTextArea:
		if cfg, ok := decodeWidgetConfig[TextArea](f.Widget.Config); ok {
			return cfg
		}
	case TypeInteger:
		if cfg, ok := decodeWidgetConfig[Integer](f.Widget.Config); ok {
			return cfg
		}
	case TypeFloat:
		if cfg, ok := decodeWidgetConfig[Float](f.Widget.Config); ok {
			return cfg
		}
	}
	return nil
}

func decodeWidgetConfig[T any](cfg interface{}) (*T, bool) {
	if cfg == nil {
		return nil, false
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, false
	}
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, false
	}
	return &out, true
}
