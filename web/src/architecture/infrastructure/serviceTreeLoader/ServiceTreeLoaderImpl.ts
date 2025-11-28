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
  constructor(private apiClient: IApiClient) {}

  /**
   * 加载服务目录树
   */
  async load(app: App): Promise<ServiceTree[]> {
    if (!app || !app.user || !app.code) {
      return []
    }

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
    }
  }
}

