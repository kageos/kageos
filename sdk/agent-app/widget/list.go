package widget

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/ai-agent-os/ai-agent-os/pkg/convert"
)

const (
	ListItemTypeNumber = "number"
	ListItemTypeText   = "text"
)

func init() {
	RegisterWidgetValidator(TypeList, validateListWidget)
}

// List 自由输入列表组件。
//
// 使用示例：
//
//	Numbers []int `json:"numbers" widget:"name:数字列表;type:list;item_type:number;separator:,;unique:true;max_count:10"`
//	Names []string `json:"names" widget:"name:姓名列表;type:list;item_type:text"`
//
// 与 multiselect 的区别：
// - list 是用户自由输入列表，不需要 options/回调；
// - multiselect 是从候选项中选择，必须有 options 或 OnSelectFuzzyMap。
//
// 校验规则：
// - Go 字段必须是 slice/array；
// - item_type 必须是 number 或 text；
// - item_type:number 要求元素是数值类型；
// - item_type:text 要求元素是 string；
// - max_count 必须是非负整数；
// - unique 必须显式写 true/false；
// - item_type:number 的 render_default 每一项必须是数值；
// - render_default 的数量不能超过 max_count；
// - separator/unique/render_default 是前端输入和初始化配置，不决定 Go 类型。
type List struct {
	ItemType      string `json:"item_type,omitempty"`      // 元素类型：number 或 text
	Separator     string `json:"separator,omitempty"`      // 输入分隔符，默认逗号；前端也会兼容空白、换行和中文逗号
	Placeholder   string `json:"placeholder,omitempty"`    // 占位符文本
	RenderDefault string `json:"render_default,omitempty"` // 前端渲染默认值，如 1,2,3 或 a,b,c
	Unique        bool   `json:"unique,omitempty"`         // 是否去重
	MaxCount      int    `json:"max_count,omitempty"`      // 最大数量，0 表示不限制
}

func (l *List) Config() interface{} {
	return l
}

func (l *List) Type() string {
	return TypeList
}

func (l *List) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := make([]SemanticFact, 0, 6)
	if l.ItemType != "" {
		facts = append(facts, SemanticFact{Key: "item_type", Value: l.ItemType})
	}
	if fact, ok := placeholderFact(l.Placeholder); ok {
		facts = append(facts, fact)
	}
	if l.RenderDefault != "" {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: l.RenderDefault})
		if field != nil && field.Data != nil && field.Data.Example == "" {
			facts = append(facts, SemanticFact{Key: "example", Value: quoteExampleValue(l.RenderDefault)})
		}
	}
	if l.Separator != "" && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "separator", Value: l.Separator})
	}
	if l.Unique && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "unique", Value: "true"})
	}
	if l.MaxCount > 0 && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "max_count", Value: fmt.Sprintf("%d", l.MaxCount)})
	}
	return facts
}

func newList(widgetParsed map[string]string) *List {
	list := &List{
		ItemType:  widgetParsed["item_type"],
		Separator: widgetParsed["separator"],
	}
	if list.Separator == "" {
		list.Separator = ","
	}
	if placeholder, exists := widgetParsed["placeholder"]; exists {
		list.Placeholder = placeholder
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		list.RenderDefault = defaultValue
	}
	if unique, exists := widgetParsed["unique"]; exists {
		list.Unique = unique == "true"
	}
	if maxCount, exists := widgetParsed["max_count"]; exists {
		list.MaxCount = convert.ToInt(maxCount, 0)
	}
	return list
}

// validateListWidget 校验 list 的集合类型、元素类型和数量配置。
//
// 注意 item_type 是必填契约：没有 item_type 时，前端无法稳定决定解析成数字列表还是文本列表。
func validateListWidget(ctx ValidateContext) error {
	var errs []error

	typ := derefType(ctx.GoType)
	if typ.Kind() != reflect.Slice && typ.Kind() != reflect.Array {
		errs = append(errs, fieldError(ctx, "list widget requires slice/array Go type, got %s", typeName(ctx.GoType)))
	} else if err := validateListElementType(ctx, typ.Elem()); err != nil {
		errs = append(errs, err)
	}

	if err := validateListItemType(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateNonNegativeIntTag(ctx, "max_count"); err != nil {
		errs = append(errs, err)
	}
	if err := validateBoolTag(ctx, "unique"); err != nil {
		errs = append(errs, err)
	}
	if err := validateListRenderDefault(ctx); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// validateListItemType 校验 item_type 枚举。
//
// 新增 list 子类型时，需要同步扩展这里、validateListElementType、前端 ListWidget 和 schema 类型说明。
func validateListItemType(ctx ValidateContext) error {
	switch ctx.Field.WidgetParsed["item_type"] {
	case ListItemTypeNumber, ListItemTypeText:
		return nil
	case "":
		return fieldError(ctx, "list widget requires item_type:number or item_type:text")
	default:
		return fieldError(ctx, "list widget item_type must be number or text, got %q", ctx.Field.WidgetParsed["item_type"])
	}
}

// validateListElementType 校验 Go slice/array 元素类型与 item_type 一致。
//
// 这里校验的是编译期契约，不解析 render_default；默认值字符串会由前端按 item_type 解析。
func validateListElementType(ctx ValidateContext, elem reflect.Type) error {
	elem = derefType(elem)
	switch ctx.Field.WidgetParsed["item_type"] {
	case ListItemTypeNumber:
		if !isNumericType(elem) {
			return fieldError(ctx, "list widget with item_type:number requires numeric slice/array element type, got %s", typeName(elem))
		}
	case ListItemTypeText:
		if elem.Kind() != reflect.String {
			return fieldError(ctx, "list widget with item_type:text requires string slice/array element type, got %s", typeName(elem))
		}
	}
	return nil
}

func validateListRenderDefault(ctx ValidateContext) error {
	rawDefault := strings.TrimSpace(ctx.Field.WidgetParsed[renderDefaultTagKey])
	if rawDefault == "" {
		return nil
	}
	values := parseListRenderDefaultValues(rawDefault, ctx.Field.WidgetParsed["separator"])
	if len(values) == 0 {
		return nil
	}

	var errs []error
	if ctx.Field.WidgetParsed["item_type"] == ListItemTypeNumber {
		for _, value := range values {
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				errs = append(errs, fieldError(ctx, "list widget render_default value %q must be a number", value))
			}
		}
	}

	rawMaxCount := strings.TrimSpace(ctx.Field.WidgetParsed["max_count"])
	if rawMaxCount != "" {
		maxCount, err := strconv.Atoi(rawMaxCount)
		if err == nil && maxCount > 0 && len(values) > maxCount {
			errs = append(errs, fieldError(ctx, "list widget render_default count must be <= max_count, got defaults=%d max_count=%d", len(values), maxCount))
		}
	}

	return errors.Join(errs...)
}

func parseListRenderDefaultValues(rawDefault, separator string) []string {
	separator = strings.TrimSpace(separator)
	if separator != "" && separator != "," {
		return compactTrimmedStrings(strings.Split(rawDefault, separator))
	}
	parts := strings.FieldsFunc(rawDefault, func(r rune) bool {
		return r == ',' || r == '，' || unicode.IsSpace(r)
	})
	return compactTrimmedStrings(parts)
}

func compactTrimmedStrings(parts []string) []string {
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
