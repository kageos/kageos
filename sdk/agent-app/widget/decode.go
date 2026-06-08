package widget

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// FieldTags 包含字段的所有标签信息
type FieldTags struct {
	// 基础标签
	Json         string // json tag value
	Gorm         string // gorm tag value
	Widget       string // widget tag value
	Validate     string // validate tag value
	Data         string // data tag value
	Hide         string // hide tag value
	HideSet      bool   // hide tag is present, even when empty
	Sensitive    string // sensitive tag value
	SensitiveSet bool   // sensitive tag is present, even when empty

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
}

func (t *FieldTags) GetCode() string {
	return normalizeJSONFieldName(t.Json, t.FieldName)
}

func normalizeJSONFieldName(jsonTag string, fieldName string) string {
	tag := strings.TrimSpace(jsonTag)
	if tag == "" {
		return fieldName
	}

	if idx := strings.Index(tag, ","); idx >= 0 {
		tag = strings.TrimSpace(tag[:idx])
	}

	if tag == "" {
		return fieldName
	}

	if tag == "-" {
		return ""
	}

	return tag
}

func NormalizeFieldCodes(fields []*Field) {
	for _, field := range fields {
		if field == nil {
			continue
		}

		field.Code = normalizeJSONFieldName(field.Code, field.FieldName)
		if len(field.Children) > 0 {
			NormalizeFieldCodes(field.Children)
		}
	}
}

// isJsonOmit 与 encoding/json 一致：json 标签中「名称」为 - 表示不参与序列化，解析模型时也应跳过（与 widget:"-" 等价）
func isJsonOmit(jsonTag string) bool {
	if jsonTag == "" {
		return false
	}
	name := strings.TrimSpace(jsonTag)
	if i := strings.Index(name, ","); i >= 0 {
		name = strings.TrimSpace(name[:i])
	}
	return name == "-"
}

// ParseModelResult 解析模型的结果
type ParseModelResult struct {
	Tags []*FieldTags
	Type reflect.Type // 整个结构体的类型
}

var parseModelResultCache sync.Map // map[reflect.Type]*ParseModelResult

// ParseModelWithType 解析结构体模型，返回字段的标签信息和类型信息
// 避免重复反射，一次性获取所有需要的信息
func ParseModelWithType(model interface{}) (*ParseModelResult, error) {
	typ, err := parseModelStructType(model)
	if err != nil {
		return nil, err
	}

	if cached, ok := parseModelResultCache.Load(typ); ok {
		return cloneParseModelResult(cached.(*ParseModelResult)), nil
	}

	fields, err := parseStructFields(typ)
	result := &ParseModelResult{
		Tags: fields,
		Type: typ,
	}
	if err != nil {
		return result, err
	}
	parseModelResultCache.Store(typ, cloneParseModelResult(result))
	return result, nil
}

func cloneParseModelResult(result *ParseModelResult) *ParseModelResult {
	if result == nil {
		return nil
	}
	return &ParseModelResult{
		Tags: cloneFieldTagsList(result.Tags),
		Type: result.Type,
	}
}

func cloneFieldTagsList(tags []*FieldTags) []*FieldTags {
	if tags == nil {
		return nil
	}
	cloned := make([]*FieldTags, 0, len(tags))
	for _, item := range tags {
		cloned = append(cloned, cloneFieldTags(item))
	}
	return cloned
}

func cloneFieldTags(tags *FieldTags) *FieldTags {
	if tags == nil {
		return nil
	}
	cloned := *tags
	cloned.WidgetParsed = cloneStringMap(tags.WidgetParsed)
	cloned.DataParsed = cloneStringMap(tags.DataParsed)
	cloned.Children = cloneFieldTagsList(tags.Children)
	return &cloned
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func parseModelStructType(model interface{}) (reflect.Type, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}

	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("model must be a struct or pointer to struct")
	}
	return val.Type(), nil
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

func parseSceneList(raw string) []string {
	parts := strings.Split(raw, ",")
	scenes := make([]string, 0, len(parts))
	for _, part := range parts {
		scene := strings.TrimSpace(part)
		if scene != "" {
			scenes = append(scenes, scene)
		}
	}
	return scenes
}

func buildFieldHide(tags *FieldTags) *FieldHide {
	if tags == nil || !tags.HideSet || strings.TrimSpace(tags.Hide) == "" {
		return nil
	}

	scenes := parseSceneList(tags.Hide)
	if len(scenes) == 0 {
		return nil
	}
	return &FieldHide{Scenes: scenes}
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
				parentTags.Children = children
				if err != nil {
					return fmt.Errorf("failed to parse slice element struct: %w", err)
				}
			} else if elemType.Kind() == reflect.Ptr && elemType.Elem().Kind() == reflect.Struct {
				// 处理指针切片 []*Struct
				children, err := parseStructFields(elemType.Elem())
				parentTags.Children = children
				if err != nil {
					return fmt.Errorf("failed to parse slice element struct pointer: %w", err)
				}
			}
		}
	} else if widgetType == TypeForm {
		// 只有明确指定为 form 时，才解析结构体
		if fieldType.Kind() == reflect.Struct {
			children, err := parseStructFields(fieldType)
			parentTags.Children = children
			if err != nil {
				return fmt.Errorf("failed to parse struct fields: %w", err)
			}
		} else if fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct {
			// 处理指针类型的结构体
			children, err := parseStructFields(fieldType.Elem())
			parentTags.Children = children
			if err != nil {
				return fmt.Errorf("failed to parse struct pointer fields: %w", err)
			}
		}
	}

	return nil
}

// parseStructFields 解析结构体的所有字段
func parseStructFields(structType reflect.Type) ([]*FieldTags, error) {
	fields := make([]*FieldTags, 0, structType.NumField())
	var errs []error

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		tags, ok, err := parseStructField(field)
		if err != nil {
			errs = append(errs, fmt.Errorf("field %s: %w", field.Name, err))
			continue
		}
		if !ok {
			continue
		}
		fields = append(fields, tags)
	}

	return fields, errors.Join(errs...)
}

func parseStructField(field reflect.StructField) (*FieldTags, bool, error) {
	if !field.IsExported() {
		return nil, false, nil
	}

	tags := newFieldTags(field)
	if tags.Widget == "-" || isJsonOmit(tags.Json) {
		return nil, false, nil
	}

	if tags.Widget != "" {
		if err := parseTagValue(tags.Widget, tags.WidgetParsed); err != nil {
			return nil, false, fmt.Errorf("failed to parse widget tag for field %s: %w", field.Name, err)
		}
	}

	if tags.Data != "" {
		if err := parseTagValue(tags.Data, tags.DataParsed); err != nil {
			return nil, false, fmt.Errorf("failed to parse data tag for field %s: %w", field.Name, err)
		}
	}

	widgetType := tags.WidgetParsed["type"]
	if widgetType == TypeTable || widgetType == TypeForm {
		if err := parseNestedStructOrSlice(field.Type, tags); err != nil {
			return nil, false, fmt.Errorf("failed to parse nested struct for field %s: %w", field.Name, err)
		}
		return tags, true, nil
	}

	return tags, true, nil
}

func newFieldTags(field reflect.StructField) *FieldTags {
	hideValue, hideSet := field.Tag.Lookup("hide")
	sensitiveValue, sensitiveSet := field.Tag.Lookup("sensitive")
	return &FieldTags{
		Json:         field.Tag.Get("json"),
		Gorm:         field.Tag.Get("gorm"),
		Widget:       field.Tag.Get("widget"),
		Validate:     field.Tag.Get("validate"),
		Data:         field.Tag.Get("data"),
		Callback:     field.Tag.Get("callback"),
		Hide:         hideValue,
		HideSet:      hideSet,
		Sensitive:    sensitiveValue,
		SensitiveSet: sensitiveSet,
		WidgetParsed: make(map[string]string),
		DataParsed:   make(map[string]string),
		Type:         field.Type,
		FieldName:    field.Name,
	}
}

// IsSkipField 检查是否应该跳过该字段的解析
func IsSkipField(fieldName string, fieldType reflect.Type, fieldTags *FieldTags) bool {
	// 跳过分页请求嵌套结构体
	if strings.Contains(fieldType.String(), "PageSortReq") &&
		strings.Contains(fieldName, "PageSortReq") {
		return true
	}

	// 跳过 widget="-" 或 json:"-"（与 encoding/json 省略语义一致）
	if fieldTags.Widget == "-" || isJsonOmit(fieldTags.Json) {
		return true
	}

	// 可以根据需要添加更多跳过规则
	return false
}

// ConvertTagsToField 将 FieldTags 转换为 Field 结构体
func ConvertTagsToField(tags *FieldTags) *Field {
	field := &Field{
		Code:       tags.GetCode(),
		Name:       tags.WidgetParsed["name"], // 从widget标签中获取显示名称
		Desc:       tags.WidgetParsed["desc"], // 从widget标签中获取详细说明
		FieldName:  tags.FieldName,
		Validation: tags.Validate,
		Hide:       buildFieldHide(tags),
		Data:       &FieldData{},
		DependOn:   tags.WidgetParsed["depend_on"], // 从widget标签中获取依赖字段
		Sensitive:  buildFieldSensitive(tags),
	}
	if tags.Callback != "" {
		field.Callbacks = parseCallbackTag(tags.Callback)
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

	// files 的表单协议与落库协议都是字符串 refs，Go 字段也应定义为 string。
	// datetime 的 API/schema 协议是 "YYYY-MM-DD HH:mm:ss" 字符串；数据库侧用 types.Time 落真实时间列。
	if widgetType == TypeFiles || widgetType == TypeDatetime {
		field.Data.Type = DataTypeString
	} else {
		// 根据Go类型推断数据类型，完全基于Go类型，与widget type无关
		field.Data.Type = inferDataType(tags.Type)
	}
	if format := strings.TrimSpace(tags.DataParsed["format"]); format != "" {
		field.Data.Format = format
	}
	if example := strings.TrimSpace(tags.DataParsed["example"]); example != "" {
		field.Data.Example = example
	}

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

func buildFieldSensitive(tags *FieldTags) bool {
	if tags == nil {
		return false
	}
	return explicitSensitiveTagEnabled(tags)
}

func explicitSensitiveTagEnabled(tags *FieldTags) bool {
	return tags.SensitiveSet && strings.EqualFold(strings.TrimSpace(tags.Sensitive), "true")
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
	var requestTags []*FieldTags
	var responseTags []*FieldTags
	var errs []error

	// 解析request模型
	if request != nil {
		requestResult, err := ParseModelWithType(request)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse request model: %w", err))
		}
		if requestResult != nil {
			requestTags = requestResult.Tags
			if err := validateFieldTags(requestTags, fieldsCallback, validateFieldTagOptions{}); err != nil {
				errs = append(errs, fmt.Errorf("failed to validate request model: %w", err))
			}
			requestFields = convertTagsToFields(requestResult.Tags, fieldsCallback, true)
		}
	}

	// 解析tableModel（response表模型）
	if tableModel != nil {
		responseResult, err := ParseModelWithType(tableModel)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse table model: %w", err))
		}
		if responseResult != nil {
			responseTags = responseResult.Tags
			if err := ValidateFieldTags(responseTags, fieldsCallback); err != nil {
				errs = append(errs, fmt.Errorf("failed to validate table model: %w", err))
			}
			responseTableFields = convertTagsToFields(responseResult.Tags, fieldsCallback, true)
		}
	}

	if err := ValidateFieldCallbackTargets(append(requestTags, responseTags...), fieldsCallback); err != nil {
		errs = append(errs, fmt.Errorf("failed to validate table callbacks: %w", err))
	}
	if shouldValidateTableRequestFieldConflicts(request) {
		if err := ValidateTableRequestFieldConflicts(requestTags, responseTags); err != nil {
			errs = append(errs, fmt.Errorf("failed to validate table request fields: %w", err))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return requestFields, responseTableFields, err
	}

	return requestFields, responseTableFields, nil
}

func shouldValidateTableRequestFieldConflicts(request interface{}) bool {
	if request == nil {
		return false
	}
	if requestEmbedsType(request, "PageSortReq") {
		return false
	}
	return true
}

func requestEmbedsType(model interface{}, typeName string) bool {
	t := reflect.TypeOf(model)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldType := field.Type
		for fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}
		if field.Anonymous && fieldType.Name() == typeName {
			return true
		}
	}
	return false
}

func ValidateTableRequestFieldConflicts(requestTags []*FieldTags, tableTags []*FieldTags) error {
	if len(requestTags) == 0 || len(tableTags) == 0 {
		return nil
	}
	tableCodes := make(map[string]string, len(tableTags))
	for _, tags := range tableTags {
		if tags == nil {
			continue
		}
		code := tags.GetCode()
		if code == "" {
			continue
		}
		tableCodes[code] = tags.FieldName
	}
	var errs []error
	for _, tags := range requestTags {
		if tags == nil {
			continue
		}
		code := tags.GetCode()
		if code == "" {
			continue
		}
		if tableFieldName, exists := tableCodes[code]; exists {
			errs = append(errs, fmt.Errorf("table request field %q (%s) conflicts with table model field %q (%s); request fields must not duplicate table field codes", code, tags.FieldName, code, tableFieldName))
		}
	}
	return errors.Join(errs...)
}

// DecodeForm form 函数有两个，request是对应前端的提交表单参数，response是提交后后端处理后返回的响应参数
func DecodeForm(fieldsCallback map[string][]string, request, response interface{}) (requestFields []*Field, responseFields []*Field, err error) {
	var requestTags []*FieldTags
	var responseTags []*FieldTags
	var errs []error

	// 解析request模型（表单提交参数）
	if request != nil {
		requestResult, err := ParseModelWithType(request)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse request model: %w", err))
		}
		if requestResult != nil {
			requestTags = requestResult.Tags
			if err := validateFieldTags(requestTags, fieldsCallback, validateFieldTagOptions{}); err != nil {
				errs = append(errs, fmt.Errorf("failed to validate request model: %w", err))
			}
			requestFields = convertTagsToFields(requestResult.Tags, fieldsCallback, true)
		}
	}

	// 解析response模型（表单响应参数）
	if response != nil {
		responseResult, err := ParseModelWithType(response)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to parse response model: %w", err))
		}
		if responseResult != nil {
			responseTags = responseResult.Tags
			if err := validateFieldTags(responseTags, fieldsCallback, validateFieldTagOptions{}); err != nil {
				errs = append(errs, fmt.Errorf("failed to validate response model: %w", err))
			}
			responseFields = convertTagsToFields(responseResult.Tags, fieldsCallback, false)
		}
	}

	if err := ValidateFieldCallbackTargets(append(requestTags, responseTags...), fieldsCallback); err != nil {
		errs = append(errs, fmt.Errorf("failed to validate form callbacks: %w", err))
	}
	if err := errors.Join(errs...); err != nil {
		return requestFields, responseFields, err
	}

	return requestFields, responseFields, nil
}

func convertTagsToFields(tags []*FieldTags, fieldsCallback map[string][]string, attachCallbacks bool) []*Field {
	fields := make([]*Field, 0, len(tags))
	for _, fieldTags := range tags {
		if fieldTags == nil {
			continue
		}
		if IsSkipField(fieldTags.FieldName, fieldTags.Type, fieldTags) {
			continue
		}
		field := ConvertTagsToField(fieldTags)
		if attachCallbacks {
			attachFieldCallbacks(field, fieldsCallback)
		}
		fields = append(fields, field)
	}
	return fields
}

func attachFieldCallbacks(field *Field, fieldsCallback map[string][]string) {
	if field == nil {
		return
	}
	if calls, ok := fieldsCallback[field.Code]; ok && supportsDynamicChoiceCallback(field.Widget.Type) {
		field.Callbacks = normalizeCallbackList(append(field.Callbacks, calls...))
	}
	for _, child := range field.Children {
		attachFieldCallbacks(child, fieldsCallback)
	}
}
