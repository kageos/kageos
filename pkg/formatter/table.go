package formatter

import (
	"fmt"
	"html"
	"reflect"
	"strings"
)

// TableFormatter 表格格式化器
type TableFormatter struct {
	// 字段映射：结构体字段名 -> 显示名称
	FieldNames map[string]string
	// 要显示的字段列表（按顺序），如果为空则显示所有字段
	Fields []string
	// 要排除的字段列表
	ExcludeFields []string
}

// NewTableFormatter 创建表格格式化器
func NewTableFormatter() *TableFormatter {
	return &TableFormatter{
		FieldNames:    make(map[string]string),
		Fields:        []string{},
		ExcludeFields: []string{},
	}
}

// SetFieldName 设置字段显示名称
func (tf *TableFormatter) SetFieldName(fieldName, displayName string) *TableFormatter {
	tf.FieldNames[fieldName] = displayName
	return tf
}

// SetFields 设置要显示的字段列表（按顺序）
func (tf *TableFormatter) SetFields(fields ...string) *TableFormatter {
	tf.Fields = fields
	return tf
}

// ExcludeFields 设置要排除的字段列表
func (tf *TableFormatter) Exclude(fields ...string) *TableFormatter {
	tf.ExcludeFields = fields
	return tf
}

// ToMarkdown 将结构体切片转换为 Markdown 表格
func (tf *TableFormatter) ToMarkdown(data interface{}) (string, error) {
	sliceValue := reflect.ValueOf(data)
	if sliceValue.Kind() != reflect.Slice {
		return "", fmt.Errorf("data must be a slice")
	}

	if sliceValue.Len() == 0 {
		return "", nil
	}

	// 获取第一个元素的结构体类型
	firstElem := sliceValue.Index(0)
	if firstElem.Kind() == reflect.Ptr {
		firstElem = firstElem.Elem()
	}
	if firstElem.Kind() != reflect.Struct {
		return "", fmt.Errorf("slice elements must be structs")
	}

	// 获取字段列表
	fields := tf.getFields(firstElem.Type())
	if len(fields) == 0 {
		return "", nil
	}

	// 构建表头
	headers := make([]string, len(fields))
	for i, field := range fields {
		headers[i] = field.DisplayName
	}
	headerRow := "| " + strings.Join(headers, " | ") + " |"

	// 构建分隔行
	separatorRow := "| " + strings.Repeat("---|", len(fields))

	// 构建数据行
	var rows []string
	for i := 0; i < sliceValue.Len(); i++ {
		elem := sliceValue.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		cells := make([]string, len(fields))
		for j, field := range fields {
			// 支持字段路径（如 "Product.Name"）
			var fieldValue reflect.Value
			if field.IsPath {
				fieldValue = tf.getFieldValueByPath(elem, field.Name)
			} else {
				fieldValue = elem.FieldByName(field.Name)
			}
			
			// Markdown 表格需要转义管道符
			value := tf.formatValue(fieldValue)
			value = strings.ReplaceAll(value, "|", "\\|")
			cells[j] = value
		}
		rows = append(rows, "| "+strings.Join(cells, " | ")+" |")
	}

	// 组合结果
	result := []string{headerRow, separatorRow}
	result = append(result, rows...)
	return strings.Join(result, "\n"), nil
}

// ToHTML 将结构体切片转换为 HTML 表格
func (tf *TableFormatter) ToHTML(data interface{}) (string, error) {
	sliceValue := reflect.ValueOf(data)
	if sliceValue.Kind() != reflect.Slice {
		return "", fmt.Errorf("data must be a slice")
	}

	if sliceValue.Len() == 0 {
		return "", nil
	}

	// 获取第一个元素的结构体类型
	firstElem := sliceValue.Index(0)
	if firstElem.Kind() == reflect.Ptr {
		firstElem = firstElem.Elem()
	}
	if firstElem.Kind() != reflect.Struct {
		return "", fmt.Errorf("slice elements must be structs")
	}

	// 获取字段列表
	fields := tf.getFields(firstElem.Type())
	if len(fields) == 0 {
		return "", nil
	}

	// 构建表头
	var headerCells []string
	for _, field := range fields {
		displayName := html.EscapeString(field.DisplayName)
		headerCells = append(headerCells, fmt.Sprintf("<th>%s</th>", displayName))
	}
	headerRow := "<tr>" + strings.Join(headerCells, "") + "</tr>"

	// 构建数据行
	var rows []string
	for i := 0; i < sliceValue.Len(); i++ {
		elem := sliceValue.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		var cells []string
		for _, field := range fields {
			// 支持字段路径（如 "Product.Name"）
			var fieldValue reflect.Value
			if field.IsPath {
				fieldValue = tf.getFieldValueByPath(elem, field.Name)
			} else {
				fieldValue = elem.FieldByName(field.Name)
			}
			
			formattedValue := html.EscapeString(tf.formatValue(fieldValue))
			cells = append(cells, fmt.Sprintf("<td>%s</td>", formattedValue))
		}
		rows = append(rows, "<tr>"+strings.Join(cells, "")+"</tr>")
	}

	// 组合结果
	result := "<table style=\"width:100%;border-collapse:collapse;font-size:12px;\">"
	result += "<thead>" + headerRow + "</thead>"
	result += "<tbody>" + strings.Join(rows, "") + "</tbody>"
	result += "</table>"
	return result, nil
}

// ToCSV 将结构体切片转换为 CSV 格式
func (tf *TableFormatter) ToCSV(data interface{}) (string, error) {
	sliceValue := reflect.ValueOf(data)
	if sliceValue.Kind() != reflect.Slice {
		return "", fmt.Errorf("data must be a slice")
	}

	if sliceValue.Len() == 0 {
		return "", nil
	}

	// 获取第一个元素的结构体类型
	firstElem := sliceValue.Index(0)
	if firstElem.Kind() == reflect.Ptr {
		firstElem = firstElem.Elem()
	}
	if firstElem.Kind() != reflect.Struct {
		return "", fmt.Errorf("slice elements must be structs")
	}

	// 获取字段列表
	fields := tf.getFields(firstElem.Type())
	if len(fields) == 0 {
		return "", nil
	}

	// 构建 CSV 内容
	var csvRows []string

	// 表头
	headers := make([]string, len(fields))
	for i, field := range fields {
		headers[i] = tf.escapeCSV(field.DisplayName)
	}
	csvRows = append(csvRows, strings.Join(headers, ","))

	// 数据行
	for i := 0; i < sliceValue.Len(); i++ {
		elem := sliceValue.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		cells := make([]string, len(fields))
		for j, field := range fields {
			// 支持字段路径（如 "Product.Name"）
			var fieldValue reflect.Value
			if field.IsPath {
				fieldValue = tf.getFieldValueByPath(elem, field.Name)
			} else {
				fieldValue = elem.FieldByName(field.Name)
			}

			value := tf.formatValueForCSV(fieldValue)
			cells[j] = tf.escapeCSV(value)
		}
		csvRows = append(csvRows, strings.Join(cells, ","))
	}

	return strings.Join(csvRows, "\n"), nil
}

// escapeCSV 转义 CSV 字段值
func (tf *TableFormatter) escapeCSV(value string) string {
	// 如果包含逗号、引号或换行符，需要用引号包裹并转义引号
	if strings.Contains(value, ",") || strings.Contains(value, "\"") || strings.Contains(value, "\n") {
		value = strings.ReplaceAll(value, "\"", "\"\"")
		return fmt.Sprintf("\"%s\"", value)
	}
	return value
}

// formatValueForCSV 格式化字段值为 CSV 格式（纯文本，不包含 HTML）
func (tf *TableFormatter) formatValueForCSV(value reflect.Value) string {
	if !value.IsValid() {
		return ""
	}

	switch value.Kind() {
	case reflect.String:
		return value.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", value.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%.2f", value.Float())
	case reflect.Bool:
		if value.Bool() {
			return "是"
		}
		return "否"
	case reflect.Ptr:
		if value.IsNil() {
			return ""
		}
		return tf.formatValueForCSV(value.Elem())
	case reflect.Interface:
		if value.IsNil() {
			return ""
		}
		return tf.formatValueForCSV(value.Elem())
	default:
		return fmt.Sprintf("%v", value.Interface())
	}
}

// FieldInfo 字段信息（支持路径字段）
type FieldInfo struct {
	Name       string // 字段名或路径（如 "Product.Name"）
	DisplayName string // 显示名称
	IsPath     bool   // 是否为路径字段
}

// getFields 获取要显示的字段列表（支持路径字段）
func (tf *TableFormatter) getFields(structType reflect.Type) []FieldInfo {
	var fields []FieldInfo

	// 如果指定了字段列表，按顺序返回
	if len(tf.Fields) > 0 {
		for _, fieldName := range tf.Fields {
			// 检查是否为路径字段（包含 "."）
			if strings.Contains(fieldName, ".") {
				// 路径字段：创建一个虚拟的 FieldInfo
				displayName := tf.getFieldDisplayNameForPath(fieldName)
				fields = append(fields, FieldInfo{
					Name:        fieldName,
					DisplayName: displayName,
					IsPath:      true,
				})
			} else {
				// 普通字段：从结构体中查找
				field, ok := structType.FieldByName(fieldName)
				if ok {
					displayName := tf.getFieldDisplayNameFromTag(field)
					fields = append(fields, FieldInfo{
						Name:        fieldName,
						DisplayName: displayName,
						IsPath:      false,
					})
				}
			}
		}
		return fields
	}

	// 否则返回所有字段（排除指定字段）
	excludeMap := make(map[string]bool)
	for _, fieldName := range tf.ExcludeFields {
		excludeMap[fieldName] = true
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		// 跳过私有字段、gorm:"-" 字段、json:"-" 字段
		if !field.IsExported() {
			continue
		}
		if strings.Contains(field.Tag.Get("gorm"), "-") {
			continue
		}
		if strings.Contains(field.Tag.Get("json"), "-") {
			continue
		}
		if excludeMap[field.Name] {
			continue
		}
		displayName := tf.getFieldDisplayNameFromTag(field)
		fields = append(fields, FieldInfo{
			Name:        field.Name,
			DisplayName: displayName,
			IsPath:      false,
		})
	}

	return fields
}

// getFieldDisplayNameForPath 获取路径字段的显示名称
func (tf *TableFormatter) getFieldDisplayNameForPath(fieldPath string) string {
	// 优先使用 FieldNames 映射
	if displayName, ok := tf.FieldNames[fieldPath]; ok {
		return displayName
	}
	
	// 从路径中提取最后一个字段名作为显示名称
	parts := strings.Split(fieldPath, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	
	return fieldPath
}

// getFieldDisplayName 获取字段显示名称
func (tf *TableFormatter) getFieldDisplayName(fieldName string) string {
	if displayName, ok := tf.FieldNames[fieldName]; ok {
		return displayName
	}
	return fieldName
}

// getFieldDisplayNameFromTag 从 widget tag 中提取字段显示名称
func (tf *TableFormatter) getFieldDisplayNameFromTag(field reflect.StructField) string {
	// 优先使用 FieldNames 映射
	if displayName, ok := tf.FieldNames[field.Name]; ok {
		return displayName
	}

	// 从 widget tag 中提取 name
	widgetTag := field.Tag.Get("widget")
	if widgetTag != "" {
		// 解析 widget tag，格式：name:字段名;type:xxx
		// 使用字符串分割，性能更好，代码更清晰
		parts := strings.Split(widgetTag, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "name:") {
				name := strings.TrimPrefix(part, "name:")
				return strings.TrimSpace(name)
			}
		}
	}

	// 从 json tag 中提取（作为备选）
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if len(parts) > 0 && parts[0] != "-" {
			return parts[0]
		}
	}

	// 默认返回字段名
	return field.Name
}

// getFieldValueByPath 根据字段路径获取值（支持嵌套字段，如 "Product.Name"）
func (tf *TableFormatter) getFieldValueByPath(elem reflect.Value, fieldPath string) reflect.Value {
	parts := strings.Split(fieldPath, ".")
	currentValue := elem
	
	for _, part := range parts {
		if !currentValue.IsValid() {
			return reflect.Value{}
		}
		
		// 处理指针
		if currentValue.Kind() == reflect.Ptr {
			if currentValue.IsNil() {
				return reflect.Value{}
			}
			currentValue = currentValue.Elem()
		}
		
		// 处理结构体
		if currentValue.Kind() == reflect.Struct {
			field := currentValue.FieldByName(part)
			if !field.IsValid() {
				return reflect.Value{}
			}
			currentValue = field
		} else {
			return reflect.Value{}
		}
	}
	
	return currentValue
}

// formatValue 格式化字段值
func (tf *TableFormatter) formatValue(value reflect.Value) string {
	if !value.IsValid() {
		return "-"
	}

	switch value.Kind() {
	case reflect.String:
		return value.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", value.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%.2f", value.Float())
	case reflect.Bool:
		if value.Bool() {
			return "是"
		}
		return "否"
	case reflect.Ptr:
		if value.IsNil() {
			return "-"
		}
		return tf.formatValue(value.Elem())
	case reflect.Interface:
		if value.IsNil() {
			return "-"
		}
		return tf.formatValue(value.Elem())
	default:
		return fmt.Sprintf("%v", value.Interface())
	}
}

