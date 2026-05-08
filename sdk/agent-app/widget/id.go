package widget

func init() {
	RegisterWidgetValidator(TypeID, validateIDWidget)
}

// ID 标识字段组件。
//
// 使用场景：
// - 表格主键；
// - 后端对象 ID；
// - 前端需要展示但通常不手动编辑的字段。
//
// 使用示例：
//
//	ID uint `json:"id" widget:"name:ID;type:ID"`
//
// 校验规则：
// - Go 字段必须是整数类型或字符串类型；
// - 不允许 float/bool/struct/slice；
// - 如果 ID 不希望出现在创建/编辑表单，应配合 hide tag 控制场景，而不是改 widget 类型。
type ID struct {
}

func (i *ID) Config() interface{} {
	return i
}

func (i *ID) Type() string {
	return TypeID
}

func newID(widgetParsed map[string]string) *ID {
	id := &ID{}

	return id
}

// validateIDWidget 只校验 ID 的基础承载类型。
//
// 是否自增、是否可编辑、是否为数据库主键，属于 GORM/业务层约束，不在 widget validator 内判断。
func validateIDWidget(ctx ValidateContext) error {
	if !isIntegerType(ctx.GoType) && !isStringLikeType(ctx.GoType) {
		return fieldError(ctx, "ID widget requires integer or string Go type, got %s", typeName(ctx.GoType))
	}
	return nil
}
