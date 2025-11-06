package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/trace"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/env"
	"github.com/go-playground/form/v4"
)

func newCallbackContext() *Context {
	msgInfo := trace.Msg{
		User:    env.User,
		App:     env.App,
		Version: env.Version,
	}
	return &Context{
		msg:   &msgInfo,
		token: "", // 回调context可能没有token
	}
}
func NewContext(ctx context.Context, req *dto.RequestAppReq) (*Context, error) {
	msgInfo := trace.Msg{
		User:        env.User,
		App:         env.App,
		Version:     env.Version,
		Method:      req.Method,
		Router:      req.Router,
		RequestUser: req.RequestUser,
		TraceId:     req.TraceId,
	}
	//var req dto.RequestAppReq
	//if err := json.Unmarshal(msg.Data, &req); err != nil {
	//	return nil, err
	//}

	return &Context{
		body:     req.Body,
		urlQuery: req.UrlQuery,
		Context:  ctx,
		msg:      &msgInfo,
		token:    req.Token, // ✨ 保存token，用于调用存储服务
	}, nil
}

type Context struct {
	context.Context
	msg      *trace.Msg
	body     []byte
	urlQuery string
	token    string // ✨ Token（用于调用存储服务等）
}

func (c *Context) ShouldBind(req interface{}) error {
	if c.msg == nil {
		return fmt.Errorf("msg is nil")
	}
	if c.body != nil {
		return json.Unmarshal(c.body, req)
	}
	if strings.ToUpper(c.msg.Method) == "GET" {
		if c.urlQuery == "" {
			return nil
		}
		query, err := url.ParseQuery(c.urlQuery)
		if err != nil {
			return fmt.Errorf("解析查询参数失败: %w", err)
		}
		err = form.NewDecoder().Decode(req, query)
		if err != nil {
			return fmt.Errorf("解码表单数据失败: %w", err)
		}
	} else {
		return json.Unmarshal(c.body, req)
	}
	return nil
}

func (c *Context) ShouldBindValidate(req interface{}) error {
	if c.msg == nil {
		return fmt.Errorf("msg is nil")
	}

	if c.body != nil {
		return json.Unmarshal(c.body, req)
	}
	if strings.ToUpper(c.msg.Method) == "GET" {
		if c.urlQuery == "" {
			return nil
		}
		query, err := url.ParseQuery(c.urlQuery)
		if err != nil {
			return fmt.Errorf("解析查询参数失败: %w", err)
		}
		err = form.NewDecoder().Decode(req, query)
		if err != nil {
			return fmt.Errorf("解码表单数据失败: %w", err)
		}
	} else {
		return json.Unmarshal(c.body, req)
	}
	return nil
}
