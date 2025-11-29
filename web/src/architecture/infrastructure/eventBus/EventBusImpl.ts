/**
 * EventBusImpl - 事件总线实现
 * 
 * 职责：实现 IEventBus 接口，提供内存事件总线功能
 * 
 * 特点：
 * - 基于内存实现，简单高效
 * - 支持事件监听、取消监听、一次性监听
 * - 可以轻松替换为其他实现（如 WebSocket 事件总线）
 */

import { Logger } from '@/core/utils/logger'
import type { IEventBus } from '../../domain/interfaces/IEventBus'

/**
 * 事件总线实现（内存版本）
 */
export class EventBusImpl implements IEventBus {
  private handlers = new Map<string, Set<Function>>()

  /**
   * 触发事件
   */
  emit(event: string, payload?: any): void {
    const handlers = this.handlers.get(event)
    if (handlers) {
      handlers.forEach(handler => {
        try {
          handler(payload)
        } catch (error) {
          Logger.error('EventBus', `事件处理函数执行失败: ${event}`, error)
        }
      })
    }
  }

  /**
   * 监听事件
   * @returns 取消监听的函数
   */
  on(event: string, handler: (payload?: any) => void): () => void {
    if (!this.handlers.has(event)) {
      this.handlers.set(event, new Set())
    }
    this.handlers.get(event)!.add(handler)

    // 返回取消监听的函数
    return () => {
      this.off(event, handler)
    }
  }

  /**
   * 取消监听事件
   */
  off(event: string, handler: (payload?: any) => void): void {
    const handlers = this.handlers.get(event)
    if (handlers) {
      handlers.delete(handler)
      // 如果没有监听器了，删除该事件的 Map 条目
      if (handlers.size === 0) {
        this.handlers.delete(event)
      }
    }
  }

  /**
   * 监听事件（仅触发一次）
   */
  once(event: string, handler: (payload?: any) => void): void {
    const onceHandler = (payload?: any) => {
      handler(payload)
      this.off(event, onceHandler)
    }
    this.on(event, onceHandler)
  }

  /**
   * 清空所有监听器（用于测试或清理）
   */
  clear(): void {
    this.handlers.clear()
  }

  /**
   * 获取所有已注册的事件名称（用于调试）
   */
  getRegisteredEvents(): string[] {
    return Array.from(this.handlers.keys())
  }

  /**
   * 获取指定事件的监听器数量（用于调试）
   */
  getListenerCount(event: string): number {
    return this.handlers.get(event)?.size || 0
  }
}

