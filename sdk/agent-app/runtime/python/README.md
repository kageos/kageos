# Python Runtime SDK

Python 代码执行 SDK，支持在 Go 代码中执行 Python 脚本，并按固定入口函数协议解析结构化结果与输出文件。

## 🚀 快速开始

### 基本使用（Builder 模式）⭐

使用 Builder 模式，灵活且易于扩展。

```go
import "github.com/kageos/kageos/sdk/agent-app/runtime/python"

ctx := context.Background()

code := `
def kageos_entry(args, output_dir):
    return {
        "data": {
            "sum": args["a"] + args["b"],
        },
    }
`

// 定义请求结构体
type Request struct {
    A int `json:"a"`
    B int `json:"b"`
}

// 定义结果结构体
type Result struct {
    Sum int `json:"sum"`
}

var result Result
req := Request{A: 10, B: 20}

// Builder 模式：链式调用，灵活且易于扩展
executor := python.NewExecutor(code).
    WithRequest(req).
    WithTimeout(30 * time.Second)
defer executor.Close() // 默认临时工作目录须释放，避免磁盘泄漏

err := executor.ExecuteJSON(ctx, &result)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("结果: %+v\n", result)
// 输出: 结果: {Sum:30}
```

### 复杂场景示例

```go
// Builder 模式支持链式调用，可以逐步添加配置
executor := python.NewExecutor(code).
    WithRequest(req).
    WithPackages("pandas", "numpy").  // 可以添加多个包
    WithTimeout(2 * time.Minute).      // 设置超时
    WithWorkDir("/tmp/my-work")        // 设置工作目录（可选）
defer executor.Close() // WithWorkDir 时不会删除你指定的目录，仅清理 SDK 创建的临时目录（若有）

err := executor.ExecuteJSON(ctx, &result)
```

## 📚 API 文档

### NewExecutor（推荐，Builder 模式）

创建新的 Python 执行器，使用 Builder 模式灵活配置。

```go
func NewExecutor(code string) *Executor
```

**参数**：
- `code`: Python 代码字符串

**返回**：`*Executor` 执行器实例

**优势**：
- ✅ 链式调用，代码清晰
- ✅ 易于扩展，可以逐步添加配置
- ✅ 灵活，可以根据条件动态配置

### WithRequest

设置请求结构体（会序列化为 JSON 传递给 Python）。

```go
func (e *Executor) WithRequest(req interface{}) *Executor
```

**参数**：
- `req`: 请求结构体（必须是可 JSON 序列化的类型）

**返回**：`*Executor` 支持链式调用

**说明**：
- 请求结构体会自动序列化为 JSON，作为 `kageos_entry(args, output_dir)` 的第一个参数传给 Python
- Python 端必须定义固定入口函数 `kageos_entry(args, output_dir)`
- 返回值必须为 dict，支持 `data`、`output_files`、`warnings` 三个字段
- 支持嵌套结构体、数组、字典等复杂类型

### WithPackages

设置需要安装的 Python 包。

```go
func (e *Executor) WithPackages(packages ...string) *Executor
```

**参数**：
- `packages`: 包名列表（例如: "pandas", "numpy"）

**返回**：`*Executor` 支持链式调用

### WithTimeout

设置执行超时时间。

```go
func (e *Executor) WithTimeout(timeout time.Duration) *Executor
```

**参数**：
- `timeout`: 超时时间（例如: `2 * time.Minute`）

**返回**：`*Executor` 支持链式调用

### WithWorkDir

设置工作目录。

```go
func (e *Executor) WithWorkDir(workDir string) *Executor
```

**参数**：
- `workDir`: 工作目录路径（如果为空，则使用临时目录）

**返回**：`*Executor` 支持链式调用

### WithPythonPath

设置 Python 解释器路径。

```go
func (e *Executor) WithPythonPath(pythonPath string) *Executor
```

**参数**：
- `pythonPath`: Python 解释器路径（如果为空，则自动检测）

**返回**：`*Executor` 支持链式调用

### Execute

执行 Python 代码，返回原始输出。

```go
func (e *Executor) Execute(ctx context.Context) ([]byte, error)
```

**参数**：
- `ctx`: 上下文（用于超时控制）

**返回**：
- `[]byte`: 执行输出
- `error`: 错误

### ExecuteJSON

执行 Python 代码，自动解析 JSON 输出到 result。

```go
func (e *Executor) ExecuteJSON(ctx context.Context, result interface{}) error
```

**参数**：
- `ctx`: 上下文（用于超时控制）
- `result`: 结果结构体指针（必须是可 JSON 反序列化的类型）

**返回**：
- `error`: 错误

### Close（资源释放）⭐

使用**默认临时工作目录**（未设置 `WithWorkDir`）时，`Execute` / `ExecuteJSON` 会在系统临时目录下创建工作区，**用完后必须调用 `Close`**，否则会泄漏磁盘。

```go
func (e *Executor) Close() error
```

- **推荐写法**：在创建 `Executor` 后立刻 `defer executor.Close()`，与 `defer f.Close()` 习惯一致。
- **`WithWorkDir` 指定的目录不会被删除**，`Close` 只 `RemoveAll` 由 SDK 内部 `MkdirTemp` 创建的目录。
- **幂等**：可安全多次调用 `Close()`。
- **Close 之后**不要再调用 `Execute` / `ExecuteJSON`（会返回错误）。

## 💡 使用示例

### 示例 1：简单计算

```go
code := `
import json
result = {"sum": a + b, "product": a * b}
print(json.dumps(result))
`

// 定义请求结构体
type Request struct {
    A int `json:"a"`
    B int `json:"b"`
}

var result struct {
    Sum     int `json:"sum"`
    Product int `json:"product"`
}

req := Request{A: 10, B: 20}
executor := python.NewExecutor(code).
    WithRequest(req)
defer executor.Close()

err := executor.ExecuteJSON(ctx, &result)
// result.Sum = 30, result.Product = 200
```

### 示例 2：使用 Pandas 分析数据

```go
code := `
import pandas as pd
import json

df = pd.DataFrame(data)
summary = {
    "total": len(df),
    "columns": df.columns.tolist(),
    "mean": df["value"].mean()
}
print(json.dumps(summary))
`

// 定义请求结构体
type Request struct {
    Data []map[string]interface{} `json:"data"`
}

req := Request{
    Data: []map[string]interface{}{
        {"name": "Alice", "value": 100},
        {"name": "Bob", "value": 200},
    },
}

var result struct {
    Total   int      `json:"total"`
    Columns []string `json:"columns"`
    Mean    float64  `json:"mean"`
}

executor := python.NewExecutor(code).
    WithRequest(req).
    WithPackages("pandas").
    WithTimeout(2 * time.Minute)
defer executor.Close()

err := executor.ExecuteJSON(ctx, &result)
```

### 示例 3：在 Form 函数中使用

```go
func MyFunction(ctx *app.Context, resp response.Response) error {
    var req MyRequest
    if err := ctx.ShouldBindValidate(&req); err != nil {
        return err
    }

    code := `
import json
result = {"message": f"Hello, {name}!"}
print(json.dumps(result))
`

    var result struct {
        Message string `json:"message"`
    }

    // 直接使用请求结构体
    executor := python.NewExecutor(code).
        WithRequest(req).
        WithTimeout(30 * time.Second)
    defer executor.Close()

    err := executor.ExecuteJSON(ctx, &result)
    if err != nil {
        return err
    }

    return resp.Form(&MyResponse{
        Message: result.Message,
    }).Build()
}
```

### 示例 4：使用字典作为请求（简单场景）

```go
// 对于简单场景，可以使用 map[string]interface{}
request := map[string]interface{}{
    "name": "Alice",
    "age":  30,
    "city": "Beijing",
}

executor := python.NewExecutor(code).
    WithRequest(request).
    WithTimeout(30 * time.Second)
defer executor.Close()

output, err := executor.Execute(ctx)
```

## ⚠️ 注意事项

### 1. JSON 输出要求

- Python 代码必须输出 JSON 格式
- 建议使用 `json.dumps()` 输出
- 可以包含其他输出（如日志），但 JSON 必须在最后

**正确示例**：
```python
import json
print("开始计算...")  # 可以输出日志
result = {"sum": a + b}
print(json.dumps(result))  # JSON 输出必须在最后
```

### 2. 错误处理

- Python 异常会被捕获并返回错误
- 建议在 Python 代码中使用 try-except

**示例**：
```python
try:
    result = a / b
except ZeroDivisionError:
        result = 0
print(json.dumps({"result": result}))
```

### 3. 性能考虑

- 每次执行都会启动新的 Python 进程
- 如果需要高性能，可以考虑 Python HTTP 服务
- **包安装优化**：SDK 实现了三步检查机制，大幅提升性能
  1. **环境变量快速检查**（< 0.001 秒）：从 `PYTHON_INSTALLED_PACKAGES` 环境变量快速查找
  2. **导入检查**（< 0.1 秒）：尝试导入包，验证是否已安装
  3. **pip 安装**：仅在未安装时执行，避免重复安装
- 预装的包（如 pandas、numpy、matplotlib 等）可以瞬间跳过，无需等待

### 4. 安全性

- 代码执行需要沙箱隔离
- 限制可安装的包
- 限制执行时间（使用 WithTimeout）

### 5. Python 路径检测

自动检测顺序：
1. `WithPythonPath()` 设置的路径
2. 环境变量 `PYTHON_PATH`
3. 常见路径：`/usr/bin/python3`, `/usr/local/bin/python3`
4. 系统 PATH 中的 `python3` 或 `python`

## 🔧 高级用法

### 自定义工作目录

```go
executor := python.NewExecutor(code).
    WithWorkDir("/tmp/my-python-scripts").
    WithTimeout(30 * time.Second)
defer executor.Close()
```

### 自定义 Python 路径

```go
executor := python.NewExecutor(code).
    WithPythonPath("/usr/local/bin/python3.11").
    WithTimeout(30 * time.Second)
defer executor.Close()
```

### 安装多个包

```go
executor := python.NewExecutor(code).
    WithPackages("pandas", "numpy", "matplotlib").
    WithTimeout(2 * time.Minute)
defer executor.Close()
```

## 📝 最佳实践

1. **总是设置超时**：避免 Python 代码无限执行
   ```go
   executor.WithTimeout(30 * time.Second)
   ```

2. **使用 ExecuteJSON**：自动解析 JSON，类型安全
   ```go
   var result MyResult
   err := executor.ExecuteJSON(ctx, &result)
   ```

3. **默认临时目录必须 Close**：`defer executor.Close()`，避免泄漏

4. **错误处理**：检查错误并记录日志
   ```go
   if err != nil {
       logger.Errorf(ctx, "Python 执行失败: %v", err)
       return err
   }
   ```

5. **包管理**：只在需要时安装包
   ```go
   executor.WithPackages("pandas")  // 只在需要时安装
   ```

## 🐛 故障排查

### 问题：找不到 Python 解释器

**解决方案**：
- 确保系统已安装 Python 3
- 使用 `WithPythonPath()` 指定 Python 路径
- 检查环境变量 `PYTHON_PATH`

### 问题：包安装失败

**解决方案**：
- 检查网络连接
- 检查包名是否正确
- 查看日志输出

### 问题：JSON 解析失败

**解决方案**：
- 确保 Python 代码输出 JSON 格式
- 检查 JSON 是否在输出的最后
- 使用 `Execute()` 查看原始输出

---

**最后更新**：2025-01-XX
