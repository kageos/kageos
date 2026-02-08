package response

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
	"gorm.io/gorm"
)

type RunFunctionResp struct {
	Type      string     `json:"type"`
	TableData *TableData `json:"table_data"`
	FormData  *FormData  `json:"form_data"`
	ChartData *ChartData `json:"chart_data"`

	//系统错误
	err error

	//是否是业务错误？
	BizError interface{}

	// AutoSearchFilterPaged 参数（延迟到 Build 时执行）
	autoPagedDB       *gorm.DB
	autoPagedModel    interface{}
	autoPagedPageInfo *query.SearchFilterPageReq
}

func (r *RunFunctionResp) Data() interface{} {
	if r.Type == "form" {
		return r.FormData.Data
	}
	if r.Type == "table" {
		return r.TableData
	}
	if r.Type == "chart" {
		return r.ChartData
	}
	return nil
}

func (t *RunFunctionResp) AutoSearchFilterPaged(dbAndWhere *gorm.DB, model interface{}, pageInfo *query.SearchFilterPageReq) Table {
	// 只保存参数，延迟到 Build 时执行
	t.autoPagedDB = dbAndWhere
	t.autoPagedModel = model
	if pageInfo == nil {
		t.autoPagedPageInfo = new(query.SearchFilterPageReq)
	} else {
		t.autoPagedPageInfo = pageInfo
	}
	return t
}

type BizErr struct {
	Msg string `json:"msg"`
}

func (e *BizErr) Error() string {
	return e.Msg
}

func (r *RunFunctionResp) Build() error {
	if r.BizError != nil {
		return &BizErr{Msg: fmt.Sprintf("%v", r.BizError)}
	}

	if r.Type == "form" {
		return nil
	}

	if r.Type == "chart" {
		return nil
	}

	// 如果是 table 类型且有自动分页参数，执行查询
	if r.Type == "table" {
		if r.autoPagedDB != nil && r.autoPagedModel != nil {
			return r.executeAutoSearchFilterPaged()
		} else {
			//todo 
		}

	}

	return nil
}

// executeAutoSearchFilterPaged 执行自动搜索、过滤和分页
func (t *RunFunctionResp) executeAutoSearchFilterPaged() error {
	if t.autoPagedPageInfo == nil {
		t.autoPagedPageInfo = new(query.SearchFilterPageReq)
	}

	// 使用query库的公开方法应用搜索条件
	dbWithConditions, err := query.ApplySearchConditions(t.autoPagedDB, t.autoPagedPageInfo)
	if err != nil {
		t.err = fmt.Errorf("AutoPaginated.ApplySearchConditions failed: %v", err)
		return t.err
	}

	// 获取分页大小
	pageSize := t.autoPagedPageInfo.GetLimit()
	offset := t.autoPagedPageInfo.GetOffset()

	// 查询总数
	var totalCount int64
	if err := dbWithConditions.Model(t.autoPagedModel).Count(&totalCount).Error; err != nil {
		t.err = fmt.Errorf("AutoPaginated.Count :%+v failed to count records: %v", t.TableData.Items, err)
		return t.err
	}

	// 应用排序
	if t.autoPagedPageInfo.GetSorts() != "" {
		dbWithConditions = dbWithConditions.Order(t.autoPagedPageInfo.GetSorts())
	}

	// 查询当前页数据
	queryDB := dbWithConditions.Offset(offset).Limit(pageSize)

	if err := queryDB.Find(t.TableData.Items).Error; err != nil {
		t.err = fmt.Errorf("AutoPaginated.Find :%+v failed to find records: %v", t.TableData.Items, err)
		return t.err
	}

	// 计算总页数
	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	// 构造分页结果
	t.TableData.Paginated = &Paginated{
		CurrentPage: t.autoPagedPageInfo.Page,
		TotalCount:  int(totalCount),
		TotalPages:  totalPages,
		PageSize:    pageSize,
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

type ChartData struct {
	Chart types.Charter `json:"chart"` // Charter 实现体，resp.Chart() 时调用 SetChartType 注入 ChartType / Series.Type
}

type Builder interface {
	Build() error
}

type Response interface {
	Form(data interface{}) Form
	BizErrorf(format string, a ...any) Form
	Table(resultList interface{}) Table
	Chart(chart types.Charter) Chart
}

func (r *RunFunctionResp) Form(data interface{}) Form {
	r.Type = "form"
	r.FormData = &FormData{
		Data: data,
	}
	return r
}

// Chart 接收 Charter 接口；调用 SetChartType(GetChartType()) 注入 ChartType（及 Series.Type），无需反射
func (r *RunFunctionResp) Chart(chart types.Charter) Chart {
	r.Type = "chart"
	chart.SetChartType(chart.GetChartType())
	r.ChartData = &ChartData{Chart: chart}
	return r
}
