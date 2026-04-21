package widget

import (
	"fmt"
	"strings"
)

type MultiSelect struct {
	Options       []string `json:"options,omitempty"`        // 选项列表
	OptionsColors []string `json:"options_colors,omitempty"` // 选项的颜色，支持warning，info，success，danger，primary 还支持自定义颜色例如：#FF9800 橙色，#9C27B0 紫色，每个颜色都可以可以重复
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
		}
		// 注意：如果需要在 widget 标签中设置具体数值，前端会解析字符串
		// Go 端只需要知道是否有限制即可，具体数值由前端处理
	}
	if creatable, exists := widgetParsed["creatable"]; exists {
		multiSelect.Creatable = creatable == "true"
	}
	if optionsColors, exists := widgetParsed["options_colors"]; exists {
		// 解析逗号分隔的颜色选项
		multiSelect.OptionsColors = parseOptions(optionsColors)
	}

	return multiSelect
}
