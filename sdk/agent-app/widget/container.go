package widget

import (
	"errors"
	"reflect"
)

func init() {
	RegisterWidgetValidator(TypeTable, validateTableWidget)
	RegisterWidgetValidator(TypeForm, validateFormWidget)
}

// validateTableWidget 校验嵌套 table 容器。
//
// 使用示例：
//
//	Items []OrderItem `json:"items" widget:"name:明细;type:table"`
//
// 校验规则：
// - Go 字段必须是 slice/array；
// - slice/array 元素必须是 struct 或 *struct；
// - 子字段必须能被 parseNestedStructOrSlice 解析出来；
// - 子字段自己的 widget/display/validate 仍会递归走对应组件校验。
func validateTableWidget(ctx ValidateContext) error {
	typ := derefType(ctx.GoType)
	var errs []error
	if typ.Kind() != reflect.Slice && typ.Kind() != reflect.Array {
		errs = append(errs, fieldError(ctx, "table widget requires slice/array Go type, got %s", typeName(ctx.GoType)))
	} else {
		elem := derefType(typ.Elem())
		if elem.Kind() != reflect.Struct {
			errs = append(errs, fieldError(ctx, "table widget requires slice/array element to be struct, got %s", typeName(typ.Elem())))
		}
	}
	if len(ctx.Field.Children) == 0 {
		errs = append(errs, fieldError(ctx, "table widget requires parseable child fields"))
	}
	return errors.Join(errs...)
}

// validateFormWidget 校验嵌套 form 容器。
//
// 使用示例：
//
//	Profile UserProfile `json:"profile" widget:"name:档案;type:form"`
//
// 校验规则：
// - Go 字段必须是 struct 或 *struct；
// - 子字段必须能被解析；
// - form 只表达嵌套对象，不负责校验子字段业务必填，子字段仍使用自己的 validate tag。
func validateFormWidget(ctx ValidateContext) error {
	typ := derefType(ctx.GoType)
	var errs []error
	if typ.Kind() != reflect.Struct {
		errs = append(errs, fieldError(ctx, "form widget requires struct Go type, got %s", typeName(ctx.GoType)))
	}
	if len(ctx.Field.Children) == 0 {
		errs = append(errs, fieldError(ctx, "form widget requires parseable child fields"))
	}
	return errors.Join(errs...)
}
