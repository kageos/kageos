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
import type { IServiceTreeLoader } from '../interfaces/IServiceTreeLoader'
import { WorkspaceEvent } from '../interfaces/IEventBus'

/**
 * 应用类型（从 types 导入）
 */
import type { App, ServiceTree } from '@/types'

// 重新导出，方便使用
export type { App, ServiceTree }

/**
 * 工作空间状态
 */
export interface WorkspaceState {
  currentApp: App | null
  currentFunction: ServiceTree | null
  currentDirectory: ServiceTree | null // 当前目录
  serviceTree: ServiceTree[]
  functionDetails: Map<string, FunctionDetail> // 🔥 保留字段以兼容接口，但不再使用（移除缓存机制）
  loading: boolean // 加载状态
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
   * 🔥 移除缓存机制，每次切换函数时都重新加载，确保数据一致性
   * @param node 函数节点
   */
  async loadFunction(node: ServiceTree): Promise<FunctionDetail> {
    // 直接加载函数详情，不使用缓存
    let detail: FunctionDetail
    if (node.ref_id && node.ref_id > 0) {
      detail = await this.functionLoader.loadById(node.ref_id)
    } else if (node.full_code_path) {
      detail = await this.functionLoader.loadByPath(node.full_code_path)
    } else {
      throw new Error('节点没有 ref_id 和 full_code_path，无法加载函数详情')
    }

    // 更新状态（不缓存函数详情）
    const state = this.stateManager.getState()
    this.stateManager.setState({
      ...state,
      currentFunction: node
    })

    // 触发事件
    this.eventBus.emit(WorkspaceEvent.functionLoaded, { node, detail })

    return detail
  }

  /**
   * 设置当前函数（加载函数详情并更新状态）
   * 🔥 简化：不再使用 Tab，直接设置当前函数
   */
  setCurrentFunctionWithDetail(node: ServiceTree, detail?: FunctionDetail): void {
    const state = this.stateManager.getState()
    
    // 更新函数详情缓存
    const key = node.ref_id ? `id:${node.ref_id}` : `path:${node.full_code_path}`
    const newFunctionDetails = new Map(state.functionDetails)
    if (detail) {
      newFunctionDetails.set(key, detail)
    }

    this.stateManager.setState({
      ...state,
      currentFunction: node,
      functionDetails: newFunctionDetails
    })
  }

  /**
   * 切换应用
   * 注意：这个方法不应该触发 appSwitched 事件，因为事件应该在 Application Service 层触发
   * 这个方法只负责更新状态
   */
  async switchApp(app: App): Promise<void> {
    const state = this.stateManager.getState()
    
    // 更新状态：设置当前应用，清空服务树和当前目录，设置 loading 为 true
    this.stateManager.setState({
      ...state,
      currentApp: app,
      currentFunction: null,
      currentDirectory: null,
      serviceTree: [], // 清空服务树，等待重新加载
      loading: true    // 开始加载
    })

    // 不在这里触发 appSwitched 事件，避免循环触发
    // 事件应该在 Application Service 层统一管理
  }

  /**
   * 加载服务目录树（使用已获取的数据，避免重复调用 API）
   */
  async loadServiceTreeWithData(app: App, tree: ServiceTree[]): Promise<ServiceTree[]> {
    try {
      const state = this.stateManager.getState()

      console.log('[WorkspaceDomainService] 使用已获取的服务目录树，节点数:', tree?.length || 0)

      // 更新状态
      this.stateManager.setState({
        ...state,
        serviceTree: tree || [],
        loading: false // 🔥 加载完成
      })

      // 触发事件
      this.eventBus.emit(WorkspaceEvent.serviceTreeLoaded, { app, tree: tree || [] })

      console.log('[WorkspaceDomainService] 已触发 serviceTreeLoaded 事件')

      return tree || []
    } catch (error) {
      console.error('[WorkspaceDomainService] 设置服务目录树失败', error)

      // 更新状态：即使失败也要重置 loading
      const state = this.stateManager.getState()
      this.stateManager.setState({
        ...state,
        serviceTree: [],
        loading: false // 🔥 加载失败，结束 loading
      })

      // 即使失败也要触发事件，确保 loading 状态能正确更新
      this.eventBus.emit(WorkspaceEvent.serviceTreeLoaded, { app, tree: [] })
      return []
    }
  }

  /**
   * 加载服务目录树
   */
  async loadServiceTree(app: App): Promise<ServiceTree[]> {
    if (!this.serviceTreeLoader) {
      console.warn('[WorkspaceDomainService] ServiceTreeLoader 未注入，无法加载服务目录树')
      return []
    }

    try {
      const state = this.stateManager.getState()
      
      console.log('[WorkspaceDomainService] 开始加载服务目录树:', app.user, app.code, 'app.id:', app.id)
      
      // 从 ServiceTreeLoader 加载服务目录树
      const tree = await this.serviceTreeLoader.load(app)
      
      console.log('[WorkspaceDomainService] 服务目录树加载完成，节点数:', tree?.length || 0)

      // 🔥 注意：如果 app.id 是 0（临时值），应用信息的更新由 Application Service 层处理
      // 这里只更新服务树，应用信息的更新在 handleAppSwitch 中处理
      let updatedApp = app
      
      // 更新状态
      this.stateManager.setState({
        ...state,
        serviceTree: tree || [],
        loading: false // 🔥 加载完成
      })

      // 触发事件
      this.eventBus.emit(WorkspaceEvent.serviceTreeLoaded, { app: updatedApp, tree: tree || [] })
      
      console.log('[WorkspaceDomainService] 已触发 serviceTreeLoaded 事件')

      return tree || []
    } catch (error) {
      console.error('[WorkspaceDomainService] 加载服务目录树失败', error)
      
      // 更新状态：即使失败也要重置 loading
      const state = this.stateManager.getState()
      this.stateManager.setState({
        ...state,
        serviceTree: [],
        loading: false // 🔥 加载失败，结束 loading
      })
      
      // 即使失败也要触发事件，确保 loading 状态能正确更新
      this.eventBus.emit(WorkspaceEvent.serviceTreeLoaded, { app, tree: [] })
      return []
    }
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
   * 设置当前目录（切换目录时调用）
   * @param directory 目录节点
   * @param setAsCurrentFunction 是否同时将目录设置为当前函数（默认 true，用于点击目录节点时）
   */
  setCurrentDirectory(directory: ServiceTree | null, setAsCurrentFunction: boolean = true): void {
    const state = this.stateManager.getState()
    
    // 如果目录相同，不执行任何操作
    if (state.currentDirectory?.id === directory?.id) {
      return
    }
    
    this.stateManager.setState({
      ...state,
      currentDirectory: directory,
      // 🔥 优化：只有在明确要求时才将目录设置为当前函数
      // 这样可以避免在加载函数详情时先显示目录详情
      currentFunction: setAsCurrentFunction ? directory : state.currentFunction
    })
  }

  /**
   * 获取当前目录
   */
  getCurrentDirectory(): ServiceTree | null {
    return this.stateManager.getState().currentDirectory
  }

  /**
   * 获取指定目录下的所有函数节点（只获取直接子函数，不包括子目录下的函数）
   */
  getFunctionsInDirectory(directory: ServiceTree): ServiceTree[] {
    const state = this.stateManager.getState()
    const functions: ServiceTree[] = []
    
    // 递归查找目录节点
    const findDirectoryNode = (nodes: ServiceTree[], targetId: number): ServiceTree | null => {
      for (const node of nodes) {
        if (node.id === targetId) {
          return node
        }
        if (node.children && node.children.length > 0) {
          const found = findDirectoryNode(node.children, targetId)
          if (found) return found
        }
      }
      return null
    }
    
    // 找到目录节点
    const dirNode = findDirectoryNode(state.serviceTree, directory.id)
    if (!dirNode || !dirNode.children) {
      return []
    }
    
    // 只获取直接子函数（不包括子目录）
    for (const child of dirNode.children) {
      if (child.type === 'function') {
        functions.push(child)
      }
    }
    
    return functions
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

  /**
   * 检查是否正在加载
   */
  isLoading(): boolean {
    return this.stateManager.getState().loading
  }

  /**
   * 获取状态管理器（供 Application Layer 使用，遵循依赖倒置原则）
   */
  getStateManager(): IStateManager<WorkspaceState> {
    return this.stateManager
  }

}

