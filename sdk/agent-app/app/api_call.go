package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/contextx"
)

// APICall calls an KageOS gateway API with the current request context.
//
// It is intentionally thin: SDK code passes method/path/body/response, and the
// platform side handles auth, permission checks and auditing based on the
// propagated token, trace and user context.
func (c *Context) APICall(method, path string, reqBody interface{}, respData interface{}) error {
	if c == nil {
		return fmt.Errorf("context is nil")
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	path = strings.TrimSpace(path)
	if method == "" {
		return fmt.Errorf("APICall method cannot be empty")
	}
	if path == "" {
		return fmt.Errorf("APICall path cannot be empty")
	}
	return apicall.CallAPI(c.apiCallContext(), method, path, reqBody, respData)
}

func (c *Context) apiCallContext() context.Context {
	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}
	sourceType := strings.TrimSpace(contextx.GetSourceType(ctx))
	sourceRef := strings.TrimSpace(contextx.GetSourceRef(ctx))
	if sourceType == "" && sourceRef == "" {
		if fullCodePath := strings.TrimSpace(c.GetFullCodePath()); fullCodePath != "" {
			sourceType = "sdk_function"
			sourceRef = fullCodePath
		}
	}
	return contextx.WithRequestInfo(ctx, contextx.RequestInfo{
		TraceId:            c.GetTraceId(),
		RequestUser:        c.GetRequestUser(),
		Token:              c.token,
		DepartmentFullPath: c.GetRequestUserDept(),
		ClientSource:       c.GetClientSource(),
		SourceType:         sourceType,
		SourceRef:          sourceRef,
	})
}
