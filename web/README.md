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
- ✅ **四层架构设计**：Presentation → Application → Domain → Infrastructure
- ✅ **完全遵循 SOLID 原则**：高内聚低耦合，易于扩展和维护
- ✅ **策略模式 + 工厂模式**：支持任意组件类型和数据结构
- ✅ **事件驱动架构**：组件间通过事件总线解耦

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

### 2.2 核心原则（SOLID）

| 原则 | 说明 | 体现 |
|------|------|------|
| **SRP** (单一职责原则) | 每个类/模块只负责一件事 | Domain Service 只负责业务逻辑，不负责 UI 渲染 |
| **OCP** (开闭原则) | 对扩展开放，对修改封闭 | 新增组件只需注册，无需修改现有代码 |
| **LSP** (里氏替换原则) | 子类可以替换父类 | 所有提取器实现 IFieldExtractor 接口 |
| **ISP** (接口隔离原则) | 接口设计简洁 | IStateManager、IEventBus 等接口职责明确 |
| **DIP** (依赖倒置原则) | 高层模块依赖抽象 | Domain Service 依赖 IStateManager，不依赖具体实现 |

### 2.3 设计模式

- **策略模式**：FieldExtractorRegistry（根据字段类型选择不同的提取器）
- **工厂模式**：WidgetComponentFactory（根据组件类型创建不同的组件）
- **适配器模式**：FormStateManagerAdapter（适配不同的状态管理接口）
- **观察者模式**：EventBus（事件发布订阅）
- **单例模式**：Pinia Store（全局状态管理）

---

## 三、目录结构

### 3.1 完整目录树

```
web/
├── src/
│   ├── architecture/                    # 🏗️ 新架构（四层架构）
│   │   ├── presentation/                # 表示层
│   │   │   ├── views/                   # 页面组件
│   │   │   │   ├── WorkspaceView.vue   # 工作空间主页
│   │   │   │   ├── FormView.vue        # 表单页面
│   │   │   │   ├── TableView.vue       # 表格页面
│   │   │   │   └── DetailView.vue      # 详情页面
│   │   │   ├── widgets/                 # 表示层组件（高级组件）
│   │   │   │   └── WidgetComponent.vue # 通用组件包装器
│   │   │   └── composables/             # 组合式函数
│   │   │       ├── useFormInitialization.ts
│   │   │       ├── useTableInitialization.ts
│   │   │       └── useWorkspaceInitialization.ts
│   │   ├── application/                 # 应用层
│   │   │   └── services/                # 应用服务（业务流程编排）
│   │   │       ├── FormApplicationService.ts      # 表单应用服务
│   │   │       ├── TableApplicationService.ts     # 表格应用服务
│   │   │       └── WorkspaceApplicationService.ts # 工作空间应用服务
│   │   ├── domain/                      # 领域层
│   │   │   ├── services/                # 领域服务（核心业务逻辑）
│   │   │   │   ├── FormDomainService.ts      # 表单领域服务
│   │   │   │   ├── TableDomainService.ts     # 表格领域服务
│   │   │   │   └── WorkspaceDomainService.ts # 工作空间领域服务
│   │   │   ├── interfaces/              # 抽象接口（依赖倒置）
│   │   │   │   ├── IStateManager.ts     # 状态管理接口
│   │   │   │   ├── IEventBus.ts         # 事件总线接口
│   │   │   │   ├── IApiClient.ts        # API 客户端接口
│   │   │   │   ├── IServiceTreeLoader.ts # 服务树加载器接口
│   │   │   │   └── index.ts             # 统一导出
│   │   │   └── types/                   # 类型定义
│   │   │       └── index.ts
│   │   └── infrastructure/              # 基础设施层
│   │       ├── api/                     # API 实现
│   │       │   └── ApiClientImpl.ts
│   │       ├── eventBus/                # 事件总线实现
│   │       │   └── EventBusImpl.ts
│   │       ├── stateManager/            # 状态管理器实现
│   │       │   ├── StateManagerImpl.ts      # 通用状态管理器
│   │       │   ├── FormStateManager.ts      # 表单状态管理器
│   │       │   ├── TableStateManager.ts     # 表格状态管理器
│   │       │   └── WorkspaceStateManager.ts # 工作空间状态管理器
│   │       ├── serviceTreeLoader/       # 服务树加载器实现
│   │       │   └── ServiceTreeLoaderImpl.ts
│   │       └── factories/               # 工厂类
│   │           ├── ServiceFactory.ts    # 服务工厂（创建 Domain/Application Service）
│   │           └── WidgetComponentFactory.ts # 组件工厂（已移到 core/factories-v2）
│   ├── core/                            # 🎯 核心系统（独立于架构层）
│   │   ├── widgets-v2/                  # 组件库（新版本）
│   │   │   ├── components/              # 所有 UI 组件
│   │   │   │   ├── InputWidget.vue      # 文本输入框
│   │   │   │   ├── SelectWidget.vue     # 下拉选择
│   │   │   │   ├── MultiSelectWidget.vue # 多选
│   │   │   │   ├── NumberWidget.vue     # 数字输入
│   │   │   │   ├── FormWidget.vue       # 表单（form/struct）
│   │   │   │   ├── TableWidget.vue      # 表格（table/array）
│   │   │   │   ├── FilesWidget.vue      # 文件上传
│   │   │   │   └── ...                  # 其他组件
│   │   │   └── composables/             # 组件相关的组合式函数
│   │   │       ├── useTableEditMode.ts  # 表格编辑模式
│   │   │       └── ...
│   │   ├── factories-v2/                # 工厂（新版本）
│   │   │   └── index.ts                 # WidgetComponentFactory 注册所有组件
│   │   ├── stores-v2/                   # Pinia Store（新版本）
│   │   │   ├── formData.ts              # 表单数据 Store
│   │   │   ├── tableData.ts             # 表格数据 Store
│   │   │   └── extractors/              # 值提取器（策略模式）
│   │   │       ├── FieldExtractor.ts           # 提取器接口
│   │   │       ├── FieldExtractorRegistry.ts   # 提取器注册表
│   │   │       ├── BasicFieldExtractor.ts      # 基础字段提取器
│   │   │       ├── MultiSelectFieldExtractor.ts # 多选字段提取器
│   │   │       ├── FormFieldExtractor.ts       # 表单字段提取器
│   │   │       └── TableFieldExtractor.ts      # 表格字段提取器
│   │   ├── renderers-v2/                # 渲染器（新版本）
│   │   │   └── FormRenderer.vue         # 表单渲染器
│   │   ├── utils/                       # 工具函数
│   │   │   ├── logger.ts                # 日志工具
│   │   │   └── ...
│   │   └── validation/                  # 验证引擎
│   │       └── ValidationEngine.ts
│   ├── components/                      # 🧩 通用组件（不属于 widgets）
│   │   ├── TableRenderer.vue            # 表格渲染器
│   │   ├── FileUpload.vue               # 文件上传组件
│   │   ├── SearchInput.vue              # 搜索输入框
│   │   └── ...
│   ├── views/                           # 📄 页面（旧架构，保留但不推荐使用）
│   │   └── layouts/
│   │       └── MainLayout.vue           # 主布局
│   ├── router/                          # 🚦 路由
│   │   └── index.ts
│   ├── styles/                          # 🎨 样式
│   │   └── theme.scss
│   ├── types/                           # 📝 类型定义
│   │   └── field.ts                     # FieldConfig, FieldValue 等
│   ├── utils/                           # 🔧 工具函数
│   │   └── route.ts                     # 路由工具
│   ├── App.vue                          # 应用入口
│   └── main.ts                          # 应用启动
├── docs/                                # 📚 文档
│   ├── 新架构设计方案.md
│   ├── 新架构实现总结.md
│   ├── 表单值提取逻辑分析报告.md
│   └── 值提取和渲染机制完整性分析.md
└── README.md                            # 本文档
```

### 3.2 目录职责说明

#### 🏗️ architecture/ - 架构目录（新架构）

**作用**：实现四层架构，所有业务逻辑都在这里。

| 子目录 | 职责 | 示例 |
|--------|------|------|
| `presentation/` | UI 渲染、用户交互 | WorkspaceView.vue, FormView.vue |
| `application/` | 业务流程编排 | FormApplicationService |
| `domain/` | 核心业务逻辑 | FormDomainService, IStateManager |
| `infrastructure/` | 技术实现 | ApiClientImpl, EventBusImpl |

**何时添加代码**：
- 新增页面 → `presentation/views/`
- 新增业务流程 → `application/services/`
- 新增业务逻辑 → `domain/services/`
- 新增基础设施 → `infrastructure/`

#### 🎯 core/ - 核心系统

**作用**：提供可复用的核心功能，独立于具体业务。

| 子目录 | 职责 | 示例 |
|--------|------|------|
| `widgets-v2/` | UI 组件库 | InputWidget.vue, SelectWidget.vue |
| `factories-v2/` | 工厂类 | WidgetComponentFactory |
| `stores-v2/` | 状态管理 | formData.ts, extractors/ |
| `renderers-v2/` | 渲染器 | FormRenderer.vue |
| `utils/` | 工具函数 | logger.ts |
| `validation/` | 验证引擎 | ValidationEngine.ts |

**何时添加代码**：
- 新增 UI 组件 → `widgets-v2/components/`
- 新增提取器 → `stores-v2/extractors/`
- 新增工具函数 → `utils/`

#### 🧩 components/ - 通用组件

**作用**：存放不属于 widgets 的通用组件。

**何时添加代码**：
- 新增非表单组件（如布局、对话框等）→ `components/`

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
WidgetComponentFactory.getComponent(widget.type)
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

文件位置：`src/core/widgets-v2/components/RichTextEditorWidget.vue`

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
import type { FieldValue } from '@/types/field'

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

文件位置：`src/core/factories-v2/index.ts`

```typescript
import RichTextEditorWidget from '@/core/widgets-v2/components/RichTextEditorWidget.vue'

// 注册组件
widgetFactory.register('rich_text_editor', RichTextEditorWidget)
```

#### 步骤 3：（可选）创建提取器

**如果需要特殊提取逻辑**（通常简单组件不需要）：

文件位置：`src/core/stores-v2/extractors/RichTextEditorFieldExtractor.ts`

```typescript
import type { IFieldExtractor, FieldExtractorRegistry } from './FieldExtractor'
import type { FieldConfig } from '@/types/field'

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

文件位置：`src/core/stores-v2/extractors/FieldExtractorRegistry.ts`

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

**场景**：新增一个应用权限校验功能。

#### 步骤 1：确定逻辑归属

**问题**：这个功能是否属于某个已有的 Domain Service？
- 如果是表单相关 → `FormDomainService`
- 如果是表格相关 → `TableDomainService`
- 如果是工作空间相关 → `WorkspaceDomainService`
- 如果是新的领域 → 创建新的 Domain Service

**权限校验是跨领域的功能**，应该创建独立的 `PermissionDomainService`。

#### 步骤 2：定义接口（Domain Layer）

文件位置：`src/architecture/domain/interfaces/IPermission.ts`

```typescript
/**
 * 权限接口
 */
export interface IPermission {
  /**
   * 检查应用权限
   * @param appId 应用 ID
   * @param action 动作（如 'read', 'write', 'delete'）
   * @returns 是否有权限
   */
  checkAppPermission(appId: number, action: string): Promise<boolean>
  
  /**
   * 检查功能权限
   * @param functionId 功能 ID
   * @param action 动作
   * @returns 是否有权限
   */
  checkFunctionPermission(functionId: number, action: string): Promise<boolean>
}
```

导出接口：
```typescript
// src/architecture/domain/interfaces/index.ts
export * from './IPermission'
```

#### 步骤 3：创建 Domain Service

文件位置：`src/architecture/domain/services/PermissionDomainService.ts`

```typescript
import type { IStateManager } from '../interfaces/IStateManager'
import type { IEventBus } from '../interfaces/IEventBus'
import type { IApiClient } from '../interfaces/IApiClient'
import { Logger } from '@/core/utils/logger'

/**
 * 权限状态
 */
interface PermissionState {
  permissions: Map<string, boolean>  // 权限缓存
}

/**
 * 权限领域服务
 * 
 * 职责：
 * - 权限检查逻辑
 * - 权限缓存管理
 * - 权限相关的业务规则
 */
export class PermissionDomainService {
  constructor(
    private stateManager: IStateManager<PermissionState>,
    private eventBus: IEventBus,
    private apiClient: IApiClient
  ) {}

  /**
   * 检查应用权限
   */
  async checkAppPermission(appId: number, action: string): Promise<boolean> {
    const cacheKey = `app:${appId}:${action}`
    const state = this.stateManager.getState()
    
    // 从缓存中读取
    if (state.permissions.has(cacheKey)) {
      return state.permissions.get(cacheKey)!
    }
    
    // 调用 API 检查权限
    try {
      const response = await this.apiClient.get(`/api/v1/permissions/check`, {
        resource_type: 'app',
        resource_id: appId,
        action
      })
      
      const hasPermission = response?.has_permission || false
      
      // 缓存结果
      const newPermissions = new Map(state.permissions)
      newPermissions.set(cacheKey, hasPermission)
      this.stateManager.setState({ permissions: newPermissions })
      
      return hasPermission
    } catch (error) {
      Logger.error('PermissionDomainService', '权限检查失败', error)
      return false
    }
  }

  /**
   * 检查功能权限
   */
  async checkFunctionPermission(functionId: number, action: string): Promise<boolean> {
    const cacheKey = `function:${functionId}:${action}`
    const state = this.stateManager.getState()
    
    // 从缓存中读取
    if (state.permissions.has(cacheKey)) {
      return state.permissions.get(cacheKey)!
    }
    
    // 调用 API 检查权限
    try {
      const response = await this.apiClient.get(`/api/v1/permissions/check`, {
        resource_type: 'function',
        resource_id: functionId,
        action
      })
      
      const hasPermission = response?.has_permission || false
      
      // 缓存结果
      const newPermissions = new Map(state.permissions)
      newPermissions.set(cacheKey, hasPermission)
      this.stateManager.setState({ permissions: newPermissions })
      
      return hasPermission
    } catch (error) {
      Logger.error('PermissionDomainService', '权限检查失败', error)
      return false
    }
  }

  /**
   * 清空权限缓存
   */
  clearCache(): void {
    this.stateManager.setState({ permissions: new Map() })
  }
}
```

#### 步骤 4：创建 State Manager（Infrastructure Layer）

文件位置：`src/architecture/infrastructure/stateManager/PermissionStateManager.ts`

```typescript
import { ref } from 'vue'
import { StateManagerImpl } from './StateManagerImpl'
import type { IStateManager } from '../../domain/interfaces/IStateManager'

interface PermissionState {
  permissions: Map<string, boolean>
}

export class PermissionStateManager extends StateManagerImpl<PermissionState> implements IStateManager<PermissionState> {
  constructor() {
    const initialState = ref<PermissionState>({
      permissions: new Map()
    })
    super(initialState)
  }
}
```

#### 步骤 5：创建 Application Service（Application Layer）

文件位置：`src/architecture/application/services/PermissionApplicationService.ts`

```typescript
import { PermissionDomainService } from '../../domain/services/PermissionDomainService'
import type { IEventBus } from '../../domain/interfaces/IEventBus'

/**
 * 权限应用服务
 * 
 * 职责：
 * - 协调权限检查流程
 * - 处理权限相关的业务场景
 */
export class PermissionApplicationService {
  constructor(
    private domainService: PermissionDomainService,
    private eventBus: IEventBus
  ) {}

  /**
   * 检查应用权限（供外部调用）
   */
  async checkAppPermission(appId: number, action: string): Promise<boolean> {
    return await this.domainService.checkAppPermission(appId, action)
  }

  /**
   * 检查功能权限（供外部调用）
   */
  async checkFunctionPermission(functionId: number, action: string): Promise<boolean> {
    return await this.domainService.checkFunctionPermission(functionId, action)
  }

  /**
   * 清空权限缓存（供外部调用）
   */
  clearPermissionCache(): void {
    this.domainService.clearCache()
  }
}
```

#### 步骤 6：注册到 ServiceFactory

文件位置：`src/architecture/infrastructure/factories/ServiceFactory.ts`

```typescript
import { PermissionStateManager } from '../stateManager/PermissionStateManager'
import { PermissionDomainService } from '../../domain/services/PermissionDomainService'
import { PermissionApplicationService } from '../../application/services/PermissionApplicationService'

class ServiceFactory {
  // ... 其他服务
  
  private permissionStateManager?: PermissionStateManager
  private permissionDomainService?: PermissionDomainService
  private permissionApplicationService?: PermissionApplicationService
  
  // 获取权限应用服务
  getPermissionApplicationService(): PermissionApplicationService {
    if (!this.permissionApplicationService) {
      const stateManager = this.getPermissionStateManager()
      const domainService = this.getPermissionDomainService()
      this.permissionApplicationService = new PermissionApplicationService(
        domainService,
        this.eventBus
      )
    }
    return this.permissionApplicationService
  }
  
  private getPermissionDomainService(): PermissionDomainService {
    if (!this.permissionDomainService) {
      const stateManager = this.getPermissionStateManager()
      this.permissionDomainService = new PermissionDomainService(
        stateManager,
        this.eventBus,
        this.apiClient
      )
    }
    return this.permissionDomainService
  }
  
  private getPermissionStateManager(): PermissionStateManager {
    if (!this.permissionStateManager) {
      this.permissionStateManager = new PermissionStateManager()
    }
    return this.permissionStateManager
  }
}
```

#### 步骤 7：使用

在 Vue 组件中使用：

```vue
<script setup lang="ts">
import { serviceFactory } from '@/architecture/infrastructure/factories/ServiceFactory'

const permissionService = serviceFactory.getPermissionApplicationService()

async function checkPermission() {
  const hasPermission = await permissionService.checkAppPermission(120, 'write')
  if (!hasPermission) {
    ElMessage.error('没有权限')
    return
  }
  // 继续操作
}
</script>
```

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

文件位置：`src/router/index.ts`

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
   - 在 `src/router/index.ts` 中添加路由

6. **（可选）新增组件**
   - 在 `src/core/widgets-v2/components/` 中创建工作流相关的组件

---

## 六、典型场景示例

### 6.1 场景 1：新增权限校验功能

**问题**：用户打开某个功能时，需要检查是否有权限。

**解决方案**：

1. 创建 `PermissionDomainService`（如 5.2 所示）
2. 在 `FormApplicationService.submitForm()` 中添加权限检查：

```typescript
async submitForm(functionDetail: FunctionDetail): Promise<any> {
  // 🔥 权限检查
  const permissionService = serviceFactory.getPermissionApplicationService()
  const hasPermission = await permissionService.checkFunctionPermission(
    functionDetail.id,
    'execute'
  )
  
  if (!hasPermission) {
    throw new Error('没有执行权限')
  }
  
  // 继续提交流程
  // ...
}
```

**代码位置**：
- Domain Service: `src/architecture/domain/services/PermissionDomainService.ts`
- Application Service: `src/architecture/application/services/PermissionApplicationService.ts`
- State Manager: `src/architecture/infrastructure/stateManager/PermissionStateManager.ts`

---

### 6.2 场景 2：新增日志记录功能

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
// src/core/utils/export.ts
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
- 工具类: `src/core/utils/export.ts`
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
| 服务类 | PascalCase + Service 后缀 | `FormDomainService`, `PermissionApplicationService` |
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

- **表单组件（Widget）** → `src/core/widgets-v2/components/`
- **通用组件（非 Widget）** → `src/components/`
- **页面组件（View）** → `src/architecture/presentation/views/`

### Q2: 新增业务逻辑时，应该放在哪一层？

**A**: 根据职责决定：

- **核心业务逻辑** → Domain Layer (`src/architecture/domain/services/`)
- **流程编排** → Application Layer (`src/architecture/application/services/`)
- **UI 交互** → Presentation Layer (`src/architecture/presentation/views/`)
- **技术实现** → Infrastructure Layer (`src/architecture/infrastructure/`)

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
import { useFormDataStore } from '@/core/stores-v2/formData'

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
import { Logger } from '@/core/utils/logger'

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

- [新架构设计方案](./docs/新架构设计方案.md)
- [新架构实现总结](./docs/新架构实现总结.md)
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
