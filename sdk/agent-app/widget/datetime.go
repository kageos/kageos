package widget

import "errors"

func init() {
	RegisterWidgetValidator(TypeDatetime, validateDatetimeWidget)
}

// DateTime 日期时间组件。
//
// 使用场景：
// - 表单里输入或展示日期时间；
// - 表格里展示真实时间列；
// - Table 筛选时在 Request 中显式声明开始/结束时间字段，并在 Handler 里手写范围查询。
//
// 推荐用法：
//
//	CreatedAt types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
//	Deadline  types.Time `json:"deadline" gorm:"column:deadline;type:datetime" widget:"name:截止时间;type:datetime;format:YYYY-MM-DD HH:mm:ss;render_default:DATE_ADD(CURRENT_DATE, INTERVAL 1 DAY)"`
//
// 协议约定：
// - 前端表单/API raw value 使用 "YYYY-MM-DD HH:mm:ss" 字符串；
// - schema 输出时 data.type 会被 decode.go 固定为 string，避免前端拿到 struct 时间对象；
// - 数据库存储推荐使用 sdk/agent-app/types.Time + gorm:"type:datetime"，不要为了前端协议把数据库列降级成 varchar；
// - render_default 是前端渲染默认值，不等于数据库默认值。
//
// 当前支持的时间默认值函数在前端 dynamicDefaultValue.ts 中解析：
// - CURRENT_TIMESTAMP / CURRENT_TIMESTAMP()
// - CURRENT_DATE / CURRENT_DATE()
// - DATE_ADD(CURRENT_TIMESTAMP, INTERVAL 1 HOUR)
// - DATE_SUB(CURRENT_DATE, INTERVAL 7 DAY)
// INTERVAL 单位支持 SECOND、MINUTE、HOUR、DAY、WEEK、MONTH、YEAR。
//
// 本文件里的函数职责：
// - Config(): 返回写入 schema 的组件配置；
// - Type(): 返回 widget 类型 datetime；
// - WidgetLLMFacts(): 给 LLM 摘要输出示例值、存储语义和默认值；
// - newDateTime(): 从 widget tag 解析 format/disabled/render_default；
// - validateDatetimeWidget(): 启动期校验 Go 字段类型是否适合 datetime。
//
// 校验规则：
// - 合法 Go 类型：string、time.Time、sdk/agent-app/types.Time，以及它们的指针；
// - 不允许 int/float/bool/slice/struct 业务对象；
// - disabled 必须显式写 true/false；
// - render_default 支持静态时间字符串，或前端可解析的 CURRENT_TIMESTAMP/CURRENT_DATE/DATE_ADD/DATE_SUB 表达式。
type DateTime struct {
	Format        string `json:"format,omitempty"`         // 日期格式，如 YYYY-MM-DD HH:mm:ss
	Disabled      bool   `json:"disabled,omitempty"`       // 是否禁用
	RenderDefault string `json:"render_default,omitempty"` // 前端渲染默认值，支持 CURRENT_TIMESTAMP、DATE_ADD 等白名单 SQL 风格表达式
}

func (t *DateTime) Config() interface{} {
	return t
}

func (t *DateTime) Type() string {
	return TypeDatetime
}

func (t *DateTime) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := []SemanticFact{
		{Key: "example", Value: `"2026-04-21 16:30:00"`},
		{Key: "storage", Value: "database datetime"},
	}
	if t.Format != "" {
		facts = append(facts, SemanticFact{Key: "display_format", Value: t.Format})
	}
	if t.RenderDefault != "" {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: t.RenderDefault})
	}
	if t.Disabled && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "disabled", Value: "true"})
	}
	return facts
}

func newDateTime(widgetParsed map[string]string) *DateTime {
	datetime := &DateTime{}

	if format, exists := widgetParsed["format"]; exists {
		datetime.Format = format
	}
	if disabled, exists := widgetParsed["disabled"]; exists {
		datetime.Disabled = disabled == "true"
	}
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		datetime.RenderDefault = defaultValue
	}

	return datetime
}

// validateDatetimeWidget 保证 datetime 组件不会绑定到无法表达时间的 Go 字段。
//
// 这里特意允许 string：
// - request DTO 常常只需要接收前端提交的时间字符串；
// - response/schema 协议也统一输出字符串。
//
// 这里同时允许 time.Time 和 sdk/agent-app/types.Time：
// - time.Time 适合一般 Go 逻辑；
// - sdk/agent-app/types.Time 是平台推荐的数据库真实时间类型，能配合 GORM datetime 列使用。
func validateDatetimeWidget(ctx ValidateContext) error {
	var errs []error
	if !isDatetimeCompatibleType(ctx.GoType) {
		errs = append(errs, fieldError(ctx, "datetime widget requires string, time.Time, or sdk/agent-app/types.Time Go type, got %s", typeName(ctx.GoType)))
	}
	if err := validateBoolTag(ctx, "disabled"); err != nil {
		errs = append(errs, err)
	}
	if err := validateDatetimeRenderDefault(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
