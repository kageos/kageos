package app

import "github.com/ai-agent-os/ai-agent-os/pkg/contextx"

func (c *Context) GetRequestUser() string {
	return c.msg.RequestUser
}

func (c *Context) GetRequestUserDept() string {
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
