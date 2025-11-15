package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
)

type PackageContext struct {
	RouterGroup string `json:"router_group"`
}

type RouterGroup struct {
	RouterGroup      string `json:"router_group"`
	*RouterGroupInfo `json:"router_group_info"`
}

// RegisterOptions 路由注册选项
type RegisterOptions struct {
	PackagePath string // 服务目录路径（package路径），用于获取对应的数据库连接
	RouterGroup *RouterGroup
}

func (r *RegisterOptions) GetDBName(user string, app string) string {
	trim := strings.Trim(r.PackagePath, "/")
	split := strings.Split(trim, "/")
	join := strings.Join(split, "_")
	dbName := fmt.Sprintf("%s_%s_%s.db", user, app, join)
	return dbName
}

// registerWithGroup 通用的注册方法，处理 FunctionGroup 设置和路由路径构建
func (p *RouterGroup) registerWithGroup(method string, router string, handleFunc HandleFunc, templater Templater) {

	// 确保 app 已初始化
	if app == nil {
		initApp()
	}
	// 设置 FunctionGroup
	config := templater.GetBaseConfig()
	config.FunctionGroup.Name = p.GroupName
	config.FunctionGroup.Code = p.GroupCode

	packagePath := strings.Trim(p.RouterGroup, "/")
	// 构建完整路由路径
	fullRouter := fmt.Sprintf("/%s/%s", packagePath, strings.Trim(router, "/"))

	// 创建 options，设置 PackagePath
	options := &RegisterOptions{
		PackagePath: packagePath,
		RouterGroup: p,
	}

	// 根据方法类型调用对应的注册函数，传递 options
	switch method {
	case "GET":
		GET(fullRouter, handleFunc, templater, options)
	case "POST":
		POST(fullRouter, handleFunc, templater, options)
	case "PUT":
		PUT(fullRouter, handleFunc, templater, options)
	case "DELETE":
		DELETE(fullRouter, handleFunc, templater, options)
	default:
		logger.Errorf(context.Background(), "unsupported HTTP method: %s", method)
	}
}

func (p *RouterGroup) GET(router string, handleFunc HandleFunc, templater Templater) {
	p.registerWithGroup("GET", router, handleFunc, templater)
}

func (p *RouterGroup) POST(router string, handleFunc HandleFunc, templater Templater) {
	p.registerWithGroup("POST", router, handleFunc, templater)
}

func (p *RouterGroup) PUT(router string, handleFunc HandleFunc, templater Templater) {
	p.registerWithGroup("PUT", router, handleFunc, templater)
}

func (p *RouterGroup) DELETE(router string, handleFunc HandleFunc, templater Templater) {
	p.registerWithGroup("DELETE", router, handleFunc, templater)
}

type RouterGroupInfo struct {
	GroupCode string `json:"group_code"`
	GroupName string `json:"group_name"`
}

func NewRouterGroup(pkgCtx *PackageContext, routerGroup *RouterGroupInfo) *RouterGroup {

	return &RouterGroup{
		RouterGroup:     pkgCtx.RouterGroup,
		RouterGroupInfo: routerGroup,
	}
}

func routerKey(router string, method string) string {
	router = strings.Trim(router, "/")
	key := router + "." + method
	return key
}

func register(router string, method string, handleFunc HandleFunc, templater Templater, options *RegisterOptions) {
	// 确保 app 已初始化
	if app == nil {
		initApp()
	}

	// 如果初始化失败，app 可能仍然是 nil，延迟注册到 Run() 时
	if app == nil {
		logger.Errorf(context.Background(), "Cannot register router %s %s: app initialization failed", method, router)
		return
	}

	app.routerInfo[routerKey(router, method)] = &routerInfo{
		HandleFunc: handleFunc,
		Router:     router,
		Method:     method,
		Options:    options,
		Template:   templater,
	}
}

// GET 注册 GET 路由
// options 可以为 nil，表示使用默认值（PackagePath 为空）
func GET(router string, handleFunc HandleFunc, templater Templater, options ...*RegisterOptions) {
	var opts *RegisterOptions
	if len(options) > 0 {
		opts = options[0]
	}
	register(router, "GET", handleFunc, templater, opts)
}

// POST 注册 POST 路由
// options 可以为 nil，表示使用默认值（PackagePath 为空）
func POST(router string, handleFunc HandleFunc, templater Templater, options ...*RegisterOptions) {
	var opts *RegisterOptions
	if len(options) > 0 {
		opts = options[0]
	}
	register(router, "POST", handleFunc, templater, opts)
}

// PUT 注册 PUT 路由
// options 可以为 nil，表示使用默认值（PackagePath 为空）
func PUT(router string, handleFunc HandleFunc, templater Templater, options ...*RegisterOptions) {
	var opts *RegisterOptions
	if len(options) > 0 {
		opts = options[0]
	}
	register(router, "PUT", handleFunc, templater, opts)
}

// DELETE 注册 DELETE 路由
// options 可以为 nil，表示使用默认值（PackagePath 为空）
func DELETE(router string, handleFunc HandleFunc, templater Templater, options ...*RegisterOptions) {
	var opts *RegisterOptions
	if len(options) > 0 {
		opts = options[0]
	}
	register(router, "DELETE", handleFunc, templater, opts)
}

func initRouter(a *App) {
	//a.registerRouter(MethodPost, "/test/add", AddHandle, Temp)
	//a.registerRouter(MethodPost, "/test/get", GetHandle, Temp)

	// ⚠️ 重要：必须直接操作 a.routerInfo，不能调用 register() 或 a.registerRouter()
	//
	// 原因：死锁问题
	// 1. initRouter() 在 NewApp() 中被调用
	// 2. NewApp() 本身在 initApp() 的 sync.Once.Do() 中执行
	// 3. 此时全局变量 app 还没有被赋值（NewApp() 还没返回）
	// 4. 如果调用 register()，它会检查 app == nil，然后再次调用 initApp()
	// 5. sync.Once.Do() 会阻塞等待第一次执行完成，但第一次执行就是 NewApp()
	// 6. 而 NewApp() 又调用了 initRouter()，形成死锁
	//
	// 解决方案：直接操作传入的 App 实例的 routerInfo，避免触发全局 app 的检查
	a.routerInfo[routerKey("/_callback", MethodPost)] = &routerInfo{
		HandleFunc: a.CallbackRouter,
		Router:     "/_callback",
		Method:     MethodPost,
		Options:    nil, // 系统路由没有 PackagePath
		Template:   &FormTemplate{},
	}
	a.routerInfo[routerKey("/_callback", MethodGet)] = &routerInfo{
		HandleFunc: a.CallbackRouter,
		Router:     "/_callback",
		Method:     MethodGet,
		Options:    nil,
		Template:   &FormTemplate{},
	}
	a.routerInfo[routerKey("/_callback", MethodDelete)] = &routerInfo{
		HandleFunc: a.CallbackRouter,
		Router:     "/_callback",
		Method:     MethodDelete,
		Options:    nil,
		Template:   &FormTemplate{},
	}
	a.routerInfo[routerKey("/_callback", MethodPut)] = &routerInfo{
		HandleFunc: a.CallbackRouter,
		Router:     "/_callback",
		Method:     MethodPut,
		Options:    nil,
		Template:   &FormTemplate{},
	}
}

type CallbackRouterReq struct {
	Type   string `json:"type" binding:"required" example:""`
	Method string `json:"method" binding:"required" example:""`
	Router string `json:"router" binding:"required" example:"/users/app/xxxx"`
	Body   []byte `json:"body" example:"eyJpZCI6MX0="`
}

func (a *App) CallbackRouter(ctx *Context, resp response.Response) error {
	var req CallbackRouterReq
	if err := json.Unmarshal(ctx.body, &req); err != nil {
		logger.Errorf(ctx, "CallbackRouter Unmarshal body:%s err: %v", ctx.body, err)
		return err
	}

	router, err := a.getRouter(req.Router, req.Method)
	if err != nil {
		return err
	}

	//callback只是代理路由，要重定向到真正的路由
	ctx.msg.Router = req.Router
	ctx.msg.Method = req.Method
	ctx.body = req.Body
	// 设置 routerInfo，方便后续获取 PackagePath
	ctx.routerInfo = router

	switch req.Type {
	case CallbackTypeOnTableAddRow:
		v, ok := router.Template.(*TableTemplate)
		if !ok {
			return errors.New("invalid type of TableTemplate")
		}
		var onTableReq callback.OnTableAddRowReq
		onTableResp, err := v.OnTableAddRow(ctx, &onTableReq)
		if err != nil {
			logger.Errorf(ctx, "callback onTableAddRow router:%s call error:%s", req.Type, err.Error())
			return err
		}
		err = resp.Form(onTableResp).Build()
		if err != nil {
			logger.Errorf(ctx, "callback onTableAddRow  router:%s Build error:%s", req.Type, err.Error())
			return err
		}
		logger.Infof(ctx, "CallbackRouter onTableAddRow success")
		return nil
	case CallbackTypeOnTableUpdateRow:
		v, ok := router.Template.(*TableTemplate)
		if !ok {
			return errors.New("invalid type of TableTemplate")
		}
		var onTableReq callback.OnTableUpdateRowReq
		err := json.Unmarshal(ctx.body, &onTableReq.Updates)
		if err != nil {
			return err
		}
		onTableResp, err := v.OnTableUpdateRow(ctx, &onTableReq)
		if err != nil {
			return err
		}
		err = resp.Form(onTableResp).Build()
		if err != nil {
			logger.Errorf(ctx, "callback OnTableUpdateRows router:%s error:%s", req.Type, err.Error())
			return err
		}
		logger.Infof(ctx, "CallbackRouter OnTableUpdateRows success")
		return nil
	case CallbackTypeOnTableDeleteRows:
		v, ok := router.Template.(*TableTemplate)
		if !ok {
			return errors.New("invalid type of TableTemplate")
		}
		var onTableReq callback.OnTableDeleteRowsReq
		err := json.Unmarshal(ctx.body, &onTableReq)
		if err != nil {
			return err
		}
		onTableResp, err := v.OnTableDeleteRows(ctx, &onTableReq)
		if err != nil {
			return err
		}
		err = resp.Form(onTableResp).Build()
		if err != nil {
			logger.Errorf(ctx, "callback OnTableDeleteRows router:%s error:%s", req.Type, err.Error())
			return err
		}
		logger.Infof(ctx, "CallbackRouter OnTableDeleteRows success")
		return nil
	case CallbackTypeOnSelectFuzzy:
		var onCallback callback.OnSelectFuzzyReq
		base := router.Template.GetBaseConfig()
		err := json.Unmarshal(ctx.body, &onCallback)
		if err != nil {
			return err
		}

		fuzzy := base.OnSelectFuzzyMap[onCallback.Code]
		if fuzzy == nil {
			return errors.New("invalid code " + onCallback.Code)
		}
		fuzzyResp, err := fuzzy(ctx, &onCallback)
		if err != nil {
			return err
		}
		err = resp.Form(fuzzyResp).Build()
		if err != nil {
			logger.Errorf(ctx, "callback OnSelectFuzzy router:%s error:%s", req.Type, err.Error())
			return err
		}
		logger.Infof(ctx, "CallbackRouter OnSelectFuzzy success")
	}
	return nil

}
