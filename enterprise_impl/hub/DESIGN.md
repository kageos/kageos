# Hub 设计方案

## 📋 需求分析

### 核心需求

1. **应用发布到Hub**
   - 场景A：主服务（SaaS）用户 - 有用户身份，直接发布
   - 场景B：私有化部署用户 - 无用户身份，需要替代方案

2. **应用试用和克隆**
   - Hub → OS 跳转试用：用户在 Hub 浏览应用，点击"试用"跳转到 OS 试用
   - Hub → OS 跳转克隆：用户在 Hub 浏览应用，点击"克隆"跳转到 OS 克隆
   - 类似 Git clone，填写 URL 即可自动克隆应用

3. **服务费模式**
   - 从"克隆费"到"服务费"：强调后续服务，而不是软件本身
   - 代码完全开源：代码完全开源，增强信任，促进付费
   - 构造记录完全开源：构造记录完全开源，增强信任，促进付费

---

## 🎯 方案设计

### 1. 应用发布方案

#### 场景A：主服务（SaaS）用户发布

**流程：**
```
用户在主系统选择应用
  ↓
点击"发布到Hub"按钮
  ↓
填写应用信息（名称、描述、分类、标签）
  ↓
主系统调用 Hub API 发布
  ↓
Hub 存储应用元数据（不存储代码）
  ↓
返回 Hub 应用 URL
```

**技术实现：**
- 主系统 → Hub：通过 REST API 发布
- 认证：使用 JWT Token（用户已登录）
- 数据：只存储元数据，代码引用（user/app/package）

**API 设计：**
```go
POST /api/v1/apps/publish
Authorization: Bearer {jwt_token}
{
  "source_user": "user1",
  "source_app": "my_app",
  "packages": ["crm/ticket", "crm/customer"],
  "name": "CRM管理系统",
  "description": "...",
  "category": "CRM",
  "tags": ["crm", "business"]
}
```

#### 场景B：私有化部署用户发布

**问题：**
- 没有用户身份
- 无法使用 JWT 认证
- 需要替代认证方案

**解决方案：API Key 认证**

**流程：**
```
用户在私有化部署中生成 API Key
  ↓
在 Hub 网站注册并绑定 API Key
  ↓
发布时使用 API Key 认证
  ↓
Hub 验证 API Key 并发布
```

**技术实现：**
- 认证方式：API Key（类似 GitHub Personal Access Token）
- 生成位置：私有化部署的管理后台
- 绑定方式：在 Hub 网站输入 API Key 绑定

**API 设计：**
```go
POST /api/v1/apps/publish
X-API-Key: {api_key}
{
  "source_url": "https://private-deploy.example.com",
  "source_user": "user1",
  "source_app": "my_app",
  "packages": ["crm/ticket"],
  "name": "CRM管理系统",
  "description": "...",
  "category": "CRM"
}
```

**API Key 生成（主系统侧）：**
```go
// 在主系统管理后台生成 API Key
POST /api/v1/admin/api-keys
Authorization: Bearer {admin_token}
{
  "name": "Hub发布密钥",
  "expires_at": "2025-12-31T23:59:59Z"
}

// 返回
{
  "api_key": "hub_xxxxx...",
  "created_at": "..."
}
```

**API Key 验证（Hub侧）：**
```go
// Hub 验证 API Key
func (s *AppService) ValidateAPIKey(apiKey string) (*APIKeyInfo, error) {
    // 1. 查询 API Key
    // 2. 验证是否过期
    // 3. 返回关联的用户/组织信息
}
```

---

### 2. 应用克隆方案

#### URL 格式设计

**Hub 应用 URL：**
```
https://hub.ai-agent-os.com/apps/{app_id}
或
hub://{app_id}
```

**示例：**
```
https://hub.ai-agent-os.com/apps/123
hub://123
```

#### 克隆流程

**方案1：URL 输入（推荐）**
```
用户在主系统输入 Hub URL
  ↓
主系统解析 URL，获取 app_id
  ↓
主系统调用 Hub API 获取应用信息
  ↓
Hub 返回应用元数据（source_user, source_app, packages）
  ↓
主系统调用 Fork API 克隆应用
  ↓
完成克隆，跳转到新应用
```

**方案2：Hub 网站一键克隆**
```
用户在 Hub 网站浏览应用
  ↓
点击"Clone"按钮
  ↓
选择目标应用（或创建新应用）
  ↓
Hub 调用主系统 API 克隆
  ↓
跳转到主系统新应用
```

#### 技术实现

**主系统侧：Clone API**
```go
POST /api/v1/hub/clone
Authorization: Bearer {jwt_token}
{
  "hub_app_id": "123",
  "target_user": "user1",
  "target_app": "my_app"
}
```

**Hub 侧：获取应用信息 API**
```go
GET /api/v1/apps/{app_id}
{
  "id": "123",
  "name": "CRM管理系统",
  "source_user": "demo",
  "source_app": "crm_demo",
  "packages": [
    {
      "package": "crm/ticket",
      "full_group_code": "/demo/crm_demo/crm/ticket/ticket"
    }
  ]
}
```

**主系统侧：解析 Hub URL**
```go
func ParseHubURL(url string) (appID string, err error) {
    // 解析 https://hub.ai-agent-os.com/apps/123
    // 或 hub://123
    // 返回 app_id
}
```

---

## 🏗️ 架构设计

### 数据流

```
┌─────────────────┐         ┌─────────────────┐
│   主系统 (SaaS)  │◄───────►│   Hub (独立)     │
│                 │  API    │                 │
│  - 用户认证      │         │  - 应用市场      │
│  - 应用管理      │         │  - 应用元数据    │
│  - Fork服务      │         │  - API Key管理   │
└─────────────────┘         └─────────────────┘
         │                           │
         │                           │
         ▼                           ▼
┌─────────────────┐         ┌─────────────────┐
│  私有化部署      │         │   Hub 数据库     │
│                 │         │                 │
│  - API Key生成   │         │  - 应用元数据    │
│  - 应用管理      │         │  - 用户/组织     │
└─────────────────┘         │  - API Key       │
                            └─────────────────┘
```

### Hub 数据库设计

```sql
-- 应用表
CREATE TABLE hub_apps (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50),
    tags TEXT[],
    
    -- 源应用信息（主系统）
    source_type VARCHAR(20) NOT NULL, -- 'saas' | 'private'
    source_url VARCHAR(255),          -- 私有化部署的URL（仅private类型）
    source_user VARCHAR(100) NOT NULL,
    source_app VARCHAR(100) NOT NULL,
    
    -- 发布信息
    publisher_username VARCHAR(100),   -- 发布者用户名（OS 用户）
    api_key_id BIGINT,                 -- API Key ID（私有化用户）
    published_at TIMESTAMP,
    
    -- 服务费信息
    service_fee_personal DECIMAL(10,2),   -- 个人用户服务费
    service_fee_enterprise DECIMAL(10,2), -- 企业用户服务费
    
    -- 统计信息
    download_count INT DEFAULT 0,
    trial_count INT DEFAULT 0,
    rating DECIMAL(3,2),
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_category (category),
    INDEX idx_publisher_username (publisher_username),
    INDEX idx_published_at (published_at)
);

-- Hub 函数组表（存储源代码）
CREATE TABLE hub_function_groups (
    id BIGSERIAL PRIMARY KEY,
    hub_app_id BIGINT NOT NULL REFERENCES hub_apps(id) ON DELETE CASCADE,
    
    -- 函数组信息
    full_group_code VARCHAR(500) NOT NULL,  -- 完整函数组代码：/user/app/package/group_code
    group_code VARCHAR(255) NOT NULL,       -- 函数组代码：tools_cashier
    package_path VARCHAR(500) NOT NULL,     -- package 路径：plugins/cashier
    
    -- 🔥 源代码存储
    source_code TEXT NOT NULL,              -- Go 源代码内容
    source_code_hash VARCHAR(64),           -- 源代码 hash（用于去重和版本管理）
    
    -- 元数据
    function_count INT DEFAULT 0,            -- 函数数量
    api_count INT DEFAULT 0,                 -- API 数量
    
    -- 版本信息
    version VARCHAR(50),                     -- 版本号（对应主系统的 App.Version）
    published_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_hub_app_id (hub_app_id),
    INDEX idx_full_group_code (full_group_code),
    INDEX idx_source_code_hash (source_code_hash)
);

-- 构造记录表
CREATE TABLE code_generation_logs (
    id BIGSERIAL PRIMARY KEY,
    hub_app_id BIGINT NOT NULL REFERENCES hub_apps(id) ON DELETE CASCADE,
    publisher_username VARCHAR(100) NOT NULL,
    version VARCHAR(50) NOT NULL,
    
    -- 对话记录
    conversation JSONB NOT NULL,  -- 存储完整的对话记录
    
    -- 元数据
    total_turns INT DEFAULT 0,     -- 对话轮数
    total_tokens INT DEFAULT 0,   -- 总 token 数
    generation_time TIMESTAMP,      -- 生成时间
    
    -- 统计信息
    functions_generated INT DEFAULT 0,  -- 生成的函数数量
    apis_generated INT DEFAULT 0,       -- 生成的 API 数量
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_hub_app_id (hub_app_id),
    INDEX idx_publisher_username (publisher_username),
    INDEX idx_version (version)
);

-- 服务费支付记录表
CREATE TABLE service_fee_payments (
    id BIGSERIAL PRIMARY KEY,
    hub_app_id BIGINT NOT NULL REFERENCES hub_apps(id),
    buyer_username VARCHAR(100) NOT NULL,
    payment_type VARCHAR(20) NOT NULL, -- 'personal' | 'enterprise'
    amount DECIMAL(10,2) NOT NULL,
    
    -- 收益分配
    developer_username VARCHAR(100) NOT NULL,
    developer_amount DECIMAL(10,2) NOT NULL,  -- 开发者收益（80%）
    hub_amount DECIMAL(10,2) NOT NULL,        -- Hub 平台收益（10%）
    os_amount DECIMAL(10,2) NOT NULL,         -- OS 平台收益（10%）
    
    -- 支付信息
    payment_method VARCHAR(50),               -- 支付方式
    payment_status VARCHAR(20) NOT NULL,      -- 'pending' | 'paid' | 'failed'
    payment_time TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_hub_app_id (hub_app_id),
    INDEX idx_buyer_username (buyer_username),
    INDEX idx_developer_username (developer_username),
    INDEX idx_payment_status (payment_status)
);

-- API Key表
CREATE TABLE hub_api_keys (
    id BIGSERIAL PRIMARY KEY,
    key_hash VARCHAR(255) UNIQUE NOT NULL, -- 存储hash，不存储明文
    name VARCHAR(255),
    user_id BIGINT,                        -- Hub用户ID
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 用户表（Hub独立用户系统，或与主系统共享）
CREATE TABLE hub_users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE,
    email VARCHAR(255),
    -- ... 其他字段
);
```

---

## 🔐 安全设计

### 1. API Key 安全

- **存储**：只存储 hash，不存储明文
- **传输**：使用 HTTPS
- **权限**：API Key 只能发布应用，不能读取其他数据
- **过期**：支持设置过期时间

### 2. 私有化部署安全

- **验证**：Hub 需要验证 source_url 是否可访问
- **白名单**：可以设置允许的私有化部署域名
- **限流**：防止滥用

### 3. 克隆安全

- **权限检查**：克隆时检查用户是否有权限
- **数据隔离**：确保数据不会泄露

---

## 📝 实现步骤

### Phase 1：核心功能（MVP）

1. **数据库设计**
   - [ ] 创建 Hub 应用表
   - [ ] 创建 Hub 函数组表（存储源代码）
   - [ ] 创建构造记录表
   - [ ] 创建服务费支付记录表

2. **Hub 后端**
   - [ ] 应用发布 API（POST /api/v1/apps/publish）
   - [ ] 应用列表/详情 API（GET /api/v1/apps、GET /api/v1/apps/{app_id}）
   - [ ] 应用试用 API（GET /api/v1/apps/{app_id}/trial）
   - [ ] 应用克隆 API（POST /api/v1/apps/{app_id}/clone）
   - [ ] OS API 客户端（获取源代码、构造记录、克隆应用）

3. **OS 集成**
   - [ ] 获取源代码 API（GET /api/v1/apps/{user}/{app}/source-code）
   - [ ] 获取构造记录 API（GET /api/v1/apps/{user}/{app}/construction-log）
   - [ ] 从 Hub 克隆 API（POST /api/v1/hub/clone-from-hub）
   - [ ] 发布到Hub功能（UI + API调用）
   - [ ] 从Hub克隆功能（URL输入 + API调用）

4. **Hub 前端**
   - [ ] 应用浏览页面（搜索、筛选、分页）
   - [ ] 应用详情页（代码预览、构造记录摘要、服务费信息）
   - [ ] 试用和克隆功能（跳转到 OS）
   - [ ] 用户认证（调用 OS API）

### Phase 2：私有化支持

1. **API Key 系统**
   - [ ] API Key 生成（主系统）
   - [ ] API Key 验证（Hub）
   - [ ] API Key 管理（Hub）

2. **私有化发布**
   - [ ] 使用 API Key 发布
   - [ ] 私有化部署URL验证

### Phase 3：增强功能

1. **应用管理**
   - [ ] 应用版本管理
   - [ ] 应用更新通知

2. **用户体验**
   - [ ] 应用搜索和筛选
   - [ ] 应用评分和评论

---

## ✅ 方案可行性分析

### 优点

1. **灵活性强**
   - 支持 SaaS 和私有化两种场景
   - API Key 方案简单易用

2. **安全性好**
   - API Key 只存储 hash
   - 权限隔离清晰

3. **用户体验好**
   - URL 克隆方式直观
   - 类似 Git clone，用户熟悉

### 潜在问题

1. **私有化部署的网络访问**
   - 问题：Hub 需要访问私有化部署获取代码
   - 解决：不直接访问，通过主系统 API 获取

2. **API Key 管理**
   - 问题：用户需要管理多个 API Key
   - 解决：提供管理界面，支持撤销和过期

3. **代码同步**
   - 问题：私有化部署的代码更新，Hub 如何同步
   - 解决：不存储代码，只存储引用，实时获取

### 建议

1. **MVP阶段**：先实现 SaaS 用户发布和克隆
2. **后续迭代**：再实现私有化部署支持
3. **代码获取**：始终从主系统实时获取，不存储代码快照

---

## 🎯 总结

**方案可行性：✅ 高度可行**

- 技术实现简单
- 安全性可控
- 用户体验好
- 扩展性强

**建议实施顺序：**
1. SaaS 用户发布和克隆（MVP）
2. 私有化部署支持（Phase 2）
3. 增强功能（Phase 3）

