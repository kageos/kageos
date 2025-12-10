/**
 * ServiceTreeLoaderImpl - 服务目录树加载器实现
 * 
 * 职责：加载服务目录树数据
 * 
 * 特点：
 * - 实现服务目录树的加载逻辑
 * - 可以缓存服务树数据
 */

import { Logger } from '@/core/utils/logger'
import type { IApiClient } from '../../domain/interfaces/IApiClient'
import type { IServiceTreeLoader } from '../../domain/interfaces/IServiceTreeLoader'
import type { App, ServiceTree } from '@/types'

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

    // 生成缓存键，用于去重
    const cacheKey = `${app.user}/${app.code}`
    
    // 如果正在加载，返回同一个 Promise，避免重复请求
    const existingPromise = this.loadingPromises.get(cacheKey)
    if (existingPromise) {
      Logger.debug('ServiceTreeLoader', '检测到重复请求，返回已存在的 Promise', cacheKey)
      return existingPromise
    }

    // 创建新的加载 Promise
    const loadPromise = (async () => {
      try {
        Logger.debug('ServiceTreeLoader', '开始加载服务目录树', app.user, app.code)
        // 注意：API 路径是 /api/v1/service_tree（下划线），不是 /api/v1/service-tree/list
        const response = await this.apiClient.get<any>('/api/v1/service_tree', {
          user: app.user,
          app: app.code
        })
        
        Logger.debug('ServiceTreeLoader', 'API 响应', response)
        
        // 处理响应数据：可能是数组，也可能是分页对象
        let tree: ServiceTree[] = []
        if (Array.isArray(response)) {
          tree = response
        } else if (response && typeof response === 'object' && 'items' in response) {
          tree = response.items || []
        } else if (response && typeof response === 'object' && 'data' in response) {
          tree = Array.isArray(response.data) ? response.data : []
        }
        
        Logger.debug('ServiceTreeLoader', '解析后的服务目录树，节点数', tree.length)
        return tree
      } catch (error) {
        Logger.error('ServiceTreeLoader', '加载服务目录树失败', error)
        return []
      } finally {
        // 加载完成后，从 Map 中移除
        this.loadingPromises.delete(cacheKey)
        Logger.debug('ServiceTreeLoader', '清理加载 Promise', cacheKey)
      }
    })()

    // 将 Promise 存入 Map，用于去重
    this.loadingPromises.set(cacheKey, loadPromise)
    
    return loadPromise
  }
}

