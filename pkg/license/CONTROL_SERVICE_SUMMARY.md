# Control Service 设计方案总结

这份总结描述当前使用的 License 方案。

## 当前定位

Control Service 是一个 License 管理服务。

当前职责只有：

- 读取、验证、激活、停用 License
- 通过 NATS 提供 License 查询
- 通过 NATS 主动推送或刷新 License 状态

## 当前主题

- `control.v1.query.license-key.get`
  各服务实例主动请求最新 License
- `license.v1.event.key.updated`
  Control Service 主动推送最新 License
- `license.v1.event.key.refresh`
  Control Service 通知各服务重新拉取或停用 License

## 当前工作流

### 启动拉取

1. 服务启动时先读取本地 License 文件
2. 本地无可用数据时，请求 `control.v1.query.license-key.get`
3. Control Service 返回加密后的 License
4. 客户端解密并写回本地

### 激活推送

1. Control Service 激活新 License
2. 发布 `license.v1.event.key.updated`
3. 已订阅实例直接刷新内存和本地文件

### 更新或停用

1. Control Service 检测到 License 更新或停用
2. 发布 `license.v1.event.key.refresh`
3. 客户端收到后重新请求 `control.v1.query.license-key.get`
4. 返回空值时降级为社区版

## 边界说明

- 如果未来增加新的控制类主题，必须先按 [README.md](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/pkg/subjects/README.md) 定义

## 相关文档

- [Control Service 设计说明](./CONTROL_SERVICE_DESIGN.md)
- [Control Service 集成实现总结](./CONTROL_SERVICE_INTEGRATION.md)
