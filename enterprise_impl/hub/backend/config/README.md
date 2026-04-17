# Hub 配置文件

这个目录下的 [hub.yaml](hub.yaml) 可以作为配置结构参考，但 Hub 服务运行时实际走的是全局 `pkg/config` 配置加载逻辑。

## 实际加载路径

- `APP_ENV=dev`：`deploy/dev/config/hub.yaml`
- `APP_ENV!=dev`：`deploy/prod/config/runtime/hub.yaml`
- 如果上一步不存在：`deploy/prod/config/template/hub.yaml`

当前仓库已提供的开发配置文件在：

- [deploy/dev/config/hub.yaml](../../../../deploy/dev/config/hub.yaml)

## 字段说明

### server

- `port`: 服务端口，默认 `9094`
- `log_level`: 日志级别
- `debug`: 是否启用 Gin 调试模式

### db

- `type`: 当前实现使用 `mysql`
- `host`
- `port`
- `user`
- `password`
- `name`
- `log_level`
- `slow_threshold`

### public_host

- 用于生成 `copy_url` 的主站 `host:port`
- 未配置时会回退请求头中的 host 信息

### os.base_url

- 主站前端地址
- 用于 Hub 前端“试用”跳转
- 这里填的是前端入口，不是后端 API 地址

示例：

- 开发环境：`http://localhost:5173`
- 线上环境：实际主站前端地址
