package service

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

func validateToolArguments(schema map[string]interface{}, args map[string]interface{}) error {
	if len(schema) == 0 {
		return nil
	}
	if args == nil {
		args = map[string]interface{}{}
	}
	return validateToolSchemaValue(schema, args, "arguments")
}

func validateToolSchemaValue(schema map[string]interface{}, value interface{}, path string) error {
	if len(schema) == 0 || value == nil {
		return nil
	}
	if err := validateToolSchemaType(schema, value, path); err != nil {
		return err
	}
	if err := validateToolSchemaEnum(schema, value, path); err != nil {
		return err
	}
	switch firstSchemaType(schema) {
	case "object":
		obj, ok := toolSchemaObjectMap(value)
		if !ok {
			return nil
		}
		return validateToolSchemaObject(schema, obj, path)
	case "array":
		items, ok := schemaMap(schema["items"])
		if !ok {
			return nil
		}
		arr, ok := toolSchemaArray(value)
		if !ok {
			return nil
		}
		var errs []error
		for i, item := range arr {
			if err := validateToolSchemaValue(items, item, path+"["+strconv.Itoa(i)+"]"); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	default:
		return nil
	}
}

func validateToolSchemaObject(schema map[string]interface{}, args map[string]interface{}, path string) error {
	var errs []error
	required := schemaStringSet(schema["required"])
	properties, _ := schema["properties"].(map[string]interface{})
	for name := range required {
		value, ok := args[name]
		if !ok || value == nil {
			errs = append(errs, fmt.Errorf("%s.%s 缺少必填参数", path, name))
			continue
		}
		if prop, ok := schemaMap(properties[name]); ok && firstSchemaType(prop) == "string" {
			if s, ok := value.(string); ok && strings.TrimSpace(s) == "" {
				errs = append(errs, fmt.Errorf("%s.%s 不能为空", path, name))
			}
		}
	}
	for name, value := range args {
		prop, ok := schemaMap(properties[name])
		if !ok {
			continue
		}
		if err := validateToolSchemaValue(prop, value, path+"."+name); err != nil {
			errs = append(errs, err)
		}
	}
	if additional, ok := schemaMap(schema["additionalProperties"]); ok {
		for name, value := range args {
			if _, declared := properties[name]; declared {
				continue
			}
			if err := validateToolSchemaValue(additional, value, path+"."+name); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func validateToolSchemaType(schema map[string]interface{}, value interface{}, path string) error {
	typ := firstSchemaType(schema)
	if typ == "" {
		return nil
	}
	valid := false
	switch typ {
	case "object":
		_, valid = toolSchemaObjectMap(value)
	case "array":
		_, valid = toolSchemaArray(value)
	case "string":
		_, valid = value.(string)
	case "boolean":
		_, valid = value.(bool)
	case "integer":
		valid = isToolSchemaInteger(value)
	case "number":
		valid = isToolSchemaNumber(value)
	default:
		return nil
	}
	if valid {
		return nil
	}
	return fmt.Errorf("%s 类型错误: 期望 %s，实际 %s", path, typ, toolSchemaValueType(value))
}

func validateToolSchemaEnum(schema map[string]interface{}, value interface{}, path string) error {
	enumValues, ok := schema["enum"].([]interface{})
	if !ok || len(enumValues) == 0 {
		return nil
	}
	for _, candidate := range enumValues {
		if reflect.DeepEqual(candidate, value) {
			return nil
		}
		if fmt.Sprint(candidate) == fmt.Sprint(value) {
			return nil
		}
	}
	return fmt.Errorf("%s 值 %q 不在允许范围内: %s", path, fmt.Sprint(value), joinEnumValues(enumValues))
}

func firstSchemaType(schema map[string]interface{}) string {
	switch raw := schema["type"].(type) {
	case string:
		return raw
	case []interface{}:
		for _, item := range raw {
			if s, ok := item.(string); ok && s != "null" {
				return s
			}
		}
	}
	return ""
}

func schemaMap(value interface{}) (map[string]interface{}, bool) {
	if value == nil {
		return nil, false
	}
	out, ok := value.(map[string]interface{})
	return out, ok
}

func schemaStringSet(value interface{}) map[string]struct{} {
	out := map[string]struct{}{}
	switch items := value.(type) {
	case []interface{}:
		for _, item := range items {
			if s, ok := item.(string); ok && s != "" {
				out[s] = struct{}{}
			}
		}
	case []string:
		for _, item := range items {
			if item != "" {
				out[item] = struct{}{}
			}
		}
	}
	return out
}

func toolSchemaObjectMap(value interface{}) (map[string]interface{}, bool) {
	if value == nil {
		return nil, false
	}
	if obj, ok := value.(map[string]interface{}); ok {
		return obj, true
	}
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
		return nil, false
	}
	out := make(map[string]interface{}, rv.Len())
	for _, key := range rv.MapKeys() {
		out[key.String()] = rv.MapIndex(key).Interface()
	}
	return out, true
}

func toolSchemaArray(value interface{}) ([]interface{}, bool) {
	if value == nil {
		return nil, false
	}
	if arr, ok := value.([]interface{}); ok {
		return arr, true
	}
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, false
	}
	out := make([]interface{}, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out = append(out, rv.Index(i).Interface())
	}
	return out, true
}

func isToolSchemaInteger(value interface{}) bool {
	switch v := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	case float64:
		return math.Trunc(v) == v
	case float32:
		return math.Trunc(float64(v)) == float64(v)
	default:
		return false
	}
}

func isToolSchemaNumber(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	default:
		return false
	}
}

func toolSchemaValueType(value interface{}) string {
	if value == nil {
		return "null"
	}
	if _, ok := toolSchemaObjectMap(value); ok {
		return "object"
	}
	if _, ok := toolSchemaArray(value); ok {
		return "array"
	}
	switch value.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "integer"
	case float32, float64:
		return "number"
	default:
		return fmt.Sprintf("%T", value)
	}
}

func joinEnumValues(values []interface{}) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprint(value))
	}
	return strings.Join(parts, ", ")
}
