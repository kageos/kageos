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
  loading: boolean // 加载状态
}

// 🔥 空服务树常量：避免每次创建新数组导致引用变化
const EMPTY_SERVICE_TREE: ServiceTree[] = []

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
    // ⭐ 先设置 currentFunction，即使加载失败也能显示权限错误
    const state = this.stateManager.getState()
    this.stateManager.setState({
      ...state,
      currentFunction: node
    })

    // 直接加载函数详情，不使用缓存
    let detail: FunctionDetail
    try {
      // ⭐ 优先使用 full_code_path 加载（新 API 只支持 full-code-path）
      if (node.full_code_path) {
        // ⭐ 从 node.template_type 获取函数类型，传递给 API（后端无需查询数据库即可构造权限点）
        const funcType = node.template_type || 'table'
        detail = await this.functionLoader.loadByPath(node.full_code_path, funcType)
      } else if (node.ref_id && node.ref_id > 0) {
        // ⭐ 如果没有 full_code_path，使用 ref_id 加载（向后兼容，但建议后端总是返回 full_code_path）
        detail = await this.functionLoader.loadById(node.ref_id)
      } else {
        throw new Error('节点没有 full_code_path 和 ref_id，无法加载函数详情')
      }

      // ⭐ 权限信息已从树接口返回，不需要缓存
      // 直接使用 node.permissions（后端返回的最新数据，已包含继承）

      // 触发事件
      this.eventBus.emit(WorkspaceEvent.functionLoaded, { node, detail })

      return detail
    } catch (error: any) {
      // ⭐ 捕获错误（包括 403 权限不足）
      // 即使加载失败，也已经设置了 currentFunction，详情页面可以显示权限错误
      
      // 重新抛出错误，让调用方知道加载失败
      // 但 currentFunction 已经设置，详情页面可以显示权限错误
      throw error
    }
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
      serviceTree: EMPTY_SERVICE_TREE, // 🔥 使用常量空数组，避免引用变化
      loading: true    // 开始加载
    })

    // 不在这里触发 appSwitched 事件，避免循环触发
    // 事件应该在 Application Service 层统一管理
  }

  /**
   * 加载服务目录树（使用已获取的数据，避免重复调用 API）
   */
  async loadServiceTreeWithData(app: App, tree: ServiceTree[], expandedKeys?: number[]): Promise<ServiceTree[]> {
    try {
      console.log('[WorkspaceDomainService] loadServiceTreeWithData 开始，expandedKeys:', expandedKeys)
      const state = this.stateManager.getState()

      // 使用已获取的服务目录树和 expanded_keys

      // 更新状态
      console.log('[WorkspaceDomainService] 调用 setState 更新 serviceTree')
      this.stateManager.setState({
        ...state,
        serviceTree: tree || [],
        loading: false // 🔥 加载完成
      })

      // 触发事件（包含 expandedKeys）
      console.log('[WorkspaceDomainService] 触发 serviceTreeLoaded 事件，expandedKeys:', expandedKeys)
      this.eventBus.emit(WorkspaceEvent.serviceTreeLoaded, { app, tree: tree || [], expandedKeys })

      return tree || []
    } catch (error) {
      console.error('[WorkspaceDomainService] 设置服务目录树失败', error)

      // 更新状态：即使失败也要重置 loading
      const state = this.stateManager.getState()
      this.stateManager.setState({
        ...state,
        serviceTree: EMPTY_SERVICE_TREE, // 🔥 使用常量空数组，避免引用变化
        loading: false // 🔥 加载失败，结束 loading
      })

      // 即使失败也要触发事件，确保 loading 状态能正确更新
      this.eventBus.emit(WorkspaceEvent.serviceTreeLoaded, { app, tree: EMPTY_SERVICE_TREE, expandedKeys: undefined })
      return EMPTY_SERVICE_TREE
    }
  }

  /**
   * 加载服务目录树
   */
  async loadServiceTree(app: App): Promise<ServiceTree[]> {
    if (!this.serviceTreeLoader) {
      return []
    }

    try {
      const state = this.stateManager.getState()
      
      // 从 ServiceTreeLoader 加载服务目录树（现在返回包含 expandedKeys 的结果）
      const result = await this.serviceTreeLoader.load(app)
      const tree = result.tree || []
      const expandedKeys = result.expandedKeys
      const appInfo = result.app

      // 🔥 注意：如果 app.id 是 0（临时值），应用信息的更新由 Application Service 层处理
      // 这里只更新服务树，应用信息的更新在 handleAppSwitch 中处理
      let updatedApp = appInfo || app
      
      // 更新状态
      this.stateManager.setState({
        ...state,
        serviceTree: tree,
        loading: false // 🔥 加载完成
      })

      // 🔥 触发事件，包含 expandedKeys（如果后端返回了）
      console.log('[WorkspaceDomainService] 触发 serviceTreeLoaded 事件，expandedKeys:', expandedKeys)
      this.eventBus.emit(WorkspaceEvent.serviceTreeLoaded, { app: updatedApp, tree, expandedKeys })

      return tree
    } catch (error) {
      console.error('[WorkspaceDomainService] 加载服务目录树失败', error)
      
      // 更新状态：即使失败也要重置 loading
      const state = this.stateManager.getState()
      this.stateManager.setState({
        ...state,
        serviceTree: EMPTY_SERVICE_TREE, // 🔥 使用常量空数组，避免引用变化
        loading: false // 🔥 加载失败，结束 loading
      })
      
      // 即使失败也要触发事件，确保 loading 状态能正确更新
      this.eventBus.emit(WorkspaceEvent.serviceTreeLoaded, { app, tree: EMPTY_SERVICE_TREE, expandedKeys: undefined })
      return EMPTY_SERVICE_TREE
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
    
    // 如果目录相同，但需要设置 currentFunction，仍然需要更新
    if (state.currentDirectory?.id === directory?.id) {
      // 如果需要将目录设置为当前函数，且当前函数不是这个目录，则更新
      if (setAsCurrentFunction && state.currentFunction?.id !== directory?.id) {
        this.stateManager.setState({
          ...state,
          currentFunction: directory
        })
      }
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
