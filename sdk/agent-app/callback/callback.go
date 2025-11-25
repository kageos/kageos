package callback

import (
	"fmt"
	"github.com/ai-agent-os/ai-agent-os/pkg/jsonx"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/widget"
)

type OnApiCreateReq struct {
}

type OnApiCreateResp struct {
}

type OnPageLoadReq struct {
}

type OnPageLoadResp struct {
}

type OnSelectFuzzyReq struct {
	Code      string      `json:"code"`
	Type      string      `json:"type"`
	Request   interface{} `json:"request"`
	Value     interface{} `json:"value"`
	ValueType string      `json:"value_type"`
}

func (r *OnSelectFuzzyReq) Keyword() string {
	return fmt.Sprintf("%v", r.Value)
}

func (r *OnSelectFuzzyReq) IsByValue() bool {
	return r.Type == "by_value"
}
func (r *OnSelectFuzzyReq) IsByValues() bool {
	return r.Type == "by_values"
}
func (r *OnSelectFuzzyReq) IsByKeyword() bool {
	return r.Type == "by_keyword"
}

func (r *OnSelectFuzzyReq) GetValue() interface{} {
	if r.ValueType == widget.TypeNumber {
		return int(r.Value.(float64))
	}
	return r.Value
}

func (r *OnSelectFuzzyReq) GetValues() interface{} {

	switch r.ValueType {
	case widget.DataTypeFloat, widget.DataTypeFloats:
		switch r.Value.(type) {
		case []interface{}:
			var floats []float64
			jsonx.Convert(r.Value, &floats)
			return floats
		}
	case widget.DataTypeInt, widget.DataTypeInts:
		switch r.Value.(type) {
		case []interface{}:
			var ints []int
			jsonx.Convert(r.Value, &ints)
			return ints
		}

	case widget.DataTypeString, widget.DataTypeStrings:
		switch r.Value.(type) {
		case []interface{}:
			var strs []string
			jsonx.Convert(r.Value, &strs)
			return strs
		}
	}

	return r.Value
}

type SelectFuzzyItem struct {
	Value       interface{}            `json:"value"`
	Label       string                 `json:"label"`
	Icon        string                 `json:"icon"`
	DisplayInfo map[string]interface{} `json:"display_info"`
}

type OnSelectFuzzyResp struct {
	MaxSelections int `json:"max_selections,omitempty"` //为0表示不限制，只有在限制时候需要填写
	//只有在结构体数组或者切片下的select和multiselect组件才会有聚合计算的功能，场景例如收银，我一个[]Orders
	//下面有ProductId，然后每个产品虽然选择产品id，但是DisplayInfo里返回了价格，这时候我想价格求和来计算，statistics"价格":"sum"即可

	Statistics map[string]interface{} `json:"statistics"`
	Items      []*SelectFuzzyItem     `json:"items"`

	ErrorMsg string `json:"error_msg"`
}
