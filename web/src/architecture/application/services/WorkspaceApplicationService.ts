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
   */
  async handleNodeClick(node: ServiceTree): Promise<void> {
    if (node.type === 'function') {
      // 加载函数详情
      await this.domainService.loadFunction(node)
    } else {
      // 目录节点，只设置当前函数，不加载详情
      this.domainService.setCurrentFunction(node)
    }
  }

  /**
   * 处理应用切换
   */
  async handleAppSwitch(app: App): Promise<void> {
    // 切换应用
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

