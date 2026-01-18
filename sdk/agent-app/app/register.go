package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
)

type PackageContext struct {
	RouterGroup string `json:"router_group"`
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
		// ⚠️ 关键：现在解析整个结构，包括 id、updates、old_values
		// 前端传递格式：{"id": 2, "updates": {"name": "802"}, "old_values": {"name": "801"}}
		err := json.Unmarshal(ctx.body, &onTableReq)
		if err != nil {
			return err
		}
		if onTableReq.BindUpdatesMap == nil {
			onTableReq.BindUpdatesMap = make(map[string]interface{})
		}
		for k, vv := range onTableReq.Updates {
			onTableReq.BindUpdatesMap[k] = vv
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
	case CallbackTypeOnTableCreateInBatches:
		// 系统内置批量创建回调，直接批量插入数据库，不触发用户侧的回调
		v, ok := router.Template.(*TableTemplate)
		if !ok {
			return errors.New("invalid type of TableTemplate")
		}

		var batchReq callback.OnTableCreateInBatchesReq
		err := json.Unmarshal(ctx.body, &batchReq)
		if err != nil {
			return fmt.Errorf("解析批量创建请求失败: %w", err)
		}

		// 调用系统内置的批量创建逻辑
		batchResp, err := handleTableCreateInBatches(ctx, v, &batchReq)
		if err != nil {
			logger.Errorf(ctx, "callback OnTableCreateInBatches router:%s error:%s", req.Type, err.Error())
			return err
		}

		err = resp.Form(batchResp).Build()
		if err != nil {
			logger.Errorf(ctx, "callback OnTableCreateInBatches router:%s Build error:%s", req.Type, err.Error())
			return err
		}
		logger.Infof(ctx, "CallbackRouter OnTableCreateInBatches success: success=%d, fail=%d", batchResp.SuccessCount, batchResp.FailCount)
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

// handleTableCreateInBatches 系统内置的批量创建处理函数
// 通过反射获取 AutoCrudTable 结构类型，批量插入数据库
func handleTableCreateInBatches(ctx *Context, template *TableTemplate, req *callback.OnTableCreateInBatchesReq) (*callback.OnTableCreateInBatchesResp, error) {
	if template.AutoCrudTable == nil {
		return nil, errors.New("AutoCrudTable 不能为空")
	}

	// 获取数据库连接
	db := ctx.GetGormDB()
	if db == nil {
		return nil, errors.New("获取数据库连接失败")
	}

	// 获取 AutoCrudTable 的结构类型
	tableType := reflect.TypeOf(template.AutoCrudTable)
	if tableType.Kind() == reflect.Ptr {
		tableType = tableType.Elem()
	}
	if tableType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("AutoCrudTable 必须是结构体类型，当前类型: %v", tableType.Kind())
	}

	// 创建切片类型
	sliceType := reflect.SliceOf(reflect.PtrTo(tableType))

	// 创建切片实例
	sliceValue := reflect.New(sliceType).Elem()

	// 将 JSON 数据反序列化到切片
	jsonData, err := json.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("序列化数据失败: %w", err)
	}

	// 创建切片指针并反序列化
	slicePtr := reflect.New(sliceType).Interface()
	if err := json.Unmarshal(jsonData, slicePtr); err != nil {
		return nil, fmt.Errorf("反序列化数据失败: %w", err)
	}

	// 获取切片值
	sliceValue = reflect.ValueOf(slicePtr).Elem()

	// 批量插入数据库
	successCount := 0
	failCount := 0
	var errors []callback.OnTableCreateBatchError

	// 使用 CreateInBatches 批量插入（每批 100 条）
	batchSize := 100
	totalCount := sliceValue.Len()

	for i := 0; i < totalCount; i += batchSize {
		end := i + batchSize
		if end > totalCount {
			end = totalCount
		}

		// 获取当前批次
		batchSlice := sliceValue.Slice(i, end)

		// 转换为 []interface{}
		batchInterface := make([]interface{}, batchSlice.Len())
		for j := 0; j < batchSlice.Len(); j++ {
			batchInterface[j] = batchSlice.Index(j).Interface()
		}

		// 批量插入
		if err := db.CreateInBatches(batchInterface, batchSize).Error; err != nil {
			// 如果批量插入失败，尝试逐条插入以获取详细的错误信息
			for j := 0; j < batchSlice.Len(); j++ {
				item := batchSlice.Index(j).Interface()
				if err := db.Create(item).Error; err != nil {
					failCount++
					errors = append(errors, callback.OnTableCreateBatchError{
						Index: i + j,
						Error: err.Error(),
					})
				} else {
					successCount++
				}
			}
		} else {
			successCount += batchSlice.Len()
		}
	}

	logger.Infof(ctx, "[handleTableCreateInBatches] 批量创建完成: 总数=%d, 成功=%d, 失败=%d", totalCount, successCount, failCount)

	return &callback.OnTableCreateInBatchesResp{
		SuccessCount: successCount,
		FailCount:    failCount,
		Errors:       errors,
	}, nil
}
