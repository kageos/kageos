# Hub 配置说明

## 概述

Hub 是 AI-Agent-OS 的应用市场，提供应用发布、浏览、克隆等功能。

## 配置方式

### 1. 环境变量配置（推荐）

在项目根目录创建 `.env` 文件（参考 `.env.example`）：

```bash
# Hub API 基础地址
# 开发环境：默认通过网关代理 /hub（无需配置）
# 生产环境：默认使用官方地址 https://www.ai-agent-os.com/hub
# 自定义：可以设置为自己的 Hub 实例地址
VITE_HUB_BASE_URL=

# 是否启用 Hub 功能（默认 true）
# 设置为 false 可禁用 Hub 功能，相关 UI 将自动隐藏
VITE_HUB_ENABLED=true
```

### 2. 默认配置

**开发环境**：
- `baseURL`: `/hub`（通过 Vite 代理到网关，网关转发到 Hub 服务）
- `enabled`: `true`

**生产环境**：
- `baseURL`: `https://www.ai-agent-os.com/hub`（官方 Hub）
- `enabled`: `true`

### 3. 使用示例

```typescript
import { getHubBaseURL, isHubEnabled, HUB_CONFIG } from '@/config/hub'
import { publishApp } from '@/api/hub'

// 检查 Hub 是否启用
if (isHubEnabled()) {
  // 使用 Hub 功能
  const baseURL = getHubBaseURL()
  console.log('Hub URL:', baseURL)
  
  // 发布应用
  await publishApp({
    api_key: 'your-api-key',
    source_user: 'user1',
    source_app: 'my_app',
    name: 'My App',
    // ...
  })
}
```

## 网关配置

Hub API 通过网关代理，需要在 `api-gateway.yaml` 中配置：

```yaml
routes:
  - path: "/hub"
    service_name: "hub"
    targets:
      - url: "http://localhost:9094"  # hub-server
    timeout: 30
```

## 开发环境

开发环境使用本地地址，通过网关代理：

1. **前端请求**：`/hub/api/v1/apps/publish`
2. **Vite 代理**：转发到 `http://localhost:9090/hub/api/v1/apps/publish`（网关）
3. **网关转发**：转发到 `http://localhost:9094/api/v1/apps/publish`（Hub 服务）

## 生产环境

生产环境可以：

1. **使用官方 Hub**（默认）：
   ```bash
   # 无需配置，使用默认值
   ```

2. **使用自己的 Hub 实例**：
   ```bash
   VITE_HUB_BASE_URL=https://my-hub.example.com
   ```

3. **禁用 Hub 功能**：
   ```bash
   VITE_HUB_ENABLED=false
   ```

## 注意事项

1. **开发环境**：使用相对路径 `/hub`，通过 Vite 代理
2. **生产环境**：使用绝对路径，直接访问 Hub 服务
3. **网关代理**：确保网关已配置 Hub 路由
4. **功能开关**：可以通过 `VITE_HUB_ENABLED` 禁用 Hub 功能

