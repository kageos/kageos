# Link/Button 组件实现方案

## 需求分析

### 核心需求
1. **动态链接跳转**：根据当前行数据动态构造 URL 参数
2. **条件跳转**：根据数据状态（如是否投票过）跳转到不同页面
3. **参数格式支持**：
   - **Table 函数**：使用 search 参数格式（`like=field:value&sort=id:desc`）
   - **Form 函数**：使用 k=v 格式（`topic_id=123`）
4. **外链支持**：支持跳转到外部链接（如 `www.baidu.com`）

### 关键问题
- **`in` 条件处理**：如何传递多个值（如 `status:待处理,处理中`）？
- **参数格式转换**：如何自动根据 search 标签转换为正确的格式？
- **类型安全**：如何避免手动构造 URL 字符串的错误？
- **自动判断**：如何自动判断目标函数是 Table 还是 Form？

---

## 实现方案

### 设计思路
- **直接使用目标函数的结构体**：不需要定义新的参数结构体
- **Table 函数**：使用 `AutoCrudTable` 的 Model（如 `CrmMeetingRoom`）
- **Form 函数**：使用 `Request` 结构体（如 `VoteSystemResultReq`）
- **自动判断函数类型**：后端自动判断是 Table 还是 Form，无需手动指定
- **自动转换**：后端根据结构体的 search 标签自动转换参数格式
- **支持外链**：传递字符串（如 `www.baidu.com`）时，自动识别为外链

### 核心优势
1. ✅ **极简**：不需要定义新的参数结构体，直接使用目标函数的结构体
2. ✅ **类型安全**：结构体字段有类型检查，编译时就能发现错误
3. ✅ **自动转换**：后端根据 search 标签自动转换为正确的格式
4. ✅ **易于维护**：修改字段名时只需要改目标函数的结构体
5. ✅ **支持复杂条件**：`in` 条件使用 `[]string` 类型，自然支持多值
6. ✅ **零学习成本**：开发者已经熟悉目标函数的结构体定义
7. ✅ **支持外链**：自动识别外链，无需特殊处理

---

## 后端实现

### 1. Table 参数转换（放在 `pkg/gormx/query` 下）

```go
// pkg/gormx/query/url_builder.go
package query

import (
    "fmt"
    "net/url"
    "reflect"
    "strings"
)

// StructToTableParams 将结构体转换为 Table 函数的参数格式
// 根据 search 标签自动转换
// 返回 URL 查询字符串，格式：eq=field:value&in=field:value1,value2&sorts=id:desc
func StructToTableParams(params interface{}) (string, error) {
    if params == nil {
        return "", nil
    }
    
    v := reflect.ValueOf(params)
    if v.Kind() == reflect.Ptr {
        v = v.Elem()
    }
    if v.Kind() != reflect.Struct {
        return "", fmt.Errorf("参数必须是结构体类型")
    }
    
    t := v.Type()
    var result SearchFilterPageReq
    
    for i := 0; i < v.NumField(); i++ {
        field := t.Field(i)
        fieldValue := v.Field(i)
        
        // 获取 search 标签
        searchTag := field.Tag.Get("search")
        if searchTag == "" {
            // 如果没有 search 标签，检查是否是排序字段
            if field.Name == "Sorts" {
                result.Sorts = fieldValue.String()
            }
            continue
        }
        
        // 根据 search 标签类型转换
        searchTypes := strings.Split(searchTag, ",")
        fieldName := getJSONTag(field) // 获取 json 标签作为字段名
        
        for _, searchType := range searchTypes {
            searchType = strings.TrimSpace(searchType)
            
            switch searchType {
            case "eq":
                if !fieldValue.IsZero() {
                    result.Eq = append(result.Eq, fmt.Sprintf("%s:%v", fieldName, fieldValue.Interface()))
                }
            case "in":
                // 处理 []string 类型（推荐方式）
                if fieldValue.Kind() == reflect.Slice {
                    values := make([]string, 0)
                    for j := 0; j < fieldValue.Len(); j++ {
                        val := fieldValue.Index(j).Interface()
                        values = append(values, fmt.Sprintf("%v", val))
                    }
                    if len(values) > 0 {
                        // 格式：field:value1,value2
                        result.In = append(result.In, fmt.Sprintf("%s:%s", fieldName, strings.Join(values, ",")))
                    }
                } else if !fieldValue.IsZero() {
                    // 单个值也支持（兼容性）
                    result.In = append(result.In, fmt.Sprintf("%s:%v", fieldName, fieldValue.Interface()))
                }
            case "like":
                if !fieldValue.IsZero() {
                    result.Like = append(result.Like, fmt.Sprintf("%s:%v", fieldName, fieldValue.Interface()))
                }
            case "gte":
                if !fieldValue.IsZero() {
                    result.Gte = append(result.Gte, fmt.Sprintf("%s:%v", fieldName, fieldValue.Interface()))
                }
            case "lte":
                if !fieldValue.IsZero() {
                    result.Lte = append(result.Lte, fmt.Sprintf("%s:%v", fieldName, fieldValue.Interface()))
                }
            }
        }
    }
    
    // 转换为 URL 查询字符串
    return buildSearchParamsURL(&result), nil
}

// buildSearchParamsURL 将 SearchFilterPageReq 转换为 URL 查询字符串
func buildSearchParamsURL(req *SearchFilterPageReq) string {
    values := url.Values{}
    
    if len(req.Eq) > 0 {
        values.Set("eq", strings.Join(req.Eq, ","))
    }
    if len(req.In) > 0 {
        values.Set("in", strings.Join(req.In, ","))
    }
    if len(req.Like) > 0 {
        values.Set("like", strings.Join(req.Like, ","))
    }
    if len(req.Gte) > 0 {
        values.Set("gte", strings.Join(req.Gte, ","))
    }
    if len(req.Lte) > 0 {
        values.Set("lte", strings.Join(req.Lte, ","))
    }
    if req.Sorts != "" {
        values.Set("sorts", req.Sorts)
    }
    
    return values.Encode()
}

// getJSONTag 获取字段的 json 标签
func getJSONTag(field reflect.StructField) string {
    jsonTag := field.Tag.Get("json")
    if jsonTag == "" {
        return strings.ToLower(field.Name)
    }
    // 处理 json:"field_name,omitempty" 格式
    parts := strings.Split(jsonTag, ",")
    return parts[0]
}
```

### 2. Form 参数转换（放在 `sdk/agent-app/app` 下）

```go
// sdk/agent-app/app/url_builder.go
package app

import (
    "fmt"
    "net/url"
    "reflect"
    "strings"
)

// StructToFormParams 将结构体转换为 Form 函数的参数格式（k=v）
func StructToFormParams(params interface{}) (string, error) {
    if params == nil {
        return "", nil
    }
    
    v := reflect.ValueOf(params)
    if v.Kind() == reflect.Ptr {
        v = v.Elem()
    }
    if v.Kind() != reflect.Struct {
        return "", fmt.Errorf("参数必须是结构体类型")
    }
    
    t := v.Type()
    values := url.Values{}
    
    for i := 0; i < v.NumField(); i++ {
        field := t.Field(i)
        fieldValue := v.Field(i)
        
        if fieldValue.IsZero() {
            continue
        }
        
        fieldName := getJSONTag(field)
        if fieldName == "" {
            continue
        }
        
        // 处理不同类型的值
        var value string
        switch fieldValue.Kind() {
        case reflect.Slice:
            // 数组类型：转换为逗号分隔的字符串
            values := make([]string, 0)
            for j := 0; j < fieldValue.Len(); j++ {
                values = append(values, fmt.Sprintf("%v", fieldValue.Index(j).Interface()))
            }
            value = strings.Join(values, ",")
        default:
            value = fmt.Sprintf("%v", fieldValue.Interface())
        }
        
        values.Set(fieldName, value)
    }
    
    return values.Encode(), nil
}

// getJSONTag 获取字段的 json 标签
func getJSONTag(field reflect.StructField) string {
    jsonTag := field.Tag.Get("json")
    if jsonTag == "" {
        return strings.ToLower(field.Name)
    }
    parts := strings.Split(jsonTag, ",")
    return parts[0]
}
```

### 3. 主函数：BuildFunctionUrl（放在 `sdk/agent-app/app` 下）

```go
// sdk/agent-app/app/url_builder.go
package app

import (
    "fmt"
    "strings"
    "github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
)

// BuildFunctionUrl 构建跳转 URL（支持函数跳转和外链）
// 支持两种模式：
// 1. 函数跳转：传递结构体参数，自动转换为函数 URL
// 2. 外链跳转：传递字符串（如 "www.baidu.com"），直接作为外链处理
func (ctx *Context) BuildFunctionUrl(
    target string, // 函数路径（如 "meeting_room_list"）或外链（如 "www.baidu.com"）
    params interface{}, // 结构体参数（函数跳转）或 nil（外链跳转）
) (string, error) {
    // 1. 判断是否是外链
    if isExternalLink(target) {
        // 外链模式：直接处理字符串
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
        // Table 函数：根据 search 标签转换为 search 格式
        // 使用 AutoCrudTable 的 Model（包含 search 标签）
        if params != nil {
            queryString, err = query.StructToTableParams(params)
        }
    case TemplateTypeForm:
        // Form 函数：转换为 k=v 格式
        // 使用 Request 结构体
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

// GetFunctionTemplate 根据函数路径获取函数模板
// 判断逻辑：
// 1. 从函数路径构建完整的路由路径（相对路径需要加上 RouterGroup）
// 2. 从 app.routerInfo 中查找对应的路由信息（需要 HTTP 方法，默认使用 GET）
// 3. 获取 Template 字段（实现了 Templater 接口）
// 4. 调用 TemplateType() 方法判断类型（Table 或 Form）
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
    
    // 2. 从 app.routerInfo 中查找路由信息
    // 注意：需要知道 HTTP 方法，默认使用 GET（因为大多数 Table 和 Form 函数都是 GET）
    // 如果找不到 GET，可以尝试其他方法（POST、PUT、DELETE）
    methods := []string{"GET", "POST", "PUT", "DELETE"}
    for _, method := range methods {
        router, err := app.getRouter(fullRouter, method)
        if err == nil && router != nil {
            // 找到了路由信息，返回 Template
            return router.Template, nil
        }
    }
    
    return nil, fmt.Errorf("未找到函数 %s 的路由信息", functionPath)
}

// GetRouterGroup 获取当前请求的 RouterGroup
func (ctx *Context) GetRouterGroup() string {
    if ctx.routerInfo != nil && ctx.routerInfo.Options != nil && ctx.routerInfo.Options.RouterGroup != nil {
        return ctx.routerInfo.Options.RouterGroup.RouterGroup
    }
    return ""
}

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

## 使用示例

### 示例 1：会议室预约跳转到会议室详情（Table 函数）

```go
// 在 CrmMeetingRoomBooking 中添加链接字段
type CrmMeetingRoomBooking struct {
    // ... 其他字段
    
    RoomLink string `json:"room_link" gorm:"-" widget:"name:会议室详情;type:link;target:_blank"`
}

// 在 CrmMeetingRoomBookingList 函数中
func CrmMeetingRoomBookingList(ctx *app.Context, resp response.Response) error {
    // ... 查询数据
    
    for i := range bookings {
        // ✅ 直接使用目标函数的结构体（CrmMeetingRoom）
        params := CrmMeetingRoom{
            ID: bookings[i].RoomID, // 只需要设置需要传递的字段
        }
        
        bookings[i].RoomLink = ctx.BuildFunctionUrl(
            "meeting_room_list",
            params,
        )
        // 生成的 URL: meeting_room_list?eq=id:1
    }
    
    return resp.Table(&bookings).Build()
}
```

### 示例 2：外链跳转

```go
// 在 CrmMeetingRoomBooking 中添加外链字段
type CrmMeetingRoomBooking struct {
    // ... 其他字段
    
    ExternalLink string `json:"external_link" gorm:"-" widget:"name:外部链接;type:link;target:_blank"`
}

// 在 CrmMeetingRoomBookingList 函数中
func CrmMeetingRoomBookingList(ctx *app.Context, resp response.Response) error {
    // ... 查询数据
    
    for i := range bookings {
        // ✅ 外链模式：直接传递字符串
        bookings[i].ExternalLink = ctx.BuildFunctionUrl(
            "www.baidu.com", // 外链，自动识别
            nil,             // 外链不需要参数
        )
        // 生成的 URL: https://www.baidu.com
        
        // 也可以传递完整的 URL
        bookings[i].ExternalLink = ctx.BuildFunctionUrl(
            "https://www.google.com", // 完整的 URL
            nil,
        )
        // 生成的 URL: https://www.google.com
    }
    
    return resp.Table(&bookings).Build()
}
```

### 示例 3：工单列表 - Table 跳转（使用 in 条件，多个状态值）

```go
// ✅ 不需要定义新的参数结构体，直接使用目标函数的 AutoCrudTable Model
// 假设 CrmTicketList 的 AutoCrudTable 是 CrmTicket

// 在列表函数中
params := CrmTicket{
    Status:   []string{"待处理", "处理中"}, // 多个值，自动转换为 in=status:待处理,处理中
    Priority: "高",
}

link := ctx.BuildFunctionUrl(
    "crm_ticket_list",
    params,
)
// 生成的 URL: crm_ticket_list?in=status:待处理,处理中&eq=priority:高
```

### 示例 4：投票系统 - 动态跳转（Form 函数）

```go
// 后端代码
type VoteSystemTopic struct {
    // ... 其他字段
    
    VoteActionLink string `json:"vote_action_link" gorm:"-" widget:"name:投票操作;type:link;target:_blank"`
}

// ✅ 不需要定义新的参数结构体，直接使用目标函数的 Request 结构体
// 假设 VoteSystemResult 的 Request 是 VoteSystemResultReq
// 假设 VoteSystemSubmit 的 Request 是 VoteSystemSubmitReq

func VoteSystemTopicList(ctx *app.Context, resp response.Response) error {
    // ... 查询数据
    
    for i := range topics {
        // 根据状态和投票情况动态生成链接
        if topics[i].Status == "已结束" || topics[i].Status == "未开始" {
            // ✅ 直接使用目标函数的 Request 结构体
            params := VoteSystemResultReq{
                TopicID: topics[i].ID, // 只需要设置需要传递的字段
            }
            topics[i].VoteActionLink = ctx.BuildFunctionUrl(
                "vote_system_result",
                params,
            )
        } else if topics[i].Status == "进行中" {
            if hasUserVoted {
                // 已投票，跳转到结果查询页面
                params := VoteSystemResultReq{
                    TopicID: topics[i].ID,
                }
                topics[i].VoteActionLink = ctx.BuildFunctionUrl(
                    "vote_system_result",
                    params,
                )
            } else {
                // 未投票，跳转到参与投票页面
                params := VoteSystemSubmitReq{
                    TopicID: topics[i].ID,
                }
                topics[i].VoteActionLink = ctx.BuildFunctionUrl(
                    "vote_system_submit",
                    params,
                )
            }
        }
    }
    
    return resp.Table(&topics).Build()
}
```

---

## 关键问题解答

### Q1: `in` 条件如何处理？是否难以实现？
**A**: 非常简单！直接使用目标函数的结构体，使用 `[]string` 类型，后端自动转换。

```go
// ✅ 不需要定义新的参数结构体，直接使用目标函数的结构体
// 假设 CrmTicketList 的 AutoCrudTable 是 CrmTicket

// 使用
params := CrmTicket{
    Status: []string{"待处理", "处理中"}, // 直接传递数组
}

// 后端自动转换为：in=status:待处理,处理中
// 生成的 URL: crm_ticket_list?in=status:待处理,处理中
```

**优势**：
- ✅ 不需要定义新的参数结构体
- ✅ 不需要手动拼接字符串
- ✅ 类型安全，编译时检查
- ✅ 自动处理 URL 编码

### Q2: Table 和 Form 的参数格式如何自动区分？
**A**: 后端自动判断，无需手动指定！通过获取目标函数的模板信息自动判断：

```go
// Table 函数：自动判断，根据 search 标签自动转换
params := VoteOption{
    TopicID: 123,
}
link := ctx.BuildFunctionUrl(
    "vote_system_option_list",
    params,
)
// 生成的 URL: vote_system_option_list?eq=topic_id:123

// Form 函数：自动判断，自动转换为 k=v 格式
params := VoteSystemResultReq{
    TopicID: 123,
}
link := ctx.BuildFunctionUrl(
    "vote_system_result",
    params,
)
// 生成的 URL: vote_system_result?topic_id=123
```

### Q3: 后端如何自动判断是 Table 还是 Form？
**A**: 通过获取目标函数的模板信息，调用 `TemplateType()` 方法判断：

**判断逻辑**：
1. **路由信息存储**：所有注册的路由信息存储在 `app.routerInfo` 中，key 是 `{router}.{method}`
2. **模板接口**：`TableTemplate` 和 `FormTemplate` 都实现了 `Templater` 接口
3. **类型判断**：通过 `TemplateType()` 方法返回 `TemplateTypeTable` 或 `TemplateTypeForm`
4. **路径构建**：相对路径需要加上当前 `RouterGroup`，绝对路径直接使用

```go
// GetFunctionTemplate 根据函数路径获取函数模板
func (ctx *Context) GetFunctionTemplate(functionPath string) (Templater, error) {
    // 1. 构建完整的路由路径
    var fullRouter string
    if strings.HasPrefix(functionPath, "/") {
        fullRouter = strings.Trim(functionPath, "/")
    } else {
        routerGroup := ctx.GetRouterGroup()
        fullRouter = fmt.Sprintf("%s/%s", strings.Trim(routerGroup, "/"), strings.Trim(functionPath, "/"))
    }
    
    // 2. 从 app.routerInfo 中查找路由信息（尝试多个 HTTP 方法）
    methods := []string{"GET", "POST", "PUT", "DELETE"}
    for _, method := range methods {
        router, err := app.getRouter(fullRouter, method)
        if err == nil && router != nil {
            // 找到了路由信息，返回 Template
            return router.Template, nil
        }
    }
    
    return nil, fmt.Errorf("未找到函数 %s 的路由信息", functionPath)
}

// 使用示例
template, err := ctx.GetFunctionTemplate("meeting_room_list")
if err != nil {
    return err
}

// 判断类型
switch template.TemplateType() {
case TemplateTypeTable:
    // Table 函数：使用 AutoCrudTable 的 Model
    queryString, err = query.StructToTableParams(params)
case TemplateTypeForm:
    // Form 函数：使用 Request 结构体
    queryString, err = StructToFormParams(params)
}
```

### Q4: 如何支持外链跳转？
**A**: 直接传递字符串即可，后端自动识别为外链：

```go
// ✅ 外链模式：直接传递字符串
link := ctx.BuildFunctionUrl(
    "www.baidu.com", // 外链，自动识别
    nil,             // 外链不需要参数
)
// 生成的 URL: https://www.baidu.com

// ✅ 也可以传递完整的 URL
link := ctx.BuildFunctionUrl(
    "https://www.google.com", // 完整的 URL
    nil,
)
// 生成的 URL: https://www.google.com

// ✅ 函数跳转模式：传递结构体参数
params := CrmMeetingRoom{
    ID: 1,
}
link := ctx.BuildFunctionUrl(
    "meeting_room_list", // 函数路径
    params,              // 结构体参数
)
// 生成的 URL: meeting_room_list?eq=id:1
```

**外链识别规则**：
- 包含 `http://` 或 `https://`：直接识别为外链
- 包含 `www.`：识别为外链
- 包含常见顶级域名（`.com`、`.cn`、`.org` 等）：识别为外链
- 其他情况：识别为函数路径

### Q5: 如何知道使用哪个结构体？
**A**: 
- **Table 函数**：使用 `AutoCrudTable` 的 Model（如 `CrmMeetingRoom`）
- **Form 函数**：使用 `Request` 结构体（如 `VoteSystemResultReq`）

```go
// 示例：从 CrmMeetingRoomBooking 跳转到 CrmMeetingRoom
// 查看 CrmMeetingRoomListTemplate，AutoCrudTable 是 CrmMeetingRoom
// 所以直接使用 CrmMeetingRoom 结构体
params := CrmMeetingRoom{
    ID: bookings[i].RoomID,
}
```

---

## 前端实现

### LinkWidget 组件

```vue
<!-- web/src/core/widgets-v2/components/LinkWidget.vue -->
<template>
  <div class="link-widget">
    <!-- 编辑模式：不显示（链接是只读的） -->
    <div v-if="mode === 'edit'" class="link-disabled">
      <el-icon><Link /></el-icon>
      <span>{{ field.name }}</span>
    </div>
    
    <!-- 响应/详情/表格模式：显示链接 -->
    <el-link
      v-else-if="resolvedUrl"
      :href="resolvedUrl"
      :target="linkConfig.target || '_self'"
      :type="linkConfig.type || 'primary'"
      @click="handleClick"
    >
      <el-icon v-if="linkConfig.icon"><component :is="linkConfig.icon" /></el-icon>
      {{ linkText }}
    </el-link>
    
    <!-- 空值显示 -->
    <span v-else class="empty-text">-</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { Link } from '@element-plus/icons-vue'
import type { WidgetComponentProps } from '../types'

const props = defineProps<WidgetComponentProps>()
const router = useRouter()

// 解析 URL（后端已经生成完整 URL，前端直接使用）
const resolvedUrl = computed(() => {
  const url = props.value?.raw || ''
  if (!url) return ''
  
  // 如果是绝对路径（以 / 开头），直接使用
  if (url.startsWith('/')) {
    return url
  }
  
  // 如果是外链（包含 http:// 或 https://），直接使用
  if (url.startsWith('http://') || url.startsWith('https://')) {
    return url
  }
  
  // 相对路径，需要转换为完整路径
  return buildFullPath(url)
})

// 链接文本
const linkText = computed(() => {
  return props.field.widget?.text || props.value?.display || props.field.name || '链接'
})

// 链接配置
const linkConfig = computed(() => {
  const widget = props.field.widget
  if (!widget || widget.type !== 'link') {
    return {}
  }
  
  return {
    type: widget.type || 'primary',
    target: widget.target || '_self',
    icon: widget.icon,
  }
})

// 处理点击事件
const handleClick = () => {
  const url = resolvedUrl.value
  if (!url) return
  
  // 如果是新窗口打开
  if (linkConfig.value.target === '_blank') {
    window.open(url, '_blank')
  } else {
    // 当前窗口跳转
    // 需要将 URL 转换为路由路径
    const routePath = convertUrlToRoute(url)
    router.push(routePath)
  }
}

// 构建完整路径
function buildFullPath(relativePath: string): string {
  // 解析相对路径：function_name?query
  const [functionPath, query] = relativePath.split('?')
  
  // 从当前路由获取 user 和 app
  const currentRoute = router.currentRoute.value
  const pathParts = currentRoute.path.split('/').filter(Boolean)
  
  if (pathParts.length < 3) {
    // 如果路径格式不正确，返回原路径
    return relativePath
  }
  
  const user = pathParts[1]
  const app = pathParts[2]
  
  // 构建完整路径
  const fullPath = `/workspace/${user}/${app}/${functionPath}`
  return query ? `${fullPath}?${query}` : fullPath
}

// 将 URL 转换为路由路径
function convertUrlToRoute(url: string): string {
  // 如果已经是完整路径，直接使用
  if (url.startsWith('/workspace/')) {
    return url
  }
  
  // 如果是外链，直接返回
  if (url.startsWith('http://') || url.startsWith('https://')) {
    return url
  }
  
  // 否则使用 buildFullPath
  return buildFullPath(url)
}
</script>

<style scoped>
.link-widget {
  display: inline-flex;
  align-items: center;
}

.link-disabled {
  display: inline-flex;
  align-items: center;
  color: var(--el-text-color-placeholder);
  gap: 4px;
}

.empty-text {
  color: var(--el-text-color-placeholder);
}
</style>
```

### 注册组件

```typescript
// web/src/core/factories-v2/index.ts
import LinkWidget from '../widgets-v2/components/LinkWidget.vue'

// 注册 Link 组件
widgetComponentFactory.registerRequestComponent(WidgetType.LINK, LinkWidget)
widgetComponentFactory.registerResponseComponent(WidgetType.LINK, LinkWidget)
```

---

## 总结

### 核心设计
1. **直接使用目标函数的结构体**：不需要定义新的参数结构体
   - Table 函数：使用 `AutoCrudTable` 的 Model（如 `CrmMeetingRoom`）
   - Form 函数：使用 `Request` 结构体（如 `VoteSystemResultReq`）
2. **自动判断函数类型**：后端自动判断是 Table 还是 Form，无需手动指定
3. **自动转换**：
   - Table 函数：根据 search 标签自动转换为 `eq=field:value`、`in=field:value1,value2` 等格式
   - Form 函数：自动转换为 `k=v` 格式
4. **支持外链**：传递字符串（如 `www.baidu.com`）时，自动识别为外链
5. **类型安全**：结构体字段有类型检查，编译时就能发现错误
6. **支持复杂条件**：`in` 条件使用 `[]string` 类型，自然支持多值

### 实现步骤
1. ✅ 后端：在 `pkg/gormx/query` 下实现 `StructToTableParams` 函数
2. ✅ 后端：在 `sdk/agent-app/app` 下实现 `StructToFormParams` 函数
3. ✅ 后端：在 `sdk/agent-app/app` 下实现 `BuildFunctionUrl` 主函数
4. ✅ 后端：在 `sdk/agent-app/app` 下实现 `GetFunctionTemplate` 获取模板信息
5. ✅ 前端：实现 `LinkWidget` 组件（接收完整 URL）
6. ✅ 前端：注册组件到工厂
7. ✅ 测试：验证各种场景

### 优势对比

| 特性 | 手动构造 URL | 结构体参数（简化版） |
|------|------------|-------------------|
| **简单性** | ⚠️ 需要手动构造字符串 | ✅ 直接使用目标函数的结构体 |
| **类型安全** | ❌ 字符串，容易出错 | ✅ 结构体，编译时检查 |
| **维护性** | ❌ 修改字段名需要改字符串 | ✅ 修改字段名只需要改目标函数的结构体 |
| **复杂条件** | ⚠️ 需要手动处理 `in` 条件 | ✅ `[]string` 自然支持 |
| **自动转换** | ❌ 需要手动处理格式 | ✅ 根据 search 标签自动转换 |
| **定义结构体** | ❌ 不需要 | ✅ **不需要**（直接使用目标函数的结构体） |
| **指定类型** | ❌ 不需要 | ✅ **不需要**（后端自动判断） |
| **支持外链** | ⚠️ 需要手动判断 | ✅ 自动识别外链 |

### 使用示例对比

#### 旧方式（手动构造）
```go
// ❌ 容易出错，需要手动处理格式
optionsParams := fmt.Sprintf("eq=topic_id:%d&sort=id:desc", topics[i].ID)
topics[i].OptionsLink = fmt.Sprintf("vote_system_option_list?%s", optionsParams)

// ❌ in 条件需要手动处理
statusParams := fmt.Sprintf("in=status:%s", strings.Join([]string{"待处理", "处理中"}, ","))
```

#### 新方式（直接使用目标函数的结构体）
```go
// ✅ 不需要定义新的参数结构体
// ✅ 不需要指定 targetType，后端自动判断
// ✅ 类型安全，自动转换
params := VoteOption{
    TopicID: topics[i].ID,
}
topics[i].OptionsLink = ctx.BuildFunctionUrl(
    "vote_system_option_list",
    params,
)

// ✅ in 条件自然支持
params := CrmTicket{
    Status: []string{"待处理", "处理中"}, // 直接使用数组
}
link := ctx.BuildFunctionUrl("crm_ticket_list", params)
```

### 关键优势
- ✅ **更简单**：不需要手动构造 URL 字符串，直接传递结构体
- ✅ **类型安全**：结构体字段有类型检查，编译时就能发现错误
- ✅ **自动转换**：后端根据 search 标签自动转换为正确的格式
- ✅ **支持复杂条件**：`in` 条件使用 `[]string` 类型，自然支持多值，不需要手动拼接
- ✅ **易于维护**：修改字段名时只需要改结构体，不需要改字符串
- ✅ **代码清晰**：结构体定义一目了然，比字符串更容易理解
- ✅ **支持外链**：自动识别外链，无需特殊处理

### 后续优化
1. ✅ 支持外链跳转（已实现）
2. 支持嵌套结构体（如范围查询 `Progress: {Min: 50, Max: 80}`）
3. 支持变量替换（如 `$id`、`$create_by`，如果需要）
4. 支持条件表达式（如根据状态动态选择参数）
5. 支持 URL 编码和安全性验证
6. 支持链接图标和样式自定义
7. 外链安全性验证（白名单、黑名单等）
