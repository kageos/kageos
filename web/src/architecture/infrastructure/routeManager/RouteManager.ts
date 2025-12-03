/**
 * RouteManager - 路由管理器
 * 
 * 职责：
 * - 唯一的路由更新入口
 * - 统一处理参数保留逻辑
 * - 监听 Vue Router 变化，发出事件
 * - 管理 Tab 路由状态
 * - 防止路由更新循环
 */

import { watch, nextTick } from 'vue'
import type { WatchSource } from 'vue'
import type { Router, RouteLocationNormalized } from 'vue-router'
import type { IEventBus } from '../../domain/interfaces/IEventBus'
import { RouteEvent, WorkspaceEvent } from '../../domain/interfaces/IEventBus'
import { TabStateManager, type TabRouteState } from './TabStateManager'
import { TABLE_PARAM_KEYS, SEARCH_PARAM_KEYS } from '@/utils/urlParams'
import { Logger } from '@/core/utils/logger'

export interface RouteUpdateRequest {
  path?: string                    // 路径（可选，不提供则使用当前路径）
  query?: Record<string, any>      // 查询参数（可选）
  replace?: boolean                // 是否使用 replace（默认 true）
  preserveParams?: {               // 参数保留策略
    table?: boolean                 // 保留 table 参数（page, page_size, sorts）
    search?: boolean                // 保留搜索参数（eq, like, in 等）
    state?: boolean                 // 保留状态参数（_ 开头）
    custom?: string[]               // 自定义要保留的参数
    linkNavigation?: boolean        // 是否是 link 跳转（保留所有参数）
  }
  source?: string                   // 更新来源（用于调试）
}

export class RouteManager {
  private router: Router
  private route: RouteLocationNormalized
  private eventBus: IEventBus
  private tabStateManager: TabStateManager
  private getCurrentTabId: () => string | null
  private isUpdating = false  // 防止循环更新
  private enableDebugLog = false  // 调试日志开关
  
  constructor(
    router: Router, 
    route: RouteLocationNormalized, 
    eventBus: IEventBus,
    getCurrentTabId: () => string | null
  ) {
    this.router = router
    this.route = route
    this.eventBus = eventBus
    this.getCurrentTabId = getCurrentTabId
    this.tabStateManager = new TabStateManager()
    
    // 监听路由变化，发出事件
    this.setupRouteWatch()
    
    // 监听路由更新请求事件
    this.setupUpdateListener()
    
    // 监听 Tab 切换事件
    this.setupTabSwitchListener()
    
    this.log('RouteManager 初始化完成')
  }
  
  /**
   * 设置调试日志
   */
  setDebugLog(enabled: boolean): void {
    this.enableDebugLog = enabled
  }
  
  /**
   * 调试日志
   */
  private log(message: string, data?: any): void {
    if (this.enableDebugLog) {
      Logger.debug('RouteManager', message, data)
    }
  }
  
  /**
   * 监听 Vue Router 变化
   */
  private setupRouteWatch(): void {
    watch(() => [this.route.path, this.route.query] as [string, Record<string, any>], ([newPath, newQuery], [oldPath, oldQuery]) => {
      if (this.isUpdating) {
        // 如果是程序触发的更新，不发出事件（避免循环）
        this.log('路由更新（程序触发），跳过事件', { path: newPath })
        return
      }
      
      // 用户操作（浏览器前进/后退）或外部变化，发出事件
      this.log('路由变化（用户操作）', { 
        oldPath, 
        newPath, 
        oldQuery, 
        newQuery 
      })
      
      this.eventBus.emit(RouteEvent.routeChanged, {
        path: newPath,
        query: newQuery,
        oldPath,
        oldQuery,
        source: 'router-change'
      })
      
      // 同时发出 pathChanged 和 queryChanged 事件
      if (oldPath !== newPath) {
        this.eventBus.emit(RouteEvent.pathChanged, {
          path: newPath,
          oldPath,
          source: 'router-change'
        })
      }
      
      if (JSON.stringify(oldQuery) !== JSON.stringify(newQuery)) {
        this.eventBus.emit(RouteEvent.queryChanged, {
          query: newQuery,
          oldQuery,
          source: 'router-change'
        })
      }
    })
  }
  
  /**
   * 监听路由更新请求
   */
  private setupUpdateListener(): void {
    this.eventBus.on(RouteEvent.updateRequested, async (request: RouteUpdateRequest) => {
      await this.handleUpdateRequest(request)
    })
  }
  
  /**
   * 监听 Tab 切换事件
   */
  private setupTabSwitchListener(): void {
    this.eventBus.on(WorkspaceEvent.tabSwitching, (payload: { oldTabId: string, newTabId: string }) => {
      this.handleTabSwitch(payload.oldTabId, payload.newTabId)
    })
  }
  
  /**
   * 处理 Tab 切换
   */
  private handleTabSwitch(oldTabId: string, newTabId: string): void {
    this.log('Tab 切换', { oldTabId, newTabId })
    
    // 1. 🔥 保存旧 Tab 的路由状态
    // 注意：此时 getCurrentTabId() 可能已经返回 newTabId（因为 activateTab 已经更新了状态）
    // 所以，我们需要先获取旧 Tab 的路由状态（如果已保存），或者使用当前路由
    // 但是，如果当前路由对应的 Tab 不是 oldTabId，说明路由已经更新了，我们需要使用当前路由
    const currentRoute = this.getCurrentRoute()
    const currentTabId = this.getCurrentTabId()
    
    this.log('保存旧 Tab 路由状态 - 当前状态', { 
      oldTabId, 
      newTabId, 
      currentTabId, 
      currentPath: currentRoute.path,
      currentQuery: currentRoute.query
    })
    
    // 🔥 如果当前 Tab ID 已经是 newTabId，说明 activateTab 已经更新了状态
    // 此时，我们需要使用当前路由作为 oldTabId 的状态（因为路由还没有更新）
    // 但是，如果路由已经更新了，我们需要使用当前路由
    if (currentTabId === newTabId) {
      // 当前 Tab ID 已经是 newTabId，说明 activateTab 已经更新了状态
      // 此时，当前路由应该还是旧 Tab 的路由（因为路由更新是异步的）
      // 所以，我们可以使用当前路由作为 oldTabId 的状态
      this.tabStateManager.saveTabRouteState(oldTabId, currentRoute)
      this.log('保存 Tab 路由状态（activateTab 已更新状态）', { tabId: oldTabId, route: currentRoute })
    } else if (currentTabId === oldTabId) {
      // 当前 Tab ID 还是 oldTabId，说明状态还没有更新
      // 直接使用当前路由作为 oldTabId 的状态
      this.tabStateManager.saveTabRouteState(oldTabId, currentRoute)
      this.log('保存 Tab 路由状态', { tabId: oldTabId, route: currentRoute })
    } else {
      // 当前 Tab ID 既不是 oldTabId 也不是 newTabId，说明状态已经更新到其他 Tab
      // 这种情况下，我们无法确定 oldTabId 的路由状态，只能使用当前路由
      this.tabStateManager.saveTabRouteState(oldTabId, currentRoute)
      this.log('保存 Tab 路由状态（状态已更新到其他 Tab）', { tabId: oldTabId, route: currentRoute, currentTabId })
    }
    
    // 2. 恢复目标 Tab 的路由状态
    const targetRouteState = this.tabStateManager.getTabRouteState(newTabId)
    if (targetRouteState) {
      // 🔥 验证：确保恢复的路由状态路径与 newTabId 对应的 Tab 路径匹配
      // 如果路径不匹配，说明保存的状态是错误的，应该使用 Tab 的默认路径
      const expectedPath = `/workspace${newTabId}`
      const isPathValid = targetRouteState.path === expectedPath || targetRouteState.path.startsWith(expectedPath + '?')
      
      this.log('检查恢复的 Tab 路由状态', { 
        tabId: newTabId, 
        savedPath: targetRouteState.path, 
        expectedPath,
        isPathValid,
        savedQuery: targetRouteState.query
      })
      
      if (isPathValid) {
        this.log('恢复 Tab 路由状态', { tabId: newTabId, route: targetRouteState })
        
        // 发出路由更新请求，恢复目标 Tab 的路由状态
        // 🔥 传递 newTabId 作为元数据，用于在路由更新完成后保存新 Tab 的路由状态
        this.requestUpdate({
          path: targetRouteState.path,
          query: targetRouteState.query,
          source: 'tab-switch',
          preserveParams: {
            linkNavigation: false  // Tab 切换不是 link 跳转，使用目标 Tab 保存的状态
          },
          // 🔥 传递 newTabId，用于在路由更新完成后保存新 Tab 的路由状态
          meta: { newTabId }
        } as RouteUpdateRequest & { meta?: { newTabId: string } })
      } else {
        // 路径不匹配，说明保存的状态是错误的，使用 Tab 的默认路径
        this.log('恢复的 Tab 路由状态路径不匹配，删除错误状态并使用默认路径', { 
          tabId: newTabId, 
          savedPath: targetRouteState.path, 
          expectedPath 
        })
        // 删除错误的状态
        this.tabStateManager.deleteTabRouteState(newTabId)
        // 使用默认路径（由 useWorkspaceTabs 处理）
      }
    } else {
      this.log('Tab 没有保存的路由状态，使用默认路由', { tabId: newTabId })
      // 🔥 即使没有保存的状态，也需要发出路由更新请求（使用默认路径）
      // 这样，useWorkspaceTabs 就不需要再发出 tab-click 请求了
      const defaultPath = `/workspace${newTabId}`
      this.requestUpdate({
        path: defaultPath,
        query: {},
        source: 'tab-switch',
        preserveParams: {
          linkNavigation: false
        },
        meta: { newTabId }
      } as RouteUpdateRequest & { meta?: { newTabId: string } })
    }
  }
  
  /**
   * 处理路由更新请求
   */
  private async handleUpdateRequest(request: RouteUpdateRequest): Promise<void> {
    // 🔥 sync-route-to-tab-save-state 是特殊请求，只用于保存 Tab 路由状态，不实际更新路由
    if ((request as any).source === 'sync-route-to-tab-save-state') {
      const tabId = (request as any).meta?.tabId
      const savedPath = (request as any).meta?.path
      const savedQuery = (request as any).meta?.query
      if (tabId) {
        // 🔥 使用传递过来的 path 和 query，而不是当前路由的 path 和 query
        // 因为当前路由可能已经更新了（如果用户快速切换）
        const routeState = {
          path: savedPath || this.route.path,
          query: savedQuery || { ...this.route.query }
        }
        this.tabStateManager.saveTabRouteState(tabId, routeState)
        this.log('保存 Tab 路由状态（sync-route-to-tab）', { 
          tabId, 
          route: routeState,
          savedPath,
          savedQuery,
          currentPath: this.route.path,
          currentQuery: this.route.query
        })
      }
      return
    }
    
    if (this.isUpdating) {
      this.log('路由更新中，跳过重复请求', { source: request.source })
      return
    }
    
    this.isUpdating = true
    this.log('处理路由更新请求', { request })
    
    try {
      // 1. 构建新的查询参数（应用参数保留策略）
      const newQuery = this.buildQuery(request)
      
      // 2. 执行路由更新
      const targetPath = request.path || this.route.path
      const replace = request.replace !== false
      
      this.log('执行路由更新', { 
        path: targetPath, 
        query: newQuery, 
        replace,
        source: request.source 
      })
      
      if (replace) {
        await this.router.replace({ path: targetPath, query: newQuery })
      } else {
        await this.router.push({ path: targetPath, query: newQuery })
      }
      
      // 3. 🔥 更新当前 Tab 的路由状态
      // Tab 切换时，使用 request.meta.newTabId（如果存在）来保存新 Tab 的路由状态
      // workspace-node-click 时，等待 syncRouteToTab 完成后再保存（通过 RouteEvent.updateCompleted 事件）
      // 否则，使用 getCurrentTabId() 获取当前 Tab ID
      if (request.source === 'tab-switch') {
        // Tab 切换时，使用 request.meta.newTabId 保存新 Tab 的路由状态
        const newTabId = (request as any).meta?.newTabId
        if (newTabId) {
          // 🔥 验证：确保保存的路由状态与 newTabId 对应的 Tab 路径匹配
          // 如果 targetPath 不匹配 newTabId 对应的 Tab 路径，说明恢复的状态是错误的，不应该保存
          // 但是，由于 targetPath 是从恢复的状态中获取的，所以应该是匹配的
          // 这里我们直接保存，因为 targetPath 就是从 targetRouteState 中获取的
          this.tabStateManager.saveTabRouteState(newTabId, {
            path: targetPath,
            query: newQuery
          })
          this.log('更新 Tab 路由状态（Tab 切换）', { tabId: newTabId, route: { path: targetPath, query: newQuery } })
        }
      } else if (request.source === 'workspace-node-click') {
        // 🔥 workspace-node-click 时，不立即保存 Tab 路由状态
        // 因为此时 Tab 可能还没有激活，getCurrentTabId() 返回的是旧 Tab ID
        // 路由状态会在 syncRouteToTab 完成后，通过 RouteEvent.updateCompleted 事件保存
        this.log('workspace-node-click：等待 syncRouteToTab 完成后再保存 Tab 路由状态')
      } else {
        // 用户操作、link 跳转等需要更新 Tab 的路由状态
        const currentTabId = this.getCurrentTabId()
        if (currentTabId) {
          // 🔥 验证：确保保存的路由状态与 currentTabId 对应的 Tab 路径匹配
          // 如果 targetPath 不匹配 currentTabId 对应的 Tab 路径，说明路由状态不一致，不应该保存
          // 但是，由于这些操作通常是直接更新路由的，所以应该是匹配的
          // 这里我们直接保存
          this.tabStateManager.saveTabRouteState(currentTabId, {
            path: targetPath,
            query: newQuery
          })
          this.log('更新 Tab 路由状态', { tabId: currentTabId, route: { path: targetPath, query: newQuery } })
        }
      }
      
      // 4. 发出更新完成事件
      this.eventBus.emit(RouteEvent.updateCompleted, {
        path: targetPath,
        query: newQuery,
        source: request.source
      })
      
      this.log('路由更新完成', { path: targetPath, source: request.source })
    } catch (error) {
      Logger.error('RouteManager', '路由更新失败', error)
    } finally {
      // 使用 nextTick 确保路由更新完成后再重置标志
      await nextTick()
      this.isUpdating = false
    }
  }
  
  /**
   * 构建查询参数（应用参数保留策略）
   * 
   * 🔥 注意：如果 request.query 已经包含了完整的查询参数（如 TableView 的 syncToURL），
   * 则直接使用 request.query，不再应用参数保留策略。
   * 否则，根据 preserveParams 策略从当前路由中保留参数，然后合并 request.query。
   */
  private buildQuery(request: RouteUpdateRequest): Record<string, string | string[]> {
    const preserve = request.preserveParams || {}
    const currentQuery = { ...this.route.query }
    
    // 🔥 如果 request.query 已经包含了完整的查询参数（如 TableView 的 syncToURL），
    // 则直接使用，不再应用参数保留策略
    // 注意：TableView 的 syncToURL 已经通过 preserveExistingParams 计算好了完整的 newQuery
    if (request.query && Object.keys(request.query).length > 0) {
      // 检查是否是 link 跳转
      if (preserve.linkNavigation) {
        this.log('link 跳转：保留所有参数（除了 _link_type），然后合并新参数')
        const result: Record<string, string | string[]> = {}
        // 先保留当前路由的所有参数（除了 _link_type）
        Object.keys(currentQuery).forEach(key => {
          if (key !== '_link_type') {
            const value = currentQuery[key]
            if (value !== null && value !== undefined) {
              result[key] = Array.isArray(value) 
                ? value.filter(v => v !== null).map(v => String(v))
                : String(value)
            }
          }
        })
        // 然后合并新参数（覆盖旧参数）
        // 注意：request.query 已经包含了完整的参数（包括 preserveExistingParams 的结果）
        Object.assign(result, this.normalizeQuery(request.query))
        return result
      } else {
        // 非 link 跳转：直接使用 request.query（已经包含了 preserveExistingParams 的结果）
        // 注意：TableView 的 syncToURL 已经通过 preserveExistingParams 计算好了完整的 newQuery
        this.log('使用完整的查询参数（已包含参数保留逻辑）', { query: request.query })
        return this.normalizeQuery(request.query)
      }
    }
    
    // 🔥 如果 request.query 为空或未提供，则根据 preserveParams 策略从当前路由中保留参数
    const newQuery: Record<string, string | string[]> = {}
    
    // link 跳转：保留所有参数（除了临时参数）
    if (preserve.linkNavigation) {
      this.log('link 跳转：保留所有参数')
      Object.keys(currentQuery).forEach(key => {
        if (key !== '_link_type') {
          const value = currentQuery[key]
          if (value !== null && value !== undefined) {
            newQuery[key] = Array.isArray(value) 
              ? value.filter(v => v !== null).map(v => String(v))
              : String(value)
          }
        }
      })
      return newQuery
    }
    
    // 非 link 跳转：应用参数保留策略
    Object.keys(currentQuery).forEach(key => {
      const value = currentQuery[key]
      if (value === null || value === undefined) return
      
      let shouldPreserve = false
      
      // 保留状态参数（_ 开头）
      if (preserve.state !== false && key.startsWith('_')) {
        shouldPreserve = true
      }
      // 保留 table 参数
      else if (preserve.table && TABLE_PARAM_KEYS.includes(key as any)) {
        shouldPreserve = true
      }
      // 保留搜索参数
      else if (preserve.search && SEARCH_PARAM_KEYS.includes(key as any)) {
        shouldPreserve = true
      }
      // 保留自定义参数
      else if (preserve.custom?.includes(key)) {
        shouldPreserve = true
      }
      
      if (shouldPreserve) {
        newQuery[key] = Array.isArray(value) 
          ? value.filter(v => v !== null).map(v => String(v))
          : String(value)
      }
    })
    
    return newQuery
  }
  
  /**
   * 规范化查询参数
   */
  private normalizeQuery(query: Record<string, any>): Record<string, string | string[]> {
    const normalized: Record<string, string | string[]> = {}
    Object.keys(query).forEach(key => {
      const value = query[key]
      if (value !== null && value !== undefined) {
        normalized[key] = Array.isArray(value) 
          ? value.filter(v => v !== null).map(v => String(v))
          : String(value)
      }
    })
    return normalized
  }
  
  /**
   * 请求更新路由（公开方法）
   */
  requestUpdate(request: RouteUpdateRequest): void {
    this.eventBus.emit(RouteEvent.updateRequested, request)
  }
  
  /**
   * 获取当前路由
   */
  getCurrentRoute(): { path: string, query: Record<string, any> } {
    return {
      path: this.route.path,
      query: { ...this.route.query }
    }
  }
  
  /**
   * 检查是否是 link 跳转
   */
  isLinkNavigation(): boolean {
    return this.route.query._link_type === 'table' || this.route.query._link_type === 'form'
  }
  
  /**
   * 获取 Tab 状态管理器（用于外部访问）
   */
  getTabStateManager(): TabStateManager {
    return this.tabStateManager
  }
}

