/**
 * CacheManagerImpl - 缓存管理实现
 * 
 * 职责：实现 ICacheManager 接口，提供内存缓存功能
 * 
 * 特点：
 * - 基于内存实现，简单高效
 * - 支持设置、获取、删除、清空操作
 * - 可以轻松替换为其他实现（如 Redis 缓存）
 */

import type { ICacheManager } from '../../domain/interfaces/ICacheManager'

/**
 * 缓存管理实现（内存版本）
 */
export class CacheManagerImpl implements ICacheManager {
  private cache = new Map<string, any>()

  /**
   * 获取缓存
   */
  get<T>(key: string): T | null {
    const value = this.cache.get(key)
    return value !== undefined ? (value as T) : null
  }

  /**
   * 设置缓存
   */
  set<T>(key: string, value: T): void {
    this.cache.set(key, value)
  }

  /**
   * 删除缓存
   */
  delete(key: string): void {
    this.cache.delete(key)
  }

  /**
   * 清空所有缓存
   */
  clear(): void {
    this.cache.clear()
  }

  /**
   * 检查缓存是否存在
   */
  has(key: string): boolean {
    return this.cache.has(key)
  }

  /**
   * 获取所有缓存键（用于调试）
   */
  getKeys(): string[] {
    return Array.from(this.cache.keys())
  }

  /**
   * 获取缓存数量（用于调试）
   */
  getSize(): number {
    return this.cache.size
  }
}

