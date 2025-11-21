/**
 * 用户信息缓存 Store
 * 
 * 功能：
 * - 统一管理所有用户信息的查询和缓存
 * - 避免重复查询相同的用户信息
 * - 支持缓存过期机制（默认5分钟）
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
import { getUsersByUsernames } from '@/api/user'
import type { UserInfo } from '@/types'

/**
 * 缓存项接口
 */
interface CacheItem {
  user: UserInfo
  timestamp: number  // 缓存时间戳
}

export const useUserInfoStore = defineStore('userInfo', () => {
  // 用户信息缓存（username -> CacheItem）
  const userInfoCache = ref<Map<string, CacheItem>>(new Map())
  
  // 正在查询的用户名集合（避免重复查询）
  const loadingUsernames = ref<Set<string>>(new Set())
  
  // 缓存过期时间（毫秒），默认5分钟
  const CACHE_EXPIRY_TIME = 5 * 60 * 1000  // 5分钟
  
  // 🔥 降级策略：接口超时时间（毫秒），超过此时间使用过期缓存
  const API_TIMEOUT = 300  // 300ms
  
  /**
   * 检查缓存项是否过期
   */
  function isCacheExpired(cacheItem: CacheItem): boolean {
    const now = Date.now()
    return (now - cacheItem.timestamp) > CACHE_EXPIRY_TIME
  }
  
  /**
   * 🔥 清除过期的缓存（懒加载：只在真正使用时才清除，不主动批量清除）
   * 注意：这个方法不会主动调用，只在需要时调用
   */
  function clearExpiredCacheForUsernames(usernames: string[]): void {
    const now = Date.now()
    let clearedCount = 0
    
    usernames.forEach(username => {
      const cacheItem = userInfoCache.value.get(username)
      if (cacheItem && (now - cacheItem.timestamp) > CACHE_EXPIRY_TIME) {
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
      const stored = localStorage.getItem('user-info-cache')
      if (stored) {
        const data = JSON.parse(stored)
        if (data && data.userInfoCache) {
          // 将存储的数据转换为 Map
          const map = new Map<string, CacheItem>()
          Object.entries(data.userInfoCache).forEach(([username, cacheItem]: [string, any]) => {
            // 🔥 恢复时不过滤过期缓存，允许使用过期值作为降级策略
            if (cacheItem && cacheItem.timestamp) {
              map.set(username, cacheItem as CacheItem)
            }
          })
          userInfoCache.value = map
          console.log(`[UserInfoStore] 从 localStorage 恢复 ${map.size} 个缓存项（包括过期项）`)
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
      const cacheObj: Record<string, CacheItem> = {}
      userInfoCache.value.forEach((cacheItem, username) => {
        cacheObj[username] = cacheItem
      })
      
      localStorage.setItem('user-info-cache', JSON.stringify({ userInfoCache: cacheObj }))
    } catch (error) {
      console.error('[UserInfoStore] 保存缓存失败:', error)
    }
  }
  
  /**
   * 获取单个用户信息
   * 🔥 降级策略：如果缓存过期但接口慢，先返回过期值，后台异步刷新
   * 
   * @param username 用户名
   * @param forceRefresh 是否强制刷新（忽略缓存）
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
      if (cacheItem) {
        // 如果缓存未过期，直接返回
        if (!isCacheExpired(cacheItem)) {
          return cacheItem.user
        }
        // 🔥 缓存已过期，但不立即清除，先返回过期值，后台异步刷新（降级策略）
      }
    }
    
    // 如果正在查询，等待查询完成（但不超过超时时间）
    if (loadingUsernames.value.has(username)) {
      // 等待查询完成（带超时）
      return new Promise((resolve) => {
        const startTime = Date.now()
        const checkInterval = setInterval(() => {
          const cacheItem = userInfoCache.value.get(username)
          if (cacheItem && !isCacheExpired(cacheItem)) {
            clearInterval(checkInterval)
            resolve(cacheItem.user)
          } else if (!loadingUsernames.value.has(username)) {
            clearInterval(checkInterval)
            // 如果查询失败，尝试返回过期缓存
            const expiredCache = userInfoCache.value.get(username)
            resolve(expiredCache?.user || null)
          } else if (Date.now() - startTime > API_TIMEOUT) {
            // 🔥 超时后，返回过期缓存（降级策略）
            clearInterval(checkInterval)
            const expiredCache = userInfoCache.value.get(username)
            resolve(expiredCache?.user || null)
          }
        }, 50)
        
        // 超时保护（5秒）
        setTimeout(() => {
          clearInterval(checkInterval)
          const expiredCache = userInfoCache.value.get(username)
          resolve(expiredCache?.user || null)
        }, 5000)
      })
    }
    
    // 🔥 如果有过期缓存，先返回过期值，然后异步刷新（降级策略）
    const expiredCache = userInfoCache.value.get(username)
    if (expiredCache && isCacheExpired(expiredCache)) {
      // 异步刷新，不阻塞
      batchGetUserInfo([username], false).catch(error => {
        console.error(`[UserInfoStore] 后台刷新用户 ${username} 失败:`, error)
        // 刷新失败不影响，继续使用过期缓存
      })
      return expiredCache.user
    }
    
    // 批量查询（即使只有一个用户，也使用批量接口）
    return batchGetUserInfo([username], forceRefresh).then(users => users[0] || null)
  }
  
  /**
   * 批量获取用户信息
   * 🔥 降级策略：如果接口慢或失败，先返回过期缓存值，后台异步刷新
   * 
   * @param usernames 用户名列表
   * @param forceRefresh 是否强制刷新（忽略缓存）
   */
  async function batchGetUserInfo(usernames: string[], forceRefresh: boolean = false): Promise<UserInfo[]> {
    if (!usernames || usernames.length === 0) return []
    
    // 去重
    const uniqueUsernames = [...new Set(usernames)].filter(u => u)
    if (uniqueUsernames.length === 0) return []
    
    // 🔥 懒加载：只清除本次查询涉及的过期缓存，不批量清除所有过期缓存
    clearExpiredCacheForUsernames(uniqueUsernames)
    
    // 分离已缓存（且未过期）、过期缓存、正在加载和未缓存的用户名
    const cachedUsers: UserInfo[] = []
    const expiredUsers: { username: string; user: UserInfo }[] = []
    const loadingUsernamesList: string[] = []
    const uncachedUsernames: string[] = []
    
    uniqueUsernames.forEach(username => {
      if (forceRefresh) {
        // 强制刷新，清除缓存
        userInfoCache.value.delete(username)
        uncachedUsernames.push(username)
      } else {
        const cacheItem = userInfoCache.value.get(username)
        if (cacheItem) {
          if (!isCacheExpired(cacheItem)) {
            // 缓存有效，直接使用
            cachedUsers.push(cacheItem.user)
          } else {
            // 🔥 缓存过期，先收集起来，作为降级值
            expiredUsers.push({ username, user: cacheItem.user })
            // 也需要刷新
            if (!loadingUsernames.value.has(username)) {
              uncachedUsernames.push(username)
            }
          }
        } else if (loadingUsernames.value.has(username)) {
          // 正在加载中，等待加载完成
          loadingUsernamesList.push(username)
        } else {
          // 缓存不存在，需要查询
          uncachedUsernames.push(username)
        }
      }
    })
    
    // 🔥 如果有正在加载的用户，等待它们加载完成（但不超过超时时间）
    if (loadingUsernamesList.length > 0) {
      console.log(`[UserInfoStore] 等待正在加载的用户:`, loadingUsernamesList)
      const startTime = Date.now()
      await new Promise<void>((resolve) => {
        const checkInterval = setInterval(() => {
          const allLoaded = loadingUsernamesList.every(username => {
            const cacheItem = userInfoCache.value.get(username)
            return cacheItem && !isCacheExpired(cacheItem)
          })
          if (allLoaded) {
            clearInterval(checkInterval)
            resolve()
          } else if (Date.now() - startTime > API_TIMEOUT) {
            // 🔥 超时后，不再等待，使用过期缓存（降级策略）
            clearInterval(checkInterval)
            resolve()
          }
        }, 50)
        
        // 超时保护（5秒）
        setTimeout(() => {
          clearInterval(checkInterval)
          resolve()
        }, 5000)
      })
      
      // 加载完成后，从缓存中获取
      loadingUsernamesList.forEach(username => {
        const cacheItem = userInfoCache.value.get(username)
        if (cacheItem) {
          if (!isCacheExpired(cacheItem)) {
            cachedUsers.push(cacheItem.user)
          } else {
            // 如果还是过期，使用过期值（降级策略）
            expiredUsers.push({ username, user: cacheItem.user })
          }
        }
      })
    }
    
    // 🔥 如果所有用户都已缓存或正在加载，直接返回（包括过期缓存）
    if (uncachedUsernames.length === 0) {
      // 合并有效缓存和过期缓存
      const result: UserInfo[] = []
      uniqueUsernames.forEach(username => {
        const cached = cachedUsers.find(u => u.username === username)
        if (cached) {
          result.push(cached)
        } else {
          const expired = expiredUsers.find(e => e.username === username)
          if (expired) {
            result.push(expired.user)
          }
        }
      })
      return result
    }
    
    // 标记正在查询
    uncachedUsernames.forEach(username => loadingUsernames.value.add(username))
    
    // 🔥 降级策略：构建降级结果（过期缓存 + 有效缓存），用于超时或失败时返回
    const buildFallbackResult = (): UserInfo[] => {
      const result: UserInfo[] = []
      uniqueUsernames.forEach(username => {
        // 优先使用有效缓存
        const cached = cachedUsers.find(u => u.username === username)
        if (cached) {
          result.push(cached)
        } else {
          // 其次使用过期缓存（降级策略）
          const expired = expiredUsers.find(e => e.username === username)
          if (expired) {
            result.push(expired.user)
          }
          // 如果都没有，返回 null（会在最终结果中过滤）
        }
      })
      return result
    }
    
    const fallbackResult = buildFallbackResult()
    
    // 🔥 使用 Promise.race 实现超时降级
    const fetchPromise = (async () => {
      try {
        // 批量查询未缓存的用户
        console.log(`[UserInfoStore] 批量查询用户信息:`, uncachedUsernames)
        const response = await getUsersByUsernames(uncachedUsernames)
        const loadedUsers = response.users || []
        console.log(`[UserInfoStore] 批量查询完成，获取到 ${loadedUsers.length} 个用户`)
        
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
        uncachedUsernames.forEach(username => loadingUsernames.value.delete(username))
        
        // 返回所有用户（已缓存 + 新加载的），按照 uniqueUsernames 的顺序
        const allUsers: UserInfo[] = []
        uniqueUsernames.forEach(username => {
          const cacheItem = userInfoCache.value.get(username)
          if (cacheItem) {
            allUsers.push(cacheItem.user)
          } else {
            // 如果新加载的也没有，尝试使用过期缓存（降级策略）
            const expired = expiredUsers.find(e => e.username === username)
            if (expired) {
              allUsers.push(expired.user)
            }
          }
        })
        
        return allUsers
      } catch (error) {
        // 移除查询标记
        uncachedUsernames.forEach(username => loadingUsernames.value.delete(username))
        console.error('[UserInfoStore] 批量查询用户信息失败:', error)
        // 🔥 降级策略：查询失败时，返回过期缓存
        throw error
      }
    })()
    
    // 🔥 超时降级：如果接口超过300ms未返回，先返回过期缓存，后台继续刷新
    const timeoutPromise = new Promise<UserInfo[]>((resolve) => {
      setTimeout(() => {
        console.log(`[UserInfoStore] 接口超时（${API_TIMEOUT}ms），使用过期缓存（降级策略）`)
        // 返回过期缓存，不阻塞用户
        resolve(fallbackResult)
        // 后台继续等待接口返回（不阻塞）
        fetchPromise.then(users => {
          console.log(`[UserInfoStore] 后台刷新完成，更新缓存`)
          // 缓存已更新，下次获取时会使用新值
        }).catch(() => {
          // 刷新失败不影响，继续使用过期缓存
        })
      }, API_TIMEOUT)
    })
    
    try {
      // 如果接口在超时时间内返回，使用新数据
      return await Promise.race([fetchPromise, timeoutPromise])
    } catch (error) {
      // 🔥 降级策略：如果接口失败，返回过期缓存
      console.warn('[UserInfoStore] 接口失败，使用过期缓存（降级策略）')
      return fallbackResult
    }
  }
  
  /**
   * 刷新指定用户的缓存
   * 
   * @param usernames 用户名列表，如果为空则刷新所有缓存
   */
  async function refreshCache(usernames?: string[]): Promise<void> {
    if (usernames && usernames.length > 0) {
      // 刷新指定用户
      await batchGetUserInfo(usernames, true)
    } else {
      // 刷新所有缓存
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
    localStorage.removeItem('user-info-cache')
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
    const now = Date.now()
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
    getCacheStats
  }
}, {
  // 🔥 启用持久化，将缓存保存到 localStorage
  persist: {
    key: 'user-info-cache',
    storage: localStorage,
    // 自定义序列化和反序列化，因为 Map 不能直接序列化
    serializer: {
      deserialize: (value: string) => {
        try {
          const data = JSON.parse(value)
          if (data && data.userInfoCache) {
            // 将存储的对象转换为 Map
            const map = new Map<string, CacheItem>()
            Object.entries(data.userInfoCache).forEach(([username, cacheItem]: [string, any]) => {
              if (cacheItem && cacheItem.timestamp) {
                map.set(username, cacheItem as CacheItem)
              }
            })
            return { userInfoCache: map }
          }
        } catch (error) {
          console.error('[UserInfoStore] 反序列化失败:', error)
        }
        return { userInfoCache: new Map() }
      },
      serialize: (value: any) => {
        try {
          // 将 Map 转换为对象
          const cacheObj: Record<string, CacheItem> = {}
          if (value.userInfoCache && value.userInfoCache instanceof Map) {
            value.userInfoCache.forEach((cacheItem: CacheItem, username: string) => {
              cacheObj[username] = cacheItem
            })
          }
          return JSON.stringify({ userInfoCache: cacheObj })
        } catch (error) {
          console.error('[UserInfoStore] 序列化失败:', error)
          return JSON.stringify({ userInfoCache: {} })
        }
      }
    },
    // 只持久化 userInfoCache，不持久化 loadingUsernames（运行时状态）
    paths: ['userInfoCache']
  }
})
