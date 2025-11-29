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

import { watch, type WatchStopHandle, shallowRef, triggerRef, unref } from 'vue'
import { Logger } from '@/core/utils/logger'
import type { IStateManager } from '../../domain/interfaces/IStateManager'

/**
 * 状态管理实现（基于 Vue shallowRef）
 */
export class StateManagerImpl<T> implements IStateManager<T> {
  // 使用 shallowRef 包装状态，使其具有响应式，但避免深层转换的性能开销
  // 我们的状态更新通常是整个替换或顶层属性替换
  private stateRef: any 
  
  private subscribers: Set<(state: T) => void> = new Set()
  private watchStopHandle: WatchStopHandle | null = null

  constructor(initialState: T) {
    this.stateRef = shallowRef(initialState)
  }

  /**
   * 获取当前状态
   * 注意：在 Vue 组件的 setup/render 函数中调用此方法会建立响应式依赖
   */
  getState(): T {
    return this.stateRef.value
  }

  /**
   * 设置状态
   */
  setState(newState: T): void {
    // 更新 ref 的值，触发 Vue 的响应式更新
    this.stateRef.value = newState
    
    // 通知手动订阅者（非 Vue 组件）
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
        Logger.error('StateManager', '订阅者回调执行失败', error)
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

