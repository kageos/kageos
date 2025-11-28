/**
 * ServiceTreeLoader 导出
 */

import { ServiceTreeLoaderImpl } from './ServiceTreeLoaderImpl'
import type { IServiceTreeLoader } from './ServiceTreeLoaderImpl'
import { apiClient } from '../apiClient'

// 导出接口
export type { IServiceTreeLoader } from './ServiceTreeLoaderImpl'

// 导出实现
export { ServiceTreeLoaderImpl }

// 导出单例实例
export const serviceTreeLoader: IServiceTreeLoader = new ServiceTreeLoaderImpl(apiClient)

