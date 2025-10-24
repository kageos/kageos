package env

// 这些变量在编译时通过 -X 参数注入
var (
	User    string
	App     string
	Version string
)
