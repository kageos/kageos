/**
 * StateManagerImpl - 状态管理实现
 * 
 * 职责：实现 IStateManager 接口，提供状态管理功能
 * 
 * 特点：
 * - 基于 Pinia Store 实现
 * - 支持状态订阅
 * - 可以轻松替换为其他实现（如 Vuex、Zustand 等）
 */

import { watch, type WatchStopHandle } from 'vue'
import type { IStateManager } from '../../domain/interfaces/IStateManager'

/**
 * 状态管理实现（基于响应式对象）
 */
export class StateManagerImpl<T> implements IStateManager<T> {
  private state: T
  private subscribers: Set<(state: T) => void> = new Set()
  private watchStopHandle: WatchStopHandle | null = null

  constructor(initialState: T) {
    this.state = initialState
  }

  /**
   * 获取当前状态
   */
  getState(): T {
    return this.state
  }

  /**
   * 设置状态
   */
  setState(newState: T): void {
    this.state = newState
    // 通知所有订阅者
    this.notifySubscribers()
  }

  /**
   * 订阅状态变化
   * @returns 取消订阅的函数
   */
  subscribe(callback: (state: T) => void): () => void {
    this.subscribers.add(callback)

    // 返回取消订阅的函数
    return () => {
      this.subscribers.delete(callback)
    }
  }

  /**
   * 清空状态
   */
  clear(): void {
    // 清空所有订阅者
    this.subscribers.clear()
    
    // 停止 watch（如果有）
    if (this.watchStopHandle) {
      this.watchStopHandle()
      this.watchStopHandle = null
    }
  }

  /**
   * 通知所有订阅者
   */
  private notifySubscribers(): void {
    this.subscribers.forEach(callback => {
      try {
        callback(this.state)
      } catch (error) {
        console.error('[StateManager] 订阅者回调执行失败', error)
      }
    })
  }

  /**
   * 设置 watch（用于响应式状态，如 Pinia Store）
   * @param getter 获取响应式状态的函数
   */
  setWatch(getter: () => T): void {
    // 停止之前的 watch
    if (this.watchStopHandle) {
      this.watchStopHandle()
    }

    // 创建新的 watch
    this.watchStopHandle = watch(
      getter,
      (newState) => {
        this.state = newState
        this.notifySubscribers()
      },
      { deep: true, immediate: true }
    )
  }
}

