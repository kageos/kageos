/**
 * IServiceTreeLoader - 服务目录树加载器接口
 * 
 * 职责：定义服务目录树加载的标准接口，实现依赖倒置原则
 * 
 * 使用场景：
 * - 加载应用的服务目录树
 * - 支持缓存和去重
 */

import type { App, ServiceTree } from '@/types'

/**
 * 服务目录树加载器接口
 */
export interface IServiceTreeLoader {
  /**
   * 加载服务目录树
   * @param app 应用信息
   * @returns Promise<ServiceTree[]>
   */
  load(app: App): Promise<ServiceTree[]>
}

