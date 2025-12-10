# Agent-Server

Agent-Server 是一个代码生成服务，作为代理/网关层，负责调用下游智能体、管理知识库、调用 LLM 生成代码，并构建和部署应用。

## 架构设计

Agent-Server 作为代理层，前端只调用 Agent-Server，不直接调用智能体：

```
前端 → Agent-Server → 智能体 → Agent-Server → 知识库 + LLM → 代码生成 → 构建部署
```

## 目录结构

```
core/agent-server/
├── cmd/app/main.go          # 入口文件
├── server/                  # 服务器和路由
│   ├── server.go
│   └── router.go
├── api/v1/                  # API 处理器（待实现）
├── model/                   # 数据模型
│   ├── init.go
│   ├── agent.go
│   ├── knowledge.go
│   ├── llm.go
│   ├── code_gen.go
│   └── task.go
├── repository/             # 数据访问层（待实现）
├── service/                # 业务逻辑层（待实现）
└── docs/                   # Swagger 文档
    └── docs.go
```

## 配置

配置文件位于 `configs/agent-server.yaml`：

```yaml
server:
  port: 9092
  log_level: "info"
  debug: false

# 数据库配置（共用 app-server 的数据库）
db:
  type: "mysql"
  host: "127.0.0.1"
  port: 3306
  user: "app"
  password: "app"
  name: "app_db"  # 共用 app-server 的数据库

agent:
  timeout: 30
  retry:
    max_attempts: 3
    backoff: "exponential"
```

**注意**：
- LLM 配置不在配置文件中，后续会有专门的 LLM 管理功能
- 数据库共用 app-server 的数据库（`app_db`）

## 开发规范

### 1. Model 规范
- 所有 model 必须使用 `models.Base`（包含 ID, CreatedAt, UpdatedAt, CreatedBy, UpdatedBy 等）
- 时间字段使用 `models.Time` 类型

### 2. 路由规范
- **不使用 RESTful 风格**，参数放在 query 和 body 里
- 路由示例：`GET /api/v1/agent/list`, `POST /api/v1/agent/create`
- 不使用 `/:id` 这种路径参数

### 3. DTO 规范
- 每个接口都要定义 DTO 结构体（放在 `dto/` 目录）
- 请求 DTO：`XxxReq`
- 响应 DTO：`XxxResp`

### 4. 响应规范
- 使用统一的 `response` 包返回
- 使用 `response.OkWithData()`, `response.FailWithMessage()` 等

### 5. 中间件规范
- 使用 `middleware.WithUserInfo()` 获取用户信息
- 用户信息从请求头获取（X-Request-User, X-User 等）

## 运行

```bash
# 编译
go build -o bin/agent-server ./core/agent-server/cmd/app

# 运行
./bin/agent-server
```

## 智能体类型

智能体分为两种类型：

1. **纯知识库类型（knowledge_only）**
   - 只需要用户调用然后查询对应知识库直接生成代码即可
   - 不需要配置接口地址（NATS 主题由后端自动生成）
   - 必须关联知识库（KnowledgeBaseID）

2. **插件调用类型（plugin）**
   - 配置有消息主题（MsgSubject），需要调用外部插件处理
   - 插件处理完后再调用知识库生成代码
   - 必须配置消息主题和知识库

## 开发状态

当前框架已搭建完成，包含：

- ✅ 基础目录结构
- ✅ 配置文件（共用 app 数据库，移除 LLM 配置）
- ✅ 数据模型（使用 models.Base 和 models.Time）
  - ✅ Agent 模型（支持两种类型：纯知识库类型、接口调用类型）
  - ✅ KnowledgeBase 模型
  - ✅ LLMConfig 模型
  - ✅ CodeGenConfig 模型
  - ✅ Task 模型
- ✅ 服务器和路由框架
- ✅ 统一的 response 和中间件

待实现功能：

- ⏳ Repository 层（数据访问）
- ⏳ Service 层（业务逻辑）
- ⏳ API 层（HTTP 处理器，定义 DTO）
  - ⏳ 智能体管理 API
  - ⏳ 知识库管理 API
  - ⏳ LLM 管理 API
- ⏳ 智能体客户端（代理调用智能体，区分两种类型）
- ⏳ 代码生成逻辑
- ⏳ 应用构建逻辑

## 下一步

1. 先实现一个简单的 API 接口，验证框架流程
2. 实现 Repository 层
3. 实现 Service 层
4. 实现 API 层（定义 DTO）
5. 逐步完善功能

