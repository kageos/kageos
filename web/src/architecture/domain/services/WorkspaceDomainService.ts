/**
 * WorkspaceDomainService - 工作空间领域服务
 * 
 * 职责：工作空间相关的业务逻辑
 * - 加载函数详情
 * - 切换应用
 * - 加载服务目录树
 * - 管理当前函数和应用状态
 * 
 * 特点：
 * - 依赖接口，不依赖具体实现
 * - 通过事件总线通信
 * - 通过状态管理器管理状态
 */

import type { IFunctionLoader, FunctionDetail } from '../interfaces/IFunctionLoader'
import type { IStateManager } from '../interfaces/IStateManager'
import type { IEventBus } from '../interfaces/IEventBus'
import { WorkspaceEvent } from '../interfaces/IEventBus'

/**
 * 应用类型（从 types 导入）
 */
import type { App, ServiceTree } from '@/types'

// 重新导出，方便使用
export type { App, ServiceTree }

/**
 * 服务目录树加载器接口
 */
export interface IServiceTreeLoader {
  load(app: App): Promise<ServiceTree[]>
}

/**
 * 工作空间状态
 */
export interface WorkspaceState {
  currentApp: App | null
  currentFunction: ServiceTree | null
  serviceTree: ServiceTree[]
  functionDetails: Map<string, FunctionDetail>
}

/**
 * 工作空间领域服务
 */
export class WorkspaceDomainService {
  constructor(
    private functionLoader: IFunctionLoader,
    private stateManager: IStateManager<WorkspaceState>,
    private eventBus: IEventBus,
    private serviceTreeLoader?: IServiceTreeLoader
  ) {}

  /**
   * 加载函数详情
   */
  async loadFunction(node: ServiceTree): Promise<FunctionDetail> {
    const state = this.stateManager.getState()
    
    // 生成缓存键
    const key = node.ref_id ? `id:${node.ref_id}` : `path:${node.full_code_path}`
    
    // 先检查缓存
    const cached = state.functionDetails.get(key)
    if (cached) {
      // 触发事件（使用缓存）
      this.eventBus.emit(WorkspaceEvent.functionLoaded, { node, detail: cached })
      return cached
    }

    // 加载函数详情
    let detail: FunctionDetail
    if (node.ref_id && node.ref_id > 0) {
      detail = await this.functionLoader.loadById(node.ref_id)
    } else if (node.full_code_path) {
      detail = await this.functionLoader.loadByPath(node.full_code_path)
    } else {
      throw new Error('节点没有 ref_id 和 full_code_path，无法加载函数详情')
    }

    // 更新状态
    const newFunctionDetails = new Map(state.functionDetails)
    newFunctionDetails.set(key, detail)
    
    this.stateManager.setState({
      ...state,
      currentFunction: node,
      functionDetails: newFunctionDetails
    })

    // 触发事件
    this.eventBus.emit(WorkspaceEvent.functionLoaded, { node, detail })

    return detail
  }

  /**
   * 切换应用
   */
  async switchApp(app: App): Promise<void> {
    const state = this.stateManager.getState()
    
    // 更新状态
    this.stateManager.setState({
      ...state,
      currentApp: app,
      currentFunction: null,
      serviceTree: [] // 清空服务树，等待重新加载
    })

    // 触发事件
    this.eventBus.emit(WorkspaceEvent.appSwitched, { app })
  }

  /**
   * 加载服务目录树
   */
  async loadServiceTree(app: App): Promise<ServiceTree[]> {
    if (!this.serviceTreeLoader) {
      console.warn('[WorkspaceDomainService] ServiceTreeLoader 未注入，无法加载服务目录树')
      return []
    }

    const state = this.stateManager.getState()
    
    // 从 ServiceTreeLoader 加载服务目录树
    const tree = await this.serviceTreeLoader.load(app)
    
    // 更新状态
    this.stateManager.setState({
      ...state,
      serviceTree: tree
    })

    // 触发事件
    this.eventBus.emit(WorkspaceEvent.serviceTreeLoaded, { app, tree })

    return tree
  }

  /**
   * 设置当前函数（不加载详情）
   */
  setCurrentFunction(node: ServiceTree | null): void {
    const state = this.stateManager.getState()
    
    this.stateManager.setState({
      ...state,
      currentFunction: node
    })
  }

  /**
   * 获取当前应用
   */
  getCurrentApp(): App | null {
    return this.stateManager.getState().currentApp
  }

  /**
   * 获取当前函数
   */
  getCurrentFunction(): ServiceTree | null {
    return this.stateManager.getState().currentFunction
  }

  /**
   * 获取服务树
   */
  getServiceTree(): ServiceTree[] {
    return this.stateManager.getState().serviceTree
  }

  /**
   * 获取函数详情（从缓存）
   */
  getFunctionDetail(node: ServiceTree): FunctionDetail | null {
    const state = this.stateManager.getState()
    const key = node.ref_id ? `id:${node.ref_id}` : `path:${node.full_code_path}`
    return state.functionDetails.get(key) || null
  }
}

