/**
 * FunctionLoaderImpl - 函数加载器实现
 * 
 * 职责：实现 IFunctionLoader 接口，提供函数加载功能（带防抖和去重）
 * 
 * 特点：
 * - 支持根据 ID 或路径加载函数详情
 * - 内置缓存机制
 * - 防抖和去重，避免重复调用
 * - 解决当前架构的重复调用问题
 */

import type { IFunctionLoader, FunctionDetail } from '../../domain/interfaces/IFunctionLoader'
import type { IApiClient } from '../../domain/interfaces/IApiClient'
import type { ICacheManager } from '../../domain/interfaces/ICacheManager'

/**
 * 函数加载器实现
 */
export class FunctionLoaderImpl implements IFunctionLoader {
  // 正在进行的请求（用于去重）
  private pendingRequests = new Map<string, Promise<FunctionDetail>>()

  // 防抖定时器
  private debounceTimers = new Map<string, NodeJS.Timeout>()

  constructor(
    private apiClient: IApiClient,
    private cacheManager: ICacheManager,
    private debounceDelay: number = 300 // 防抖延迟（毫秒）
  ) {}

  /**
   * 根据 ID 加载函数详情
   */
  async loadById(id: number): Promise<FunctionDetail> {
    const cacheKey = `function:id:${id}`
    
    // 先检查缓存
    const cached = this.cacheManager.get<FunctionDetail>(cacheKey)
    if (cached) {
      return cached
    }

    // 检查是否有正在进行的请求（去重）
    const pendingKey = `id:${id}`
    if (this.pendingRequests.has(pendingKey)) {
      return this.pendingRequests.get(pendingKey)!
    }

    // 创建新请求
    const request = this.loadFunctionById(id, cacheKey)
    this.pendingRequests.set(pendingKey, request)

    try {
      const result = await request
      return result
    } finally {
      // 请求完成后移除
      this.pendingRequests.delete(pendingKey)
    }
  }

  /**
   * 根据路径加载函数详情（带防抖）
   */
  async loadByPath(path: string): Promise<FunctionDetail> {
    const cacheKey = `function:path:${path}`
    
    // 先检查缓存
    const cached = this.cacheManager.get<FunctionDetail>(cacheKey)
    if (cached) {
      return cached
    }

    // 防抖处理
    return new Promise<FunctionDetail>((resolve, reject) => {
      const debounceKey = `path:${path}`
      
      // 清除之前的定时器
      const existingTimer = this.debounceTimers.get(debounceKey)
      if (existingTimer) {
        clearTimeout(existingTimer)
      }

      // 检查是否有正在进行的请求（去重）
      if (this.pendingRequests.has(debounceKey)) {
        this.pendingRequests.get(debounceKey)!.then(resolve).catch(reject)
        return
      }

      // 设置新的防抖定时器
      const timer = setTimeout(async () => {
        this.debounceTimers.delete(debounceKey)
        
        try {
          const request = this.loadFunctionByPath(path, cacheKey)
          this.pendingRequests.set(debounceKey, request)
          
          const result = await request
          resolve(result)
        } catch (error) {
          reject(error)
        } finally {
          this.pendingRequests.delete(debounceKey)
        }
      }, this.debounceDelay)

      this.debounceTimers.set(debounceKey, timer)
    })
  }

  /**
   * 获取缓存的函数详情
   */
  getCached(id?: number, path?: string): FunctionDetail | null {
    if (id !== undefined) {
      const cacheKey = `function:id:${id}`
      return this.cacheManager.get<FunctionDetail>(cacheKey)
    }
    
    if (path !== undefined) {
      const cacheKey = `function:path:${path}`
      return this.cacheManager.get<FunctionDetail>(cacheKey)
    }

    return null
  }

  /**
   * 清空缓存
   */
  clearCache(): void {
    // 清空所有 function: 开头的缓存
    const keys = this.cacheManager.getKeys?.() || []
    keys.forEach(key => {
      if (key.startsWith('function:')) {
        this.cacheManager.delete(key)
      }
    })
  }

  /**
   * 根据 ID 加载函数详情（内部方法）
   */
  private async loadFunctionById(id: number, cacheKey: string): Promise<FunctionDetail> {
    const response = await this.apiClient.get<FunctionDetail>('/workspace/api/v1/function/get', {
      function_id: id
    })
    
    // 缓存结果
    this.cacheManager.set(cacheKey, response)
    
    return response
  }

  /**
   * 根据路径加载函数详情（内部方法）
   */
  private async loadFunctionByPath(path: string, cacheKey: string): Promise<FunctionDetail> {
    const response = await this.apiClient.get<FunctionDetail>('/workspace/api/v1/function/by-path', {
      path: path
    })
    
    // 缓存结果
    this.cacheManager.set(cacheKey, response)
    
    return response
  }
}

