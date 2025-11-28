# Application Layer (应用层)

## 职责

- 业务流程编排
- 监听事件，调用 Domain Services
- 协调多个 Domain Services 完成业务场景

## 目录结构

- `services/`：Application Services
  - `WorkspaceApplicationService.ts`：工作空间业务流程
  - `FormApplicationService.ts`：表单业务流程
  - `TableApplicationService.ts`：表格业务流程

## 特点

- 不包含业务逻辑，只负责编排
- 通过事件监听和触发实现流程控制
- 依赖 Domain Layer 接口

## 使用示例

```typescript
import { WorkspaceApplicationService } from '@/architecture/application/services/WorkspaceApplicationService'
import { WorkspaceDomainService } from '@/architecture/domain/services/WorkspaceDomainService'
import { eventBus } from '@/architecture/infrastructure/eventBus'

const domainService = new WorkspaceDomainService(...)
const applicationService = new WorkspaceApplicationService(domainService, eventBus)
```

