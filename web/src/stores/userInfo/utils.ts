/**
 * 用户信息缓存工具函数
 */
import type { UserInfo } from '@/types'
import { USER_INFO_CACHE_CONFIG } from './config'

/**
 * 缓存项接口
 */
export interface CacheItem {
  user: UserInfo
  timestamp: number  // 缓存时间戳
}

/**
 * 检查缓存项是否过期
 */
export function isCacheExpired(cacheItem: CacheItem): boolean {
  const now = Date.now()
  return (now - cacheItem.timestamp) > USER_INFO_CACHE_CONFIG.CACHE_EXPIRY_TIME
}

/**
 * 等待加载完成的工具函数
 * @param checkFn 检查函数，返回 true 表示加载完成
 * @param timeout 超时时间（毫秒）
 * @returns Promise，加载完成时 resolve
 */
export function waitForLoading(
  checkFn: () => boolean,
  timeout: number = USER_INFO_CACHE_CONFIG.API_TIMEOUT
): Promise<void> {
  return new Promise<void>((resolve) => {
    const startTime = Date.now()
    const checkInterval = setInterval(() => {
      if (checkFn()) {
        clearInterval(checkInterval)
        resolve()
      } else if (Date.now() - startTime > timeout) {
        // 超时后不再等待
        clearInterval(checkInterval)
        resolve()
      }
    }, USER_INFO_CACHE_CONFIG.LOADING_CHECK_INTERVAL)
    
    // 最大超时保护
    setTimeout(() => {
      clearInterval(checkInterval)
      resolve()
    }, USER_INFO_CACHE_CONFIG.LOADING_MAX_TIMEOUT)
  })
}

/**
 * 从对象构建 Map（用于序列化）
 */
export function mapToObject<T>(map: Map<string, T>): Record<string, T> {
  const obj: Record<string, T> = {}
  map.forEach((value, key) => {
    obj[key] = value
  })
  return obj
}

/**
 * 从对象构建 Map（用于反序列化）
 */
export function objectToMap<T>(obj: Record<string, T>): Map<string, T> {
  const map = new Map<string, T>()
  Object.entries(obj).forEach(([key, value]) => {
    map.set(key, value)
  })
  return map
}

