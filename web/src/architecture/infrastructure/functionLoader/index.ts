/**
 * FunctionLoader 导出
 */

import { FunctionLoaderImpl } from './FunctionLoaderImpl'
import type { IFunctionLoader } from '../../domain/interfaces/IFunctionLoader'
import { apiClient } from '../apiClient'
import { cacheManager } from '../cacheManager'

// 导出接口
export type { IFunctionLoader, FunctionDetail } from '../../domain/interfaces/IFunctionLoader'

// 导出实现
export { FunctionLoaderImpl }

// 导出单例实例（使用默认的 apiClient 和 cacheManager）
export const functionLoader: IFunctionLoader = new FunctionLoaderImpl(
  apiClient,
  cacheManager,
  300 // 防抖延迟 300ms
)

