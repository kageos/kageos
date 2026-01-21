# Hub 配置文件

## 配置文件位置

配置文件应该放在 `hub/backend/config/hub.yaml`。

## 配置示例

参考 `hub.yaml.example` 文件。

## 快速开始

1. 复制配置文件示例：
```bash
cd hub/backend
cp config/hub.yaml.example config/hub.yaml
```

2. 修改 `config/hub.yaml` 中的配置：
- 数据库连接信息（MySQL）
- OS 平台基础 URL
- 服务器端口等

## 配置说明

### server
- `port`: 服务器端口（默认：9094）
- `log_level`: 日志级别（info, warn, error）
- `debug`: 是否调试模式

### db
- `type`: 数据库类型（mysql）
- `host`: 数据库主机
- `port`: 数据库端口（默认：3306）
- `user`: 数据库用户名
- `password`: 数据库密码
- `name`: 数据库名称
- `log_level`: 数据库日志级别（silent, error, warn, info）
- `slow_threshold`: 慢查询阈值（毫秒）

### os
- `base_url`: OS 平台基础 URL

