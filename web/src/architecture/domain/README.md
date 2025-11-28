# Domain Layer (领域层)

## 职责

- 核心业务逻辑
- 定义业务规则和领域模型
- 定义接口（依赖倒置）

## 目录结构

- `services/`：Domain Services
  - `WorkspaceDomainService.ts`：工作空间业务逻辑
  - `FormDomainService.ts`：表单业务逻辑
  - `TableDomainService.ts`：表格业务逻辑
- `interfaces/`：接口定义
  - `IEventBus.ts`：事件总线接口
  - `IStateManager.ts`：状态管理接口
  - `IApiClient.ts`：API 客户端接口
  - `IFunctionLoader.ts`：函数加载器接口
  - `ICacheManager.ts`：缓存管理接口
- `types/`：领域类型定义
  - `FieldConfig.ts`：字段配置类型
  - `FieldValue.ts`：字段值类型
  - `FunctionDetail.ts`：函数详情类型

## 特点

- 不依赖 Infrastructure Layer 的具体实现
- 只依赖接口，实现依赖倒置
- 包含核心业务逻辑

## 使用示例

```typescript
import { FormDomainService } from '@/architecture/domain/services/FormDomainService'
import type { IStateManager } from '@/architecture/domain/interfaces/IStateManager'
import type { IEventBus } from '@/architecture/domain/interfaces/IEventBus'

const domainService = new FormDomainService(stateManager, eventBus)
```

