package app

type Templater interface {
}

type BaseConfig struct {
	// 名称配置
	Code string   `json:"code"`
	Name string   `json:"name"`
	Desc string   `json:"desc"`
	Tags []string `json:"tags"`

	// 请求响应
	Request  interface{} `json:"-"`
	Response interface{} `json:"-"`
}
