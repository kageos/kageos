# License 使用指南

## 社区版

不需要 License 文件。系统没有检测到 License 时，默认使用社区能力，主链路不应被 License 阻断。

## 企业版

将签发后的 `license.json` 放到以下任一位置：

1. `LICENSE_PATH` 指定的路径
2. 当前目录 `./license.json`
3. 用户目录 `~/.ai-agent-os/license.json`

License 会在服务启动时加载和验证。验证失败时降级到社区版。

## 示例

```json
{
  "license": {
    "id": "license-xxx",
    "edition": "enterprise",
    "issued_at": "2025-01-01 00:00:00",
    "expires_at": "2026-01-01 00:00:00",
    "customer": "Your Company Name",
    "max_apps": 100,
    "max_users": 50,
    "features": {
      "operate_log": true,
      "permission": true
    }
  },
  "signature": "RSA签名（Base64编码）"
}
```

## 当前支持的功能

| 功能位 | 说明 |
|---|---|
| `operate_log` | 查询 Form / Table 操作日志 |
| `permission` | 高级权限治理，包含角色、权限点、授权申请和审批 |

不要在 License 中加入尚未实现的功能位。当前 MVP License 模型只接受上表列出的能力。
