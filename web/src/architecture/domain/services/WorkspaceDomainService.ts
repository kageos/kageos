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
 * 工作空间 Tab
 */
export interface WorkspaceTab {
  id: string
  title: string
  type: 'function' | 'welcome' | 'settings'
  path: string
  icon?: string
  closable: boolean
  node?: ServiceTree // 关联的服务树节点
  data?: any // 保存的状态
}

/**
 * 工作空间状态
 */
export interface WorkspaceState {
  currentApp: App | null
  currentFunction: ServiceTree | null
  serviceTree: ServiceTree[]
  functionDetails: Map<string, FunctionDetail>
  loading: boolean // 🔥 添加 loading 状态，统一管理加载状态
  tabs: WorkspaceTab[] // 🔥 多标签页支持
  activeTabId: string | null // 🔥 当前激活的 Tab ID
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
   * 打开 Tab（如果已存在则激活，否则创建）
   * 🔥 修复：如果 Tab 已存在，只激活，不重新加载函数详情
   */
  openTab(node: ServiceTree, detail?: FunctionDetail): void {
    const state = this.stateManager.getState()
    const tabId = node.full_code_path || String(node.id)
    
    // 检查是否已存在
    const existingTab = state.tabs.find(t => t.id === tabId)
    if (existingTab) {
      // 🔥 Tab 已存在，只激活，不重新加载函数详情（避免重复加载）
      this.activateTab(tabId)
      return
    }

    // 🔥 创建新 Tab（使用传入的 detail 或从缓存中获取）
    const functionDetail = detail || this.getFunctionDetail(node)
    
    const newTab: WorkspaceTab = {
      id: tabId,
      title: node.name || node.code,
      path: node.full_code_path || String(node.id),
      node: node
    }
    
    const newFunctionDetails = new Map(state.functionDetails)
    if (functionDetail) {
      newFunctionDetails.set(tabId, functionDetail)
    }

    this.stateManager.setState({
      ...state,
      tabs: [...state.tabs, newTab],
      activeTabId: tabId,
      currentFunction: node,
      functionDetails: newFunctionDetails
    })

    // 🔥 触发路由更新事件（让 Presentation Layer 更新路由）
    this.eventBus.emit(WorkspaceEvent.tabOpened, { tab: newTab, shouldUpdateRoute: true })
  }

  /**
   * 激活 Tab
   */
  activateTab(tabId: string): void {
    const state = this.stateManager.getState()
    const tab = state.tabs.find(t => t.id === tabId)
    if (tab) {
      this.stateManager.setState({
        ...state,
        activeTabId: tabId,
        currentFunction: tab.node || null
      })

      // 🔥 触发路由更新事件（让 Presentation Layer 更新路由）
      this.eventBus.emit(WorkspaceEvent.tabActivated, { tab, shouldUpdateRoute: true })
    }
  }

  /**
   * 关闭 Tab
   */
  closeTab(tabId: string): void {
    const state = this.stateManager.getState()
    const tabIndex = state.tabs.findIndex(t => t.id === tabId)
    if (tabIndex === -1) return

    const newTabs = state.tabs.filter(t => t.id !== tabId)
    let newActiveId = state.activeTabId
    let newCurrentFunction = state.currentFunction

    // 如果关闭的是当前激活的 Tab
    if (tabId === state.activeTabId) {
      if (newTabs.length > 0) {
        // 激活相邻的 Tab（优先右侧，如果没有则左侧）
        // 注意：filter 后的索引可能发生变化
        const nextTab = newTabs[Math.min(tabIndex, newTabs.length - 1)]
        newActiveId = nextTab.id
        newCurrentFunction = nextTab.node || null
      } else {
        newActiveId = null
        newCurrentFunction = null
      }
    }

    this.stateManager.setState({
      ...state,
      tabs: newTabs,
      activeTabId: newActiveId,
      currentFunction: newCurrentFunction
    })
  }

  /**
   * 切换应用
   * 注意：这个方法不应该触发 appSwitched 事件，因为事件应该在 Application Service 层触发
   * 这个方法只负责更新状态
   */
  async switchApp(app: App): Promise<void> {
    const state = this.stateManager.getState()
    
    // 更新状态：设置当前应用，清空服务树，设置 loading 为 true
    this.stateManager.setState({
      ...state,
      currentApp: app,
      currentFunction: null,
      serviceTree: [], // 清空服务树，等待重新加载
      loading: true    // 🔥 开始加载
    })

    // 🔥 不在这里触发 appSwitched 事件，避免循环触发
    // 事件应该在 Application Service 层统一管理
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
      
      console.log('[WorkspaceDomainService] 开始加载服务目录树:', app.user, app.code)
      
      // 从 ServiceTreeLoader 加载服务目录树
      const tree = await this.serviceTreeLoader.load(app)
      
      console.log('[WorkspaceDomainService] 服务目录树加载完成，节点数:', tree?.length || 0)
      
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
  isLoading(): boolean {
    return this.stateManager.getState().loading
  }
}

