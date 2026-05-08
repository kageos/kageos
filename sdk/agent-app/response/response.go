package response

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/chart"
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

	// Table 查询参数（延迟到 Build 时执行）
	tableQueryDB       *gorm.DB
	tableQueryModel    interface{}
	tableQueryPageInfo *query.PageSortReq
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

type BizErr struct {
	Msg string `json:"msg"`
}

func (e *BizErr) Error() string {
	return e.Msg
}

func (r *RunFunctionResp) Build() error {
	if r.err != nil {
		return r.err
	}

	if r.BizError != nil {
		return &BizErr{Msg: fmt.Sprintf("%v", r.BizError)}
	}

	if r.Type == "form" {
		return nil
	}

	if r.Type == "chart" {
		return nil
	}

	// 如果是 table 类型且有查询参数，执行分页查询
	if r.Type == "table" {
		if r.tableQueryDB != nil && r.tableQueryModel != nil {
			return r.executeTableQuery()
		} else {
			//todo
		}

	}

	return nil
}

// executeTableQuery 执行显式筛选后的分页查询，只处理 Count、排序、Offset、Limit、Find。
func (t *RunFunctionResp) executeTableQuery() error {
	if t.tableQueryPageInfo == nil {
		t.tableQueryPageInfo = new(query.PageSortReq)
	}

	dbWithConditions := t.tableQueryDB.Session(&gorm.Session{})

	// 获取分页大小
	pageSize := t.tableQueryPageInfo.GetLimit()
	offset := t.tableQueryPageInfo.GetOffset()

	// 查询总数
	var totalCount int64
	if err := dbWithConditions.Session(&gorm.Session{}).Model(t.tableQueryModel).Count(&totalCount).Error; err != nil {
		t.err = fmt.Errorf("Table.Count :%+v failed to count records: %v", t.TableData.Items, err)
		return t.err
	}

	// 应用排序
	if t.tableQueryPageInfo.GetSorts() != "" {
		dbWithConditions = dbWithConditions.Order(t.tableQueryPageInfo.GetSorts())
	}

	// 查询当前页数据
	queryDB := dbWithConditions.Offset(offset).Limit(pageSize)
	if err := queryDB.Find(t.TableData.Items).Error; err != nil {
		t.err = fmt.Errorf("Table.Find :%+v failed to find records: %v", t.TableData.Items, err)
		return t.err
	}

	// 计算总页数
	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	// 构造分页结果
	t.TableData.Paginated = &Paginated{
		CurrentPage: t.tableQueryPageInfo.GetPage(),
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
	Chart chart.Charter `json:"chart"` // Charter 实现体，resp.Chart() 时调用 SetChartType 注入 ChartType / Series.Type
}

type Builder interface {
	Build() error
}

type Response interface {
	Form(data interface{}) Form
	BizErrorf(format string, a ...any) Form
	Table(resultList interface{}, queryArgs ...interface{}) Table
	Chart(c chart.Charter) Chart
}

func (r *RunFunctionResp) Form(data interface{}) Form {
	r.Type = "form"
	r.FormData = &FormData{
		Data: data,
	}
	return r
}

// Chart 接收 chart.Charter 接口；调用 SetChartType(GetChartType()) 注入 ChartType（及 Series.Type），无需反射
func (r *RunFunctionResp) Chart(c chart.Charter) Chart {
	r.Type = "chart"
	c.SetChartType(c.GetChartType())
	r.ChartData = &ChartData{Chart: c}
	return r
}
