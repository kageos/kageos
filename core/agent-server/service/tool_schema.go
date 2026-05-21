package service

import (
	"reflect"
	"strings"
	"sync"

	"github.com/kageos/kageos/dto"
)

var toolSchemaCache sync.Map

type toolSchemaCacheKey struct {
	typ    reflect.Type
	output bool
}

func toolDefinition[T any](name string, description string) dto.ToolDef {
	return toolDefinitionWithOutput[T, ToolResult](name, description)
}

func toolDefinitionWithOutput[T any, O any](name string, description string) dto.ToolDef {
	return dto.ToolDef{
		Name:         name,
		Description:  description,
		InputSchema:  toolInputSchema[T](),
		OutputSchema: toolOutputSchema[O](),
	}
}

func toolInputSchema[T any]() map[string]interface{} {
	typ := reflect.TypeOf((*T)(nil)).Elem()
	key := toolSchemaCacheKey{typ: typ}
	if cached, ok := toolSchemaCache.Load(key); ok {
		return cached.(map[string]interface{})
	}
	schema := buildToolSchema(typ)
	toolSchemaCache.Store(key, schema)
	return schema
}

func buildToolSchema(typ reflect.Type) map[string]interface{} {
	schema := buildSchemaForType(typ)
	if schema == nil {
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []interface{}{},
		}
	}
	return schema
}

func toolOutputSchema[T any]() map[string]interface{} {
	typ := reflect.TypeOf((*T)(nil)).Elem()
	key := toolSchemaCacheKey{typ: typ, output: true}
	if cached, ok := toolSchemaCache.Load(key); ok {
		return cached.(map[string]interface{})
	}
	schema := buildToolSchema(typ)
	toolSchemaCache.Store(key, schema)
	return schema
}

func buildSchemaForType(typ reflect.Type) map[string]interface{} {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Struct:
		properties := make(map[string]interface{})
		required := make([]interface{}, 0)
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() || field.Tag.Get("schema_ignore") == "true" {
				continue
			}
			name := jsonFieldName(field)
			if name == "" {
				continue
			}
			fieldSchema := buildSchemaForField(field)
			if fieldSchema == nil {
				continue
			}
			properties[name] = fieldSchema
			if field.Tag.Get("schema_required") == "true" {
				required = append(required, name)
			}
		}
		return map[string]interface{}{
			"type":       "object",
			"properties": properties,
			"required":   required,
		}
	case reflect.Slice, reflect.Array:
		return map[string]interface{}{
			"type":  "array",
			"items": buildSchemaForType(typ.Elem()),
		}
	case reflect.Map:
		schema := map[string]interface{}{
			"type": "object",
		}
		if typ.Key().Kind() == reflect.String && typ.Elem().Kind() != reflect.Interface {
			schema["additionalProperties"] = buildSchemaForType(typ.Elem())
		}
		return schema
	case reflect.String:
		return map[string]interface{}{"type": "string"}
	case reflect.Bool:
		return map[string]interface{}{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]interface{}{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]interface{}{"type": "number"}
	case reflect.Interface:
		return map[string]interface{}{"type": "object"}
	default:
		return nil
	}
}

func buildSchemaForField(field reflect.StructField) map[string]interface{} {
	schema := buildSchemaForType(field.Type)
	if schema == nil {
		return nil
	}
	if desc := field.Tag.Get("schema_desc"); desc != "" {
		schema["description"] = desc
	}
	if enumTag := field.Tag.Get("schema_enum"); enumTag != "" {
		parts := strings.Split(enumTag, ",")
		values := make([]interface{}, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				values = append(values, part)
			}
		}
		if len(values) > 0 {
			schema["enum"] = values
		}
	}
	return schema
}

func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return ""
	}
	if tag == "" {
		return field.Name
	}
	name := strings.Split(tag, ",")[0]
	if name == "" {
		return field.Name
	}
	return name
}
