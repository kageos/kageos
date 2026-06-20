package notion

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

const notionMaxPageSize = 100

func notionEndpoint(method, urlPath, name string, requiredScopes ...string) app.ConnectorEndpoint {
	return app.ConnectorEndpoint{
		Provider:       "notion",
		Method:         method,
		URL:            urlPath,
		Name:           name,
		RequiredScopes: requiredScopes,
	}
}

func callNotionJSON(ctx *app.Context, method, path string, queryValues map[string]string, body interface{}, out interface{}) (*app.ConnectorResponse, error) {
	req := app.ConnectorRequest{
		Method: method,
		Path:   path,
		Query:  queryValues,
	}
	if body != nil {
		raw, err := marshalNotionBody(body)
		if err != nil {
			return nil, err
		}
		req.Body = raw
	}
	proxyResp, err := ctx.CallConnector("notion", req)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return proxyResp, nil
	}
	if len(proxyResp.Body) == 0 {
		return nil, fmt.Errorf("Notion API 响应为空")
	}
	if err := json.Unmarshal(proxyResp.Body, out); err != nil {
		return nil, fmt.Errorf("解析 Notion API 响应失败: %w", err)
	}
	return proxyResp, nil
}

func marshalNotionBody(body interface{}) (json.RawMessage, error) {
	switch typed := body.(type) {
	case json.RawMessage:
		return typed, nil
	case []byte:
		return json.RawMessage(typed), nil
	default:
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化 Notion API 请求体失败: %w", err)
		}
		return json.RawMessage(raw), nil
	}
}

func notionAPIPath(parts ...string) string {
	escaped := make([]string, 0, len(parts)+1)
	escaped = append(escaped, "v1")
	for _, part := range parts {
		part = strings.Trim(part, "/")
		if part == "" {
			continue
		}
		escaped = append(escaped, url.PathEscape(part))
	}
	return "/" + strings.Join(escaped, "/")
}

func notionPageQuery(pageSize int, startCursor string) map[string]string {
	values := map[string]string{
		"page_size": strconv.Itoa(pageSize),
	}
	if startCursor = strings.TrimSpace(startCursor); startCursor != "" {
		values["start_cursor"] = startCursor
	}
	return values
}

func notionPageSize(pageInfo *query.PageSortReq, defaultSize int) int {
	if pageInfo == nil {
		pageInfo = &query.PageSortReq{}
	}
	pageSize := pageInfo.GetLimit(defaultSize)
	if pageSize > notionMaxPageSize {
		pageSize = notionMaxPageSize
		pageInfo.PageSize = notionMaxPageSize
	}
	return pageSize
}

func notionEstimatedTotal(pageInfo *query.PageSortReq, pageSize, itemCount int, hasMore bool) int64 {
	if pageInfo == nil {
		pageInfo = &query.PageSortReq{}
	}
	total := int64((pageInfo.GetPage()-1)*pageSize + itemCount)
	if hasMore {
		total++
	}
	return total
}

func notionEmptyTable[T any](resp response.Response, pageInfo *query.PageSortReq) error {
	if pageInfo == nil {
		pageInfo = &query.PageSortReq{}
	}
	return resp.Table(response.TableResult{Items: []T{}, TotalCount: 0, PageInfo: pageInfo}).Build()
}

func notionDisplayValue(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return "-"
}

func notionTitleFromObject(raw map[string]interface{}) string {
	if title := notionRichTextPlainText(raw["title"]); title != "" {
		return title
	}
	properties, _ := raw["properties"].(map[string]interface{})
	for _, value := range properties {
		prop, ok := value.(map[string]interface{})
		if !ok || notionString(prop["type"]) != "title" {
			continue
		}
		if title := notionRichTextPlainText(prop["title"]); title != "" {
			return title
		}
	}
	if name := notionString(raw["name"]); name != "" {
		return name
	}
	return "Untitled"
}

func notionBlockText(raw map[string]interface{}) string {
	blockType := notionString(raw["type"])
	if blockType == "" {
		return ""
	}
	typed := notionBlockTyped(raw)
	for _, key := range []string{"rich_text", "title", "caption"} {
		if text := notionRichTextPlainText(typed[key]); text != "" {
			return text
		}
	}
	switch blockType {
	case "child_page", "child_database":
		return notionString(typed["title"])
	case "table_row":
		return notionTableRowText(typed["cells"])
	case "equation":
		return notionString(typed["expression"])
	case "embed", "bookmark", "link_preview":
		return notionString(typed["url"])
	case "file", "image", "pdf", "video", "audio":
		if urlValue := notionFileBlockURL(typed); urlValue != "" {
			return urlValue
		}
	case "divider":
		return "---"
	}
	if textObj, ok := typed["text"].(map[string]interface{}); ok {
		return notionString(textObj["content"])
	}
	return ""
}

func notionBlockTyped(raw map[string]interface{}) map[string]interface{} {
	blockType := notionString(raw["type"])
	typed, _ := raw[blockType].(map[string]interface{})
	if typed == nil {
		return map[string]interface{}{}
	}
	return typed
}

func notionTableRowText(value interface{}) string {
	cells, ok := value.([]interface{})
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(cells))
	for _, cell := range cells {
		parts = append(parts, notionRichTextPlainText(cell))
	}
	return strings.TrimSpace(strings.Join(parts, " | "))
}

func notionFileBlockURL(typed map[string]interface{}) string {
	for _, key := range []string{"file", "external"} {
		item, _ := typed[key].(map[string]interface{})
		if urlValue := notionString(item["url"]); urlValue != "" {
			return urlValue
		}
	}
	return notionString(typed["name"])
}

func notionRichTextPlainText(value interface{}) string {
	items, ok := value.([]interface{})
	if !ok {
		return notionString(value)
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if text := notionString(obj["plain_text"]); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, ""))
}

func notionString(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func notionPrettyJSON(value interface{}) string {
	if value == nil {
		return ""
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ""
	}
	return string(raw)
}

func notionSearchBody(queryText, objectType, startCursor string, pageSize int) map[string]interface{} {
	body := map[string]interface{}{
		"page_size": pageSize,
	}
	if queryText = strings.TrimSpace(queryText); queryText != "" {
		body["query"] = queryText
	}
	if startCursor = strings.TrimSpace(startCursor); startCursor != "" {
		body["start_cursor"] = startCursor
	}
	switch strings.ToLower(strings.TrimSpace(objectType)) {
	case "页面", "page", "pages":
		body["filter"] = map[string]string{"property": "object", "value": "page"}
	case "数据源", "data_source", "data sources", "数据库", "database", "databases":
		body["filter"] = map[string]string{"property": "object", "value": "data_source"}
	}
	return body
}

func notionPageOnSelectFuzzy(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	if req != nil && !req.IsByKeyword() {
		items := notionSelectedPageItems(req)
		if len(items) > 0 {
			return &callback.OnSelectFuzzyResp{Items: items}, nil
		}
	}
	keyword := ""
	if req != nil {
		keyword = req.Keyword()
	}
	body := notionSearchBody(keyword, "page", "", 20)
	var payload notionSearchResp
	if _, err := callNotionJSON(ctx, "POST", "/v1/search", nil, body, &payload); err != nil {
		return nil, err
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(payload.Results))
	for _, raw := range payload.Results {
		if notionString(raw["object"]) != "page" {
			continue
		}
		items = append(items, notionSelectItem(raw))
	}
	return &callback.OnSelectFuzzyResp{MaxSelections: 1, Items: items}, nil
}

func notionSelectedPageItems(req *callback.OnSelectFuzzyReq) []*callback.SelectFuzzyItem {
	values := make([]string, 0)
	switch value := req.GetValue().(type) {
	case string:
		values = append(values, value)
	case []string:
		values = append(values, value...)
	case []interface{}:
		for _, item := range value {
			values = append(values, fmt.Sprintf("%v", item))
		}
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		items = append(items, &callback.SelectFuzzyItem{
			Value: value,
			Label: value,
		})
	}
	return items
}

func notionSelectItem(raw map[string]interface{}) *callback.SelectFuzzyItem {
	id := notionString(raw["id"])
	title := notionTitleFromObject(raw)
	parent, _ := raw["parent"].(map[string]interface{})
	display := map[string]interface{}{
		"类型":   notionString(raw["object"]),
		"父级":   notionString(parent["type"]),
		"更新时间": notionString(raw["last_edited_time"]),
	}
	if urlValue := notionString(raw["url"]); urlValue != "" {
		display["链接"] = urlValue
	}
	return &callback.SelectFuzzyItem{
		Value:       id,
		Label:       title,
		DisplayInfo: display,
	}
}
