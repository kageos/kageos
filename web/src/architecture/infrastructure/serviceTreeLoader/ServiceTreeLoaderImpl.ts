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
        // 使用合并接口获取应用详情和服务目录树（减少请求次数）
        // 接口路径：/workspace/api/v1/app/{code}/tree
        const response = await this.apiClient.get<any>(`/workspace/api/v1/app/${app.code}/tree`, {})
        
        Logger.debug('ServiceTreeLoader', 'API 响应', response)
        
        // 处理响应数据：合并接口返回 { app: App, service_tree: ServiceTree[] }
        let tree: ServiceTree[] = []
        if (response && typeof response === 'object') {
          // 如果是合并接口的响应格式
          if ('service_tree' in response && Array.isArray(response.service_tree)) {
            tree = response.service_tree
          }
          // 兼容旧的单独接口格式（数组或分页对象）
          else if (Array.isArray(response)) {
            tree = response
          } else if ('items' in response && Array.isArray(response.items)) {
            tree = response.items || []
          } else if ('data' in response && Array.isArray(response.data)) {
            tree = response.data || []
          }
        }
        
        Logger.debug('ServiceTreeLoader', '解析后的服务目录树，节点数', tree.length)
        return tree
      } catch (error) {
        Logger.error('ServiceTreeLoader', '加载服务目录树失败', error)
        // 如果合并接口失败，回退到旧的单独接口
        try {
          Logger.debug('ServiceTreeLoader', '回退到旧的单独接口')
          const fallbackResponse = await this.apiClient.get<any>('/workspace/api/v1/service_tree', {
            user: app.user,
            app: app.code
          })
          
          let tree: ServiceTree[] = []
          if (Array.isArray(fallbackResponse)) {
            tree = fallbackResponse
          } else if (fallbackResponse && typeof fallbackResponse === 'object' && 'items' in fallbackResponse) {
            tree = fallbackResponse.items || []
          } else if (fallbackResponse && typeof fallbackResponse === 'object' && 'data' in fallbackResponse) {
            tree = Array.isArray(fallbackResponse.data) ? fallbackResponse.data : []
          }
          
          Logger.debug('ServiceTreeLoader', '回退接口解析后的服务目录树，节点数', tree.length)
          return tree
        } catch (fallbackError) {
          Logger.error('ServiceTreeLoader', '回退接口也失败', fallbackError)
          return []
        }
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

