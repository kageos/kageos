/**
 * IStateManager - 状态管理接口
 * 
 * 职责：定义状态管理的标准接口，实现依赖倒置原则
 * 
 * 使用场景：
 * - 表单数据管理
 * - 工作空间状态管理
 * - 表格数据管理
 */

/**
 * 状态管理接口
 * @template T 状态类型
 */
export interface IStateManager<T> {
  /**
   * 获取当前状态
   * @returns 当前状态
   */
  getState(): T

  /**
   * 设置状态
   * @param state 新状态
   */
  setState(state: T): void

  /**
   * 订阅状态变化
   * @param callback 状态变化回调函数
   * @returns 取消订阅的函数
   */
  subscribe(callback: (state: T) => void): () => void

  /**
   * 清空状态
   */
  clear(): void
}

