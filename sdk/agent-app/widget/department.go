package widget

func init() {
	RegisterWidgetValidator(TypeDepartment, validateDepartmentWidget)
}

// Department 组织架构选择器组件
//
// 功能：
// - 支持组织架构搜索和选择
// - 支持动态默认值函数：MyDepartment()（当前用户所在部门）
//
// 使用示例：
//
//	widget:"name:所属部门;type:department;render_default:MyDepartment()"
//
// 动态默认值函数说明：
//   - MyDepartment(): 自动填充当前登录用户所在部门的 full_code_path
//     适用于：所属部门、创建部门等字段，大部分情况下默认是当前用户所在部门
//
// 注意：
//   - render_default 参数支持函数调用（如 MyDepartment()）
//   - 如果用户未登录或没有部门，MyDepartment() 会返回 null
//   - 值存储格式：full_code_path（如 "/dept/subdept"）
//   - show_full_path: 是否显示全路径（默认 false，显示最后一段名称）
//
// 校验规则：
// - 注册的是本文件的 validateDepartmentWidget；
// - Go 字段必须是 string 或 *string；
// - 字段值存储部门 full_code_path；
// - render_default 里的 MyDepartment() 由前端动态默认值逻辑解析。
type Department struct {
	RenderDefault string `json:"render_default,omitempty"` // 前端渲染默认值，支持函数调用 MyDepartment()（当前用户所在部门）
}

func (d *Department) Config() interface{} {
	return d
}

func (d *Department) Type() string {
	return TypeDepartment
}

func (d *Department) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := []SemanticFact{
		{Key: "example", Value: `"/org/hr"`},
	}
	if d.RenderDefault != "" {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: d.RenderDefault})
	}
	return facts
}

// validateDepartmentWidget 校验 department 的单部门字符串协议。
//
// 部门值使用 full_code_path 字符串，不能绑定到 int、slice 或结构体。
func validateDepartmentWidget(ctx ValidateContext) error {
	return requireStringLikeGoType(ctx)
}

func newDepartment(widgetParsed map[string]string) *Department {
	department := &Department{}

	// 从widgetParsed中解析配置
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		department.RenderDefault = defaultValue
	}

	return department
}
