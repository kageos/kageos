package response

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"gorm.io/gorm"
)

type Table interface {
	Builder
}

type Paginated struct {
	CurrentPage int `json:"current_page"` // 当前页码
	TotalCount  int `json:"total_count"`  // 总数据量
	TotalPages  int `json:"total_pages"`  // 总页数
	PageSize    int `json:"page_size"`    // 每页数量
}

//type table struct {
//	Code      string      `json:"title"`
//	Items     interface{} `json:"values"`
//	Pagination Paginated   `json:"pagination"`
//}

// Table 返回表格响应。传入 db、model、pageInfo 三个额外参数时，Build 会执行分页查询。
func (r *RunFunctionResp) Table(resultList interface{}, queryArgs ...interface{}) Table {
	r.TableData = &TableData{
		Items: resultList,
	}
	r.Type = "table"
	r.configureTableQuery(queryArgs...)

	return r
}

func (r *RunFunctionResp) configureTableQuery(queryArgs ...interface{}) {
	if len(queryArgs) == 0 {
		return
	}
	if len(queryArgs) != 3 {
		r.err = fmt.Errorf("response.Table expects query args: db, model, pageInfo")
		return
	}

	dbAndWhere, ok := queryArgs[0].(*gorm.DB)
	if !ok || dbAndWhere == nil {
		r.err = fmt.Errorf("response.Table query db must be *gorm.DB")
		return
	}
	if queryArgs[1] == nil {
		r.err = fmt.Errorf("response.Table model must not be nil")
		return
	}

	pageInfo, ok := queryArgs[2].(*query.PageSortReq)
	if queryArgs[2] != nil && !ok {
		r.err = fmt.Errorf("response.Table pageInfo must be *query.PageSortReq")
		return
	}
	if pageInfo == nil {
		pageInfo = new(query.PageSortReq)
	}

	r.tableQueryDB = dbAndWhere
	r.tableQueryModel = queryArgs[1]
	r.tableQueryPageInfo = pageInfo
}
