package query

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

// StructToTableParams 将结构体转换为 Table 函数的参数格式
// 根据 search 标签自动转换
// 返回 URL 查询字符串，格式：eq=field:value&in=field:value1,value2&sorts=id:desc
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

	t := v.Type()
	var result SearchFilterPageReq

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// 跳过零值字段
		if fieldValue.IsZero() {
			continue
		}

		// 获取 search 标签
		searchTag := field.Tag.Get("search")
		if searchTag == "" {
			// 如果没有 search 标签，检查是否是排序字段
			if field.Name == "Sorts" {
				result.Sorts = fieldValue.String()
			}
			continue
		}

		// 根据 search 标签类型转换
		searchTypes := strings.Split(searchTag, ",")
		fieldName := getJSONTag(field) // 获取 json 标签作为字段名

		for _, searchType := range searchTypes {
			searchType = strings.TrimSpace(searchType)

			switch searchType {
			case "eq":
				result.Eq = append(result.Eq, fmt.Sprintf("%s:%v", fieldName, fieldValue.Interface()))
			case "in":
				// 处理 []string 类型（推荐方式）
				if fieldValue.Kind() == reflect.Slice {
					values := make([]string, 0)
					for j := 0; j < fieldValue.Len(); j++ {
						val := fieldValue.Index(j).Interface()
						values = append(values, fmt.Sprintf("%v", val))
					}
					if len(values) > 0 {
						// 格式：field:value1,value2
						result.In = append(result.In, fmt.Sprintf("%s:%s", fieldName, strings.Join(values, ",")))
					}
				} else {
					// 单个值也支持（兼容性）
					result.In = append(result.In, fmt.Sprintf("%s:%v", fieldName, fieldValue.Interface()))
				}
			case "like":
				result.Like = append(result.Like, fmt.Sprintf("%s:%v", fieldName, fieldValue.Interface()))
			case "gte":
				result.Gte = append(result.Gte, fmt.Sprintf("%s:%v", fieldName, fieldValue.Interface()))
			case "lte":
				result.Lte = append(result.Lte, fmt.Sprintf("%s:%v", fieldName, fieldValue.Interface()))
			case "gt":
				result.Gt = append(result.Gt, fmt.Sprintf("%s:%v", fieldName, fieldValue.Interface()))
			case "lt":
				result.Lt = append(result.Lt, fmt.Sprintf("%s:%v", fieldName, fieldValue.Interface()))
			}
		}
	}

	// 转换为 URL 查询字符串
	return buildSearchParamsURL(&result), nil
}

// buildSearchParamsURL 将 SearchFilterPageReq 转换为 URL 查询字符串
func buildSearchParamsURL(req *SearchFilterPageReq) string {
	values := url.Values{}

	if len(req.Eq) > 0 {
		values.Set("eq", strings.Join(req.Eq, ","))
	}
	if len(req.In) > 0 {
		values.Set("in", strings.Join(req.In, ","))
	}
	if len(req.Like) > 0 {
		values.Set("like", strings.Join(req.Like, ","))
	}
	if len(req.Gte) > 0 {
		values.Set("gte", strings.Join(req.Gte, ","))
	}
	if len(req.Lte) > 0 {
		values.Set("lte", strings.Join(req.Lte, ","))
	}
	if len(req.Gt) > 0 {
		values.Set("gt", strings.Join(req.Gt, ","))
	}
	if len(req.Lt) > 0 {
		values.Set("lt", strings.Join(req.Lt, ","))
	}
	if req.Sorts != "" {
		values.Set("sorts", req.Sorts)
	}

	return values.Encode()
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

