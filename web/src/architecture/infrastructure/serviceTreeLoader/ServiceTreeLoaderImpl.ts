/**
 * ServiceTreeLoaderImpl - 服务目录树加载器实现
 * 
 * 职责：加载服务目录树数据
 * 
 * 特点：
 * - 实现服务目录树的加载逻辑
 * - 可以缓存服务树数据
 */

import type { IApiClient } from '../../domain/interfaces/IApiClient'
import type { App, ServiceTree } from '@/types'

/**
 * 服务目录树加载器接口
 */
export interface IServiceTreeLoader {
  load(app: App): Promise<ServiceTree[]>
}

/**
 * 服务目录树加载器实现
 */
export class ServiceTreeLoaderImpl implements IServiceTreeLoader {
  private loadingPromises = new Map<string, Promise<ServiceTree[]>>()
  
  constructor(private apiClient: IApiClient) {}

  /**
   * 加载服务目录树（带防抖和去重）
   */
  async load(app: App): Promise<ServiceTree[]> {
    if (!app || !app.user || !app.code) {
      return []
    }

    // 🔥 生成缓存键，用于去重
    const cacheKey = `${app.user}/${app.code}`
    
    // 🔥 如果正在加载，返回同一个 Promise，避免重复请求
    const existingPromise = this.loadingPromises.get(cacheKey)
    if (existingPromise) {
      console.log('[ServiceTreeLoader] 检测到重复请求，返回已存在的 Promise:', cacheKey)
      return existingPromise
    }

    // 创建新的加载 Promise
    const loadPromise = (async () => {
      try {
        // 注意：API 路径是 /api/v1/service_tree（下划线），不是 /api/v1/service-tree/list
        const tree = await this.apiClient.get<ServiceTree[]>('/api/v1/service_tree', {
          user: app.user,
          app: app.code
        })
        return tree || []
      } catch (error) {
        console.error('[ServiceTreeLoader] 加载服务目录树失败', error)
        return []
      } finally {
        // 加载完成后，从 Map 中移除
        this.loadingPromises.delete(cacheKey)
      }
    })()

    // 将 Promise 存入 Map，用于去重
    this.loadingPromises.set(cacheKey, loadPromise)
    
    return loadPromise
  }
}

