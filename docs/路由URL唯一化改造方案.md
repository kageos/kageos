# 路由 URL 唯一化改造方案

## 改造目标

将路由系统的 key 从 `router.method` 改为 `router`（URL 唯一），简化路由查询逻辑，提高代码可维护性。

## 当前设计分析

### 当前实现

```go
// 当前：key = router + "." + method
func routerKey(router string, method string) string {
    router = strings.Trim(router, "/")
    key := router + "." + method
    return key
}

// 查询时需要同时提供 router 和 method
func (a *App) getRouter(router string, method string) (*routerInfo, error) {
    trim := strings.Trim(router, "/")
    key := fmt.Sprintf("%s.%s", trim, method)
    info, ok := a.routerInfo[key]
    // ...
}
```

### 当前问题

1. **查询复杂**：需要同时提供 `router` 和 `method` 才能查询
2. **实际未使用**：所有函数都是 `GET` 方法，URL 已经唯一
3. **增加复杂度**：`GetFunctionTemplate` 需要尝试多个 method
4. **容易出错**：method 不匹配时可能找不到路由

## 改造方案

### 核心改造

#### 1. 修改 `routerKey` 函数（`sdk/agent-app/app/register.go`）

**改造前**：
```go
func routerKey(router string, method string) string {
    router = strings.Trim(router, "/")
    key := router + "." + method
    return key
}
```

**改造后**：
```go
func routerKey(router string) string {
    router = strings.Trim(router, "/")
    return router  // 直接返回 router，不再拼接 method
}
```

#### 2. 修改 `getRouter` 函数（`sdk/agent-app/app/app.go`）

**改造前**：
```go
func (a *App) getRouter(router string, method string) (*routerInfo, error) {
    trim := strings.Trim(router, "/")
    key := fmt.Sprintf("%s.%s", trim, method)
    info, ok := a.routerInfo[key]
    if ok {
        return info, nil
    }
    logger.Infof(a, "Router %s not found routerInfo:%+v ", key, a.routerInfo)
    return nil, fmt.Errorf("router %s not found", key)
}
```

**改造后**：
```go
func (a *App) getRouter(router string) (*routerInfo, error) {
    trim := strings.Trim(router, "/")
    info, ok := a.routerInfo[trim]
    if ok {
        return info, nil
    }
    logger.Infof(a, "Router %s not found routerInfo:%+v ", trim, a.routerInfo)
    return nil, fmt.Errorf("router %s not found", trim)
}
```

#### 3. 修改 `register` 函数（`sdk/agent-app/app/register.go`）

**改造前**：
```go
func register(router string, method string, handleFunc HandleFunc, templater Templater, options *RegisterOptions) {
    // ...
    app.routerInfo[routerKey(router, method)] = &routerInfo{
        HandleFunc: handleFunc,
        Router:     router,
        Method:     method,
        Options:    options,
        Template:   templater,
    }
}
```

**改造后**：
```go
func register(router string, method string, handleFunc HandleFunc, templater Templater, options *RegisterOptions) {
    // ...
    key := routerKey(router)
    
    // ✅ 检查 URL 唯一性：如果已存在，报错（不允许重复注册）
    if existing, exists := app.routerInfo[key]; exists {
        logger.Errorf(context.Background(), 
            "路由 %s 已存在，不允许重复注册。已存在的路由信息: Router=%s, Method=%s", 
            router, existing.Router, existing.Method)
        panic(fmt.Errorf("路由 %s 已存在，不允许重复注册", router))
    }
    
    app.routerInfo[key] = &routerInfo{
        HandleFunc: handleFunc,
        Router:     router,
        Method:     method,  // 仍然保存 method（用于 HTTP 处理）
        Options:    options,
        Template:   templater,
    }
}
```

#### 4. 修改 `initRouter` 函数（`sdk/agent-app/app/register.go`）

**问题**：`/_callback` 路由当前注册了 4 次（GET、POST、PUT、DELETE），改造后只能注册一次。

**方案选择**：**方案A - 只注册一次，在 `CallbackRouter` 中根据请求的 method 处理**

**改造前**：
```go
func initRouter(a *App) {
    a.routerInfo[routerKey("/_callback", MethodPost)] = &routerInfo{...}
    a.routerInfo[routerKey("/_callback", MethodGet)] = &routerInfo{...}
    a.routerInfo[routerKey("/_callback", MethodDelete)] = &routerInfo{...}
    a.routerInfo[routerKey("/_callback", MethodPut)] = &routerInfo{...}
}
```

**改造后**：
```go
func initRouter(a *App) {
    // ✅ 只注册一次，method 字段保存为 "ANY" 或保留第一个 method（如 "POST"）
    // 在 CallbackRouter 中根据请求的 method 处理
    key := routerKey("/_callback")
    if _, exists := a.routerInfo[key]; exists {
        panic(fmt.Errorf("路由 /_callback 已存在，不允许重复注册"))
    }
    
    a.routerInfo[key] = &routerInfo{
        HandleFunc: a.CallbackRouter,
        Router:     "/_callback",
        Method:     "ANY",  // 或保留 "POST"，表示支持所有 method
        Options:    nil,
        Template:   &FormTemplate{},
    }
}
```

**注意**：`CallbackRouter` 函数本身已经根据 `req.Method` 处理，所以不需要修改。

### 调用点改造

#### 1. `handle.go` - 消息处理

**文件**：`sdk/agent-app/app/handle.go`

**改造前**：
```go
router, err := a.getRouter(newContext.msg.Router, newContext.msg.Method)
```

**改造后**：
```go
router, err := a.getRouter(newContext.msg.Router)
```

#### 2. `register.go` - CallbackRouter

**文件**：`sdk/agent-app/app/register.go`

**改造前**：
```go
router, err := a.getRouter(req.Router, req.Method)
```

**改造后**：
```go
router, err := a.getRouter(req.Router)
```

#### 3. `on_app_update.go` - API 更新处理

**文件**：`sdk/agent-app/app/on_app_update.go`

**改造前**：
```go
router, err := a.getRouter(aa.Router, aa.Method)
```

**改造后**：
```go
router, err := a.getRouter(aa.Router)
```

### 未来功能支持

#### `GetFunctionTemplate` 函数（Link 组件需要）

**文件**：`sdk/agent-app/app/context.go`（需要新增）

**改造前**（文档中的设计）：
```go
func (ctx *Context) GetFunctionTemplate(functionPath string) (Templater, error) {
    // ...
    methods := []string{"GET", "POST", "PUT", "DELETE"}
    for _, method := range methods {
        router, err := app.getRouter(fullRouter, method)
        if err == nil && router != nil {
            return router.Template, nil
        }
    }
    return nil, fmt.Errorf("未找到函数 %s 的路由信息", functionPath)
}
```

**改造后**（简化版）：
```go
func (ctx *Context) GetFunctionTemplate(functionPath string) (Templater, error) {
    // 1. 构建完整的路由路径
    var fullRouter string
    if strings.HasPrefix(functionPath, "/") {
        fullRouter = strings.Trim(functionPath, "/")
    } else {
        routerGroup := ctx.GetRouterGroup()
        fullRouter = fmt.Sprintf("%s/%s", strings.Trim(routerGroup, "/"), strings.Trim(functionPath, "/"))
    }
    
    // 2. 直接查询，不再需要尝试多个 method
    router, err := app.getRouter(fullRouter)
    if err != nil {
        return nil, fmt.Errorf("未找到函数 %s 的路由信息: %w", functionPath, err)
    }
    
    return router.Template, nil
}
```

## 改造影响分析

### 1. 向后兼容性

**影响**：**不兼容**，需要修改所有调用 `getRouter` 的地方。

**处理方案**：
- 一次性修改所有调用点
- 确保所有调用点都移除 `method` 参数

### 2. `/_callback` 路由处理

**影响**：`/_callback` 路由当前注册了 4 次，改造后只能注册一次。

**处理方案**：
- 只注册一次，method 字段保存为 "ANY"
- `CallbackRouter` 函数已经根据 `req.Method` 处理，不需要修改

### 3. 注册时冲突检测

**影响**：如果同一个 URL 被注册多次，会 panic。

**处理方案**：
- 在 `register` 函数中检查 URL 唯一性
- 如果已存在，panic 并提示错误信息

### 4. 现有代码检查

**影响**：需要确保所有函数都使用唯一的 URL。

**处理方案**：
- 检查所有注册的路由，确保 URL 唯一
- 如果发现重复，需要修改 URL（如添加后缀）

## 改造步骤

### 阶段 1：核心函数改造

1. ✅ 修改 `routerKey` 函数：移除 `method` 参数
2. ✅ 修改 `getRouter` 函数：移除 `method` 参数
3. ✅ 修改 `register` 函数：添加 URL 唯一性检查
4. ✅ 修改 `initRouter` 函数：`/_callback` 只注册一次

### 阶段 2：调用点改造

1. ✅ 修改 `handle.go:108` - `a.getRouter(newContext.msg.Router)`
2. ✅ 修改 `register.go:230` - `a.getRouter(req.Router)`
3. ✅ 修改 `on_app_update.go:427` - `a.getRouter(aa.Router)`

### 阶段 3：未来功能支持

1. ✅ 实现 `GetFunctionTemplate` 函数（简化版，不再需要尝试多个 method）

### 阶段 4：测试验证

1. ✅ 测试路由注册：确保 URL 唯一性检查生效
2. ✅ 测试路由查询：确保所有调用点正常工作
3. ✅ 测试 `/_callback` 路由：确保所有 method 都能正常处理
4. ✅ 测试现有功能：确保所有业务功能正常

## 改造清单

### 需要修改的文件

1. **`sdk/agent-app/app/register.go`**
   - [ ] 修改 `routerKey` 函数签名和实现
   - [ ] 修改 `register` 函数，添加 URL 唯一性检查
   - [ ] 修改 `initRouter` 函数，`/_callback` 只注册一次
   - [ ] 修改 `CallbackRouter` 中的 `getRouter` 调用

2. **`sdk/agent-app/app/app.go`**
   - [ ] 修改 `getRouter` 函数签名和实现

3. **`sdk/agent-app/app/handle.go`**
   - [ ] 修改 `handle` 函数中的 `getRouter` 调用

4. **`sdk/agent-app/app/on_app_update.go`**
   - [ ] 修改 `onAppUpdate` 函数中的 `getRouter` 调用

5. **`sdk/agent-app/app/context.go`**（未来功能）
   - [ ] 实现 `GetFunctionTemplate` 函数（简化版）

### 不需要修改的部分

1. **`routerInfo` 结构体**：`Method` 字段保留（用于 HTTP 处理）
2. **`CallbackRouter` 函数逻辑**：已经根据 `req.Method` 处理，不需要修改
3. **其他使用 `routerInfo.Method` 的地方**：不需要修改

## 风险评估

### 高风险

1. **`/_callback` 路由处理**：需要确保所有 method 都能正常处理
   - **缓解措施**：`CallbackRouter` 已经根据 `req.Method` 处理，不需要修改

2. **URL 唯一性检查**：如果现有代码有重复 URL，会 panic
   - **缓解措施**：先检查所有注册的路由，确保 URL 唯一

### 中风险

1. **调用点遗漏**：如果遗漏某个调用点，会导致编译错误
   - **缓解措施**：使用 IDE 的全局搜索，确保所有调用点都修改

2. **测试覆盖**：需要确保所有功能都测试通过
   - **缓解措施**：逐步改造，每个阶段都进行测试

### 低风险

1. **性能影响**：改造后查询更简单，性能更好
2. **代码可维护性**：改造后代码更简洁，可维护性更好

## 测试计划

### 单元测试

1. ✅ 测试 `routerKey` 函数：确保返回正确的 key
2. ✅ 测试 `getRouter` 函数：确保能正确查询路由
3. ✅ 测试 `register` 函数：确保 URL 唯一性检查生效
4. ✅ 测试 `initRouter` 函数：确保 `/_callback` 只注册一次

### 集成测试

1. ✅ 测试路由注册：注册多个路由，确保 URL 唯一
2. ✅ 测试路由查询：查询已注册的路由，确保能正确返回
3. ✅ 测试 `/_callback` 路由：使用不同 method 调用，确保都能正常处理
4. ✅ 测试现有功能：确保所有业务功能正常

### 回归测试

1. ✅ 测试所有 Table 函数：确保能正常查询和处理
2. ✅ 测试所有 Form 函数：确保能正常查询和处理
3. ✅ 测试回调功能：确保 `OnSelectFuzzy`、`OnTableAddRow` 等回调正常
4. ✅ 测试 API 更新功能：确保 `onAppUpdate` 正常

## 总结

### 改造优势

1. ✅ **简化查询**：不再需要 method 参数，查询更简单
2. ✅ **提高性能**：直接通过 URL 查询，不需要尝试多个 method
3. ✅ **代码清晰**：一个 URL 对应一个函数，逻辑更清晰
4. ✅ **易于维护**：减少复杂度，提高可维护性

### 改造风险

1. ⚠️ **不兼容**：需要修改所有调用点
2. ⚠️ **`/_callback` 路由**：需要确保所有 method 都能正常处理
3. ⚠️ **URL 唯一性**：需要确保所有路由 URL 唯一

### 改造建议

1. ✅ **分阶段改造**：先改造核心函数，再改造调用点
2. ✅ **充分测试**：每个阶段都进行充分测试
3. ✅ **文档更新**：更新相关文档，说明改造内容

