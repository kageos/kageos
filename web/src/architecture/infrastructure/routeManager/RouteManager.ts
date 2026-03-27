/**
 * RouteManager - 路由管理器
 * 
 * ============================================
 * 📋 需求说明
 * ============================================
 * 
 * 1. **统一路由更新入口**：
 *    - 所有路由更新都通过 RouteManager 处理
 *    - 统一处理参数保留逻辑
 *    - 防止路由更新循环
 * 
 * 2. **参数保留策略**：
 *    - 表格参数：page、page_size、sorts
 *    - 搜索参数：eq、like、in、contains 等
 *    - 状态参数：_ 开头的参数（如 _tab、_id）
 *    - 自定义参数：通过 `custom` 配置指定
 * 
 * 3. **事件驱动**：
 *    - 监听 `RouteEvent.updateRequested` 事件
 *    - 处理路由更新请求
 *    - 发出 `RouteEvent.updateCompleted` 事件
 *    - 监听 Vue Router 变化，发出 `RouteEvent.routeChanged` 事件
 * 
 * ============================================
 * 🎯 设计思路
 * ============================================
 * 
 * 1. **单一职责**：
 *    - 唯一的路由更新入口
 *    - 统一处理所有路由更新逻辑
 *    - 避免分散的路由更新代码
 * 
 * 2. **参数保留机制**：
 *    - 使用 `preserveParams` 配置保留哪些参数
 *    - 支持表格参数、搜索参数、状态参数、自定义参数
 *    - 合并新参数和保留参数
 * 
 * 3. **循环防护**：
 *    - 使用 `isUpdating` 标志防止路由更新循环
 *    - 程序触发的路由更新不触发事件
 *    - 用户触发的路由更新才触发事件
 * 
 * ============================================
 * 📝 关键功能
 * ============================================
 * 
 * 1. **handleUpdateRequest**：
 *    - 处理路由更新请求事件
 *    - 构建查询参数（合并新参数和保留参数）
 *    - 执行路由更新
 *    - 发出路由更新完成事件
 * 
 * 2. **buildQuery**：
 *    - 构建查询参数
 *    - 合并新参数和保留参数
 *    - 处理参数类型转换
 * 
 * 3. **监听路由变化**：
 *    - 监听 Vue Router 的 `route` 变化
 *    - 发出 `RouteEvent.routeChanged` 事件
 *    - 区分程序触发和用户触发
 * 
 * ============================================
 * ⚠️ 注意事项
 * ============================================
 * 
 * 1. **参数保留**：
 *    - 必须正确配置 `preserveParams`
 *    - 表格参数、搜索参数等必须保留
 *    - 避免丢失重要的状态信息
 * 
 * 2. **循环防护**：
 *    - 必须使用 `isUpdating` 标志
 *    - 程序触发的路由更新不触发事件
 *    - 避免无限循环
 * 
 * 3. **参数类型**：
 *    - 查询参数必须是字符串或字符串数组
 *    - 复杂类型需要序列化为 JSON
 *    - 空值不添加到 URL
 * 
 * ============================================
 * 📚 相关文档
 * ============================================
 * 
 * - 路由事件类型：`web/src/architecture/domain/interfaces/IEventBus.ts`
 * - URL 参数工具：`web/src/utils/urlParams.ts`
 */

import { watch, nextTick } from 'vue'
import type { Router, RouteLocationNormalized } from 'vue-router'
import type { IEventBus } from '../../domain/interfaces/IEventBus'
import { RouteEvent } from '../../domain/interfaces/IEventBus'
import { TABLE_PARAM_KEYS, SEARCH_PARAM_KEYS } from '@/utils/urlParams'
import { isLinkNavigation as isLinkNavCheck } from '@/utils/linkNavigation'
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
  private isUpdating = false  // 防止循环更新
  private enableDebugLog = false  // 调试日志开关
  private stopRouteWatch: (() => void) | null = null
  private removeUpdateListener: (() => void) | null = null
  private readonly handleUpdateRequestListener = (payload?: RouteUpdateRequest): void => {
    if (!payload) return
    void this.handleUpdateRequest(payload)
  }

  constructor(
    router: Router,
    route: RouteLocationNormalized,
    eventBus: IEventBus
  ) {
    this.router = router
    this.route = route
    this.eventBus = eventBus

    this.setupRouteWatch()
    this.setupUpdateListener()
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
    this.stopRouteWatch?.()
    this.stopRouteWatch = watch(() => [this.route.path, this.route.query] as const, ([newPath, newQuery], oldValue?: readonly [string, Record<string, any>]) => {
      const [oldPath, oldQuery] = oldValue ?? [newPath, newQuery]
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
    this.removeUpdateListener?.()
    this.removeUpdateListener = this.eventBus.on(RouteEvent.updateRequested, this.handleUpdateRequestListener)
  }
  
  /**
   * 清理监听器
   */
  destroy(): void {
    this.removeUpdateListener?.()
    this.removeUpdateListener = null
    this.stopRouteWatch?.()
    this.stopRouteWatch = null
    this.log('RouteManager 已销毁')
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
    
    try {
      // 1. 构建新的查询参数（应用参数保留策略）
      const newQuery = this.buildQuery(request)
      
      // 2. 执行路由更新
      const targetPath = request.path || this.route.path
      const replace = request.replace !== false
      
      this.log('🔍 [handleUpdateRequest] 执行路由更新', { 
        path: targetPath, 
        query: newQuery,
        queryKeys: Object.keys(newQuery),
        queryLength: Object.keys(newQuery).length,
        replace,
        source: request.source 
      })
      
      if (replace) {
        await this.router.replace({ path: targetPath, query: newQuery })
      } else {
        await this.router.push({ path: targetPath, query: newQuery })
      }
      
      // 🔥 Tab 功能已删除，不再保存 Tab 路由状态
      
      // 4. 发出更新完成事件
      this.eventBus.emit(RouteEvent.updateCompleted, {
        path: targetPath,
        query: newQuery,
        source: request.source
      })
      
      this.log('🔍 [handleUpdateRequest] 路由更新完成，已发出 updateCompleted 事件', { 
        path: targetPath, 
        query: newQuery,
        queryKeys: Object.keys(newQuery),
        queryLength: Object.keys(newQuery).length,
        source: request.source 
      })
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
    
    this.log('🔍 [buildQuery] 开始构建查询参数', {
      source: request.source,
      requestQuery: request.query,
      requestQueryKeys: request.query ? Object.keys(request.query) : [],
      requestQueryLength: request.query ? Object.keys(request.query).length : 0,
      preserveParams: preserve,
      currentQuery: currentQuery,
      currentQueryKeys: Object.keys(currentQuery),
      currentPath: this.route.path
    })
    
    // 🔥 如果 request.query 已经包含了完整的查询参数（如 TableView 的 syncToURL），
    // 则直接使用，不再应用参数保留策略
    // 注意：TableView 的 syncToURL 已经通过 preserveExistingParams 计算好了完整的 newQuery
    // 🔥 修复：如果 request.query 是空对象 {}，且所有 preserveParams 都是 false，直接返回空对象
    if (request.query && Object.keys(request.query).length > 0) {
      this.log('🔍 [buildQuery] request.query 不为空，进入第一个分支', {
        queryKeys: Object.keys(request.query),
        queryLength: Object.keys(request.query).length
      })
      // 检查是否是 link 跳转
      if (preserve.linkNavigation) {
        // 🔥 特殊处理：workspace-routing-clear-link-type 请求的 query 已经包含了所有参数（除了 _link_type）
        // 直接使用 request.query，不需要再从 currentQuery 中合并参数
        if (request.source === 'workspace-routing-clear-link-type') {
          this.log('link 跳转（清除 _link_type）：直接使用 request.query，不合并当前路由参数', { 
            requestQuery: request.query 
          })
          // 只过滤 table 参数，保留其他所有参数（包括 eq、in 等搜索参数）
          const result: Record<string, string | string[]> = {}
          const normalizedQuery = this.normalizeQuery(request.query)
          Object.keys(normalizedQuery).forEach(key => {
            if (!TABLE_PARAM_KEYS.includes(key as any)) {
              const value = normalizedQuery[key]
              if (value !== undefined && value !== null && value !== '') {
                result[key] = value
              }
            }
          })
          this.log('link 跳转（清除 _link_type）：最终查询参数', { query: result })
          return result
        }
        
        this.log('link 跳转：优先使用 request.query 中的参数（来自 link URL），然后保留当前路由的非 table 参数')
        const result: Record<string, string | string[]> = {}
        
        // 🔥 修复：link 跳转时，优先使用 request.query 中的参数（这些参数来自 link URL，是用户明确指定的）
        // 先处理 request.query 中的参数（保留所有参数，包括 eq、in 等搜索参数，只清除 table 参数）
        const normalizedQuery = this.normalizeQuery(request.query)
        Object.keys(normalizedQuery).forEach(key => {
          // 🔥 只过滤 table 参数（page, page_size, sorts），保留所有其他参数（包括 eq、in 等搜索参数）
          if (!TABLE_PARAM_KEYS.includes(key as any)) {
            const value = normalizedQuery[key]
            if (value !== undefined && value !== null && value !== '') {
              result[key] = value
            }
          }
        })
        
        this.log('link 跳转：处理 request.query 后的结果', { 
          requestQuery: request.query, 
          normalizedQuery, 
          result: { ...result } 
        })
        
        // 然后保留当前路由的参数（除了 _link_type、_node_type、table 参数和已在 request.query 中的参数）
        // 🔥 这样确保 link URL 中的参数优先级最高，不会被当前路由的参数覆盖
        Object.keys(currentQuery).forEach(key => {
          // 跳过已在 request.query 中的参数（避免覆盖 link URL 中的参数）
          if (normalizedQuery.hasOwnProperty(key)) {
            return
          }
          if (key !== '_link_type' && key !== '_node_type' && !TABLE_PARAM_KEYS.includes(key as any)) {
            const value = currentQuery[key]
            if (value !== null && value !== undefined) {
              result[key] = Array.isArray(value) 
                ? value.filter(v => v !== null).map(v => String(v))
                : String(value)
            }
          }
        })
        
        this.log('link 跳转：最终查询参数', { query: result })
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
    
    this.log('🔍 [buildQuery] request.query 为空或未提供，进入第二个分支', {
      hasRequestQuery: !!request.query,
      requestQueryType: request.query ? typeof request.query : 'undefined',
      requestQueryIsObject: request.query ? (request.query instanceof Object) : false,
      requestQueryKeys: request.query ? Object.keys(request.query) : [],
      requestQueryLength: request.query ? Object.keys(request.query).length : 0,
      preserveParams: preserve
    })
    
    // 🔥 如果 request.query 是空对象 {}，且所有 preserveParams 都是 false，直接返回空对象（清空所有参数）
    // 注意：这里需要检查 request.query 是否是空对象，如果是空对象，说明调用者明确要求清空所有参数
    if (request.query && Object.keys(request.query).length === 0) {
      this.log('🔍 [buildQuery] request.query 是空对象 {}', {
        preserveParams: preserve,
        linkNavigation: preserve.linkNavigation,
        table: preserve.table,
        search: preserve.search,
        state: preserve.state,
        custom: preserve.custom
      })
      
      // request.query 是空对象 {}，说明调用者明确要求清空所有参数
      // 检查 preserveParams，如果所有都是 false，直接返回空对象
      const shouldClear = !preserve.linkNavigation && 
          preserve.table !== true && 
          preserve.search !== true && 
          preserve.state === false && 
          (!preserve.custom || preserve.custom.length === 0)
      
      this.log('🔍 [buildQuery] 检查是否应该清空参数', {
        shouldClear,
        linkNavigation: preserve.linkNavigation,
        table: preserve.table,
        search: preserve.search,
        state: preserve.state,
        custom: preserve.custom
      })
      
      if (shouldClear) {
        this.log('✅ [buildQuery] request.query 是空对象且所有 preserveParams 都是 false，清空所有查询参数，返回空对象')
        return newQuery
      } else {
        this.log('⚠️ [buildQuery] request.query 是空对象但 preserveParams 不是全部 false，继续处理')
      }
    }
    
    // 🔥 如果所有 preserveParams 都是 false，且没有自定义参数，直接返回空对象（清空所有参数）
    const allPreserveFalse = !preserve.linkNavigation && 
        preserve.table !== true && 
        preserve.search !== true && 
        preserve.state === false && 
        (!preserve.custom || preserve.custom.length === 0)
    
    this.log('🔍 [buildQuery] 检查所有 preserveParams 是否都是 false', {
      allPreserveFalse,
      linkNavigation: preserve.linkNavigation,
      table: preserve.table,
      search: preserve.search,
      state: preserve.state,
      custom: preserve.custom
    })
    
    if (allPreserveFalse) {
      this.log('✅ [buildQuery] 所有 preserveParams 都是 false，清空所有查询参数，返回空对象')
      return newQuery
    }
    
    // link 跳转：保留参数（除了临时参数和 table 参数）
    // 🔥 修复：link 跳转到 form 函数时，不应该保留 table 参数（page, page_size, sorts）
    if (preserve.linkNavigation) {
      this.log('link 跳转：保留参数（除了 _link_type、_node_type 和 table 参数）')
      Object.keys(currentQuery).forEach(key => {
        if (key !== '_link_type' && key !== '_node_type' && !TABLE_PARAM_KEYS.includes(key as any)) {
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
      
      // 🔥 排除 _node_type 参数（函数组专用参数，不应该被保留）
      if (key === '_node_type') {
        return
      }
      
      let shouldPreserve = false
      
      // 保留状态参数（_ 开头，但排除 _node_type）
      // 🔥 修复：只有当 preserve.state 明确为 true 时才保留，false 时不保留
      if (preserve.state === true && key.startsWith('_')) {
        shouldPreserve = true
      }
      // 保留 table 参数
      else if (preserve.table === true && TABLE_PARAM_KEYS.includes(key as any)) {
        shouldPreserve = true
      }
      // 保留搜索参数
      else if (preserve.search === true && SEARCH_PARAM_KEYS.includes(key as any)) {
        shouldPreserve = true
      }
      // 保留自定义参数
      else if (preserve.custom && preserve.custom.includes(key)) {
        shouldPreserve = true
      }
      
      if (shouldPreserve) {
        newQuery[key] = Array.isArray(value) 
          ? value.filter(v => v !== null).map(v => String(v))
          : String(value)
      }
    })
    
    this.log('🔍 [buildQuery] 最终构建的查询参数', {
      newQuery,
      newQueryKeys: Object.keys(newQuery),
      newQueryLength: Object.keys(newQuery).length
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
    return isLinkNavCheck(this.route.query)
  }
}
