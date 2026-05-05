package widget

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func init() {
	RegisterWidgetValidator(TypeMultiSelect, validateMultiSelectWidget)
}

// MultiSelect 下拉多选组件。
//
// 使用示例：
//
//	Tags []string `json:"tags" widget:"name:标签;type:multiselect;options:前端,后端,测试;max_count:3"`
//	MemberIDs string `json:"member_ids" widget:"name:会员;type:multiselect" callback:"OnSelectFuzzy"`
//
// 值协议：
// - Go 字段可以是 []T/[N]T，也可以是 string；
// - 使用 slice/array 时，元素必须是标量；
// - 使用 string 时，前端会按逗号分隔值处理，适合落单个数据库列；
// - render_default 支持逗号分隔多个默认值。
//
// 配置来源：
// - 静态 options：适合固定枚举；
// - OnSelectFuzzyMap：适合远程搜索、联动搜索或大数据量选项；
// - creatable:true 只是允许用户创建新值，不算选项来源。
//
// 校验规则：
// - Go 字段必须是 slice/array/string；
// - slice/array 元素必须是标量；
// - max_count 必须是非负整数；
// - 必须有非空 options 或 OnSelectFuzzyMap 回调。
// - options 不能有重复值；
// - options_colors 必须和 options 数量一致，颜色值只能是 RRGGBB；
// - render_default 在纯静态 options 场景下每一项都必须属于 options；creatable:true 或动态回调场景允许外部值；
// - 纯静态 options 且不可创建/无动态回调时，max_count 不能超过 options 数量；
// - 如果配置 validate:"oneof=..."，静态 options 必须与 oneof 集合完全一致，包含 validate:"dive,oneof=..."。
type MultiSelect struct {
	Options       []string `json:"options,omitempty"`        // 选项列表
	OptionsColors []string `json:"options_colors,omitempty"` // 选项颜色，只支持 RRGGBB，例如 F56C6C
	Placeholder   string   `json:"placeholder,omitempty"`    // 占位符文本
	RenderDefault []string `json:"render_default,omitempty"` // 前端渲染默认选中的值（多个，逗号分隔）
	MaxCount      int      `json:"max_count,omitempty"`      // 最大选择数量，0表示不限制
	Creatable     bool     `json:"creatable,omitempty"`      // 是否支持创建新选项
}

func (m *MultiSelect) Config() interface{} {
	return m
}

func (m *MultiSelect) Type() string {
	return TypeMultiSelect
}

func (m *MultiSelect) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 5)
	if len(m.Options) > 0 {
		facts = append(facts, SemanticFact{Key: "enum", Value: strings.Join(m.Options, "|")})
	}
	if fact, ok := placeholderFact(m.Placeholder); ok {
		facts = append(facts, fact)
	}
	if len(m.RenderDefault) > 0 {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: strings.Join(m.RenderDefault, "|")})
		facts = append(facts, SemanticFact{Key: "example", Value: quoteJSONArrayExample(m.RenderDefault)})
	} else if len(m.Options) > 0 {
		limit := 2
		if len(m.Options) < limit {
			limit = len(m.Options)
		}
		facts = append(facts, SemanticFact{Key: "example", Value: quoteJSONArrayExample(m.Options[:limit])})
	}
	if m.MaxCount > 0 && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "max_count", Value: fmt.Sprintf("%d", m.MaxCount)})
	}
	if m.Creatable && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "creatable", Value: "true"})
	}
	return facts
}

func newMultiSelect(widgetParsed map[string]string) *MultiSelect {
	multiSelect := &MultiSelect{}

	// 从widgetParsed中解析配置
	if options, exists := widgetParsed["options"]; exists {
		// 解析逗号分隔的选项
		multiSelect.Options = parseOptions(options)
	}
	if placeholder, exists := widgetParsed["placeholder"]; exists {
		multiSelect.Placeholder = placeholder
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		// 解析默认值，支持多个值用逗号分隔
		if defaultValue != "" {
			multiSelect.RenderDefault = parseOptions(defaultValue)
		}
	}
	if maxCount, exists := widgetParsed["max_count"]; exists {
		// 解析最大选择数量，支持 "0" 或 "" 表示不限制
		if maxCount == "0" || maxCount == "" {
			multiSelect.MaxCount = 0 // 0表示不限制
		} else if val, err := strconv.Atoi(maxCount); err == nil && val > 0 {
			multiSelect.MaxCount = val
		}
	}
	if creatable, exists := widgetParsed["creatable"]; exists {
		multiSelect.Creatable = creatable == "true"
	}
	if optionsColors, exists := widgetParsed["options_colors"]; exists {
		multiSelect.OptionsColors = parseOptionsColors(optionsColors)
	}

	return multiSelect
}

// validateMultiSelectWidget 校验 multiselect 的值形态和选项来源。
//
// 与 select 一样，multiselect 必须有 options 或 OnSelectFuzzyMap。
// creatable:true 不能单独通过校验，因为它不提供候选数据源。
func validateMultiSelectWidget(ctx ValidateContext) error {
	typ := derefType(ctx.GoType)
	var errs []error
	if typ.Kind() != reflect.Slice && typ.Kind() != reflect.Array && typ.Kind() != reflect.String {
		errs = append(errs, fieldError(ctx, "multiselect widget requires slice/array or string Go type, got %s", typeName(ctx.GoType)))
	}
	if typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
		if err := validateScalarElement(ctx, typ.Elem(), "scalar"); err != nil {
			errs = append(errs, err)
		}
	}
	if err := validateBoolTag(ctx, "creatable"); err != nil {
		errs = append(errs, err)
	}
	if err := validateUniqueOptions(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateOptionsMatchOneOf(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateOptionsColors(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateNonNegativeIntTag(ctx, "max_count"); err != nil {
		errs = append(errs, err)
	}
	if err := validateStaticMultiSelectMaxCount(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := requireChoiceSource(ctx); err != nil {
		errs = append(errs, err)
	}
	allowExternalDefault := ctx.Field.WidgetParsed["creatable"] == "true" || hasChoiceCallback(ctx)
	if err := validateChoiceDefaultsInOptions(ctx, parseOptions(ctx.Field.WidgetParsed[renderDefaultTagKey]), allowExternalDefault); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func validateStaticMultiSelectMaxCount(ctx ValidateContext) error {
	if ctx.Field.WidgetParsed["creatable"] == "true" || hasChoiceCallback(ctx) {
		return nil
	}
	options := parseOptions(ctx.Field.WidgetParsed["options"])
	if len(options) == 0 {
		return nil
	}
	rawMaxCount := strings.TrimSpace(ctx.Field.WidgetParsed["max_count"])
	if rawMaxCount == "" {
		return nil
	}
	maxCount, err := strconv.Atoi(rawMaxCount)
	if err != nil || maxCount <= 0 {
		return nil
	}
	if maxCount > len(options) {
		return fieldError(ctx, "multiselect widget max_count must be <= options length for static options, got max_count=%d options=%d", maxCount, len(options))
	}
	return nil
}
