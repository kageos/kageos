/**
 * FunctionLoaderImpl - 函数加载器实现
 * 
 * 职责：实现 IFunctionLoader 接口，提供函数加载功能（带防抖和去重）
 * 
 * 特点：
 * - 只支持根据 full-code-path 加载函数详情
 * - 不做持久化缓存，确保字段 schema 始终最新
 * - 仅保留请求级防抖和并发去重，避免同一时刻重复调用
 * - 解决当前架构的重复调用问题
 */

import type { IFunctionLoader } from '../../domain/interfaces/IFunctionLoader'
import type { IApiClient } from '../../domain/interfaces/IApiClient'
import type { ICacheManager } from '../../domain/interfaces/ICacheManager'
import type { FunctionDetail } from '../../domain/types'

/**
 * 函数加载器实现
 */
export class FunctionLoaderImpl implements IFunctionLoader {
  // 正在等待或执行中的请求（用于去重）
  private pendingRequests = new Map<string, Promise<FunctionDetail>>()

  private pendingRequestResolvers = new Map<string, {
    resolve: (value: FunctionDetail) => void
    reject: (reason?: any) => void
  }>()

  // 防抖定时器
  private debounceTimers = new Map<string, ReturnType<typeof setTimeout>>()

  constructor(
    private apiClient: IApiClient,
    private cacheManager: ICacheManager,
    private debounceDelay: number = 300 // 防抖延迟（毫秒）
  ) {}

  /**
   * 根据路径加载函数详情（带防抖）
   */
  async loadByPath(path: string, funcType: string = 'table'): Promise<FunctionDetail> {
    const debounceKey = `path:${path}`
    const existingRequest = this.pendingRequests.get(debounceKey)
    if (existingRequest) {
      const existingTimer = this.debounceTimers.get(debounceKey)
      if (existingTimer) {
        clearTimeout(existingTimer)
        this.debounceTimers.set(
          debounceKey,
          this.createDebounceTimer(debounceKey, path, funcType)
        )
      }
      return existingRequest
    }

    const requestPromise = new Promise<FunctionDetail>((resolve, reject) => {
      this.pendingRequestResolvers.set(debounceKey, { resolve, reject })
    })

    this.pendingRequests.set(debounceKey, requestPromise)
    this.debounceTimers.set(
      debounceKey,
      this.createDebounceTimer(debounceKey, path, funcType)
    )

    return requestPromise
  }

  /**
   * 获取缓存的函数详情
   */
  getCached(path: string): FunctionDetail | null {
    return null
  }

  /**
   * 清空缓存
   */
  clearCache(): void {
    // 已移除持久化缓存，保留接口仅用于兼容调用方。
  }

  /**
   * 根据路径加载函数详情（内部方法）
   * ⭐ 使用新的路由：/function/info/:func-type/*full-code-path
   * @param path 函数完整路径
   * @param funcType 函数类型：table、form、chart（可选，默认为 table）
   */
  private async loadFunctionByPath(path: string, funcType: string = 'table'): Promise<FunctionDetail> {
    // 确保路径以 / 开头
    const fullCodePath = path.startsWith('/') ? path : `/${path}`
    // ⭐ 函数类型作为路径参数，这样后端无需查询数据库即可构造权限点
    return await this.apiClient.get<FunctionDetail>(`/workspace/api/v1/function/info/${funcType}${fullCodePath}`)
  }

  private createDebounceTimer(
    debounceKey: string,
    path: string,
    funcType: string
  ): ReturnType<typeof setTimeout> {
    return setTimeout(async () => {
      this.debounceTimers.delete(debounceKey)

      try {
        const result = await this.loadFunctionByPath(path, funcType)
        this.pendingRequestResolvers.get(debounceKey)?.resolve(result)
      } catch (error) {
        this.pendingRequestResolvers.get(debounceKey)?.reject(error)
      } finally {
        this.pendingRequestResolvers.delete(debounceKey)
        this.pendingRequests.delete(debounceKey)
      }
    }, this.debounceDelay)
  }
}
