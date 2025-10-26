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

	// 反射类型信息
	Type      reflect.Type // 字段的Go类型
	FieldName string       // 字段名称（用于调试）
}

// ParseModelResult 解析模型的结果
type ParseModelResult struct {
	Tags []*FieldTags
	Type reflect.Type // 整个结构体的类型
}

// ParseModel 解析结构体模型，返回字段的标签信息
// 这个方法是核心的反射解析工具，可以用于任何结构体的字段标签提取
func ParseModel(model interface{}) ([]*FieldTags, error) {
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
			Permission:   field.Tag.Get("permission"),
			WidgetParsed: make(map[string]string),
			DataParsed:   make(map[string]string),
			Type:         field.Type, // 保存字段类型
			FieldName:    field.Name, // 保存字段名称
		}

		// 解析widget标签（widget:"name:工单标题;type:input"）
		if tags.Widget != "" && tags.Widget != "-" {
			if err := parseTagValue(tags.Widget, tags.WidgetParsed); err != nil {
				return nil, fmt.Errorf("failed to parse widget tag for field %s: %w", field.Name, err)
			}
		}

		// 解析data标签
		if tags.Data != "" {
			if err := parseTagValue(tags.Data, tags.DataParsed); err != nil {
				return nil, fmt.Errorf("failed to parse data tag for field %s: %w", field.Name, err)
			}
		}

		fields = append(fields, tags)
	}

	return fields, nil
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
			Permission:   field.Tag.Get("permission"),
			WidgetParsed: make(map[string]string),
			DataParsed:   make(map[string]string),
			Type:         field.Type, // 保存字段类型
			FieldName:    field.Name, // 保存字段名称
		}

		// 解析widget标签（widget:"name:工单标题;type:input"）
		if tags.Widget != "" && tags.Widget != "-" {
			if err := parseTagValue(tags.Widget, tags.WidgetParsed); err != nil {
				return nil, fmt.Errorf("failed to parse widget tag for field %s: %w", field.Name, err)
			}
		}

		// 解析data标签
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
		Code:            tags.Json,
		Name:            tags.WidgetParsed["name"], // 从widget标签中获取显示名称
		Desc:            tags.WidgetParsed["desc"], // 从widget标签中获取详细说明
		Search:          tags.Search,
		Validation:      tags.Validate,
		TablePermission: tags.Permission,
		Data:            &FieldData{},
	}

	// 使用 NewWidget 创建具体的Widget配置
	if widgetType := tags.WidgetParsed["type"]; widgetType != "" {
		// 设置Widget类型
		field.Widget.Type = widgetType

		// 使用工厂方法创建Widget，自动处理各种组件的配置
		widget := NewWidget(widgetType, tags.WidgetParsed)
		field.Widget.Config = widget.Config()
	}

	// 根据widget类型推断数据类型，使用FieldTags中保存的类型信息
	field.Data.Type = inferDataType(tags.WidgetParsed["type"], tags.Type)

	return field
}

// inferDataType 根据widget类型和Go类型推断数据类型
func inferDataType(widgetType string, goType reflect.Type) string {
	// 优先使用widget类型来推断
	switch widgetType {
	case TypeInput, TypeTextArea:
		return DataTypeString
	case TypeSelect:
		return DataTypeString
	case TypeSwitch:
		return DataTypeBool
	case TypeTimestamp:
		return DataTypeTimestamp
	case TypeNumber:
		return DataTypeInt
	case TypeUser:
		return DataTypeString
	case TypeFiles:
		return DataTypeFiles
	case TypeID:
		return DataTypeInt
	case TypeFloat:
		return DataTypeFloat
	case TypeCheckbox, TypeRadio:
		return DataTypeString
	default:
		// 根据Go类型推断
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
			if goType.Elem().Kind() == reflect.String {
				return DataTypeStrings
			}
			if goType.Elem().Kind() == reflect.Int || goType.Elem().Kind() == reflect.Float32 || goType.Elem().Kind() == reflect.Float64 {
				return DataTypeNumbers
			}
			return DataTypeStructs
		case reflect.Struct:
			return DataTypeStruct
		default:
			return DataTypeString
		}
	}
}

// DecodeTable table
func DecodeTable(request, tableModel interface{}) (requestFields []*Field, responseTableFields []*Field, err error) {
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

			// 转换为Field结构
			field := ConvertTagsToField(fieldTags)
			responseTableFields = append(responseTableFields, field)
		}
	}

	return requestFields, responseTableFields, nil
}

// DecodeForm form 函数有两个，request是对应前端的提交表单参数，response是提交后后端处理后返回的响应参数
func DecodeForm(request, response interface{}) (requestFields []*Field, responseFields []*Field, err error) {
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

			// 转换为Field结构
			field := ConvertTagsToField(fieldTags)
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
