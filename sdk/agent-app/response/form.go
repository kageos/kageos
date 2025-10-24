package response

type Form interface {
	Builder
}
type formData struct {
	resp *RunFunctionResp
	Data interface{} `json:"data"`
}

func newForm(data interface{}, resp *RunFunctionResp) Form {
	return &formData{Data: data, resp: resp}
}

func (d *formData) Build() error {
	return build(d.resp, d.Data)
}
