package metadata

/*


yaml




*/
import (
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseMetadata 从代码中解析 metadata 并反序列化到结构体
// code: 包含 metadata 注释的代码
// result: 目标结构体指针（必须是指针类型）
func ParseMetadata(code string, result interface{}) error {
	// 验证 result 是指针类型
	rv := reflect.ValueOf(result)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("result must be a pointer to struct")
	}

	// 获取结构体类型
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("result must be a pointer to struct")
	}

	rt := rv.Type()

	// 解析所有 metadata 键值对
	metadataMap := parseMetadataMap(code)

	// 遍历结构体字段，设置值
	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)

		// 跳过不可设置的字段
		if !field.CanSet() {
			continue
		}

		// 获取 json tag 作为 key
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// 解析 json tag（可能包含 omitempty 等选项）
		jsonKey := strings.Split(jsonTag, ",")[0]
		if jsonKey == "" {
			continue
		}

		// 从 metadata 中获取值
		value, exists := metadataMap[jsonKey]
		if !exists {
			continue
		}

		// 根据字段类型设置值
		if err := setFieldValue(field, value); err != nil {
			return fmt.Errorf("设置字段 %s 失败: %w", fieldType.Name, err)
		}
	}

	return nil
}

// parseMetadataMap 解析代码中的 metadata，返回键值对 map
// 使用 /* */ 多行注释包裹 YAML 格式的 metadata
// 格式：在 /* */ 中直接写 YAML，不需要 // 前缀
// /*
// file: requirement.go
// directory_name: 需求管理系统
// directory_code: requirement
// directory_desc: |
//
//	主要是进行需求管理，包括：
//	- 需求的创建
//	- 需求的更新
//	- 需求的查询
//
// tags:
//   - 需求管理
//   - 项目管理
//
// */
func parseMetadataMap(code string) map[string]string {
	result := make(map[string]string)

	// 查找 /* 到 */ 之间的内容（整个 metadata 块）
	startMarker := "/*"
	endMarker := "*/"

	startIdx := strings.Index(code, startMarker)
	if startIdx == -1 {
		return result
	}

	actualStart := startIdx + len(startMarker)
	endIdx := strings.Index(code[actualStart:], endMarker)
	if endIdx == -1 {
		return result
	}

	// 提取整个 metadata 块的内容（YAML 格式）
	yamlContent := strings.TrimSpace(code[actualStart : actualStart+endIdx])
	if yamlContent == "" {
		return result
	}

	// 使用 YAML 解析库解析
	var yamlMap map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &yamlMap); err != nil {
		// 如果 YAML 解析失败，返回空 map（可能不是 metadata 注释）
		return result
	}

	// 转换为 map[string]string
	for k, v := range yamlMap {
		switch val := v.(type) {
		case string:
			result[k] = val
		case []interface{}:
			// 数组类型，转换为逗号分隔的字符串
			strs := make([]string, 0, len(val))
			for _, item := range val {
				if str, ok := item.(string); ok {
					strs = append(strs, str)
				}
			}
			result[k] = strings.Join(strs, ",")
		default:
			// 其他类型转换为字符串
			result[k] = fmt.Sprintf("%v", val)
		}
	}

	// 注意：YAML 解析会保留字符串的原始格式（包括末尾换行），这是正常的

	return result
}

// setFieldValue 根据字段类型设置值
func setFieldValue(field reflect.Value, value string) error {
	if value == "" {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Slice:
		// 假设是 []string，使用逗号分隔
		if field.Type().Elem().Kind() == reflect.String {
			parts := strings.Split(value, ",")
			slice := reflect.MakeSlice(field.Type(), len(parts), len(parts))
			for i, part := range parts {
				slice.Index(i).SetString(strings.TrimSpace(part))
			}
			field.Set(slice)
		} else {
			return fmt.Errorf("不支持的切片类型: %v", field.Type().Elem().Kind())
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var intVal int64
		if _, err := fmt.Sscanf(value, "%d", &intVal); err != nil {
			return fmt.Errorf("无法解析为整数: %w", err)
		}
		field.SetInt(intVal)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var uintVal uint64
		if _, err := fmt.Sscanf(value, "%d", &uintVal); err != nil {
			return fmt.Errorf("无法解析为无符号整数: %w", err)
		}
		field.SetUint(uintVal)

	case reflect.Bool:
		boolVal := strings.ToLower(value) == "true" || value == "1" || strings.ToLower(value) == "yes"
		field.SetBool(boolVal)

	case reflect.Float32, reflect.Float64:
		var floatVal float64
		if _, err := fmt.Sscanf(value, "%f", &floatVal); err != nil {
			return fmt.Errorf("无法解析为浮点数: %w", err)
		}
		field.SetFloat(floatVal)

	default:
		return fmt.Errorf("不支持的字段类型: %v", field.Kind())
	}

	return nil
}

// RemoveMetadata 从代码中移除 metadata（用于保存到文件）
// 移除 /* ... */ 注释块（如果包含 YAML 格式的 metadata）
func RemoveMetadata(code string) string {
	startMarker := "/*"
	endMarker := "*/"

	startIdx := strings.Index(code, startMarker)
	if startIdx == -1 {
		return code
	}

	endIdx := strings.Index(code[startIdx:], endMarker)
	if endIdx == -1 {
		return code
	}

	actualEnd := startIdx + endIdx + len(endMarker)

	// 检查是否是 metadata 注释（包含 YAML 格式）
	content := strings.TrimSpace(code[startIdx+len(startMarker) : startIdx+endIdx])
	// 简单判断：如果包含常见的 metadata key，认为是 metadata 注释
	if strings.Contains(content, "file:") || strings.Contains(content, "directory_name:") || strings.Contains(content, "directory_code:") {
		// 删除这个注释块（包括前后的空白）
		before := code[:startIdx]
		after := code[actualEnd:]

		// 清理前后的空白
		before = strings.TrimRight(before, " \t\r\n")
		after = strings.TrimLeft(after, " \t\r\n")

		// 如果前后都有内容，保留一个换行
		if before != "" && after != "" {
			before += "\n"
		}

		return before + after
	}

	// 不是 metadata 注释，保留原样
	return code
}
