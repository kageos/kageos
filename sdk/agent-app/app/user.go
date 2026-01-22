package app

func (c *Context) GetRequestUser() string {
	return c.msg.RequestUser
}

func (c *Context) GetRequestUserDept() string {
	return c.msg.RequestUserDept
}
