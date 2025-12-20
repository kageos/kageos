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
   * 🔥 简化：不再使用 Tab，直接加载函数详情
   * - 点击目录节点：切换到该目录
   * - 点击函数节点：加载函数详情并设置当前函数
   */
  async handleNodeClick(node: ServiceTree): Promise<void> {
    if (node.type === 'function') {
      // 检查函数是否在当前目录下
      const currentDirectory = this.domainService.getCurrentDirectory()
      const functionDirectory = this.getFunctionDirectory(node)
      
      // 如果函数不在当前目录，先切换到函数所在目录
      if (!currentDirectory || currentDirectory.id !== functionDirectory?.id) {
        if (functionDirectory) {
          this.domainService.setCurrentDirectory(functionDirectory)
        }
      }
      
      // 加载函数详情并设置当前函数
      const detail = await this.domainService.loadFunction(node)
      this.domainService.setCurrentFunctionWithDetail(node, detail)
    } else {
      // 目录节点：切换到该目录
      this.domainService.setCurrentDirectory(node)
    }
  }

  /**
   * 获取函数所在的目录节点
   */
  private getFunctionDirectory(functionNode: ServiceTree): ServiceTree | null {
    const serviceTree = this.domainService.getServiceTree()
    
    // 方法1：通过 parent_id 查找（如果函数节点有 parent_id）
    if (functionNode.parent_id && functionNode.parent_id > 0) {
      const findNodeById = (nodes: ServiceTree[], targetId: number): ServiceTree | null => {
        for (const node of nodes) {
          if (node.id === targetId && node.type === 'package') {
            return node
          }
          if (node.children && node.children.length > 0) {
            const found = findNodeById(node.children, targetId)
            if (found) return found
          }
        }
        return null
      }
      
      const directory = findNodeById(serviceTree, functionNode.parent_id)
      if (directory) {
        return directory
      }
    }
    
    // 方法2：从 full_code_path 提取目录路径（回退方案）
    if (!functionNode.full_code_path) {
      return null
    }
    
    const pathParts = functionNode.full_code_path.split('/').filter(Boolean)
    if (pathParts.length < 3) {
      // 路径格式：/user/app/...，至少需要 3 段
      return null
    }
    
    // 移除最后一段（函数名），得到目录路径
    const directoryPath = '/' + pathParts.slice(0, -1).join('/')
    
    // 在服务树中查找目录节点
    const findNodeByPath = (nodes: ServiceTree[], targetPath: string): ServiceTree | null => {
      for (const node of nodes) {
        if (node.full_code_path === targetPath && node.type === 'package') {
          return node
        }
        if (node.children && node.children.length > 0) {
          const found = findNodeByPath(node.children, targetPath)
          if (found) return found
        }
      }
      return null
    }
    
    return findNodeByPath(serviceTree, directoryPath)
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

