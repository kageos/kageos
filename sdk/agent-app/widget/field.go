package widget

type Field struct {
	Code      string     `json:"code"`                    // 从json标签里解析，必需字段
	Name      string     `json:"name"`                    // 必需字段
	FieldName string     `json:"field_name,omitempty"`    // Go 字段名，用于验证规则中的字段引用（如 required_if=MemberType vip）
	Desc      string     `json:"desc,omitempty"`          // 字段描述，用于 placeholder 等
	Search    string     `json:"search,omitempty"`        // 搜索类型，如 "like", "eq", "in" 等
	Data      *FieldData `json:"data,omitempty"`          // 字段数据类型信息
	Widget    struct {
		Type   string      `json:"type"`                    // 组件类型，必需
		Config interface{} `json:"config,omitempty"`         // 组件配置，可以为空
	} `json:"widget"`
	Children        []*Field `json:"children,omitempty"`        // 嵌套字段（用于 list/struct 类型）
	Callbacks       []string `json:"callbacks,omitempty"`       // 字段级别的回调，如 ['OnSelectFuzzy']
	TablePermission string   `json:"table_permission,omitempty"` // 表格权限：read,update,create
	Validation      string   `json:"validation,omitempty"`      // 验证规则，完全照搬 github.com/go-playground/validator/v10
}

// FieldData 字段数据类型信息
type FieldData struct {
	Type    string `json:"type"`                // 数据类型（string/int/float/bool等），建议保留用于类型判断
	Format  string `json:"format,omitempty"`    // 格式化类型，默认不格式化，特殊场景可格式化成 csv/markdown/json/yaml/html 等
	Example string `json:"example,omitempty"`    // 示例数据，如 "10", "紧急" 等，方便前端展示示例
}
