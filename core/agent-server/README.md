# Agent Server

Agent Server 是工作台对话与工具编排服务。它接收前端工作台请求，加载本地内嵌 prompt，调用 LLM，并通过 ToolRegistry 执行内置工具来读写代码、生成 PRD、运行表单/表格/图表以及构建工作空间。

## 当前职责

- 工作台会话、消息、阶段交接与运行状态管理
- LLM 配置读取与 Chat/Tool streaming 编排
- ToolRegistry 内置工具注册与调用
- PRD contract 校验、角色提示词与样例 catalog 管理
- 工作空间上下文、文件、目录与运行工具的服务端封装

## 不再包含

- 旧 Agent/Plugin 管理模型与 API
- workspace mode 数据表与配置管理链路
- function generation 的异步回调尾巴
- PRD 顶层 workflow 字段

这些概念已经从主链路移除；历史表名如 `agent_chat_sessions`、`agent_chat_messages` 暂时保留，用于兼容既有数据库。

## 主要目录

```text
core/agent-server/
├── cmd/app/                 # 服务入口
├── api/v1/                  # HTTP handler
├── model/                   # GORM model
├── repository/              # 数据访问
├── service/                 # 工作台会话、工具、LLM 编排
├── prompt/                  # 本地内嵌系统提示词
├── streamloop/              # LLM tool calling 循环
├── workspace/               # PRD contract、角色与工作区领域定义
└── utils/                   # 小型通用工具
```

## 配置

配置文件位于 `deploy/dev/config/agent-server.yaml` 或 `deploy/prod/config/{runtime|template}/agent-server.yaml`，由 `APP_ENV` 决定，默认 `prod`。

Agent Server 复用 app-server 数据库。LLM provider/model/key 等运行时配置从数据库读取，不放在服务配置文件中。

## 运行

```bash
go build -o bin/agent-server ./core/agent-server/cmd/app
./bin/agent-server
```

## 验证

```bash
go test ./core/agent-server/...
```
