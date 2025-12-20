/**
 * 路由工具函数
 * 
 * 职责：
 * - 统一管理路由更新
 * - 通过事件总线触发路由更新，而不是直接使用 router
 * - 简化路由更新逻辑
 */

import { eventBus, RouteEvent } from '@/architecture/infrastructure/eventBus'
import type { RouteUpdateRequest } from '@/architecture/infrastructure/routeManager/RouteManager'

/**
 * 更新路由查询参数
 * 
 * @param query 新的查询参数
 * @param options 选项
 */
export function updateRouteQuery(
  query: Record<string, any>,
  options: {
    replace?: boolean
    preserveParams?: RouteUpdateRequest['preserveParams']
    source?: string
  } = {}
): void {
  eventBus.emit(RouteEvent.updateRequested, {
    query,
    replace: options.replace !== false, // 默认使用 replace
    preserveParams: options.preserveParams || {},
    source: options.source || 'route-utils'
  })
}

/**
 * 更新路由路径和查询参数
 * 
 * @param path 新的路径
 * @param query 新的查询参数
 * @param options 选项
 */
export function updateRoute(
  path: string,
  query: Record<string, any> = {},
  options: {
    replace?: boolean
    preserveParams?: RouteUpdateRequest['preserveParams']
    source?: string
  } = {}
): void {
  eventBus.emit(RouteEvent.updateRequested, {
    path,
    query,
    replace: options.replace !== false, // 默认使用 replace
    preserveParams: options.preserveParams || {},
    source: options.source || 'route-utils'
  })
}

/**
 * 删除查询参数
 * 
 * @param keys 要删除的参数键
 * @param options 选项
 */
export function removeRouteQuery(
  keys: string[],
  options: {
    replace?: boolean
    preserveParams?: RouteUpdateRequest['preserveParams']
    source?: string
  } = {}
): void {
  // 从当前路由获取查询参数
  const currentQuery = { ...(window.location.search ? new URLSearchParams(window.location.search) : {}) }
  
  // 删除指定的键
  const newQuery: Record<string, any> = {}
  Object.keys(currentQuery).forEach(key => {
    if (!keys.includes(key)) {
      newQuery[key] = currentQuery[key]
    }
  })
  
  updateRouteQuery(newQuery, options)
}

