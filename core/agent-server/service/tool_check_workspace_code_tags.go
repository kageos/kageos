package service

import (
	"reflect"
	"strings"
)

func parseSemicolonTag(tag string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(tag, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), ":")
		if !ok {
			continue
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out
}

func invalidSemicolonTagSegments(tag string) []string {
	var invalid []string
	for _, part := range strings.Split(tag, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, _, ok := strings.Cut(part, ":")
		if !ok || strings.TrimSpace(key) == "" {
			invalid = append(invalid, part)
		}
	}
	return invalid
}

func splitNonEmpty(s string, sep string) []string {
	var out []string
	for _, part := range strings.Split(s, sep) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func stringSetFromSlice(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out[item] = struct{}{}
		}
	}
	return out
}

func firstNonEmpty(items ...string) string {
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			return item
		}
	}
	return ""
}

func structTagValue(tag string, key string) string {
	return reflect.StructTag(tag).Get(key)
}

func structTagCode(tag string, key string) string {
	value := structTagValue(tag, key)
	if value == "" {
		return ""
	}
	code, _, _ := strings.Cut(value, ",")
	return strings.TrimSpace(code)
}
