package widget

// Department 组织架构选择器组件
//
// 功能：
// - 支持组织架构搜索和选择
// - 支持动态默认值函数：MyDepartment()（当前用户所在部门）
//
// 使用示例：
//
//	widget:"name:所属部门;type:department;default:MyDepartment()"
//
// 动态默认值函数说明：
//   - MyDepartment(): 自动填充当前登录用户所在部门的 full_code_path
//     适用于：所属部门、创建部门等字段，大部分情况下默认是当前用户所在部门
//
// 注意：
//   - default 参数支持函数调用（如 MyDepartment()）
//   - 如果用户未登录或没有部门，MyDepartment() 会返回 null
//   - 值存储格式：full_code_path（如 "/dept/subdept"）
//   - show_full_path: 是否显示全路径（默认 false，显示最后一段名称）
type Department struct {
	Default string `json:"default,omitempty"` // 默认值，支持函数调用 MyDepartment()（当前用户所在部门）
}

func (d *Department) Config() interface{} {
	return d
}

func (d *Department) Type() string {
	return TypeDepartment
}

func newDepartment(widgetParsed map[string]string) *Department {
	department := &Department{}

	// 从widgetParsed中解析配置
	if defaultValue, exists := widgetParsed["default"]; exists {
		department.Default = defaultValue
	}

	return department
}
