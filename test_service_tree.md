# 服务目录功能测试指南

## 功能概述

服务目录功能允许为每个应用创建树形结构的服务目录，类似于文件夹的概念。每个服务目录会在应用的代码目录下创建对应的Go包，并生成`init_.go`文件。

## API接口

### 1. 创建服务目录

**POST** `/api/v1/service_tree`

```json
{
  "user": "beiluo",
  "app": "test1", 
  "title": "CRM管理系统",
  "name": "crm",
  "parent_id": 0,
  "description": "客户关系管理系统",
  "tags": "crm,customer"
}
```

**响应示例：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": 1,
    "title": "CRM管理系统",
    "name": "crm",
    "parent_id": 0,
    "type": "package",
    "description": "客户关系管理系统",
    "tags": "crm,customer",
    "app_id": 1,
    "full_id_path": "/1",
    "full_name_path": "/crm",
    "status": "created"
  }
}
```

### 2. 获取服务目录树

**GET** `/api/v1/service_tree?user=beiluo&app=test1`

**响应示例：**
```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "id": 1,
      "title": "CRM管理系统",
      "name": "crm",
      "parent_id": 0,
      "type": "package",
      "description": "客户关系管理系统",
      "tags": "crm,customer",
      "app_id": 1,
      "full_id_path": "/1",
      "full_name_path": "/crm",
      "children": [
        {
          "id": 2,
          "title": "工单管理",
          "name": "ticket",
          "parent_id": 1,
          "type": "package",
          "description": "工单管理系统",
          "tags": "ticket,support",
          "app_id": 1,
          "full_id_path": "/1/2",
          "full_name_path": "/crm/ticket",
          "children": []
        }
      ]
    }
  ]
}
```

### 3. 更新服务目录

**PUT** `/api/v1/service_tree`

```json
{
  "id": 1,
  "title": "CRM管理系统(更新)",
  "name": "crm",
  "description": "客户关系管理系统 - 更新版本"
}
```

**响应示例：**
```json
{
  "code": 200,
  "message": "更新成功"
}
```

### 4. 删除服务目录

**DELETE** `/api/v1/service_tree`

```json
{
  "id": 1
}
```

**响应示例：**
```json
{
  "code": 200,
  "message": "删除成功"
}
```

## 目录结构生成

当创建服务目录时，系统会在以下位置创建目录和文件：

```
namespace/beiluo/test1/
├── code/
│   └── api/
│       └── crm/           # 服务目录名称
│           └── init_.go   # 自动生成的初始化文件
```

## init_.go 文件内容

```go
package crm

import "fmt"

const (
	RouterGroup = "/crm"
)

func WithCurrentRouterGroup(router string) string {
	return fmt.Sprintf("%s/%s", RouterGroup, router)
}
```

## 多级目录支持

支持创建多级服务目录，例如：

1. 创建一级目录 `crm`
2. 在 `crm` 下创建二级目录 `ticket`
3. 在 `ticket` 下创建三级目录 `support`

生成的目录结构：
```
namespace/beiluo/test1/
├── code/
│   └── api/
│       └── crm/
│           ├── init_.go
│           └── ticket/
│               ├── init_.go
│               └── support/
│                   └── init_.go
```

## 测试步骤

1. **创建根级服务目录**
   ```bash
   curl -X POST http://localhost:8080/api/v1/service_tree \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer <token>" \
     -d '{
       "user": "beiluo",
       "app": "test1",
       "title": "CRM管理系统", 
       "name": "crm",
       "parent_id": 0,
       "description": "客户关系管理系统"
     }'
   ```

2. **创建子级服务目录**
   ```bash
   curl -X POST http://localhost:8080/api/v1/service_tree \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer <token>" \
     -d '{
       "user": "beiluo",
       "app": "test1",
       "title": "工单管理",
       "name": "ticket", 
       "parent_id": 1,
       "description": "工单管理系统"
     }'
   ```

3. **查看服务目录树**
   ```bash
   curl -X GET "http://localhost:8080/api/v1/service_tree?user=beiluo&app=test1" \
     -H "Authorization: Bearer <token>"
   ```

4. **验证目录结构**
   检查 `namespace/beiluo/test1/code/api/` 目录下是否生成了对应的目录和文件。

## 消息流程

### 1. 创建服务目录的完整流程（同步模式）

```
1. 用户调用 API: POST /api/v1/service-tree
2. ServiceTreeService.CreateServiceTree()
   ├── 验证参数和权限
   ├── 创建 ServiceTree 数据库记录
   └── 调用 sendCreateServiceTreeMessage()
3. ServiceTreeService.sendCreateServiceTreeMessage()
   ├── 获取应用信息（HostID）
   ├── 构建 CreateServiceTreeRuntimeReq
   └── 调用 appRuntime.CreateServiceTree()
4. AppRuntime.CreateServiceTree()
   ├── 获取 NATS 连接（根据 HostID）
   ├── 发送 NATS 消息到 app-runtime
   └── 等待同步响应
5. app-runtime 接收消息
   ├── handleServiceTreeCreate()
   ├── ServiceTreeService.CreateServiceTree()
   ├── 创建目录结构
   ├── 生成 init_.go 文件
   └── 返回 CreateServiceTreeRuntimeResp
6. app-server 收到响应
   ├── 解析响应结果
   └── 返回给用户
```

### 2. NATS 消息主题

- **请求主题**: `app_runtime.service_tree.create`
- **响应模式**: 同步请求-响应（使用 `msgx.RequestMsgWithTimeout`）
- **超时时间**: 配置的 NATS 请求超时时间

### 3. 目录结构生成

当创建服务目录时，系统会在以下位置创建目录和文件：

```
namespace/beiluo/test1/
├── code/
│   └── api/
│       └── crm/           # 服务目录名称
│           └── init_.go   # 自动生成的初始化文件
```

## 性能优化

### ServiceTreeService 单例模式
- **app-runtime** 在Server启动时初始化ServiceTreeService实例
- 避免每次处理请求时重复创建服务实例
- 提高性能，减少内存分配

```go
// Server结构体中包含ServiceTreeService
type Server struct {
    // ... 其他字段
    serviceTreeService *service.ServiceTreeService
}

// 在initServices中初始化
func (s *Server) initServices(ctx context.Context) error {
    // ... 其他初始化
    s.serviceTreeService = service.NewServiceTreeService(&s.cfg.AppManage)
    return nil
}

// 在handler中直接使用
func (s *Server) handleServiceTreeCreate(msg *nats.Msg) {
    resp, err := s.serviceTreeService.CreateServiceTree(ctx, &msgInfo.Data)
    // ...
}
```

## 注意事项

1. 服务目录名称必须唯一（在同一父目录下）
2. 删除服务目录会级联删除所有子目录
3. 目录结构会在app-runtime端自动生成
4. 无需重新编译应用，目录创建是实时的
5. 通过NATS消息实现app-server和app-runtime的通信
6. 支持多主机部署，根据应用的HostID选择对应的NATS连接
7. 使用单例模式优化性能，避免重复创建服务实例
