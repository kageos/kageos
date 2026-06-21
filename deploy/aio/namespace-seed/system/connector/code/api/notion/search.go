package notion

import (
	"net/http"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type NotionSearchReq struct {
	Query             string `json:"query" form:"query" widget:"name:搜索关键词;type:input;placeholder:搜索已授权页面或数据源标题"`
	ObjectType        string `json:"object_type" form:"object_type" widget:"name:对象类型;type:select;options:全部,页面,数据源;options_colors:909399,409EFF,67C23A;render_default:全部"`
	query.PageSortReq `widget:"-"`
}

type NotionSearchItem struct {
	ID                string `json:"id" widget:"name:ID;type:input"`
	Object            string `json:"object" widget:"name:对象;type:input"`
	Title             string `json:"title" widget:"name:标题;type:input"`
	URL               string `json:"url" widget:"name:链接;type:link;target:_blank;link_type:primary"`
	ParentType        string `json:"parent_type" widget:"name:父级类型;type:input"`
	Archived          bool   `json:"archived" widget:"name:归档;type:switch"`
	InTrash           bool   `json:"in_trash" widget:"name:回收站;type:switch"`
	NotionCreatedTime string `json:"notion_created_time" widget:"name:创建时间;type:input"`
	NotionEditedTime  string `json:"notion_edited_time" widget:"name:编辑时间;type:input"`
}

type notionSearchResp struct {
	Results    []map[string]interface{} `json:"results"`
	NextCursor string                   `json:"next_cursor"`
	HasMore    bool                     `json:"has_more"`
}

func NotionSearch(ctx *app.Context, resp response.Response) error {
	var req NotionSearchReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	pageSize := notionPageSize(&req.PageSortReq, 20)
	startCursor := ""
	for i := 1; i < req.PageSortReq.GetPage(); i++ {
		var prev notionSearchResp
		body := notionSearchBody(req.Query, req.ObjectType, startCursor, pageSize)
		if _, err := callNotionJSON(ctx, http.MethodPost, "/v1/search", nil, body, &prev); err != nil {
			return err
		}
		if !prev.HasMore || prev.NextCursor == "" {
			return notionEmptyTable[NotionSearchItem](resp, &req.PageSortReq)
		}
		startCursor = prev.NextCursor
	}

	var payload notionSearchResp
	body := notionSearchBody(req.Query, req.ObjectType, startCursor, pageSize)
	if _, err := callNotionJSON(ctx, http.MethodPost, "/v1/search", nil, body, &payload); err != nil {
		return err
	}
	items := make([]NotionSearchItem, 0, len(payload.Results))
	for _, raw := range payload.Results {
		items = append(items, notionSearchItem(raw))
	}
	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: notionEstimatedTotal(&req.PageSortReq, pageSize, len(items), payload.HasMore),
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func notionSearchItem(raw map[string]interface{}) NotionSearchItem {
	parent, _ := raw["parent"].(map[string]interface{})
	return NotionSearchItem{
		ID:                notionString(raw["id"]),
		Object:            notionString(raw["object"]),
		Title:             notionTitleFromObject(raw),
		URL:               notionString(raw["url"]),
		ParentType:        notionString(parent["type"]),
		Archived:          raw["archived"] == true,
		InTrash:           raw["in_trash"] == true,
		NotionCreatedTime: notionString(raw["created_time"]),
		NotionEditedTime:  notionString(raw["last_edited_time"]),
	}
}

var NotionSearchTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:       "Notion 搜索页面和数据源",
		Desc:       "使用当前用户已授权的 Notion 连接器调用 /v1/search，搜索授权时选择过的页面、数据库或数据源。",
		Tags:       []string{"Notion", "连接器", "搜索", "页面", "数据源"},
		Connectors: []string{"notion"},
		ConnectorEndpoints: []app.ConnectorEndpoint{
			notionEndpoint("POST", "/v1/search", "搜索页面和数据源"),
		},
		Request: &NotionSearchReq{},
	},
	AutoCrudTable: &NotionSearchItem{},
}

func init() {
	packageContext.GET("search.table", NotionSearch, NotionSearchTemplate)
}
