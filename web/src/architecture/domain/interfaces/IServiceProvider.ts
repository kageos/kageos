/**
 * IServiceProvider - 服务提供者接口
 * 
 * 职责：定义服务提供者的标准接口，实现依赖倒置原则
 * 
 * 特点：
 * - Presentation Layer 只依赖此接口，不依赖具体实现
 * - 可以轻松替换实现，提高可测试性
 * - 遵循依赖倒置原则（DIP）
 */

import type { IStateManager } from './IStateManager'
import type { IEventBus } from './IEventBus'
import type { IApiClient } from './IApiClient'
import type { ICacheManager } from './ICacheManager'
import type { IFunctionLoader } from './IFunctionLoader'
import type { IServiceTreeLoader } from './IServiceTreeLoader'

// 导入状态类型
import type { FormState } from '../services/FormDomainService'
import type { TableState } from '../services/TableDomainService'
import type { WorkspaceState } from '../services/WorkspaceDomainService'

// 导入服务类型（使用具体类型，因为 Domain Services 是业务逻辑层）
import type { FormDomainService } from '../services/FormDomainService'
import type { TableDomainService } from '../services/TableDomainService'
import type { WorkspaceDomainService } from '../services/WorkspaceDomainService'
import type { FormApplicationService } from '../../application/services/FormApplicationService'
import type { TableApplicationService } from '../../application/services/TableApplicationService'
import type { WorkspaceApplicationService } from '../../application/services/WorkspaceApplicationService'

/**
 * 服务提供者接口
 * 
 * 🔥 依赖倒置原则：Presentation Layer 只依赖此接口，不依赖 ServiceFactory 具体实现
 */
export interface IServiceProvider {
  // ========== State Managers ==========
  /**
   * 获取表单状态管理器
   */
  getFormStateManager(): IStateManager<FormState>

  /**
   * 获取表格状态管理器
   */
  getTableStateManager(): IStateManager<TableState>

  /**
   * 获取工作空间状态管理器
   */
  getWorkspaceStateManager(): IStateManager<WorkspaceState>

  // ========== Domain Services ==========
  /**
   * 获取表单领域服务
   */
  getFormDomainService(): FormDomainService

  /**
   * 获取表格领域服务
   */
  getTableDomainService(): TableDomainService

  /**
   * 获取工作空间领域服务
   */
  getWorkspaceDomainService(): WorkspaceDomainService

  // ========== Application Services ==========
  /**
   * 获取表单应用服务
   */
  getFormApplicationService(): FormApplicationService

  /**
   * 获取表格应用服务
   */
  getTableApplicationService(): TableApplicationService

  /**
   * 获取工作空间应用服务
   */
  getWorkspaceApplicationService(): WorkspaceApplicationService

  // ========== Infrastructure Services ==========
  /**
   * 获取事件总线
   */
  getEventBus(): IEventBus

  /**
   * 获取 API 客户端
   */
  getApiClient(): IApiClient

  /**
   * 获取缓存管理器
   */
  getCacheManager(): ICacheManager

  /**
   * 获取函数加载器
   */
  getFunctionLoader(): IFunctionLoader

  /**
   * 获取服务树加载器
   */
  getServiceTreeLoader(): IServiceTreeLoader
}

