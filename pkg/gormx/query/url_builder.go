package query

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

// StructToTableParams 将结构体转换为 Table 函数的 URL 查询字符串。
func StructToTableParams(params interface{}) (string, error) {
	if params == nil {
		return "", nil
	}

	v := reflect.ValueOf(params)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", fmt.Errorf("参数必须是结构体类型")
	}

	values := url.Values{}
	if err := appendStructQueryValues(values, v); err != nil {
		return "", err
	}
	return values.Encode(), nil
}

func appendStructQueryValues(values url.Values, v reflect.Value) error {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("参数必须是结构体类型")
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}
		if fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				continue
			}
			fieldValue = fieldValue.Elem()
		}
		if fieldValue.IsZero() {
			continue
		}
		if field.Anonymous && fieldValue.Kind() == reflect.Struct {
			if err := appendStructQueryValues(values, fieldValue); err != nil {
				return err
			}
			continue
		}

		fieldName := getJSONTag(field)
		if fieldName == "" || fieldName == "-" {
			continue
		}
		values.Set(fieldName, queryValueString(fieldValue))
	}
	return nil
}

func queryValueString(v reflect.Value) string {
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		parts := make([]string, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			parts = append(parts, fmt.Sprintf("%v", v.Index(i).Interface()))
		}
		return strings.Join(parts, ",")
	}
	return fmt.Sprintf("%v", v.Interface())
}

// getJSONTag 获取字段的 json 标签
func getJSONTag(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return strings.ToLower(field.Name)
	}
	// 处理 json:"field_name,omitempty" 格式
	parts := strings.Split(jsonTag, ",")
	return parts[0]
}
