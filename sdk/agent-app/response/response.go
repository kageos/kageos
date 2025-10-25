package response

import (
	"fmt"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"gorm.io/gorm"
)

type RunFunctionResp struct {
	Type      string     `json:"type"`
	TableData *TableData `json:"table_data"`
	FormData  *FormData  `json:"form_data"`
	err       error
}

func (r *RunFunctionResp) Data() interface{} {
	if r.Type == "form" {
		return r.FormData.Data
	}
	if r.Type == "table" {
		return r.TableData
	}
	return nil
}

func (t *RunFunctionResp) AutoSearchFilterPaged(dbAndWhere *gorm.DB, model interface{}, pageInfo *query.SearchFilterPageReq) Table {
	if pageInfo == nil {
		pageInfo = new(query.SearchFilterPageReq)
	}

	// 使用query库的公开方法应用搜索条件
	dbWithConditions, err := query.ApplySearchConditions(dbAndWhere, pageInfo)
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
		t.err = fmt.Errorf("AutoPaginated.Count :%+v failed to count records: %v", t.TableData.Items, err)
		return t
	}

	// 应用排序
	if pageInfo.GetSorts() != "" {
		dbWithConditions = dbWithConditions.Order(pageInfo.GetSorts())
	}

	// 查询当前页数据
	queryDB := dbWithConditions.Offset(offset).Limit(pageSize)

	if err := queryDB.Find(t.TableData.Items).Error; err != nil {
		t.err = fmt.Errorf("AutoPaginated.Find :%+v failed to find records: %v", t.TableData.Items, err)
		return t
	}

	// 计算总页数
	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	// 构造分页结果
	t.TableData.Paginated = &Paginated{
		CurrentPage: pageInfo.Page,
		TotalCount:  int(totalCount),
		TotalPages:  totalPages,
		PageSize:    pageSize,
	}

	return t
}

func (r *RunFunctionResp) Build() error {
	if r.Type == "form" {
		return nil
	}
	return nil
}

type TableData struct {
	Items     interface{} `json:"items"`
	Paginated *Paginated  `json:"paginated"`
}
type FormData struct {
	Data interface{} `json:"data"`
}

type Builder interface {
	Build() error
}

type Response interface {
	Form(data interface{}) Form
	Table(resultList interface{}) Table
}

func (r *RunFunctionResp) Form(data interface{}) Form {
	r.Type = "form"
	r.FormData = &FormData{
		Data: data,
	}
	return r
}
