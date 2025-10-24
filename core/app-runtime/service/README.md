# App Runtime Service 代码结构

## 文件组织

为了提高代码可读性和可维护性，我们将 `AppManageService` 拆分成多个文件，按功能模块组织：

```
service/
├── app_manage_service.go      # 主服务：结构定义、创建/删除应用
├── app_manage_build.go         # 编译模块：编译新版本、更新元数据
├── app_manage_startup.go       # 启动模块：启动通知、等待器、启动应用
├── app_manage_cleanup.go       # 清理模块：定时巡检、清理旧版本
├── app_manage_shutdown.go      # 关闭模块：关闭命令、状态更新
├── qps_tracker.go             # QPS 追踪
├── container_service.go       # 容器服务
└── app_discovery_service.go   # 应用发现服务
```

## 各文件职责

### 1. app_manage_service.go（主服务）
**职责**：服务结构定义、基础CRUD操作

**主要内容**：
- `AppManageService` 结构定义
- `NewAppManageService()` - 服务初始化
- `CreateApp()` - 创建应用
- `DeleteApp()` - 删除应用
- `initAppDirectory()` - 初始化目录结构
- `createAndStartContainer()` - 创建容器
- `getCurrentVersion()` - 获取当前版本
- `getAllApps()` - 获取所有应用

**特点**：
- ✅ 依赖注入（Builder, Config, ContainerService, Repository, NATS）
- ✅ 完整的应用生命周期管理

### 2. app_manage_build.go（编译模块）
**职责**：应用编译和版本管理

**主要内容**：
- `UpdateApp()` - 更新应用（主流程）
- `buildNewVersion()` - 编译新版本
- `updateMetadata()` - 更新元数据文件
- `updateAppVersionRecord()` - 更新数据库记录

**编译流程**：
```
UpdateApp()
  ├─ buildNewVersion()      # 编译新版本
  ├─ updateMetadata()        # 更新 current_version.txt
  ├─ StartAppVersion()       # 启动新版本
  └─ updateAppVersionRecord() # 更新数据库
```

**元数据文件**：
- `current_version.txt` - 当前版本（纯文本，极速读取）
- `current_app.txt` - 二进制前缀
- `version.json` - 兼容性保留

### 3. app_manage_startup.go（启动模块）
**职责**：应用启动和启动通知管理

**主要内容**：
- `StartAppVersion()` - 启动指定版本
- `NotifyStartup()` - 接收启动通知
- `RegisterStartupWaiter()` - 注册等待器
- `UnregisterStartupWaiter()` - 注销等待器
- `GetStartupWaiter()` - 获取等待器
- `waitForStartup()` - 等待启动完成

**启动流程**：
```
StartAppVersion()
  ├─ 读取 current_app.txt
  ├─ 钻进容器执行启动命令（setsid nohup）
  ├─ 注册启动等待器
  └─ 等待 NATS 启动通知（30秒超时）
```

**特点**：
- ✅ 基于 NATS 事件的启动确认
- ✅ 超时保护（30秒）
- ✅ 自动清理等待器

### 4. app_manage_cleanup.go（清理模块/巡检）
**职责**：定时巡检和清理旧版本

**主要内容**：
- `StartCleanupTask()` - 启动定时任务
- `StopCleanupTask()` - 停止定时任务
- `performCleanup()` - 执行清理
- `CleanupNonCurrentVersions()` - 清理非当前版本

**巡检流程**：
```
每 30 秒执行一次
  ├─ 获取所有应用
  ├─ 遍历每个应用
  │   ├─ 读取 current_version.txt
  │   ├─ 获取所有运行中版本
  │   └─ 关闭非当前版本且无流量的版本
  └─ 记录日志
```

**清理策略**：
- ✅ 只保留 `current_version`
- ✅ QPS 为 0 才关闭
- ✅ 支持回滚（current_version 可能不是最新版本）

### 5. app_manage_shutdown.go（关闭模块）
**职责**：应用关闭和状态更新

**主要内容**：
- `ShutdownAppVersion()` - 关闭指定版本
- `UpdateAppStatus()` - 更新应用状态
- `ShutdownOldVersions()` - 已废弃（保留兼容性）

**关闭流程**：
```
ShutdownAppVersion()
  ├─ 构建关闭消息
  ├─ 发送 NATS 命令到 runtime.app.shutdown.*.*.*
  └─ SDK App 收到命令
      ├─ 拒绝新请求
      ├─ 等待运行中函数完成
      ├─ 发送关闭通知
      └─ 退出进程
```

**特点**：
- ✅ 优雅关闭
- ✅ 等待运行中函数
- ✅ NATS 双向通信

### 6. qps_tracker.go（QPS 追踪）
**职责**：追踪每个应用版本的 QPS

**主要内容**：
- `QPSTracker` - QPS 追踪器
- `RecordRequest()` - 记录请求
- `GetQPS()` - 获取 QPS
- `IsSafeToShutdown()` - 检查是否可以安全关闭
- `StartCleanup()` - 清理旧数据

**特点**：
- ✅ 滑动窗口（60秒）
- ✅ 自动清理
- ✅ 线程安全

## 调用关系

```
Server (handlers.go)
  ↓
AppManageService
  ├─ CreateApp() → createAndStartContainer()
  ├─ DeleteApp() → StopContainer()
  ├─ UpdateApp() → buildNewVersion() → StartAppVersion()
  │                                    → waitForStartup()
  ↓
CleanupTask (每30秒)
  └─ CleanupNonCurrentVersions() → ShutdownAppVersion()
                                  → UpdateAppStatus()
```

## 优势

1. **职责单一**：每个文件只负责一个功能模块
2. **易于维护**：修改编译逻辑只需要看 `app_manage_build.go`
3. **易于测试**：可以单独测试每个模块
4. **代码清晰**：不再有1000+行的巨型文件
5. **便于协作**：不同开发者可以同时修改不同文件

## 注意事项

1. **所有文件都在同一个 package**：`package service`
2. **共享结构体**：`AppManageService` 结构定义在主文件中
3. **方法接收者**：所有方法都是 `(s *AppManageService)` 的方法
4. **导入路径相同**：可以直接调用其他文件中的方法
5. **编译单元**：Go 会自动将同一 package 下的所有文件编译在一起

## 未来扩展

如果需要添加新功能，可以继续拆分：

- `app_manage_rollback.go` - 回滚功能
- `app_manage_health.go` - 健康检查
- `app_manage_metrics.go` - 指标收集
- `app_manage_backup.go` - 备份恢复

## 最佳实践

1. **单一职责**：每个文件只负责一个功能领域
2. **命名规范**：文件名使用 `app_manage_<功能>.go`
3. **注释完整**：每个导出函数都有注释说明
4. **错误处理**：统一的错误返回格式
5. **日志记录**：关键步骤都有详细日志
