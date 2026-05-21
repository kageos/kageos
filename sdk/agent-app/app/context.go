package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-playground/form/v4"
	"github.com/go-playground/validator/v10"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/trace"
	"github.com/kageos/kageos/sdk/agent-app/env"
)

var (
	// defaultValidator 使用 struct 标签 "validate"（如 validate:"required,min=1"）
	defaultValidator *validator.Validate
)

func init() {
	defaultValidator = validator.New()
}

func newCallbackContext(info *routerInfo) *Context {
	msgInfo := trace.Msg{
		User:    env.User,
		App:     env.App,
		Version: env.Version,
	}
	return &Context{
		msg:        &msgInfo,
		routerInfo: info,
		token:      "", // 回调context可能没有token
	}
}
func (a *App) NewContext(ctx context.Context, req *dto.RequestAppReq) (*Context, error) {
	msgInfo := trace.Msg{
		User:            env.User,
		App:             env.App,
		Version:         env.Version,
		Method:          req.Method,
		Router:          req.Router,
		RequestUser:     req.RequestUser,
		RequestUserDept: req.RequestUserDept,
		ClientSource:    strings.TrimSpace(req.ClientSource),
		TraceId:         req.TraceId,
	}
	//var req dto.RequestAppReq
	//if err := json.Unmarshal(msg.Data, &req); err != nil {
	//	return nil, err
	//}
	ctx = contextx.WithRequestInfo(ctx, contextx.RequestInfo{
		TraceId:            msgInfo.TraceId,
		RequestUser:        msgInfo.RequestUser,
		Token:              req.Token,
		DepartmentFullPath: msgInfo.RequestUserDept,
		ClientSource:       msgInfo.ClientSource,
	})
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
	msg        *trace.Msg
	body       []byte
	urlQuery   string
	token      string      // ✨ Token（用于调用存储服务等）
	routerInfo *routerInfo // 当前请求对应的路由信息（包含 PackagePath）
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

// ShouldBindValidate 绑定请求体/查询参数并在绑定成功后做 struct 校验。
// 使用 go-playground/validator 语法，在字段上写 validate 标签即可，如 validate:"required,min=1"。
func (c *Context) ShouldBindValidate(req interface{}) error {
	if c.msg == nil {
		return fmt.Errorf("msg is nil")
	}

	if err := c.ShouldBind(req); err != nil {
		return err
	}

	if err := defaultValidator.Struct(req); err != nil {
		return wrapValidationError(err)
	}
	return nil
}

// wrapValidationError 将 validator 错误转为可读文案（只取第一个错误）
func wrapValidationError(err error) error {
	if err == nil {
		return nil
	}
	if vErr, ok := err.(validator.ValidationErrors); ok && len(vErr) > 0 {
		e := vErr[0]
		return fmt.Errorf("参数校验失败: %s 不满足规则 %s", e.Field(), e.Tag())
	}
	return err
}

// GetTraceId 获取当前请求的 TraceId（用于日志串联）
func (ctx *Context) GetTraceId() string {
	if ctx.msg != nil {
		return ctx.msg.TraceId
	}
	return ""
}

// GetFullCodePath 获取当前函数的完整目录路径，用于审计和消息来源标记。
func (ctx *Context) GetFullCodePath() string {
	if ctx != nil && ctx.msg != nil {
		return ctx.msg.GetFullRouter()
	}
	return ""
}

// GetRouterGroup 获取当前请求的 RouterGroup
// 返回当前请求所属的 RouterGroup 路径（如 "/tools/pdftools"）
// 如果无法获取（系统路由或未设置），返回空字符串
func (ctx *Context) GetRouterGroup() string {
	if ctx.routerInfo != nil && ctx.routerInfo.Options != nil {
		// 从 PackagePath 构建 RouterGroup 路径（单一数据源）
		packagePath := ctx.routerInfo.Options.PackagePath
		if packagePath != "" {
			return "/" + strings.Trim(packagePath, "/")
		}
	}
	return ""
}

// GetFunctionTemplate 根据函数路径获取函数模板
// 利用路由系统优化：URL 唯一，直接获取，不需要遍历 method
func (ctx *Context) GetFunctionTemplate(functionPath string) (Templater, error) {
	// 1. 构建完整的路由路径
	var fullRouter string
	if strings.HasPrefix(functionPath, "/") {
		// 绝对路径，直接使用
		fullRouter = strings.Trim(functionPath, "/")
	} else {
		// 相对路径，需要加上当前 RouterGroup
		routerGroup := ctx.GetRouterGroup()
		if routerGroup == "" {
			return nil, fmt.Errorf("无法获取当前 RouterGroup，请使用绝对路径")
		}
		fullRouter = fmt.Sprintf("%s/%s", strings.Trim(routerGroup, "/"), strings.Trim(functionPath, "/"))
	}

	// 2. 从 app.routerInfo 中查找路由信息（URL 唯一，不需要 method）
	// 需要通过 Context 获取 App 实例，但 Context 没有直接引用 App
	// 所以需要通过 routerInfo 来获取，或者使用全局 app 变量
	// 这里使用全局 app 变量（与 register.go 中的用法一致）
	if app == nil {
		return nil, fmt.Errorf("app 未初始化")
	}
	router, err := app.getRoute(fullRouter)
	if err != nil {
		return nil, fmt.Errorf("未找到函数 %s 的路由信息: %w", functionPath, err)
	}

	return router.Template, nil
}
