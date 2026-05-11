package response

import "github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"

type Table interface {
	Builder
}

type Paginated struct {
	CurrentPage int   `json:"current_page"` // 当前页码
	TotalCount  int64 `json:"total_count"`  // 总数据量
	TotalPages  int   `json:"total_pages"`  // 总页数
	PageSize    int   `json:"page_size"`    // 每页数量
}

// TableResult 是 Table 的唯一响应输入。业务 Handler 负责从任意数据源拿到当前页数据和总数；
// response 层只负责按前端协议渲染 items + paginated。
type TableResult struct {
	Items      interface{}
	TotalCount int64
	PageInfo   *query.PageSortReq
}

// Table 返回表格响应。数据查询、总数计算和排序分页都由业务 Handler 显式完成。
func (r *RunFunctionResp) Table(result TableResult) Table {
	pageInfo := result.PageInfo
	if pageInfo == nil {
		pageInfo = new(query.PageSortReq)
	}

	pageSize := pageInfo.GetLimit()
	totalPages := 0
	if result.TotalCount > 0 {
		totalPages = int((result.TotalCount + int64(pageSize) - 1) / int64(pageSize))
	}

	r.TableData = &TableData{
		Items: result.Items,
		Paginated: &Paginated{
			CurrentPage: pageInfo.GetPage(),
			TotalCount:  result.TotalCount,
			TotalPages:  totalPages,
			PageSize:    pageSize,
		},
	}
	r.Type = "table"

	return r
}
