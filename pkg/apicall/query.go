package apicall

import (
	"net/url"
	"strconv"
	"strings"
)

type queryOption func(url.Values)

func buildQueryParams(options ...queryOption) url.Values {
	params := url.Values{}
	for _, option := range options {
		if option != nil {
			option(params)
		}
	}
	return params
}

func withTrimmedQueryValue(key, value string) queryOption {
	return func(params url.Values) {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			params.Set(key, trimmed)
		}
	}
}

func withPositiveIntQueryValue(key string, value int) queryOption {
	return func(params url.Values) {
		if value > 0 {
			params.Set(key, strconv.Itoa(value))
		}
	}
}

func withBoolQueryValue(key string, value bool) queryOption {
	return func(params url.Values) {
		if value {
			params.Set(key, "true")
		}
	}
}

func withCSVQueryValue(key string, values []string) queryOption {
	return func(params url.Values) {
		filtered := make([]string, 0, len(values))
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				filtered = append(filtered, trimmed)
			}
		}
		if len(filtered) > 0 {
			params.Set(key, strings.Join(filtered, ","))
		}
	}
}

func withPaginationQuery(page, pageSize int) queryOption {
	return func(params url.Values) {
		params.Set("page", strconv.Itoa(page))
		params.Set("page_size", strconv.Itoa(pageSize))
	}
}

func withFullCodePathQuery(fullCodePath string) queryOption {
	return withTrimmedQueryValue("full_code_path", fullCodePath)
}
