/**
 * 用户信息缓存 Store
 * 
 * 功能：
 * - 统一管理所有用户信息的查询和缓存
 * - 避免重复查询相同的用户信息
 * - 支持缓存过期机制（默认30分钟）
 * - 支持手动刷新缓存
 * - 🔥 支持持久化到 localStorage（刷新后缓存仍然有效）
 * 
 * 🔥 优化策略：
 * 1. **懒加载刷新**：不过期缓存不会主动刷新，只有真正使用时发现过期才刷新
 *    这样可以避免在大量用户信息过期时一次性刷新造成服务压力
 * 2. **降级策略**：如果接口响应慢（超过300ms）或失败，先使用过期缓存值，不阻塞用户
 *    后台异步刷新，等接口返回后再更新，提升用户体验
 */

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getUsersByUsernames } from '@/architecture/infrastructure/api/user'
import type { UserInfo } from '@/architecture/domain/types'
import { USER_INFO_CACHE_CONFIG } from './config'
import { 
  type CacheItem, 
  isCacheExpired, 
  waitForLoading, 
  mapToObject, 
  objectToMap 
} from './utils'

export const useUserInfoStore = defineStore('userInfo', () => {
  // 用户信息缓存（username -> CacheItem）
  const userInfoCache = ref<Map<string, CacheItem>>(new Map())
  
  // 正在查询的用户名集合（避免重复查询）
  const loadingUsernames = ref<Set<string>>(new Set())
  
  /**
   * 🔥 清除过期的缓存（懒加载：只在真正使用时才清除，不主动批量清除）
   */
  function clearExpiredCacheForUsernames(usernames: string[]): void {
    const now = Date.now()
    let clearedCount = 0
    
    usernames.forEach(username => {
      const cacheItem = userInfoCache.value.get(username)
      if (cacheItem && (now - cacheItem.timestamp) > USER_INFO_CACHE_CONFIG.CACHE_EXPIRY_TIME) {
        userInfoCache.value.delete(username)
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
      const stored = localStorage.getItem(USER_INFO_CACHE_CONFIG.STORAGE_KEY)
      if (stored) {
        const data = JSON.parse(stored)
        if (data?.userInfoCache) {
          // 将存储的对象转换为 Map
          const map = objectToMap<CacheItem>(data.userInfoCache)
          userInfoCache.value = map
        }
      }
    } catch (error) {
      console.error('[UserInfoStore] 恢复缓存失败:', error)
    }
  }
  
  /**
   * 🔥 保存缓存到 localStorage
   */
  function saveCacheToStorage(): void {
    try {
      // 🔥 保存所有缓存（包括过期的），用于降级策略
      const cacheObj = mapToObject(userInfoCache.value)
      localStorage.setItem(USER_INFO_CACHE_CONFIG.STORAGE_KEY, JSON.stringify({ userInfoCache: cacheObj }))
    } catch (error) {
      console.error('[UserInfoStore] 保存缓存失败:', error)
    }
  }
  
  /**
   * 等待用户加载完成（带超时）
   */
  async function waitForUserLoading(username: string): Promise<UserInfo | null> {
    return new Promise<UserInfo | null>((resolve) => {
      waitForLoading(() => {
        const cacheItem = userInfoCache.value.get(username)
        if (cacheItem && !isCacheExpired(cacheItem)) {
          resolve(cacheItem.user)
          return true
        }
        if (!loadingUsernames.value.has(username)) {
          // 查询失败，尝试返回过期缓存
          const expiredCache = userInfoCache.value.get(username)
          resolve(expiredCache?.user || null)
          return true
        }
        return false
      }, USER_INFO_CACHE_CONFIG.API_TIMEOUT).then(() => {
        // 超时后，返回过期缓存（降级策略）
        const expiredCache = userInfoCache.value.get(username)
        resolve(expiredCache?.user || null)
      })
    })
  }
  
  /**
   * 获取单个用户信息
   * 🔥 降级策略：如果缓存过期但接口慢，先返回过期值，后台异步刷新
   */
  async function getUserInfo(username: string, forceRefresh: boolean = false): Promise<UserInfo | null> {
    if (!username) return null
    
    // 如果强制刷新，清除该用户的缓存
    if (forceRefresh) {
      userInfoCache.value.delete(username)
      saveCacheToStorage()
    } else {
      // 检查缓存
      const cacheItem = userInfoCache.value.get(username)
      if (cacheItem && !isCacheExpired(cacheItem)) {
        return cacheItem.user
      }
      // 🔥 缓存已过期，但不立即清除，先返回过期值，后台异步刷新（降级策略）
    }
    
    // 如果正在查询，等待查询完成（但不超过超时时间）
    if (loadingUsernames.value.has(username)) {
      return waitForUserLoading(username)
    }
    
    // 🔥 如果有过期缓存，先返回过期值，然后异步刷新（降级策略）
    const expiredCache = userInfoCache.value.get(username)
    if (expiredCache && isCacheExpired(expiredCache)) {
      // 异步刷新，不阻塞
      batchGetUserInfo([username], false).catch(error => {
        console.error(`[UserInfoStore] 后台刷新用户 ${username} 失败:`, error)
      })
      return expiredCache.user
    }
    
    // 批量查询（即使只有一个用户，也使用批量接口）
    return batchGetUserInfo([username], forceRefresh).then(users => users[0] || null)
  }
  
  /**
   * 分类用户名：有效缓存、过期缓存、正在加载、未缓存
   */
  interface ClassifiedUsernames {
    cached: UserInfo[]
    expired: Array<{ username: string; user: UserInfo }>
    loading: string[]
    uncached: string[]
  }
  
  function classifyUsernames(
    usernames: string[],
    forceRefresh: boolean
  ): ClassifiedUsernames {
    const result: ClassifiedUsernames = {
      cached: [],
      expired: [],
      loading: [],
      uncached: []
    }
    
    usernames.forEach(username => {
      if (forceRefresh) {
        userInfoCache.value.delete(username)
        result.uncached.push(username)
      } else {
        const cacheItem = userInfoCache.value.get(username)
        if (cacheItem) {
          if (!isCacheExpired(cacheItem)) {
            result.cached.push(cacheItem.user)
          } else {
            result.expired.push({ username, user: cacheItem.user })
            if (!loadingUsernames.value.has(username)) {
              result.uncached.push(username)
            }
          }
        } else if (loadingUsernames.value.has(username)) {
          result.loading.push(username)
        } else {
          result.uncached.push(username)
        }
      }
    })
    
    return result
  }
  
  /**
   * 等待正在加载的用户完成
   */
  async function waitForLoadingUsers(usernames: string[]): Promise<UserInfo[]> {
    const loadedUsers: UserInfo[] = []
    
    await waitForLoading(() => {
      const allLoaded = usernames.every(username => {
        const cacheItem = userInfoCache.value.get(username)
        return cacheItem && !isCacheExpired(cacheItem)
      })
      return allLoaded
    }, USER_INFO_CACHE_CONFIG.API_TIMEOUT)
    
    // 加载完成后，从缓存中获取
    usernames.forEach(username => {
      const cacheItem = userInfoCache.value.get(username)
      if (cacheItem) {
        if (!isCacheExpired(cacheItem)) {
          loadedUsers.push(cacheItem.user)
        }
      }
    })
    
    return loadedUsers
  }
  
  /**
   * 构建降级结果（过期缓存 + 有效缓存）
   */
  function buildFallbackResult(
    usernames: string[],
    cached: UserInfo[],
    expired: Array<{ username: string; user: UserInfo }>
  ): UserInfo[] {
    const result: UserInfo[] = []
    usernames.forEach(username => {
      const cachedUser = cached.find(u => u.username === username)
      if (cachedUser) {
        result.push(cachedUser)
      } else {
        const expiredUser = expired.find(e => e.username === username)
        if (expiredUser) {
          result.push(expiredUser.user)
        }
      }
    })
    return result
  }
  
  /**
   * 批量获取用户信息
   * 🔥 降级策略：如果接口慢或失败，先返回过期缓存值，后台异步刷新
   */
  async function batchGetUserInfo(usernames: string[], forceRefresh: boolean = false): Promise<UserInfo[]> {
    if (!usernames?.length) return []
    
    // 去重
    const uniqueUsernames = [...new Set(usernames)].filter(Boolean)
    if (!uniqueUsernames.length) return []
    
    // 🔥 懒加载：只清除本次查询涉及的过期缓存
    clearExpiredCacheForUsernames(uniqueUsernames)
    
    // 分类用户名
    const { cached, expired, loading, uncached } = classifyUsernames(uniqueUsernames, forceRefresh)
    
    // 🔥 如果有正在加载的用户，等待它们加载完成（但不超过超时时间）
    if (loading.length > 0) {
      const loadedUsers = await waitForLoadingUsers(loading)
      cached.push(...loadedUsers)
    }
    
    // 🔥 如果所有用户都已缓存或正在加载，直接返回（包括过期缓存）
    if (!uncached.length) {
      return buildFallbackResult(uniqueUsernames, cached, expired)
    }
    
    // 标记正在查询
    uncached.forEach(username => loadingUsernames.value.add(username))
    
    // 🔥 降级策略：构建降级结果
    const fallbackResult = buildFallbackResult(uniqueUsernames, cached, expired)
    
    // 🔥 使用 Promise.race 实现超时降级
    const fetchPromise = fetchAndUpdateUsers(uncached)
    
    // 🔥 超时降级：如果接口超过300ms未返回，先返回过期缓存，后台继续刷新
    const timeoutPromise = new Promise<UserInfo[]>((resolve) => {
      setTimeout(() => {
        resolve(fallbackResult)
        // 后台继续等待接口返回（不阻塞）
        fetchPromise.then(() => {
          // 后台刷新完成，已自动更新缓存
        }).catch(() => {
          // 刷新失败不影响，继续使用过期缓存
        })
      }, USER_INFO_CACHE_CONFIG.API_TIMEOUT)
    })
    
    try {
      // 如果接口在超时时间内返回，使用新数据
      const freshUsers = await Promise.race([fetchPromise, timeoutPromise])
      return buildResultFromCache(uniqueUsernames, freshUsers, expired)
    } catch (error) {
      // 🔥 降级策略：如果接口失败，返回过期缓存
      console.warn('[UserInfoStore] 接口失败，使用过期缓存（降级策略）')
      return fallbackResult
    }
  }
  
  /**
   * 获取并更新用户信息
   */
  async function fetchAndUpdateUsers(usernames: string[]): Promise<UserInfo[]> {
    const response = await getUsersByUsernames(usernames)
    const loadedUsers = response.users || []
    
    const now = Date.now()
    
    // 更新缓存
    loadedUsers.forEach(user => {
      if (user.username) {
        userInfoCache.value.set(user.username, {
          user,
          timestamp: now
        })
      }
    })
    
    // 🔥 保存到 localStorage
    saveCacheToStorage()
    
    // 移除查询标记
    usernames.forEach(username => loadingUsernames.value.delete(username))
    
    return loadedUsers
  }
  
  /**
   * 从缓存构建结果（按顺序）
   */
  function buildResultFromCache(
    usernames: string[],
    freshUsers: UserInfo[],
    expired: Array<{ username: string; user: UserInfo }>
  ): UserInfo[] {
    const result: UserInfo[] = []
    const freshUserMap = new Map(freshUsers.map(u => [u.username, u]))
    
    usernames.forEach(username => {
      const cacheItem = userInfoCache.value.get(username)
      if (cacheItem) {
        result.push(cacheItem.user)
      } else {
        // 如果新加载的也没有，尝试使用过期缓存（降级策略）
        const expiredUser = expired.find(e => e.username === username)
        if (expiredUser) {
          result.push(expiredUser.user)
        }
      }
    })
    
    return result
  }
  
  /**
   * 刷新指定用户的缓存
   */
  async function refreshCache(usernames?: string[]): Promise<void> {
    if (usernames?.length) {
      await batchGetUserInfo(usernames, true)
    } else {
      const allUsernames = Array.from(userInfoCache.value.keys())
      if (allUsernames.length > 0) {
        await batchGetUserInfo(allUsernames, true)
      }
    }
  }
  
  /**
   * 清除所有缓存
   */
  function clearCache(): void {
    userInfoCache.value.clear()
    loadingUsernames.value.clear()
    localStorage.removeItem(USER_INFO_CACHE_CONFIG.STORAGE_KEY)
  }
  
  /**
   * 清除指定用户的缓存
   */
  function clearUserCache(usernames: string[]): void {
    usernames.forEach(username => {
      userInfoCache.value.delete(username)
    })
    saveCacheToStorage()
  }
  
  /**
   * 获取缓存统计信息
   */
  function getCacheStats() {
    let validCount = 0
    let expiredCount = 0
    
    userInfoCache.value.forEach(cacheItem => {
      if (isCacheExpired(cacheItem)) {
        expiredCount++
      } else {
        validCount++
      }
    })
    
    return {
      total: userInfoCache.value.size,
      valid: validCount,
      expired: expiredCount,
      loading: loadingUsernames.value.size
    }
  }

  /**
   * 🔥 获取缓存详情列表（用于调试）
   */
  function getCacheDetails() {
    const details: Array<{
      username: string
      nickname: string
      isExpired: boolean
      cachedTime: number
      expiredTime: number
      age: number
    }> = []
    
    const now = Date.now()
    userInfoCache.value.forEach((cacheItem, username) => {
      const isExpired = isCacheExpired(cacheItem)
      const expiredTime = cacheItem.timestamp + USER_INFO_CACHE_CONFIG.CACHE_EXPIRY_TIME
      const age = now - cacheItem.timestamp
      
      details.push({
        username,
        nickname: cacheItem.user?.nickname || '',
        isExpired,
        cachedTime: cacheItem.timestamp,
        expiredTime,
        age
      })
    })
    
    // 按过期状态和用户名排序（过期在前，然后按用户名排序）
    details.sort((a, b) => {
      if (a.isExpired !== b.isExpired) {
        return a.isExpired ? -1 : 1
      }
      return a.username.localeCompare(b.username)
    })
    
    return details
  }
  
  // 🔥 初始化：从 localStorage 恢复缓存
  restoreCacheFromStorage()
  
  return {
    userInfoCache: computed(() => {
      // 返回只读的缓存映射（username -> UserInfo），包括过期项（用于降级）
      const map = new Map<string, UserInfo>()
      userInfoCache.value.forEach((cacheItem, username) => {
        map.set(username, cacheItem.user)
      })
      return map
    }),
    getUserInfo,
    batchGetUserInfo,
    refreshCache,
    clearCache,
    clearUserCache,
    getCacheStats,
    getCacheDetails // 🔥 导出缓存详情方法
  }
}, {
  // 🔥 启用持久化，将缓存保存到 localStorage
  persist: {
    key: USER_INFO_CACHE_CONFIG.STORAGE_KEY,
    storage: localStorage,
    // 自定义序列化和反序列化，因为 Map 不能直接序列化
    serializer: {
      deserialize: (value: string) => {
        try {
          const data = JSON.parse(value)
          if (data?.userInfoCache) {
            const map = objectToMap<CacheItem>(data.userInfoCache)
            return { userInfoCache: map }
          }
        } catch (error) {
          console.error('[UserInfoStore] 反序列化失败:', error)
        }
        return { userInfoCache: new Map() }
      },
      serialize: (value: any) => {
        try {
          if (value.userInfoCache instanceof Map) {
            const cacheObj = mapToObject(value.userInfoCache)
            return JSON.stringify({ userInfoCache: cacheObj })
          }
        } catch (error) {
          console.error('[UserInfoStore] 序列化失败:', error)
        }
        return JSON.stringify({ userInfoCache: {} })
      }
    }
  }
})
