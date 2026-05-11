package workflowexpr

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	OpConst = "$const"
	OpRef   = "$ref"
)

type Context map[string]interface{}

type Options struct {
	AllowedOps map[string]bool
}

func MVPOptions() Options {
	return Options{
		AllowedOps: map[string]bool{
			OpConst: true,
			OpRef:   true,
		},
	}
}

func ResolveRaw(raw json.RawMessage, ctx Context) (interface{}, error) {
	var spec interface{}
	if len(raw) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, fmt.Errorf("invalid expression json: %w", err)
	}
	return Resolve(spec, ctx)
}

func Resolve(spec interface{}, ctx Context) (interface{}, error) {
	switch typed := spec.(type) {
	case map[string]interface{}:
		for op, value := range typed {
			if !strings.HasPrefix(op, "$") {
				continue
			}
			if len(typed) != 1 {
				return nil, fmt.Errorf("expression operator %s must be the only key", op)
			}
			switch op {
			case OpConst:
				return value, nil
			case OpRef:
				path, ok := value.(string)
				if !ok || strings.TrimSpace(path) == "" {
					return nil, fmt.Errorf("%s must be a non-empty string", OpRef)
				}
				return Lookup(ctx, path)
			default:
				return nil, fmt.Errorf("unsupported expression operator: %s", op)
			}
		}
		if len(typed) == 1 {
			if value, ok := typed[OpConst]; ok {
				return value, nil
			}
			if value, ok := typed[OpRef]; ok {
				path, ok := value.(string)
				if !ok || strings.TrimSpace(path) == "" {
					return nil, fmt.Errorf("%s must be a non-empty string", OpRef)
				}
				return Lookup(ctx, path)
			}
		}
		resolved := make(map[string]interface{}, len(typed))
		for key, value := range typed {
			child, err := Resolve(value, ctx)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			resolved[key] = child
		}
		return resolved, nil
	case []interface{}:
		resolved := make([]interface{}, 0, len(typed))
		for i, value := range typed {
			child, err := Resolve(value, ctx)
			if err != nil {
				return nil, fmt.Errorf("[%d]: %w", i, err)
			}
			resolved = append(resolved, child)
		}
		return resolved, nil
	default:
		return typed, nil
	}
}

func ValidateRaw(raw json.RawMessage, opts Options) error {
	var spec interface{}
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		return fmt.Errorf("invalid expression json: %w", err)
	}
	return Validate(spec, opts)
}

func Validate(spec interface{}, opts Options) error {
	allowed := opts.AllowedOps
	if len(allowed) == 0 {
		allowed = MVPOptions().AllowedOps
	}
	switch typed := spec.(type) {
	case map[string]interface{}:
		for op, value := range typed {
			if !strings.HasPrefix(op, "$") {
				continue
			}
			if !allowed[op] {
				return fmt.Errorf("unsupported expression operator: %s", op)
			}
			if len(typed) != 1 {
				return fmt.Errorf("expression operator %s must be the only key", op)
			}
			if op == OpRef {
				path, ok := value.(string)
				if !ok || strings.TrimSpace(path) == "" {
					return fmt.Errorf("%s must be a non-empty string", OpRef)
				}
			}
			return nil
		}
		if len(typed) == 1 {
			for op, value := range typed {
				if strings.HasPrefix(op, "$") {
					if op == OpRef {
						path, ok := value.(string)
						if !ok || strings.TrimSpace(path) == "" {
							return fmt.Errorf("%s must be a non-empty string", OpRef)
						}
					}
					return nil
				}
			}
		}
		for key, value := range typed {
			if err := Validate(value, opts); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
		}
	case []interface{}:
		for i, value := range typed {
			if err := Validate(value, opts); err != nil {
				return fmt.Errorf("[%d]: %w", i, err)
			}
		}
	}
	return nil
}

func Lookup(ctx Context, path string) (interface{}, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("ref path is empty")
	}
	parts := strings.Split(path, ".")
	var current interface{} = map[string]interface{}(ctx)
	for _, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("invalid ref path %q", path)
		}
		next, ok := lookupPart(current, part)
		if !ok {
			return nil, fmt.Errorf("ref path %q not found at %q", path, part)
		}
		current = next
	}
	return current, nil
}

func lookupPart(current interface{}, part string) (interface{}, bool) {
	switch typed := current.(type) {
	case map[string]interface{}:
		value, ok := typed[part]
		return value, ok
	case []interface{}:
		index, err := strconv.Atoi(part)
		if err != nil || index < 0 || index >= len(typed) {
			return nil, false
		}
		return typed[index], true
	default:
		return nil, false
	}
}
