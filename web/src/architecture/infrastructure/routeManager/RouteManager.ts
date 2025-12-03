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
    
    // 1. 保存当前 Tab 的路由状态
    const currentRoute = this.getCurrentRoute()
    this.tabStateManager.saveTabRouteState(oldTabId, currentRoute)
    this.log('保存 Tab 路由状态', { tabId: oldTabId, route: currentRoute })
    
    // 2. 恢复目标 Tab 的路由状态
    const targetRouteState = this.tabStateManager.getTabRouteState(newTabId)
    if (targetRouteState) {
      this.log('恢复 Tab 路由状态', { tabId: newTabId, route: targetRouteState })
      
      // 发出路由更新请求，恢复目标 Tab 的路由状态
      this.requestUpdate({
        path: targetRouteState.path,
        query: targetRouteState.query,
        source: 'tab-switch',
        preserveParams: {
          linkNavigation: false  // Tab 切换不是 link 跳转，使用目标 Tab 保存的状态
        }
      })
    } else {
      this.log('Tab 没有保存的路由状态，使用默认路由', { tabId: newTabId })
    }
  }
  
  /**
   * 处理路由更新请求
   */
  private async handleUpdateRequest(request: RouteUpdateRequest): Promise<void> {
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
      
      // 3. 🔥 更新当前 Tab 的路由状态（Tab 切换时不需要更新，因为已经恢复了）
      const currentTabId = this.getCurrentTabId()
      if (currentTabId && request.source !== 'tab-switch') {
        // 用户操作、link 跳转等需要更新 Tab 的路由状态
        this.tabStateManager.saveTabRouteState(currentTabId, {
          path: targetPath,
          query: newQuery
        })
        this.log('更新 Tab 路由状态', { tabId: currentTabId, route: { path: targetPath, query: newQuery } })
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

