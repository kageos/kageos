# Link 组件实现分析（路由优化后）

## 路由系统优化带来的简化

### 优化前（文档中的方案）
```go
// 需要遍历多个 HTTP 方法
methods := []string{"GET", "POST", "PUT", "DELETE"}
for _, method := range methods {
    router, err := app.getRouter(fullRouter, method)
    if err == nil && router != nil {
        return router.Template, nil
    }
}
```

### 优化后（当前实现）
```go
// URL 唯一，直接获取路由，不需要 method
router, err := app.getRoute(fullRouter)
if err != nil {
    return nil, err
}
return router.Template, nil
```

**优势**：
- ✅ **更简单**：不需要遍历多个 method
- ✅ **更高效**：直接查找，O(1) 时间复杂度
- ✅ **更清晰**：代码逻辑更直观

---

## 实现步骤

### 阶段 1：后端核心方法（Context 扩展）

#### 1.1 `GetFunctionTemplate` - 获取函数模板（简化版）

**位置**：`sdk/agent-app/app/context.go`

**实现要点**：
- 利用路由系统优化，直接调用 `app.getRoute(router)`
- 支持绝对路径和相对路径
- 相对路径需要加上当前 RouterGroup

```go
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

#### 1.2 `GetRouterGroup` - 获取当前 RouterGroup

**位置**：`sdk/agent-app/app/context.go`

```go
// GetRouterGroup 获取当前请求的 RouterGroup
func (ctx *Context) GetRouterGroup() string {
    if ctx.routerInfo != nil && ctx.routerInfo.Options != nil && ctx.routerInfo.Options.RouterGroup != nil {
        return ctx.routerInfo.Options.RouterGroup.RouterGroup
    }
    return ""
}
```

#### 1.3 `BuildFunctionUrl` - 构建跳转 URL（主函数）

**位置**：`sdk/agent-app/app/url_builder.go`（新建文件）

**功能**：
- 支持外链跳转（自动识别）
- 支持函数跳转（Table 和 Form）
- 自动判断函数类型
- 自动转换参数格式

```go
// BuildFunctionUrl 构建跳转 URL（支持函数跳转和外链）
func (ctx *Context) BuildFunctionUrl(
    target string, // 函数路径或外链
    params interface{}, // 结构体参数或 nil
) (string, error) {
    // 1. 判断是否是外链
    if isExternalLink(target) {
        return normalizeExternalLink(target), nil
    }
    
    // 2. 函数跳转模式：获取目标函数的模板信息
    template, err := ctx.GetFunctionTemplate(target)
    if err != nil {
        return "", fmt.Errorf("获取函数模板失败: %w", err)
    }
    
    // 3. 根据模板类型判断是 Table 还是 Form
    var queryString string
    switch template.TemplateType() {
    case TemplateTypeTable:
        if params != nil {
            queryString, err = query.StructToTableParams(params)
        }
    case TemplateTypeForm:
        if params != nil {
            queryString, err = StructToFormParams(params)
        }
    default:
        return "", fmt.Errorf("不支持的模板类型")
    }
    
    if err != nil {
        return "", err
    }
    
    // 4. 构建完整 URL
    if strings.HasPrefix(target, "/") {
        // 绝对路径
        if queryString != "" {
            return fmt.Sprintf("%s?%s", target, queryString), nil
        }
        return target, nil
    } else {
        // 相对路径，需要获取当前 RouterGroup
        routerGroup := ctx.GetRouterGroup()
        fullPath := fmt.Sprintf("%s/%s", routerGroup, target)
        if queryString != "" {
            return fmt.Sprintf("%s?%s", fullPath, queryString), nil
        }
        return fullPath, nil
    }
}
```

#### 1.4 外链识别和规范化

**位置**：`sdk/agent-app/app/url_builder.go`

```go
// isExternalLink 判断是否是外链
func isExternalLink(target string) bool {
    // 如果已经是完整的 URL（包含协议），直接返回
    if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
        return true
    }
    
    // 如果包含域名特征（如 www.、.com、.cn 等），判断为外链
    if strings.Contains(target, "www.") {
        return true
    }
    
    // 检查是否包含常见的顶级域名
    tlds := []string{".com", ".cn", ".org", ".net", ".io", ".dev", ".top", ".xyz"}
    for _, tld := range tlds {
        if strings.Contains(target, tld) {
            return true
        }
    }
    
    return false
}

// normalizeExternalLink 规范化外链 URL
func normalizeExternalLink(link string) string {
    // 如果已经包含协议，直接返回
    if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
        return link
    }
    
    // 如果没有协议，默认添加 https://
    return "https://" + link
}
```

---

### 阶段 2：参数转换工具

#### 2.1 `StructToTableParams` - Table 参数转换

**位置**：`pkg/gormx/query/url_builder.go`（新建文件）

**功能**：
- 根据 `search` 标签自动转换
- 支持 `eq`、`in`、`like`、`gte`、`lte` 等
- 支持 `[]string` 类型（`in` 条件）
- 返回 URL 查询字符串

**关键点**：
- 使用反射遍历结构体字段
- 读取 `search` 标签
- 读取 `json` 标签作为字段名
- 转换为 `SearchFilterPageReq` 格式
- 最终转换为 URL 查询字符串

#### 2.2 `StructToFormParams` - Form 参数转换

**位置**：`sdk/agent-app/app/url_builder.go`

**功能**：
- 转换为 `k=v` 格式
- 支持数组类型（逗号分隔）
- 返回 URL 查询字符串

**关键点**：
- 使用反射遍历结构体字段
- 读取 `json` 标签作为字段名
- 跳过零值字段
- 处理数组类型

---

### 阶段 3：前端实现

#### 3.1 `LinkWidget` 组件

**位置**：`web/src/core/widgets-v2/components/LinkWidget.vue`

**功能**：
- 接收完整 URL（后端已生成）
- 支持编辑/响应/详情/表格模式
- 支持新窗口打开
- 支持图标和样式自定义

**关键点**：
- 后端已经生成完整 URL，前端直接使用
- 相对路径需要转换为完整路径（`/workspace/{user}/{app}/{function}?query`）
- 外链直接使用
- 内部链接使用 Vue Router 跳转

#### 3.2 注册组件

**位置**：`web/src/core/factories-v2/index.ts`

```typescript
import LinkWidget from '../widgets-v2/components/LinkWidget.vue'

// 注册 Link 组件
widgetComponentFactory.registerRequestComponent(WidgetType.LINK, LinkWidget)
widgetComponentFactory.registerResponseComponent(WidgetType.LINK, LinkWidget)
```

---

## 实现顺序建议

### 第一步：后端核心方法
1. ✅ `GetRouterGroup` - 最简单，先实现
2. ✅ `GetFunctionTemplate` - 利用路由优化，简化实现
3. ✅ 外链识别和规范化 - 独立功能，容易测试
4. ✅ `StructToFormParams` - Form 参数转换（相对简单）
5. ✅ `StructToTableParams` - Table 参数转换（需要理解 search 标签）
6. ✅ `BuildFunctionUrl` - 主函数，整合所有功能

### 第二步：前端实现
1. ✅ `LinkWidget` 组件
2. ✅ 注册组件
3. ✅ 测试各种场景

---

## 关键优势总结

### 路由系统优化带来的优势

1. **`GetFunctionTemplate` 简化**：
   - 优化前：需要遍历 4 个 method（GET、POST、PUT、DELETE）
   - 优化后：直接调用 `app.getRoute(router)`，O(1) 查找

2. **代码更清晰**：
   - 不需要处理 method 相关的逻辑
   - 代码更简洁，更容易理解

3. **性能更好**：
   - 不需要遍历多个 method
   - 直接查找，性能更好

---

## 使用示例

### 示例 1：会议室预约跳转到会议室详情（Table 函数）

```go
// 在 CrmMeetingRoomBookingList 函数中
for i := range bookings {
    // ✅ 直接使用目标函数的结构体（CrmMeetingRoom）
    params := CrmMeetingRoom{
        ID: bookings[i].RoomID,
    }
    
    bookings[i].RoomLink, _ = ctx.BuildFunctionUrl(
        "meeting_room_list",
        params,
    )
    // 生成的 URL: luobei/demo/crm/meeting_room_list?eq=id:1
}
```

### 示例 2：外链跳转

```go
// ✅ 外链模式：直接传递字符串
bookings[i].ExternalLink, _ = ctx.BuildFunctionUrl(
    "www.baidu.com", // 外链，自动识别
    nil,             // 外链不需要参数
)
// 生成的 URL: https://www.baidu.com
```

### 示例 3：使用 in 条件（多个状态值）

```go
// ✅ in 条件自然支持
params := CrmTicket{
    Status: []string{"待处理", "处理中"}, // 直接使用数组
}
link, _ := ctx.BuildFunctionUrl("crm_ticket_list", params)
// 生成的 URL: luobei/demo/crm/crm_ticket_list?in=status:待处理,处理中
```

---

## 注意事项

1. **相对路径 vs 绝对路径**：
   - 相对路径：需要加上当前 RouterGroup
   - 绝对路径：直接使用

2. **外链识别**：
   - 包含 `http://` 或 `https://`：直接识别为外链
   - 包含 `www.`：识别为外链
   - 包含常见顶级域名：识别为外链

3. **参数转换**：
   - Table 函数：根据 `search` 标签自动转换
   - Form 函数：转换为 `k=v` 格式
   - 数组类型：自动处理（`[]string` 用于 `in` 条件）

4. **错误处理**：
   - 函数不存在：返回错误
   - 参数转换失败：返回错误
   - 外链识别失败：可能被误判为函数路径

---

## 后续优化

1. ✅ 支持外链跳转（已规划）
2. 支持变量替换（如 `$id`、`$create_by`）
3. 支持嵌套结构体（如范围查询）
4. 支持条件表达式（如根据状态动态选择参数）
5. 外链安全性验证（白名单、黑名单等）

