package response

import (
	"fmt"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"

	"gorm.io/gorm"
)

type Table interface {
	Builder
	AutoSearchFilterPaged(dbAndWhere *gorm.DB, model interface{}, pageInfo *query.SearchFilterPageReq) Table
}

type paginated struct {
	CurrentPage int `json:"current_page"` // 当前页码
	TotalCount  int `json:"total_count"`  // 总数据量
	TotalPages  int `json:"total_pages"`  // 总页数
	PageSize    int `json:"page_size"`    // 每页数量
}

type tableData struct {
	err  error
	val  interface{}
	resp *RunFunctionResp
	Data table
}
type table struct {
	Title      string      `json:"title"`
	Values     interface{} `json:"values"`
	Pagination paginated   `json:"pagination"`
}

func newTable(resp *RunFunctionResp, resultList interface{}) *tableData {
	return &tableData{resp: resp, val: resultList, Data: table{}}
}
func (r *RunFunctionResp) Table(resultList interface{}) Table {
	return newTable(r, resultList)
}

func (t *tableData) Build() error {
	if t.err != nil {
		return t.err
	}
	t.Data.Values = t.val

	return nil
}

func (t *tableData) AutoSearchFilterPaged(db *gorm.DB, model interface{}, pageInfo *query.SearchFilterPageReq) Table {

	if pageInfo == nil {
		pageInfo = new(query.SearchFilterPageReq)
	}

	// 使用query库的公开方法应用搜索条件
	dbWithConditions, err := query.ApplySearchConditions(db, pageInfo)
	if err != nil {
		t.err = fmt.Errorf("AutoPaginated.ApplySearchConditions failed: %v", err)
		return t
	}

	// 获取分页大小
	pageSize := pageInfo.GetLimit()
	offset := pageInfo.GetOffset()

	// 查询总数
	var totalCount int64
	if err := dbWithConditions.Model(model).Count(&totalCount).Error; err != nil {
		t.err = fmt.Errorf("AutoPaginated.Count :%+v failed to count records: %v", t.val, err)
		return t
	}

	// 应用排序
	if pageInfo.GetSorts() != "" {
		dbWithConditions = dbWithConditions.Order(pageInfo.GetSorts())
	}

	// 查询当前页数据
	queryDB := dbWithConditions.Offset(offset).Limit(pageSize)

	if err := queryDB.Find(t.val).Error; err != nil {
		t.err = fmt.Errorf("AutoPaginated.Find :%+v failed to find records: %v", t.val, err)
		return t
	}

	// 计算总页数
	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	// 构造分页结果
	t.Data.Pagination = paginated{
		CurrentPage: pageInfo.Page,
		TotalCount:  int(totalCount),
		TotalPages:  totalPages,
		PageSize:    pageSize,
	}

	return t
}
