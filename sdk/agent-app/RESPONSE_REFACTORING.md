# 响应数据结构化重构

## 🎯 重构目标

将原本使用`map[string]interface{}`的响应数据改为结构化的类型安全的数据结构，提高代码的可维护性和类型安全性。

## 📁 重构内容

### 1. 创建专门的model目录

```
sdk/agent-app/model/
├── api.go        # API相关数据结构
└── response.go   # 响应相关数据结构
```

### 2. 数据结构重组

#### 重构前（使用map）
```go
func (a *App) sendUpdateResponse(status, message string, data interface{}) {
    response := map[string]interface{}{
        "status":    status,
        "message":   message,
        "data":      data,
        "version":   env.Version,
        "timestamp": time.Now().Format(time.RFC3339),
    }

    responseData, _ := json.Marshal(response)
    a.conn.Publish(subject, responseData)
}
```

#### 重构后（使用结构体）
```go
// 成功响应
func (a *App) sendSuccessResponse(message string, data *model.DiffData) {
    response := &model.UpdateResponse{
        Status:    "success",
        Message:   message,
        Data:      data,
        Version:   env.Version,
        Timestamp: time.Now(),
    }

    responseData, _ := json.Marshal(response)
    a.conn.Publish(subject, responseData)
}

// 错误响应
func (a *App) sendErrorResponse(message string) {
    response := &model.UpdateResponse{
        Status:    "error",
        Message:   message,
        Data:      nil,
        Version:   env.Version,
        Timestamp: time.Now(),
    }

    responseData, _ := json.Marshal(response)
    a.conn.Publish(subject, responseData)
}
```

### 3. 新的数据结构

#### UpdateResponse - 统一响应格式
```go
type UpdateResponse struct {
    Status    string      `json:"status"`     // 状态: success, error
    Message   string      `json:"message"`    // 响应消息
    Data      *DiffData  `json:"data"`       // 差异数据
    Version   string      `json:"version"`    // 当前版本
    Timestamp time.Time   `json:"timestamp"`  // 响应时间
}
```

#### DiffData - 差异数据
```go
type DiffData struct {
    Add    []*ApiInfo `json:"add"`    // 新增的API
    Update []*ApiInfo `json:"update"` // 修改的API
    Delete []*ApiInfo `json:"delete"` // 删除的API
}
```

#### ApiInfo - API信息（移到model包）
```go
type ApiInfo struct {
    Code           string          `json:"code"`
    Name           string          `json:"name"`
    Desc           string          `json:"desc"`
    Tags           []string        `json:"tags"`
    Router         string          `json:"router"`
    Method         string          `json:"method"`
    CreateTables   []string        `json:"create_tables"`
    Request        []*widget.Field `json:"request"`
    Response       []*widget.Field `json:"response"`
    AddedVersion   string          `json:"added_version"`
    UpdateVersions []string        `json:"update_versions"`
}
```

## ✅ 重构优势

### 1. 类型安全
```go
// 重构前：运行时才能发现错误
message := response["message"].(string)  // 可能panic

// 重构后：编译时就能发现错误
message := response.Message            // 类型安全
```

### 2. 代码提示和自动补全
```go
// IDE可以提供完整的代码提示
response.Status
response.Message
response.Data.Add[0].Name
```

### 3. 更好的文档和自描述性
```go
// 结构体本身就是文档
type UpdateResponse struct {
    Status    string      `json:"status"`     // 状态: success, error
    Message   string      `json:"message"`    // 响应消息
    Data      *DiffData  `json:"data"`       // 差异数据
    Version   string      `json:"version"`    // 当前版本
    Timestamp time.Time   `json:"timestamp"`  // 响应时间
}
```

### 4. 更容易测试
```go
// 重构前：难以构造测试数据
testResponse := map[string]interface{}{
    "status": "success",
    "data": map[string]interface{}{...},
}

// 重构后：类型安全的测试数据
testResponse := &model.UpdateResponse{
    Status:  "success",
    Message: "test message",
    Data: &model.DiffData{
        Add: []*model.ApiInfo{testApi},
    },
}
```

### 5. 更好的版本控制
```go
// 结构体字段变更更容易追踪和代码审查
type UpdateResponse struct {
    Status    string      `json:"status"`     // 新增字段时容易发现
    NewField  string      `json:"new_field"` // 新增字段一目了然
    Version   string      `json:"version"`    // 修改影响范围明确
    Timestamp time.Time   `json:"timestamp"`  // 类型变更容易被检测
}
```

## 📊 响应格式示例

### 成功响应
```json
{
  "status": "success",
  "message": "API diff completed successfully",
  "data": {
    "add": [
      {
        "code": "crm_ticket",
        "name": "工单管理",
        "router": "/crm/crm_ticket",
        "method": "GET",
        "added_version": "v3",
        "update_versions": []
      }
    ],
    "update": [
      {
        "code": "user_management",
        "name": "用户管理",
        "router": "/user/user_management",
        "method": "GET",
        "added_version": "v1",
        "update_versions": ["v2", "v3", "v5"]
      }
    ],
    "delete": []
  },
  "version": "v5",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 错误响应
```json
{
  "status": "error",
  "message": "Failed to get current APIs: connection timeout",
  "data": null,
  "version": "v5",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## 🔧 使用方式

### 1. 发送成功响应
```go
diffData := &model.DiffData{
    Add:    newAPIs,
    Update: modifiedAPIs,
    Delete: deletedAPIs,
}

a.sendSuccessResponse("API diff completed successfully", diffData)
```

### 2. 发送错误响应
```go
a.sendErrorResponse("Failed to get current APIs: connection timeout")
```

### 3. 接收和解析响应
```go
// 前端或其他服务接收响应
var response model.UpdateResponse
err := json.Unmarshal(responseData, &response)
if err != nil {
    return err
}

// 类型安全的访问
if response.Status == "success" {
    fmt.Printf("新增了 %d 个API\n", len(response.Data.Add))
    for _, api := range response.Data.Add {
        fmt.Printf("- %s (%s)\n", api.Name, api.AddedVersion)
    }
}
```

## 🚀 未来扩展

### 1. 响应验证
```go
// 可以添加验证方法
func (r *UpdateResponse) Validate() error {
    if r.Status != "success" && r.Status != "error" {
        return fmt.Errorf("invalid status: %s", r.Status)
    }
    if r.Status == "success" && r.Data == nil {
        return fmt.Errorf("data cannot be null for success response")
    }
    return nil
}
```

### 2. 响应转换
```go
// 可以添加转换方法
func (r *UpdateResponse) ToJSON() ([]byte, error) {
    return json.MarshalIndent(r, "", "  ")
}

func (r *UpdateResponse) ToPrettyString() string {
    data, _ := json.MarshalIndent(r, "", "  ")
    return string(data)
}
```

### 3. 响应中间件
```go
// 可以创建响应中间件
func WithLogging(handler func() *UpdateResponse) *UpdateResponse {
    start := time.Now()
    response := handler()
    log.Printf("Response took %v", time.Since(start))
    return response
}
```

## ✅ 总结

这次重构带来了以下好处：

1. **类型安全**: 编译时检查，减少运行时错误
2. **代码提示**: IDE支持，提高开发效率
3. **更好的维护性**: 结构体更易理解和修改
4. **测试友好**: 类型安全的测试数据构造
5. **文档清晰**: 结构体本身就是文档
6. **版本控制友好**: 变更更容易追踪

这次重构让Kageos的API diff功能更加健壮和易于维护！🎉