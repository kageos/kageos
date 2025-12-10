# Infrastructure Layer (基础设施层)

## 职责

- 技术实现（Pinia、EventBus、API 调用等）
- 实现 Domain Layer 定义的接口
- 提供技术能力给上层使用

## 目录结构

- `eventBus/`：事件总线实现
  - `EventBusImpl.ts`：内存事件总线实现
- `stateManager/`：状态管理实现
  - `StateManagerImpl.ts`：基于 Pinia 的状态管理实现
  - `WorkspaceStateManager.ts`：工作空间状态管理
  - `FormStateManager.ts`：表单状态管理
- `apiClient/`：API 客户端实现
  - `ApiClientImpl.ts`：基于 axios 的 API 客户端实现
- `functionLoader/`：函数加载器实现
  - `FunctionLoaderImpl.ts`：函数加载器实现（带防抖和去重）
- `cacheManager/`：缓存管理实现
  - `CacheManagerImpl.ts`：内存缓存实现
- `widgetRegistry/`：Widget 注册表
  - `WidgetComponentFactory.ts`：Widget 组件工厂（从 `core/factories-v2` 迁移）

## 特点

- 实现 Domain Layer 定义的接口
- 可以轻松替换实现（例如：从内存缓存切换到 Redis 缓存）
- 提供技术能力，不包含业务逻辑

## 使用示例

```typescript
import { EventBusImpl } from '@/architecture/infrastructure/eventBus/EventBusImpl'
import type { IEventBus } from '@/architecture/domain/interfaces/IEventBus'

const eventBus: IEventBus = new EventBusImpl()
```

