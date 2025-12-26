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
   * - 点击目录节点：切换到该目录，并获取目录权限
   * - 点击函数节点：直接加载函数详情，不先切换目录（避免闪烁），并获取函数权限
   */
  async handleNodeClick(node: ServiceTree): Promise<void> {
    // ⭐ 先获取节点权限信息（目录和函数都需要）
    await this.loadNodePermissions(node)

    if (node.type === 'function') {
      // 🔥 优化：直接加载函数详情，不先切换目录
      // 这样可以避免先显示目录详情再切换到函数详情的闪烁问题
      try {
        const detail = await this.domainService.loadFunction(node)
        
        // 加载完成后，一次性设置目录和函数，避免中间状态
        const functionDirectory = this.getFunctionDirectory(node)
        if (functionDirectory) {
          // 设置目录，但不将目录设置为当前函数（避免显示目录详情）
          this.domainService.setCurrentDirectory(functionDirectory, false)
        }
        
        // 然后设置函数（这会触发函数详情显示）
        this.domainService.setCurrentFunction(node)
      } catch (error: any) {
        // ⭐ 捕获错误（包括 403 权限不足）
        // currentFunction 已经在 loadFunction 中设置了
        // 权限错误信息已经通过 request.ts 拦截器存储到 permissionErrorStore 中
        // 这里只需要设置函数，让详情页面显示权限错误组件
        const functionDirectory = this.getFunctionDirectory(node)
        if (functionDirectory) {
          this.domainService.setCurrentDirectory(functionDirectory, false)
        }
        this.domainService.setCurrentFunction(node)
        // 不重新抛出错误，让 UI 显示权限错误组件
      }
    } else {
      // 目录节点：切换到该目录
      this.domainService.setCurrentDirectory(node)
    }
  }

  /**
   * 加载节点权限信息（从详情接口获取）
   * ⭐ 优化：添加请求去重，避免重复调用
   */
  private loadingPermissions = new Set<string>() // 正在加载的权限请求（用于去重）

  private async loadNodePermissions(node: ServiceTree): Promise<void> {
    // 检查是否有 id 或 full_code_path
    if (!node.id && !node.full_code_path) {
      return
    }

    // 生成缓存键（用于去重）
    const cacheKey = node.full_code_path || `node:${node.id}`
    
    // 检查是否正在加载（去重）
    if (this.loadingPermissions.has(cacheKey)) {
      return
    }

    try {
      // 动态导入，避免循环依赖
      const { getPackageInfo } = await import('@/api/service-tree')
      const { useNodePermissionsStore } = await import('@/stores/nodePermissions')
      
      const permissionStore = useNodePermissionsStore()
      
      // 检查缓存
      const cached = permissionStore.getPermissions(node)
      if (cached) {
        // 使用缓存的权限信息
        return
      }

      // ⭐ 函数节点的权限从函数详情接口获取，不需要单独调用
      if (node.type === 'function') {
        // 函数权限会在 loadFunction 时从函数详情接口获取并缓存
        return
      }

      // 标记为正在加载
      this.loadingPermissions.add(cacheKey)

      // ⭐ 只对目录节点调用 package_info 接口获取权限
      const params: { id?: number; full_code_path?: string } = {}
      if (node.id) {
        params.id = node.id
      }
      if (node.full_code_path) {
        params.full_code_path = node.full_code_path
      }

      const packageInfo = await getPackageInfo(params)
      if (packageInfo.permissions) {
        // 缓存权限信息
        permissionStore.setPermissions(node, packageInfo.permissions)
      }
    } catch (error) {
      // 权限获取失败不影响主流程，只是权限控制可能不准确
      console.warn('[WorkspaceApplicationService] 获取节点权限失败:', error)
    } finally {
      // 移除加载标记
      this.loadingPermissions.delete(cacheKey)
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
    if (currentApp && currentApp.id === app.id && app.id !== 0) {
      // 当前应用已经是目标应用，不需要切换
      return
    }
    
    // 🔥 修复：如果 app.id 是 0（临时值），通过合并接口获取完整的应用信息和服务目录树
    let appToSwitch = app
    let preloadedServiceTree: ServiceTree[] | null = null
    
    if (app.id === 0) {
      try {
        // 动态导入 getAppWithServiceTree，避免循环依赖
        const { getAppWithServiceTree } = await import('@/api/app')
        // ⭐ 传递 user 和 app，而不是只传 code
        const workspaceData = await getAppWithServiceTree(app.user, app.code)
        if (workspaceData && workspaceData.app) {
          appToSwitch = {
            id: workspaceData.app.id,
            user: workspaceData.app.user,
            code: workspaceData.app.code,
            name: workspaceData.app.name
          }
          console.log('[WorkspaceApplicationService] 从合并接口获取到应用信息', appToSwitch)
          
          // 🔥 修复：如果已经获取了服务目录树，直接使用，避免重复调用
          if (workspaceData.service_tree && Array.isArray(workspaceData.service_tree)) {
            preloadedServiceTree = workspaceData.service_tree
            console.log('[WorkspaceApplicationService] 从合并接口获取到服务目录树，节点数:', preloadedServiceTree.length)
          }
          
          // 🔥 修复：发出应用信息更新事件，让 Presentation Layer 更新 appList
          // 这样 currentApp 的 computed 就能找到对应的应用了
          this.eventBus.emit('workspace:app-info-updated', { app: appToSwitch })
        }
      } catch (error) {
        console.error('[WorkspaceApplicationService] 获取应用信息失败', error)
        // 如果获取失败，继续使用原始的 app 对象
      }
    }
    
    // 切换应用（只更新状态，不触发事件）
    await this.domainService.switchApp(appToSwitch)
    
    // 🔥 优化：如果已经获取了服务目录树，直接使用，避免重复调用
    if (preloadedServiceTree) {
      await this.domainService.loadServiceTreeWithData(appToSwitch, preloadedServiceTree)
    } else {
    // 加载服务目录树
      await this.domainService.loadServiceTree(appToSwitch)
    }
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

