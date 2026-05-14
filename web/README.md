# AI Agent OS - 前端架构文档

> **本文档是前端项目的核心指南，所有开发工作必须遵循本文档的架构设计和开发规范。**

## 📚 目录

- [一、项目概述](#一项目概述)
- [二、架构设计](#二架构设计)
- [三、目录结构](#三目录结构)
- [四、核心机制](#四核心机制)
- [五、开发指南](#五开发指南)
- [六、典型场景示例](#六典型场景示例)
- [七、最佳实践](#七最佳实践)
- [八、常见问题](#八常见问题)

---

## 一、项目概述

### 1.1 技术栈

- **框架**: Vue 3 + TypeScript
- **状态管理**: Pinia
- **UI 组件库**: Element Plus
- **路由**: Vue Router
- **构建工具**: Vite
- **代码规范**: ESLint + Prettier

### 1.2 核心特性

- ✅ **动态组件渲染系统**：根据后端配置动态渲染表单、表格等组件
- ✅ **分层架构设计**：Presentation → Application → Domain → Infrastructure
- ✅ **领域导向的工程组织**：主业务页面、流程编排、领域逻辑、基础设施分层清晰
- ✅ **渐进式收边界**：主页面已迁入 `architecture/`，公共底座由 `architecture/runtime`、`shared`、`utils` 承担
- ✅ **事件驱动架构**：组件间通过事件总线解耦

### 1.3 当前状态说明

- 工作空间、工作台等主页面入口已经迁移到 `src/architecture/`
- `src/architecture/runtime/`、`src/shared/`、`src/architecture/runtime/utils/` 仍然是当前线上主链路正在使用的公共底座
- `src/architecture/presentation/views/` 目前基本只保留错误页等少量遗留页面
- 默认启用产品聚焦模式，普通用户入口优先保留工作空间、服务树、工作台、Form/Table/Chart、Docs 和 LLM 管理；组织、消息、操作日志、定时任务等入口由 `src/architecture/infrastructure/config/features.ts` 统一控制
- 因此前端当前真实状态是：**主页面已迁到 `architecture/`，底层能力已收口为 `architecture/runtime` 运行时底座**

### 1.4 产品聚焦模式

前端通过 `src/architecture/infrastructure/config/features.ts` 管理可见能力。`VITE_AOS_FOCUSED_MODE` 默认开启；开启后，组织、消息、定时任务、操作日志、讨论区、企业升级等高级入口默认隐藏，但路由和后端接口仍保留兼容。

常用开关：

| 环境变量 | 默认行为 |
|---|---|
| `VITE_AOS_FOCUSED_MODE` | 默认 `true`，测试环境默认 `false` |
| `VITE_AOS_FEATURE_ORGANIZATION` | 未设置时仅完整模式开启 |
| `VITE_AOS_FEATURE_MESSAGES` | 未设置时仅完整模式开启 |
| `VITE_AOS_FEATURE_SCHEDULED_TASKS` | 未设置时仅完整模式开启 |
| `VITE_AOS_FEATURE_OPERATE_LOGS` | 未设置时仅完整模式开启 |
| `VITE_AOS_FEATURE_BOARD` | 未设置时仅完整模式开启 |
| `VITE_AOS_FEATURE_LLM_MANAGEMENT` | 默认开启 |

---

## 二、架构设计

### 2.1 四层架构

```
┌─────────────────────────────────────────────────────────┐
│  Presentation Layer (表示层)                              │
│  - Views (页面组件)                                       │
│  - Widgets (UI 组件)                                      │
│  - Composables (组合式函数)                               │
│  职责：UI 渲染、用户交互、事件监听                           │
└─────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────┐
│  Application Layer (应用层)                               │
│  - Services (应用服务)                                     │
│  职责：业务流程编排、协调多个 Domain Service                 │
└─────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────┐
│  Domain Layer (领域层)                                    │
│  - Services (领域服务)                                     │
│  - Interfaces (抽象接口)                                   │
│  - Types (类型定义)                                        │
│  职责：核心业务逻辑、领域规则、状态管理                        │
└─────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────┐
│  Infrastructure Layer (基础设施层)                         │
│  - API Client (API 调用)                                  │
│  - State Manager (状态管理器)                              │
│  - Event Bus (事件总线)                                    │
│  - Factories (工厂类)                                      │
│  职责：技术实现、外部依赖、基础设施                           │
└─────────────────────────────────────────────────────────┘
```

### 2.2 核心原则

| 原则 | 说明 | 体现 |
|------|------|------|
| **SRP** (单一职责原则) | 每个类/模块只负责一件事 | Domain Service 只负责业务逻辑，不负责 UI 渲染 |
| **OCP** (开闭原则) | 对扩展开放，对修改封闭 | 新增 Widget 类型主要通过注册表扩展 |
| **ISP** (接口隔离原则) | 接口设计简洁 | IStateManager、IEventBus、IApiClient 等接口职责明确 |
| **DIP** (依赖倒置原则) | 高层模块依赖抽象 | Domain/Application 通过接口依赖基础设施 |

> 说明：当前实现是**领域导向的分层架构**，而不是追求教科书式的“纯 DDD / 纯 SOLID”。上线前更重视边界清晰和可维护性，而不是过度抽象。

### 2.3 设计模式

- **策略模式**：FieldExtractorRegistry（根据字段类型选择不同的提取器）
- **工厂/注册表模式**：WidgetComponentFactory + widgetRegistry（根据组件类型选择组件）
- **策略模式**：FieldExtractorRegistry（根据字段结构提取提交值）
- **门面模式**：`src/architecture/domain/types`、`architecture/domain/types` 对共享类型做统一出口
- **观察者模式**：EventBus（事件发布订阅）
- **单例模式**：Pinia Store（全局状态管理）

---

## 三、目录结构

### 3.1 完整目录树

```text
web/
├── src/
│   ├── architecture/                    # 主业务架构层（四层架构）
│   │   ├── application/                 # 应用服务：流程编排
│   │   ├── domain/                      # 领域服务、接口、类型
│   │   ├── infrastructure/              # API、事件总线、状态管理、Widget 注册
│   │   ├── presentation/                # views / components / widgets / composables
│   │   └── runtime/                     # 表单状态、Widget 运行时、校验、共享常量
│   ├── features/                        # agent / auth / permission / user 等功能模块
│   ├── shared/                          # 共享组件、富文本与通用展示能力
│   ├── utils/                           # 通用工具函数
│   ├── views/                           # 遗留页面（当前基本只剩错误页）
│   ├── App.vue
│   └── main.ts
└── README.md
```

### 3.2 目录职责说明

#### 🏗️ architecture/ - 主业务架构层

**作用**：承载工作空间、工作台、表单/表格/图表等主业务页面与四层架构实现。

| 子目录 | 职责 | 示例 |
|--------|------|------|
| `presentation/` | UI 渲染、用户交互、Widget 展示 | WorkspaceView.vue, FormView.vue |
| `application/` | 业务流程编排 | FormApplicationService |
| `domain/` | 核心业务逻辑 | FormDomainService, IStateManager |
| `infrastructure/` | 技术实现、接口请求、组件注册、状态适配 | api, widgetRegistry, EventBusImpl |

**何时添加代码**：
- 新增页面 → `presentation/views/`
- 新增业务流程 → `application/services/`
- 新增业务逻辑 → `domain/services/`
- 新增基础设施 → `infrastructure/`

#### 🎯 architecture/runtime/ - 运行时基础能力层

**作用**：提供当前仍在主链路中使用的稳定基础能力，是架构内的运行时底座。

| 子目录 | 职责 | 示例 |
|--------|------|------|
| `stores/` | 状态管理 | formData.ts, extractors/ |
| `constants/` | 共享常量 | widget.ts |
| `managers/` | 运行期管理器 | 运行期加载/缓存管理 |
| `widgetRuntime/` | Widget 默认值、校验等中性运行时能力 | defaultValue.ts, validation.ts |
| `utils/` | 工具函数 | logger.ts |
| `validation/` | 验证引擎 | ValidationEngine.ts |

**何时添加代码**：
- 新增提取器 → `stores/extractors/`
- 新增基础常量/校验能力 → `constants/` / `validation/`
- 新增底层工具函数 → `utils/`

#### 🧩 shared/ / features/ / utils/

**作用**：

- `shared/`：跨业务共享组件、富文本编辑器、通用展示组件
- `features/`：组织、用户、消息等横向功能模块
- `utils/`：与具体架构层无关的通用工具函数

#### 📄 views/

**作用**：遗留页面目录。当前主业务路由已切到 `architecture/presentation/views/`，这里只保留少量兼容页面与错误页。

---

## 四、核心机制

### 4.1 值提取机制（Data Extraction）

**流程图**：
```
表单提交
  ↓
FormApplicationService.submitForm()
  ↓
FormDomainService.getSubmitData(fields)
  ↓
FormStateManager.getSubmitData(fields)
  ↓
useFormDataStore().getSubmitData(fields)
  ↓
FieldExtractorRegistry.extractField(field, fieldPath, getValue)
  ↓ 根据 widget.type 选择提取器
  ├─ BasicFieldExtractor (text, select, number, etc.)
  ├─ MultiSelectFieldExtractor (multiselect)
  ├─ FormFieldExtractor (form/struct)
  └─ TableFieldExtractor (table/array)
  ↓
递归提取所有字段的 raw 值
  ↓
返回提交数据对象
```

**关键点**：
- ✅ 使用**策略模式**（FieldExtractorRegistry）
- ✅ 支持**任意嵌套深度**（递归提取）
- ✅ 支持**任意数据结构**（form 嵌套 table，table 嵌套 form 等）

**示例**：
```typescript
// 提交数据
{
  business_info: {  // form 字段
    industry: "金融",
    products: [  // table 字段
      { product_name: "产品1", price: 100 }
    ]
  }
}
```

### 4.2 渲染机制（Rendering）

**流程图**：
```
后端返回字段配置 (FieldConfig)
  ↓
FormRenderer / TableRenderer 遍历字段
  ↓
WidgetComponent 包装每个字段
  ↓
widgetComponentFactory.getRequestComponent(widget.type, mode)
  ↓ 根据 widget.type 返回对应的 Vue 组件
  ├─ InputWidget (text)
  ├─ SelectWidget (select)
  ├─ FormWidget (form)
  ├─ TableWidget (table)
  └─ ...
  ↓
组件根据 mode 渲染不同的 UI
  ├─ edit: 可编辑（el-input, el-select）
  ├─ response: 只读（<span>）
  ├─ table-cell: 表格单元格（el-tag）
  ├─ detail: 详情（格式化展示）
  └─ search: 搜索（搜索输入框）
```

**关键点**：
- ✅ 使用**工厂模式**（WidgetComponentFactory）
- ✅ 支持**5 种渲染模式**
- ✅ 组件之间**完全解耦**

### 4.3 状态管理（State Management）

**架构**：
```
Pinia Store (useFormDataStore)
  ├─ data: Map<fieldPath, FieldValue>
  ├─ setValue(fieldPath, value)
  ├─ getValue(fieldPath)
  └─ getSubmitData(fields)
```

**FieldValue 数据结构**：
```typescript
{
  raw: any,        // 原始值（提交给后端）
  display: string, // 显示值（前端展示）
  meta: {          // 元数据
    displayInfo?: string,  // 详细信息
    statistics?: any,      // 统计信息
    [key: string]: any     // 其他自定义元数据
  }
}
```

**字段路径规则**：
```typescript
// 一级字段
'name' → { raw: '张三', display: '张三', meta: {} }

// form 嵌套字段
'business_info.industry' → { raw: '金融', display: '金融', meta: {} }

// table 嵌套字段
'products[0].product_name' → { raw: '产品1', display: '产品1', meta: {} }

// form 嵌套 table
'business_info.products[0].product_name' → { raw: '产品1', display: '产品1', meta: {} }
```

### 4.4 事件总线（Event Bus）

**架构**：
```
EventBus (IEventBus)
  ├─ emit(event, data)   # 发布事件
  ├─ on(event, handler)  # 订阅事件
  └─ off(event, handler) # 取消订阅
```

**核心事件**：
```typescript
// 工作空间事件
WorkspaceEvent.initialized        // 工作空间初始化完成
WorkspaceEvent.appSelected         // 应用选中
WorkspaceEvent.nodeClicked         // 节点点击
WorkspaceEvent.tabClosed           // Tab 关闭

// 表单事件
FormEvent.initialized              // 表单初始化完成
FormEvent.fieldUpdated             // 字段值更新
FormEvent.validated                // 表单验证完成
FormEvent.submitted                // 表单提交完成
FormEvent.responseReceived         // 响应数据接收

// 表格事件
TableEvent.initialized             // 表格初始化完成
TableEvent.dataLoaded              // 数据加载完成
TableEvent.searchChanged           // 搜索条件变化
TableEvent.rowAdded                // 行添加
TableEvent.rowUpdated              // 行更新
TableEvent.rowDeleted              // 行删除
```

---

## 五、开发指南

### 5.1 新增 UI 组件（Widget）

**场景**：新增一个自定义组件类型（如 RichTextEditor）。

#### 步骤 1：创建 Vue 组件

文件位置：`src/architecture/presentation/widgets/RichTextEditorWidget.vue`

```vue
<template>
  <!-- edit 模式：可编辑 -->
  <div v-if="mode === 'edit'">
    <el-input
      type="textarea"
      :model-value="modelValue?.raw"
      @input="handleUpdate"
      :rows="10"
    />
  </div>
  
  <!-- response 模式：只读展示 -->
  <div v-else-if="mode === 'response'">
    <div v-html="modelValue?.display"></div>
  </div>
  
  <!-- table-cell 模式：简化展示 -->
  <span v-else-if="mode === 'table-cell'">
    {{ truncate(modelValue?.display, 50) }}
  </span>
  
  <!-- detail 模式：详细展示 -->
  <div v-else-if="mode === 'detail'">
    <div v-html="modelValue?.display"></div>
  </div>
  
  <!-- search 模式：搜索输入 -->
  <el-input v-else-if="mode === 'search'" :model-value="modelValue?.raw" @input="handleUpdate" />
</template>

<script setup lang="ts">
import type { FieldValue } from '@/architecture/domain/types'

interface Props {
  fieldPath: string
  mode: 'edit' | 'response' | 'table-cell' | 'detail' | 'search'
  modelValue?: FieldValue
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: FieldValue): void
}>()

function handleUpdate(value: string): void {
  emit('update:modelValue', {
    raw: value,
    display: value,  // 或者格式化后的 HTML
    meta: {}
  })
}

function truncate(str: string | undefined, len: number): string {
  if (!str) return ''
  return str.length > len ? str.substring(0, len) + '...' : str
}
</script>
```

#### 步骤 2：注册组件到工厂

文件位置：`src/architecture/infrastructure/widgetRegistry/index.ts`

```typescript
import RichTextEditorWidget from '@/architecture/presentation/widgets/RichTextEditorWidget.vue'
import RichTextResponseWidget from '@/architecture/presentation/widgets/RichTextResponseWidget.vue'

// 注册请求参数组件
widgetComponentFactory.registerRequestComponent('rich_text_editor', RichTextEditorWidget)

// 如需单独的响应展示组件，可同时注册响应组件
widgetComponentFactory.registerResponseComponent('rich_text_editor', RichTextResponseWidget)
```

#### 步骤 3：（可选）创建提取器

**如果需要特殊提交提取逻辑**（通常简单组件不需要）：

文件位置：`src/architecture/runtime/stores/extractors/RichTextEditorFieldExtractor.ts`

```typescript
import type { IFieldExtractor, FieldExtractorRegistry } from './FieldExtractor'
import type { FieldConfig } from '@/architecture/domain/types'

export class RichTextEditorFieldExtractor implements IFieldExtractor {
  extract(
    field: FieldConfig,
    fieldPath: string,
    getValue: (path: string) => any,
    extractorRegistry: FieldExtractorRegistry
  ): any {
    const value = getValue(fieldPath)
    // 自定义提取逻辑（如需要）
    return value?.raw
  }
}
```

#### 步骤 4：（可选）注册提取器

文件位置：`src/architecture/runtime/stores/extractors/FieldExtractorRegistry.ts`

```typescript
import { RichTextEditorFieldExtractor } from './RichTextEditorFieldExtractor'

constructor() {
  // ... 其他提取器注册
  this.registerExtractor('rich_text_editor', new RichTextEditorFieldExtractor())
}
```

#### 步骤 5：使用

后端返回配置：
```json
{
  "code": "content",
  "name": "内容",
  "widget": {
    "type": "rich_text_editor"
  }
}
```

前端自动渲染对应的组件！

---

### 5.2 新增业务逻辑（Domain Logic）

**场景**：新增一个跨页面复用的业务能力，例如日志记录、草稿保存、批量导入状态管理。

#### 步骤 1：确定逻辑归属

**问题**：这个功能是否属于某个已有的 Domain Service？
- 如果是表单相关 → `FormDomainService`
- 如果是表格相关 → `TableDomainService`
- 如果是工作空间相关 → `WorkspaceDomainService`
- 如果是新的领域 → 创建新的 Domain Service

#### 步骤 2：定义领域接口

把外部依赖抽象成接口，放在 `src/architecture/domain/interfaces/`。Domain 层只依赖接口，不直接依赖 Vue、Pinia 或具体 HTTP 客户端。

```typescript
export interface IActionLogService {
  logAction(action: string, payload: Record<string, unknown>): Promise<void>
}
```

#### 步骤 3：创建 Domain Service

把核心业务规则放在 `src/architecture/domain/services/`，只处理领域状态和规则。

```typescript
export class ActionLogDomainService {
  constructor(private actionLogService: IActionLogService) {}

  async recordSubmit(functionCode: string, payload: Record<string, unknown>): Promise<void> {
    await this.actionLogService.logAction('form_submit', {
      functionCode,
      payload,
      timestamp: Date.now()
    })
  }
}
```

#### 步骤 4：创建 Application Service

Application 层负责流程编排，把页面事件、领域服务和基础设施串起来。

```typescript
export class ActionLogApplicationService {
  constructor(private domainService: ActionLogDomainService) {}

  async recordFormSubmit(functionCode: string, payload: Record<string, unknown>): Promise<void> {
    await this.domainService.recordSubmit(functionCode, payload)
  }
}
```

#### 步骤 5：注册到 ServiceFactory

在 `src/architecture/infrastructure/factories/ServiceFactory.ts` 里统一创建实例，避免组件自行 new 服务。

---

### 5.3 新增页面（View）

**场景**：新增一个"应用管理"页面。

#### 步骤 1：创建 Vue 组件

文件位置：`src/architecture/presentation/views/AppManagementView.vue`

```vue
<template>
  <div class="app-management">
    <h1>应用管理</h1>
    <!-- 页面内容 -->
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { serviceFactory } from '@/architecture/infrastructure/factories/ServiceFactory'

// 获取需要的服务
const appService = serviceFactory.getWorkspaceApplicationService()

onMounted(() => {
  // 初始化逻辑
})
</script>
```

#### 步骤 2：注册路由

文件位置：`src/architecture/infrastructure/router/index.ts`

```typescript
{
  path: '/workspace/app-management',
  name: 'AppManagement',
  component: () => import('@/architecture/presentation/views/AppManagementView.vue'),
  meta: { requiresAuth: true }
}
```

#### 步骤 3：添加导航（如果需要）

在侧边栏或导航栏中添加链接：

```vue
<router-link to="/workspace/app-management">应用管理</router-link>
```

---

### 5.4 新增功能模块（完整示例）

**场景**：新增一个"工作流系统"模块。

#### 完整步骤：

1. **Domain Layer**
   - 创建 `src/architecture/domain/interfaces/IWorkflow.ts`（接口定义）
   - 创建 `src/architecture/domain/services/WorkflowDomainService.ts`（核心业务逻辑）
   - 在 `src/architecture/domain/types/index.ts` 中定义类型

2. **Infrastructure Layer**
   - 创建 `src/architecture/infrastructure/stateManager/WorkflowStateManager.ts`（状态管理）
   - 在 `ServiceFactory` 中注册 Workflow 相关服务

3. **Application Layer**
   - 创建 `src/architecture/application/services/WorkflowApplicationService.ts`（业务流程编排）

4. **Presentation Layer**
   - 创建 `src/architecture/presentation/views/WorkflowView.vue`（页面）
   - 创建 `src/architecture/presentation/composables/useWorkflowInitialization.ts`（组合式函数）

5. **注册路由**
   - 在 `src/architecture/infrastructure/router/index.ts` 中添加路由

6. **（可选）新增组件**
   - 在 `src/architecture/presentation/widgets/` 中创建工作流相关的组件

---

## 六、典型场景示例

### 6.1 场景 1：新增日志记录功能

**问题**：需要记录用户的所有操作日志。

**解决方案**：

1. 创建 `LogDomainService`：

```typescript
// src/architecture/domain/services/LogDomainService.ts
export class LogDomainService {
  constructor(
    private apiClient: IApiClient,
    private eventBus: IEventBus
  ) {
    this.setupEventListeners()
  }

  private setupEventListeners(): void {
    // 监听所有需要记录的事件
    this.eventBus.on(FormEvent.submitted, (data) => {
      this.logAction('form_submit', data)
    })
    
    this.eventBus.on(TableEvent.rowAdded, (data) => {
      this.logAction('table_row_add', data)
    })
    
    // ... 其他事件
  }

  private async logAction(action: string, data: any): Promise<void> {
    try {
      await this.apiClient.post('/api/v1/logs', {
        action,
        data,
        timestamp: Date.now()
      })
    } catch (error) {
      Logger.error('LogDomainService', '日志记录失败', error)
    }
  }
}
```

2. 在 `ServiceFactory` 中注册并自动启动：

```typescript
class ServiceFactory {
  private logDomainService?: LogDomainService
  
  constructor() {
    // 自动启动日志服务
    this.getLogDomainService()
  }
  
  private getLogDomainService(): LogDomainService {
    if (!this.logDomainService) {
      this.logDomainService = new LogDomainService(
        this.apiClient,
        this.eventBus
      )
    }
    return this.logDomainService
  }
}
```

**代码位置**：
- Domain Service: `src/architecture/domain/services/LogDomainService.ts`

---

### 6.3 场景 3：新增定时任务功能

**问题**：需要定时执行某些任务（如定时刷新数据）。

**解决方案**：

1. 创建 `SchedulerDomainService`：

```typescript
// src/architecture/domain/services/SchedulerDomainService.ts
export class SchedulerDomainService {
  private timers: Map<string, number> = new Map()

  /**
   * 添加定时任务
   */
  addTask(taskId: string, interval: number, callback: () => void): void {
    // 清除已存在的任务
    this.removeTask(taskId)
    
    // 创建新任务
    const timerId = window.setInterval(callback, interval)
    this.timers.set(taskId, timerId)
    
    Logger.debug('SchedulerDomainService', `定时任务 ${taskId} 已启动，间隔 ${interval}ms`)
  }

  /**
   * 移除定时任务
   */
  removeTask(taskId: string): void {
    const timerId = this.timers.get(taskId)
    if (timerId) {
      window.clearInterval(timerId)
      this.timers.delete(taskId)
      Logger.debug('SchedulerDomainService', `定时任务 ${taskId} 已停止`)
    }
  }

  /**
   * 清除所有定时任务
   */
  clearAll(): void {
    this.timers.forEach((timerId) => {
      window.clearInterval(timerId)
    })
    this.timers.clear()
  }
}
```

2. 使用：

```typescript
// 在 TableApplicationService 中使用
export class TableApplicationService {
  constructor(
    private domainService: TableDomainService,
    private schedulerService: SchedulerDomainService,
    private eventBus: IEventBus
  ) {}

  /**
   * 启用自动刷新
   */
  enableAutoRefresh(functionDetail: FunctionDetail, interval: number): void {
    this.schedulerService.addTask(`table-refresh-${functionDetail.id}`, interval, () => {
      this.loadData(functionDetail)
    })
  }

  /**
   * 禁用自动刷新
   */
  disableAutoRefresh(functionDetail: FunctionDetail): void {
    this.schedulerService.removeTask(`table-refresh-${functionDetail.id}`)
  }
}
```

**代码位置**：
- Domain Service: `src/architecture/domain/services/SchedulerDomainService.ts`

---

### 6.4 场景 4：新增数据导出功能

**问题**：表格需要支持导出为 Excel。

**解决方案**：

1. 创建导出工具：

```typescript
// src/architecture/runtime/utils/export.ts
import * as XLSX from 'xlsx'

export class ExportUtil {
  /**
   * 导出为 Excel
   */
  static exportToExcel(data: any[], filename: string): void {
    const worksheet = XLSX.utils.json_to_sheet(data)
    const workbook = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(workbook, worksheet, 'Sheet1')
    XLSX.writeFile(workbook, `${filename}.xlsx`)
  }
  
  /**
   * 导出为 CSV
   */
  static exportToCSV(data: any[], filename: string): void {
    const worksheet = XLSX.utils.json_to_sheet(data)
    const csv = XLSX.utils.sheet_to_csv(worksheet)
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    link.download = `${filename}.csv`
    link.click()
  }
}
```

2. 在 `TableDomainService` 中添加导出方法：

```typescript
export class TableDomainService {
  /**
   * 导出表格数据
   */
  exportData(format: 'excel' | 'csv'): void {
    const state = this.stateManager.getState()
    const data = state.data
    
    // 提取所有行的数据（只保留可见字段）
    const exportData = data.map(row => {
      const rowData: Record<string, any> = {}
      this.fields.forEach(field => {
        rowData[field.name] = row[field.code]?.display || row[field.code]?.raw
      })
      return rowData
    })
    
    // 导出
    if (format === 'excel') {
      ExportUtil.exportToExcel(exportData, 'table-data')
    } else {
      ExportUtil.exportToCSV(exportData, 'table-data')
    }
  }
}
```

3. 在 UI 中使用：

```vue
<el-button @click="handleExport('excel')">导出 Excel</el-button>

<script setup>
function handleExport(format: 'excel' | 'csv') {
  tableService.exportData(format)
}
</script>
```

**代码位置**：
- 工具类: `src/architecture/runtime/utils/export.ts`
- Domain Service: `src/architecture/domain/services/TableDomainService.ts`

---

## 七、最佳实践

### 7.1 代码组织原则

1. **按层级组织代码**
   - Presentation Layer: UI 相关
   - Application Layer: 流程编排
   - Domain Layer: 业务逻辑
   - Infrastructure Layer: 技术实现

2. **按功能模块组织代码**
   - 同一功能的代码放在一起
   - 例如：FormDomainService, FormApplicationService, FormView

3. **遵循单一职责原则**
   - 每个类/模块只负责一件事
   - 例如：FormDomainService 只负责表单业务逻辑，不负责 API 调用

4. **依赖倒置**
   - 高层模块依赖抽象（接口），不依赖具体实现
   - 例如：FormDomainService 依赖 IStateManager，不依赖 FormStateManager

### 7.2 命名规范

| 类型 | 命名规则 | 示例 |
|------|----------|------|
| Vue 组件 | PascalCase + Widget/View 后缀 | `InputWidget.vue`, `FormView.vue` |
| 服务类 | PascalCase + Service 后缀 | `FormDomainService`, `ActionLogApplicationService` |
| 接口 | PascalCase + I 前缀 | `IStateManager`, `IEventBus` |
| 类型 | PascalCase | `FieldConfig`, `FieldValue` |
| 函数 | camelCase | `getSubmitData()`, `handleUpdate()` |
| 变量 | camelCase | `fieldPath`, `modelValue` |
| 常量 | UPPER_SNAKE_CASE | `MAX_FILE_SIZE` |
| 事件名 | camelCase | `FormEvent.fieldUpdated` |

### 7.3 注释规范

```typescript
/**
 * FormDomainService - 表单领域服务
 * 
 * 职责：
 * - 表单初始化
 * - 表单验证
 * - 表单提交数据提取
 * 
 * 依赖：
 * - IStateManager<FormState>: 状态管理
 * - IEventBus: 事件总线
 * - ValidationEngine: 验证引擎
 * 
 * 使用示例：
 * ```typescript
 * const formService = new FormDomainService(stateManager, eventBus, validationEngine)
 * formService.initializeForm(fields, initialData)
 * ```
 */
export class FormDomainService {
  /**
   * 初始化表单
   * @param fields 字段配置列表
   * @param initialData 初始数据（可选）
   */
  initializeForm(fields: FieldConfig[], initialData?: Record<string, any>): void {
    // ...
  }
}
```

### 7.4 错误处理

```typescript
try {
  // 业务逻辑
  const result = await someOperation()
} catch (error) {
  // 记录错误
  Logger.error('ServiceName', '操作失败', error)
  
  // 用户友好的错误提示
  ElMessage.error('操作失败，请稍后重试')
  
  // 可选：上报错误
  // ErrorReporter.report(error)
  
  // 可选：重新抛出错误（如果需要上层处理）
  // throw error
}
```

### 7.5 性能优化

1. **使用 v-memo 优化列表渲染**

```vue
<div v-for="item in items" :key="item.id" v-memo="[item.id, item.name]">
  {{ item.name }}
</div>
```

2. **使用 computed 缓存计算结果**

```typescript
const filteredData = computed(() => {
  return data.value.filter(item => item.status === 'active')
})
```

3. **使用 debounce 防抖**

```typescript
import { debounce } from 'lodash-es'

const handleSearch = debounce((keyword: string) => {
  // 搜索逻辑
}, 300)
```

4. **懒加载组件**

```typescript
const HeavyComponent = defineAsyncComponent(() =>
  import('@/components/HeavyComponent.vue')
)
```

---

## 八、常见问题

### Q1: 新增组件时，应该放在哪里？

**A**: 根据组件类型决定：

- **表单组件（Widget）** → `src/architecture/presentation/widgets/`
- **跨业务共享组件** → `src/shared/components/`
- **页面组件（View）** → `src/architecture/presentation/views/`
- **Widget 默认值/校验等运行时能力** → `src/architecture/runtime/widgetRuntime/`

### Q2: 新增业务逻辑时，应该放在哪一层？

**A**: 根据职责决定：

- **核心业务逻辑** → Domain Layer (`src/architecture/domain/services/`)
- **流程编排** → Application Layer (`src/architecture/application/services/`)
- **UI 交互** → Presentation Layer (`src/architecture/presentation/views/`)
- **技术实现** → Infrastructure Layer (`src/architecture/infrastructure/`)
- **跨业务共享但不属于主页面流程的通用能力** → `src/architecture/runtime/` / `src/shared/` / `src/architecture/runtime/utils/`

### Q3: 如何在组件之间通信？

**A**: 使用事件总线（EventBus）：

```typescript
// 发布事件
eventBus.emit(FormEvent.fieldUpdated, { fieldCode: 'name', value: '张三' })

// 订阅事件
eventBus.on(FormEvent.fieldUpdated, (data) => {
  console.log('字段更新', data)
})
```

### Q4: 如何访问全局状态？

**A**: 使用 Pinia Store：

```typescript
import { useFormDataStore } from '@/architecture/runtime/stores/formData'

const formDataStore = useFormDataStore()
const value = formDataStore.getValue('business_info.industry')
```

### Q5: 新增的组件如何支持所有渲染模式？

**A**: 使用 `v-if` 根据 `mode` 渲染不同的 UI：

```vue
<template>
  <div v-if="mode === 'edit'"><!-- 编辑模式 --></div>
  <div v-else-if="mode === 'response'"><!-- 响应模式 --></div>
  <div v-else-if="mode === 'table-cell'"><!-- 表格模式 --></div>
  <div v-else-if="mode === 'detail'"><!-- 详情模式 --></div>
  <div v-else-if="mode === 'search'"><!-- 搜索模式 --></div>
</template>
```

### Q6: 如何调试？

**A**: 使用 Logger：

```typescript
import { Logger } from '@/architecture/runtime/utils/logger'

Logger.debug('ComponentName', '调试信息', data)
Logger.info('ComponentName', '信息', data)
Logger.warn('ComponentName', '警告', data)
Logger.error('ComponentName', '错误', error)
```

### Q7: 如何确保代码质量？

**A**: 遵循以下原则：

1. ✅ 遵循 SOLID 原则
2. ✅ 遵循四层架构
3. ✅ 遵循命名规范
4. ✅ 添加类型注解
5. ✅ 添加注释说明
6. ✅ 进行错误处理
7. ✅ 编写单元测试（推荐）

### Q8: 如何处理复杂的嵌套数据？

**A**: 使用 FieldExtractorRegistry 自动处理：

```typescript
// 系统会自动递归提取嵌套数据，无需手动处理
const submitData = formDataStore.getSubmitData(fields)

// 支持任意深度的嵌套：
// { level1: { level2: { level3: { ... } } } }
```

---

## 🎯 核心要点总结

### ✅ 新增组件

1. 创建 Vue 组件（支持所有 mode）
2. 注册到 WidgetComponentFactory
3. （可选）创建并注册 FieldExtractor

### ✅ 新增业务逻辑

1. 确定归属层级（Domain / Application / Presentation）
2. 创建 Service 类
3. 在 ServiceFactory 中注册
4. 在需要的地方使用

### ✅ 新增功能模块

1. Domain Layer: 接口 + 服务 + 类型
2. Infrastructure Layer: StateManager + Factory 注册
3. Application Layer: Application Service
4. Presentation Layer: View + Composables
5. 注册路由

### ✅ 遵循原则

- 四层架构：Presentation → Application → Domain → Infrastructure
- SOLID 原则：SRP, OCP, LSP, ISP, DIP
- 设计模式：策略、工厂、适配器、观察者

---

## 📖 相关文档

- [表单值提取逻辑分析报告](./docs/表单值提取逻辑分析报告.md)
- [值提取和渲染机制完整性分析](./docs/值提取和渲染机制完整性分析.md)

---

## 🚀 快速开始

```bash
# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build

# 运行测试
npm run test

# 代码检查
npm run lint
```

---

**最后更新**: 2025-11-29

**维护者**: AI Agent OS 团队

**如有疑问，请参考本文档或查看相关源码。**
