/**
 * WorkspaceApplicationService - 工作空间应用服务
 * 
 * 职责：工作空间业务流程编排
 * - 监听事件，调用 Domain Services
 * - 协调多个 Domain Services 完成业务场景
 * - 不包含业务逻辑，只负责编排
 * 
 * 特点：
 * - 依赖 Domain Services
 * - 通过事件总线监听和触发事件
 * - 不包含业务逻辑，只负责流程编排
 */

import { WorkspaceDomainService } from '../../domain/services/WorkspaceDomainService'
import type { IEventBus } from '../../domain/interfaces/IEventBus'
import { WorkspaceEvent } from '../../domain/interfaces/IEventBus'
import type { App, ServiceTree } from '../../domain/services/WorkspaceDomainService'

/**
 * 工作空间应用服务
 */
export class WorkspaceApplicationService {
  constructor(
    private domainService: WorkspaceDomainService,
    private eventBus: IEventBus
  ) {
    this.setupEventHandlers()
  }

  /**
   * 设置事件处理器
   */
  private setupEventHandlers(): void {
    // 监听节点点击事件
    this.eventBus.on(WorkspaceEvent.nodeClicked, async (payload: { node: ServiceTree }) => {
      await this.handleNodeClick(payload.node)
    })

    // 监听应用切换事件
    this.eventBus.on(WorkspaceEvent.appSwitched, async (payload: { app: App }) => {
      await this.handleAppSwitch(payload.app)
    })
  }

  /**
   * 处理节点点击
   * 如果 Tab 已存在但函数详情未加载，也会加载函数详情（刷新时需要）
   */
  async handleNodeClick(node: ServiceTree): Promise<void> {
    if (node.type === 'function') {
      const tabId = node.full_code_path || String(node.id)
      
      // 使用 Domain Service 的方法检查 Tab 是否存在（遵循依赖倒置原则）
      if (this.domainService.hasTab(tabId)) {
        // Tab 已存在，检查函数详情是否已加载
        const detail = this.domainService.getFunctionDetail(node)
        if (detail) {
          // 函数详情已加载，只激活 Tab
          this.domainService.activateTab(tabId)
        } else {
          // Tab 已存在但函数详情未加载（刷新时的情况），加载函数详情
          const loadedDetail = await this.domainService.loadFunction(node)
          // 加载完成后激活 Tab（确保 currentFunction 和 functionDetails 已更新）
          this.domainService.activateTab(tabId)
        }
      } else {
        // Tab 不存在，加载函数详情并创建新 Tab
        const detail = await this.domainService.loadFunction(node)
        this.domainService.openTab(node, detail)
      }
    } else {
      // 目录节点，只设置当前函数，不加载详情
      this.domainService.setCurrentFunction(node)
    }
  }

  /**
   * 激活标签页（供 Presentation Layer 调用）
   */
  activateTab(tabId: string): void {
    console.log('[WorkspaceApplicationService] activateTab 调用', { tabId })
    this.domainService.activateTab(tabId)
  }

  /**
   * 关闭标签页（供 Presentation Layer 调用）
   */
  closeTab(tabId: string): void {
    this.domainService.closeTab(tabId)
  }

  /**
   * 清空所有标签页（供 Presentation Layer 调用）
   */
  closeAllTabs(): void {
    this.domainService.closeAllTabs()
  }

  /**
   * 处理应用切换
   */
  async handleAppSwitch(app: App): Promise<void> {
    // 🔥 检查当前应用是否已经是目标应用，避免重复切换
    const currentApp = this.domainService.getCurrentApp()
    if (currentApp && currentApp.id === app.id) {
      // 当前应用已经是目标应用，不需要切换
      return
    }
    
    // 切换应用（只更新状态，不触发事件）
    await this.domainService.switchApp(app)
    
    // 加载服务目录树
    await this.domainService.loadServiceTree(app)
  }

  /**
   * 触发节点点击事件（供 Presentation Layer 调用）
   */
  triggerNodeClick(node: ServiceTree): void {
    this.eventBus.emit(WorkspaceEvent.nodeClicked, { node })
  }

  /**
   * 触发应用切换事件（供 Presentation Layer 调用）
   */
  triggerAppSwitch(app: App): void {
    this.eventBus.emit(WorkspaceEvent.appSwitched, { app })
  }
}

