package widget

import (
	"fmt"
	"reflect"
	"strings"
)

// FieldTags 包含字段的所有标签信息
type FieldTags struct {
	// 基础标签
	Json       string // json tag value
	Gorm       string // gorm tag value
	Widget     string // widget tag value
	Search     string // search tag value
	Validate   string // validate tag value
	Data       string // data tag value
	Permission string // permission tag value

	// 解析后的widget标签
	WidgetParsed map[string]string
	// 解析后的data标签
	DataParsed map[string]string
	Callback   string

	// 反射类型信息
	Type      reflect.Type // 字段的Go类型
	FieldName string       // 字段名称（用于调试）

	// 子节点（用于嵌套的结构体或切片）
	Children []*FieldTags

	FieldsCallbackMap map[string][]string
}

func (t *FieldTags) GetCode() string {
	return t.Json
}

// ParseModelResult 解析模型的结果
type ParseModelResult struct {
	Tags []*FieldTags
	Type reflect.Type // 整个结构体的类型
}

// ParseModelWithType 解析结构体模型，返回字段的标签信息和类型信息
// 避免重复反射，一次性获取所有需要的信息
func ParseModelWithType(model interface{}) (*ParseModelResult, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}

	// 获取model的反射值
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("model must be a struct or pointer to struct")
	}

	// 获取类型信息
	typ := val.Type()
	var fields []*FieldTags

	// 遍历所有字段
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// 跳过非导出字段
		if !field.IsExported() {
			continue
		}

		// 提取所有标签
		tags := &FieldTags{
			Json:         field.Tag.Get("json"),
			Gorm:         field.Tag.Get("gorm"),
			Widget:       field.Tag.Get("widget"),
			Search:       field.Tag.Get("search"),
			Validate:     field.Tag.Get("validate"),
			Data:         field.Tag.Get("data"),
			Callback:     field.Tag.Get("callback"),
			Permission:   field.Tag.Get("permission"),
			WidgetParsed: make(map[string]string),
			DataParsed:   make(map[string]string),
			Type:         field.Type, // 保存字段类型
			FieldName:    field.Name, // 保存字段名称
			Children:     nil,        // 初始化Children为nil
		}

		// 先检查是否是 table 或 form 类型
		// 如果是，解析标签后直接递归，不需要其他处理
		if tags.Widget != "" && tags.Widget != "-" {
			// 先解析 widget 标签获取类型
			if err := parseTagValue(tags.Widget, tags.WidgetParsed); err != nil {
				return nil, fmt.Errorf("failed to parse widget tag for field %s: %w", field.Name, err)
			}

			// 判断是否是 table 或 form 类型
			widgetType := tags.WidgetParsed["type"]
			if widgetType == TypeTable || widgetType == TypeForm {
				// 是 table/form，直接递归解析子结构
				if err := parseNestedStructOrSlice(field.Type, tags); err != nil {
					return nil, fmt.Errorf("failed to parse nested struct for field %s: %w", field.Name, err)
				}
				// table/form 不需要解析 data 标签，直接添加到字段列表
				fields = append(fields, tags)
				continue
			}
		}

		// 不是 table/form 类型，正常解析 data 标签
		if tags.Data != "" {
			if err := parseTagValue(tags.Data, tags.DataParsed); err != nil {
				return nil, fmt.Errorf("failed to parse data tag for field %s: %w", field.Name, err)
			}
		}

		fields = append(fields, tags)
	}

	return &ParseModelResult{
		Tags: fields,
		Type: typ,
	}, nil
}

// parseTagValue 解析标签值，例如 "name:工单标题;type:input" -> {"name": "工单标题", "type": "input"}
func parseTagValue(tagValue string, result map[string]string) error {
	if tagValue == "" {
		return nil
	}

	// 分割多个键值对
	pairs := strings.Split(tagValue, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// 分割键和值
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid tag format: %s", pair)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		result[key] = value
	}

	return nil
}

// parseNestedStructOrSlice 递归解析嵌套的结构体或切片
// 只有明确指定 widget type 为 table 或 form 时才进行递归解析
func parseNestedStructOrSlice(fieldType reflect.Type, parentTags *FieldTags) error {
	// 获取widget类型（必须明确指定）
	widgetType := parentTags.WidgetParsed["type"]

	// 只有明确指定为 table 时，才解析切片中的结构体
	if widgetType == TypeTable {
		if fieldType.Kind() == reflect.Slice {
			elemType := fieldType.Elem()

			// 如果切片元素是结构体，递归解析
			if elemType.Kind() == reflect.Struct {
				children, err := parseStructFields(elemType)
				if err != nil {
					return fmt.Errorf("failed to parse slice element struct: %w", err)
				}
				parentTags.Children = children
			} else if elemType.Kind() == reflect.Ptr && elemType.Elem().Kind() == reflect.Struct {
				// 处理指针切片 []*Struct
				children, err := parseStructFields(elemType.Elem())
				if err != nil {
					return fmt.Errorf("failed to parse slice element struct pointer: %w", err)
				}
				parentTags.Children = children
			}
		}
	} else if widgetType == TypeForm {
		// 只有明确指定为 form 时，才解析结构体
		if fieldType.Kind() == reflect.Struct {
			children, err := parseStructFields(fieldType)
			if err != nil {
				return fmt.Errorf("failed to parse struct fields: %w", err)
			}
			parentTags.Children = children
		} else if fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct {
			// 处理指针类型的结构体
			children, err := parseStructFields(fieldType.Elem())
			if err != nil {
				return fmt.Errorf("failed to parse struct pointer fields: %w", err)
			}
			parentTags.Children = children
		}
	}

	return nil
}

// parseStructFields 解析结构体的所有字段
func parseStructFields(structType reflect.Type) ([]*FieldTags, error) {
	var children []*FieldTags

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// 跳过非导出字段
		if !field.IsExported() {
			continue
		}

		// 提取所有标签
		childTags := &FieldTags{
			Json:         field.Tag.Get("json"),
			Gorm:         field.Tag.Get("gorm"),
			Widget:       field.Tag.Get("widget"),
			Search:       field.Tag.Get("search"),
			Validate:     field.Tag.Get("validate"),
			Data:         field.Tag.Get("data"),
			Callback:     field.Tag.Get("callback"),
			Permission:   field.Tag.Get("permission"),
			WidgetParsed: make(map[string]string),
			DataParsed:   make(map[string]string),
			Type:         field.Type,
			FieldName:    field.Name,
			Children:     nil,
		}

		// 跳过 widget="-" 的字段
		if childTags.Widget == "-" {
			continue
		}

		// 先检查是否是 table 或 form 类型
		if childTags.Widget != "" && childTags.Widget != "-" {
			// 先解析 widget 标签获取类型
			if err := parseTagValue(childTags.Widget, childTags.WidgetParsed); err != nil {
				return nil, fmt.Errorf("failed to parse widget tag for field %s: %w", field.Name, err)
			}

			// 判断是否是 table 或 form 类型
			widgetType := childTags.WidgetParsed["type"]
			if widgetType == TypeTable || widgetType == TypeForm {
				// 是 table/form，直接递归解析子结构
				if err := parseNestedStructOrSlice(field.Type, childTags); err != nil {
					return nil, fmt.Errorf("failed to parse nested struct for field %s: %w", field.Name, err)
				}
				// table/form 不需要解析 data 标签，直接添加到children列表
				children = append(children, childTags)
				continue
			}
		}

		// 不是 table/form 类型，正常解析 data 标签
		if childTags.Data != "" {
			if err := parseTagValue(childTags.Data, childTags.DataParsed); err != nil {
				return nil, fmt.Errorf("failed to parse data tag for field %s: %w", field.Name, err)
			}
		}

		children = append(children, childTags)
	}

	return children, nil
}

// IsSkipField 检查是否应该跳过该字段的解析
func IsSkipField(fieldName string, fieldType reflect.Type, fieldTags *FieldTags) bool {
	// 跳过 SearchFilterPageReq 嵌套结构体
	if strings.Contains(fieldName, "SearchFilterPageReq") &&
		strings.Contains(fieldType.String(), "SearchFilterPageReq") {
		return true
	}

	// 跳过 widget="-" 的字段
	if fieldTags.Widget == "-" {
		return true
	}

	// 可以根据需要添加更多跳过规则
	return false
}

// ConvertTagsToField 将 FieldTags 转换为 Field 结构体
func ConvertTagsToField(tags *FieldTags) *Field {
	field := &Field{
		Code:            tags.GetCode(),
		Name:            tags.WidgetParsed["name"], // 从widget标签中获取显示名称
		Desc:            tags.WidgetParsed["desc"], // 从widget标签中获取详细说明
		FieldName:       tags.FieldName,
		Search:          tags.Search,
		Validation:      tags.Validate,
		TablePermission: tags.Permission,
		Data:            &FieldData{},
		DependOn:        tags.WidgetParsed["depend_on"], // 从widget标签中获取依赖字段
	}
	if tags.Callback != "" {
		field.Callbacks = strings.Split(tags.Callback, ",")
	}

	// 获取widget类型（必须明确指定，不自动推断）
	widgetType := tags.WidgetParsed["type"]

	// 设置Widget类型
	field.Widget.Type = widgetType

	// 使用 NewWidget 创建具体的Widget配置（只对基础组件类型，不包括 table 和 form）
	if widgetType != "" && widgetType != TypeTable && widgetType != TypeForm {
		// 使用工厂方法创建Widget，自动处理各种组件的配置
		widget := NewWidget(widgetType, tags.WidgetParsed)
		field.Widget.Config = widget.Config()
	}

	// 根据Go类型推断数据类型，完全基于Go类型，与widget type无关
	field.Data.Type = inferDataType(tags.Type)

	// 递归转换Children字段
	if len(tags.Children) > 0 {
		field.Children = make([]*Field, 0, len(tags.Children))
		for _, childTags := range tags.Children {
			childField := ConvertTagsToField(childTags)
			field.Children = append(field.Children, childField)
		}
	}

	return field
}

// inferDataType 根据Go类型推断数据类型（完全基于Go类型，与widget type无关）
func inferDataType(goType reflect.Type) string {
	// 处理指针类型
	if goType.Kind() == reflect.Ptr {
		elemType := goType.Elem()
		if elemType.Kind() == reflect.Struct {
			return DataTypeStruct
		}
		// 其他指针类型继续递归推断
		return inferDataType(elemType)
	}

	// 完全根据Go类型推断
	switch goType.Kind() {
	case reflect.String:
		return DataTypeString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return DataTypeInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return DataTypeInt
	case reflect.Float32, reflect.Float64:
		return DataTypeFloat
	case reflect.Bool:
		return DataTypeBool
	case reflect.Slice:
		elemType := goType.Elem()
		if elemType.Kind() == reflect.String {
			return DataTypeStrings
		}
		if elemType.Kind() == reflect.Int || elemType.Kind() == reflect.Int8 ||
			elemType.Kind() == reflect.Int16 || elemType.Kind() == reflect.Int32 ||
			elemType.Kind() == reflect.Int64 ||
			elemType.Kind() == reflect.Uint || elemType.Kind() == reflect.Uint8 ||
			elemType.Kind() == reflect.Uint16 || elemType.Kind() == reflect.Uint32 ||
			elemType.Kind() == reflect.Uint64 {
			return DataTypeInts
		}
		if elemType.Kind() == reflect.Float32 || elemType.Kind() == reflect.Float64 {
			return DataTypeFloats
		}
		return DataTypeStructs
	case reflect.Struct:
		return DataTypeStruct
	default:
		return DataTypeString
	}
}

// DecodeTable table
func DecodeTable(fieldsCallback map[string][]string, request, tableModel interface{}) (requestFields []*Field, responseTableFields []*Field, err error) {
	// 解析request模型
	if request != nil {
		requestResult, err := ParseModelWithType(request)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse request model: %w", err)
		}

		// 遍历request字段并转换为Field结构
		for _, fieldTags := range requestResult.Tags {
			// 检查是否应该跳过该字段
			if IsSkipField(fieldTags.FieldName, fieldTags.Type, fieldTags) {
				continue
			}
			fieldTags.FieldsCallbackMap = fieldsCallback
			// 转换为Field结构
			field := ConvertTagsToField(fieldTags)

			requestFields = append(requestFields, field)
		}
	}

	// 解析tableModel（response表模型）
	if tableModel != nil {
		responseResult, err := ParseModelWithType(tableModel)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse table model: %w", err)
		}

		// 遍历tableModel字段并转换为Field结构
		for _, fieldTags := range responseResult.Tags {
			// 检查是否应该跳过该字段
			if IsSkipField(fieldTags.FieldName, fieldTags.Type, fieldTags) {
				continue
			}
			fieldTags.FieldsCallbackMap = fieldsCallback

			// 转换为Field结构
			field := ConvertTagsToField(fieldTags)
			responseTableFields = append(responseTableFields, field)
		}
	}

	return requestFields, responseTableFields, nil
}

// DecodeForm form 函数有两个，request是对应前端的提交表单参数，response是提交后后端处理后返回的响应参数
func DecodeForm(fieldsCallback map[string][]string, request, response interface{}) (requestFields []*Field, responseFields []*Field, err error) {
	// 解析request模型（表单提交参数）
	if request != nil {
		requestResult, err := ParseModelWithType(request)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse request model: %w", err)
		}

		// 遍历request字段并转换为Field结构
		for _, fieldTags := range requestResult.Tags {
			// 检查是否应该跳过该字段
			if IsSkipField(fieldTags.FieldName, fieldTags.Type, fieldTags) {
				continue
			}
			//todo
			field := ConvertTagsToField(fieldTags)
			calls, ok := fieldsCallback[field.Code]
			if ok {
				field.Callbacks = calls
			}
			requestFields = append(requestFields, field)
		}
	}

	// 解析response模型（表单响应参数）
	if response != nil {
		responseResult, err := ParseModelWithType(response)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse response model: %w", err)
		}

		// 遍历response字段并转换为Field结构
		for _, fieldTags := range responseResult.Tags {
			// 检查是否应该跳过该字段
			if IsSkipField(fieldTags.FieldName, fieldTags.Type, fieldTags) {
				continue
			}

			// 转换为Field结构
			field := ConvertTagsToField(fieldTags)
			responseFields = append(responseFields, field)
		}
	}

	return requestFields, responseFields, nil
}
