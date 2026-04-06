# Hub Backend

Hub 后端服务，负责目录发布、版本导出、收藏和发布密钥管理。

## 当前结构

```
backend/
├── cmd/app/main.go
├── server/
├── api/v1/
├── service/
├── repository/
├── model/
├── dto/
├── docs/
├── config/
└── README.md
```

当前 `api/v1` 实际包含：

- `config.go`
- `directory.go`
- `pub_key.go`

当前 `service` 实际包含：

- `directory_service.go`
- `directory_bundle.go`
- `directory_tree_validation.go`
- `pub_key_service.go`

## 启动流程

`cmd/app/main.go` 会完成这几步：

1. 读取 `HubConfig`
2. 初始化 `pkg/logger`
3. 创建 `server.Server`
4. 初始化 MySQL / GORM
5. 初始化 repository 和 service
6. 初始化 Gin 路由
7. 启动 HTTP 服务

数据库当前使用 MySQL，不是 PostgreSQL。

## 已注册路由

基础路由：

- `GET /health`
- `GET /swagger/*any`

Hub API 前缀：

- `GET /hub/api/v1/config`
- `GET /hub/api/v1/directories`
- `GET /hub/api/v1/directories/detail`
- `GET /hub/api/v1/directories/export_bundle`
- `GET /hub/api/v1/directories/versions`
- `POST /hub/api/v1/directories/increment_download`
- `POST /hub/api/v1/directories/publish`
- `PUT /hub/api/v1/directories/update`
- `POST /hub/api/v1/directories/:id/star`
- `DELETE /hub/api/v1/directories/:id/star`
- `DELETE /hub/api/v1/directories/:id`
- `POST /hub/api/v1/pub_key/generate`
- `GET /hub/api/v1/pub_key/list`
- `DELETE /hub/api/v1/pub_key/:id`

鉴权规则：

- 公开接口：`config`、目录列表/详情/导出/版本、下载计数
- 发布接口：支持 `JWT` 或 `Pub Key`
- 收藏、删除目录、管理 `Pub Key`：要求 `JWT`

## 配置来源

运行时配置不是从当前目录直接读取，而是走全局 `pkg/config` 加载逻辑：

- `APP_ENV=dev` 时优先读取 `deploy/dev/config/hub.yaml`
- `APP_ENV!=dev` 时会尝试 `deploy/prod/config/runtime/hub.yaml`
- 然后回退 `deploy/prod/config/template/hub.yaml`

当前仓库内已经存在的开发配置文件是：

- [deploy/dev/config/hub.yaml](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/deploy/dev/config/hub.yaml)

当前目录下的 [hub.yaml](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/enterprise_impl/hub/backend/config/hub.yaml) 更适合作为结构参考，不是主运行入口。

关键配置项：

- `server.port`: Hub 服务端口，默认 `9094`
- `server.log_level`: 日志级别
- `server.debug`: Gin 调试模式
- `db.*`: MySQL 连接配置
- `public_host`: 生成 `copy_url` 时优先使用的主站 `host:port`
- `os.base_url`: Hub 前端“试用”跳转使用的主站前端地址

`public_host` 和 `os.base_url` 不是一回事：

- `public_host` 面向复制/分享 URL 生成
- `os.base_url` 面向浏览器跳转

## 本地运行

在仓库根目录准备好 `deploy/dev/config/hub.yaml` 和 MySQL 后，从 backend 目录运行：

```bash
go run cmd/app/main.go
```

默认健康检查：

```bash
curl http://localhost:9094/health
```

Swagger：

```bash
http://localhost:9094/swagger/index.html
```

## 备注

- 当前 README 描述的是仓库现状，不再保留“所有业务均 TODO”那类旧状态说明。
- 表结构初始化由 `model.InitTables` 在启动时执行。
- 路由和服务以 `server/router.go`、`server/server.go` 为准。
