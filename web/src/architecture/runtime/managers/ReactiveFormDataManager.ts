/**
 * ReactiveFormDataManager - 响应式表单数据管理器
 * 🔥 增强版：集成事件总线，支持组件间通信
 */

import { reactive, type UnwrapNestedRefs } from 'vue'
import type { FieldValue } from '../types/field'
import { Logger } from '../utils/logger'

/**
 * 简单的事件发射器
 */
class EventEmitter {
  private events: Map<string, Function[]> = new Map()

  on(event: string, handler: Function): void {
    if (!this.events.has(event)) {
      this.events.set(event, [])
    }
    this.events.get(event)!.push(handler)
  }

  off(event: string, handler: Function): void {
    const handlers = this.events.get(event)
    if (handlers) {
      const index = handlers.indexOf(handler)
      if (index > -1) {
        handlers.splice(index, 1)
      }
    }
  }

  emit(event: string, payload?: any): void {
    const handlers = this.events.get(event)
    if (handlers) {
      handlers.forEach(handler => handler(payload))
    }
  }

  clear(): void {
    this.events.clear()
  }
}

export class ReactiveFormDataManager {
  // 存储所有字段的值（field_path -> FieldValue）
  private data: UnwrapNestedRefs<Map<string, FieldValue>>
  
  // 🔥 事件总线
  private eventBus: EventEmitter

  constructor() {
    this.data = reactive(new Map<string, FieldValue>())
    this.eventBus = new EventEmitter()
  }

  /**
   * 获取字段值
   */
  getValue(fieldPath: string): FieldValue {
    const value = this.data.get(fieldPath)
    if (!value) {
      // 返回默认值
      return {
        raw: '',
        display: '',
        meta: {}
      }
    }
    return value
  }

  /**
   * 设置字段值
   */
  setValue(fieldPath: string, value: FieldValue): void {
    this.data.set(fieldPath, value)
  }

  /**
   * 初始化字段值
   */
  initializeField(fieldPath: string, initialValue?: FieldValue): void {
    // 🔥 如果提供了 initialValue，直接设置（即使字段已存在也要更新）
    if (initialValue) {
      this.data.set(fieldPath, initialValue)
    } else if (!this.data.has(fieldPath)) {
      // 如果没有提供 initialValue 且字段不存在，使用默认空值
      const defaultFieldValue: FieldValue = {
        raw: '',
        display: '',
        meta: {}
      }
      this.data.set(fieldPath, defaultFieldValue)
    }
  }

  /**
   * ❌ 已删除 prepareSubmitData()
   * 原因：实现太简单（不处理嵌套），已被 FormRenderer.prepareSubmitDataWithTypeConversion() 取代
   * 新方法使用 Widget 递归收集，支持任意深度嵌套
   */

  /**
   * 获取所有字段路径
   */
  getAllFieldPaths(): string[] {
    return Array.from(this.data.keys())
  }

  /**
   * 清空所有数据
   */
  clear(): void {
    this.data.clear()
    this.eventBus.clear()
  }

  /**
   * 🔥 发出事件
   * @param eventType 事件类型，如 'field:search', 'field:change'
   * @param payload 事件数据
   */
  emit(eventType: string, payload: any): void {
    // 发出完整事件
    this.eventBus.emit(eventType, payload)
    
    // 🔥 支持通配符匹配：field:change:products[0].product_id
    // → 也触发 field:change:products[].product_id
    const patterns = this.extractPatterns(eventType)
    patterns.forEach(pattern => {
      this.eventBus.emit(pattern, payload)
    })
    
  }

  /**
   * 🔥 监听事件
   * @param eventPattern 事件模式，支持通配符 [] 和 *
   * @param handler 事件处理函数
   * @returns 取消监听的函数
   */
  on(eventPattern: string, handler: Function): () => void {
    this.eventBus.on(eventPattern, handler)
    
    // 返回取消监听函数
    return () => {
      this.eventBus.off(eventPattern, handler)
    }
  }

  /**
   * 🔥 取消监听事件
   * @param eventPattern 事件模式
   * @param handler 事件处理函数
   */
  off(eventPattern: string, handler: Function): void {
    this.eventBus.off(eventPattern, handler)
  }

  /**
   * 🔥 提取事件模式（支持通配符）
   * 例如：'field:change:products[0].product_id'
   * → ['field:change:products[0].product_id', 'field:change:products[].product_id']
   */
  private extractPatterns(eventType: string): string[] {
    const patterns: string[] = []
    
    // 如果包含数组索引 [0], [1] 等，生成通配符版本
    if (/\[\d+\]/.test(eventType)) {
      // 替换 [数字] 为 []
      const wildcardPattern = eventType.replace(/\[\d+\]/g, '[]')
      patterns.push(wildcardPattern)
    }
    
    return patterns
  }
}

