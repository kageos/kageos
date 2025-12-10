package plugin

// Context 插件执行上下文
// 包含请求的追踪信息、用户信息等
type Context struct {
	// TraceID 追踪ID（用于日志追踪）
	TraceID string

	// RequestUser 请求用户（实际发起请求的用户）
	RequestUser string
}

// GetTraceID 获取追踪ID
func (c *Context) GetTraceID() string {
	return c.TraceID
}

// GetRequestUser 获取请求用户
func (c *Context) GetRequestUser() string {
	return c.RequestUser
}

