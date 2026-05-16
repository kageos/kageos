# License 系统说明

License 系统只负责当前已经落地的企业能力开关，避免 MVP 阶段暴露没有实现的功能位。

## 当前功能位

| 字段 | 说明 | 实现位置 |
|---|---|---|
| `operate_log` | Form / Table 操作日志查询能力 | `enterprise_impl/operatelog` |
| `permission` | 高级权限治理：角色、权限点、授权申请和审批 | `enterprise_impl/permission` |

未实现或默认隐藏的能力不再作为 License feature 暴露，后续真正形成稳定实现后再重新加入。

## License 文件格式

```json
{
  "license": {
    "id": "license-xxx",
    "edition": "enterprise",
    "issued_at": "2025-01-01 00:00:00",
    "expires_at": "2026-01-01 00:00:00",
    "customer": "Company Name",
    "max_apps": 100,
    "max_users": 50,
    "features": {
      "operate_log": true,
      "permission": true
    },
    "hardware_id": "optional-hardware-binding"
  },
  "signature": "RSA签名（Base64编码）"
}
```

## 设计原则

- 没有 License 文件时自动使用社区版。
- License 验证失败时降级为社区版，不中断主链路启动。
- 新增企业能力前，必须先有代码实现、路由或服务边界，再加入 `Features`。
- MVP 阶段优先保证个人用户和小企业能完成 AI 创建、修改、构建、运行轻应用的主链路。

## 使用方式

```go
licenseMgr := license.GetManager()

if licenseMgr.HasFeature("operate_log") {
    // 查询企业操作日志
}
```

企业实现通过 `enterprise.RegisterOperateLogger`、`enterprise.RegisterPermissionService` 注册。开源/社区默认实现保留在 `enterprise/` 包中，企业实现位于 `enterprise_impl/`。
