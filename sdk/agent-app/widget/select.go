package widget

import (
	"errors"
	"strings"
)

func init() {
	RegisterWidgetValidator(TypeSelect, validateSelectWidget)
}

// Select 下拉单选组件。
//
// 使用示例：
//
//	Status string `json:"status" widget:"name:状态;type:select;options:启用,禁用;render_default:启用" validate:"required,oneof=启用 禁用"`
//	ProductID int `json:"product_id" widget:"name:商品;type:select" callback:"OnSelectFuzzy"`
//
// 配置来源：
// - 静态选项：通过 options 配置，逗号分隔；
// - 动态选项：通过模板的 OnSelectFuzzyMap 注册字段 code；
// - creatable:true 只表示前端允许创建新值，不算选项来源，不能单独通过校验。
//
// 校验规则：
// - Go 字段必须是标量类型：string/bool/int/uint/float 等；
// - 必须有非空 options 或 OnSelectFuzzyMap 回调；
// - options 不能有重复值；
// - options_colors 必须和 options 数量一致，颜色值只能是 RRGGBB；
// - render_default 在纯静态 options 场景下必须属于 options；creatable:true 或动态回调场景允许外部值；
// - creatable 必须显式写 true/false，拼错会启动失败；
// - 如果配置 validate:"oneof=..."，静态 options 必须与 oneof 集合完全一致，避免前端能选、后端却拒绝。
type Select struct {
	Options       []string `json:"options,omitempty"`        // 选项列表
	OptionsColors []string `json:"options_colors,omitempty"` // 选项颜色，只支持 RRGGBB，例如 F56C6C
	Placeholder   string   `json:"placeholder,omitempty"`    // 占位符文本
	RenderDefault string   `json:"render_default,omitempty"` // 前端渲染默认值
	Creatable     bool     `json:"creatable"`                // 是否支持创建新选项
}

func (s *Select) Config() interface{} {
	return s
}

func (s *Select) Type() string {
	return TypeSelect
}

func (s *Select) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 5)
	if len(s.Options) > 0 {
		facts = append(facts, SemanticFact{Key: "enum", Value: strings.Join(s.Options, "|")})
	}
	if fact, ok := placeholderFact(s.Placeholder); ok {
		facts = append(facts, fact)
	}
	if strings.TrimSpace(s.RenderDefault) != "" {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: s.RenderDefault})
		if field != nil && field.Data != nil && strings.TrimSpace(field.Data.Example) == "" {
			facts = append(facts, SemanticFact{Key: "example", Value: quoteExampleValue(s.RenderDefault)})
		}
	} else if len(s.Options) > 0 {
		facts = append(facts, SemanticFact{Key: "example", Value: quoteExampleValue(s.Options[0])})
	}
	if s.Creatable && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "creatable", Value: "true"})
	}
	return facts
}

func newSelect(widgetParsed map[string]string) *Select {
	selectWidget := &Select{}

	// 从widgetParsed中解析配置
	if options, exists := widgetParsed["options"]; exists {
		// 解析逗号分隔的选项
		selectWidget.Options = parseOptions(options)
	}
	if placeholder, exists := widgetParsed["placeholder"]; exists {
		selectWidget.Placeholder = placeholder
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		selectWidget.RenderDefault = defaultValue
	}
	if creatable, exists := widgetParsed["creatable"]; exists {
		selectWidget.Creatable = creatable == "true"
	}
	if optionsColors, exists := widgetParsed["options_colors"]; exists {
		selectWidget.OptionsColors = parseOptionsColors(optionsColors)
	}

	return selectWidget
}

// parseOptions 解析选项字符串 "低,中,高" -> []string{"低", "中", "高"}
func parseOptions(optionsStr string) []string {
	if optionsStr == "" {
		return []string{}
	}

	// 简单分割，可以后续优化为更复杂的解析
	options := strings.Split(optionsStr, ",")
	var result []string
	for _, option := range options {
		option = strings.TrimSpace(option)
		if option != "" {
			result = append(result, option)
		}
	}
	return result
}

// validateSelectWidget 校验 select 的 Go 类型和选项来源。
//
// 注意不要把 creatable:true 当成来源。creatable 是“允许用户新增选项”的交互能力，
// 但下拉组件仍然需要 options 或 OnSelectFuzzyMap 告诉前端候选项从哪里来。
func validateSelectWidget(ctx ValidateContext) error {
	var errs []error
	if !isScalarType(ctx.GoType) {
		errs = append(errs, fieldError(ctx, "select widget requires scalar Go type, got %s", typeName(ctx.GoType)))
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
	if err := requireChoiceSource(ctx); err != nil {
		errs = append(errs, err)
	}
	allowExternalDefault := ctx.Field.WidgetParsed["creatable"] == "true" || hasChoiceCallback(ctx)
	if err := validateChoiceDefaultsInOptions(ctx, []string{ctx.Field.WidgetParsed[renderDefaultTagKey]}, allowExternalDefault); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
