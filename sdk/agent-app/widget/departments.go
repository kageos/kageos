package widget

import (
	"errors"
	"fmt"
	"strconv"
)

func init() {
	RegisterWidgetValidator(TypeDepartments, validateDepartmentsWidget)
}

// Departments 多组织架构选择器组件
//
// 功能：
// - 支持多个组织架构搜索和选择
// - 支持动态默认值函数：MyDepartment()（当前用户所在部门）
// - 值使用逗号分隔的字符串格式存储（如 "/dept1,/dept2"）
//
// 使用示例：
//
//	widget:"name:关联部门;type:departments;render_default:MyDepartment()"
//	widget:"name:管理部门;type:departments;max_count:5"
//
// 动态默认值函数说明：
//   - MyDepartment(): 自动填充当前登录用户所在部门的 full_code_path
//   - 支持多个默认值，用逗号分隔：MyDepartment(),/dept2
//
// 注意：
//   - render_default 参数支持函数调用（如 MyDepartment()）
//   - 如果用户未登录或没有部门，MyDepartment() 会返回 null
//   - 值存储格式：逗号分隔的 full_code_path（如 "/dept1,/dept2"），便于存储到数据库
//   - 前端会自动处理字符串和数组之间的转换
//
// 校验规则：
// - Go 字段可以是 string/*string，也可以是 []string/[N]string；
// - max_count 必须是非负整数；
// - 不允许 []int/[]struct，因为部门标识协议是 full_code_path 字符串；
// - render_default 里的 MyDepartment() 由前端动态默认值逻辑解析。
type Departments struct {
	RenderDefault string `json:"render_default,omitempty"` // 前端渲染默认值，支持函数调用 MyDepartment()，多个值用逗号分隔
	MaxCount      int    `json:"max_count,omitempty"`      // 最大选择数量，0表示不限制
}

func (d *Departments) Config() interface{} {
	return d
}

func (d *Departments) Type() string {
	return TypeDepartments
}

func (d *Departments) WidgetLLMFacts(field *Field, opts SummaryOptions) []SemanticFact {
	facts := []SemanticFact{
		{Key: "example", Value: `"/org/hr,/org/finance"`},
	}
	if d.RenderDefault != "" {
		facts = append(facts, SemanticFact{Key: llmUIDefaultLabel, Value: d.RenderDefault})
	}
	if d.MaxCount > 0 && opts.Mode == SummaryFull {
		facts = append(facts, SemanticFact{Key: "max_count", Value: fmt.Sprintf("%d", d.MaxCount)})
	}
	return facts
}

func newDepartments(widgetParsed map[string]string) *Departments {
	departments := &Departments{}

	// 从widgetParsed中解析配置
	if defaultValue, exists := getRenderDefault(widgetParsed); exists {
		departments.RenderDefault = defaultValue
	}
	if maxCount, exists := widgetParsed["max_count"]; exists {
		// 解析最大选择数量，支持 "0" 或 "" 表示不限制
		if maxCount == "0" || maxCount == "" {
			departments.MaxCount = 0 // 0表示不限制
		} else if val, err := strconv.Atoi(maxCount); err == nil && val > 0 {
			departments.MaxCount = val
		}
	}

	return departments
}

// validateDepartmentsWidget 校验多部门组件的字符串协议。
//
// 推荐落库字段使用 string，值为逗号分隔 full_code_path；如果只是接口 DTO，也可以使用 []string。
func validateDepartmentsWidget(ctx ValidateContext) error {
	var errs []error
	if err := requireStringOrStringSliceGoType(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := validateNonNegativeIntTag(ctx, "max_count"); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
