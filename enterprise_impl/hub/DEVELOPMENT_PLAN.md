# Hub 开发计划

## 一、项目概述

### 1.1 核心定位

**Hub + OS 一体化生态系统**：
- ✅ **Hub**：应用市场，卖软件、介绍软件
- ✅ **OS**：运行平台，可以试用、克隆、运行软件
- ✅ **互通**：Hub 和 OS 用户系统完全互通

### 1.2 商业模式

**服务费模式**：
- ✅ **从"克隆费"到"服务费"**：强调后续服务，而不是软件本身
- ✅ **代码完全开源**：代码完全开源，增强信任，促进付费
- ✅ **构造记录完全开源**：构造记录完全开源，增强信任，促进付费
- ✅ **开发者收益分成**：开发者 80%，Hub 平台 10%，OS 平台 10%

### 1.3 核心价值

**软件可以模仿，但服务无法模仿**：
- ✅ **技术支持**：有问题可以随时找开发者
- ✅ **需求调整**：可以帮你调整，例如新增字段（免费 3 个需求）
- ✅ **文档知识库**：免费文档知识库共享
- ✅ **版本升级**：可以同步升级新的版本，无缝升级
- ✅ **生态支持**：无法复用别人的生态
- ✅ **社区支持**：社区的活跃度

---

## 二、技术架构

### 2.1 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Hub + OS 一体化生态系统                    │
└─────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │   API 网关         │
                    │  (统一入口)       │
                    └─────────┬─────────┘
                              │
        ┌─────────────────────┴─────────────────────┐
        │                                           │
┌───────▼────────┐                      ┌───────────▼────────┐
│   Hub 平台     │                      │   OS 平台         │
│  (应用市场)    │                      │  (运行平台)       │
│                │                      │                   │
│ - 应用浏览     │                      │ - 应用运行         │
│ - 应用介绍     │                      │ - 应用试用         │
│ - 应用搜索     │                      │ - 应用克隆         │
│ - 服务费支付   │                      │ - 云托管           │
│                │                      │ - 私有化部署       │
└───────┬────────┘                      └───────────┬────────┘
        │                                           │
        └─────────────────────┬─────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │   OS (app-server) │
                    │                   │
                    │ - AuthService     │
                    │ - UserService     │
                    │ - 用户数据库      │
                    └───────────────────┘
```

### 2.2 用户系统互通

**统一用户系统**：
- ✅ **共享用户数据库**：Hub 和 OS 共享同一个用户数据库
- ✅ **统一认证**：使用统一的 JWT Token 认证
- ✅ **单点登录**：Hub 和 OS 之间单点登录（SSO）

**实现方式**：
- ✅ **Hub 调用 OS API**：Hub 通过 API 调用 app-server 的用户服务
- ✅ **OS 使用本地服务**：OS 直接使用 app-server 的用户服务

### 2.3 技术栈

**后端**：
- ✅ **语言**：Go
- ✅ **框架**：Gin
- ✅ **数据库**：PostgreSQL
- ✅ **缓存**：Redis（可选）

**前端**：
- ✅ **框架**：Vue 3
- ✅ **UI 库**：Element Plus
- ✅ **构建工具**：Vite

---

## 三、数据库设计

### 3.1 核心表设计

#### 1. Hub 应用表

```sql
-- Hub 应用表
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
    service_fee_personal DECIMAL(10,2), -- 个人用户服务费
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
```

#### 2. Hub 函数组表（存储源代码）

```sql
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
```

#### 3. 构造记录表

```sql
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
```

#### 4. 服务费支付记录表

```sql
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
```

#### 5. 服务记录表

```sql
-- 服务记录表（技术支持、需求调整等）
CREATE TABLE service_records (
    id BIGSERIAL PRIMARY KEY,
    hub_app_id BIGINT NOT NULL REFERENCES hub_apps(id),
    buyer_username VARCHAR(100) NOT NULL,
    service_type VARCHAR(50) NOT NULL, -- 'support' | 'requirement' | 'upgrade'
    service_content TEXT,
    service_status VARCHAR(20) NOT NULL, -- 'pending' | 'processing' | 'completed'
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    INDEX idx_hub_app_id (hub_app_id),
    INDEX idx_buyer_username (buyer_username),
    INDEX idx_service_status (service_status)
);
```

---

## 四、API 设计

### 4.1 应用发布 API

#### 1. 发布应用到 Hub

```go
// POST /api/v1/apps/publish
type PublishAppReq struct {
    SourceUser  string   `json:"source_user" binding:"required"`   // 源应用用户
    SourceApp   string   `json:"source_app" binding:"required"`    // 源应用代码
    Packages    []string `json:"packages" binding:"required"`       // 要发布的 package 列表
    Name        string   `json:"name" binding:"required"`           // 应用名称
    Description string   `json:"description"`                       // 应用描述
    Category    string   `json:"category"`                          // 分类
    Tags        []string `json:"tags"`                              // 标签
    
    // 服务费信息
    ServiceFeePersonal   float64 `json:"service_fee_personal"`    // 个人用户服务费
    ServiceFeeEnterprise float64 `json:"service_fee_enterprise"`  // 企业用户服务费
    
    // 构造记录（可选）
    IncludeConstructionLog bool `json:"include_construction_log"` // 是否包含构造记录
}

type PublishAppResp struct {
    HubAppID int64  `json:"hub_app_id"`  // Hub 应用 ID
    HubURL   string `json:"hub_url"`     // Hub 应用 URL
    Message  string `json:"message"`     // 响应消息
}
```

**实现流程**：
```go
func (s *AppService) PublishApp(ctx context.Context, req *PublishAppReq) (*PublishAppResp, error) {
    // 1. 验证权限（JWT 或 API Key）
    
    // 2. 从 OS 获取源代码
    sourceCodeList, err := s.osClient.GetSourceCode(req.SourceUser, req.SourceApp, req.Packages)
    
    // 3. 从 OS 获取构造记录（如果包含）
    var constructionLog *CodeGenerationLog
    if req.IncludeConstructionLog {
        constructionLog, err = s.osClient.GetConstructionLog(req.SourceUser, req.SourceApp)
    }
    
    // 4. 存储到 Hub 数据库
    hubApp := &model.HubApp{
        Name: req.Name,
        Description: req.Description,
        SourceUser: req.SourceUser,
        SourceApp: req.SourceApp,
        ServiceFeePersonal: req.ServiceFeePersonal,
        ServiceFeeEnterprise: req.ServiceFeeEnterprise,
        // ...
    }
    s.hubAppRepo.Create(hubApp)
    
    // 5. 存储函数组源代码
    for _, sourceCode := range sourceCodeList {
        hubFunctionGroup := &model.HubFunctionGroup{
            HubAppID: hubApp.ID,
            FullGroupCode: sourceCode.FullGroupCode,
            GroupCode: sourceCode.GroupCode,
            SourceCode: sourceCode.Content,  // 🔥 存储源代码
            Version: sourceCode.Version,
        }
        s.hubFunctionGroupRepo.Create(hubFunctionGroup)
    }
    
    // 6. 存储构造记录（如果包含）
    if constructionLog != nil {
        constructionLog.HubAppID = hubApp.ID
        s.constructionLogRepo.Create(constructionLog)
    }
    
    // 7. 返回 Hub URL
    return &PublishAppResp{
        HubAppID: hubApp.ID,
        HubURL: fmt.Sprintf("https://hub.ai-agent-os.com/apps/%d", hubApp.ID),
    }, nil
}
```

### 4.2 应用浏览 API

#### 1. 应用列表

```go
// GET /api/v1/apps
type AppListReq struct {
    Category string `form:"category"`  // 分类筛选
    Tag      string `form:"tag"`       // 标签筛选
    Keyword  string `form:"keyword"`   // 关键词搜索
    Page     int    `form:"page"`      // 页码
    PageSize int    `form:"page_size"` // 每页数量
}

type AppListResp struct {
    Apps      []*AppInfo `json:"apps"`
    Total     int64      `json:"total"`
    Page      int        `json:"page"`
    PageSize  int        `json:"page_size"`
}

type AppInfo struct {
    ID          int64    `json:"id"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Category    string   `json:"category"`
    Tags        []string `json:"tags"`
    ServiceFeePersonal   float64 `json:"service_fee_personal"`
    ServiceFeeEnterprise float64 `json:"service_fee_enterprise"`
    DownloadCount int    `json:"download_count"`
    TrialCount    int    `json:"trial_count"`
    Rating        float64 `json:"rating"`
    PublisherUsername string `json:"publisher_username"`
    PublishedAt   string `json:"published_at"`
}
```

#### 2. 应用详情

```go
// GET /api/v1/apps/{app_id}
type AppDetailResp struct {
    ID          int64    `json:"id"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Category    string   `json:"category"`
    Tags        []string `json:"tags"`
    
    // 服务费信息
    ServiceFeePersonal   float64 `json:"service_fee_personal"`
    ServiceFeeEnterprise float64 `json:"service_fee_enterprise"`
    
    // 源代码信息（开源）
    SourceCodeAvailable bool `json:"source_code_available"` // 源代码是否可用
    SourceCodePreview    string `json:"source_code_preview"` // 源代码预览（前 100 行）
    
    // 构造记录信息（开源）
    ConstructionLogAvailable bool `json:"construction_log_available"` // 构造记录是否可用
    ConstructionLogSummary     *ConstructionLogSummary `json:"construction_log_summary"` // 构造记录摘要
    
    // 统计信息
    DownloadCount int    `json:"download_count"`
    TrialCount    int    `json:"trial_count"`
    Rating        float64 `json:"rating"`
    
    // 开发者信息
    PublisherUsername string `json:"publisher_username"`
    PublishedAt       string `json:"published_at"`
    
    // 服务内容
    ServiceContent *ServiceContent `json:"service_content"`
}

type ConstructionLogSummary struct {
    TotalTurns      int    `json:"total_turns"`
    GenerationTime  string `json:"generation_time"`
    FunctionsGenerated int `json:"functions_generated"`
    APIsGenerated   int    `json:"apis_generated"`
    Preview         string `json:"preview"` // 前 3 轮对话预览
}

type ServiceContent struct {
    SupportDays      int    `json:"support_days"`       // 技术支持天数
    FreeRequirements int    `json:"free_requirements"`  // 免费需求调整数量
    Documentation    bool   `json:"documentation"`      // 文档知识库
    UpgradePeriod    string `json:"upgrade_period"`     // 版本升级期限
}
```

### 4.3 应用试用 API

#### 1. 跳转到 OS 试用

```go
// GET /api/v1/apps/{app_id}/trial
// 返回跳转 URL（带 Token）
type TrialAppResp struct {
    TrialURL string `json:"trial_url"` // https://os.ai-agent-os.com/trial?app_id=123&token=xxx
}
```

### 4.4 应用克隆 API

#### 1. 克隆应用

```go
// POST /api/v1/apps/{app_id}/clone
type CloneAppReq struct {
    TargetUser string `json:"target_user" binding:"required"`   // 目标用户
    TargetApp  string `json:"target_app" binding:"required"`    // 目标应用代码
    PaymentType string `json:"payment_type" binding:"required"` // 'personal' | 'enterprise'
}

type CloneAppResp struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
    CloneURL string `json:"clone_url"` // 跳转到 OS 克隆页面
}
```

**实现流程**：
```go
func (s *AppService) CloneApp(ctx context.Context, appID int64, req *CloneAppReq) (*CloneAppResp, error) {
    // 1. 验证服务费支付状态
    payment, err := s.checkPaymentStatus(ctx, appID, req.TargetUser, req.PaymentType)
    if err != nil {
        return nil, err
    }
    if !payment.Paid {
        return &CloneAppResp{
            Success: false,
            Message: "请先支付服务费",
            CloneURL: fmt.Sprintf("https://hub.ai-agent-os.com/apps/%d/payment?type=%s", appID, req.PaymentType),
        }, nil
    }
    
    // 2. 调用 OS API 克隆应用
    cloneReq := &OSCloneReq{
        HubAppID: appID,
        TargetUser: req.TargetUser,
        TargetApp: req.TargetApp,
    }
    err = s.osClient.CloneFromHub(cloneReq)
    if err != nil {
        return nil, err
    }
    
    // 3. 更新统计
    s.hubAppRepo.IncrementDownloadCount(appID)
    
    return &CloneAppResp{
        Success: true,
        Message: "克隆成功",
        CloneURL: fmt.Sprintf("https://os.ai-agent-os.com/apps/%s/%s", req.TargetUser, req.TargetApp),
    }, nil
}
```

### 4.5 服务费支付 API

#### 1. 创建支付订单

```go
// POST /api/v1/apps/{app_id}/payment
type CreatePaymentReq struct {
    PaymentType string `json:"payment_type" binding:"required"` // 'personal' | 'enterprise'
}

type CreatePaymentResp struct {
    PaymentID   int64   `json:"payment_id"`
    Amount      float64 `json:"amount"`
    PaymentURL  string  `json:"payment_url"` // 支付页面 URL
}
```

#### 2. 支付回调

```go
// POST /api/v1/payments/{payment_id}/callback
type PaymentCallbackReq struct {
    PaymentID     int64  `json:"payment_id"`
    PaymentStatus string `json:"payment_status"` // 'paid' | 'failed'
    PaymentMethod string `json:"payment_method"`
}
```

---

## 五、开发任务清单

### 5.1 Phase 1：核心功能（MVP）

#### 1. 数据库设计 ✅

- [ ] 创建 Hub 应用表
- [ ] 创建 Hub 函数组表（存储源代码）
- [ ] 创建构造记录表
- [ ] 创建服务费支付记录表
- [ ] 创建服务记录表

#### 2. 后端 API ✅

**应用发布**：
- [ ] POST /api/v1/apps/publish（发布应用）
- [ ] 从 OS 获取源代码
- [ ] 从 OS 获取构造记录
- [ ] 存储到 Hub 数据库

**应用浏览**：
- [ ] GET /api/v1/apps（应用列表）
- [ ] GET /api/v1/apps/{app_id}（应用详情）
- [ ] 支持搜索、分类、标签筛选

**应用试用**：
- [ ] GET /api/v1/apps/{app_id}/trial（生成试用 URL）

**应用克隆**：
- [ ] POST /api/v1/apps/{app_id}/clone（克隆应用）
- [ ] 验证服务费支付状态
- [ ] 调用 OS API 克隆

**服务费支付**：
- [ ] POST /api/v1/apps/{app_id}/payment（创建支付订单）
- [ ] POST /api/v1/payments/{payment_id}/callback（支付回调）

#### 3. OS 集成 ✅

**OS API（需要实现）**：
- [ ] GET /api/v1/apps/{user}/{app}/source-code（获取源代码）
- [ ] GET /api/v1/apps/{user}/{app}/construction-log（获取构造记录）
- [ ] POST /api/v1/hub/clone-from-hub（从 Hub 克隆）

**OS 前端**：
- [ ] "发布到 Hub" 按钮
- [ ] "从 Hub 克隆" 功能
- [ ] Hub → OS 跳转处理

#### 4. Hub 前端 ✅

**应用浏览**：
- [ ] 应用列表页面
- [ ] 应用搜索和筛选
- [ ] 应用详情页面
- [ ] 代码预览（开源）
- [ ] 构造记录摘要（开源）

**应用试用**：
- [ ] "试用" 按钮
- [ ] 跳转到 OS 试用

**应用克隆**：
- [ ] "克隆" 按钮
- [ ] 服务费支付页面
- [ ] 支付成功后的克隆流程

**用户认证**：
- [ ] 登录/注册页面
- [ ] 调用 OS API 进行认证
- [ ] Token 管理

---

## 六、实施步骤

### 6.1 Phase 1：核心功能（MVP）

**目标**：实现基本的应用发布、浏览、试用、克隆功能

**时间估算**：2-3 周

**任务优先级**：
1. ✅ **数据库设计**（1 天）
2. ✅ **后端 API**（1 周）
3. ✅ **OS 集成**（3 天）
4. ✅ **Hub 前端**（1 周）

### 6.2 Phase 2：服务费支付

**目标**：实现服务费支付功能

**时间估算**：1 周

**任务**：
- [ ] 支付接口集成
- [ ] 支付回调处理
- [ ] 收益分配逻辑

### 6.3 Phase 3：服务管理

**目标**：实现服务管理功能（技术支持、需求调整等）

**时间估算**：1 周

**任务**：
- [ ] 服务工单系统
- [ ] 服务记录管理
- [ ] 服务统计

---

## 七、技术实现细节

### 7.1 Hub 调用 OS API

**OS API 客户端**：
```go
// hub/backend/client/os_client.go
type OSClient struct {
    baseURL string
    httpClient *http.Client
}

func (c *OSClient) GetSourceCode(user, app string, packages []string) ([]*SourceCodeInfo, error) {
    // 调用 OS API
    url := fmt.Sprintf("%s/api/v1/apps/%s/%s/source-code?packages=%s",
        c.baseURL, user, app, strings.Join(packages, ","))
    resp, err := c.httpClient.Get(url)
    // ...
}

func (c *OSClient) GetConstructionLog(user, app string) (*CodeGenerationLog, error) {
    // 调用 OS API
    url := fmt.Sprintf("%s/api/v1/apps/%s/%s/construction-log", c.baseURL, user, app)
    resp, err := c.httpClient.Get(url)
    // ...
}

func (c *OSClient) CloneFromHub(req *CloneFromHubReq) error {
    // 调用 OS API
    url := fmt.Sprintf("%s/api/v1/hub/clone-from-hub", c.baseURL)
    resp, err := c.httpClient.Post(url, "application/json", req)
    // ...
}
```

### 7.2 Hub 用户认证

**调用 OS API 认证**：
```go
// hub/backend/service/auth_service.go
type HubAuthService struct {
    osClient *OSClient
}

func (s *HubAuthService) Login(username, password string) (*User, string, error) {
    // 调用 OS API 进行认证
    req := &OSLoginReq{
        Username: username,
        Password: password,
    }
    resp, err := s.osClient.Login(req)
    if err != nil {
        return nil, "", err
    }
    
    return resp.User, resp.Token, nil
}
```

### 7.3 Hub → OS 跳转

**生成跳转 URL**：
```go
// hub/backend/service/app_service.go
func (s *AppService) GenerateTrialURL(appID int64, token string) string {
    return fmt.Sprintf("https://os.ai-agent-os.com/trial?app_id=%d&token=%s", appID, token)
}

func (s *AppService) GenerateCloneURL(appID int64, token string) string {
    return fmt.Sprintf("https://os.ai-agent-os.com/clone?app_id=%d&token=%s", appID, token)
}
```

---

## 八、开发规范

### 8.1 代码规范

**后端**：
- ✅ 遵循 Go 代码规范
- ✅ 使用依赖注入
- ✅ 错误处理完善
- ✅ 日志记录完整

**前端**：
- ✅ 遵循 Vue 3 代码规范
- ✅ 使用 TypeScript
- ✅ 组件化开发
- ✅ 响应式设计

### 8.2 数据库规范

- ✅ 使用 GORM
- ✅ 表名使用下划线命名
- ✅ 字段名使用下划线命名
- ✅ 索引设计合理

### 8.3 API 规范

- ✅ RESTful API 设计
- ✅ 统一的响应格式
- ✅ 完善的错误处理
- ✅ API 文档完整

---

## 九、总结

### 9.1 核心设计

**Hub + OS 一体化生态系统**：
- ✅ **Hub**：应用市场，卖软件、介绍软件
- ✅ **OS**：运行平台，可以试用、克隆、运行软件
- ✅ **互通**：Hub 和 OS 用户系统完全互通

**服务费模式**：
- ✅ **代码完全开源**：代码完全开源，增强信任，促进付费
- ✅ **构造记录完全开源**：构造记录完全开源，增强信任，促进付费
- ✅ **服务费模式**：通过服务费模式，实现可持续的商业模式

### 9.2 开发优先级

**Phase 1（MVP）**：
1. ✅ 数据库设计
2. ✅ 后端 API（发布、浏览、试用、克隆）
3. ✅ OS 集成
4. ✅ Hub 前端

**Phase 2**：
1. ⚠️ 服务费支付
2. ⚠️ 收益分配

**Phase 3**：
1. ⚠️ 服务管理
2. ⚠️ 服务统计

### 9.3 关键成功因素

1. ✅ **统一用户系统**：Hub 和 OS 共享用户系统，单点登录
2. ✅ **无缝跳转**：Hub 和 OS 之间无缝跳转，用户体验流畅
3. ✅ **服务费模式**：通过服务费模式，实现可持续的商业模式
4. ✅ **完全开源**：代码和构造记录完全开源，增强信任

**这样既实现了完整的用户体验，又实现了可持续的商业模式，还满足了不同用户的需求。** ✅

