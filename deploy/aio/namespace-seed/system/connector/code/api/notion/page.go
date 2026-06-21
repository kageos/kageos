package notion

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

const (
	notionPageContentDefaultLimit = 100
	notionPageContentMaxBlocks    = 300
	notionPageContentMaxDepth     = 6
)

type NotionPageDetailReq struct {
	PageID string `json:"page_id" widget:"name:页面;type:select;placeholder:搜索并选择 Notion 页面" callback:"OnSelectFuzzy"`
}

type NotionPageDetailResp struct {
	ID                string `json:"id" widget:"name:页面ID;type:input"`
	Object            string `json:"object" widget:"name:对象;type:input"`
	Title             string `json:"title" widget:"name:标题;type:input"`
	URL               string `json:"url" widget:"name:链接;type:link;target:_blank;link_type:primary"`
	ParentType        string `json:"parent_type" widget:"name:父级类型;type:input"`
	Archived          bool   `json:"archived" widget:"name:归档;type:switch"`
	InTrash           bool   `json:"in_trash" widget:"name:回收站;type:switch"`
	NotionCreatedTime string `json:"notion_created_time" widget:"name:创建时间;type:input"`
	NotionEditedTime  string `json:"notion_edited_time" widget:"name:编辑时间;type:input"`
}

type NotionPageContentReq struct {
	PageID string `json:"page_id" widget:"name:页面;type:select;placeholder:搜索并选择 Notion 页面" callback:"OnSelectFuzzy"`
	Limit  int    `json:"limit" widget:"name:最大 Block 数;type:integer;min:1;max:300;render_default:100"`
}

type NotionPageContentResp struct {
	PageID         string `json:"page_id" widget:"name:页面ID;type:input"`
	Title          string `json:"title" widget:"name:标题;type:input"`
	URL            string `json:"url" widget:"name:链接;type:link;target:_blank;link_type:primary"`
	BlockCount     int    `json:"block_count" widget:"name:Block 数量;type:integer"`
	HasMore        bool   `json:"has_more" widget:"name:还有更多;type:switch"`
	NextCursor     string `json:"next_cursor" widget:"name:下一页游标;type:input"`
	Truncated      bool   `json:"truncated" widget:"name:已截断;type:switch"`
	ContentPreview string `json:"content_preview" widget:"name:正文预览;type:text_area"`
}

func NotionPageDetail(ctx *app.Context, resp response.Response) error {
	var req NotionPageDetailReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	req.PageID = strings.TrimSpace(req.PageID)
	if req.PageID == "" {
		return fmt.Errorf("请先选择 Notion 页面：可以在页面字段搜索标题，或从 Notion 搜索结果复制页面 ID")
	}
	var raw map[string]interface{}
	if _, err := callNotionJSON(ctx, http.MethodGet, notionAPIPath("pages", req.PageID), nil, nil, &raw); err != nil {
		return err
	}
	parent, _ := raw["parent"].(map[string]interface{})
	return resp.Form(&NotionPageDetailResp{
		ID:                notionString(raw["id"]),
		Object:            notionString(raw["object"]),
		Title:             notionTitleFromObject(raw),
		URL:               notionString(raw["url"]),
		ParentType:        notionString(parent["type"]),
		Archived:          raw["archived"] == true,
		InTrash:           raw["in_trash"] == true,
		NotionCreatedTime: notionString(raw["created_time"]),
		NotionEditedTime:  notionString(raw["last_edited_time"]),
	}).Build()
}

func NotionPageContent(ctx *app.Context, resp response.Response) error {
	var req NotionPageContentReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	req.PageID = strings.TrimSpace(req.PageID)
	if req.PageID == "" {
		return fmt.Errorf("请先选择 Notion 页面：页面内容 Form 需要一个页面 ID")
	}
	limit := req.Limit
	if limit <= 0 {
		limit = notionPageContentDefaultLimit
	}
	if limit > notionPageContentMaxBlocks {
		limit = notionPageContentMaxBlocks
	}

	var pageRaw map[string]interface{}
	if _, err := callNotionJSON(ctx, http.MethodGet, notionAPIPath("pages", req.PageID), nil, nil, &pageRaw); err != nil {
		return err
	}
	blockTree, err := notionFetchBlockTree(ctx, req.PageID, limit)
	if err != nil {
		return err
	}
	return resp.Form(&NotionPageContentResp{
		PageID:         notionString(pageRaw["id"]),
		Title:          notionTitleFromObject(pageRaw),
		URL:            notionString(pageRaw["url"]),
		BlockCount:     blockTree.Count,
		HasMore:        blockTree.HasMore,
		NextCursor:     blockTree.NextCursor,
		Truncated:      blockTree.Truncated,
		ContentPreview: notionBlocksText(blockTree.Blocks),
	}).Build()
}

type NotionBlockChildrenReq struct {
	BlockID           string `json:"block_id" form:"block_id" widget:"name:页面或 Block;type:select;placeholder:留空默认展示已授权页面/数据源前 20 条" callback:"OnSelectFuzzy"`
	query.PageSortReq `widget:"-"`
}

type NotionBlockItem struct {
	ID                string `json:"id" widget:"name:Block ID;type:input"`
	Object            string `json:"object" widget:"name:对象;type:input"`
	Type              string `json:"type" widget:"name:Block 类型;type:input"`
	Text              string `json:"text" widget:"name:文本;type:text_area"`
	HasChildren       bool   `json:"has_children" widget:"name:有子块;type:switch"`
	Archived          bool   `json:"archived" widget:"name:归档;type:switch"`
	InTrash           bool   `json:"in_trash" widget:"name:回收站;type:switch"`
	NotionCreatedTime string `json:"notion_created_time" widget:"name:创建时间;type:input"`
	NotionEditedTime  string `json:"notion_edited_time" widget:"name:编辑时间;type:input"`
}

type notionBlockChildrenResp struct {
	Results    []map[string]interface{} `json:"results"`
	NextCursor string                   `json:"next_cursor"`
	HasMore    bool                     `json:"has_more"`
}

type notionBlockTreeResult struct {
	Blocks     []map[string]interface{}
	Count      int
	HasMore    bool
	NextCursor string
	Truncated  bool
}

func notionFetchBlockTree(ctx *app.Context, blockID string, limit int) (*notionBlockTreeResult, error) {
	if limit <= 0 {
		limit = notionPageContentDefaultLimit
	}
	if limit > notionPageContentMaxBlocks {
		limit = notionPageContentMaxBlocks
	}
	remaining := limit
	blocks, hasMore, nextCursor, err := notionFetchBlockChildrenTree(ctx, blockID, &remaining, 0)
	if err != nil {
		return nil, err
	}
	count := limit - remaining
	return &notionBlockTreeResult{
		Blocks:     blocks,
		Count:      count,
		HasMore:    hasMore,
		NextCursor: nextCursor,
		Truncated:  hasMore && remaining <= 0,
	}, nil
}

func notionFetchBlockChildrenTree(ctx *app.Context, blockID string, remaining *int, depth int) ([]map[string]interface{}, bool, string, error) {
	if remaining == nil || *remaining <= 0 {
		return nil, true, "", nil
	}
	startCursor := ""
	blocks := make([]map[string]interface{}, 0)
	hasMore := false
	for {
		pageSize := *remaining
		if pageSize > notionMaxPageSize {
			pageSize = notionMaxPageSize
		}
		var payload notionBlockChildrenResp
		if _, err := callNotionJSON(ctx, http.MethodGet, notionAPIPath("blocks", blockID, "children"), notionPageQuery(pageSize, startCursor), nil, &payload); err != nil {
			return nil, false, "", err
		}
		for _, raw := range payload.Results {
			if *remaining <= 0 {
				return blocks, true, payload.NextCursor, nil
			}
			*remaining--
			if raw["has_children"] == true {
				if depth+1 <= notionPageContentMaxDepth {
					children, childHasMore, _, err := notionFetchBlockChildrenTree(ctx, notionString(raw["id"]), remaining, depth+1)
					if err != nil {
						return nil, false, "", err
					}
					if len(children) > 0 {
						raw["children"] = children
					}
					hasMore = hasMore || childHasMore
				} else {
					hasMore = true
				}
			}
			blocks = append(blocks, raw)
		}
		if !payload.HasMore || payload.NextCursor == "" {
			return blocks, hasMore, "", nil
		}
		startCursor = payload.NextCursor
		if *remaining <= 0 {
			return blocks, true, payload.NextCursor, nil
		}
	}
}

func NotionBlockChildren(ctx *app.Context, resp response.Response) error {
	var req NotionBlockChildrenReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	req.BlockID = strings.TrimSpace(req.BlockID)
	if req.BlockID == "" {
		return NotionDefaultOverview(ctx, resp, &req.PageSortReq)
	}
	pageSize := notionPageSize(&req.PageSortReq, 50)
	startCursor := ""
	path := notionAPIPath("blocks", req.BlockID, "children")
	for i := 1; i < req.PageSortReq.GetPage(); i++ {
		var prev notionBlockChildrenResp
		if _, err := callNotionJSON(ctx, http.MethodGet, path, notionPageQuery(pageSize, startCursor), nil, &prev); err != nil {
			return err
		}
		if !prev.HasMore || prev.NextCursor == "" {
			return notionEmptyTable[NotionBlockItem](resp, &req.PageSortReq)
		}
		startCursor = prev.NextCursor
	}

	var payload notionBlockChildrenResp
	if _, err := callNotionJSON(ctx, http.MethodGet, path, notionPageQuery(pageSize, startCursor), nil, &payload); err != nil {
		return err
	}
	items := make([]NotionBlockItem, 0, len(payload.Results))
	for _, raw := range payload.Results {
		items = append(items, notionBlockItem(raw))
	}
	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: notionEstimatedTotal(&req.PageSortReq, pageSize, len(items), payload.HasMore),
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func NotionDefaultOverview(ctx *app.Context, resp response.Response, pageInfo *query.PageSortReq) error {
	pageSize := notionPageSize(pageInfo, 20)
	startCursor := ""
	for i := 1; i < pageInfo.GetPage(); i++ {
		var prev notionSearchResp
		body := notionSearchBody("", "", startCursor, pageSize)
		if _, err := callNotionJSON(ctx, http.MethodPost, "/v1/search", nil, body, &prev); err != nil {
			return err
		}
		if !prev.HasMore || prev.NextCursor == "" {
			return notionEmptyTable[NotionBlockItem](resp, pageInfo)
		}
		startCursor = prev.NextCursor
	}

	var payload notionSearchResp
	body := notionSearchBody("", "", startCursor, pageSize)
	if _, err := callNotionJSON(ctx, http.MethodPost, "/v1/search", nil, body, &payload); err != nil {
		return err
	}
	items := make([]NotionBlockItem, 0, len(payload.Results))
	for _, raw := range payload.Results {
		items = append(items, notionOverviewItem(raw))
	}
	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: notionEstimatedTotal(pageInfo, pageSize, len(items), payload.HasMore),
		PageInfo:   pageInfo,
	}).Build()
}

func notionOverviewItem(raw map[string]interface{}) NotionBlockItem {
	objectType := notionString(raw["object"])
	return NotionBlockItem{
		ID:                notionString(raw["id"]),
		Object:            objectType,
		Type:              objectType,
		Text:              notionTitleFromObject(raw),
		HasChildren:       objectType == "page",
		Archived:          raw["archived"] == true,
		InTrash:           raw["in_trash"] == true,
		NotionCreatedTime: notionString(raw["created_time"]),
		NotionEditedTime:  notionString(raw["last_edited_time"]),
	}
}

func notionBlockItem(raw map[string]interface{}) NotionBlockItem {
	return NotionBlockItem{
		ID:                notionString(raw["id"]),
		Object:            notionString(raw["object"]),
		Type:              notionString(raw["type"]),
		Text:              notionBlockText(raw),
		HasChildren:       raw["has_children"] == true,
		Archived:          raw["archived"] == true,
		InTrash:           raw["in_trash"] == true,
		NotionCreatedTime: notionString(raw["created_time"]),
		NotionEditedTime:  notionString(raw["last_edited_time"]),
	}
}

func notionBlocksText(blocks []map[string]interface{}) string {
	lines := notionBlockLines(blocks, 0)
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func notionBlockLines(blocks []map[string]interface{}, depth int) []string {
	lines := make([]string, 0, len(blocks))
	for _, raw := range blocks {
		text := notionBlockText(raw)
		if line := notionBlockMarkdownLine(raw, text, depth); line != "" {
			lines = append(lines, line)
		}
		if children := notionBlockChildren(raw); len(children) > 0 {
			lines = append(lines, notionBlockLines(children, depth+1)...)
		}
	}
	return lines
}

func notionBlockMarkdownLine(raw map[string]interface{}, text string, depth int) string {
	blockType := notionString(raw["type"])
	indent := strings.Repeat("  ", depth)
	switch blockType {
	case "heading_1":
		return notionLineIfText("# ", text)
	case "heading_2":
		return notionLineIfText("## ", text)
	case "heading_3":
		return notionLineIfText("### ", text)
	case "bulleted_list_item", "toggle":
		return notionLineIfText(indent+"- ", text)
	case "numbered_list_item":
		return notionLineIfText(indent+"1. ", text)
	case "to_do":
		marker := "[ ] "
		if notionBlockTyped(raw)["checked"] == true {
			marker = "[x] "
		}
		return notionLineIfText(indent+"- "+marker, text)
	case "quote", "callout":
		return notionLineIfText(indent+"> ", text)
	case "code":
		language := notionString(notionBlockTyped(raw)["language"])
		if text == "" {
			return ""
		}
		return indent + "```" + language + "\n" + text + "\n" + indent + "```"
	case "divider":
		return indent + "---"
	case "child_page":
		return notionLineIfText(indent+"- 页面: ", text)
	case "child_database":
		return notionLineIfText(indent+"- 数据源: ", text)
	case "table_row":
		text = strings.TrimSpace(text)
		if text == "" {
			return ""
		}
		return indent + "| " + text + " |"
	default:
		return notionLineIfText(indent, text)
	}
}

func notionLineIfText(prefix, text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return prefix + text
}

func notionBlockChildren(raw map[string]interface{}) []map[string]interface{} {
	switch children := raw["children"].(type) {
	case []map[string]interface{}:
		return children
	case []interface{}:
		items := make([]map[string]interface{}, 0, len(children))
		for _, child := range children {
			if item, ok := child.(map[string]interface{}); ok {
				items = append(items, item)
			}
		}
		return items
	default:
		return nil
	}
}

var NotionPageDetailTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "Notion 页面详情",
		Desc:       "使用当前用户已授权的 Notion 连接器调用 /v1/pages/{page_id}，读取页面属性。页面正文需要使用 Block 子内容接口读取。",
		Tags:       []string{"Notion", "连接器", "页面"},
		Connectors: []string{"notion"},
		ConnectorEndpoints: []app.ConnectorEndpoint{
			notionEndpoint("GET", "/v1/pages/{page_id}", "读取页面详情"),
		},
		Request:          &NotionPageDetailReq{},
		Response:         &NotionPageDetailResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{"page_id": notionPageOnSelectFuzzy},
	},
}

var NotionPageContentTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "Notion 页面内容",
		Desc:       "使用当前用户已授权的 Notion 连接器递归读取页面 Block 内容，并拼成适合阅读的正文预览。",
		Tags:       []string{"Notion", "连接器", "页面内容"},
		Connectors: []string{"notion"},
		ConnectorEndpoints: []app.ConnectorEndpoint{
			notionEndpoint("GET", "/v1/pages/{page_id}", "读取页面详情"),
			notionEndpoint("GET", "/v1/blocks/{block_id}/children", "读取页面 Block 子内容"),
		},
		Request:          &NotionPageContentReq{},
		Response:         &NotionPageContentResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{"page_id": notionPageOnSelectFuzzy},
	},
}

var NotionBlockChildrenTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "Notion 页面/Block 子内容",
		Desc:       "留空时展示已授权页面和数据源概览；选择页面或填写 Block ID 后调用 /v1/blocks/{block_id}/children 读取第一层子内容。",
		Tags:       []string{"Notion", "连接器", "Block", "页面内容"},
		Connectors: []string{"notion"},
		ConnectorEndpoints: []app.ConnectorEndpoint{
			notionEndpoint("GET", "/v1/blocks/{block_id}/children", "读取 Block 子内容"),
		},
		Request:          &NotionBlockChildrenReq{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{"block_id": notionPageOnSelectFuzzy},
	},
	AutoCrudTable: &NotionBlockItem{},
}

func init() {
	packageContext.POST("page_detail.form", NotionPageDetail, NotionPageDetailTemplate)
	packageContext.POST("page_content.form", NotionPageContent, NotionPageContentTemplate)
	packageContext.GET("block_children.table", NotionBlockChildren, NotionBlockChildrenTemplate)
}
