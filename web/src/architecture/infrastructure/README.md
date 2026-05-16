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
- `stores/`：应用级 Pinia Store
  - `auth.ts`、`userInfo/`、`theme.ts` 等全局状态
- `apiClient/`：API 客户端实现
  - `ApiClientImpl.ts`：基于 axios 的 API 客户端实现
  - `request.ts`、`authSession.ts`、`apiError.ts`：HTTP 请求、鉴权刷新与响应解包
- `api/`：平台业务接口封装
  - `app.ts`、`service-tree.ts`、`workspace.ts` 等按后端资源域组织
- `formGateway/`：表单提交网关实现
  - 封装表单提交对应的 HTTP 路由和方法分发
- `tableGateway/`：表格数据网关实现
  - 封装表格查询、创建、更新、删除对应的 HTTP 路由
- `upload/`：文件上传基础设施
  - 预签名 URL、表单上传、SDK 上传适配器及上传类型
- `config/`：前端运行时配置
  - `features.ts`、`runtime.ts`、`terminology.ts` 等环境与产品开关配置
- `functionLoader/`：函数加载器实现
  - `FunctionLoaderImpl.ts`：函数加载器实现（带防抖和去重）
- `cacheManager/`：缓存管理实现
  - `CacheManagerImpl.ts`：内存缓存实现

## 特点

- 实现 Domain Layer 定义的接口
- 可以轻松替换实现（例如：从内存缓存切换到 Redis 缓存）
- 提供技术能力，不包含业务逻辑
- 运行期 Pinia store、HTTP、上传、缓存等技术实现收口在 infrastructure；导航端口从 `shared/routing/navigation` 引用
- 不直接依赖 `presentation` 页面或组件；需要跳转时通过运行时导航端口完成

## 使用示例

```typescript
import { EventBusImpl } from '@/architecture/infrastructure/eventBus/EventBusImpl'
import type { IEventBus } from '@/architecture/domain/interfaces/IEventBus'

const eventBus: IEventBus = new EventBusImpl()
```
