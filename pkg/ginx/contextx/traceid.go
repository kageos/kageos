package contextx

import (
	"context"

	"github.com/gin-gonic/gin"
)

func GetTraceId(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		value := c.Value("trace_id")
		return value.(string)
	}
	return v.GetString("trace_id")
}

func GetUserInfo(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		value := c.Value("user")
		return value.(string)
	}
	return v.GetString("user")
}

func GetTenantUser(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		value := c.Value("tenant_user")
		if value != nil {
			return value.(string)
		}
	}
	if v != nil {
		return v.GetString("tenant_user")
	}
	return ""
}

func GetRequestUser(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		value := c.Value("request_user")
		if value != nil {
			return value.(string)
		}
	}
	if v != nil {
		return v.GetString("request_user")
	}
	return ""
}
