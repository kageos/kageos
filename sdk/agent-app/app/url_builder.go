package app

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
)

// LinkValue Link 组件的值结构（JSON 格式）
type LinkValue struct {
	Type string `json:"type"` // "table" 或 "form"
	Name string `json:"name"` // 链接文本
	URL  string `json:"url"`  // 链接 URL
}

// BuildFunctionUrl 构建跳转 URL（支持函数跳转和外链）
// 支持两种模式：
// 1. 函数跳转：传递结构体参数，自动转换为函数 URL
// 2. 外链跳转：传递字符串（如 "www.baidu.com"），直接作为外链处理
// 返回格式：如果提供了 text，返回 "[text]url"，否则只返回 url
func (ctx *Context) BuildFunctionUrl(
	target string, // 函数路径（如 "meeting_room_list"）或外链（如 "www.baidu.com"）
	params interface{}, // 结构体参数（函数跳转）或 nil（外链跳转）
) (string, error) {
	return ctx.BuildFunctionUrlWithText(target, params, "")
}

// BuildFunctionUrlWithText 构建跳转 URL（带文本信息）
// 返回格式：JSON 格式 {"type":"table","name":"文本","url":"/path"}
func (ctx *Context) BuildFunctionUrlWithText(
	target string, // 函数路径（如 "meeting_room_list"）或外链（如 "www.baidu.com"）
	params interface{}, // 结构体参数（函数跳转）或 nil（外链跳转）
	text string, // 链接文本（可选）
) (string, error) {
	// 1. 判断是否是外链
	if isExternalLink(target) {
		// 外链模式：直接处理字符串
		url := normalizeExternalLink(target)
		// 外链也返回 JSON 格式（type 为空）
		return buildLinkValueJSON("", text, url)
	}

	// 2. 提取函数路径和检查是否存在 _tab 参数
	// 如果 target 是 "meeting_room_list?eq=id:1&_tab=OnTableAddRow"，需要先提取 "meeting_room_list"
	functionPath := target
	var existingQuery string
	var hasTabParam bool
	if idx := strings.Index(target, "?"); idx >= 0 {
		functionPath = target[:idx]
		existingQuery = target[idx+1:]
		
		// 检查是否存在 _tab 参数
		existingValues, err := url.ParseQuery(existingQuery)
		if err == nil {
			if tabVals, ok := existingValues["_tab"]; ok && len(tabVals) > 0 {
				hasTabParam = true
			}
		}
	}

	// 3. 函数跳转模式：获取目标函数的模板信息
	template, err := ctx.GetFunctionTemplate(functionPath)
	if err != nil {
		return "", fmt.Errorf("获取函数模板失败: %w", err)
	}

	// 4. 根据模板类型判断是 Table 还是 Form，转换 params 为查询字符串
	// 如果存在 _tab 参数（如 _tab=OnTableAddRow），则按照 Form 格式构建参数（k=v）
	var newQueryString string
	if params != nil {
		// 如果存在 _tab 参数，强制使用 Form 格式（k=v）
		if hasTabParam {
			// 使用 Form 格式：转换为 k=v 格式
			newQueryString, err = StructToFormParams(params)
			if err != nil {
				return "", fmt.Errorf("转换 Form 参数失败: %w", err)
			}
		} else {
			// 根据模板类型决定参数格式
			switch template.TemplateType() {
			case TemplateTypeTable:
				// Table 函数：根据 search 标签转换为 search 格式
				// 使用 AutoCrudTable 的 Model（包含 search 标签）
				newQueryString, err = query.StructToTableParams(params)
				if err != nil {
					return "", fmt.Errorf("转换 Table 参数失败: %w", err)
				}
			case TemplateTypeForm:
				// Form 函数：转换为 k=v 格式
				// 使用 Request 结构体
				newQueryString, err = StructToFormParams(params)
				if err != nil {
					return "", fmt.Errorf("转换 Form 参数失败: %w", err)
				}
			default:
				return "", fmt.Errorf("不支持的模板类型")
			}
		}
	}

	// 5. 构建完整 URL（使用 functionPath，不包含查询参数）
	var basePath string
	if strings.HasPrefix(functionPath, "/") {
		// 绝对路径
		basePath = functionPath
	} else {
		// 相对路径，需要获取当前 RouterGroup
		routerGroup := ctx.GetRouterGroup()
		if routerGroup == "" {
			return "", fmt.Errorf("无法获取当前 RouterGroup，请使用绝对路径")
		}
		basePath = fmt.Sprintf("%s/%s", routerGroup, functionPath)
	}
	basePath = fmt.Sprintf("/%s/%s%s", ctx.msg.User, ctx.msg.App, basePath) //前面补充租户和app信息

	// 6. 处理 target 中可能已存在的查询参数和 params 转换后的参数
	// 如果 target 已经包含参数（如 "meeting_room_list?eq=id:1&_tab=OnTableAddRow"），需要合并参数
	// 注意：existingQuery 已经在步骤 2 中提取了

	// 合并查询参数
	if existingQuery != "" || newQueryString != "" {
		values := url.Values{}

		// 解析现有参数（如果存在）
		if existingQuery != "" {
			existingValues, err := url.ParseQuery(existingQuery)
			if err != nil {
				return "", fmt.Errorf("解析现有查询参数失败: %w", err)
			}
			// 将现有参数添加到 values（包括 _tab 参数）
			for key, vals := range existingValues {
				values[key] = vals
			}
		}

		// 解析新参数（如果存在）
		if newQueryString != "" {
			newValues, err := url.ParseQuery(newQueryString)
			if err != nil {
				return "", fmt.Errorf("解析新查询参数失败: %w", err)
			}
			// 合并新参数（新参数会覆盖现有参数中相同的 key）
			for key, vals := range newValues {
				values[key] = vals
			}
		}

		// 重新构建 URL
		finalUrl := fmt.Sprintf("%s?%s", basePath, values.Encode())

		// 返回 JSON 格式（包含函数类型信息）
		return buildLinkValueJSON(template.TemplateType(), text, finalUrl)
	}

	// 没有查询参数，直接返回 basePath
	finalUrl := basePath

	// 返回 JSON 格式（包含函数类型信息）
	return buildLinkValueJSON(template.TemplateType(), text, finalUrl)
}

// StructToFormParams 将结构体转换为 Form 函数的参数格式（k=v）
func StructToFormParams(params interface{}) (string, error) {
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
	values := url.Values{}

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if fieldValue.IsZero() {
			continue
		}

		fieldName := getJSONTag(field)
		if fieldName == "" {
			continue
		}

		// 处理不同类型的值
		var value string
		switch fieldValue.Kind() {
		case reflect.Slice:
			// 数组类型：转换为逗号分隔的字符串
			sliceValues := make([]string, 0)
			for j := 0; j < fieldValue.Len(); j++ {
				sliceValues = append(sliceValues, fmt.Sprintf("%v", fieldValue.Index(j).Interface()))
			}
			value = strings.Join(sliceValues, ",")
		default:
			value = fmt.Sprintf("%v", fieldValue.Interface())
		}

		values.Set(fieldName, value)
	}

	return values.Encode(), nil
}

// getJSONTag 获取字段的 json 标签
func getJSONTag(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return strings.ToLower(field.Name)
	}
	parts := strings.Split(jsonTag, ",")
	return parts[0]
}

// isExternalLink 判断是否是外链
func isExternalLink(target string) bool {
	// 如果已经是完整的 URL（包含协议），直接返回
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return true
	}

	// 如果包含域名特征（如 www.、.com、.cn 等），判断为外链
	if strings.Contains(target, "www.") {
		return true
	}

	// 检查是否包含常见的顶级域名
	tlds := []string{".com", ".cn", ".org", ".net", ".io", ".dev", ".top", ".xyz"}
	for _, tld := range tlds {
		if strings.Contains(target, tld) {
			return true
		}
	}

	return false
}

// normalizeExternalLink 规范化外链 URL
func normalizeExternalLink(link string) string {
	// 如果已经包含协议，直接返回
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		return link
	}

	// 如果没有协议，默认添加 https://
	return "https://" + link
}

// buildLinkValueJSON 构建 Link 值的 JSON 格式
// 统一返回 JSON 格式：{"type":"table","name":"文本","url":"/path"}
// 如果 templateType 为空字符串，表示外链（type 字段为空）
// 如果 text 为空，name 字段为空字符串
func buildLinkValueJSON(templateType TemplateType, text string, url string) (string, error) {
	// 构建 JSON 格式
	linkValue := LinkValue{
		Type: string(templateType), // "table" 或 "form" 或 ""（外链）
		Name: text,                 // 链接文本，可能为空
		URL:  url,
	}

	// 序列化为 JSON 字符串
	jsonBytes, err := json.Marshal(linkValue)
	if err != nil {
		return "", fmt.Errorf("序列化 link 值失败: %w", err)
	}

	return string(jsonBytes), nil
}
