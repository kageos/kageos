package response

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/chart"
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
	Table(result TableResult) Table
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
