/**
 * ICacheManager - 缓存管理接口
 * 
 * 职责：定义缓存管理的标准接口，实现依赖倒置原则
 * 
 * 使用场景：
 * - 函数详情缓存
 * - 服务树缓存
 * - 临时数据缓存
 */

/**
 * 缓存管理接口
 */
export interface ICacheManager {
  /**
   * 获取缓存
   * @param key 缓存键
   * @returns T | null
   */
  get<T>(key: string): T | null

  /**
   * 设置缓存
   * @param key 缓存键
   * @param value 缓存值
   */
  set<T>(key: string, value: T): void

  /**
   * 删除缓存
   * @param key 缓存键
   */
  delete(key: string): void

  /**
   * 清空所有缓存
   */
  clear(): void

  /**
   * 检查缓存是否存在
   * @param key 缓存键
   * @returns boolean
   */
  has(key: string): boolean
}

