# App Runtime Service

`core/app-runtime/service` 负责 runtime 进程内的应用生命周期、版本元数据、发现机制和 runtime -> app 控制调用。

## 当前文件分工

- `app_manage_service.go`
  应用生命周期主服务，负责创建、更新、启动、关闭、清理版本。
- `app_manage_waiters.go`
  启动确认和 update callback 等待器。
- `app_manage_version_files.go`
  维护 `workplace/metadata` 下的版本文件。
- `app_nats_clients.go`
  runtime 主动发出的 NATS 调用，包括 discovery 广播和 app control request-reply。
- `app_discovery_service.go`
  运行中应用注册表和 discovery 调度。
- `app_discovery_handler.go`
  解析生命周期事件并更新 discovery 状态。
- `app_discovery_transport.go`
  discovery 相关的订阅与广播 transport。
- `qps_tracker.go`
  追踪版本级流量，辅助旧版本安全关闭。
- `container_service.go`
  容器运行时适配。
- `create_function_service.go`
  新应用初始化时的函数脚手架生成。
- `service_tree_service.go`
  目录树相关服务能力。

## 版本元数据

应用版本元数据位于 `workplace/metadata`：

- `current_version.txt`
  当前生效版本，启动和查询时优先读取。
- `current_app.txt`
  当前应用二进制名前缀。
- `version.json`
  版本索引，记录当前版本、最新版本和历史版本状态。

## 关键链路

### 更新发布

`UpdateApp()` 主流程会：

1. 编译新版本产物到 `workplace/bin/releases`
2. 更新 `workplace/metadata` 下的版本文件
3. 启动新版本
4. 等待启动确认和 update callback
5. 更新数据库中的版本记录

### 应用发现

`AppDiscoveryService` 会：

1. 广播 `app.v1.cmd.discovery.request`
2. 监听 `runtime.v1.event.lifecycle.*.*.*`
3. 把 startup / close / discovery 响应收敛到内存中的运行状态表

`runtime_id` 来自 runtime 配置，未配置时回退为基于 hostname 的稳定值。

### 优雅关闭

`ShutdownAppVersion()` 会通过 `app.v1.cmd.control.{user}.{app}.{version}` 向 SDK app 发送 shutdown 控制消息。

SDK app 收到后会：

1. 拒绝新请求
2. 等待运行中的函数完成
3. 上报 close 生命周期事件
4. 退出进程

### 旧版本清理

清理逻辑只会关闭“非当前版本且无流量”的容器，避免直接切断仍在处理请求的旧版本。

## 入口关系

- `server/server.go`
  初始化数据库、NATS、业务服务和订阅。
- `server/nats_router.go`
  注册 runtime 侧 command/query 路由。
- `api/v1/*.go`
  NATS handler，负责解码请求并调用 service。

## 当前原则

- `pkg/subjects` 是主题真值
- `pkg/msgx` 提供基础 request-reply 原语
- service 负责业务，transport/handler 负责 NATS 适配
- 版本状态以 `workplace/metadata` 为准
