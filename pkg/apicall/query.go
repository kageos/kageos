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

func withQueryValue(key, value string) queryOption {
	return func(params url.Values) {
		params.Set(key, value)
	}
}

func withTrimmedQueryValue(key, value string) queryOption {
	return func(params url.Values) {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			params.Set(key, trimmed)
		}
	}
}

func withIntQueryValue(key string, value int) queryOption {
	return func(params url.Values) {
		params.Set(key, strconv.Itoa(value))
	}
}

func withPositiveIntQueryValue(key string, value int) queryOption {
	return func(params url.Values) {
		if value > 0 {
			params.Set(key, strconv.Itoa(value))
		}
	}
}

func withPositiveInt64QueryValue(key string, value int64) queryOption {
	return func(params url.Values) {
		if value > 0 {
			params.Set(key, strconv.FormatInt(value, 10))
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

func withBoolLiteralQueryValue(key string, value bool) queryOption {
	return func(params url.Values) {
		params.Set(key, strconv.FormatBool(value))
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

func withRepeatedQueryValues(key string, values []string) queryOption {
	return func(params url.Values) {
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				params.Add(key, trimmed)
			}
		}
	}
}

func withPaginationQuery(page, pageSize int) queryOption {
	return func(params url.Values) {
		params.Set("page", strconv.Itoa(page))
		params.Set("page_size", strconv.Itoa(pageSize))
	}
}

func withOptionalPaginationQuery(page, pageSize int) queryOption {
	return func(params url.Values) {
		if page > 0 {
			params.Set("page", strconv.Itoa(page))
		}
		if pageSize > 0 {
			params.Set("page_size", strconv.Itoa(pageSize))
		}
	}
}

func withFullCodePathQuery(fullCodePath string) queryOption {
	return withTrimmedQueryValue("full_code_path", fullCodePath)
}

func withStatusQuery(status string) queryOption {
	return withTrimmedQueryValue("status", status)
}

func withVersionQuery(version string) queryOption {
	return withTrimmedQueryValue("version", version)
}

func withIncludeTreeQuery(includeTree bool) queryOption {
	return withBoolQueryValue("include_tree", includeTree)
}
