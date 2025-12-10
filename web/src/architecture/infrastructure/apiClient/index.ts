/**
 * ApiClient 导出
 */

import { ApiClientImpl } from './ApiClientImpl'
import type { IApiClient } from '../../domain/interfaces/IApiClient'

// 导出接口
export type { IApiClient } from '../../domain/interfaces/IApiClient'

// 导出实现
export { ApiClientImpl }

// 导出单例实例（可选，也可以在使用时创建新实例）
export const apiClient: IApiClient = new ApiClientImpl()

