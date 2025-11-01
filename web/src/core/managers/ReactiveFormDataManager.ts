/**
 * ReactiveFormDataManager - 响应式表单数据管理器
 * 🔥 简化版：先实现核心功能
 */

import { reactive, type UnwrapNestedRefs } from 'vue'
import type { FieldValue } from '../types/field'

export class ReactiveFormDataManager {
  // 存储所有字段的值（field_path -> FieldValue）
  private data: UnwrapNestedRefs<Map<string, FieldValue>>

  constructor() {
    this.data = reactive(new Map<string, FieldValue>())
    console.log('[ReactiveFormDataManager] 初始化')
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
    console.log(`[ReactiveFormDataManager] 设置值: ${fieldPath}`, value)
  }

  /**
   * 初始化字段值
   */
  initializeField(fieldPath: string, initialValue?: FieldValue): void {
    if (!this.data.has(fieldPath)) {
      // 如果提供了 FieldValue，直接使用；否则使用默认空值
      const defaultFieldValue: FieldValue = initialValue || {
        raw: '',
        display: '',
        meta: {}
      }
      
      this.data.set(fieldPath, defaultFieldValue)
      console.log(`[ReactiveFormDataManager] 初始化字段: ${fieldPath}`, defaultFieldValue)
    }
  }

  /**
   * 准备提交数据（提取所有 raw 值）
   */
  prepareSubmitData(): Record<string, any> {
    const result: Record<string, any> = {}

    for (const [fieldPath, fieldValue] of this.data) {
      // 简单处理：只取顶层字段
      // TODO: 后续处理嵌套结构
      if (!fieldPath.includes('.') && !fieldPath.includes('[')) {
        result[fieldPath] = fieldValue.raw
      }
    }

    console.log('[ReactiveFormDataManager] 提交数据:', result)
    return result
  }

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
    console.log('[ReactiveFormDataManager] 清空数据')
  }
}

