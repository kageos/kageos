# Control Service

Control Service 负责 License 管理。

## 当前职责

- 读取、验证、激活、停用 License
- 通过 HTTP 提供 License 管理接口
- 通过 NATS 为各服务实例提供 License 查询、推送和刷新能力

## 当前 NATS 主题

- `control.v1.query.license-key.get`
  各服务启动或刷新时，向 Control Service 请求最新 License
- `license.v1.event.key.updated`
  Control Service 主动推送最新 License
- `license.v1.event.key.refresh`
  Control Service 通知各服务重新拉取或停用 License

主题命名规范见 [README.md](../../pkg/subjects/README.md)。

## HTTP API

当前实际暴露的接口如下：

- `GET /control/api/v1/license/status`
- `POST /control/api/v1/license/activate`
- `POST /control/api/v1/license/deactivate`

## 启动

### 环境变量

```bash
export LICENSE_PATH=./license.json
export NATS_URL=nats://127.0.0.1:4222
export LICENSE_ENCRYPTION_KEY=your-32-byte-encryption-key-here!!
```

### 运行

```bash
cd core/control-service
go run cmd/app/main.go
```

## 工作流

### 启动拉取

1. 服务实例先尝试加载本地 License
2. 本地没有可用 License 时，通过 `control.v1.query.license-key.get` 请求
3. Control Service 返回加密后的 License
4. 客户端解密并写入本地文件

### 激活后推送

1. Control Service 激活新 License
2. 发布 `license.v1.event.key.updated`
3. 各服务收到后刷新内存和本地文件

### 刷新或停用

1. Control Service 检测到 License 更新或停用
2. 发布 `license.v1.event.key.refresh`
3. 客户端重新请求 `control.v1.query.license-key.get`
4. 若返回空值，则降级到社区版

## 说明

- 如果未来引入新的控制类主题，必须先更新 `pkg/subjects`
