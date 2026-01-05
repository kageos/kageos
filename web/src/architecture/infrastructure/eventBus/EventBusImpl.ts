/**
 * EventBusImpl - 事件总线实现
 * 
 * ============================================
 * 📋 需求说明
 * ============================================
 * 
 * 1. **事件通信**：
 *    - 组件间通信（跨层级、跨组件）
 *    - 解耦组件依赖
 *    - 支持事件监听、触发、取消监听
 * 
 * 2. **事件类型**：
 *    - WorkspaceEvent：工作空间相关事件（节点点击、应用切换等）
 *    - FormEvent：表单相关事件（初始化、提交、验证等）
 *    - TableEvent：表格相关事件（数据加载、行操作等）
 *    - RouteEvent：路由相关事件（路由更新、路径变化等）
 * 
 * 3. **可扩展性**：
 *    - 可以轻松替换为其他实现（如 WebSocket 事件总线）
 *    - 新功能可以监听现有事件，不需要修改现有代码
 * 
 * ============================================
 * 🎯 设计思路
 * ============================================
 * 
 * 1. **依赖倒置原则**：
 *    - 实现 `IEventBus` 接口
 *    - 所有组件依赖接口，不依赖具体实现
 *    - 可以轻松替换实现
 * 
 * 2. **内存实现**：
 *    - 基于 Map 存储事件处理器
 *    - 简单高效，适合单页应用
 *    - 可以替换为 WebSocket 实现（分布式场景）
 * 
 * 3. **事件管理**：
 *    - 支持多个处理器监听同一事件
 *    - 支持取消监听
 *    - 支持一次性监听（`once`）
 * 
 * ============================================
 * 📝 关键功能
 * ============================================
 * 
 * 1. **emit**：
 *    - 触发事件，调用所有注册的处理器
 *    - 支持事件数据（payload）
 * 
 * 2. **on**：
 *    - 监听事件，注册处理器
 *    - 返回取消监听的函数
 * 
 * 3. **off**：
 *    - 取消监听事件
 *    - 移除指定的处理器
 * 
 * 4. **once**：
 *    - 监听事件（仅触发一次）
 *    - 触发后自动取消监听
 * 
 * ============================================
 * ⚠️ 注意事项
 * ============================================
 * 
 * 1. **事件命名**：
 *    - 使用 `模块名:动作名` 格式（如 `workspace:node-clicked`）
 *    - 使用 camelCase，更易读
 * 
 * 2. **事件数据**：
 *    - 事件数据（payload）可以是任意类型
 *    - 建议使用对象类型，便于扩展
 * 
 * 3. **内存泄漏**：
 *    - 组件卸载时应该取消监听
 *    - 使用 `on` 返回的取消函数取消监听
 * 
 * ============================================
 * 📚 相关文档
 * ============================================
 * 
 * - 事件总线接口：`web/src/architecture/domain/interfaces/IEventBus.ts`
 * - 事件类型定义：`web/src/architecture/domain/interfaces/IEventBus.ts`
 * - 事件类型注册表：`web/src/architecture/infrastructure/eventBus/EventTypeRegistry.ts`
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

