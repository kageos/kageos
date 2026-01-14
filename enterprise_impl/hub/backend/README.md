# Hub Backend

Hub 后端服务框架

## 项目结构

```
backend/
├── cmd/
│   └── app/
│       └── main.go          # 入口文件
├── server/
│   ├── server.go            # 服务器结构体
│   └── router.go            # 路由配置
├── api/
│   └── v1/                  # API 处理器
│       └── app.go
├── service/                 # 业务服务
│   └── app_service.go
├── repository/              # 数据访问层
│   └── hub_app_repository.go
├── model/                   # 数据模型
│   ├── init.go
│   └── hub_app.go
├── client/                  # 外部 API 客户端
│   └── os_client.go
├── config/                  # 配置文件示例
│   └── hub.yaml.example
├── go.mod
└── README.md
```

## 框架说明

### 1. 服务器初始化流程

```
main.go
  ↓
NewServer()
  ↓
initDatabase()    # 初始化数据库（PostgreSQL）
  ↓
initServices()    # 初始化服务（Repository、Service、Client）
  ↓
initRouter()      # 初始化路由
  ↓
Start()           # 启动服务器
```

### 2. 依赖注入

- **Repository**：数据访问层，在 `initServices` 中创建
- **Service**：业务逻辑层，依赖 Repository 和 Client
- **Handler**：API 处理器，依赖 Service

### 3. 架构层次

```
API Handler (api/v1/)
  ↓
Service (service/)
  ↓
Repository (repository/) + Client (client/)
  ↓
Model (model/) + External API
```

### 4. 待实现的功能

所有业务代码都标记了 `TODO`，需要根据设计文档实现：

- [ ] 数据库模型（model/）- 参考 DESIGN.md 中的数据库设计
- [ ] 数据访问层（repository/）- 实现 CRUD 操作
- [ ] 业务服务（service/）- 实现业务逻辑
- [ ] API 处理器（api/v1/）- 实现 HTTP 接口
- [ ] OS API 客户端（client/）- 调用 OS 平台 API

## 开发指南

### 1. 配置

1. 复制配置文件示例：
```bash
cp config/hub.yaml.example config/hub.yaml
```

2. 修改 `config/hub.yaml` 中的配置：
- 数据库连接信息
- OS 平台基础 URL
- 服务器端口等

3. 配置文件需要放在项目根目录或通过环境变量指定路径。

### 2. 数据库

使用 MySQL，需要在配置中设置数据库连接信息。

数据库表结构参考 `DEVELOPMENT_PLAN.md` 中的数据库设计。

### 3. 路由

路由在 `server/router.go` 中配置，目前只有健康检查路由。

所有业务路由都标记了 `TODO`，需要根据设计文档实现。

### 4. 中间件

使用 `pkg/middleware` 中的 JWT 认证中间件。

### 5. 日志

使用 `pkg/logger` 进行日志记录。

日志文件默认保存在 `./logs/hub-server.log`。

## 快速开始

### 1. 安装依赖

```bash
go mod download
```

### 2. 配置数据库

创建 MySQL 数据库：
```sql
CREATE DATABASE hub_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 3. 配置应用

复制并修改配置文件：
```bash
cp config/hub.yaml.example config/hub.yaml
# 编辑 config/hub.yaml
```

### 4. 运行

```bash
go run cmd/app/main.go
```

### 5. 健康检查

```bash
curl http://localhost:9094/health
```

## 下一步

1. **完善数据库模型**（参考 DESIGN.md 和 DEVELOPMENT_PLAN.md）
   - 实现 `model/hub_app.go` 等模型
   - 在 `model/init.go` 中添加表迁移

2. **实现 Repository 层**
   - 实现 `repository/hub_app_repository.go` 等
   - 实现 CRUD 操作

3. **实现 Service 层**
   - 实现 `service/app_service.go` 等
   - 实现业务逻辑

4. **实现 API 处理器**
   - 实现 `api/v1/app.go` 等
   - 实现 HTTP 接口

5. **实现 OS API 客户端**
   - 实现 `client/os_client.go`
   - 调用 OS 平台 API

## 注意事项

1. **配置管理**：需要在 `pkg/config` 中添加 `HubConfig` 支持（已完成）
2. **数据库迁移**：使用 GORM 的 AutoMigrate 功能
3. **错误处理**：统一使用 `pkg/ginx/response` 返回响应
4. **日志记录**：使用 `pkg/logger` 记录日志
5. **依赖注入**：所有依赖都通过构造函数注入
