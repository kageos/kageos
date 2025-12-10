/**
 * CacheManager 导出
 */

import { CacheManagerImpl } from './CacheManagerImpl'
import type { ICacheManager } from '../../domain/interfaces/ICacheManager'

// 导出接口
export type { ICacheManager } from '../../domain/interfaces/ICacheManager'

// 导出实现
export { CacheManagerImpl }

// 导出单例实例（可选，也可以在使用时创建新实例）
export const cacheManager: ICacheManager = new CacheManagerImpl()

