package function

type UserInfo struct {
	Username   string `json:"username"`
	IsLoggedIn bool   `json:"is_logged_in"`
}

type Namespace struct {
	Name string `json:"name"`
}

type RequestInfo struct {
	ServerName string `json:"server_name"`
	Method     string `json:"method"`
	Router     string `json:"router"`
	UrlQuery   string `json:"url_query"`
	Body       []byte `json:"body"`
	Version    string `json:"version"`
}

type ResponseInfo struct {
	Body    []byte `json:"body"`
	Version string `json:"version"`
}

type CallReq struct {
	//追踪id
	TraceId string `json:"trace_id"`
	//所属命名空间
	Namespace *Namespace `json:"namespace"`
	//用户信息
	UserInfo *UserInfo `json:"user_info"`
	//请求信息
	RequestInfo *RequestInfo `json:"request_info"`

	Metadata map[string]interface{} `json:"metadata"`
}
type CallResp struct {
	ResponseInfo *ResponseInfo          `json:"response_info"`
	Metadata     map[string]interface{} `json:"metadata"`
}
