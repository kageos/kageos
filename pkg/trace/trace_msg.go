package trace

import (
	"fmt"
	"strings"
)

type UserInfo struct {
	Username   string `json:"username"`
	IsLoggedIn bool   `json:"is_logged_in"`
}

type RequestInfo struct {
	Method   string `json:"method"`
	Router   string `json:"router"`
	UrlQuery string `json:"url_query"`
	Body     []byte `json:"body"`
}

type Msg struct {

	//追踪id
	TraceId string `json:"trace_id"`

	//所属命名空间
	User string `json:"user"`

	//所属app
	App string `json:"app"`

	//version
	Version string `json:"version"`

	RequestUser string `json:"request_user" example:"beiluo"`              // 请求用户（由中间件自动填充）
	Router      string `json:"router" binding:"required" example:"/users"` // 路由路径
	Method      string `json:"method" example:"GET"`                       // 应用内部方法名（可选）

}

func (m *Msg) GetFullRouter() string {
	return fmt.Sprintf("/%s/%s/%s", m.User, m.App, strings.Trim(m.Router, "/"))
}
