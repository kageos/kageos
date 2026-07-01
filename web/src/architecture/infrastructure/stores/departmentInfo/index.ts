/**
 * 部门信息缓存 Store
 * 
 * 功能：
 * - 统一管理所有部门信息的查询和缓存
 * - 避免重复查询相同的部门信息
 * - 支持缓存过期机制（默认30分钟）
 * - 支持手动刷新缓存
 * - 🔥 支持持久化到 localStorage（刷新后缓存仍然有效）
 * 
 * 🔥 优化策略：
 * 1. **懒加载刷新**：不过期缓存不会主动刷新，只有真正使用时发现过期才刷新
 *    这样可以避免在大量部门信息过期时一次性刷新造成服务压力
 * 2. **降级策略**：如果接口响应慢（超过300ms）或失败，先使用过期缓存值，不阻塞用户
 *    后台异步刷新，等接口返回后再更新，提升用户体验
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getDepartmentsByPaths } from '@/architecture/infrastructure/api/department'
import type { Department } from '@/architecture/infrastructure/api/department'
import { DEPARTMENT_INFO_CACHE_CONFIG } from './config'
import { 
  type CacheItem, 
  isCacheExpired, 
  waitForLoading, 
  mapToObject, 
  objectToMap 
} from './utils'

export const useDepartmentInfoStore = defineStore('departmentInfo', () => {
  // 部门信息缓存（full_code_path -> CacheItem）
  const departmentInfoCache = ref<Map<string, CacheItem>>(new Map())
  
  // 正在查询的路径集合（避免重复查询）
  const loadingPaths = ref<Set<string>>(new Set())
  
  /**
   * 🔥 清除过期的缓存（懒加载：只在真正使用时才清除，不主动批量清除）
   */
  function clearExpiredCacheForPaths(paths: string[]): void {
    const now = Date.now()
    let clearedCount = 0
    
    paths.forEach(path => {
      const cacheItem = departmentInfoCache.value.get(path)
      if (cacheItem && (now - cacheItem.timestamp) > DEPARTMENT_INFO_CACHE_CONFIG.CACHE_EXPIRY_TIME) {
        departmentInfoCache.value.delete(path)
        clearedCount++
      }
    })
    
    if (clearedCount > 0) {
      saveCacheToStorage()
    }
  }
  
  /**
   * 🔥 从 localStorage 恢复缓存（在 store 初始化时调用）
   */
  function restoreCacheFromStorage(): void {
    try {
      const stored = localStorage.getItem(DEPARTMENT_INFO_CACHE_CONFIG.STORAGE_KEY)
      if (stored) {
        const data = JSON.parse(stored)
        if (data?.departmentInfoCache) {
          // 将存储的对象转换为 Map
          const map = objectToMap<CacheItem>(data.departmentInfoCache)
          departmentInfoCache.value = map
        }
      }
    } catch (error) {
      console.error('[DepartmentInfoStore] 恢复缓存失败:', error)
    }
  }
  
  /**
   * 🔥 保存缓存到 localStorage
   */
  function saveCacheToStorage(): void {
    try {
      // 🔥 保存所有缓存（包括过期的），用于降级策略
      const cacheObj = mapToObject(departmentInfoCache.value)
      localStorage.setItem(DEPARTMENT_INFO_CACHE_CONFIG.STORAGE_KEY, JSON.stringify({ departmentInfoCache: cacheObj }))
    } catch (error) {
      console.error('[DepartmentInfoStore] 保存缓存失败:', error)
    }
  }
  
  /**
   * 等待部门加载完成（带超时）
   */
  async function waitForDepartmentLoading(path: string): Promise<Department | null> {
    return new Promise<Department | null>((resolve) => {
      waitForLoading(() => {
        const cacheItem = departmentInfoCache.value.get(path)
        if (cacheItem && !isCacheExpired(cacheItem)) {
          resolve(cacheItem.department)
          return true
        }
        // 如果超时，返回过期缓存（降级策略）
        const expiredCache = departmentInfoCache.value.get(path)
        if (expiredCache) {
          resolve(expiredCache.department)
          return true
        }
        return false
      }, DEPARTMENT_INFO_CACHE_CONFIG.API_TIMEOUT).then(() => {
        const expiredCache = departmentInfoCache.value.get(path)
        resolve(expiredCache?.department || null)
      })
    })
  }
  
  /**
   * 获取单个部门信息
   * 🔥 降级策略：如果缓存过期但接口慢，先返回过期值，后台异步刷新
   */
  async function getDepartmentInfo(path: string, forceRefresh: boolean = false): Promise<Department | null> {
    if (!path) return null
    
    // 如果强制刷新，清除该部门的缓存
    if (forceRefresh) {
      departmentInfoCache.value.delete(path)
      saveCacheToStorage()
    } else {
      // 检查缓存
      const cacheItem = departmentInfoCache.value.get(path)
      if (cacheItem && !isCacheExpired(cacheItem)) {
        return cacheItem.department
      }
      // 🔥 缓存已过期，但不立即清除，先返回过期值，后台异步刷新（降级策略）
    }
    
    // 如果正在查询，等待查询完成（但不超过超时时间）
    if (loadingPaths.value.has(path)) {
      return waitForDepartmentLoading(path)
    }
    
    // 🔥 如果有过期缓存，先返回过期值，然后异步刷新（降级策略）
    const expiredCache = departmentInfoCache.value.get(path)
    if (expiredCache && isCacheExpired(expiredCache)) {
      // 异步刷新，不阻塞
      batchGetDepartmentInfo([path], false).catch(error => {
        console.error(`[DepartmentInfoStore] 后台刷新部门 ${path} 失败:`, error)
      })
      return expiredCache.department
    }
    
    // 批量查询（即使只有一个部门，也使用批量接口）
    return batchGetDepartmentInfo([path], forceRefresh).then(departments => departments[0] || null)
  }
  
  /**
   * 分类路径：有效缓存、过期缓存、正在加载、未缓存
   */
  interface ClassifiedPaths {
    cached: Department[]
    expired: Array<{ path: string; department: Department }>
    loading: string[]
    uncached: string[]
  }
  
  function classifyPaths(
    paths: string[],
    forceRefresh: boolean
  ): ClassifiedPaths {
    const result: ClassifiedPaths = {
      cached: [],
      expired: [],
      loading: [],
      uncached: []
    }
    
    paths.forEach(path => {
      if (!path) return
      
      if (forceRefresh) {
        result.uncached.push(path)
        return
      }
      
      if (loadingPaths.value.has(path)) {
        result.loading.push(path)
        return
      }
      
      const cacheItem = departmentInfoCache.value.get(path)
      if (cacheItem) {
        if (isCacheExpired(cacheItem)) {
          result.expired.push({ path, department: cacheItem.department })
        } else {
          result.cached.push(cacheItem.department)
        }
      } else {
        result.uncached.push(path)
      }
    })
    
    return result
  }
  
  /**
   * 等待正在加载的部门完成
   */
  async function waitForLoadingDepartments(paths: string[]): Promise<Department[]> {
    const loadedDepartments: Department[] = []
    
    await waitForLoading(() => {
      const allLoaded = paths.every(path => {
        const cacheItem = departmentInfoCache.value.get(path)
        return cacheItem && !isCacheExpired(cacheItem)
      })
      return allLoaded
    }, DEPARTMENT_INFO_CACHE_CONFIG.API_TIMEOUT)
    
    // 加载完成后，从缓存中获取
    paths.forEach(path => {
      const cacheItem = departmentInfoCache.value.get(path)
      if (cacheItem) {
        if (!isCacheExpired(cacheItem)) {
          loadedDepartments.push(cacheItem.department)
        }
      }
    })
    
    return loadedDepartments
  }
  
  /**
   * 构建降级结果（过期缓存 + 有效缓存）
   */
  function buildFallbackResult(
    paths: string[],
    cached: Department[],
    expired: Array<{ path: string; department: Department }>
  ): Department[] {
    const result: Department[] = []
    paths.forEach(path => {
      const cachedDept = cached.find(d => d.full_code_path === path)
      if (cachedDept) {
        result.push(cachedDept)
      } else {
        const expiredDept = expired.find(e => e.path === path)
        if (expiredDept) {
          result.push(expiredDept.department)
        }
      }
    })
    return result
  }
  
  /**
   * 批量获取部门信息
   * 🔥 降级策略：如果接口慢或失败，先返回过期缓存值，后台异步刷新
   */
  async function batchGetDepartmentInfo(paths: string[], forceRefresh: boolean = false): Promise<Department[]> {
    if (!paths?.length) return []
    
    // 去重
    const uniquePaths = [...new Set(paths)].filter(Boolean)
    if (!uniquePaths.length) return []
    
    // 🔥 懒加载：只清除本次查询涉及的过期缓存
    clearExpiredCacheForPaths(uniquePaths)
    
    // 分类路径
    const { cached, expired, loading, uncached } = classifyPaths(uniquePaths, forceRefresh)
    
    // 🔥 如果有正在加载的部门，等待它们加载完成（但不超过超时时间）
    if (loading.length > 0) {
      const loadedDepartments = await waitForLoadingDepartments(loading)
      cached.push(...loadedDepartments)
    }
    
    // 🔥 如果所有部门都已缓存或正在加载，直接返回（包括过期缓存）
    if (!uncached.length) {
      return buildFallbackResult(uniquePaths, cached, expired)
    }
    
    // 标记正在查询
    uncached.forEach(path => loadingPaths.value.add(path))
    
    // 🔥 降级策略：构建降级结果
    const fallbackResult = buildFallbackResult(uniquePaths, cached, expired)
    
    // 🔥 使用 Promise.race 实现超时降级
    const fetchPromise = fetchAndUpdateDepartments(uncached)
    
    // 🔥 超时降级：如果接口超过300ms未返回，先返回过期缓存，后台继续刷新
    const timeoutPromise = new Promise<Department[]>((resolve) => {
      setTimeout(() => {
        resolve(fallbackResult)
        // 后台继续等待接口返回（不阻塞）
        fetchPromise.then(() => {
          // 后台刷新完成，已自动更新缓存
        }).catch(() => {
          // 刷新失败不影响，继续使用过期缓存
        })
      }, DEPARTMENT_INFO_CACHE_CONFIG.API_TIMEOUT)
    })
    
    try {
      // 如果接口在超时时间内返回，使用新数据
      await Promise.race([fetchPromise, timeoutPromise])
      return buildResultFromCache(uniquePaths, expired)
    } catch (_error) {
      // 🔥 降级策略：如果接口失败，返回过期缓存
      console.warn('[DepartmentInfoStore] 接口失败，使用过期缓存（降级策略）')
      return fallbackResult
    }
  }
  
  /**
   * 获取并更新部门信息
   */
  async function fetchAndUpdateDepartments(paths: string[]): Promise<Department[]> {
    const response = await getDepartmentsByPaths(paths)
    const loadedDepartments = response.departments || []
    
    const now = Date.now()
    
    // 更新缓存
    loadedDepartments.forEach(dept => {
      if (dept.full_code_path) {
        departmentInfoCache.value.set(dept.full_code_path, {
          department: dept,
          timestamp: now
        })
      }
    })
    
    // 🔥 保存到 localStorage
    saveCacheToStorage()
    
    // 移除查询标记
    paths.forEach(path => loadingPaths.value.delete(path))
    
    return loadedDepartments
  }
  
  /**
   * 从缓存构建结果（按顺序）
   */
  function buildResultFromCache(
    paths: string[],
    expired: Array<{ path: string; department: Department }>
  ): Department[] {
    const result: Department[] = []
    
    paths.forEach(path => {
      const cacheItem = departmentInfoCache.value.get(path)
      if (cacheItem) {
        result.push(cacheItem.department)
      } else {
        // 如果新加载的也没有，尝试使用过期缓存（降级策略）
        const expiredDept = expired.find(e => e.path === path)
        if (expiredDept) {
          result.push(expiredDept.department)
        }
      }
    })
    
    return result
  }
  
  /**
   * 刷新指定部门的缓存
   */
  async function refreshCache(paths?: string[]): Promise<void> {
    if (paths?.length) {
      await batchGetDepartmentInfo(paths, true)
    } else {
      const allPaths = Array.from(departmentInfoCache.value.keys())
      if (allPaths.length > 0) {
        await batchGetDepartmentInfo(allPaths, true)
      }
    }
  }
  
  /**
   * 清除所有缓存
   */
  function clearCache(): void {
    departmentInfoCache.value.clear()
    loadingPaths.value.clear()
    localStorage.removeItem(DEPARTMENT_INFO_CACHE_CONFIG.STORAGE_KEY)
  }
  
  /**
   * 清除指定部门的缓存
   */
  function clearDepartmentCache(paths: string[]): void {
    paths.forEach(path => {
      departmentInfoCache.value.delete(path)
    })
    saveCacheToStorage()
  }
  
  /**
   * 获取缓存统计信息
   */
  function getCacheStats() {
    let validCount = 0
    let expiredCount = 0
    
    departmentInfoCache.value.forEach(cacheItem => {
      if (isCacheExpired(cacheItem)) {
        expiredCount++
      } else {
        validCount++
      }
    })
    
    return {
      total: departmentInfoCache.value.size,
      valid: validCount,
      expired: expiredCount,
      loading: loadingPaths.value.size
    }
  }

  /**
   * 🔥 获取缓存详情列表（用于调试）
   */
  function getCacheDetails() {
    const details: Array<{
      path: string
      name: string
      isExpired: boolean
      cachedTime: number
      expiredTime: number
      age: number
    }> = []
    
    const now = Date.now()
    departmentInfoCache.value.forEach((cacheItem, path) => {
      const isExpired = isCacheExpired(cacheItem)
      const expiredTime = cacheItem.timestamp + DEPARTMENT_INFO_CACHE_CONFIG.CACHE_EXPIRY_TIME
      const age = now - cacheItem.timestamp
      
      details.push({
        path,
        name: cacheItem.department?.name || '',
        isExpired,
        cachedTime: cacheItem.timestamp,
        expiredTime,
        age
      })
    })
    
    // 按过期状态和路径排序（过期在前，然后按路径排序）
    details.sort((a, b) => {
      if (a.isExpired !== b.isExpired) {
        return a.isExpired ? -1 : 1
      }
      return a.path.localeCompare(b.path)
    })
    
    return details
  }
  
  // 🔥 初始化：从 localStorage 恢复缓存
  restoreCacheFromStorage()
  
  return {
    departmentInfoCache: computed(() => {
      // 返回只读的缓存映射（full_code_path -> Department），包括过期项（用于降级）
      const map = new Map<string, Department>()
      departmentInfoCache.value.forEach((cacheItem, path) => {
        map.set(path, cacheItem.department)
      })
      return map
    }),
    getDepartmentInfo,
    batchGetDepartmentInfo,
    refreshCache,
    clearCache,
    clearDepartmentCache,
    getCacheStats,
    getCacheDetails // 🔥 导出缓存详情方法
  }
}, {
  // 🔥 启用持久化，将缓存保存到 localStorage
  persist: {
    key: DEPARTMENT_INFO_CACHE_CONFIG.STORAGE_KEY,
    storage: localStorage,
    // 自定义序列化和反序列化，因为 Map 不能直接序列化
    serializer: {
      deserialize: (value: string) => {
        try {
          const data = JSON.parse(value)
          if (data?.departmentInfoCache) {
            const map = objectToMap<CacheItem>(data.departmentInfoCache)
            return { departmentInfoCache: map }
          }
        } catch (error) {
          console.error('[DepartmentInfoStore] 反序列化失败:', error)
        }
        return { departmentInfoCache: new Map() }
      },
      serialize: (value: any) => {
        try {
          if (value.departmentInfoCache instanceof Map) {
            const cacheObj = mapToObject(value.departmentInfoCache)
            return JSON.stringify({ departmentInfoCache: cacheObj })
          }
        } catch (error) {
          console.error('[DepartmentInfoStore] 序列化失败:', error)
        }
        return JSON.stringify({ departmentInfoCache: {} })
      }
    }
  }
})
