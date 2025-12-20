/**
 * WorkspaceStateManager - 工作空间状态管理实现
 * 
 * 职责：基于响应式对象实现工作空间状态管理
 * 
 * 特点：
 * - 使用 StateManagerImpl 作为基础
 * - 提供工作空间特定的状态管理
 */

import { reactive } from 'vue'
import { StateManagerImpl } from './StateManagerImpl'
import type { IStateManager } from '../../domain/interfaces/IStateManager'
import type { WorkspaceState, App, ServiceTree } from '../../domain/services/WorkspaceDomainService'
import type { FunctionDetail } from '../../domain/interfaces/IFunctionLoader'

/**
 * 工作空间状态管理实现
 */
export class WorkspaceStateManager extends StateManagerImpl<WorkspaceState> implements IStateManager<WorkspaceState> {
  constructor(initialState?: Partial<WorkspaceState>) {
    const defaultState: WorkspaceState = {
      currentApp: null,
      currentFunction: null,
      serviceTree: [],
      functionDetails: new Map(), // 🔥 保留字段以兼容接口，但不再使用
      loading: false, // 🔥 默认 loading 为 false
      tabs: [], // 🔥 默认空 Tabs
      activeTabId: null // 🔥 默认无激活 Tab
    }

    super({
      ...defaultState,
      ...initialState
    } as WorkspaceState)

    // 注意：这里不需要设置 watch，因为 state 已经是响应式的
    // 如果需要响应式，可以在使用时通过 computed 或 watch 实现
  }

  /**
   * 获取当前应用
   */
  getCurrentApp(): App | null {
    return this.getState().currentApp
  }

  /**
   * 获取当前函数
   */
  getCurrentFunction(): ServiceTree | null {
    return this.getState().currentFunction
  }

  /**
   * 获取服务树
   */
  getServiceTree(): ServiceTree[] {
    return this.getState().serviceTree
  }

  /**
   * 获取函数详情
   * 🔥 移除缓存后，此方法已废弃，总是返回 null
   * 函数详情应该通过 WorkspaceEvent.functionLoaded 事件获取
   * @deprecated 使用事件监听获取函数详情，而不是从缓存获取
   */
  getFunctionDetail(node: ServiceTree): FunctionDetail | null {
    // 🔥 移除缓存后，不再从缓存获取，总是返回 null
    // 函数详情应该通过 WorkspaceEvent.functionLoaded 事件获取
    return null
  }

  isLoading(): boolean {
    return this.getState().loading
  }
}

