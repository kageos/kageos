# BuildFunctionUrl 路由查找分析

## 问题

`BuildFunctionUrl` 如何找到路由？因为只有一个 `meeting_room_list`，如何确定完整路径？

## 路由注册流程

### 1. packageContext 定义

**位置**：`namespace/luobei/demo/code/api/crm/init_.go`

```go
var packageContext = &app.PackageContext{
    RouterGroup: "/crm",
}
```

### 2. RouterGroup 创建

**位置**：`namespace/luobei/demo/code/api/crm/crm_meeting_room.go`

```go
var crmMeetingRoomRouterGroup = app.NewRouterGroup(packageContext, &app.RouterGroupInfo{
    GroupCode: "crm_meeting_room",
    GroupName: "会议室管理",
})
```

### 3. 路由注册

```go
crmMeetingRoomRouterGroup.GET("meeting_room_list", CrmMeetingRoomList, CrmMeetingRoomListTemplate)
```

**注册流程**：
1. `p.RouterGroup = "/crm"`（来自 `packageContext.RouterGroup`）
2. `packagePath := strings.Trim(p.RouterGroup, "/")` → `"crm"`
3. `fullRouter := fmt.Sprintf("/%s/%s", packagePath, strings.Trim(router, "/"))` → `"/crm/meeting_room_list"`
4. `routerKey(fullRouter)` → `"crm/meeting_room_list"`（去掉首尾斜杠）
5. **注册的 key**：`"crm/meeting_room_list"`

## 路由查找流程

### 场景 1：相对路径（同 package）

**调用**：
```go
// 在 CrmMeetingRoomBookingList 函数中
ctx.BuildFunctionUrl("meeting_room_list", params)
```

**查找流程**：
1. `routerGroup := ctx.GetRouterGroup()` → 从当前请求的 `routerInfo.Options.RouterGroup.RouterGroup` 获取
   - 当前请求是 `CrmMeetingRoomBookingList`，它的 RouterGroup 也是 `"/crm"`（因为都在同一个 package 下）
2. `fullRouter = fmt.Sprintf("%s/%s", strings.Trim(routerGroup, "/"), "meeting_room_list")` → `"crm/meeting_room_list"`
3. `routerKey(fullRouter)` → `"crm/meeting_room_list"`
4. **查找的 key**：`"crm/meeting_room_list"` ✅ **匹配成功！**

### 场景 2：相对路径（跨 package）

**调用**：
```go
// 在 crm/meeting_room.go 中
ctx.BuildFunctionUrl("tools_cashier", params) // 目标在 tools package
```

**查找流程**：
1. `routerGroup := ctx.GetRouterGroup()` → `"/crm"`（当前请求的 RouterGroup）
2. `fullRouter = fmt.Sprintf("%s/%s", "crm", "tools_cashier")` → `"crm/tools_cashier"`
3. **查找的 key**：`"crm/tools_cashier"` ❌ **找不到！**（实际注册的 key 是 `"tools/tools_cashier"`）

**问题**：相对路径只能用于同 package 内的跳转！

### 场景 3：绝对路径

**调用**：
```go
// 在 crm/meeting_room.go 中
ctx.BuildFunctionUrl("/tools/tools_cashier", params) // 绝对路径
```

**查找流程**：
1. `strings.HasPrefix(functionPath, "/")` → `true`（绝对路径）
2. `fullRouter = strings.Trim(functionPath, "/")` → `"tools/tools_cashier"`
3. `routerKey(fullRouter)` → `"tools/tools_cashier"`
4. **查找的 key**：`"tools/tools_cashier"` ✅ **匹配成功！**

## 关键发现

### ✅ 相对路径的优势
- **同 package 内跳转**：非常方便，只需要函数名
- **代码简洁**：不需要写完整路径

### ⚠️ 相对路径的限制
- **只能用于同 package**：跨 package 需要使用绝对路径
- **依赖当前 RouterGroup**：如果当前请求的 RouterGroup 不正确，会查找失败

### ✅ 绝对路径的优势
- **跨 package 跳转**：可以跳转到任何 package
- **不依赖当前 RouterGroup**：路径明确，不会出错

## 实现建议

### GetFunctionTemplate 实现

```go
// GetFunctionTemplate 根据函数路径获取函数模板
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
    router, err := app.getRoute(fullRouter)
    if err != nil {
        return nil, fmt.Errorf("未找到函数 %s 的路由信息: %w", functionPath, err)
    }
    
    return router.Template, nil
}
```

### GetRouterGroup 实现

```go
// GetRouterGroup 获取当前请求的 RouterGroup
func (ctx *Context) GetRouterGroup() string {
    if ctx.routerInfo != nil && ctx.routerInfo.Options != nil && ctx.routerInfo.Options.RouterGroup != nil {
        return ctx.routerInfo.Options.RouterGroup.RouterGroup
    }
    return ""
}
```

## 使用建议

### 1. 同 package 内跳转（推荐使用相对路径）

```go
// ✅ 推荐：相对路径，简洁明了
bookings[i].RoomLink, _ = ctx.BuildFunctionUrl(
    "meeting_room_list", // 同 package 内，使用相对路径
    params,
)
```

### 2. 跨 package 跳转（必须使用绝对路径）

```go
// ✅ 必须：绝对路径，跨 package 跳转
link, _ := ctx.BuildFunctionUrl(
    "/tools/tools_cashier", // 跨 package，使用绝对路径
    params,
)
```

### 3. 外链跳转

```go
// ✅ 外链：自动识别
link, _ := ctx.BuildFunctionUrl(
    "www.baidu.com", // 外链，自动识别
    nil,
)
```

## 总结

1. **相对路径**：只能用于同 package 内跳转，依赖当前 RouterGroup
2. **绝对路径**：可以跨 package 跳转，路径明确，不依赖当前 RouterGroup
3. **路由查找**：利用路由系统优化，直接调用 `app.getRoute(router)`，O(1) 查找
4. **注册的 key**：`"crm/meeting_room_list"`（去掉首尾斜杠）
5. **查找的 key**：必须与注册的 key 完全匹配

## 示例：完整的路由查找流程

### 注册时
```
RouterGroup: "/crm"
router: "meeting_room_list"
→ fullRouter: "/crm/meeting_room_list"
→ routerKey: "crm/meeting_room_list"
→ 注册 key: "crm/meeting_room_list"
```

### 查找时（相对路径）
```
当前 RouterGroup: "/crm"
functionPath: "meeting_room_list"
→ fullRouter: "crm/meeting_room_list"
→ routerKey: "crm/meeting_room_list"
→ 查找 key: "crm/meeting_room_list" ✅ 匹配成功
```

### 查找时（绝对路径）
```
functionPath: "/crm/meeting_room_list"
→ fullRouter: "crm/meeting_room_list"
→ routerKey: "crm/meeting_room_list"
→ 查找 key: "crm/meeting_room_list" ✅ 匹配成功
```

