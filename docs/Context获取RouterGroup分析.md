# Context 获取 RouterGroup 分析

## 问题

`ctx` 是否能获取到当前的 RouterGroup？

## 答案

**✅ 可以！** 但需要注意空值检查。

## 数据结构链

```
Context
  └── routerInfo *routerInfo
       └── Options *RegisterOptions
            └── RouterGroup *RouterGroup
                 └── RouterGroup string  ← 这就是我们要获取的值
```

## 设置时机

### 1. Context 创建时

**位置**：`sdk/agent-app/app/handle.go`

```go
// 第 99 行：创建 Context
newContext, err := a.NewContext(ctx, req)

// 第 108-115 行：查找路由并设置 routerInfo
router, err := a.getRoute(newContext.msg.Router)
if err != nil {
    // 错误处理
}
// 将 routerInfo 保存到 Context 中，方便后续获取 PackagePath
newContext.routerInfo = router
handleFunc := router.HandleFunc
```

**关键点**：
- `NewContext` 创建时，`routerInfo` 是 `nil`
- 在调用 `handleFunc` **之前**，会设置 `newContext.routerInfo = router`
- 所以在 `handleFunc` 执行时，`ctx.routerInfo` **一定是有值的**

### 2. routerInfo 的来源

**位置**：`sdk/agent-app/app/app.go`

```go
// addRoute 方法中
a.routerInfo[key] = &routerInfo{
    HandleFunc: handleFunc,
    Router:     router,
    Method:     method,
    Options:    options,  // ← 这里设置了 Options
    Template:   templater,
}
```

**关键点**：
- 通过 `RouterGroup` 注册的路由，`Options` 是有值的
- 系统路由（如 `/_callback`），`Options` 可能是 `nil`

## 获取 RouterGroup 的方法

### 方法 1：直接访问（需要空值检查）

```go
func (ctx *Context) GetRouterGroup() string {
    if ctx.routerInfo != nil && 
       ctx.routerInfo.Options != nil && 
       ctx.routerInfo.Options.RouterGroup != nil {
        return ctx.routerInfo.Options.RouterGroup.RouterGroup
    }
    return ""
}
```

### 方法 2：使用现有的检查模式

参考 `sdk/agent-app/app/db.go` 中的用法：

```go
// db.go 第 42 行
if c.routerInfo != nil && c.routerInfo.Options != nil {
    dbName = c.routerInfo.Options.GetDBName(c.msg.User, c.msg.App)
}
```

## 实际使用场景

### ✅ 正常情况（通过 RouterGroup 注册的路由）

```go
// 在 CrmMeetingRoomBookingList 函数中
func CrmMeetingRoomBookingList(ctx *app.Context, resp response.Response) error {
    // ctx.routerInfo 有值
    // ctx.routerInfo.Options 有值
    // ctx.routerInfo.Options.RouterGroup 有值
    // ctx.routerInfo.Options.RouterGroup.RouterGroup = "/crm"
    
    routerGroup := ctx.GetRouterGroup() // 返回 "/crm"
    // ...
}
```

### ⚠️ 特殊情况（系统路由）

```go
// 在 CallbackRouter 中
func (a *App) CallbackRouter(ctx *Context, resp response.Response) error {
    // ctx.routerInfo 有值（在 handle.go 中设置）
    // ctx.routerInfo.Options 可能是 nil（系统路由）
    // ctx.routerInfo.Options.RouterGroup 可能是 nil
    
    routerGroup := ctx.GetRouterGroup() // 返回 ""
    // 需要使用绝对路径
}
```

## 实现建议

### GetRouterGroup 方法实现

**位置**：`sdk/agent-app/app/context.go`

```go
// GetRouterGroup 获取当前请求的 RouterGroup
// 返回当前请求所属的 RouterGroup 路径（如 "/crm"）
// 如果无法获取（系统路由或未设置），返回空字符串
func (ctx *Context) GetRouterGroup() string {
    if ctx.routerInfo != nil && 
       ctx.routerInfo.Options != nil && 
       ctx.routerInfo.Options.RouterGroup != nil {
        return ctx.routerInfo.Options.RouterGroup.RouterGroup
    }
    return ""
}
```

## 注意事项

### 1. 空值检查

必须检查三层：
- `ctx.routerInfo != nil`
- `ctx.routerInfo.Options != nil`
- `ctx.routerInfo.Options.RouterGroup != nil`

### 2. 系统路由

系统路由（如 `/_callback`）的 `Options` 可能是 `nil`，所以 `GetRouterGroup()` 会返回空字符串。

### 3. 使用建议

- **同 package 内跳转**：使用相对路径，依赖 `GetRouterGroup()`
- **跨 package 跳转**：使用绝对路径，不依赖 `GetRouterGroup()`
- **系统路由**：使用绝对路径，因为 `GetRouterGroup()` 返回空字符串

## 测试场景

### 场景 1：正常路由（通过 RouterGroup 注册）

```go
// 注册
crmMeetingRoomRouterGroup.GET("meeting_room_list", ...)
// Options.RouterGroup.RouterGroup = "/crm"

// 调用
func CrmMeetingRoomBookingList(ctx *app.Context, resp response.Response) error {
    routerGroup := ctx.GetRouterGroup() // 返回 "/crm" ✅
    // ...
}
```

### 场景 2：系统路由

```go
// 注册（在 initRouter 中）
a.routerInfo[key] = &routerInfo{
    HandleFunc: a.CallbackRouter,
    Router:     "/_callback",
    Method:     "ANY",
    Options:    nil,  // ← 系统路由没有 Options
    Template:   &FormTemplate{},
}

// 调用
func (a *App) CallbackRouter(ctx *Context, resp response.Response) error {
    routerGroup := ctx.GetRouterGroup() // 返回 "" ⚠️
    // 需要使用绝对路径
}
```

## 总结

1. **✅ 可以获取**：在 `handleFunc` 执行时，`ctx.routerInfo` 一定有值
2. **⚠️ 需要空值检查**：必须检查三层（routerInfo、Options、RouterGroup）
3. **✅ 正常路由**：通过 RouterGroup 注册的路由，`GetRouterGroup()` 会返回正确的值
4. **⚠️ 系统路由**：系统路由的 `Options` 可能是 `nil`，`GetRouterGroup()` 返回空字符串

