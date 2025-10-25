package app

func (c *Context) GetRequestUser() string {
	return c.msg.RequestUser
}
