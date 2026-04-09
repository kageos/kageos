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
import type { IServiceTreeLoader, ServiceTreeLoadResult } from '../../domain/interfaces/IServiceTreeLoader'
import type { App, ServiceTree } from '@/types'

/**
 * 服务目录树加载器实现
 */
export class ServiceTreeLoaderImpl implements IServiceTreeLoader {
  private loadingPromises = new Map<string, Promise<ServiceTreeLoadResult>>()
  
  constructor(private apiClient: IApiClient) {}

  /**
   * 加载服务目录树（带防抖和去重）
   */
  async load(app: App): Promise<ServiceTreeLoadResult> {
    if (!app || !app.user || !app.code) {
      return { tree: [] }
    }

    // 生成缓存键，用于去重
    const cacheKey = `${app.user}/${app.code}`
    
    // 如果正在加载，返回同一个 Promise，避免重复请求
    const existingPromise = this.loadingPromises.get(cacheKey)
    if (existingPromise) {
      return existingPromise
    }

    // 创建新的加载 Promise
    const loadPromise = (async (): Promise<ServiceTreeLoadResult> => {
      try {
        // ⭐ 使用合并接口获取应用详情和服务目录树（减少请求次数）
        // 接口路径：/workspace/api/v1/app/tree?resource_path=/user/app
        const response = await this.apiClient.get<any>('/workspace/api/v1/app/tree', {
          resource_path: `/${app.user}/${app.code}`
        })
        
        // 处理响应数据：合并接口返回 { app: App, service_tree: ServiceTree[], expanded_keys?: number[] }
        let tree: ServiceTree[] = []
        let appInfo: App | null = null
        let expandedKeys: number[] | undefined = undefined
        
        // 处理合并接口的响应格式：{ app: App, service_tree: ServiceTree[], expanded_keys?: number[] }
        if (response && typeof response === 'object' && 'service_tree' in response && Array.isArray(response.service_tree)) {
          tree = response.service_tree
          // 提取应用信息（包括正确的 id）
          if ('app' in response && response.app) {
            appInfo = response.app as App
          }
          // 提取 expanded_keys（如果后端返回了）
          if ('expanded_keys' in response && Array.isArray(response.expanded_keys)) {
            expandedKeys = response.expanded_keys
          }
        }
        
        return { tree, expandedKeys, app: appInfo || undefined }
      } catch (error) {
        Logger.error('ServiceTreeLoader', '加载服务目录树失败', error)
        return { tree: [] }
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
