package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"gorm.io/gorm"
)

type PackageContext struct {
	RouterGroup string `json:"router_group"`
	Name        string `json:"name,omitempty"` // 包名称（可选）
	Desc        string `json:"desc,omitempty"` // 包描述（可选）
}

// RegisterOptions 路由注册选项
type RegisterOptions struct {
	PackagePath string // 服务目录路径（package路径），用于获取对应的数据库连接
}

func (r *RegisterOptions) GetDBName(user string, app string) string {
	trim := strings.Trim(r.PackagePath, "/")
	split := strings.Split(trim, "/")
	join := strings.Join(split, "-")
	dbName := fmt.Sprintf("%s.db", join)
	return dbName
}

// BuildFullRouter 构建完整路由路径
// router: 相对路由路径（如 "extract_text"）
// 返回: 完整路由路径（如 "/tools/pdftools/extract_text"）
func (p *PackageContext) BuildFullRouter(router string) string {
	packagePath := strings.Trim(p.RouterGroup, "/")
	return fmt.Sprintf("/%s/%s", packagePath, strings.Trim(router, "/"))
}

// GetGormDB 获取当前包对应的 DB（与请求里 ctx.GetGormDB() 同源），用于 init 里后台任务等无请求上下文的场景
func (p *PackageContext) GetGormDB() (*gorm.DB, error) {
	return GetDBByPackagePath(p.RouterGroup)
}

// register 通用的注册方法，构建路由路径并注册
func (p *PackageContext) register(method string, router string, handleFunc HandleFunc, templater Templater) {
	// 确保 app 已初始化
	if app == nil {
		initApp()
	}

	// 如果初始化失败，app 可能仍然是 nil，延迟注册到 Run() 时
	if app == nil {
		logger.Errorf(context.Background(), "Cannot register router %s %s: app initialization failed", method, router)
		return
	}

	// 构建完整路由路径：RouterGroup + "/" + router
	// 例如："/tools/pdftools" + "/" + "extract_text" -> "/tools/pdftools/extract_text"
	fullRouter := p.BuildFullRouter(router)
	packagePath := strings.Trim(p.RouterGroup, "/") // 从 RouterGroup 提取 PackagePath

	app.packageContexts[packagePath] = p

	// 创建 options，设置 PackagePath（用于获取对应的数据库连接）
	options := &RegisterOptions{
		PackagePath: packagePath,
	}

	// 直接调用 app.addRoute，跳过中间层
	if err := app.addRoute(fullRouter, method, handleFunc, templater, options); err != nil {
		logger.Errorf(context.Background(), "Failed to register router %s %s: %v", method, fullRouter, err)
		panic(err) // 注册失败时 panic，避免静默失败
	}
}

// POST 注册 POST 路由
func (p *PackageContext) POST(router string, handleFunc HandleFunc, templater Templater) {
	p.register("POST", router, handleFunc, templater)
}

// GET 注册 GET 路由
func (p *PackageContext) GET(router string, handleFunc HandleFunc, templater Templater) {
	p.register("GET", router, handleFunc, templater)
}

// PUT 注册 PUT 路由
func (p *PackageContext) PUT(router string, handleFunc HandleFunc, templater Templater) {
	p.register("PUT", router, handleFunc, templater)
}

// DELETE 注册 DELETE 路由
func (p *PackageContext) DELETE(router string, handleFunc HandleFunc, templater Templater) {
	p.register("DELETE", router, handleFunc, templater)
}

// routerKey 构建路由 key（URL 唯一，不包含 method）
func routerKey(router string) string {
	return strings.Trim(router, "/")
}

func initRouter(a *App) {

	// ⚠️ 重要：必须直接操作 a.routerInfo，不能调用 a.registerRouter() 或 PackageContext.register()
	//
	// 原因：死锁问题
	// 1. initRouter() 在 NewApp() 中被调用
	// 2. NewApp() 本身在 initApp() 的 sync.Once.Do() 中执行
	// 3. 此时全局变量 app 还没有被赋值（NewApp() 还没返回）
	// 4. 如果调用 PackageContext.register()，它会检查 app == nil，然后再次调用 initApp()
	// 5. sync.Once.Do() 会阻塞等待第一次执行完成，但第一次执行就是 NewApp()
	// 6. 而 NewApp() 又调用了 initRouter()，形成死锁
	//
	// 解决方案：直接操作传入的 App 实例的 routerInfo，避免触发全局 app 的检查
	//
	// ✅ 改造后：URL 唯一，/_callback 只注册一次，method 设为 "ANY" 表示支持所有 method
	key := routerKey("/_callback")
	if _, exists := a.routerInfo[key]; exists {
		panic(fmt.Errorf("路由 /_callback 已存在，不允许重复注册"))
	}

	a.routerInfo[key] = &routerInfo{
		HandleFunc: a.CallbackRouter,
		Router:     "/_callback",
		Method:     "ANY", // 支持所有 method（GET、POST、PUT、DELETE）
		Options:    nil,   // 系统路由没有 PackagePath
		Template:   &FormTemplate{},
	}

	key = routerKey(runtimePythonRouter)
	if _, exists := a.routerInfo[key]; exists {
		panic(fmt.Errorf("路由 %s 已存在，不允许重复注册", runtimePythonRouter))
	}

	a.routerInfo[key] = &routerInfo{
		HandleFunc: a.RuntimePython,
		Router:     runtimePythonRouter,
		Method:     "POST",
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

	router, err := a.getRoute(req.Router)
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
		if v.OnTableAddRow == nil {
			return errors.New("callback OnTableAddRow is not registered")
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
		if v.OnTableUpdateRow == nil {
			return errors.New("callback OnTableUpdateRow is not registered")
		}
		var onTableReq callback.OnTableUpdateRowReq
		// ⚠️ 关键：现在解析整个结构，包括 id、updates、old_values
		// 前端传递格式：{"id": 2, "updates": {"name": "802"}, "old_values": {"name": "801"}}
		err := json.Unmarshal(ctx.body, &onTableReq)
		if err != nil {
			return err
		}
		if onTableReq.ChangedFieldsBindMap == nil {
			onTableReq.ChangedFieldsBindMap = make(map[string]interface{})
		}
		for k, vv := range onTableReq.Updates {
			onTableReq.ChangedFieldsBindMap[k] = vv
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
		if v.OnTableDeleteRows == nil {
			return errors.New("callback OnTableDeleteRows is not registered")
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
