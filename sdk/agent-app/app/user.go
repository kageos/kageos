package app

import "github.com/kageos/kageos/pkg/contextx"

func (c *Context) GetRequestUser() string {
	if c == nil || c.msg == nil {
		return ""
	}
	return c.msg.RequestUser
}

func (c *Context) GetRequestUserDept() string {
	if c == nil || c.msg == nil {
		return ""
	}
	return c.msg.RequestUserDept
}

func (c *Context) GetClientSource() string {
	if c == nil {
		return ""
	}
	if c.msg != nil && c.msg.ClientSource != "" {
		return c.msg.ClientSource
	}
	if c.Context == nil {
		return ""
	}
	return contextx.GetClientSource(c)
}
